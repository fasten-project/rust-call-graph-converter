# RustCallGraphConverter

Converts Rust call graphs to Fasten format

## Requirements

To run RustCallGraphConverter you should have Golang (version 1.14)

## Arguments

Accepts the following command line arguments: 
   * **-b**: Kafka broker in format host:port; default: localhost:9092
   * **-g**: Consumer group; default: default
   * **-c**: Kafka topic to consumer from; default: default.consume.topic
   * **-p**: Kafka topic to produce to; default: default.produce.topic

## Input 

RustCallGraphConverter accepts rust call graphs from a Kafka topic in the following format:
```json
{
  "nodes": [
    {
      "id": 0,
      "package": "memchr 2.3.0",
      "crate_name": "memchr",
      "relative_def_path": "memchr[8251]::fallback[0]::forward_search[0]"
    },
    {
      "id": 1,
      "package": "regex 1.3.4",
      "crate_name": "regex",
      "relative_def_path": "regex[77f8]::re_bytes[0]::{{impl}}[7]::captures_read[0]"
    }
  ],
  "edges": [
    [
      0,
      1,
      true
    ],
    [
      1,
      0,
      true
    ]
  ],
  "nodes_info": [
    {
      "id": 0,
      "num_lines": 16
    },
    {
      "id": 1,
      "num_lines": 23
    }
  ]
}
```
Code fragment 1. Example input rust call graph

## Output

Every package from `Nodes` in rust call graphs is converted to a separate call graph in Fasten format. 
The output for the example in _Code fragment 1_ will be two following Fasten call graphs:
```json
{
  "product": "memchr",
  "forge": "cratesio",
  "generator": "rust-callgraphs",
  "depset": [
    [
      {
        "product": "regex",
        "forge": "cratesio",
        "constraints": [
          "1.3.4"
        ]
      }
    ]
  ],
  "version": "2.3.0",
  "cha": {
    "memchr": {
      "methods": {
        "0": "memchr[8251]/fallback[0]/forward_search[0]"
      }
    }
  },
  "graph": {
    "externalCalls": [
      [
        0,
        "//regex[77f8]/re_bytes[0]/{{impl}}[7]/captures_read[0]"
      ]
    ]
  },
  "timestamp": -1
}
```
Code fragment 2. Fasten Call graph for package `memchr`

```json
{
  "product": "regex",
  "forge": "cratesio",
  "generator": "rust-callgraphs",
  "depset": [
    [
      {
        "product": "memchr",
        "forge": "cratesio",
        "constraints": [
          "2.3.0"
        ]
      }
    ]
  ],
  "version": "1.3.4",
  "cha": {
    "regex": {
      "methods": {
        "1": "regex[77f8]/re_bytes[0]/{{impl}}[7]/captures_read[0]"
      }
    }
  },
  "graph": {
    "externalCalls": [
      [
        1,
        "//memchr[8251]/fallback[0]/forward_search[0]"
      ]
    ]
  },
  "timestamp": -1
}
```
Code fragment 3. Fasten Call graph for package `regex`

## Run 

```shell
go build -o main ./src/cmd/converter/main.go
./main -b localhost:9092 -g defaultGroup -c rust.graphs -p fasten.rust.graphs
```
