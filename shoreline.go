// @title Shoreline API
// @version 0.0.1
// @description The purpose of this API is to provide authentication for end users and other tidepool Services
// @license.name BSD 2-Clause "Simplified" License
// @host localhost
// @BasePath /auth
// @accept json
// @produce json
// @schemes https

// @securityDefinitions.basic BasicAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey TidepoolAuth
// @in header
// @name x-tidepool-session-token
package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	common "github.com/mdblp/go-common"
	"github.com/mdblp/go-common/clients"
	"github.com/mdblp/go-common/clients/disc"
	"github.com/mdblp/go-common/clients/mongo"
	"github.com/mdblp/shoreline/user"
	"github.com/mdblp/shoreline/user/marketo"
)

var (
	failedMarketoKeyConfigurationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failedMarketoKeyConfigurationCounter",
		Help: "The total number of failures to connect to marketo due to key configuration issues. Can not be resolved via retry",
	})
)

type (
	// Config is the Shoreline main configuration
	Config struct {
		clients.Config
		Service disc.ServiceListing `json:"service"`
		Mongo   mongo.Config        `json:"mongo"`
		User    user.ApiConfig      `json:"user"`
	}
)

func main() {
	var config Config
	logger := log.New(os.Stdout, user.USER_API_PREFIX, log.LstdFlags|log.LUTC|log.Lshortfile)
	auditLogger := log.New(os.Stdout, user.USER_API_PREFIX, log.LstdFlags|log.LUTC|log.Lshortfile)
	mongoLogger := log.New(os.Stdout, "mongodb ", log.LstdFlags|log.LUTC|log.Lshortfile)
	// Init random number generator
	rand.Seed(time.Now().UnixNano())

	// Set some default config values
	config.User.MaxFailedLogin = 5
	config.User.DelayBeforeNextLoginAttempt = 10 // 10 minutes
	config.User.MaxConcurrentLogin = 100
	config.User.BlockParallelLogin = true

	if err := common.LoadEnvironmentConfig([]string{"TIDEPOOL_SHORELINE_SERVICE"}, &config); err != nil {
		logger.Fatalf("Problem loading Shoreline config: %v", err)
	}

	// server secret may be passed via a separate env variable to accomodate easy secrets injection via Kubernetes
	// The server secret is the password any Tidepool service is supposed to know and pass to shoreline for authentication and for getting token
	// With Mdblp, we consider we can have different server secrets
	// These secrets are hosted in a map[string][string] instead of single string
	// which 1st string represents Server/Service name and 2nd represents the actual secret
	// here we consider this SERVER_SECRET that can be injected via Kubernetes is the one for the default server/service (any Tidepool service)
	serverSecret, found := os.LookupEnv("SERVER_SECRET")
	if found {
		config.User.ServerSecrets["default"] = serverSecret
	}

	userSecret, found := os.LookupEnv("API_SECRET")
	if found {
		config.User.Secret = userSecret
	}

	mailchimpAPIKey, found := os.LookupEnv("MAILCHIMP_APIKEY")
	if found {
		config.User.Mailchimp.APIKey = mailchimpAPIKey
	}

	longTermKey, found := os.LookupEnv("LONG_TERM_KEY")
	if found {
		config.User.LongTermKey = longTermKey
	}

	verificationSecret, found := os.LookupEnv("VERIFICATION_SECRET")
	if found {
		config.User.VerificationSecret = verificationSecret
	}

	clinicLists, found := os.LookupEnv("CLINIC_LISTS")
	if found {
		if err := json.Unmarshal([]byte(clinicLists), &config.User.Mailchimp.ClinicLists); err != nil {
			log.Panic("Problem loading clinic lists", err)
		}
	}

	personalLists, found := os.LookupEnv("PERSONAL_LISTS")
	if found {
		if err := json.Unmarshal([]byte(personalLists), &config.User.Mailchimp.PersonalLists); err != nil {
			log.Panic("Problem loading personal lists", err)
		}
	}

	clinicDemoUserID, found := os.LookupEnv("DEMO_CLINIC_USER_ID")
	if found {
		config.User.ClinicDemoUserID = clinicDemoUserID
	}
	config.User.Marketo.ID, _ = os.LookupEnv("MARKETO_ID")

	config.User.Marketo.URL, _ = os.LookupEnv("MARKETO_URL")

	config.User.Marketo.Secret, _ = os.LookupEnv("MARKETO_SECRET")

	config.User.Marketo.ClinicRole, _ = os.LookupEnv("MARKETO_CLINIC_ROLE")

	config.User.Marketo.PatientRole, _ = os.LookupEnv("MARKETO_PATIENT_ROLE")

	unParsedTimeout, found := os.LookupEnv("MARKETO_TIMEOUT")
	if found {
		parsedTimeout64, err := strconv.ParseInt(unParsedTimeout, 10, 32)
		parsedTimeout := uint(parsedTimeout64)
		if err != nil {
			logger.Println(err)
		}
		config.User.Marketo.Timeout = parsedTimeout
	}

	mailChimpURL, found := os.LookupEnv("MAILCHIMP_URL")
	if found {
		config.User.Mailchimp.URL = mailChimpURL
	}

	salt, found := os.LookupEnv("SALT")
	if found {
		config.User.Salt = salt
	}

	config.Mongo.FromEnv()

	// Database
	clientStore, err := user.NewMongoStoreClient(&config.Mongo, mongoLogger)
	if err != nil {
		logger.Fatalf("Failed to init mongo: %v", err)
	}
	defer clientStore.Close()

	// Clients
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient := &http.Client{Transport: tr}

	rtr := mux.NewRouter()

	/*
	 * User-Api setup
	 */

	var marketoManager marketo.Manager
	if err := config.User.Marketo.Validate(); err != nil {
		logger.Println("WARNING: Marketo config is invalid", err)
		failedMarketoKeyConfigurationCounter.Inc()
	} else {
		logger.Print("initializing marketo manager")
		marketoManager, err = marketo.NewManager(logger, config.User.Marketo)
		if err != nil {
			logger.Println("WARNING: Marketo Manager not configured;", err)
		}
	}
	userapi := user.InitApi(config.User, logger, &clientStore, auditLogger, marketoManager)
	logger.Print("installing handlers")
	userapi.SetHandlers("", rtr)

	userClient := user.NewUserClient(userapi)

	logger.Print("creating gatekeeper client")

	permsClient := clients.NewGatekeeperClientBuilder().
		WithHost(os.Getenv("GATEKEEPER_SERVICE")).
		WithHttpClient(httpClient).
		WithTokenProvider(userClient).
		Build()

	userapi.AttachPerms(permsClient)

	/*
	 * Serve it up and publish
	 */
	done := make(chan bool)
	logger.Print("creating http server")
	server := common.NewServer(&http.Server{
		Addr:    config.Service.GetPort(),
		Handler: rtr,
	}, logger)

	var start func() error
	if config.Service.Scheme == "https" {
		sslSpec := config.Service.GetSSLSpec()
		start = func() error { return server.ListenAndServeTLS(sslSpec.CertFile, sslSpec.KeyFile) }
	} else {
		start = func() error { return server.ListenAndServe() }
	}

	logger.Print("starting http server")
	if err := start(); err != nil {
		logger.Fatal(err)
	}

	logger.Print("listenting for signals")

	signals := make(chan os.Signal, 40)
	signal.Notify(signals)
	go func() {
		for {
			sig := <-signals
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				server.Close()
				done <- true
			}
		}
	}()

	<-done
}
