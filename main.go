// Keep it simple and stupid sidecar - just proxy the request, and return nothing expect the http code
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MariusSchmidt/slf4go/slf4go_logrus_provider"
	"github.com/sirupsen/logrus"
)

var log = slf4go_logrus_provider.New(logrus.StandardLogger()).ForComponent("health-proxy")

func main() {
	listenAddr := getEnv("LISTEN_ADDR", ":8082")
	upstreamEndpoint := getEnv("UPSTREAM_ENDPOINT", "https://127.0.0.1:8281")
	upstreamAllowPaths := splitEnvByComma("UPSTREAM_ALLOW_PATHS")
	upstreamAllowMethods := splitEnvByComma("ALLOWED_METHODS")
	caCertPath := getEnv("CA_CERT_PATH", "")
	timeout, err := time.ParseDuration(getEnv("REQUEST_TIMEOUT_DURATION", "1s"))

	if err != nil {
		log.Fatalf("Failed to parse timeout duration: %v", err)
	}

	if caCertPath == "" {
		log.Infof("CA_CERT_PATH environment variable not set, using insecure client")
	}

	srv := &http.Server{
		Addr: listenAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tlsConfig := &tls.Config{}

			if caCertPath != "" {
				caPool := x509.NewCertPool()
				caCert, err := os.ReadFile(caCertPath)
				if err != nil {
					log.Errorf("Unable to read cert: %v", err)
					http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
					return
				}
				caPool.AppendCertsFromPEM(caCert)

				tlsConfig.RootCAs = caPool

			} else {
				tlsConfig.InsecureSkipVerify = true
			}

			if !upstreamAllowPaths[r.URL.Path] {
				log.Warnf("Path not allowed: %s: %s", r.Method, r.URL.Path)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			if !upstreamAllowMethods[r.Method] {
				log.Warnf("Method not allowed: %s: %s", r.Method, r.URL.Path)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
				Timeout: timeout,
			}

			url := fmt.Sprintf("%s%s", strings.TrimSuffix(upstreamEndpoint, "/"), r.URL.Path)
			req, err := http.NewRequest(r.Method, url, r.Body)
			if err != nil {
				log.Errorf("Error while creating client: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			req.Header = r.Header.Clone()

			resp, err := client.Do(req)
			if err != nil {
				log.Errorf("Error while requesting from upstream: %v", err)
				http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				return
			}
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(resp.Body)

			for k, v := range resp.Header {
				if k != "Content-Length" {
					w.Header()[k] = v
				}
			}

			w.WriteHeader(resp.StatusCode)

			_, _ = io.Copy(w, resp.Body)
		}),
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigChan
	log.Infof("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitEnvByComma(key string) map[string]bool {
	s := make(map[string]bool)
	if value := os.Getenv(key); value != "" {
		for _, path := range strings.Split(value, ",") {
			s[strings.TrimSpace(path)] = true
		}
	}
	return s
}
