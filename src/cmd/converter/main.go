package main

import (
	"RustCallGraphConverter/src/internal/fasten"
	"RustCallGraphConverter/src/internal/rust"
	"encoding/json"
	"flag"
	"github.com/lovoo/goka"
	"github.com/lovoo/goka/codec"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var broker = flag.String("b", "localhost:9092", "broker address in format host:port")
var produceKafkaTopic = flag.String("o", "default.produce.topic", "kafka topic to send to")
var inputDirectory = flag.String("i", ".", "directory containing rust call graphs")
var threads = flag.Int("threads", 1, "number of threads goroutines")

var brokers []string
var topic goka.Stream
var emitter *goka.Emitter

func main() {
	flag.Parse()

	brokers = append(brokers, *broker)
	topic = goka.Stream(*produceKafkaTopic)

	var err error
	emitter, err = goka.NewEmitter(brokers, topic, new(codec.String))
	if err != nil {
		log.Fatalf("error creating emitter: %v", err)
	}
	defer emitter.Finish()

	callgraphs := getCallGraphs()

	// Read type hierarchy of a standard library
	var rawStdTypeHierarchy rust.TypeHierarchy
	stdTypeHierarchyFile, _ := ioutil.ReadFile("src/internal/rust/standardlibrary/type_hierarchy.json")
	_ = json.Unmarshal(stdTypeHierarchyFile, &rawStdTypeHierarchy)
	stdTypeHierarchy := rawStdTypeHierarchy.ConvertToMap()

	guard := make(chan struct{}, *threads)

	var wg sync.WaitGroup
	wg.Add(len(callgraphs))
	totalStart := time.Now()
	for pkg, files := range callgraphs {
		guard <- struct{}{}
		go func(pkg string, files []string) {
			defer wg.Done()

			cgFile, typeHierarchyFile := getFiles(files)

			var callGraph rust.JSON
			var typeHierarchy rust.TypeHierarchy
			err := json.Unmarshal(cgFile, &callGraph)
			err = json.Unmarshal(typeHierarchyFile, &typeHierarchy)

			start := time.Now()
			fastenCallGraphs, err := callGraph.ConvertToFastenJson(typeHierarchy, stdTypeHierarchy)
			end := time.Since(start).Seconds()

			err = writeCallGraphs(fastenCallGraphs)

			if err == nil {
				log.Printf("Succesfully converted: %s in %f sec", pkg, end)
			} else {
				log.Printf("Failed to convert: %s, ERROR: %s", pkg, err)
			}
			<-guard
		}(pkg, files)
	}
	wg.Wait()
	totalEnd := time.Since(totalStart).Seconds()
	log.Printf("Processign of %d callgraphs took %f seconds", len(callgraphs), totalEnd)
}

// Walk the current directory and return a map containing a /packageName/packageVersion/
// as a key and an array of containing callgraph.json and type_hierarchy.json paths
func getCallGraphs() map[string][]string {
	callgraphs := make(map[string][]string)
	cgs, _ := ioutil.ReadDir(*inputDirectory)

	for _, cg := range cgs {
		if !strings.HasPrefix(cg.Name(), ".") {
			_ = filepath.Walk(*inputDirectory+"/"+cg.Name(), func(path string, f os.FileInfo, err error) error {
				if f.Mode().IsRegular() && !strings.HasPrefix(f.Name(), ".") && strings.Contains(f.Name(), ".json") {
					packageName := strings.TrimPrefix(path, *inputDirectory)
					filename := strings.Split(packageName, string(filepath.Separator))
					packageName = strings.TrimSuffix(packageName, filename[len(filename)-1])
					callgraphs[packageName] = append(callgraphs[packageName], path)
				}
				return nil
			})
		}
	}
	return callgraphs
}

// Given an array containing paths to callgraph.json and type_hierarchy.json return
// content of those files in order (Callgraph, TypeHierarchy)
func getFiles(files []string) ([]byte, []byte) {
	var cgFile []byte
	var typeHierarchyFile []byte

	if strings.Contains(files[0], "callgraph.json") {
		cgFile, _ = ioutil.ReadFile(files[0])
		typeHierarchyFile, _ = ioutil.ReadFile(files[1])
	} else {
		cgFile, _ = ioutil.ReadFile(files[1])
		typeHierarchyFile, _ = ioutil.ReadFile(files[0])
	}

	return cgFile, typeHierarchyFile
}

// Writes an array of given fasten call graphs to "specified_at_startup_directory"/fasten/pkg
func writeCallGraphs(fastenCallGraphs []fasten.JSON) error {
	var err error
	for _, fastenCallGraph := range fastenCallGraphs {
		if !fastenCallGraph.IsEmpty() {
			fastenJson, _ := json.Marshal(fastenCallGraph)
			_ = fastenJson
			err = runEmitter(fastenJson)
		}
	}
	return err
}

func runEmitter(msg []byte) error {
	err := emitter.EmitSync("placeholder", string(msg))
	if err != nil {
		log.Fatalf("error emitting message: %v", err)
	}
	return err
}