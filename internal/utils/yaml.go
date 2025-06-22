package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// DeepMerge merges b into a (recursively for nested maps) and returns a new map.
func DeepMerge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, vb := range b {
		if va, ok := out[k]; ok {
			ma, maOK := va.(map[string]any)
			mb, mbOK := vb.(map[string]any)
			if maOK && mbOK {
				out[k] = DeepMerge(ma, mb)
				continue
			}
		}
		out[k] = vb
	}
	return out
}

// ConvertToStringKeyMap recursively converts map[interface{}]interface{} to map[string]interface{}
func ConvertToStringKeyMap(m interface{}) interface{} {
	switch x := m.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range x {
			result[fmt.Sprintf("%v", k)] = ConvertToStringKeyMap(v)
		}
		return result
	case []interface{}:
		for i, v := range x {
			x[i] = ConvertToStringKeyMap(v)
		}
	}
	return m
}

// LoadYAML reads a YAML file into map[string]any (empty if missing)
func LoadYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[interface{}]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return map[string]any{}, nil
	}

	// Convert to map[string]any
	result := ConvertToStringKeyMap(m).(map[string]interface{})
	return result, nil
}
