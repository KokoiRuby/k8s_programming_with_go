package main

import (
	"log"
	"net/http"
	"net/http/httputil"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", logHandler(http.NotFoundHandler()))

	// API Disocvery
	mux.Handle("/apis", logHandler(http.HandlerFunc(apis)))
	mux.Handle("/apis/mygroup.com", logHandler(http.HandlerFunc(apisGroup)))
	mux.Handle("/apis/mygroup.com/v1", logHandler(http.HandlerFunc(apisGroupVersion)))

}

// logHandler simply decorates http.Handler with printing response
func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, _ := httputil.DumpRequest(r, true)
		log.Println(string(response))
		h.ServeHTTP(w, r)
	})
}
