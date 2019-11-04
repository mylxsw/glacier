package web

// RawResponse 原生响应
type RawResponse struct {
	response ResponseCreator
}

func (resp *RawResponse) Code() int {
	return resp.response.GetCode()
}

// NewRawResponse create a RawResponse
func NewRawResponse(response ResponseCreator) *RawResponse {
	return &RawResponse{response: response}
}

// response get real response object
func (resp *RawResponse) Response() ResponseCreator {
	return resp.response
}

// CreateResponse flush response to client
func (resp *RawResponse) CreateResponse() error {
	resp.response.Flush()
	return nil
}
