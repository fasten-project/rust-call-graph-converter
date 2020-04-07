package rust

import (
	"RustCallGraphConverter/src/internal/fasten"
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
}

//Converts rustJSON to FastenJSON.
func (rustJSON JSON) ConvertToFastenJson(rawTypeHierarchy TypeHierarchy, stdTypeHierarchy MapTypeHierarchy) ([]fasten.JSON, error) {
	var jsons = make(map[string]*fasten.JSON)
	var methods = make(map[int64]string)
	var edgeMap = make(map[int64][]int64)

	typeHierarchy := rawTypeHierarchy.ConvertToMap()

	for _, node := range append(rustJSON.Functions, rustJSON.Macros...) {
		if _, ok := jsons[node.CrateName]; !ok {
			jsons[node.CrateName] = &fasten.JSON{
				Product:   node.CrateName,
				Forge:     "cratesio",
				Generator: "rust-callgraphs",
				Depset:    [][]fasten.Dependency{},
				Version:   node.PackageVersion,
				Cha:       map[string]fasten.Type{},
				Graph:     fasten.CallGraph{},
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

	var result []fasten.JSON
	for _, value := range jsons {
		result = append(result, *value)
	}

	return result, nil
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
				source.AddExternalCall(sourceMethod, "//"+target.Product+targetMethod)
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
	if len(namespaces) > 0 {
		fastenJSON.AddInterfaceToCHA(namespaces[0], typeHierarchy.getTraitFromTypeHierarchy(node.RelativeDefId))
	}

	return ids
}
