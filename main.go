package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Cloudbase-Project/serverless/handlers"
	"github.com/Cloudbase-Project/serverless/middlewares"
	"github.com/Cloudbase-Project/serverless/models"
	"github.com/Cloudbase-Project/serverless/services"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	// dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	dsn := os.Getenv("POSTGRES_URI")
	fmt.Printf("dsn: %v\n", dsn)
	var db *gorm.DB

	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Println("failed to connect database")
			time.Sleep(time.Second * 10)
			continue
		}
		logger.Print("Connected to DB")
		break

	}

	RequestCounter := promauto.NewCounter(
		prometheus.CounterOpts{Name: "serverless_requests_total"},
	)

	db.AutoMigrate(&models.Function{}, &models.Config{})

	fs := services.NewFunctionService(db, logger)
	cs := services.NewConfigService(db, logger)
	ps := services.NewProxyService(db, logger)

	function := handlers.NewFunctionHandler(clientset, logger, fs)
	configHandler := handlers.NewConfigHandler(logger, cs)
	// proxyHandler := handlers.NewProxyHandler(logger, ps)
	// add function
	router.HandleFunc("/function/{projectId}", middlewares.AuthMiddleware(function.CreateFunction)).
		Methods(http.MethodPost)

	router.HandleFunc(
		"/function/{projectId}/{codeId}/build",
		middlewares.AuthMiddleware(function.BuildFunction),
	).
		Methods(http.MethodPost)

	// list functions created by the user
	router.HandleFunc("/functions/{projectId}", middlewares.AuthMiddleware(function.ListFunctions)).
		Methods(http.MethodGet)

	// update function
	router.HandleFunc("/function/{projectId}/{codeId}", middlewares.AuthMiddleware(function.UpdateFunction)).
		Methods(http.MethodPatch)

	// delete function
	router.HandleFunc("/function/{projectId}/{codeId}", middlewares.AuthMiddleware(function.DeleteFunction)).
		Methods(http.MethodDelete)

	// View a function. View status/replicas RPS etc
	router.HandleFunc("/function/{projectId}/{codeId}", middlewares.AuthMiddleware(function.GetFunction)).
		Methods(http.MethodGet)

	// Get logs of a function
	router.HandleFunc("/logs/{codeId}/logs", function.GetFunctionLogs).
		Methods(http.MethodGet)

	// Create function creates function image. User has to deploy/redeploy for deployments to take effect.
	router.HandleFunc("/function/{projectId}/{codeId}/deploy", middlewares.AuthMiddleware(function.DeployFunction)).
		Methods(http.MethodPost)

	router.HandleFunc("/function/{projectId}/{codeId}/redeploy", middlewares.AuthMiddleware(function.RedeployFunction)).
		Methods(http.MethodPost)

		// ------------------ CONFIG ROUTES
	router.HandleFunc("/config/", configHandler.CreateConfig).Methods(http.MethodPost)

	router.HandleFunc("/serve/{functionId}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		functionId := vars["functionId"]

		_, err := ps.VerifyFunction(functionId)
		if err != nil {
			http.Error(rw, err.Error(), 400)
		}

		urlString := r.URL.String()
		fmt.Printf("urlString: %v\n", urlString)
		x := strings.Split(urlString, "/serve/"+functionId)

		functionURL := "http://cloudbase-serverless-" + functionId + "-srv:4000" + x[0]
		fmt.Printf("functionURL: %v\n", functionURL)

		finalURL, err := url.Parse(functionURL)
		fmt.Printf("finalURL: %v\n", finalURL)
		if err != nil {
			http.Error(rw, err.Error(), 400)
		}
		_, err = http.Get(functionURL)
		RequestCounter.Inc()

		proxy := httputil.NewSingleHostReverseProxy(finalURL)
		r.URL.Host = finalURL.Host
		r.URL.Scheme = finalURL.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = finalURL.Host
		r.URL.Path = finalURL.Path
		r.URL.RawPath = finalURL.RawPath
		proxy.ServeHTTP(rw, r)

		// http://backend.cloudbase.dev/deploy/asdadjpiqwjdpqidjp/qwwe?123=qwe -> proxy to -> http://cloudbase-serverless-asdadjpiqwjdpqidjp-srv:4000qwwe?123=qwe

	}).Methods(http.MethodGet)

	router.HandleFunc("/testing", func(w http.ResponseWriter, r *http.Request) {
	})
	router.Handle("/metrics", promhttp.Handler())

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
