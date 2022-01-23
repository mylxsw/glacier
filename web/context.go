package web

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/mylxsw/container"
	"github.com/pkg/errors"
)

// WebContext 作为一个web请求的上下文信息
type WebContext struct {
	response *HttpResponse
	request  *HttpRequest
	cc       container.Container
	conf     Config
}

type webHandler struct {
	handle    WebHandler
	container container.Container
	router    *routerImpl
	conf      *Config
}

// WebHandler 控制器方法
type WebHandler func(context Context) Response

// newWebHandler 创建一个WebHandler，用于传递给Router
func newWebHandler(router *routerImpl, handler WebHandler, decors ...HandlerDecorator) webHandler {
	for i := range decors {
		d := decors[len(decors)-i-1]
		handler = d(handler)
	}

	cc := router.container
	return webHandler{
		handle:    handler,
		container: cc,
		router:    router,
		conf:      cc.MustGet(&Config{}).(*Config),
	}
}

// ServeHTTP 实现http.HandlerFunc接口
func (h webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	_ = r.Body.Close()
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	ctx := &WebContext{
		response: &HttpResponse{
			w:       w,
			headers: make(map[string]string),
		},
		request: &HttpRequest{r: r, body: body, cc: h.container, conf: *h.conf, router: h.router},
		cc:      h.container,
		conf:    *h.conf,
	}

	resp := h.handle(ctx)
	if resp != nil {
		_ = resp.CreateResponse()
	}
}

// Request return underlying Request
func (ctx *WebContext) Request() Request {
	return ctx.request
}

// Response return underlying Response
func (ctx *WebContext) Response() ResponseCreator {
	return ctx.response
}

// Container return underlying container.Container
func (ctx *WebContext) Container() container.Container {
	return ctx.cc
}

// JSON is a shortcut for NewJSONResponse func
func (ctx *WebContext) JSON(res interface{}) *JSONResponse {
	return ctx.NewJSONResponse(res)
}

// NewJSONResponse create a new JSONResponse with the http status code equal 200
func (ctx *WebContext) NewJSONResponse(res interface{}) *JSONResponse {
	return NewJSONResponse(ctx.response, http.StatusOK, res)
}

// YAML is a shortcut for NewYAMLResponse func
func (ctx *WebContext) YAML(res interface{}) *YAMLResponse {
	return ctx.NewYAMLResponse(res)
}

// NewYAMLResponse create a new YAMLResponse with http status code equal 200
func (ctx *WebContext) NewYAMLResponse(res interface{}) *YAMLResponse {
	return NewYAMLResponse(ctx.response, http.StatusOK, res)
}

// JSONWithCode create a json response with a http status code
func (ctx *WebContext) JSONWithCode(res interface{}, code int) *JSONResponse {
	return NewJSONResponse(ctx.response, code, res)
}

// Nil return a NilResponse
func (ctx *WebContext) Nil() *NilResponse {
	return NewNilResponse(ctx.response)
}

// API is a shortcut for NewAPIResponse func
func (ctx *WebContext) API(businessCode string, message string, data interface{}) *JSONResponse {
	return ctx.NewAPIResponse(businessCode, message, data)
}

// NewAPIResponse create a new APIResponse
func (ctx *WebContext) NewAPIResponse(businessCode string, message string, data interface{}) *JSONResponse {
	return ctx.NewJSONResponse(struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}{
		Code:    businessCode,
		Message: message,
		Data:    data,
	})
}

// NewRawResponse create a new RawResponse
func (ctx *WebContext) NewRawResponse(handler func(w http.ResponseWriter)) *RawResponse {
	return NewRawResponse(ctx.response, handler)
}

// Raw create a new RawResponse
func (ctx *WebContext) Raw(handler func(w http.ResponseWriter)) *RawResponse {
	return NewRawResponse(ctx.response, handler)
}

// NewHTMLResponse create a new HTMLResponse
func (ctx *WebContext) NewHTMLResponse(res string) *HTMLResponse {
	return NewHTMLResponse(ctx.response, http.StatusOK, res)
}

// HTML is a shortcut for NewHTMLResponse func
func (ctx *WebContext) HTML(res string) *HTMLResponse {
	return ctx.NewHTMLResponse(res)
}

// HTMLWithCode create a HTMLResponse with http status code
func (ctx *WebContext) HTMLWithCode(res string, code int) *HTMLResponse {
	return NewHTMLResponse(ctx.response, code, res)
}

// Error is a shortcut for NewErrorResponse func
func (ctx *WebContext) Error(res string, code int) *ErrorResponse {
	return ctx.NewErrorResponse(res, code)
}

// JSONError return a json error response for error
func (ctx *WebContext) JSONError(res string, code int) *JSONResponse {
	return ctx.JSONWithCode(M{"error": res}, code)
}

