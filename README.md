# NATS HTTP Publisher

[![Build Docker](https://github.com/nuric/nats-http-publisher/actions/workflows/build-docker.yaml/badge.svg)](https://github.com/nuric/nats-http-publisher/actions/workflows/build-docker.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nuric/nats-http-publisher)](https://goreportcard.com/report/github.com/nuric/nats-http-publisher)
![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/nuric/nats-http-publisher)
![GitHub License](https://img.shields.io/github/license/nuric/nats-http-publisher)

An HTTP publishing bridge for [NATS](https://nats.io/) messaging system. It is a simple HTTP server that listens for incoming HTTP POST requests and publishes the request body to a NATS subject.

## Why?

NATS uses its own protocol to be fast but sometimes there are legacy systems that can only communicate over HTTP. The most common case I encountered in some projects is just to publish messages to a NATS subject from a legacy system.

Or there are cases where only HTTP traffic is allowed through a proxy and direct NATS connection is not possible.

## Getting Started

You can run the server with Docker locally:

```bash
docker run -p 8080:8080 -e NATS_URL=nats://localhost:4222 ghcr.io/nuric/nats-http-publisher:main
```

This will start the server and expose it on port 8080. The port and other options can configuration using environment variables:

```go
type Config struct {
	// Debug enables debug and pretty logging
	Debug bool `env:"DEBUG" envDefault:"false"`
	// Port at which the HTTP server will listen
	Port int `env:"PORT" envDefault:"8080"`
	// NATSURL is the URL of the NATS server
	NATSURL string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	// Name of publisher client, this is used to identify the client to NATS
	Name string `env:"NAME" envDefault:"publisher"`
}
```

## Usage

To publish a message to a NATS subject, you send HTTP POST request to `/publish` with the subject and message in the request body:

```bash
curl -XPOST -H "Content-type: application/json" -d '{"subject": "gondor", "message": "Helms Deep has fallen!"}' 'http://localhost:8080/publish'
```

You can also add [Basic Auth](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) to the request which will match the users authentication in NATS.

After you have a server running, you can use the [sample](sample.http) file to see some example requests that can be made to the server. To make the most of it, install the [REST Client extension](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) which will allow you to make requests directly in the editor and show the results.

### Local Testing

To see the server in action, you can run a local NATS server. There is an example [configuration](nats.conf) file that you can use to run the server.

```bash
docker run --name nats --rm -v ./nats.conf:/etc/nats/nats.conf -p 4222:4222 -p 8222:8222 nats:latest -c /etc/nats/nats.conf
```

This will start a NATS server on port 4222 and expose the monitoring port on 8222. You can use the monitoring port to see the messages being published as well using the [nats cli](https://docs.nats.io/nats-tools/natscli) to listen:

```bash
nats sub gondor
```

You may need to edit the local context using `nats context` to provide user and password to match the server configuration.

## Limitations

- The server only supports publishing messages to NATS. It does not support subscribing to messages, or other patterns. This is initially by design to keep the server simple and focused on the use case. But we can look to extend it.
- The server only supports anonymous or Basic Auth. But it should be easy to extend to JWT or other authentication methods. Look inside the [handlers](handlers.go) file where the authentication is handled.

