package daemon

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"

	"bastionzero.com/bctl-daemon/v1/Daemon/handlers/handleExec"
	"bastionzero.com/bctl-daemon/v1/Daemon/handlers/handleREST"
	"bastionzero.com/bctl-daemon/v1/websocketClient"
)

type contextKey struct {
	key string
}

var ConnContextKey = &contextKey{"http-conn"}

func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

// Main Daemon Thread to execute
func DaemonInit(serviceURL string, authHeader string, sessionId string, assumeRole string, daemonPort string) {
	// Open a Websocket to Bastion
	log.Printf("Opening websocket to Bastion: %s", serviceURL)
	wsClient := websocketClient.NewWebsocketClient(authHeader, sessionId, assumeRole, serviceURL, "")

	go func() {
		// Define our http handlers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			rootCallback(w, r, wsClient)
		})

		// Start the server on the given port
		server := http.Server{
			Addr:        ":" + daemonPort,
			ConnContext: SaveConnInContext,
		}
		log.Fatal(server.ListenAndServe())
	}()
	select {}
}

func rootCallback(w http.ResponseWriter, r *http.Request, wsClient *websocketClient.WebsocketClient) {
	log.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// Determin if its an exec or normal rest
	// For now assume normal
	if strings.Contains(r.URL.Path, "exec") {
		handleExec.HandleExec(w, r, wsClient)
	} else {
		handleREST.HandleREST(w, r, wsClient)
	}
}