// NewErrorResponse create a error response
func (ctx *WebContext) NewErrorResponse(res string, code int) *ErrorResponse {
	return NewErrorResponse(ctx.response, res, code)
}

// Redirect replies to the request with a redirect to url
func (ctx *WebContext) Redirect(location string, code int) *RedirectResponse {
	return NewRedirectResponse(ctx.response, ctx.request, location, code)
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocket struct {
	WS    *websocket.Conn
	Error error
}

// Resolve resolve implements dependency injection for http handler
func (ctx *WebContext) Resolve(callback interface{}) Response {
	ctxFunc := func() *WebContext { return ctx }
	ctxFuncInterface := func() Context { return ctx }
	requestFunc := func() *HttpRequest { return ctx.request }
	requestFuncInterface := func() Request { return ctx.request }
	wsFunc := func() *WebSocket {
		ws, err := Upgrader.Upgrade(ctx.response.ResponseWriter(), ctx.request.Raw(), nil)
		return &WebSocket{
			WS:    ws,
			Error: err,
		}
	}
	results, err := ctx.cc.CallWithProvider(callback, ctx.cc.Provider(ctxFunc, ctxFuncInterface, requestFunc, requestFuncInterface, wsFunc))
	if err != nil {
		return ctx.NewErrorResponse(
			fmt.Sprintf("resolve dependency error: %s", err.Error()),
			http.StatusInternalServerError,
		)
	}

	if len(results) == 0 {
		return ctx.Nil()
	}

	if len(results) > 1 {
		if err, ok := results[1].(error); ok {
			if err != nil {
				panic(err)
			}
		}
	}

	switch results[0].(type) {
	case Response:
		return results[0].(Response)
	case string:
		return ctx.NewHTMLResponse(results[0].(string))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return ctx.NewHTMLResponse(fmt.Sprintf("%d", results[0]))
	case float32, float64:
		return ctx.NewHTMLResponse(fmt.Sprintf("%f", results[0]))
	case error:
		if results[0] == nil {
			return ctx.HTML("")
		}

		panic(results[0])
	default:
		if jsonAble, ok := results[0].(JSONAble); ok {
			return ctx.NewJSONResponse(jsonAble.ToJSON())
		}

		return ctx.NewJSONResponse(results[0])
	}
}

// Unmarshal is a proxy to request.Unmarshal
func (ctx *WebContext) Unmarshal(v interface{}) error {
	return ctx.request.Unmarshal(v)
}

// UnmarshalYAML is a proxy to request.UnmarshalYAML
func (ctx *WebContext) UnmarshalYAML(v interface{}) error {
	return ctx.request.UnmarshalYAML(v)
}

// PathVar is a proxy to request.PathVar
func (ctx *WebContext) PathVar(key string) string {
	return ctx.request.PathVar(key)
}

// PathVars is a proxy to request.PathVars
func (ctx *WebContext) PathVars() map[string]string {
	return ctx.request.PathVars()
}

// Input is a proxy to request.Input
func (ctx *WebContext) Input(key string) string {
	return ctx.request.Input(key)
}

// JSONGet is a proxy to request.JSONGet
func (ctx *WebContext) JSONGet(keys ...string) string {
	return ctx.request.JSONGet(keys...)
}

// InputWithDefault is a proxy to request.InputWithDefault
func (ctx *WebContext) InputWithDefault(key string, defaultVal string) string {
	return ctx.request.InputWithDefault(key, defaultVal)
}

// ToInt is a proxy to request.ToInt
func (ctx *WebContext) ToInt(val string, defaultVal int) int {
	return ctx.request.ToInt(val, defaultVal)
}

// ToInt64 is a proxy to request.ToInt64
func (ctx *WebContext) ToInt64(val string, defaultVal int64) int64 {
	return ctx.request.ToInt64(val, defaultVal)
}

// ToFloat32 is a proxy to request.ToFloat32
func (ctx *WebContext) ToFloat32(val string, defaultVal float32) float32 {
	return ctx.request.ToFloat32(val, defaultVal)
}

// ToFloat64 is a proxy to request.ToFloat64
func (ctx *WebContext) ToFloat64(val string, defaultVal float64) float64 {
	return ctx.request.ToFloat64(val, defaultVal)
}

// IntInput is a proxy to request.IntInput
func (ctx *WebContext) IntInput(key string, defaultVal int) int {
	return ctx.request.IntInput(key, defaultVal)
}

// Int64Input is a proxy to request.Int64Input
func (ctx *WebContext) Int64Input(key string, defaultVal int64) int64 {
	return ctx.request.Int64Input(key, defaultVal)
}

// Float32Input is a proxy to request.Float32Input
func (ctx *WebContext) Float32Input(key string, defaultVal float32) float32 {
	return ctx.request.Float32Input(key, defaultVal)
}

// Float64Input is a proxy to request.Float64Input
func (ctx *WebContext) Float64Input(key string, defaultVal float64) float64 {
	return ctx.request.Float64Input(key, defaultVal)
}

// File is a proxy to request.File
func (ctx *WebContext) File(key string) (*UploadedFile, error) {
	return ctx.request.File(key)
}

// IsXMLHTTPRequest is a proxy to IsXMLHTTPRequest
func (ctx *WebContext) IsXMLHTTPRequest() bool {
	return ctx.request.IsXMLHTTPRequest()
}

// AJAX is a proxy to request.AJAX
func (ctx *WebContext) AJAX() bool {
	return ctx.request.AJAX()
}

// IsJSON is a proxy to request.IsJSON
func (ctx *WebContext) IsJSON() bool {
	return ctx.request.IsJSON()
}

// ContentType is a proxy to request.ContentType
func (ctx *WebContext) ContentType() string {
	return ctx.request.ContentType()
}

// AllHeaders is a proxy to request.AllHeaders
func (ctx *WebContext) AllHeaders() http.Header {
	return ctx.request.AllHeaders()
}

// Headers is a proxy to request.Headers
func (ctx *WebContext) Headers(key string) []string {
	return ctx.request.Headers(key)
}

// Header is a proxy to request.Header
func (ctx *WebContext) Header(key string) string {
	return ctx.request.Header(key)
}

// Is is a proxy to request.Is
func (ctx *WebContext) Is(method string) bool {
	return ctx.request.Is(method)
}

// IsGet is a proxy to request.IsGet
func (ctx *WebContext) IsGet() bool {
	return ctx.request.IsGet()
}

// IsPost is a proxy to request.IsPost
func (ctx *WebContext) IsPost() bool {
	return ctx.request.IsPost()
}

// IsHead is a proxy to request.IsHead
func (ctx *WebContext) IsHead() bool {
	return ctx.request.IsHead()
}

// IsDelete is a proxy to request.IsDelete
func (ctx *WebContext) IsDelete() bool {
	return ctx.request.IsDelete()
}

// IsPut is a proxy to request.IsPut
func (ctx *WebContext) IsPut() bool {
	return ctx.request.IsPut()
}

// IsPatch is a proxy to request.IsPatch
func (ctx *WebContext) IsPatch() bool {
	return ctx.request.IsPatch()
}

// IsOptions is a proxy to request.IsOptions
func (ctx *WebContext) IsOptions() bool {
	return ctx.request.IsOptions()
}

// Method is a proxy to request.Method
func (ctx *WebContext) Method() string {
	return ctx.request.Method()
}

// Body is a proxy to request.Body
func (ctx *WebContext) Body() []byte {
	return ctx.request.Body()
}

// Set is a proxy to request.setData
func (ctx *WebContext) Set(key string, value interface{}) {
	ctx.request.Set(key, value)
}

// Get is a proxy to request.Get
func (ctx *WebContext) Get(key string) interface{} {
	return ctx.request.Get(key)
}

// Context returns the request's context
func (ctx *WebContext) Context() context.Context {
	return ctx.request.Context()
}

// Cookie returns the named cookie provided in the request or ErrNoCookie if not found.
// If multiple cookies match the given name, only one cookie will be returned.
func (ctx *WebContext) Cookie(name string) (*http.Cookie, error) {
	return ctx.request.r.Cookie(name)
}

// Cookies parses and returns the HTTP cookies sent with the request.
func (ctx *WebContext) Cookies() []*http.Cookie {
	return ctx.request.r.Cookies()
}

// Session return session instance
func (ctx *WebContext) Session() *sessions.Session {
	return ctx.request.session
}

// View is a helper function for template rendering
func (ctx *WebContext) View(tplPath string, data interface{}) *HTMLResponse {
	content, err := View(filepath.Join(ctx.conf.ViewTemplatePathPrefix, tplPath), data)
	if err != nil {
		panic(errors.Wrap(err, "parse template failed"))
	}

	return ctx.HTML(content)
}

// Decode decodes form request to a struct
func (ctx *WebContext) Decode(v interface{}) error {
	return ctx.request.Decode(v)
}

// Validate execute a validator, if error happens, panic it
func (ctx *WebContext) Validate(validator Validator, jsonResponse bool) {
	ctx.request.Validate(validator, jsonResponse)
}

func (ctx *WebContext) RouteURL(name string, pairs ...string) (*url.URL, error) {
	return ctx.request.RouteURL(name, pairs...)
}

func (ctx *WebContext) RouteByName(name string) RouteAware {
	return ctx.request.RouteByName(name)
}

func (ctx *WebContext) CurrentRoute() RouteAware {
	return ctx.request.CurrentRoute()
}
