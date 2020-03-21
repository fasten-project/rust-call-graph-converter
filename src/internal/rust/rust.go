package rust

import (
	"RustCallGraphConverter/src/internal/fasten"
	"regexp"
)

type JSON struct {
	Nodes     []Node          `json:"nodes"`
	Edges     [][]interface{} `json:"edges"`
	NodesInfo []Metadata      `json:"nodes_info"`
}

type Node struct {
	Id              int64  `json:"id"`
	Package         string `json:"package"`
	CrateName       string `json:"crate_name"`
	RelativeDefPath string `json:"relative_def_path"`
}

type Metadata struct {
	Id            int64 `json:"id"`
	NumberOfLines int64 `json:"num_lines"`
}

//Converts rustJSON to FastenJSON
func (rustJSON JSON) ConvertToFastenJson() []fasten.JSON {
	var jsons = make(map[string]*fasten.JSON)
	var methods = make(map[int64]string)

	for _, node := range rustJSON.Nodes {
		if _, ok := jsons[node.Package]; !ok {
			name, version := resolveNameAndVersion(node.Package)
			jsons[node.Package] = &fasten.JSON{
				Product:   name,
				Forge:     "cratesio",
				Generator: "rust-callgraphs",
				Depset:    [][]fasten.Dependency{},
				Version:   version,
				Cha:       map[string]fasten.Type{},
				Graph:     fasten.CallGraph{},
				Timestamp: -1,
			}
		}
		addMethodToCHA(jsons, node)
		methods[node.Id] = node.Package
	}

	for _, edge := range rustJSON.Edges {
		addCallToGraph(jsons, methods, edge)
	}

	// TODO: Something should be done about nodes_info

	var result []fasten.JSON
	for _, value := range jsons {
		result = append(result, *value)
	}

	return result
}

// Add a call to graph of a source package
func addCallToGraph(jsons map[string]*fasten.JSON, methods map[int64]string, edge []interface{}) {
	if edge[2] == true {
		sourceIndex := int64(edge[0].(float64))
		targetIndex := int64(edge[1].(float64))
		sourcePkg := methods[sourceIndex]
		targetPkg := methods[targetIndex]

		if targetPkg != sourcePkg {
			addDependency(jsons[sourcePkg], jsons[targetPkg])

			jsons[sourcePkg].Graph.ExternalCalls = append(jsons[sourcePkg].Graph.ExternalCalls,
				[]interface{}{sourceIndex, "///" + getTargetMethod(jsons[targetPkg].Cha, targetIndex)})
		} else {
			jsons[sourcePkg].Graph.InternalCalls = append(jsons[sourcePkg].Graph.InternalCalls,
				[]int64{sourceIndex, targetIndex})
		}
	}
}

// In case target does not belong to the source package, resolves the
// full target method from a class hierarchy of a target package.
func getTargetMethod(cha map[string]fasten.Type, targetIndex int64) string {
	for _, value := range cha {
		for key, method := range value.Methods {
			if key == targetIndex {
				return parseMethod(method)
			}
		}
	}
	return ""
}

// Add method to Class Hierarchy.
func addMethodToCHA(jsons map[string]*fasten.JSON, node Node) {
	fastenJSON := jsons[node.Package]
	if _, exists := fastenJSON.Cha[node.CrateName]; !exists {
		fastenJSON.Cha[node.CrateName] = fasten.Type{
			Methods: map[int64]string{},
		}
	}
	typeValue := fastenJSON.Cha[node.CrateName]
	typeValue.Methods[node.Id] = parseMethod(node.RelativeDefPath)
}

// In case package of source method is different from the package of
// target method adds a dependency too the current JSON depset.
func addDependency(source *fasten.JSON, target *fasten.JSON) {
	if target.Product == "NULL" {
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
	source.Depset = append(source.Depset, []fasten.Dependency{{
		Product:     target.Product,
		Forge:       "cratesio",
		Constraints: []string{target.Version},
	}})
}

// Given package field from rust.Node returns name and version of it.
func resolveNameAndVersion(pkg string) (string, string) {
	pattern := regexp.MustCompile(" [0-9]")
	slice := pattern.FindStringIndex(pkg)

	if len(slice) > 0 {
		return pkg[:slice[0]], pkg[slice[0]+1:]
	} else {
		return pkg, ""
	}
}

// TODO: learn how to parse relative_def_path
// WIP: Supposed to parse a relative_def_path and convert
// it to Fasten format.
func parseMethod(defPath string) string {
	pattern := regexp.MustCompile("::")
	return pattern.ReplaceAllString(defPath, "/")
}
