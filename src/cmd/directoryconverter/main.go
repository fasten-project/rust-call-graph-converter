package main

import (
	"RustCallGraphConverter/src/internal/fasten"
	"RustCallGraphConverter/src/internal/rust"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var inputDirectory = flag.String("i", ".", "directory containing rust call graphs")
var outputDirectory = flag.String("o", ".", "output directory for fasten call graphs")

func main() {
	flag.Parse()
	callgraphs := getCallGraphs()

	for pkg, files := range callgraphs {
		cgFile, typeHierarchyFile := getFiles(files)

		var callGraph rust.JSON
		var typeHierarchy rust.TypeHierarchy
		err := json.Unmarshal(cgFile, &callGraph)
		err = json.Unmarshal(typeHierarchyFile, &typeHierarchy)

		start := time.Now()
		fastenCallGraphs, err := callGraph.ConvertToFastenJson(typeHierarchy)
		end := time.Since(start).Seconds()

		err = writeCallGraphs(fastenCallGraphs, pkg)

		if err == nil {
			log.Printf("Succesfully converted: %s in %f sec", pkg, end)
		} else {
			log.Printf("Failed to convert: %s, ERROR: %s", pkg, err)
		}
	}
}

// Walk the current directory and return a map containing a /packageName/packageVersion/
// as a key and an array of containing callgraph.json and type_hierarchy.json paths
func getCallGraphs() map[string][]string {
	callgraphs := make(map[string][]string)
	cgs, _ := ioutil.ReadDir(*inputDirectory)

	for _, cg := range cgs {
		if !strings.HasPrefix(cg.Name(), ".") {
			_ = filepath.Walk(*inputDirectory+"/"+cg.Name(), func(path string, f os.FileInfo, err error) error {
				if f.Mode().IsRegular() && !strings.HasPrefix(f.Name(), ".") {
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
func writeCallGraphs(fastenCallGraphs []fasten.JSON, pkg string) error {
	path := *outputDirectory + "/fasten" + pkg
	err := os.MkdirAll(path, 0755)

	for _, fastenCallGraph := range fastenCallGraphs {
		if !fastenCallGraph.IsEmpty() {
			fastenJson, _ := json.Marshal(fastenCallGraph)
			f, err := os.Create(path + fastenCallGraph.Product + ".json")
			if err == nil {
				_, err = f.Write(fastenJson)
				err = f.Close()
			}
		}
	}
	return err
}
