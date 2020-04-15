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

// Convert data of type hierarchy read from json to a map for simplifying queries.
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

// Converts a relativeDefId to the path in Fasten format.
func (typeHierarchy MapTypeHierarchy) getFullPath(relativeDefId string) (string, error) {
	var err error
	modules, impl, nestedElements, method, err := typeHierarchy.parseRelativeDefPath(relativeDefId)

	fullPath := "/"
	fullPath += strings.Join(modules, ".")

	if strings.Contains(impl, "[") {
		patternBrackets := regexp.MustCompile("\\[(.*?)\\]")
		index := patternBrackets.FindAllIndex([]byte(impl), -1)
		if len(index) > 1 {
			implElements := strings.Split(impl, "::")
			lastElement := implElements[len(implElements)-1]
			impl = patternBrackets.ReplaceAllString(lastElement, "")
			impl = "[" + impl[:len(impl)-1] + "]"
		}

		index = patternBrackets.FindAllIndex([]byte(impl), -1)
		for i := 0; i < len(index); i++ {
			insideBrackets := impl[index[i][0]+1 : index[i][1]-1]
			if strings.Contains(insideBrackets, "generic") {
				patternColon := regexp.MustCompile(":")
				indices := patternColon.FindAllIndex([]byte(insideBrackets), -1)
				for _, genericIndex := range indices {
					insideBrackets = insideBrackets[:genericIndex[0]] + "[]" + insideBrackets[genericIndex[0]:]
				}
				impl = impl[:index[i][0]] + insideBrackets + impl[index[i][1]:]
			} else {
				impl = impl[:index[i][0]] + insideBrackets + "[]" + impl[index[i][1]:]
			}
		}
	}
	impl = url.PathEscape(impl)
	impl = strings.ReplaceAll(impl, ":", "%3A")
	impl = strings.ReplaceAll(impl, "&", "%26")
	fullPath += "/" + impl

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

// Parses relativeDefId and returns a tuple containing slice of modules,
// resolved type name, nested functions and types, function name.
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
	if len(modules) == 0 {
		modules = append(modules, "EMPTY-NAMESPACE")
	}
 	return modules, impl, nestedElements, function, err
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait.
func (typeHierarchy MapTypeHierarchy) getTypeFromTypeHierarchy(relativeDefId string) (string, error) {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	fourCharIdPattern := regexp.MustCompile("\\[.{4}]")

	relativeDefId = pattern.FindString(relativeDefId)
	relativeDefId = fourCharIdPattern.ReplaceAllString(relativeDefId, "")

	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		return typeHierarchy.Types[implementation.TypeId].StringId, nil
	}
	return "UNKNOWN", errors.New("no type found")
}

// When {{impl}}[id] is present in the relativeDefPath finds the respective implementation
// in the list of Impls inside the type hierarchy. Returns the respective Type and Trait.
func (typeHierarchy MapTypeHierarchy) getTraitFromTypeHierarchy(relativeDefId string) string {
	pattern := regexp.MustCompile("^.*{{impl}}\\[[0-9]*]")
	fourCharIdPattern := regexp.MustCompile("\\[.{4}]")
	relativeDefId = pattern.FindString(relativeDefId)
	relativeDefId = fourCharIdPattern.ReplaceAllString(relativeDefId, "")
	if implementation, ok := typeHierarchy.Impls[relativeDefId]; ok {
		if implementation.TraitId != 0 {
			id := implementation.TraitId
			return typeHierarchy.getTraitPath(typeHierarchy.Traits[id-typeHierarchy.Traits[0].Id].RelativeDefId)
		}
	}
	return ""
}

// Extract the namespace from the full type info by removing the function name
// at the end.
func getNamespace(method string) string {
	index := strings.LastIndex(method, ".")
	return method[:index]
}

// Convert relativeDefId of a Trait to Fasten format.
func (typeHierarchy MapTypeHierarchy) getTraitPath(relativeDefId string) string {
	fullPath, _ := typeHierarchy.getFullPath(relativeDefId)
	fullPath = fullPath[:len(fullPath) - 2]

	if strings.Contains(fullPath, "NO-TYPE-DEFINITION.") {
		fullPath = strings.ReplaceAll(fullPath, "NO-TYPE-DEFINITION.", "")
	} else {
		lastDot := strings.LastIndex(fullPath, ".")
		fullPath = fullPath[:lastDot] + "$" + fullPath[lastDot + 1:]
	}

	return fullPath
}

// Check if the given RelativeDefId contains generic types.
func (typeHierarchy MapTypeHierarchy) isGenericType(relativeDefId string) bool {
	rawElements := strings.Split(relativeDefId, "::")
	length := 0
	for _, elem := range rawElements {
		length++
		if strings.Contains(elem, "{{impl}}") {
			currentRelativeDefId := strings.Join(rawElements[:length+1], "::")
			resolvedType, _ := typeHierarchy.getTypeFromTypeHierarchy(currentRelativeDefId)
			if len(resolvedType) > 2 && resolvedType[:1] == "(" {
				return true
			}
		}
	}
	return false
}

// Converts a path containing generic types to a slice of
// paths each containing one generic type.
func (typeHierarchy MapTypeHierarchy) getGenericFullPaths(fullPath string) []string {
	var types []string
	implPattern := regexp.MustCompile("(/|\\$)%28.+?%29")
	resolvedGenericTypes := implPattern.FindAllString(fullPath, -1)
	resolvedGenericTypesIndices := implPattern.FindAllIndex([]byte(fullPath), -1)

	if len(resolvedGenericTypes) == 0 {
		return []string{fullPath}
	}

	index := resolvedGenericTypesIndices[len(resolvedGenericTypesIndices)-1]
	alreadyResolvedPath := fullPath[index[1]:]
	symbol := resolvedGenericTypes[len(resolvedGenericTypes)-1][:1]
	resolved, _ := url.PathUnescape(resolvedGenericTypes[len(resolvedGenericTypes)-1][1:])
	resolved = strings.ReplaceAll(resolved, "%3A", ":")
	resolved = strings.ReplaceAll(resolved, "%26", "&")
	genericTypes := strings.Split(resolved[1:len(resolved)-3], ",")

	for _, genericType := range genericTypes {
		genericType = strings.TrimSpace(genericType)
		genericPath := fullPath[:index[0]] + symbol + url.PathEscape(genericType)
		// Manual encoding, because : and & and not ignored by url.PathEncode
		genericPath = strings.ReplaceAll(genericPath, ":", "%3A")
		genericPath = strings.ReplaceAll(genericPath, "&", "%26")

		resolvedGenericPath := typeHierarchy.getGenericFullPaths(genericPath)
		for _, path := range resolvedGenericPath {
			types = append(types, path+alreadyResolvedPath)
		}
	}
	return types
}
