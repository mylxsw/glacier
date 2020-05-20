package web

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/mylxsw/container"
)

type Context interface {
	JSON(res interface{}) *JSONResponse
	NewJSONResponse(res interface{}) *JSONResponse
	YAML(res interface{}) *YAMLResponse
	NewYAMLResponse(res interface{}) *YAMLResponse
	JSONWithCode(res interface{}, code int) *JSONResponse
	Nil() *NilResponse
	API(businessCode string, message string, data interface{}) *JSONResponse
	NewAPIResponse(businessCode string, message string, data interface{}) *JSONResponse
	NewRawResponse() *RawResponse
	NewHTMLResponse(res string) *HTMLResponse
	HTML(res string) *HTMLResponse
	HTMLWithCode(res string, code int) *HTMLResponse
	Error(res string, code int) *ErrorResponse
	JSONError(res string, code int) *JSONResponse
	NewErrorResponse(res string, code int) *ErrorResponse
	Redirect(location string, code int) *RedirectResponse
	Resolve(callback interface{}) Response
	Decode(v interface{}) error
	Unmarshal(v interface{}) error
	UnmarshalYAML(v interface{}) error
	PathVar(key string) string
	PathVars() map[string]string
	Input(key string) string
	JSONGet(keys ...string) string
	InputWithDefault(key string, defaultVal string) string
	ToInt(val string, defaultVal int) int
	ToInt64(val string, defaultVal int64) int64
	ToFloat32(val string, defaultVal float32) float32
	ToFloat64(val string, defaultVal float64) float64
	IntInput(key string, defaultVal int) int
	Int64Input(key string, defaultVal int64) int64
	Float32Input(key string, defaultVal float32) float32
	Float64Input(key string, defaultVal float64) float64
	File(key string) (*UploadedFile, error)
	IsXMLHTTPRequest() bool
	AJAX() bool
	IsJSON() bool
	ContentType() string
	AllHeaders() http.Header
	Headers(key string) []string
	Header(key string) string
	Is(method string) bool
	IsGet() bool
	IsPost() bool
	IsHead() bool
	IsDelete() bool
	IsPut() bool
	IsPatch() bool
	IsOptions() bool
	Method() string
	Body() []byte
	Set(key string, value interface{})
	Get(key string) interface{}
	Context() context.Context
	Cookie(name string) (*http.Cookie, error)
	Cookies() []*http.Cookie
	Session() *sessions.Session
	Request() Request
	Response() ResponseCreator
	Container() container.Container
	View(tplPath string, data interface{}) *HTMLResponse
	Validate(validator Validator, jsonResponse bool)
}

type Request interface {
	Raw() *http.Request
	Decode(v interface{}) error
	Unmarshal(v interface{}) error
	UnmarshalYAML(v interface{}) error
	PathVar(key string) string
	PathVars() map[string]string
	Input(key string) string
	JSONGet(keys ...string) string
	InputWithDefault(key string, defaultVal string) string
	ToInt(val string, defaultVal int) int
	ToInt64(val string, defaultVal int64) int64
	ToFloat32(val string, defaultVal float32) float32
	ToFloat64(val string, defaultVal float64) float64
	IntInput(key string, defaultVal int) int
	Int64Input(key string, defaultVal int64) int64
	Float32Input(key string, defaultVal float32) float32
	Float64Input(key string, defaultVal float64) float64
	File(key string) (*UploadedFile, error)
	IsXMLHTTPRequest() bool
	AJAX() bool
	IsJSON() bool
	ContentType() string
	AllHeaders() http.Header
	Headers(key string) []string
	Header(key string) string
	Is(method string) bool
	IsGet() bool
	IsPost() bool
	IsHead() bool
	IsDelete() bool
	IsPut() bool
	IsPatch() bool
	IsOptions() bool
	Method() string
	Body() []byte
	Set(key string, value interface{})
	Get(key string) interface{}

	Context() context.Context
	Cookie(name string) (*http.Cookie, error)
	Cookies() []*http.Cookie
	Session() *sessions.Session
	SetSession(session *sessions.Session)

	Validate(validator Validator, jsonResponse bool)
}

// Response is the response interface
type Response interface {
	CreateResponse() error
	Code() int
}

// Controller is a interface for controller
type Controller interface {
	// Register register routes for a controller
	Register(router *Router)
}

// ResponseCreator is a response creator
type ResponseCreator interface {
	Raw() http.ResponseWriter
	SetCode(code int)
	ResponseWriter() http.ResponseWriter
	SetContent(content []byte)
	Header(key, value string)
	Cookie(cookie *http.Cookie)
	GetCode() int
	Flush()
}
