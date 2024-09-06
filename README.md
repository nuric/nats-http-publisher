# NATS HTTP Publisher

An HTTP publishing bridge for NATS messaging system. It is a simple HTTP server that listens for incoming HTTP POST requests and publishes the request body to a NATS subject.

## Why?

## Getting Started

Running a local NATS server with Docker:
```bash
docker run --name nats --rm -v ./nats.conf:/etc/nats/nats.conf -p 4222:4222 -p 8222:8222 nats:latest -c /etc/nats/nats.conf
```

## Limitations

