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

func (typeHierarchy TypeHierarchy) convertToMap() MapTypeHierarchy {
	mapTypeHierarchy := MapTypeHierarchy{
		Types:  make(map[int64]Type),
		Traits: make(map[int64]Trait),
		Impls:  make(map[string]Impl),
	}

	for _, typeInstance := range typeHierarchy.Types {
		mapTypeHierarchy.Types[typeInstance.Id] = typeInstance
	}
	for _, traitInstance := range typeHierarchy.Traits {
		mapTypeHierarchy.Traits[traitInstance.Id] = traitInstance
	}
	for _, implInstance := range typeHierarchy.Impls {
		mapTypeHierarchy.Impls[implInstance.RelativeDefId] = implInstance
	}
	return mapTypeHierarchy
}

func (typeHierarchy MapTypeHierarchy) getFullPath(relativeDefId string) string {
	crate, modules, method := typeHierarchy.parseRelativeDefPath(relativeDefId)
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")

	fullPath := "/" + squareBracketsPattern.ReplaceAllString(crate, "")
	for _, module := range modules {
		if strings.Contains(module, "{{impl}}") {
			resolvedModuleName, _ := typeHierarchy.resolveTypeHierarchyReference(relativeDefId)
			fullPath += "/" + resolvedModuleName
		} else {
			fullPath += "/" + squareBracketsPattern.ReplaceAllString(module, "")
		}
	}
	fullPath += "/" + squareBracketsPattern.ReplaceAllString(method, "") + "()"

	return fullPath
}

// Parses relative_def_path and returns a tuple containing crate name,
// array of modules and method name
func (typeHierarchy MapTypeHierarchy) parseRelativeDefPath(relativeDefId string) (string, []string, string) {
	pattern := regexp.MustCompile("::\\{\\{closure}}\\[0]")
	relativeDefId = pattern.ReplaceAllString(relativeDefId, "")
	elements := strings.Split(relativeDefId, "::")
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

func (typeHierarchy MapTypeHierarchy) resolveTypeHierarchyReference(relativeDefId string) (string, string) {
	pattern := regexp.MustCompile("^.*\\{\\{impl}}\\[[0-9]*]")
	relativeDefId = pattern.FindString(relativeDefId)
	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		associatedType := typeHierarchy.Types[implementation.TypeId].StringId
		associatedTrait := ""

		if implementation.TraitId != 0 {
			id := implementation.TraitId
			associatedTrait = typeHierarchy.Traits[id-typeHierarchy.Traits[0].Id].RelativeDefId
		}
		return associatedType, associatedTrait
	}
	return relativeDefId, ""
}
