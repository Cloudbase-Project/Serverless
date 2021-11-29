package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Cloudbase-Project/serverless/handlers"
	"github.com/Cloudbase-Project/serverless/models"
	"github.com/Cloudbase-Project/serverless/services"
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

	dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	logger.Print("Connected to DB")

	db.AutoMigrate(&models.Function{})

	fs := services.NewFunctionService(db, logger)

	function := handlers.NewFunctionHandler(clientset, logger, fs)

	// add function
	router.HandleFunc("/function", function.CreateFunction).Methods(http.MethodPost)

	// list functions created by the user
	router.HandleFunc("/functions", function.ListFunctions).Methods(http.MethodGet)

	// update function
	router.HandleFunc("/function/{codeId}", function.UpdateFunction).Methods(http.MethodPatch)

	// delete function
	router.HandleFunc("/function/{codeId}", function.DeleteFunction).Methods(http.MethodDelete)

	// View a function. View status/replicas RPS etc
	router.HandleFunc("/function/{codeId}", function.GetFunction).Methods(http.MethodGet)

	// Get logs of a function
	router.HandleFunc("/function/{codeId}/logs", function.GetFunctionLogs).Methods(http.MethodGet)

	// Create function creates function image. User has to deploy/redeploy for deployments to take effect.
	router.HandleFunc("/function/{codeId}/deploy", function.DeployFunction).Methods(http.MethodPost)

	router.HandleFunc("/function/{codeId}/redeploy", function.RedeployFunction).
		Methods(http.MethodPost)

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
	logger.Println("received signal. terminating...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.Shutdown(ctx)

}
