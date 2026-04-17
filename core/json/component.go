package json

// ComponentInstanceConfig represents a component instance in JSON.
type ComponentInstanceConfig struct {
	Kind string                 `json:"kind"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}