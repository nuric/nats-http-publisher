package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// ---------------------------

// Health check handler useful for container health checks
func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	Encode(w, http.StatusOK, map[string]any{"status": "ok"})
}

// ---------------------------

// Single NATS connection, the name corresponds to the authenticated owner such
// as user:pass or anonymous so future requests can reuse the connection.
type natsConn struct {
	conn *nats.Conn
	name string
}

// Application wide handlers that utilise the NATS connections
type AppHandlers struct {
	Config Config
	conns  map[string]natsConn
	mu     sync.RWMutex
}

func NewAppHandlers(conf Config) *AppHandlers {
	return &AppHandlers{
		Config: conf,
		conns:  make(map[string]natsConn),
	}
}

func (app *AppHandlers) establishNATSConnection(name string, opts []nats.Option) (natsConn, error) {
	// ---------------------------
	log.Debug().Str("name", name).Str("url", app.Config.NATSURL).Msg("establishing nats connection")
	// ---------------------------
	commonOpts := []nats.Option{
		// This name is the name of the client as the bridge
		nats.Name(app.Config.Name),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			// These are debug messages because they include user
			// information and won't print during production
			log.Debug().Err(err).Str("name", name).Msg("nats disconnected")
			app.mu.Lock()
			delete(app.conns, name)
			app.mu.Unlock()
		}),
		nats.ConnectHandler(func(nc *nats.Conn) {
			log.Debug().Str("name", name).Msg("nats connected")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Debug().Str("name", name).Msg("nats closed")
			app.mu.Lock()
			delete(app.conns, name)
			app.mu.Unlock()
		}),
	}
	commonOpts = append(commonOpts, opts...)
	// ---------------------------
	nc, err := nats.Connect(app.Config.NATSURL, commonOpts...)
	if err != nil {
		return natsConn{}, fmt.Errorf("could not establish nats connection: %w", err)
	}
	// ---------------------------
	app.mu.Lock()
	newConn := natsConn{
		conn: nc,
		name: name,
	}
	app.conns[name] = newConn
	app.mu.Unlock()
	// ---------------------------
	return newConn, nil
}

func getAuthFromRequest(r *http.Request) (string, []nats.Option) {
	/* If we want to handle additional auth methods, it will most likely be here.
	 * Ask the question, what is the request offering and adjust options for the
	 * nats connection accordingly. */
	user, pass, ok := r.BasicAuth()
	if !ok {
		// Not authenticated, if nats allows anonymous access
		return "_anonymous", nil
	}
	return user + ":" + pass, []nats.Option{nats.UserInfo(user, pass)}
}

func (app *AppHandlers) getConnection(r *http.Request) (nc natsConn, err error) {
	name, opts := getAuthFromRequest(r)
	app.mu.RLock()
	nc, ok := app.conns[name]
	app.mu.RUnlock()
	if !ok {
		nc, err = app.establishNATSConnection(name, opts)
	}
	return
}

func (app *AppHandlers) removeConnection(name string) {
	app.mu.Lock()
	delete(app.conns, name)
	app.mu.Unlock()
}

func (app *AppHandlers) Close() {
	app.mu.Lock()
	for k, nc := range app.conns {
		nc.conn.Close()
		delete(app.conns, k)
	}
	app.mu.Unlock()
}

// ---------------------------

type PublishRequest struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (r PublishRequest) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	if strings.TrimSpace(r.Subject) == "" {
		problems["subject"] = "required"
	}
	if strings.TrimSpace(r.Message) == "" {
		problems["message"] = "required"
	}
	return problems
}

// Publishes a message to a subject by establishing a connection to NATS
func (app *AppHandlers) handlerPublish(w http.ResponseWriter, r *http.Request) {
	// ---------------------------
	// Check if post request
	if r.Method != http.MethodPost {
		Encode(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// ---------------------------
	// Decode and validate request
	req, problems := DecodeValid[PublishRequest](r)
	if len(problems) > 0 {
		Encode(w, http.StatusBadRequest, problems)
		return
	}
	// ---------------------------
	log.Debug().Str("subject", req.Subject).Str("message", req.Message).Msg("publishing")
	// ---------------------------
	// Get or create NATS connection
	nc, err := app.getConnection(r)
	switch {
	case errors.Is(err, nats.ErrAuthorization):
		Encode(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	case err != nil:
		Encode(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	// ---------------------------
	// Publish message
	if err := nc.conn.Publish(req.Subject, []byte(req.Message)); err != nil {
		Encode(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		app.removeConnection(nc.name)
		return
	}
}

// ---------------------------
