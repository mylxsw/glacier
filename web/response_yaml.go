package web

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// YAMLResponse yaml响应
type YAMLResponse struct {
	response ResponseCreator
	original interface{}
	code     int
}

func (resp *YAMLResponse) Code() int {
	return resp.code
}

// NewYAMLResponse 创建YAMLResponse对象
func NewYAMLResponse(response ResponseCreator, code int, res interface{}) *YAMLResponse {
	return &YAMLResponse{
		response: response,
		original: res,
		code:     code,
	}
}

// WithCode set response code and return itself
func (resp *YAMLResponse) WithCode(code int) *YAMLResponse {
	resp.code = code
	return resp
}

// CreateResponse create response
func (resp *YAMLResponse) CreateResponse() error {
	res, err := yaml.Marshal(resp.original)
	if err != nil {
		err = fmt.Errorf("yaml encode failed: %v [%v]", err, resp.original)

		return err
	}

	resp.response.SetCode(resp.code)
	resp.response.Header("Content-Type", "application/yaml; charset=utf-8")
	resp.response.SetContent(res)

	resp.response.Flush()
	return nil
}
