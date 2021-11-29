package utils

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
