package main

import (
	"RustCallGraphConverter/src/internal/fasten"
	"RustCallGraphConverter/src/internal/rust"
	"context"
	"encoding/json"
	"flag"
	"github.com/segmentio/kafka-go"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"time"
)

var broker = flag.String("b", "localhost:9092", "broker address in format host:port")
var group = flag.String("g", "default", "consumer group")
var consumeKafkaTopic = flag.String("c", "default.consume.topic", "kafka topic to consume")
var produceKafkaTopic = flag.String("p", "default.produce.topic", "kafka topic to send to")

var producer *kafka.Writer
var consumer *kafka.Reader

func main() {
	// Parse command line parameters
	flag.Parse()

	// Initialize Kafka consumer and producer
	consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{*broker},
		GroupID:        *group,
		Topic:          *consumeKafkaTopic,
		CommitInterval: time.Second,
	})
	log.Printf("Created consumer [broker: %s, group: %s, topic: %s]... ", *broker, *group, *consumeKafkaTopic)

	producer = kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{*broker},
		Topic:    *produceKafkaTopic,
		Balancer: &kafka.LeastBytes{},
	})
	log.Printf("Created producer [broker: %s, topic: %s]... ", *broker, *produceKafkaTopic)

	// Read type hierarchy of a standard library
	var rawStdTypeHierarchy rust.TypeHierarchy
	stdTypeHierarchyFile, _ := ioutil.ReadFile("src/internal/rust/standardlibrary/type_hierarchy.json")
	_ = json.Unmarshal(stdTypeHierarchyFile, &rawStdTypeHierarchy)
	stdTypeHierarchy := rawStdTypeHierarchy.ConvertToMap()

	// Properly close Kafka consumer and producer after successful consumption
	defer closeConnection()

	// Consume topic and convert Rust call graphs to Fasten format
	consumeTopic(stdTypeHierarchy)
}

// Consumes Kafka topic containing Rust call graphs until interrupt signal has been caught
func consumeTopic(stdTypeHierarchy rust.MapTypeHierarchy) {
	log.Printf("Started consuming topic [@%s]", *consumeKafkaTopic)
	ctx := interruptContext()

	// Consumes topic and sends FastenCG to Kafka topic until the context is canceled with Ctrl + C
	for {
		select {
		case <-ctx.Done():
			log.Printf("Successfully finished consuming topic")
			return
		default:
			convertedCG := consume(ctx, stdTypeHierarchy)
			for _, cg := range convertedCG {
				if !cg.IsEmpty() {
					sendToKafka(cg.ToJSON())
				}
			}
		}
	}
}

// Consumes rust call graphs from Kafka topic and
// converts them to an array of Fasten JSONs
func consume(ctx context.Context, stdTypeHierarchy rust.MapTypeHierarchy) []fasten.JSON {
	m, err := consumer.ReadMessage(ctx)

	if err != nil {
		// Ignore context canceled error
		if err.Error() != "context canceled" {
			log.Printf("%% Error: %v", err.Error())
		}
		return []fasten.JSON{}
	} else {
		var rustGraph rust.JSON
		var typeHierarchy rust.TypeHierarchy
		_ = json.Unmarshal(m.Value, &rustGraph)
		_ = json.Unmarshal(m.Value, &typeHierarchy)
		log.Printf("%% Consumed record [@%s] at offset %d", m.Topic, m.Offset)
		converted, _ := rustGraph.ConvertToFastenJson(typeHierarchy, stdTypeHierarchy)
		return converted
	}
}

// Sends fasten call graphs to Kafka
func sendToKafka(msg []byte) {
	_ = producer.WriteMessages(context.Background(),
		kafka.Message{
			Value: msg,
		},
	)
}

// Creates context that cancels when Ctrl + C is caught
func interruptContext() context.Context {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()
	return ctx
}

// Closes Kafka consumer and Producer
func closeConnection() {
	_ = consumer.Close()
	log.Printf("Closed consumer")
	_ = producer.Close()
	log.Printf("Closed producer")
}
