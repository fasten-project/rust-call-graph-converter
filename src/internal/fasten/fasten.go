package fasten

import "encoding/json"

type JSON struct {
	Product   string          `json:"product,omitempty"`
	Forge     string          `json:"forge,omitempty"`
	Generator string          `json:"generator,omitempty"`
	Depset    [][]Dependency  `json:"depset,omitempty"`
	Version   string          `json:"version,omitempty"`
	Cha       map[string]Type `json:"cha,omitempty"`
	Graph     CallGraph       `json:"graph,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
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
func (fastenJSON JSON) ToJSON() []byte {
	fasten, _ := json.Marshal(fastenJSON)
	return fasten
}

// Checks if this json has empty product or empty call graph
// Returns true if at least one of these parameters is empty
func (fastenJSON JSON) IsEmpty() bool {
	return fastenJSON.Graph.InternalCalls == nil &&
		fastenJSON.Graph.ExternalCalls == nil
}
