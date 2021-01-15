package terraform

// This code follows: https://github.com/gruntwork-io/terratest/blob/master/modules/terraform/output.go

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// OutputAll calls terraform and returns all the outputs as a map
func OutputAll(options *Options) (map[string]interface{}, error) {
	return OutputForKeysE(options, nil)
}

// OutputForKeysE calls terraform output for the given key list and returns values as a map.
// The returned values are of type interface{} and need to be type casted as necessary. Refer to output_test.go
func OutputForKeysE(options *Options, keys []string) (map[string]interface{}, error) {
	out, err := RunTerraformCommand(false, options, "output", "-no-color", "-json")
	if err != nil {
		return nil, err
	}

	outputMap := map[string]map[string]interface{}{}
	if err := json.Unmarshal([]byte(out), &outputMap); err != nil {
		return nil, err
	}

	if keys == nil {
		outputKeys := make([]string, 0, len(outputMap))
		for k := range outputMap {
			outputKeys = append(outputKeys, k)
		}
		keys = outputKeys
	}

	resultMap := make(map[string]interface{})
	for _, key := range keys {
		value, containsValue := outputMap[key]["value"]
		if !containsValue {
			return nil, OutputKeyNotFound(string(key))
		}
		resultMap[key] = value
	}
	return resultMap, nil
}

// OutputToString converts the value into a string representation since
// there is no easy win for string representations for e.g. lists or maps in go.
// This will work for now but starts to break down
// with complex types, e.g. list of maps.
func OutputToString(value interface{}) string {
	var converted string
	defaultFormat := fmt.Sprintf("%v", value)
	valueType := reflect.TypeOf(value).String() // Get the type as a string, e.g. []interface {} for an array

	if strings.HasPrefix(valueType, "map") || strings.HasPrefix(valueType, "[]") {
		j, _ := json.Marshal(value)
		converted = string(j)
	} else {
		converted = defaultFormat
	}

	return converted
}

// Gets the first and last index of the first and last substrings
func getFirstAndLastIndex(s string, firstSubstr string, lastSubstr string) (first int, last int) {
	first = strings.Index(s, firstSubstr)
	last = strings.LastIndex(s, lastSubstr)
	return
}
