package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

// HandlerDecorator 该函数是http handler的装饰器
type HandlerDecorator func(WebHandler) WebHandler

// RequestMiddleware is a middleware collections
type RequestMiddleware struct{}

// NewRequestMiddleware create a new RequestMiddleware
func NewRequestMiddleware() RequestMiddleware {
	return RequestMiddleware{}
}

// AccessLog create a access log middleware
func (rm RequestMiddleware) AccessLog() HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			startTs := time.Now()
			resp := handler(ctx)

			logger.Debugf(
				"%s %s [%d] [%.4fms]",
				ctx.Method(),
				ctx.Request().Raw().URL.String(),
				resp.Code(),
				time.Now().Sub(startTs).Seconds()*1000,
			)

			return resp
		}
	}
}

type CustomAccessLog struct {
	Context      Context       `json:"-"`
	Method       string        `json:"method"`
	URL          string        `json:"url"`
	ResponseCode int           `json:"response_code"`
	Elapse       time.Duration `json:"elapse"`
}

// CustomAccessLog create a custom access log handler middleware
func (rm RequestMiddleware) CustomAccessLog(fn func(cal CustomAccessLog)) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			startTs := time.Now()
			resp := handler(ctx)

			go fn(CustomAccessLog{
				Context:      ctx,
				Method:       ctx.Method(),
				URL:          ctx.Request().Raw().URL.String(),
				ResponseCode: resp.Code(),
				Elapse:       time.Now().Sub(startTs),
			})

			return resp
		}
	}
}

// BeforeInterceptor is a interceptor intercept a request before processing
func (rm RequestMiddleware) BeforeInterceptor(fn func(ctx Context) Response) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			if resp := fn(ctx); resp != nil {
				return resp
			}

			return handler(ctx)
		}
	}
}

// AfterInterceptor is a interceptor intercept a response before it's been sent to user
func (rm RequestMiddleware) AfterInterceptor(fn func(ctx Context, resp Response) Response) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			return fn(ctx, handler(ctx))
		}
	}
}

// CORS create a CORS middleware
func (rm RequestMiddleware) CORS(origin string) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			ctx.Response().Header("Access-Control-Allow-Origin", origin)
			ctx.Response().Header("Access-Control-Allow-Headers", "*")
			ctx.Response().Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS,HEAD,PUT,PATCH,DELETE")

			return handler(ctx)
		}
	}
}

// AuthHandler is a middleware for http auth
// typ is one of:
// Basic (see RFC 7617, base64-encoded credentials. See below for more information.),
// Bearer (see RFC 6750, bearer tokens to access OAuth 2.0-protected resources),
// Digest (see RFC 7616, only md5 hashing is supported in Firefox, see bug 472823 for SHA encryption support),
// HOBA (see RFC 7486, Section 3, HTTP Origin-Bound Authentication, digital-signature-based),
// Mutual (see RFC 8120),
// AWS4-HMAC-SHA256 (see AWS docs).
func (rm RequestMiddleware) AuthHandler(cb func(ctx Context, typ string, credential string) error) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) (resp Response) {
			segs := strings.SplitN(ctx.Header("Authorization"), " ", 2)
			if len(segs) != 2 {
				return ctx.JSONError("auth failed: invalid auth header", http.StatusUnauthorized)
			}

			if !inStringArray(segs[0], []string{"Basic", "Bearer", "Digest", "HOBA", "Mutual", "AWS4-HMAC-SHA256"}) {
				return ctx.JSONError("auth failed: invalid auth type", http.StatusUnauthorized)
			}

			if err := cb(ctx, segs[0], segs[1]); err != nil {
				return ctx.JSONError(fmt.Sprintf("auth failed: %s", err), http.StatusUnauthorized)
			}

			return handler(ctx)
		}
	}
}

// Session is a middleware for session support
func (rm RequestMiddleware) Session(store sessions.Store, name string, options *sessions.Options) HandlerDecorator {
	return func(handler WebHandler) WebHandler {
		return func(ctx Context) Response {
			var err error
			session, err := store.Get(ctx.Request().Raw(), name)
			if err != nil {
				panic(errors.Wrap(err, "can not create session"))
			}

			if options != nil {
				session.Options = options
			}

			ctx.Request().SetSession(session)
			resp := handler(ctx)
			if err := session.Save(ctx.Request().Raw(), ctx.Response().Raw()); err != nil {
				panic(errors.Wrap(err, "can not save session"))
			}

			return resp
		}
	}
}

func inStringArray(key string, items []string) bool {
	for _, item := range items {
		if item == key {
			return true
		}
	}

	return false
}
