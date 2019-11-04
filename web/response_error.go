package web

// ErrorResponse Error response
type ErrorResponse struct {
	response ResponseCreator
	original string
	code     int
}

func (resp *ErrorResponse) Code() int {
	return resp.code
}

// NewErrorResponse Create error response
func NewErrorResponse(response ResponseCreator, res string, code int) *ErrorResponse {
	return &ErrorResponse{
		response: response,
		original: res,
		code:     code,
	}
}

// WithCode set response code and return itself
func (resp *ErrorResponse) WithCode(code int) *ErrorResponse {
	resp.code = code
	return resp
}

// CreateResponse 创建响应内容
func (resp *ErrorResponse) CreateResponse() error {
	resp.response.SetCode(resp.code)
	resp.response.SetContent([]byte(resp.original))

	resp.response.Flush()
	return nil
}
