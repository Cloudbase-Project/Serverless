package utils

import "net/http"

// returns a fully qualified image name given a function id.
//
// eg: quay.io/ubuntu-project/ubuntu:latest
func BuildImageName(functionId string) string {
	// TODO: implement this
	Registry := "ghcr.io"
	Project := ""

	imageName := Registry + "/" + Project + "/" + functionId + ":latest"
	return imageName
}

// returns a service name given a functionId
//
// eg: 127319ey71e291y2e12e01u-srv
func BuildServiceName(functionId string) string {
	return functionId + "-srv"
}

// set http headers
func SetSSEHeaders(rw http.ResponseWriter) http.ResponseWriter {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	return rw
}
