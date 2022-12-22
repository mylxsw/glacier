package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	gorillaCtx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/mylxsw/go-ioc"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// HttpRequest 请求对象封装
type HttpRequest struct {
	r       *http.Request
	body    []byte
	session *sessions.Session
	cc      ioc.Container
	router  *routerImpl
	conf    Config
}

// Context returns the request's context
func (req *HttpRequest) Context() context.Context {
	return req.r.Context()
}

// Cookie returns the named cookie provided in the request or ErrNoCookie if not found.
// If multiple cookies match the given name, only one cookie will be returned.
func (req *HttpRequest) Cookie(name string) (*http.Cookie, error) {
	return req.r.Cookie(name)
}

// Cookies parses and returns the HTTP cookies sent with the request.
func (req *HttpRequest) Cookies() []*http.Cookie {
	return req.r.Cookies()
}

// SetSession set a session to request
func (req *HttpRequest) SetSession(session *sessions.Session) {
	req.session = session
}

// Session return session instance
func (req *HttpRequest) Session() *sessions.Session {
	return req.session
}

// Raw get the underlying http.request
func (req *HttpRequest) Raw() *http.Request {
	return req.r
}

// Decode decodes form request to a struct
func (req *HttpRequest) Decode(v interface{}) error {
	decoder := req.cc.MustGet(&schema.Decoder{}).(*schema.Decoder)
	if req.ContentType() == "multipart/form-data" {
		if err := req.r.ParseMultipartForm(req.conf.MultipartFormMaxMemory); err != nil {
			return errors.Wrap(err, "parse multipart form failed")
		}

		if err := decoder.Decode(v, req.r.MultipartForm.Value); err != nil {
			return errors.Wrap(err, "decode multipart-form failed")
		}

		return nil
	}

	if err := req.r.ParseForm(); err != nil {
		return errors.Wrap(err, "parse form failed")
	}

	if err := decoder.Decode(v, req.r.Form); err != nil {
		return errors.Wrap(err, "decode form failed")
	}

	return nil
}

// Unmarshal request body as json object
// result must be reference to a variable
func (req *HttpRequest) Unmarshal(v interface{}) error {
	return json.Unmarshal(req.body, v)
}

// UnmarshalYAML unmarshal request body as yaml object
// result must be reference to a variable
func (req *HttpRequest) UnmarshalYAML(v interface{}) error {
	return yaml.Unmarshal(req.body, v)
}

// Set 设置一个变量，存储到当前请求
func (req *HttpRequest) Set(key string, value interface{}) {
	gorillaCtx.Set(req.r, key, value)
}

// Get 从当前请求提取设置的变量
func (req *HttpRequest) Get(key string) interface{} {
	return gorillaCtx.Get(req.r, key)
}

// Clear all variables in request
func (req *HttpRequest) Clear() {
	gorillaCtx.Clear(req.r)
}

// HTTPRequest return a http.request
func (req *HttpRequest) HTTPRequest() *http.Request {
	return req.r
}

// PathVar return a path parameter
func (req *HttpRequest) PathVar(key string) string {
	if res, ok := mux.Vars(req.r)[key]; ok {
		return res
	}

	return ""
}

// PathVars return all path parameters
func (req *HttpRequest) PathVars() map[string]string {
	return mux.Vars(req.r)
}

// Input return form parameter from request
func (req *HttpRequest) Input(key string) string {
	if req.IsJSON() {
		val := req.JSONGet(key)
		if val != "" {
			return val
		}
	}

	return req.r.FormValue(key)
}

func (req *HttpRequest) JSONGet(keys ...string) string {
	value, dataType, _, err := jsonparser.Get(req.body, keys...)
	if err != nil {
		return ""
	}

	switch dataType {
	case jsonparser.String:
		if res, err := jsonparser.ParseString(value); err == nil {
			return res
		}
	case jsonparser.Number:
		if res, err := jsonparser.ParseFloat(value); err == nil {
			return strconv.FormatFloat(res, 'f', -1, 32)
		}
		if res, err := jsonparser.ParseInt(value); err == nil {
			return fmt.Sprintf("%d", res)
		}
	case jsonparser.Object:
		fallthrough
	case jsonparser.Array:
		return fmt.Sprintf("%x", value)
	case jsonparser.Boolean:
		if res, err := jsonparser.ParseBoolean(value); err == nil {
			if res {
				return "true"
			} else {
				return "false"
			}
		}
	case jsonparser.NotExist:
		fallthrough
	case jsonparser.Null:
		fallthrough
	case jsonparser.Unknown:
		return ""
	}

	return ""
}

// InputWithDefault return a form parameter with a default value
func (req *HttpRequest) InputWithDefault(key string, defaultVal string) string {
	val := req.Input(key)
	if val == "" {
		return defaultVal
	}

	return val
}

func (req *HttpRequest) ToInt(val string, defaultVal int) int {
	res, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return res
}

func (req *HttpRequest) ToInt64(val string, defaultVal int64) int64 {
	res, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}

	return res
}

