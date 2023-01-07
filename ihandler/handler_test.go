// Copyright 2023, Initialize All Once Authors
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
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluelamar/abstract-logger-go/alogger"
)

var (
	updateRespErr   = "missing cookie for MyWebSite"
	loginResp       = "hello buddy"
	cookieValPrefix = "MyWebSite="
)

func myDummyAuthorizer(w InterceptResponseWriterI, r *http.Request) (error, int, string) {
	log.Println("myDummyAuthorizer: do nothing - or perhaps log stats, per url etc.")

	return nil, 0, ""
}

func myRealAuthorizer(w InterceptResponseWriterI, r *http.Request) (error, int, string) {
	log.Println("myRealAuthorizer: check cookie")
	ck, err := r.Cookie("MyWebSite")
	if err != nil {
		log.Println("myRealAuthorizer: error checking cookie")
		return err, http.StatusUnauthorized, "missing cookie for MyWebSite"
	}

	// Perform validation of the cookie..., expiry, signature, etc

	log.Printf("myRealAuthorizer: got cookie=%v\n", ck)

	// Add this cookie into the response
	w.SetCookie(ck)

	return nil, 0, ""
}

func loginPage(w InterceptResponseWriterI, r *http.Request) {
	// Process the user login params.
	// ...

	// Ordering of setting the cookie and performing the Write's doesnt matter.
	txt1 := "hello"
	w.Write([]byte(txt1))

	// If valid login, add a cookie for the user.
	ck := &http.Cookie{
		Name:    "MyWebSite",
		Value:   "a1b2c3",
		Expires: time.Now(),
	}
	w.SetCookie(ck)

	// Ordering of setting the cookie and performing the Write's doesnt matter.
	txt2 := " buddy"
	w.Write([]byte(txt2))
}

func myRespChecker(w InterceptResponseWriterI, r *http.Request, respBytes *[][]byte) {
	// log number of bytes, log if Set-Cookie header was set
	nbytes := 0
	for _, chunk := range *respBytes {
		nbytes += len(chunk)
	}

	sc := w.Header().Get("Set-Cookie")

	log.Printf("response checker: num bytes returned=%d : cookie=%s", nbytes, sc)
}

func updateMyResource(w InterceptResponseWriterI, r *http.Request) {
	// Update authorized users resource
	// ...

	w.AddHeader("ETag", "a1")

	txt := "updated successfully"
	w.Write([]byte(txt))
}

type testHandler struct {
	irw InterceptResponseWriterI
}

func (h testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.irw.HandleFunc(w, r)
}

func TestMissingCookie(t *testing.T) {

	l := alogger.New(nil, true)
	ihu := New(updateMyResource, myRealAuthorizer, nil, l)
	// http.HandleFunc("/update", ihu.HandleFunc)
	// http.HandleFunc("/update", updateMyResource)

	th := &testHandler{
		irw: ihu,
	}

	ts := httptest.NewServer(th)
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf(`Expected status code(%d) since missing cookie: received status code: %d`, http.StatusUnauthorized, res.StatusCode)
	}

	respMsg, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	respMsgStr := string(respMsg)

	if !strings.HasPrefix(respMsgStr, updateRespErr) {
		t.Fatalf(`Expected error(%s) for missing cookie: received msg(%s)`, updateRespErr, respMsgStr)
	}
}

func TestReturnedCookie(t *testing.T) {

	// http.HandleFunc("/login", loginPage)
	// http.HandleFunc("/login", ihd.HandleFunc)
	l := alogger.New(nil, true)
	ihd := New(loginPage, myDummyAuthorizer, myRespChecker, l)

	th := &testHandler{
		irw: ihd,
	}

	ts := httptest.NewServer(th)
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf(`Expected status code(%d) for successful login: received status code: %d`, http.StatusOK, res.StatusCode)
	}

	respMsg, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	respMsgStr := string(respMsg)

	if !strings.HasPrefix(respMsgStr, loginResp) {
		t.Fatalf(`Expected msg(%s) for login response but received msg(%s)`, loginResp, respMsgStr)
	}

	hdrs := res.Header
	cookieVal := hdrs.Get("Set-Cookie")

	if !strings.HasPrefix(cookieVal, cookieValPrefix) {
		t.Fatalf(`Expected cookie(%s) for login response but received msg(%s)`, cookieValPrefix, cookieVal)
	}
}
