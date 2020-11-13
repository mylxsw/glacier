package web

import "net/http"

// RawResponse 原生响应
type RawResponse struct {
	response ResponseCreator
	handler  func(w http.ResponseWriter)
}

func (resp *RawResponse) Code() int {
	return resp.response.GetCode()
}

// NewRawResponse create a RawResponse
func NewRawResponse(response ResponseCreator, handler func(w http.ResponseWriter)) *RawResponse {
	return &RawResponse{response: response, handler: handler}
}

// response get real response object
func (resp *RawResponse) Response() ResponseCreator {
	return resp.response
}

// CreateResponse flush response to client
func (resp *RawResponse) CreateResponse() error {
	resp.handler(resp.response.Raw())
	return nil
}
