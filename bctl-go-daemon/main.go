package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"bastionzero.com/bctl-daemon/v1/handlers/handleREST"
	"bastionzero.com/bctl-daemon/v1/websocketClient"
)

type contextKey struct {
	key string
}

var ConnContextKey = &contextKey{"http-conn"}

func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

func main() {
	// Define our command line args
	sessionIdPtr := flag.String("sessionId", "", "Session ID From Zli")
	authHeaderPtr := flag.String("authHeader", "", "Auth Header From Zli")
	assumeRolePtr := flag.String("assumeRole", "", "Role we are trying to assume")
	assumeClusterPtr := flag.String("assumeCluster", "", "Cluster we are trying to connect to")
	daemonPortPtr := flag.String("daemonPort", "", "Daemon port we should run on")
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")

	// Parse and ensure they all exist
	flag.Parse()
	if *sessionIdPtr == "" || *assumeRolePtr == "" || *assumeClusterPtr == "" || *daemonPortPtr == "" || *serviceURLPtr == "" || *authHeaderPtr == "" {
		// TODO: This should be better parsing
		log.Printf("Error, some required vars not passed \n")
		os.Exit(1)
	}

	// Open a Websocket to Bastion
	log.Printf("Opening websocket to Bastion: %s", *serviceURLPtr)

	wsClient := websocketClient.NewWebsocketClient(*authHeaderPtr, *sessionIdPtr, *assumeRolePtr, *serviceURLPtr)

	// Imagine some incoming message coming in
	// dataFromClientMessage := new(websocketClientTypes.DataFromClientMessage)
	// dataFromClientMessage.LogId = "e80d6510-fb36-4de1-9478-397d80ac43d8"
	// dataFromClientMessage.KubeCommand = "bctl test"
	// dataFromClientMessage.Endpoint = "/test"
	// dataFromClientMessage.Headers = nil
	// dataFromClientMessage.Method = "Get"
	// dataFromClientMessage.Body = "test"
	// dataFromClientMessage.RequestIdentifier = 1

	// // First send that message to Bastion
	// wsClient.SendDataFromClientMessage(*dataFromClientMessage)

	// Wait for the response
	// dataToClientMessage := new(DataToClientMessage)
	// dataToClientMessage := <-wsClient.DataToClientChan
	// fmt.Println(dataToClientMessage)

	// time.Sleep(time.Second * 3)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rootCallback(w, r, *wsClient)
	})

	server := http.Server{
		Addr:        ":" + *daemonPortPtr,
		ConnContext: SaveConnInContext,
	}
	log.Fatal(server.ListenAndServe())
}

func rootCallback(w http.ResponseWriter, r *http.Request, wsClient websocketClient.WebsocketClient) {
	log.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// Determin if its an exec or normal rest
	// For now assume normal
	handleREST.HandleREST(w, r, wsClient)
}
