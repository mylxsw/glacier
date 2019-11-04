package web

// NilResponse 空响应
type NilResponse struct {
	response ResponseCreator
}

func (resp *NilResponse) Code() int {
	return resp.response.GetCode()
}

// NewNilResponse create a RawResponse
func NewNilResponse(response ResponseCreator) *NilResponse {
	return &NilResponse{response: response}
}

// response get real response object
func (resp *NilResponse) Response() ResponseCreator {
	return resp.response
}

// CreateResponse flush response to client
func (resp *NilResponse) CreateResponse() error {
	return nil
}
