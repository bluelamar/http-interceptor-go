# HTTP Handler Interceptor

The library gives the ability for web applications to preprocess an HTTP request before it is
handed to the user implemented handler that is provided to the http.HandlerFunc call.
The library allows for authentication and/or authorization in the preprocessing of a request.
That allows clean separation of concerns between auth and feature resource handling.
The library provides a plugin replacement for the `ResponseWriter` interface with 
additional functionality to add cookies or headers to the response.

Example of typical *http* package usage:
```go
	// setup routes
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/create", createResourcePage)
	http.HandleFunc("/read",   readMyResourcePage)
	http.HandleFunc("/update", updateResourcePage)
	http.HandleFunc("/delete", deleteResourcePage)
	...
```

In the above example, typically the *CRUD* resources (ex. `/update`) should be authorized.
Any authorization in the above example would have to be done within each handler (ex. `updateResourcePage`).

Using this package will allow an application to register authorization/stats-reporting etc.
per handler.
This separation of concerns allows the handler to perform only the resource work for the feature it supports.
The authorization will *intercept* the request before it is processed by the handler.
So if authorization is denied the handler will not be called.


## Usage

Example Registering the *http* handler functions:

```go
package main

import ihandler "github.com/bluelamar/http-interceptor-go"

func loginPage(w InterceptResponseWriterI, r *http.Request) {
	// Process the user login params.
	// ...

	// Ordering of setting the cookie and performing the Write's doesnt matter.
    txt1 := "hello"
    w.Write([]byte(txt1))

    // If valid login, add a cookie for the user. 
    ck := &http.Cookie{
        Name:  "MyWebSite",
        Value: "a1b2c3",
        Expires: time.Now(),
    }
    w.SetCookie(ck)

	// Ordering of setting the cookie and performing the Write's doesnt matter.
    txt2 := "buddy"
    w.Write([]byte(txt2))
}

func updateMyResource(w InterceptResponseWriterI, r *http.Request) {
    // Update authorized users resource
	// ...

    w.AddHeader("ETag", "a1")

    txt := "updated successfully"
    w.Write([]byte(txt))
}

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

func main() {

	// setup routes with the interceptors

	// http.HandleFunc("/login", loginPage)
    ihd := ihandler.New(loginPage, myDummyAuthorizer)
    http.HandleFunc("/login", ihd.HandleFunc)

	// http.HandleFunc("/create", createResourcePage)
    ihc := ihandler.New(createMyResource, myRealAuthorizer)
    http.HandleFunc("/create", ihc.HandleFunc)

	// http.HandleFunc("/update", updateResourcePage)
    ihu := ihandler.New(updateMyResource, myRealAuthorizer)
    http.HandleFunc("/update", ihu.HandleFunc)

    err := http.ListenAndServe("127.0.0.1:8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```

