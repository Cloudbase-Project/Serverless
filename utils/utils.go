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
