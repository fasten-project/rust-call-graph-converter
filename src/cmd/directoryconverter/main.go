package main

import (
	"RustCallGraphConverter/src/internal/rust"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var directory = flag.String("d", ".", "directory containing rust call graphs")

func main() {
	flag.Parse()
	callgraphs := getCallGraphs()

	for key, files := range callgraphs {
		log.Printf("Parsing package: %s", key)
		cgFile, typeHierarchyFile := getFiles(files)

		var callGraph rust.JSON
		var typeHierarchy rust.TypeHierarchy
		_ = json.Unmarshal(cgFile, &callGraph)
		_ = json.Unmarshal(typeHierarchyFile, &typeHierarchy)

		// TODO: not very elegant placeholder for testing purposes. Should be replaced.
		fasten := callGraph.ConvertToFastenJson(typeHierarchy)
		_ = fasten
		_ = callGraph
		_ = typeHierarchy
	}
}

// Walk the current directory and return a map containing a /packageName/packageVersion/
// as a key and an array of containing callgraph.json and type_hierarchy.json paths
func getCallGraphs() map[string][]string {
	callgraphs := make(map[string][]string)
	cgs, _ := ioutil.ReadDir(*directory)

	for _, cg := range cgs {
		if !strings.HasPrefix(cg.Name(), ".") {
			_ = filepath.Walk(*directory+"/"+cg.Name(), func(path string, f os.FileInfo, err error) error {
				if f.Mode().IsRegular() && !strings.HasPrefix(f.Name(), ".") {
					packageName := strings.TrimPrefix(path, *directory)
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
