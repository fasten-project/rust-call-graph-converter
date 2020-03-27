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

// Type Hierarchy converted to maps
type MapTypeHierarchy struct {
	Types  map[int64]Type
	Traits map[int64]Trait
	Impls  map[string]Impl
}

// Convert data of type hierarchy read from json to a map for simplifying queries
func (typeHierarchy TypeHierarchy) convertToMap() MapTypeHierarchy {
	mapTypeHierarchy := MapTypeHierarchy{
		Types:  make(map[int64]Type),
		Traits: make(map[int64]Trait),
		Impls:  make(map[string]Impl),
	}

	for _, typeInstance := range typeHierarchy.Types {
		mapTypeHierarchy.Types[typeInstance.Id] = typeInstance
	}
	typeHierarchy.Types = nil

	for _, traitInstance := range typeHierarchy.Traits {
		mapTypeHierarchy.Traits[traitInstance.Id] = traitInstance
	}
	typeHierarchy.Traits = nil

	for _, implInstance := range typeHierarchy.Impls {
		mapTypeHierarchy.Impls[implInstance.RelativeDefId] = implInstance
	}
	typeHierarchy.Impls = nil

	return mapTypeHierarchy
}

// Converts a relativeDefPath to the path in Fasten format.
func (typeHierarchy MapTypeHierarchy) getFullPath(relativeDefId string) (string, error) {
	crate, modules, method, err := typeHierarchy.parseRelativeDefPath(relativeDefId)
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")

	fullPath := "/" + squareBracketsPattern.ReplaceAllString(crate, "")
	for _, module := range modules {
		if strings.Contains(module, "{{impl}}") {
			resolvedModuleName := typeHierarchy.getTypeFromTypeHierarchy(relativeDefId)
			fullPath += "/" + resolvedModuleName
		} else {
			fullPath += "." + squareBracketsPattern.ReplaceAllString(module, "")
		}
	}
	fullPath += "." + squareBracketsPattern.ReplaceAllString(method, "") + "()"

	return fullPath, err
}

// Parses relative_def_path and returns a tuple containing crate name,
// array of modules and method name
func (typeHierarchy MapTypeHierarchy) parseRelativeDefPath(relativeDefId string) (string, []string, string, error) {
	pattern := regexp.MustCompile("::{{closure}}\\[[0-9]*]")
	relativeDefId = pattern.ReplaceAllString(relativeDefId, "")
	elements := strings.Split(relativeDefId, "::")
	if len(elements) < 2 {
		panic("Incorrect path")
	}
	if len(elements) == 2 {
		return elements[0], []string{}, elements[1], nil
	}

	var modules []string
	for i := 1; i < len(elements)-1; i++ {
		modules = append(modules, elements[i])
	}
	return elements[0], modules, elements[len(elements)-1], nil
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait
func (typeHierarchy MapTypeHierarchy) getTypeFromTypeHierarchy(relativeDefId string) string {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	relativeDefId = pattern.FindString(relativeDefId)
	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		return typeHierarchy.Types[implementation.TypeId].StringId
	}
	return relativeDefId
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait
func (typeHierarchy MapTypeHierarchy) getTraitFromTypeHierarchy(relativeDefId string) string {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	relativeDefId = pattern.FindString(relativeDefId)
	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		if implementation.TraitId != 0 {
			id := implementation.TraitId
			return getTraitPath(typeHierarchy.Traits[id-typeHierarchy.Traits[0].Id].RelativeDefId)
		}
	}
	return ""
}

// Extract the namespace from the full type info by removing the the function name
// at the end
func getNamespace(method string) string {
	index := strings.LastIndex(method, ".")
	return method[:index]
}

func getTraitPath(relativeDefId string) string {
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")
	relativeDefId = squareBracketsPattern.ReplaceAllString(relativeDefId, "")

	elements := strings.Split(relativeDefId, "::")

	path := "/"
	for _, elem := range elements[:len(elements) - 2] {
		path += elem + "."
	}
	path += elements[len(elements) - 2] + "/" + elements[len(elements) - 1]

	return path
}
