package constants

const (
	NodejsDockerfile  = "FROM node:alpine \n workdir /app \n copy package.json . \n run npm install \n copy . . \n cmd ['npm', 'start']"
	NodejsPackageJSON = "{\r\n  \"name\": \"user-code-worker\",\r\n  \"version\": \"1.0.0\",\r\n  \"main\": \"index.js\",\r\n  \"license\": \"MIT\",\r\n  \"dependencies\": {\r\n    \"express\": \"^4.17.1\"\r\n  }\r\n}\r\n"
)
