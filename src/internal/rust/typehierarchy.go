package rust

import (
	"regexp"
	"strings"
)

// TypeHierarchy
type TypeHierarchy struct {
	Types  []Type  `json:"types"`
	Traits []Trait `json:"traits"`
	Impls  []Impl  `json:"impls"`
}

type Type struct {
	Id             int64  `json:"id"`
	StringId       string `json:"string_id"`
	PackageName    string `json:"package_name"`
	PackageVersion string `json:"package_version"`
	RelativeDefId  string `json:"relative_def_id"`
}

type Trait struct {
	Id             int64  `json:"id"`
	PackageName    string `json:"package_name"`
	PackageVersion string `json:"package_version"`
	RelativeDefId  string `json:"relative_def_id"`
}

type Impl struct {
	Id             int64  `json:"id"`
	TypeId         int64  `json:"type_id"`
	TraitId        int64  `json:"trait_id"`
	PackageName    string `json:"package_name"`
	PackageVersion string `json:"package_version"`
	RelativeDefId  string `json:"relative_def_id"`
}

func (typeHierarchy TypeHierarchy) getFullPath(relativeDefPath string) {
	crate, modules, method := parseRelativeDefPath(relativeDefPath)
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")

	methodName := "/" + squareBracketsPattern.ReplaceAllString(crate, "")
	for _, module := range modules {
		methodName += "/" + squareBracketsPattern.ReplaceAllString(module, "")
	}
	methodName += "/" + squareBracketsPattern.ReplaceAllString(method, "") + "()"

}

// Parses relative_def_path and returns a tuple containing crate name,
// array of modules and method name
func parseRelativeDefPath(relativeDefPath string) (string, []string, string) {
	pattern := regexp.MustCompile("::\\{\\{closure}}\\[0]")
	relativeDefPath = pattern.ReplaceAllString(relativeDefPath, "")
	elements := strings.Split(relativeDefPath, "::")
	if len(elements) < 2 {
		panic("Incorrect path")
	}
	if len(elements) == 2 {
		return elements[0], []string{}, elements[1]
	}

	var modules []string
	for i := 1; i < len(elements)-1; i++ {
		if strings.Contains(elements[i], "{{impl}}") {

		}
		modules = append(modules, elements[i])
	}
	return elements[0], modules, elements[len(elements)-1]
}
