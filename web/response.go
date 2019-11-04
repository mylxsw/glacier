package web

import "net/http"

// HttpResponse is a response object which wrap http.ResponseWriter
type HttpResponse struct {
	w        http.ResponseWriter
	headers  map[string]string
	cookie   *http.Cookie
	original []byte
	code     int
}

func (resp *HttpResponse) Raw() http.ResponseWriter {
	return resp.w
}

// GetCode get response code
func (resp *HttpResponse) GetCode() int {
	return resp.code
}

// SetCode set response code
func (resp *HttpResponse) SetCode(code int) {
	resp.code = code
}

// ResponseWriter return the http.ResponseWriter
func (resp *HttpResponse) ResponseWriter() http.ResponseWriter {
	return resp.w
}

// SetContent set response content
func (resp *HttpResponse) SetContent(content []byte) {
	resp.original = content
}

// Header set response header
func (resp *HttpResponse) Header(key, value string) {
	resp.headers[key] = value
}

// Cookie set cookie
func (resp *HttpResponse) Cookie(cookie *http.Cookie) {
	// http.SetCookie(resp.w, cookie)
	resp.cookie = cookie
}

// Flush send all response contents to client
func (resp *HttpResponse) Flush() {
	// set response headers
	for key, value := range resp.headers {
		resp.w.Header().Set(key, value)
	}

	// set cookies
	if resp.cookie != nil {
		http.SetCookie(resp.w, resp.cookie)
	}

	// set response code
	resp.w.WriteHeader(resp.code)

	// send response body
	_, _ = resp.w.Write(resp.original)
}

// M represents a kv response items
type M map[string]interface{}
