package proxy

import (
	"reflect"
)

// CleanJSONSchema recursively removes unsupported fields (like additionalProperties, default)
// from a JSON schema for strict providers like Gemini.
func CleanJSONSchema(schema any) any {
	if schema == nil {
		return nil
	}

	// Handle pointer indirection
	val := reflect.ValueOf(schema)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		// We expect schemas to be map[string]any when decoded from JSON
		m, ok := schema.(map[string]any)
		if !ok {
			// If it's a map but not map[string]any, we try to reconstruct it
			newMap := make(map[string]any)
			iter := val.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				v := iter.Value().Interface()

				// Remove unsupported Gemini keys
				if k == "additionalProperties" || k == "default" {
					continue
				}

				// Check for format compatibility
				if k == "format" {
					if s, ok := v.(string); ok {
						allowed := map[string]bool{"enum": true, "date-time": true}
						if !allowed[s] {
							continue
						}
					}
				}

				newMap[k] = CleanJSONSchema(v)
			}
			return newMap
		}

		newMap := make(map[string]any)
		for k, v := range m {
			if k == "additionalProperties" || k == "default" {
				continue
			}

			if k == "format" {
				if s, ok := v.(string); ok {
					allowed := map[string]bool{"enum": true, "date-time": true}
					if !allowed[s] {
						continue
					}
				}
			}

			newMap[k] = CleanJSONSchema(v)
		}
		return newMap

	case reflect.Slice:
		newSlice := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			newSlice[i] = CleanJSONSchema(val.Index(i).Interface())
		}
		return newSlice

	default:
		return schema
	}
}
