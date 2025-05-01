package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Request struct {
	Message string `json:"message"`
}

type Response struct {
	Status string `json:"status"`
}

type LoggerConfig struct {
	LogLevel     string `json:"logLevel"`
	GreetMessage string `json:"greetMessage"`
	Port         string `json:"port"`
}

var (
	jsonContent = "application/json"
	textContent = "text/plain; charset=utf-8"

	logMutex    = &sync.Mutex{}
	logFilePath = "/app/logs/app.log"

	logConfig LoggerConfig
	log       = logrus.New()
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	scheduleConfigReload(5 * time.Second)

	http.HandleFunc("GET /", defaultHandler)
	http.HandleFunc("GET /status", statusHandler)
	http.HandleFunc("POST /log", addLogHandler)
	http.HandleFunc("GET /logs", returnLogsHandler)

	addr := "0.0.0.0:" + logConfig.Port
	fmt.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}

// Config loads

func loadConfig() error {
	configPath := "/app/config/config.json"
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Errorf("Unable to open config file \"%s\": %v", configPath, err)
		return err
	}

	if err := json.Unmarshal(configData, &logConfig); err != nil {
		log.Errorf("Invalid JSON config for logger: %v", err)
		return err
	}

	if logConfig.GreetMessage == "" {
		logConfig.GreetMessage = "Welcome to the custom logger app!"
	}

	if logConfig.GreetMessage[0] != '[' {
		hostname, _ := os.Hostname()
		logConfig.GreetMessage = fmt.Sprintf("[%s] %s", hostname, logConfig.GreetMessage)
	}

	if logConfig.LogLevel == "" {
		logConfig.LogLevel = "info"
	}

	if logConfig.Port == "" {
		logConfig.Port = "8080"
	}

	level, err := logrus.ParseLevel(logConfig.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	log.Infof("Config loaded successfully. LogLevel=\"%s\", greeting=\"%s\", port=\"%s\"",
		level.String(), logConfig.GreetMessage, logConfig.Port)

	return nil
}

func scheduleConfigReload(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := loadConfig(); err != nil {
				log.Errorf("Error reloading config: %v", err)
			} else {
				log.Infof("Config reloaded")
			}
		}
	}()
}

// Handlers

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second)
	responseWithData(w, http.StatusOK, textContent, []byte(logConfig.GreetMessage))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(Response{Status: "ok"})
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}
	responseWithData(w, http.StatusOK, jsonContent, data)
}

func addLogHandler(w http.ResponseWriter, r *http.Request) {
	var logRequest Request
	if json.NewDecoder(r.Body).Decode(&logRequest) != nil {
		responseWithError(w,
			http.StatusBadRequest,
			errors.New("invalid JSON with log message in request body"),
		)
		return
	}

	if logRequest.Message == "" {
		responseWithError(w,
			http.StatusBadRequest,
			errors.New("empty JSON with log message in request body"),
		)
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}
	defer logFile.Close()

	logFile.WriteString(logRequest.Message + "\n")

	data, err := json.Marshal(Response{Status: "message logged"})
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}
	responseWithData(w, http.StatusOK, jsonContent, data)
}

func returnLogsHandler(w http.ResponseWriter, r *http.Request) {
	logMutex.Lock()
	defer logMutex.Unlock()

	logs, err := os.ReadFile(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			responseWithData(w, http.StatusOK, textContent, []byte("No logs recorded"))
			return
		}
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}
	responseWithData(w, http.StatusOK, textContent, logs)
}

// Response helpers

func responseWithData(w http.ResponseWriter, status int, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	w.Write(body)
}

func responseWithError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}
