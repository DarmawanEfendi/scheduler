package modules

import (
	"fmt"
	"strings"
)

func validateFuncParams(params map[string]interface{}, funcParams []string) (map[string]interface{}, error) {
	if len(params) != len(funcParams) {
		return nil, fmt.Errorf("number of params must be: %v", len(funcParams))
	}

	for _, funcParam := range funcParams {
		if _, ok := params[funcParam]; !ok {
			return nil, fmt.Errorf("params not exists %v: %v", funcParam, strings.Join(funcParams, ","))
		}

	}
	return params, nil
}

// func convertParamValue(dataType string, dataValue interface{}) (interface{}, error) {
// 	switch dataType {
// 	case "int":
// 		switch dataValue.(type) {
// 		case float64:
// 			val := int(dataValue.(float64))
// 			return val, nil
// 		case string:
// 			val, err := strconv.Atoi(dataValue.(string))
// 			return val, err
// 		case int:
// 			return int(dataValue.(int)), nil
// 		default:
// 			return nil, errors.New("failed")
// 		}
// 	case "string":
// 		return fmt.Sprintf("%v", dataValue), nil
// 	}
// 	return nil, errors.New("failed")
// }
