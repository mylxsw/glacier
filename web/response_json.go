package web

import (
	"encoding/json"
	"fmt"
)

// JSONResponse json响应
type JSONResponse struct {
	response ResponseCreator
	original interface{}
	code     int
}

func (resp *JSONResponse) Code() int {
	return resp.code
}

// NewJSONResponse 创建JSONResponse对象
func NewJSONResponse(response ResponseCreator, code int, res interface{}) *JSONResponse {
	return &JSONResponse{
		response: response,
		original: res,
		code:     code,
	}
}

// WithCode set response code and return itself
func (resp *JSONResponse) WithCode(code int) *JSONResponse {
	resp.code = code
	return resp
}

// CreateResponse create response
func (resp *JSONResponse) CreateResponse() error {
	res, err := json.Marshal(resp.original)
	if err != nil {
		err = fmt.Errorf("json encode failed: %v [%v]", err, resp.original)

		return err
	}

	resp.response.SetCode(resp.code)
	resp.response.Header("Content-Type", "application/json; charset=utf-8")
	resp.response.SetContent(res)

	resp.response.Flush()
	return nil
}
