// Copyright 2022, Initialize All Once Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ihandler

import (
	"log"
	"net/http"
)

// InterceptResponseWriterI is a plugin replacement for the RespoonseWriter interface.
// It has additional functionality for http handlers to m
type InterceptResponseWriterI interface {
	// HandleFunc used as the handler in the http.HandleFunc call.
	HandleFunc(w http.ResponseWriter, r *http.Request)

	// Methods that match the http.ResponseWriter interface
	Header() http.Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)

	// Additional methods to support the users handler functionality

	// SetCookie can be called multiple times to add cookies to the response
	SetCookie(cookie *http.Cookie)

	// AddHeader can be called multiple times to add headers to the response
	AddHeader(name, value string)
}

// UserHandlerFunc matches closely with the handler function signature of http.HandleFunc.
// This will be implementeation specific to your web feature.
type UserHandlerFunc func(InterceptResponseWriterI, *http.Request)

// AuthorizerFunc method to support incoming request authentication and authorization.
// Headers and cookies can be pulled so that auth may be performed on the request.
// It could be used by the implementor to perform logging or stats reporting.
// The return values should include a valid HTTP status code upon error.
// The returned string may be empty (ie. ""), inwhich case the returned auth error will come from the error.
// The valid HTTP status codes as registered with IANA.
// See: https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
type AuthorizerFunc func(w InterceptResponseWriterI, r *http.Request) (error, int, string)

type interceptResponseWriter struct {
	rw          http.ResponseWriter
	userHandler UserHandlerFunc
	authorizer  AuthorizerFunc
	respBytes   *[][]byte
}

// New returns the interface used for the handler with the http.HandleFunc registration call.
// TODO add logger interface param
func New(userHandler UserHandlerFunc, authorizer AuthorizerFunc) InterceptResponseWriterI {
	respBytes := make([][]byte, 0)
	return &interceptResponseWriter{
		userHandler: userHandler,
		authorizer:  authorizer,
		respBytes:   &respBytes,
	}
}

func (i interceptResponseWriter) Header() http.Header {
	return i.Header()
}

func (i interceptResponseWriter) WriteHeader(statusCode int) {
	i.WriteHeader(statusCode)
}

func (i interceptResponseWriter) Write(b []byte) (int, error) {
	(*i.respBytes) = append((*i.respBytes), b)
	return len(b), nil
}

// SetCookie can be called multiple times to add cookies to the response
func (i interceptResponseWriter) SetCookie(cookie *http.Cookie) {
	http.SetCookie(i.rw, cookie)
}

func (i interceptResponseWriter) AddHeader(name, value string) {
	i.rw.Header().Add(name, value)
}

// HandleFunc is the handler you pass to http.HandleFunc
// Direct the logging via log.SetOutput(logger)
func (i *interceptResponseWriter) HandleFunc(w http.ResponseWriter, r *http.Request) {
	i.rw = w

	// ex: http.StatusUnauthorized, "Not authorized"
	err, statusCode, msg := i.authorizer(i, r)
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}

		// TODO warn, error, info - use logger interface
		log.Printf("interceptor:HandleFunc: authorizer failed: error=%v\n", err)
		http.Error(w, msg, statusCode)
		return
	}

	i.userHandler(i, r)
	for _, chunk := range *i.respBytes {
		n, err := w.Write(chunk)
		if err != nil {
			log.Printf("interceptor:HandleFunc: n=%d failed: error=%v\n", n, err)
			return
		}
	}
}
