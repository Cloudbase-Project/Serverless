package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	Function "github.com/Cloudbase-Project/serverless/function"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	function := Function.NewFunction(clientset, logger)

	// add function
	router.HandleFunc("/function", function.CreateFunction).Methods(http.MethodPost)

	// list functions
	router.HandleFunc("/function", function.ListFunctions).Methods(http.MethodGet)

	// update function
	router.HandleFunc("/function/{id}", function.UpdateFunction).Methods(http.MethodPatch)

	// delete function
	router.HandleFunc("/function/{id}", function.DeleteFunction).Methods(http.MethodDelete)

	// View a function. View status/replicas RPS etc
	router.HandleFunc("/function/{id}", function.GetFunction).Methods(http.MethodGet)

	// Get logs of a function
	router.HandleFunc("/function/{id}/logs", function.GetFunctionLogs).Methods(http.MethodGet)

	// Create function creates function image. User has to deploy/redeploy for deployments to take effect.
	router.HandleFunc("/function/{id}/deploy", function.DeployFunction).Methods(http.MethodPost)

	server := http.Server{
		Addr:    ":" + PORT,
		Handler: router,
	}

	// handle os signals to shutoff server
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
