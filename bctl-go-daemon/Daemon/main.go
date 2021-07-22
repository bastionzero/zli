package main

import (
	"compress/gzip"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"bastionzero.com/bctl/v1/Daemon/src/DaemonWebsocket"
	"bastionzero.com/bctl/v1/Daemon/src/HandleExec"
	"bastionzero.com/bctl/v1/Daemon/src/HandleREST"

	"github.com/google/uuid"
)

type contextKey struct {
	key string
}

var ConnContextKey = &contextKey{"http-conn"}

func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func main() {
	// TODO: Remove this requirement
	sessionIdPtr := flag.String("sessionId", "", "Session ID From Zli")
	authHeaderPtr := flag.String("authHeader", "", "Auth Header From Zli")

	// Our expected flags we need to start
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")
	assumeRolePtr := flag.String("assumeRole", "", "Kube Role to Assume")
	assumeClusterPtr := flag.String("assumeCluster", "", "Kube Cluster to Connect to")
	daemonPortPtr := flag.String("daemonPort", "", "Daemon Port To Use")
	localhostTokenPtr := flag.String("localhostToken", "", "Localhost Token to Validate Kubectl commands")

	// Parse and TODO: ensure they all exist
	flag.Parse()

	// Open a Websocket to Bastion
	log.Printf("Opening websocket to Bastion: %s", *serviceURLPtr)
	wsClient := DaemonWebsocket.NewDaemonWebsocketClient(*sessionIdPtr, *authHeaderPtr, *serviceURLPtr, *assumeRolePtr, *assumeClusterPtr)

	go func() {
		// Define our http handlers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			rootCallback(w, r, *localhostTokenPtr, wsClient)
		})

		// Start the server on the given port
		// server := http.Server{
		// 	Addr:        ":" + *daemonPortPtr,
		// 	ConnContext: SaveConnInContext,
		// }

		// Open our key/cert
		// keyFile, _ := ioutil.ReadFile("/Users/sidpremkumar/Library/Preferences/bastionzero-zli-nodejs/kubeKey.pem")
		// certFile, _ := ioutil.ReadFile("/Users/sidpremkumar/Library/Preferences/bastionzero-zli-nodejs/kubeCert.pem")

		log.Fatal(http.ListenAndServeTLS(":"+*daemonPortPtr, "/Users/sidpremkumar/Library/Preferences/bastionzero-zli-nodejs/kubeCert.pem", "/Users/sidpremkumar/Library/Preferences/bastionzero-zli-nodejs/kubeKey.pem", nil))
	}()
	select {}
}

func rootCallback(w http.ResponseWriter, r *http.Request, localhostToken string, wsClient *DaemonWebsocket.DaemonWebsocket) {
	log.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// Trim off and localhost token
	// TODO: Fix this
	localhostToken = strings.Replace(localhostToken, "++++", "", -1)

	// First verify our token and extract any commands if we can
	tokenToValidate := r.Header.Get("Authorization")
	tokenToValidate = strings.Replace(tokenToValidate, "Bearer ", "", -1)

	// Remove the `Bearer `
	tokensSplit := strings.Split(tokenToValidate, "++++")

	// Validate the token
	if tokensSplit[0] != localhostToken {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check if we have a command to extract
	// TODO: Maybe we can push this work to the bastion
	commandBeingRun := "N/A"
	logId := "N/A"
	if len(tokensSplit) == 3 {
		commandBeingRun = tokensSplit[1]
		logId = tokensSplit[2]
	} else {
		commandBeingRun = "N/A"
		logId = uuid.New().String()
	}

	// Determin if its an exec or normal rest
	if strings.Contains(r.URL.Path, "exec") {
		HandleExec.HandleExec(w, r, wsClient)
	} else {
		HandleREST.HandleREST(w, r, commandBeingRun, logId, wsClient)
	}
}
