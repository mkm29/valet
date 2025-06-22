package utils

import (
	"reflect"
)

// GetFieldInt extracts an int field from a reflect.Value by field name
func GetFieldInt(v reflect.Value, name string) int {
	f := v.FieldByName(name)
	if f.IsValid() && f.Kind() == reflect.Int {
		return int(f.Int())
	}
	return 0
}

// GetFieldInt64 extracts an int64 field from a reflect.Value by field name
func GetFieldInt64(v reflect.Value, name string) int64 {
	f := v.FieldByName(name)
	if f.IsValid() && f.Kind() == reflect.Int64 {
		return f.Int()
	}
	return 0
}

// GetFieldFloat64 extracts a float64 field from a reflect.Value by field name
func GetFieldFloat64(v reflect.Value, name string) float64 {
	f := v.FieldByName(name)
	if f.IsValid() && f.Kind() == reflect.Float64 {
		return f.Float()
	}
	return 0
}

// IsEmptyValue checks if a value represents an empty value (empty string, array, map)
func IsEmptyValue(val any) bool {
	if val == nil {
		return true
	}

	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case map[interface{}]interface{}:
		return len(v) == 0
	}

	// Use reflection for other types
	rv := reflect.ValueOf(val)

	// Handle nil values
	if !rv.IsValid() || (rv.Kind() == reflect.Ptr && rv.IsNil()) {
		return true
	}

	// Get the underlying value if it's a pointer
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Map {
		return rv.Len() == 0
	} else if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len() == 0
	}

	return false
}
