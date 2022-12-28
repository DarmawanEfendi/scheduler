package httputil

type SuccessObject struct {
	Meta map[string]interface{} `json:"meta,omitempty"`
	Data interface{}            `json:"data"`
}

type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
