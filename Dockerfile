FROM golang:1.22.5-bookworm as build
WORKDIR /app

COPY . .

# Download go modules
RUN go mod download && go mod verify

RUN go build -v -o /nats-http-publisher ./

# FROM gcr.io/distroless/static-debian12
FROM debian:bookworm-slim

COPY --from=build /nats-http-publisher /

EXPOSE 8080

CMD ["/nats-http-publisher"]