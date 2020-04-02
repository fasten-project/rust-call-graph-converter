package rust

import (
	"errors"
	"net/url"
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
func (typeHierarchy TypeHierarchy) ConvertToMap() MapTypeHierarchy {
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
		fourCharIdPattern := regexp.MustCompile("\\[.{4}]")
		relativeDefId := fourCharIdPattern.ReplaceAllString(implInstance.RelativeDefId, "")
		mapTypeHierarchy.Impls[relativeDefId] = implInstance
	}
	typeHierarchy.Impls = nil

	return mapTypeHierarchy
}

// Converts a relativeDefPath to the path in Fasten format.
func (typeHierarchy MapTypeHierarchy) getFullPath(relativeDefId string) (string, error) {
	var err error
	modules, impl, nestedElements, method, err := typeHierarchy.parseRelativeDefPath(relativeDefId)

	fullPath := "/"
	fullPath += strings.Join(modules, ".")

	if strings.Contains(impl, "[") {
		impl = impl[1:len(impl)-1] + "[]"
	}
	fullPath += "/" + url.PathEscape(impl)

	for _, element := range nestedElements {
		if element[:1] == "$" {
			fullPath += "$" + url.PathEscape(element[1:])
		} else {
			fullPath += "." + url.PathEscape(element)
		}
	}
	fullPath += "." + method + "()"

	return fullPath, err
}

// Parses relative_def_path and returns a tuple containing slice of modules,
// resolved type name, nested functions and types, function name
func (typeHierarchy MapTypeHierarchy) parseRelativeDefPath(relativeDefId string) ([]string, string, []string, string, error) {
	patternClosure := regexp.MustCompile("::{{closure}}\\[[0-9]*]")
	patternConstant := regexp.MustCompile("::{{constant}}\\[[0-9]*]")
	squareBracketsPattern := regexp.MustCompile("\\[.*?]")

	var formattedRelativeDefId string
	formattedRelativeDefId = patternClosure.ReplaceAllString(relativeDefId, "")
	formattedRelativeDefId = patternConstant.ReplaceAllString(formattedRelativeDefId, "")
	formattedRelativeDefId = squareBracketsPattern.ReplaceAllString(formattedRelativeDefId, "")

	rawElements := strings.Split(relativeDefId, "::")
	elements := strings.Split(formattedRelativeDefId, "::")

	var err error = nil
	if len(elements) < 2 {
		err = errors.New("incorrect path")
	}

	var gotFirstImpl = false
	var relativeDefPathCurrentLength = 0

	var modules []string
	var impl string
	var nestedElements []string
	var function = elements[len(elements)-1]

	for i := 1; i < len(elements)-1; i++ {
		relativeDefPathCurrentLength++
		if elements[i] != "" {
			if !gotFirstImpl {
				if strings.Contains(elements[i], "{{impl}}") {
					gotFirstImpl = true
					currentRelativeDefId := strings.Join(rawElements[:relativeDefPathCurrentLength+1], "::")
					impl, err = typeHierarchy.getTypeFromTypeHierarchy(currentRelativeDefId)
				} else {
					modules = append(modules, url.PathEscape(elements[i]))
				}
			} else {
				if strings.Contains(elements[i], "{{impl}}") {
					var nestedType string
					currentRelativeDefId := strings.Join(rawElements[:relativeDefPathCurrentLength+1], "::")
					nestedType, err = typeHierarchy.getTypeFromTypeHierarchy(currentRelativeDefId)

					nestedElements = append(nestedElements, "$"+nestedType)
				} else {
					nestedElements = append(nestedElements, elements[i]+"()")
				}
			}
		}
	}
	if !gotFirstImpl {
		impl = "NO-TYPE-DEFINITION"
	}
	return modules, impl, nestedElements, function, err
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait
func (typeHierarchy MapTypeHierarchy) getTypeFromTypeHierarchy(relativeDefId string) (string, error) {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	fourCharIdPattern := regexp.MustCompile("\\[.{4}]")

	relativeDefId = pattern.FindString(relativeDefId)
	relativeDefId = fourCharIdPattern.ReplaceAllString(relativeDefId, "")

	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		return cleanPath(typeHierarchy.Types[implementation.TypeId].StringId), nil
	}
	return "UNKNOWN", errors.New("no type found")
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait
func (typeHierarchy MapTypeHierarchy) getTraitFromTypeHierarchy(relativeDefId string) string {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	fourCharIdPattern := regexp.MustCompile("\\[.{4}]")
	relativeDefId = pattern.FindString(relativeDefId)
	relativeDefId = fourCharIdPattern.ReplaceAllString(relativeDefId, "")
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

	// TODO: check if including crate is necessary
	path := "/" + elements[0]
	for _, elem := range elements[1 : len(elements)-1] {
		path += "." + elem
	}
	path += "/" + elements[len(elements)-1]

	return path
}

func cleanPath(path string) string {
	whitespacePattern := regexp.MustCompile("\\s")
	referencePattern := regexp.MustCompile("&")
	mutPattern := regexp.MustCompile("mut")
	pointerPattern := regexp.MustCompile("\\*")
	constPattern := regexp.MustCompile("const")
	dynPattern := regexp.MustCompile("dyn")

	path = whitespacePattern.ReplaceAllString(path, "")
	path = referencePattern.ReplaceAllString(path, "")
	path = mutPattern.ReplaceAllString(path, "")
	path = pointerPattern.ReplaceAllString(path, "")
	path = constPattern.ReplaceAllString(path, "")
	path = dynPattern.ReplaceAllString(path, "")

	return path
}
