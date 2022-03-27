FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

# RUN go build -o /serverless

# CMD [ "/serverless" ]

CMD ["go", "run", "main.go"]
