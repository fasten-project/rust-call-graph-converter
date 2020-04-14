package fasten

import (
	"encoding/json"
	"strconv"
)

type JSON struct {
	Product   string          `json:"product"`
	Forge     string          `json:"forge"`
	Generator string          `json:"generator"`
	Depset    [][]Dependency  `json:"depset"`
	Version   string          `json:"version"`
	Cha       map[string]Type `json:"cha"`
	Graph     CallGraph       `json:"graph"`
	Timestamp int64           `json:"timestamp"`

	Counter int64 `json:"-"`
}

type Dependency struct {
	Product     string   `json:"product"`
	Forge       string   `json:"forge"`
	Constraints []string `json:"constraints,nilasempty"`
}

type Type struct {
	Methods         map[int64]string `json:"methods"`
	SuperInterfaces []string         `json:"superInterfaces"`
	SourceFile      string           `json:"sourceFile"`
	SuperClasses    []string         `json:"superClasses,nilasempty"`
}

type CallGraph struct {
	InternalCalls [][]int64       `json:"internalCalls"`
	ExternalCalls [][]interface{} `json:"externalCalls"`
}

// Converts this fastenJSON type to JSON format
func (fastenJSON *JSON) ToJSON() []byte {
	fasten, _ := json.Marshal(fastenJSON)
	return fasten
}

// Checks if this json has empty product or empty call graph
// Returns true if at least one of these parameters is empty
func (fastenJSON *JSON) IsEmpty() bool {
	return fastenJSON.Graph.InternalCalls == nil &&
		fastenJSON.Graph.ExternalCalls == nil
}

// Adds a dependency too the current JSON depset.
func (fastenJSON *JSON) AddDependency(target *JSON) {
	if target.Product == "" {
		return
	}

	for _, inner := range fastenJSON.Depset {
		for _, dependency := range inner {
			if dependency.Product == target.Product {
				found := false
				if target.Version == "" {
					return
				}
				for _, constraint := range dependency.Constraints {
					 if constraint == target.Version {
						found = true
						break
					}
				}
				if found {
					return
				} else if target.Version != "" {
					return
				}
			}
		}
	}
	if len(fastenJSON.Depset) == 0 {
		fastenJSON.Depset = append(fastenJSON.Depset, []Dependency{})
	}
	fastenJSON.Depset[0] = append(fastenJSON.Depset[0], Dependency{
		Product:     target.Product,
		Forge:       "cratesio",
		Constraints: []string{"[" + target.Version + "]"},
	})
}

// Add method to Class Hierarchy.
func (fastenJSON *JSON) AddMethodToCHA(namespace string, methodName string) int64 {
	if methodName == "" {
		return -1
	}

	fastenJSON.initializeCHANamespace(namespace)

	fastenJSON.Cha[namespace].Methods[fastenJSON.Counter] = methodName
	fastenJSON.Counter++

	return fastenJSON.Counter - 1
}

// Add interface to Class Hierarchy.
func (fastenJSON *JSON) AddInterfaceToCHA(namespace string, interfaceName string) {
	if interfaceName == "" {
		return
	}

	fastenJSON.initializeCHANamespace(namespace)

	typeValue := fastenJSON.Cha[namespace]
	for _, trait := range typeValue.SuperInterfaces {
		if trait == interfaceName {
			return
		}
	}
	typeValue.SuperInterfaces = append(typeValue.SuperInterfaces, interfaceName)
	fastenJSON.Cha[namespace] = typeValue
}

// Add filename to Class Hierarchy.
func (fastenJSON *JSON) AddFilenameToCHA(namespace string, filename string) {
	if filename == "" {
		return
	}

	fastenJSON.initializeCHANamespace(namespace)

	typeValue := fastenJSON.Cha[namespace]
	typeValue.SourceFile = filename
	fastenJSON.Cha[namespace] = typeValue
}

// Create a new instance in the class hierarchy map of ra give namespace if not yet present
func (fastenJSON *JSON) initializeCHANamespace(namespace string) {
	if _, exists := fastenJSON.Cha[namespace]; !exists {
		fastenJSON.Cha[namespace] = Type{
			Methods:         map[int64]string{},
			SuperInterfaces: []string{},
			SourceFile:      "",
			SuperClasses:    make([]string, 0),
		}
	}
}

// Add internal call to the Graph.
func (fastenJSON *JSON) AddInternalCall(sourceId int64, targetId int64) {
	fastenJSON.Graph.InternalCalls = append(fastenJSON.Graph.InternalCalls, []int64{sourceId, targetId})
}

// Add external call to the Graph.
func (fastenJSON *JSON) AddExternalCall(sourceId int64, target string) {
	fastenJSON.Graph.ExternalCalls = append(fastenJSON.Graph.ExternalCalls, []interface{}{strconv.FormatInt(sourceId, 10), target, map[string]string{}})
}
