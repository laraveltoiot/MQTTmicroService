package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MQTTmicroService/internal/api"
	"MQTTmicroService/internal/auth"
	"MQTTmicroService/internal/broker"
	"MQTTmicroService/internal/config"
	"MQTTmicroService/internal/database"
	"MQTTmicroService/internal/logger"
	"MQTTmicroService/internal/metrics"
	"MQTTmicroService/internal/mqtt"
)

func main() {
	// Parse command line flags
	httpAddr := flag.String("http-addr", ":8080", "HTTP server address")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", "text", "Log format (text, json)")
	logFile := flag.String("log-file", "mqtt-service.log", "Log file path")
	enableFileLogging := flag.Bool("file-logging", true, "Enable logging to file")
	flag.Parse()

	// Initialize logger
	var log *logger.Logger
	var err error

	if *enableFileLogging {
		// Create file logger
		logConfig := &logger.Config{
			Level:      *logLevel,
			Format:     *logFormat,
			TimeFormat: "2006-01-02 15:04:05",
		}
		log, err = logger.NewFileLogger(*logFile, logConfig)
		if err != nil {
			// Fall back to console logging if file logging fails
			fmt.Printf("Failed to initialize file logger: %v, falling back to console logger\n", err)
			log = logger.NewConsoleLogger(logConfig)
		} else {
			// Also log to console
			consoleLog := logger.NewConsoleLogger(logConfig)
			fmt.Printf("Logging to file: %s\n", *logFile)
			// Use console logger for initial startup message
			consoleLog.Info("Starting MQTT microservice")
		}
	} else {
		// Create console logger
		logConfig := &logger.Config{
			Level:      *logLevel,
			Format:     *logFormat,
			Output:     os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
		}
		log = logger.New(logConfig)
	}

	log.Info("Starting MQTT microservice")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize metrics collector
	metricsCollector := metrics.New(log)
	log.Info("Metrics collector initialized")

	// Initialize authentication service
	authConfig := &auth.Config{
		EnableAPIKey: cfg.EnableAPIKey,
		APIKeys:      cfg.APIKeys,
	}
	authService := auth.New(authConfig, log)
	log.WithField("enableAPIKey", cfg.EnableAPIKey).Info("Authentication service initialized")

	// Initialize database
	var db database.Database
	if cfg.Database != nil && cfg.Database.Type != "" {
		log.WithField("type", cfg.Database.Type).Info("Initializing database")

		// Create database instance
		var err error
		dbConfig := &database.Config{
			Type:       cfg.Database.Type,
			Connection: cfg.Database.Connection,
		}

		// Copy MongoDB settings
		dbConfig.MongoDB.URI = cfg.Database.MongoDB.URI
		dbConfig.MongoDB.Database = cfg.Database.MongoDB.Database
		dbConfig.MongoDB.Username = cfg.Database.MongoDB.Username
		dbConfig.MongoDB.Password = cfg.Database.MongoDB.Password
		dbConfig.MongoDB.Port = cfg.Database.MongoDB.Port

		// Copy SQLite settings
		dbConfig.SQLite.Path = cfg.Database.SQLite.Path

		db, err = database.New(dbConfig)
		if err != nil {
			log.WithError(err).Fatal("Failed to create database instance")
		}

		// Connect to database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.Connect(ctx); err != nil {
			log.WithError(err).Fatal("Failed to connect to database")
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := db.Close(ctx); err != nil {
				log.WithError(err).Error("Failed to close database connection")
			}
		}()

		log.Info("Connected to database")
	} else {
		log.Warn("No database configuration found, messages will not be stored")
	}

	// Initialize MQTT client manager
	mqttManager := mqtt.NewManager(cfg, log, metricsCollector, db)

	// Connect to default MQTT broker
	defaultClient, err := mqttManager.GetDefaultClient()
	if err != nil {
		log.WithError(err).Fatal("Failed to get default MQTT client")
	}

	if err := defaultClient.Connect(); err != nil {
		log.WithError(err).Fatal("Failed to connect to default MQTT broker")
	}
	defer defaultClient.Disconnect()

	log.WithField("broker", cfg.DefaultConnection).Info("Connected to default MQTT broker")

	// Initialize MQTT broker if enabled
	var mqttBroker *broker.Broker
	if cfg.MQTTBroker != nil && cfg.MQTTBroker.Enable {
		// Convert config.MQTTBrokerConfig to broker.Config
		brokerConfig := &broker.Config{
			Enable:         cfg.MQTTBroker.Enable,
			Host:           cfg.MQTTBroker.Host,
			Port:           cfg.MQTTBroker.Port,
			TLSEnable:      cfg.MQTTBroker.TLSEnable,
			TLSCertFile:    cfg.MQTTBroker.TLSCertFile,
			TLSKeyFile:     cfg.MQTTBroker.TLSKeyFile,
			AuthEnable:     cfg.MQTTBroker.AuthEnable,
			AllowAnonymous: cfg.MQTTBroker.AllowAnonymous,
			Credentials:    cfg.MQTTBroker.Credentials,
			EnableLogging:  cfg.MQTTBroker.EnableLogging,
		}

		var err error
		mqttBroker, err = broker.New(brokerConfig, log)
		if err != nil {
			log.WithError(err).Fatal("Failed to create MQTT broker")
		}

		// Start the broker
		if err := mqttBroker.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start MQTT broker")
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := mqttBroker.Stop(ctx); err != nil {
				log.WithError(err).Error("Failed to stop MQTT broker")
			}
		}()

		log.WithFields(map[string]interface{}{
			"host": cfg.MQTTBroker.Host,
			"port": cfg.MQTTBroker.Port,
		}).Info("MQTT broker started")
	}

	// Initialize HTTP API server
	apiServer := api.NewServer(mqttManager, log, metricsCollector, authService, db, cfg, mqttBroker, *httpAddr)

	// Start HTTP server in a goroutine
	go func() {
		if err := apiServer.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	log.WithField("addr", *httpAddr).Info("HTTP server started")

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")

	// Create a deadline to wait for (not used in this simple implementation)
	// but would be used in a more complex shutdown process
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline
	if err := apiServer.Stop(); err != nil {
		log.WithError(err).Error("Error shutting down HTTP server")
	}

	log.Info("Server gracefully stopped")
}
