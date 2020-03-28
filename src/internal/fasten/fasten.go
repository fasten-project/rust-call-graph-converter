package fasten

import (
	"encoding/json"
)

type JSON struct {
	Product   string          `json:"product,omitempty"`
	Forge     string          `json:"forge,omitempty"`
	Generator string          `json:"generator,omitempty"`
	Depset    [][]Dependency  `json:"depset,omitempty"`
	Version   string          `json:"version,omitempty"`
	Cha       map[string]Type `json:"cha,omitempty"`
	Graph     CallGraph       `json:"graph,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`

	Counter int64 `json:"-"`
}

type Dependency struct {
	Product     string   `json:"product,omitempty"`
	Forge       string   `json:"forge,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

type Type struct {
	Methods         map[int64]string `json:"methods,omitempty"`
	SuperInterfaces []string         `json:"superInterfaces,omitempty"`
	SourceFile      string           `json:"sourceFile,omitempty"`
	SuperClasses    []string         `json:"superClasses,omitempty"`
}

type CallGraph struct {
	InternalCalls [][]int64       `json:"internalCalls,omitempty"`
	ExternalCalls [][]interface{} `json:"externalCalls,omitempty"`
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
			if dependency.Product == target.Product &&
				dependency.Constraints[0] == target.Version {
				return
			}
		}
	}
	if len(fastenJSON.Depset) == 0 {
		fastenJSON.Depset = append(fastenJSON.Depset, []Dependency{})
	}
	fastenJSON.Depset[0] = append(fastenJSON.Depset[0], Dependency{
		Product:     target.Product,
		Forge:       "cratesio",
		Constraints: []string{target.Version},
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
	typeValue.SuperInterfaces = append(typeValue.SuperInterfaces, interfaceName)
	fastenJSON.Cha[namespace] = typeValue
}

// Create a new instance in the class hierarchy map of ra give namespace if not yet present
func (fastenJSON *JSON) initializeCHANamespace(namespace string) {
	if _, exists := fastenJSON.Cha[namespace]; !exists {
		fastenJSON.Cha[namespace] = Type{
			Methods:         map[int64]string{},
			SuperInterfaces: []string{},
			SourceFile:      "",
			SuperClasses:    []string{},
		}
	}
}

// Add internal call to the Graph.
func (fastenJSON *JSON) AddInternalCall(sourceId int64, targetId int64) {
	fastenJSON.Graph.InternalCalls = append(fastenJSON.Graph.InternalCalls, []int64{sourceId, targetId})
}

// Add external call to the Graph.
func (fastenJSON *JSON) AddExternalCall(sourceId int64, target string) {
	fastenJSON.Graph.ExternalCalls = append(fastenJSON.Graph.ExternalCalls, []interface{}{sourceId, target})
}
