package rust

import (
	"RustCallGraphConverter/src/internal/fasten"
	"regexp"
	"strings"
)

type JSON struct {
	Nodes     []Node          `json:"nodes"`
	Edges     [][]interface{} `json:"edges"`
}

type Node struct {
	Id                int64  `json:"id"`
	PackageName       string `json:"package_name"`
	PackageVersion    string `json:"package_version"`
	CrateName         string `json:"crate_name"`
	RelativeDefPath   string `json:"relative_def_id"`
	ExternallyVisible bool	 `json:"is_externally_visible"`
	NumberOfLines     int64  `json:"num_lines"`
}

//Converts rustJSON to FastenJSON
func (rustJSON JSON) ConvertToFastenJson() []fasten.JSON {
	var jsons = make(map[string]*fasten.JSON)
	var methods = make(map[int64]string)

	for _, node := range rustJSON.Nodes {
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
		addMethodToCHA(jsons, node)
		methods[node.Id] = node.PackageName
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
		source := jsons[sourcePkg]
		target := jsons[targetPkg]

		if targetPkg != sourcePkg {
			addDependency(source, target)

			source.Graph.ExternalCalls = append(source.Graph.ExternalCalls,
				[]interface{}{sourceIndex, "///" + getTargetMethod(target.Cha, targetIndex)})
		} else {
			source.Graph.InternalCalls = append(source.Graph.InternalCalls,
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
				return method
			}
		}
	}
	return ""
}

// Add method to Class Hierarchy.
func addMethodToCHA(jsons map[string]*fasten.JSON, node Node) {
	fastenJSON := jsons[node.PackageName]
	if _, exists := fastenJSON.Cha[node.CrateName]; !exists {
		fastenJSON.Cha[node.CrateName] = fasten.Type{
			Methods: map[int64]string{},
		}
	}
	typeValue := fastenJSON.Cha[node.CrateName]
	typeValue.Methods[node.Id] = formatRelativeDefPath(node.RelativeDefPath)
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

// TODO: improve parsing of relative_def_path
// Converts relative_def_path to namespace/method() fasten format
func formatRelativeDefPath(relativeDefPath string) string {
	crate, modules, method := parseRelativeDefPath(relativeDefPath)
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")

	methodName := "/" + squareBracketsPattern.ReplaceAllString(crate, "")
	for _, module := range modules {
		methodName += "/" + squareBracketsPattern.ReplaceAllString(module, "")
	}
	methodName += "/" + squareBracketsPattern.ReplaceAllString(method, "") + "()"

	return methodName
}

// Parses relative_def_path and returns a tuple containing crate name,
// array of modules and method name
func parseRelativeDefPath(relativeDefPath string) (string, []string, string) {
	elements := strings.Split(relativeDefPath, "::")
	if len(elements) < 2 {
		panic("Incorrect path")
	}
	if len(elements) == 2 {
		return elements[0], []string{}, elements[1]
	}

	var modules []string
	for i := 1; i < len(elements)-1; i++ {
		modules = append(modules, elements[i])
	}
	return elements[0], modules, elements[len(elements)-1]
}
