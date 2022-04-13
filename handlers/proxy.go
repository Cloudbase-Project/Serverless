package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Cloudbase-Project/serverless/services"
	"github.com/gorilla/mux"
)

type ProxyHandler struct {
	l       *log.Logger
	service *services.ProxyService
}

// create new function
func NewProxyHandler(
	l *log.Logger,
	s *services.ProxyService,
) *ProxyHandler {
	return &ProxyHandler{l: l, service: s}
}

func (p *ProxyHandler) ProxyRequest(rw http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	functionId := vars["functionId"]

	function, err := p.service.VerifyFunction(functionId)
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	fmt.Printf("function: %v\n", function)

	urlString := r.URL.String()
	fmt.Printf("urlString: %v\n", urlString)
	x := strings.Split(urlString, "/serve/"+functionId)
	fmt.Println("xxxx : ", x)

	functionURL := "http://cloudbase-serverless-" + functionId + "-srv:4000" + x[0]
	fmt.Printf("functionURL: %v\n", functionURL)

	finalURL, err := url.Parse(functionURL)
	fmt.Printf("finalURL: %v\n", finalURL)
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}
	fmt.Println("this")
	resp, err := http.Get(functionURL)
	fmt.Printf("resp: %v\n", resp)

	proxy := httputil.NewSingleHostReverseProxy(finalURL)
	r.URL.Host = finalURL.Host
	r.URL.Scheme = finalURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = finalURL.Host
	r.URL.Path = finalURL.Path
	r.URL.RawPath = finalURL.RawPath
	proxy.ServeHTTP(rw, r)
	fmt.Println("after")

	// http://backend.cloudbase.dev/deploy/asdadjpiqwjdpqidjp/qwwe?123=qwe -> proxy to -> http://cloudbase-serverless-asdadjpiqwjdpqidjp-srv:4000qwwe?123=qwe

}
