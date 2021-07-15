package main

import (
	"flag"
	"log"
	"os"

	daemon "bastionzero.com/bctl-daemon/v1/Daemon"
	server "bastionzero.com/bctl-daemon/v1/Server"
)

func main() {
	// Define our command line args
	sessionIdPtr := flag.String("sessionId", "", "Session ID From Zli")
	assumeRolePtr := flag.String("assumeRole", "", "Assume role if we are running as a Daemon")
	clientIdentifierPtr := flag.String("clientIdentifier", "", "Client Identifier to use if we are running as the server")
	authHeaderPtr := flag.String("authHeader", "", "Auth Header From Zli")
	assumeClusterPtr := flag.String("assumeCluster", "", "Cluster we are trying to connect to") // TODO: This is currently unused
	daemonPortPtr := flag.String("daemonPort", "", "Daemon port we should run on")
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")
	if *assumeClusterPtr == "" || *assumeRolePtr == "" || *daemonPortPtr == "" {
	}

	// Try to parse some from the env
	if *sessionIdPtr == "" {
		*sessionIdPtr = os.Getenv("SESSION_ID")
	}

	if *authHeaderPtr == "" {
		*authHeaderPtr = os.Getenv("AUTH_HEADER")
	}
	if *clientIdentifierPtr == "" {
		*clientIdentifierPtr = os.Getenv("CLIENT_IDENTIFIER")
	}
	if *serviceURLPtr == "" {
		*serviceURLPtr = os.Getenv("SERVICE_URL")
	}

	// Parse and ensure they all exist
	flag.Parse()
	if *sessionIdPtr == "" || *serviceURLPtr == "" || *authHeaderPtr == "" {
		// TODO: This should be better parsing
		log.Printf("Error, some required vars not passed \n")
		os.Exit(1)
	}

	if *assumeRolePtr != "" {
		// This means we are the Daemon
		// Need to check for required vars, daemonPort, assumeCluster
		daemon.DaemonInit(*serviceURLPtr, *authHeaderPtr, *sessionIdPtr, *assumeRolePtr, *daemonPortPtr)
	} else if *clientIdentifierPtr != "" {
		server.ServerInit(*serviceURLPtr, *authHeaderPtr, *sessionIdPtr, *clientIdentifierPtr)
	}
}
