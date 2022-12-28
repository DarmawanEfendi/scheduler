package httputil

import (
	"encoding/json"
	"net/http"
)

func ReturnSuccess(w http.ResponseWriter, code int, data interface{}, meta map[string]interface{}) {
	success := SuccessObject{
		Meta: meta,
		Data: data,
	}
	obj, _ := json.Marshal(success)
	w.Header().Set("content-type", "application/json")
	w.Write(obj)
}

func ReturnError(w http.ResponseWriter, code int, message string) {
	errorObj := ErrorObject{
		Code:    code,
		Message: message,
	}
	obj, _ := json.Marshal(errorObj)
	w.Header().Set("content-type", "application/json")
	w.Write(obj)
}
