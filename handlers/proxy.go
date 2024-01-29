package handlers

import (
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

	_, err := p.service.VerifyFunction(functionId)
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	urlString := r.URL.String()
	x := strings.Split(urlString, "/serve/"+functionId)

	functionURL := "http://cloudbase-serverless-" + functionId + "-srv:4000" + x[0]

	finalURL, err := url.Parse(functionURL)
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	proxy := httputil.NewSingleHostReverseProxy(finalURL)
	r.URL.Host = finalURL.Host
	r.URL.Scheme = finalURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = finalURL.Host
	r.URL.Path = finalURL.Path
	r.URL.RawPath = finalURL.RawPath
	proxy.ServeHTTP(rw, r)

}
