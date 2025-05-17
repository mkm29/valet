package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "path/filepath"
   "reflect"
   "debug/buildinfo"

   "gopkg.in/yaml.v3"
)

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

func main() {
   // version flag prints build information and exits
   versionFlag := flag.Bool("version", false, "print version information")
   overridesFlag := flag.String("overrides", "", "path (relative to context dir) to overrides YAML (optional)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s [flags] <context-dir>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

   // if version requested, print build info and exit
   if *versionFlag {
       exe, err := os.Executable()
       if err != nil {
           fmt.Fprintf(os.Stderr, "error retrieving executable path: %v\n", err)
           os.Exit(1)
       }
       info, err := buildinfo.ReadFile(exe)
       if err != nil {
           fmt.Fprintf(os.Stderr, "error reading build info: %v\n", err)
           os.Exit(1)
       }
       // print main module path, version, and VCS revision if available
       revision := ""
       for _, setting := range info.Settings {
           if setting.Key == "vcs.revision" {
               revision = setting.Value
               break
           }
       }
       if revision != "" {
           fmt.Printf("%s@%s (commit %s)\n", info.Main.Path, info.Main.Version, revision)
       } else {
           fmt.Printf("%s@%s\n", info.Main.Path, info.Main.Version)
       }
       os.Exit(0)
   }

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}
	ctx := args[0]

	msg, err := Generate(ctx, *overridesFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Println(msg)
}