func (req *HttpRequest) ToFloat32(val string, defaultVal float32) float32 {
	res, err := strconv.ParseFloat(val, 32)
	if err != nil {
		return defaultVal
	}

	return float32(res)
}

func (req *HttpRequest) ToFloat64(val string, defaultVal float64) float64 {
	res, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal
	}

	return res
}

// IntInput return a integer form parameter
func (req *HttpRequest) IntInput(key string, defaultVal int) int {
	return req.ToInt(req.Input(key), defaultVal)
}

// Int64Input return a integer form parameter
func (req *HttpRequest) Int64Input(key string, defaultVal int64) int64 {
	return req.ToInt64(req.Input(key), defaultVal)
}

// Float32Input return a float32 form parameter
func (req *HttpRequest) Float32Input(key string, defaultVal float32) float32 {
	return req.ToFloat32(req.Input(key), defaultVal)
}

// Float64Input return a float64 form parameter
func (req *HttpRequest) Float64Input(key string, defaultVal float64) float64 {
	return req.ToFloat64(req.Input(key), defaultVal)
}

// File Retrieving Uploaded Files
func (req *HttpRequest) File(key string) (*UploadedFile, error) {
	file, header, err := req.r.FormFile(key)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = file.Close()
	}()

	tempFile, err := os.CreateTemp(req.conf.TempDir, req.conf.TempFilePattern)
	if err != nil {
		return nil, fmt.Errorf("can not create temporary file %s", err.Error())
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return nil, err
	}

	return &UploadedFile{
		Header:   header,
		SavePath: tempFile.Name(),
	}, nil
}

// IsXMLHTTPRequest return whether the request is a ajax request
func (req *HttpRequest) IsXMLHTTPRequest() bool {
	return req.r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// AJAX return whether the request is a ajax request
func (req *HttpRequest) AJAX() bool {
	return req.IsXMLHTTPRequest()
}

// IsJSON return whether the request is a json request
func (req *HttpRequest) IsJSON() bool {
	return req.ContentType() == "application/json"
}

// ContentType return content type for request
func (req *HttpRequest) ContentType() string {
	t := req.r.Header.Get("Content-Type")
	if t == "" {
		return "text/html"
	}

	return strings.ToLower(strings.Split(t, ";")[0])
}

// AllHeaders return all http request headers
func (req *HttpRequest) AllHeaders() http.Header {
	return req.r.Header
}

// Headers gets all values associated with given key
func (req *HttpRequest) Headers(key string) []string {
	res, ok := req.r.Header[key]
	if !ok {
		return make([]string, 0)
	}

	return res
}

// Header gets the first value associated with the given key.
func (req *HttpRequest) Header(key string) string {
	return req.r.Header.Get(key)
}

// Is 判断请求方法
func (req *HttpRequest) Is(method string) bool {
	return req.Method() == method
}

// IsGet 判断是否是Get请求
func (req *HttpRequest) IsGet() bool {
	return req.Is("GET")
}

// IsPost 判断是否是Post请求
func (req *HttpRequest) IsPost() bool {
	return req.Is("POST")
}

// IsHead 判断是否是HEAD请求
func (req *HttpRequest) IsHead() bool {
	return req.Is("HEAD")
}

// IsDelete 判断是是否是Delete请求
func (req *HttpRequest) IsDelete() bool {
	return req.Is("DELETE")
}

// IsPut 判断是否是Put请求
func (req *HttpRequest) IsPut() bool {
	return req.Is("PUT")
}

// IsPatch 判断是否是Patch请求
func (req *HttpRequest) IsPatch() bool {
	return req.Is("PATCH")
}

// IsOptions 判断是否是Options请求
func (req *HttpRequest) IsOptions() bool {
	return req.Is("OPTIONS")
}

// Method 获取请求方法
func (req *HttpRequest) Method() string {
	return req.r.Method
}

// Body return request body
func (req *HttpRequest) Body() []byte {
	return req.body
}

// Validator is an interface for validator
type Validator interface {
	Validate(request Request) error
}

// Validate execute a validator, if there has an error, panic error to framework
func (req *HttpRequest) Validate(validator Validator, jsonResponse bool) {
	if err := validator.Validate(req); err != nil {
		if jsonResponse {
			panic(WrapJSONError(fmt.Errorf("invalid request: %v", err), http.StatusUnprocessableEntity))
		} else {
			panic(WrapPlainError(fmt.Errorf("invalid request: %v", err), http.StatusUnprocessableEntity))
		}
	}
}

// RouteURL builds a URL for the route
func (req *HttpRequest) RouteURL(name string, pairs ...string) (*url.URL, error) {
	return req.router.router.Get(name).URL(pairs...)
}

// RouteByName get a route by name
func (req *HttpRequest) RouteByName(name string) RouteAware {
	return req.router.router.Get(name)
}

// CurrentRoute return current route
func (req *HttpRequest) CurrentRoute() RouteAware {
	return mux.CurrentRoute(req.r)
}

// RemoteAddr return remote address
func (req *HttpRequest) RemoteAddr() string {
	return req.r.RemoteAddr
}
