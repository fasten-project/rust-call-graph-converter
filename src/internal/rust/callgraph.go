package rust

import (
	"RustCallGraphConverter/src/internal/fasten"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// CallGraph
type JSON struct {
	Functions     []Node          `json:"functions"`
	Macros        []Node          `json:"macros"`
	FunctionCalls [][]interface{} `json:"function_calls"`
}

type Node struct {
	Id                int64  `json:"id"`
	PackageName       string `json:"package_name"`
	PackageVersion    string `json:"package_version"`
	CrateName         string `json:"crate_name"`
	RelativeDefId     string `json:"relative_def_id"`
	ExternallyVisible bool   `json:"is_externally_visible"`
	NumberOfLines     int64  `json:"num_lines"`
	SourceLocation    string `json:"source_location"`
}

type CratesioAPI struct {
	Version CratesioVersion `json:"version"`
}

type CratesioVersion struct {
	CreatedAt string `json:"created_at"`
}

//Converts rustJSON to FastenJSON.
func (rustJSON JSON) ConvertToFastenJson(rawTypeHierarchy TypeHierarchy, stdTypeHierarchy MapTypeHierarchy, pkg string) (fasten.JSON, error) {
	var jsons = make(map[string]*fasten.JSON)
	var methods = make(map[int64]string)
	var edgeMap = make(map[int64][]int64)

	typeHierarchy := rawTypeHierarchy.ConvertToMap()

	for _, node := range append(rustJSON.Functions, rustJSON.Macros...) {
		if _, ok := jsons[node.CrateName]; !ok {
			var version string
			if node.PackageVersion == "" {
				version = "0.0.0"
			} else {
				version = node.PackageVersion
			}
			jsons[node.CrateName] = &fasten.JSON{
				Product:   node.CrateName,
				Forge:     "cratesio",
				Generator: "rust-callgraphs",
				Depset:    [][]fasten.Dependency{},
				Version:   version,
				Cha:       map[string]fasten.Type{},
				Graph:     fasten.CallGraph{
					InternalCalls: make([][]int64, 0),
					ExternalCalls: make([][]interface{}, 0),
				},
				Timestamp: -1,
			}
		}
		id := addMethodToCHA(jsons, node, typeHierarchy)
		edgeMap[node.Id] = id
		methods[node.Id] = node.CrateName
	}

	for _, edge := range rustJSON.FunctionCalls {
		rustJSON.addCallToGraph(jsons, methods, edge, typeHierarchy, stdTypeHierarchy, edgeMap)
	}

	pkgCrate := strings.Split(pkg, "/")[1]
	pkgCrate = strings.ReplaceAll(pkgCrate, "-", "_")
	result := jsons[pkgCrate]
	if result != nil {
		resolveTimestamp(result)
		return *result, nil
	}

	return fasten.JSON{}, nil
}

// Add a call to graph of a source package.
func (rustJSON JSON) addCallToGraph(jsons map[string]*fasten.JSON, methods map[int64]string,
	edge []interface{}, typeHierarchy MapTypeHierarchy, stdTypeHierarchy MapTypeHierarchy, edgeMap map[int64][]int64) {
	sourceIndex := int64(edge[0].(float64))
	targetIndex := int64(edge[1].(float64))
	sourcePkg := methods[sourceIndex]
	targetPkg := methods[targetIndex]
	source := jsons[sourcePkg]
	target := jsons[targetPkg]

	if targetPkg != sourcePkg {
		source.AddDependency(target)

		for _, sourceMethod := range edgeMap[sourceIndex] {
			for _, targetMethod := range rustJSON.getTargetMethod(typeHierarchy, stdTypeHierarchy, targetIndex) {
				var metadata = make(map[string]string)
				if edge[2] == true {
					metadata["dispatch"] = "static"
				} else {
					metadata["dispatch"] = "dynamic"
				}
				source.AddExternalCall(sourceMethod, "//"+target.Forge+"!"+target.Product+"$"+target.Version+targetMethod, metadata)
			}
		}
	} else {
		for _, sourceMethod := range edgeMap[sourceIndex] {
			for _, targetMethod := range edgeMap[targetIndex] {
				source.AddInternalCall(sourceMethod, targetMethod)
			}
		}
	}
}

// Resolves the full target method path from a type hierarchy of the target package
// or from the type hierarchy of the standard library.
func (rustJSON JSON) getTargetMethod(typeHierarchy MapTypeHierarchy, stdTypeHierarchy MapTypeHierarchy, targetIndex int64) []string {
	if path, err := typeHierarchy.getFullPath(rustJSON.Functions[targetIndex].RelativeDefId); err == nil {
		if typeHierarchy.isGenericType(rustJSON.Functions[targetIndex].RelativeDefId) {
			return typeHierarchy.getGenericFullPaths(path)
		}
		return []string{path}
	}
	if path, err := stdTypeHierarchy.getFullPath(rustJSON.Functions[targetIndex].RelativeDefId); err == nil {
		if typeHierarchy.isGenericType(rustJSON.Functions[targetIndex].RelativeDefId) {
			return typeHierarchy.getGenericFullPaths(path)
		}
		return []string{path}
	}
	return []string{"UNKNOWN"}
}

// Add method to Class Hierarchy or passes control to addGenericMethodToCHA
// in case the method is has generic types.
func addMethodToCHA(jsons map[string]*fasten.JSON, node Node, typeHierarchy MapTypeHierarchy) []int64 {
	fastenJSON := jsons[node.CrateName]
	path, _ := typeHierarchy.getFullPath(node.RelativeDefId)
	namespace := getNamespace(path)

	if typeHierarchy.isGenericType(node.RelativeDefId) {
		return addGenericMethodToCHA(jsons, node, typeHierarchy)
	} else {
		id := fastenJSON.AddMethodToCHA(namespace, path)
		fastenJSON.AddInterfaceToCHA(namespace, typeHierarchy.getTraitFromTypeHierarchy(node.RelativeDefId))
		fastenJSON.AddFilenameToCHA(namespace, getFileName(node.SourceLocation, node.CrateName+"-"+node.PackageVersion))
		return []int64{id}
	}
}

// Processes a method with generic types and adds each generic type
// to CHA separately.
func addGenericMethodToCHA(jsons map[string]*fasten.JSON, node Node, typeHierarchy MapTypeHierarchy) []int64 {
	fastenJSON := jsons[node.CrateName]
	fullPath, _ := typeHierarchy.getFullPath(node.RelativeDefId)
	var ids []int64

	paths := typeHierarchy.getGenericFullPaths(fullPath)
	var namespaces []string
	for _, path := range paths {
		namespaces = append(namespaces, getNamespace(path))
	}

	for i := 0; i < len(paths) && i < len(namespaces); i++ {
		id := fastenJSON.AddMethodToCHA(namespaces[i], paths[i])
		ids = append(ids, id)
	}
	for _, namespace := range namespaces {
		fastenJSON.AddInterfaceToCHA(namespace, typeHierarchy.getTraitFromTypeHierarchy(node.RelativeDefId))
	}

	return ids
}

// Format file information from rust call graph.
func getFileName(rawFileName string, productVersion string) string {
	if rawFileName == "" {
		return rawFileName
	}
	elements := strings.Split(rawFileName, productVersion)
	if len(elements) > 1 {
		rawFileName = elements[1]
	}

	rawFileName = strings.Split(rawFileName, ":")[0]
	if strings.Contains(rawFileName, ".rs") {
		return rawFileName
	} else {
		return ""
	}
}

// Resolve a timestamp for the given fastenJson
func resolveTimestamp(fastenJSON *fasten.JSON) {
	uri := "https://crates.io/api/v1/crates/" + fastenJSON.Product + "/" + fastenJSON.Version
	resp, _ := http.Get(uri)
	respBody, _ := ioutil.ReadAll(resp.Body)
	var api CratesioAPI
	_ = json.Unmarshal(respBody, &api)
	_ = resp.Body.Close()

	layout := "2006-01-02T15:04:05.999999999Z07:00"
	date := api.Version.CreatedAt
	if date != "" {
		timestamp, _ := time.Parse(layout, date)
		fastenJSON.Timestamp = timestamp.Unix()
	}
}
