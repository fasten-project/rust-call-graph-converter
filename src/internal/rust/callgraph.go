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

//Converts rustJSON to FastenJSON
func (rustJSON JSON) ConvertToFastenJson(rawTypeHierarchy TypeHierarchy) ([]fasten.JSON, error) {
	var jsons = make(map[string]*fasten.JSON)
	var methods = make(map[int64]string)

	typeHierarchy := rawTypeHierarchy.convertToMap()

	for _, node := range append(rustJSON.Functions, rustJSON.Macros...) {
		if _, ok := jsons[node.PackageName]; !ok {
			jsons[node.PackageName] = &fasten.JSON{
				Product:   node.PackageName,
				Forge:     "cratesio",
				Generator: "rust-callgraphs",
				Depset:    [][]fasten.Dependency{},
				Version:   node.PackageVersion,
				Cha:       map[string]fasten.Type{},
				Graph:     fasten.CallGraph{},
				Timestamp: -1,
			}
		}
		addMethodToCHA(jsons, node, typeHierarchy)
		methods[node.Id] = node.PackageName
	}

	for _, edge := range rustJSON.FunctionCalls {
		rustJSON.addCallToGraph(jsons, methods, edge, typeHierarchy)
	}

	var result []fasten.JSON
	for _, value := range jsons {
		result = append(result, *value)
	}

	return result, nil
}

// Add a call to graph of a source package
func (rustJSON JSON) addCallToGraph(jsons map[string]*fasten.JSON, methods map[int64]string,
	edge []interface{}, typeHierarchy MapTypeHierarchy) {
	if edge[2] == true {
		sourceIndex := int64(edge[0].(float64))
		targetIndex := int64(edge[1].(float64))
		sourcePkg := methods[sourceIndex]
		targetPkg := methods[targetIndex]
		source := jsons[sourcePkg]
		target := jsons[targetPkg]

		if targetPkg != sourcePkg {
			source.AddDependency(target)

			source.AddExternalCall(sourceIndex, "//"+target.Product+
				rustJSON.getTargetMethod(typeHierarchy, targetIndex))
		} else {
			source.AddInternalCall(sourceIndex, targetIndex)
		}
	}
}

// In case target does not belong to the source package, resolves the
// full target method from a class hierarchy of a target package.
func (rustJSON JSON) getTargetMethod(typeHierarchy MapTypeHierarchy, targetIndex int64) string {
	if path, err := typeHierarchy.getFullPath(rustJSON.Functions[targetIndex].RelativeDefId); err == nil {
		return path
	}
	return ""
}

// Add method to Class Hierarchy.
func addMethodToCHA(jsons map[string]*fasten.JSON, node Node, typeHierarchy MapTypeHierarchy) {
	fastenJSON := jsons[node.PackageName]
	path, _ := typeHierarchy.getFullPath(node.RelativeDefId)
	namespace := getNamespace(path)
	
	fastenJSON.AddMethodToCHA(namespace, node.Id, path)
	fastenJSON.AddInterfaceToCHA(namespace, typeHierarchy.getTraitFromTypeHierarchy(node.RelativeDefId))
}

