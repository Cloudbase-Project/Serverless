package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cloudbase-Project/serverless/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	logger := log.New(os.Stdout, "SERVERLESS_SERVER ", log.LstdFlags)

	err := godotenv.Load()
	if err != nil {
		logger.Fatal("Cannot load env variables")
	}

	PORT, ok := os.LookupEnv("PORT")
	if !ok {
		PORT = "3000"
	}

	router := mux.NewRouter()

	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("hello world"))
	})
	router.HandleFunc("/code", handlers.CodeHandler).Methods("POST")

	server := http.Server{
		Addr:    ":" + PORT,
		Handler: router,
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Println("Starting server on port : ", PORT)
		logger.Fatal(server.ListenAndServe())
	}()

	<-c
	logger.Println("received signal. terminating")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.Shutdown(ctx)

}
