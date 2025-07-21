# Health Proxy Sidecar

A lightweight HTTP proxy container to forward health-check requests to a TLS-protected upstream service. Ideal as a
Kubernetes sidecar to enable HTTP-based liveness/readiness probes for services exposing only HTTPS endpoints.

## Features

* Forwards incoming HTTP requests to a configurable HTTPS upstream endpoint
* Optional custom CA trust via mounted PEM certificate
* Path and method whitelisting for security
* Graceful shutdown on SIGINT/SIGTERM
* Simple, zero-dependency Go binary

## Configuration

| Variable                   | Default                  | Description                                                                               |
|----------------------------|--------------------------|-------------------------------------------------------------------------------------------|
| `LISTEN_ADDR`              | `:8082`                  | Address and port for the proxy to listen on                                               |
| `UPSTREAM_ENDPOINT`        | `https://127.0.0.1:8281` | Target HTTPS endpoint to proxy requests to                                                |
| `UPSTREAM_ALLOW_PATHS`     | -                        | Comma-separated list of allowed paths (e.g., `/health,/ready`)                            |
| `ALLOWED_METHODS`          | -                        | Comma-separated list of allowed HTTP methods (e.g., `GET,POST`)                           |
| `CA_CERT_PATH`             | -                        | Path to CA certificate file for TLS verification. If not set, TLS verification is skipped |
| `REQUEST_TIMEOUT_DURATION` | `1s`                     | Timeout for upstream requests (Go duration format, e.g., `5s`, `100ms`)                   |

**Note:** Both `UPSTREAM_ALLOW_PATHS` and `ALLOWED_METHODS` must be configured to allow any requests through the proxy.

## Example

Mount the CA cert and configure an HTTP probe:

```yaml
containers:
  - name: app
    image: your-registry/app:latest
    # ...
  - name: ghcr.io/jschlarb/health-proxy/health-proxy:v0.1.0
    image: gcr
    env:
      - name: UPSTREAM_ENDPOINT
        value: "https://127.0.0.1:8443"
      - name: UPSTREAM_ALLOW_PATHS
        value: "/health,/ready,/metrics"
      - name: ALLOWED_METHODS
        value: "GET,POST"
      - name: CA_CERT_PATH
        value: "/etc/healthproxy/ca.crt"
      - name: REQUEST_TIMEOUT_DURATION
        value: "5s"
    volumeMounts:
      - name: ca-cert
        mountPath: /etc/healthproxy
volumes:
  - name: ca-cert
    secret:
      secretName: my-service-ca

readinessProbe:
  httpGet:
    scheme: HTTP
    path: /status/health
    port: 8082
  initialDelaySeconds: 10
  periodSeconds: 10
```
