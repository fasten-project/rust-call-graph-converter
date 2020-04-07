# RustCallGraphConverter

Converts Rust call graphs to Fasten format

## Requirements

To run RustCallGraphConverter you should have Golang (version 1.14)

## Arguments

Accepts the following command line arguments: 
   * **-b**: Kafka broker in format host:port; default: localhost:9092
   * **-i**: Directory containing rust call graphs; default: .
   * **-o**: Kafka topic to produce to; default: default.produce.topic
   * **--threads**: Number of threads; default: 1

## Input 

RustCallGraphConverter accepts two JSON file as its input - `callgraph.json` & `type_hierarchy.json`
```json
{
  "functions": [
    {
      "id": 0,
      "package_name": "some_package",
      "package_version": "0.8.0",
      "crate_name": "first_crate",
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::{{impl}}[1]::nestedFunction::{{impl}}[0]::function[0]",
      "is_externally_visible": false,
      "num_lines": 2
    },
    {
      "id": 1,
      "package_name": "some_package",
      "package_version": "0.8.0",
      "crate_name": "first_crate",
      "relative_def_id": "crate_name[fa9d]::other_name_space[0]::function[0]::{{closure}}[0]",
      "is_externally_visible": false,
      "num_lines": 2
    },
    {
      "id": 2,
      "package_name": "some_package",
      "package_version": "1.3.7",
      "crate_name": "other_crate",
      "relative_def_id": "other_crate[fa9d]::function[0]::{{constant}}[0]",
      "is_externally_visible": false,
      "num_lines": 2
    }
  ],
  "function_calls": [
    [ 0, 1, true ],
    [ 0, 2, false ],
    [ 2, 0, true ]
  ]
}
```
Code fragment 1. Example of `callgraph.json`


```json
{
  "types": [
    {
      "id": 0,
      "string_id": "&mut [S: generic]",
      "package_name": "some_package",
      "package_version": "0.8.0",
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::ConcreteType[0]"
    },
    {
      "id": 1,
      "string_id": "(A1: generic, A2: generic, A3: generic, )",
      "package_name": null,
      "package_version": null,
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::ConcreteType[0]::nestedFunction::OtherType[0]"
    }
  ],
  "traits": [
    {
      "id": 2,
      "package_name": "some_package",
      "package_version": "0.8.0",
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::SomeTrait[0]"
    }
  ],
  "impls": [
    {
      "id": 3,
      "type_id": 0,
      "trait_id": 2,
      "package_name": "some_package",
      "package_version": "0.8.0",
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::{{impl}}[1]"
    },
    {
      "id": 4,
      "type_id": 1,
      "trait_id": null,
      "package_name": "some_package",
      "package_version": "0.8.0",
      "relative_def_id": "crate_name[fa9d]::name[0]::space[0]::{{impl}}[1]::nestedFunction::{{impl}}[0]"
    }
  ]
}
```
Code fragment 2. Example of `type_hierarchy.json`

## Output

Every crate from `Functions` in rust call graphs is converted to a separate call graph in Fasten format. 
The output for the example in _Code fragment 1_  will be two following Fasten call graphs:
```json
{
  "product": "first_crate",
  "forge": "cratesio",
  "generator": "rust-callgraphs",
  "depset": [
    [
      {
        "product": "other_crate",
        "forge": "cratesio",
        "constraints": [
          "1.3.7"
        ]
      }
    ]
  ],
  "version": "0.8.0",
  "cha": {
    "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A1%3A%20generic": {
      "methods": {
        "0": "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A1%3A%20generic.function()"
      }
    },
    "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A2%3A%20generic": {
      "methods": {
        "1": "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A2%3A%20generic.function()"
      }
    },
    "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A3%3A%20generic": {
      "methods": {
        "2": "/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A3%3A%20generic.function()"
      }
    },
    "/other_name_space/NO-TYPE-DEFINITION": {
      "methods": {
        "3": "/other_name_space/NO-TYPE-DEFINITION.function()"
      }
    }
  },
  "graph": {
    "internalCalls": [
      [ 0, 3 ],
      [ 1, 3 ],
      [ 2, 3 ]
    ],
    "externalCalls": [
      [ 0, "//other_crate//NO-TYPE-DEFINITION.function()" ],
      [ 1, "//other_crate//NO-TYPE-DEFINITION.function()" ],
      [ 2, "//other_crate//NO-TYPE-DEFINITION.function()" ]
    ]
  },
  "timestamp": -1
}
```
Code fragment 3. Fasten Call graph for crate `first_crate`

```json
{
  "product": "other_crate",
  "forge": "cratesio",
  "generator": "rust-callgraphs",
  "depset": [
    [
      {
        "product": "first_crate",
        "forge": "cratesio",
        "constraints": [
          "0.8.0"
        ]
      }
    ]
  ],
  "version": "1.3.7",
  "cha": {
    "//NO-TYPE-DEFINITION": {
      "methods": {
        "0": "//NO-TYPE-DEFINITION.function()"
      }
    }
  },
  "graph": {
    "externalCalls": [
      [ 0, "//first_crate/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A1%3A%20generic.function()" ],
      [ 0, "//first_crate/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A2%3A%20generic.function()" ],
      [ 0, "//first_crate/name.space/%26mut%20S%5B%5D%3A%20generic.nestedFunction%28%29$A3%3A%20generic.function()" ]
    ]
  },
  "timestamp": -1
}
```
Code fragment 4. Fasten Call graph for package `other_crate`

## Run 

```shell
git clone https://github.com/fasten-project/rust-call-graph-converter.git
cd rust-call-graph-converter
go build -o main ./src/cmd/converter/main.go
./main -b localhost:9092 -o produce.topic.name -i /directory/with/rust/callgraphs --threads 5
```

## Docker

```shell
git clone https://github.com/fasten-project/rust-call-graph-converter.git
cd rust-call-graph-converter
docker build -t rust-converter .
docker run -it -v /directory/with/rust/callgraphs/:/data rust-converter -i /data -o host.docker.internal:9092 --threads 5
```
