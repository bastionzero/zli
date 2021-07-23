package main

import (
	"flag"
	"log"
	"os"

	"bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket"
	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Main ServerInit Thread to execute
// serviceURL string, authHeader string, sessionId string, clientIdentifier string
func main() {
	// Our expected flags we need to start
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")
	activationTokenPtr := flag.String("activationToken", "", "Activation Token to use to register the cluster")

	// Parse any flag
	flag.Parse()

	// The environment will overwrite any flags passed
	*serviceURLPtr = os.Getenv("SERVICE_URL")
	*activationTokenPtr = os.Getenv("ACTIVATION_TOKEN")

	// Ensure we have all needed vars
	if *serviceURLPtr == "" || *activationTokenPtr == "" {
		log.Printf("Missing flags!")
		os.Exit(1)
	}

	wsClient := controlWebsocket.NewControlWebsocketClient(*serviceURLPtr, *activationTokenPtr)

	// Subscribe to our handlers
	go func() {
		for {
			message := <-wsClient.ProvisionWebsocketChan

			// We have an incoming websocket request, attempt to make a new Daemon Websocket Client for the request
			token := "1234" // TODO figure this out
			daemonServerWebsocket.NewDaemonServerWebsocketClient(*serviceURLPtr, message.ConnectionId, token)
		}
	}()

	// Sleep forever
	// Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {}

}
