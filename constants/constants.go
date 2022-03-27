package constants

type Language string

const (
	NODEJS Language = "NODEJS"
	GOLANG Language = "GOLANG"
)

const (
	NodejsDockerfile  = "FROM node:alpine \n workdir /app \n copy package.json . \n run npm install \n copy . . \n cmd [\"node\", \"index.js\"]"
	NodejsPackageJSON = "{\r\n  \"name\": \"user-code-worker\",\r\n  \"version\": \"1.0.0\",\r\n  \"main\": \"index.js\",\r\n  \"license\": \"MIT\",\r\n  \"dependencies\": {\r\n    \"express\": \"^4.17.1\"\r\n  }\r\n}\r\n"
	// Namespace           = "serverless"
	Namespace           = "default"
	RegistryCredentials = "qweqwe"
)

type BuildStatus string

const (
	Building     BuildStatus = "Building"
	BuildSuccess BuildStatus = "Success"
	BuildFailed  BuildStatus = "Failed"
	NotBuilt     BuildStatus = "NotBuilt"
)

type DeploymentStatus string

const (
	DeploymentFailed DeploymentStatus = "DeploymentFailed"
	Deployed         DeploymentStatus = "Deployed"
	Deploying        DeploymentStatus = "Deploying"
	// signifies that function has be recently updated.
	RedeployRequired DeploymentStatus = "RedeployRequired"
	NotDeployed      DeploymentStatus = "NotDeployed"
)

type LastAction string

const (
	UpdateAction LastAction = "Update"
	DeployAction LastAction = "Deploy"
	BuildAction  LastAction = "Build"
	CreateAction LastAction = "Create"
)
