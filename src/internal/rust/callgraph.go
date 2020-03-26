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
func (rustJSON JSON) ConvertToFastenJson(rawTypeHierarchy TypeHierarchy) []fasten.JSON {
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
		rustJSON.addMethodToCHA(jsons, node, typeHierarchy)
		methods[node.Id] = node.PackageName
	}

	for _, edge := range rustJSON.FunctionCalls {
		rustJSON.addCallToGraph(jsons, methods, edge, typeHierarchy)
	}

	var result []fasten.JSON
	for _, value := range jsons {
		result = append(result, *value)
	}

	return result
}

// Add a call to graph of a source package
func (rustJSON JSON) addCallToGraph(jsons map[string]*fasten.JSON, methods map[int64]string, edge []interface{}, typeHierarchy MapTypeHierarchy) {
	if edge[2] == true {
		sourceIndex := int64(edge[0].(float64))
		targetIndex := int64(edge[1].(float64))
		sourcePkg := methods[sourceIndex]
		targetPkg := methods[targetIndex]
		source := jsons[sourcePkg]
		target := jsons[targetPkg]

		if targetPkg != sourcePkg {
			rustJSON.addDependency(source, target)

			source.Graph.ExternalCalls = append(source.Graph.ExternalCalls,
				[]interface{}{sourceIndex, "//" + target.Product + rustJSON.getTargetMethod(typeHierarchy, targetIndex)})
		} else {
			source.Graph.InternalCalls = append(source.Graph.InternalCalls,
				[]int64{sourceIndex, targetIndex})
		}
	}
}

// In case target does not belong to the source package, resolves the
// full target method from a class hierarchy of a target package.
func (rustJSON JSON) getTargetMethod(typeHierarchy MapTypeHierarchy, targetIndex int64) string {
	return typeHierarchy.getFullPath(rustJSON.Functions[targetIndex].RelativeDefId)
}

// Add method to Class Hierarchy.
func (rustJSON JSON) addMethodToCHA(jsons map[string]*fasten.JSON, node Node, typeHierarchy MapTypeHierarchy) {
	fastenJSON := jsons[node.PackageName]
	if _, exists := fastenJSON.Cha[getNamespace(typeHierarchy.getFullPath(node.RelativeDefId))]; !exists {
		fastenJSON.Cha[getNamespace(typeHierarchy.getFullPath(node.RelativeDefId))] = fasten.Type{
			Methods: map[int64]string{},
		}
	}
	typeValue := fastenJSON.Cha[getNamespace(typeHierarchy.getFullPath(node.RelativeDefId))]
	typeValue.Methods[node.Id] = typeHierarchy.getFullPath(node.RelativeDefId)
}

// In case package of source method is different from the package of
// target method adds a dependency too the current JSON depset.
func (rustJSON JSON) addDependency(source *fasten.JSON, target *fasten.JSON) {
	if target.Product == "" {
		return
	}
	for _, inner := range source.Depset {
		for _, dependency := range inner {
			if dependency.Product == target.Product &&
				dependency.Constraints[0] == target.Version {
				return
			}
		}
	}
	if len(source.Depset) == 0 {
		source.Depset = append(source.Depset, []fasten.Dependency{})
	}
	source.Depset[0] = append(source.Depset[0], fasten.Dependency{
		Product:     target.Product,
		Forge:       "cratesio",
		Constraints: []string{target.Version},
	})
}
