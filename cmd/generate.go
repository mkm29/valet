package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// generate subcommand

// deepMerge merges b into a (recursively for nested maps) and returns a new map.
func deepMerge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, vb := range b {
		if va, ok := out[k]; ok {
			ma, maOK := va.(map[string]any)
			mb, mbOK := vb.(map[string]any)
			if maOK && mbOK {
				out[k] = deepMerge(ma, mb)
				continue
			}
		}
		out[k] = vb
	}
	return out
}

// inferSchema builds a JSON‐Schema fragment for val, using defaultVal
// to determine which object keys are “required”.
func inferSchema(val, defaultVal any) map[string]any {
	switch v := val.(type) {
	case map[string]any:
		defMap, _ := defaultVal.(map[string]any)
		props := make(map[string]any, len(v))
		for key, sub := range v {
			props[key] = inferSchema(sub, defMap[key])
		}
		schema := map[string]any{
			"type":       "object",
			"properties": props,
			"default":    map[string]any{},
		}
		var required []string
		for k, vDefault := range defMap {
			// Only add to required if key exists, default is NOT nil and NOT null (in Go or YAML)
			if _, exists := v[k]; exists {
				isNil := vDefault == nil
				isNullString := false
				switch vDefault := vDefault.(type) {
				case string:
					// YAML nulls sometimes decode as string "null" or as empty string
					isNullString = vDefault == "null"
				}
				if !isNil && !isNullString {
					required = append(required, k)
				}
			}
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema

	case []any:
		var defItem any
		if defArr, ok := defaultVal.([]any); ok && len(defArr) > 0 {
			defItem = defArr[0]
		}
		var itemsSchema map[string]any
		if len(v) > 0 {
			itemsSchema = inferSchema(v[0], defItem)
		} else {
			itemsSchema = map[string]any{}
		}
		return map[string]any{
			"type":    "array",
			"items":   itemsSchema,
			"default": []any{},
		}

	case bool:
		return map[string]any{
			"type":    "boolean",
			"default": v,
		}

	case int, int64:
		return map[string]any{
			"type":    "integer",
			"default": v,
		}

	case float64:
		if reflect.DeepEqual(v, float64(int64(v))) {
			return map[string]any{
				"type":    "integer",
				"default": int64(v),
			}
		}
		return map[string]any{
			"type":    "number",
			"default": v,
		}

	case string:
		return map[string]any{
			"type":    "string",
			"default": v,
		}

	default:
		return map[string]any{
			"type":    "string",
			"default": fmt.Sprintf("%v", v),
		}
	}
}

// loadYAML reads a YAML file into map[string]any (empty if missing)
func loadYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// Generate a JSON Schema for the values.yaml in ctx directory,
// optionally merging an overrides YAML file relative to ctx.
// It writes the schema to values.schema.json and returns a status message.
func Generate(ctx, overridesFlag string) (string, error) {
	valuesPath := filepath.Join(ctx, "values.yaml")
	var overridesPath string
	if overridesFlag != "" {
		overridesPath = filepath.Join(ctx, overridesFlag)
	}
	yaml1, err := loadYAML(valuesPath)
	if err != nil {
		return "", fmt.Errorf("error loading %s: %w", valuesPath, err)
	}
	var merged map[string]any
	if overridesPath != "" {
		yaml2, err := loadYAML(overridesPath)
		if err != nil {
			return "", fmt.Errorf("error loading %s: %w", overridesPath, err)
		}
		merged = deepMerge(yaml1, yaml2)
	} else {
		merged = yaml1
	}
	schema := inferSchema(merged, yaml1)
	schema["$schema"] = "http://json-schema.org/schema#"
	outPath := filepath.Join(ctx, "values.schema.json")
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("error writing %s: %w", outPath, err)
	}
	if overridesPath != "" {
		return fmt.Sprintf("Generated %s by merging %s into values.yaml", outPath, overridesFlag), nil
	}
	return fmt.Sprintf("Generated %s from values.yaml", outPath), nil
}

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <context-dir>",
		Short: "Generate JSON Schema from values.yaml",
		Long:  `Generate JSON Schema from values.yaml, optionally merging an overrides YAML file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := args[0]
			overridesFlag, err := cmd.Flags().GetString("overrides")
			if err != nil {
				return err
			}
			msg, err := Generate(ctx, overridesFlag)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	}
	cmd.Flags().StringP("overrides", "f", "", "path (relative to context dir) to overrides YAML (optional)")
	return cmd
}
