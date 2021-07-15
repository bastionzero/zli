package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
)

type contextKey struct {
	key string
}

// Types for exec ->
// TerminalSize represents the width and height of a terminal.
type TerminalSize struct {
	Width  uint16
	Height uint16
}

// TerminalSizeQueue is capable of returning terminal resize events as they occur.
type TerminalSizeQueue interface {
	// Next returns the new terminal size after the terminal has been resized. It returns nil when
	// monitoring has been stopped.
	Next() *TerminalSize
}

type resizeCallback func(TerminalSize)

type termQueue struct {
	ch       chan TerminalSize
	cancel   context.CancelFunc
	done     context.Context
	onResize resizeCallback
}

type StatusError struct {
	ErrStatus metav1.Status
}

// APIStatus is exposed by errors that can be converted to an api.Status object
// for finer grained details.
type APIStatus interface {
	Status() metav1.Status
}

type remoteCommandProxy struct {
	conn         io.Closer
	stdinStream  io.ReadCloser
	stdoutStream io.WriteCloser
	stderrStream io.WriteCloser
	writeStatus  func(status *StatusError) error
	resizeStream io.ReadCloser
	tty          bool
	resizeQueue  *termQueue
}

type StreamOptions struct {
	Stdin             io.Reader
	Stdout            io.Writer
	Stderr            io.Writer
	Tty               bool
	TerminalSizeQueue TerminalSizeQueue
}

func (s *remoteCommandProxy) options() StreamOptions {
	opts := StreamOptions{
		Stdout: s.stdoutStream,
		Stdin:  s.stdinStream,
		Stderr: s.stderrStream,
		Tty:    s.tty,
	}
	// TODO: Broken Ref: https://github.com/gravitational/teleport/blob/7bff7c41bda0f36898e9063aaacd5539ce062489/lib/kube/proxy/remotecommand.go#L247
	// // done to prevent this problem: https://golang.org/doc/faq#nil_error
	// if s.resizeQueue != nil {
	// 	opts.TerminalSizeQueue = s.resizeQueue
	// }
	return opts
}

func (s *remoteCommandProxy) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *remoteCommandProxy) sendStatus(err error) error {
	if err == nil {
		return s.writeStatus(&StatusError{ErrStatus: metav1.Status{
			Status: metav1.StatusSuccess,
		}})
	} else {
		fmt.Println("Unhandled error!")
	}
	return s.writeStatus(&StatusError{ErrStatus: metav1.Status{
		Status: metav1.StatusSuccess,
	}})
	// if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
	// 	rc := exitErr.ExitStatus()
	// 	return s.writeStatus(&apierrors.StatusError{ErrStatus: metav1.Status{
	// 		Status: metav1.StatusFailure,
	// 		Reason: remotecommandconsts.NonZeroExitCodeReason,
	// 		Details: &metav1.StatusDetails{
	// 			Causes: []metav1.StatusCause{
	// 				{
	// 					Type:    remotecommandconsts.ExitCodeCauseType,
	// 					Message: fmt.Sprintf("%d", rc),
	// 				},
	// 			},
	// 		},
	// 		Message: fmt.Sprintf("command terminated with non-zero exit code: %v", exitErr),
	// 	}})
	// }
	// err = trace.BadParameter("error executing command in container: %v", err)
	// return s.writeStatus(apierrors.NewInternalError(err))
}

// Api params
const (
	// Default timeout for streams
	DefaultStreamCreationTimeout = 30 * time.Second

	// Enable stdin for remote command execution
	ExecStdinParam = "stdin"
	// Enable stdout for remote command execution
	ExecStdoutParam = "stdout"
	// Enable stderr for remote command execution
	ExecStderrParam = "stderr"
	// Enable TTY for remote command execution
	ExecTTYParam = "tty"
	// Command to run for remote command execution
	ExecCommandParam = "command"

	// Name of header that specifies stream type
	StreamType = "streamType"
	// Value for streamType header for stdin stream
	StreamTypeStdin = "stdin"
	// Value for streamType header for stdout stream
	StreamTypeStdout = "stdout"
	// Value for streamType header for stderr stream
	StreamTypeStderr = "stderr"
	// Value for streamType header for data stream
	StreamTypeData = "data"
	// Value for streamType header for error stream
	StreamTypeError = "error"
	// Value for streamType header for terminal resize stream
	StreamTypeResize = "resize"

	// Name of header that specifies the port being forwarded
	PortHeader = "port"
	// Name of header that specifies a request ID used to associate the error
	// and data streams for a single forwarded connection
	PortForwardRequestIDHeader = "requestID"
)

type streamAndReply struct {
	httpstream.Stream
	replySent <-chan struct{}
}

var ConnContextKey = &contextKey{"http-conn"}

func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}
func GetConn(r *http.Request) net.Conn {
	return r.Context().Value(ConnContextKey).(net.Conn)
}

func main() {
	http.HandleFunc("/", myHandler)

	server := http.Server{
		Addr:        ":1234",
		ConnContext: SaveConnInContext,
	}
	server.ListenAndServe()
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// First determine if this is a post exec request
	if strings.Contains(r.URL.Path, "exec") {
		fmt.Println("Handling exec")

		// Extract the options of the exec
		options := extractExecOptions(r)
		fmt.Printf("Running command: %s\n", options.Command)

		// Initiate a handshake and upgrade the request
		supportedProtocols := []string{"v4.channel.k8s.io", "v3.channel.k8s.io", "v2.channel.k8s.io", "channel.k8s.io"}
		protocol, err := httpstream.Handshake(r, w, supportedProtocols)
		fmt.Printf("Using protocol: %s\n", protocol)

		if err != nil {
			fmt.Println("FATAL ERROR!")
			return
		}

		streamCh := make(chan streamAndReply)
		upgrader := spdy.NewResponseUpgrader()
		conn := upgrader.UpgradeResponse(w, r, func(stream httpstream.Stream, replySent <-chan struct{}) error {
			streamCh <- streamAndReply{Stream: stream, replySent: replySent}
			return nil
		})

		if conn == nil {
			// The upgrader is responsible for notifying the client of any errors that
			// occurred during upgrading. All we can do is return here at this point
			// if we weren't successful in upgrading.
			fmt.Println("FATAL ERROR!")
			return
		}

		conn.SetIdleTimeout(time.Minute)

		// TODO?
		// var handler protocolHandler
		// switch protocol {
		// case "":
		// 	log.Warningf("Client did not request protocol negotiation.")
		// 	fallthrough
		// case StreamProtocolV4Name:
		// 	log.Infof("Negotiated protocol %v.", protocol)
		// 	handler = &v4ProtocolHandler{}
		// default:
		// 	return nil, trace.BadParameter("protocol %v is not supported. upgrade the client", protocol)
		// }

		expired := time.NewTimer(DefaultStreamCreationTimeout)
		defer expired.Stop()

		proxy, err := waitForStreams(r.Context(), streamCh, options.ExpectedStreams, expired.C)
		if err != nil {
			fmt.Println("FATAL ERROR!")
			return
		}

		proxy.conn = conn
		proxy.tty = options.TTY

		// TODO: Handle resize stream
		// Ref: https://github.com/gravitational/teleport/blob/7bff7c41bda0f36898e9063aaacd5539ce062489/lib/kube/proxy/remotecommand.go#L159
		// if proxy.resizeStream != nil {
		// 	proxy.resizeQueue = newTermQueue(r.Context(), r.onResize)
		// 	go proxy.resizeQueue.handleResizeEvents(proxy.resizeStream)
		// }

		defer proxy.conn.Close()

		// rd := bufio.NewReader(proxy.stdinStream)
		// for {
		// 	fmt.Println("Starting timer")
		// 	time.Sleep(time.Second * 2)
		// 	fmt.Println("Timer finished")

		// 	body, err := ioutil.ReadAll(proxy.stdinStream)
		// 	// str, _, err := rd.ReadByte()
		// 	// str, err := rd.ReadAll()
		// 	if err != nil {
		// 		fmt.Printf("Read Error: %v\n", err)
		// 		return
		// 	}
		// 	fmt.Println(body)
		// }
		buf := make([]byte, 16)
		for {
			n, err := proxy.stdinStream.Read(buf)
			// toPrint := string(buf[:n])
			word := string(buf[:n])
			fmt.Println(word == "")
			proxy.stdoutStream.Write(buf)
			fmt.Println(string(buf[:n]))
			fmt.Println(buf)
			if err == io.EOF {
				break
			}
		}

		// proxy.stdoutStream.Write([]byte("hello world! Custom text!"))

		// buf := new(bytes.Buffer)
		// buf.ReadFrom(proxy.stdinStream)
		// newStr := buf.String()

		// fmt.Println(newStr)
		// proxy.stdoutStream.Write([]byte("test?"))

		// streamOptions := proxy.options()
		// trackIn := NewTrackingReader(streamOptions.Stdin)
		// if streamOptions.Stdin != nil {
		// 	streamOptions.Stdin = trackIn
		// }
		// trackOut := NewTrackingWriter(streamOptions.Stdout)
		// if streamOptions.Stdout != nil {
		// 	streamOptions.Stdout = trackOut
		// }
		// trackErr := NewTrackingWriter(streamOptions.Stderr)
		// if streamOptions.Stderr != nil {
		// 	streamOptions.Stderr = trackErr
		// }

		// if recorder != nil {
		// 	// capture stderr and stdout writes to session recorder
		// 	streamOptions.Stdout = utils.NewBroadcastWriter(streamOptions.Stdout, recorder)
		// 	streamOptions.Stderr = utils.NewBroadcastWriter(streamOptions.Stderr, recorder)
		// }

		// Defer a cleanup handler that will mark the stream as complete on exit, regardless of
		// whether it exits successfully, or with an error.
		// NOTE that this cleanup handler MAY MODIFY the returned error value.
		defer func() {
			if err := proxy.sendStatus(err); err != nil {
				fmt.Println("Failed to send status. Exec command was aborted by client.")
			}
		}()

	} else {
		// We must just proxy the request to the bctl-server
		// Make an Http client
		client := &http.Client{}

		// Build our request
		url := "https://bctl-server.bastionzero.com" + r.URL.Path
		req, _ := http.NewRequest(r.Method, url, nil)

		// Add the expected headers
		for name, values := range r.Header {
			// Loop over all values for the name.
			for _, value := range values {
				req.Header.Set(name, value)
			}
		}

		// Add our custom headers
		req.Header.Set("X-KUBE-ROLE-IMPERSONATE", "cwc-dev-developer")
		req.Header.Set("Authorization", "Bearer 1234")

		// Set any query params
		for key, values := range r.URL.Query() {
			for value := range values {
				req.URL.Query().Add(key, values[value])
			}
		}

		// Make the request and wait for the body to close
		res, _ := client.Do(req)
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {

			// bodyString := string(bodyBytes)

			// fmt.Println(r.URL.Query())

			// http.Error(w, http.StatusText(res.StatusCode), res.StatusCode)

			// Loop over all headers, and add them to our response back to kubectl
			for name, values := range res.Header {
				for _, value := range values {
					w.Header().Set(name, value)
				}
			}

			// Get all the body
			bodyBytes, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println("FATAL ERROR!")
			}
			// Write the body to the response to kubectl
			w.Write(bodyBytes)

		} else {
			fmt.Printf("Invalid status code returned from request: %d\n", res.StatusCode)
		}
	}
	// conn.Close()
}

type Options struct {
	Stdin           bool
	Stdout          bool
	Stderr          bool
	TTY             bool
	ExpectedStreams int
	Command         []string
}

func extractExecOptions(r *http.Request) Options {
	tty := r.FormValue(ExecTTYParam) == "true"
	stdin := r.FormValue(ExecStdinParam) == "true"
	stdout := r.FormValue(ExecStdoutParam) == "true"
	stderr := r.FormValue(ExecStderrParam) == "true"

	// count the streams client asked for, starting with 1
	expectedStreams := 1
	if stdin {
		expectedStreams++
	}
	if stdout {
		expectedStreams++
	}
	if stderr {
		expectedStreams++
	}
	if tty { // TODO: && handler.supportsTerminalResizing()
		expectedStreams++
	}

	fmt.Printf("Expectred streams: %d\n", expectedStreams)

	return Options{
		Stdin:           stdin,
		Stdout:          stdout,
		Stderr:          stderr,
		TTY:             tty,
		ExpectedStreams: expectedStreams,
		Command:         r.URL.Query()["command"],
	}
}

type v4ProtocolHandler struct{}

// what does adding (*v4ProtocolHandler) after `func` here do?
func waitForStreams(connContext context.Context, streams <-chan streamAndReply, expectedStreams int, expired <-chan time.Time) (*remoteCommandProxy, error) {
	// Ref: https://github.com/gravitational/teleport/blob/7bff7c41bda0f36898e9063aaacd5539ce062489/lib/kube/proxy/remotecommand.go

	remoteProxy := &remoteCommandProxy{}
	receivedStreams := 0
	replyChan := make(chan struct{})

	stopCtx, cancel := context.WithCancel(connContext)
	defer cancel()

WaitForStreams:
	for {
		select {
		case stream := <-streams:
			streamType := stream.Headers().Get(StreamType)
			fmt.Println("Stream type: " + streamType)
			switch streamType {
			case StreamTypeError:
				remoteProxy.writeStatus = v4WriteStatusFunc(stream)
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStdin:
				remoteProxy.stdinStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStdout:
				remoteProxy.stdoutStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStderr:
				remoteProxy.stderrStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeResize:
				remoteProxy.resizeStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			default:
				fmt.Printf("Ignoring unexpected stream type: %q", streamType)
				// log.Warningf()
			}
		case <-replyChan:
			receivedStreams++
			fmt.Println(receivedStreams)
			fmt.Println(expectedStreams)
			if receivedStreams == expectedStreams {
				break WaitForStreams
			}
		case <-expired:
			return nil, errors.New("timed out waiting for client to create streams")
		case <-connContext.Done():
			return nil, errors.New("onnectoin has dropped, exiting")
		}
	}

	return remoteProxy, nil
}

// waitStreamReply waits until either replySent or stop is closed. If replySent is closed, it sends
// an empty struct to the notify channel.
func waitStreamReply(ctx context.Context, replySent <-chan struct{}, notify chan<- struct{}) {
	select {
	case <-replySent:
		notify <- struct{}{}
	case <-ctx.Done():
	}
}

// v4WriteStatusFunc returns a WriteStatusFunc that marshals a given api Status
// as json in the error channel.
func v4WriteStatusFunc(stream io.Writer) func(status *StatusError) error {
	return func(status *StatusError) error {
		bs, err := json.Marshal(status.ErrStatus)
		if err != nil {
			return err
		}
		_, err = stream.Write(bs)
		return err
	}
}

func newTermQueue(parentContext context.Context, onResize resizeCallback) *termQueue {
	ctx, cancel := context.WithCancel(parentContext)
	return &termQueue{
		ch:       make(chan TerminalSize),
		cancel:   cancel,
		done:     ctx,
		onResize: onResize,
	}
}

// Ref: https://github.com/gravitational/teleport/blob/7bff7c41bda0f36898e9063aaacd5539ce062489/lib/kube/proxy/roundtrip.go#L50
// SpdyRoundTripper knows how to upgrade an HTTP request to one that supports
// multiplexed streams. After RoundTrip() is invoked, Conn will be set
// and usable. SpdyRoundTripper implements the UpgradeRoundTripper interface.
type SpdyRoundTripper struct {
	//tlsConfig holds the TLS configuration settings to use when connecting
	//to the remote server.
	// tlsConfig *tls.Config

	// authCtx authContext

	/* TODO according to http://golang.org/pkg/net/http/#RoundTripper, a RoundTripper
	   must be safe for use by multiple concurrent goroutines. If this is absolutely
	   necessary, we could keep a map from http.Request to net.Conn. In practice,
	   a client will create an http.Client, set the transport to a new insteace of
	   SpdyRoundTripper, and use it a single time, so this hopefully won't be an issue.
	*/
	// conn is the underlying network connection to the remote server.
	conn net.Conn

	// dialWithContext is the function used connect to remote address
	dialWithContext func(context context.Context, network, address string) (net.Conn, error)

	// followRedirects indicates if the round tripper should examine responses for redirects and
	// follow them.
	followRedirects bool

	// ctx is a context for this round tripper
	ctx context.Context

	pingPeriod time.Duration
}

// func getExecutor(req *http.Request) (remotecommand.Executor, error) {
// 	// upgradeRoundTripper := &SpdyRoundTripper{
// 	// 	followRedirects: cfg.followRedirects,
// 	// 	dialWithContext: cfg.dial,
// 	// 	ctx:             req.Context(),
// 	// 	pingPeriod:      time.Second}

// 	rt := http.RoundTripper(req)
// }

// TrackingReader is an io.Reader that counts the total number of bytes read.
// It's thread-safe if the underlying io.Reader is thread-safe.
type TrackingReader struct {
	r     io.Reader
	count uint64
}

// Count returns the total number of bytes read so far.
func (r *TrackingReader) Count() uint64 {
	return atomic.LoadUint64(&r.count)
}

func (r *TrackingReader) Read(b []byte) (int, error) {
	n, _ := r.r.Read(b)
	atomic.AddUint64(&r.count, uint64(n))
	return n, nil
}

// NewTrackingReader creates a TrackingReader around r.
func NewTrackingReader(r io.Reader) *TrackingReader {
	return &TrackingReader{r: r}
}

// TrackingWriter is an io.Writer that counts the total number of bytes
// written.
// It's thread-safe if the underlying io.Writer is thread-safe.
type TrackingWriter struct {
	w     io.Writer
	count uint64
}

// NewTrackingWriter creates a TrackingWriter around w.
func NewTrackingWriter(w io.Writer) *TrackingWriter {
	return &TrackingWriter{w: w}
}

// Count returns the total number of bytes written so far.
func (w *TrackingWriter) Count() uint64 {
	return atomic.LoadUint64(&w.count)
}

func (w *TrackingWriter) Write(b []byte) (int, error) {
	n, _ := w.w.Write(b)
	atomic.AddUint64(&w.count, uint64(n))
	return n, nil
}

// ticker := time.NewTicker(time.Second * 5)
// defer ticker.Stop()

// for {
// 	select {
// 	case <-done:
// 		return ret
// 	case t := <-ticker.C:
// 		err := client.WriteMessage(websocket.TextMessage, []byte(t.String()))
// 		if err != nil {
// 			log.Println("write:", err)
// 			return ret
// 		}
// 	case <-interrupt:
// 		log.Println("interrupt")

// 		// Cleanly close the connection by sending a close message and then
// 		// waiting (with timeout) for the server to close the connection.
// 		err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
// 		fmt.Println("HERE?CLOSE?")
// 		if err != nil {
// 			log.Println("write close:", err)
// 			return ret
// 		}
// 		select {
// 		case <-done:
// 		case <-time.After(time.Second):
// 		}
// 		return ret
// 	}
// }

// if err = client.WriteMessage(websocket.TextMessage, append([]byte(`{
// 	"target": "DataFromClient",
// 	"arguments": [{"logId": "e80d6510-fb36-4de1-9478-397d80ac43d8", "kubeCommand": "test command", "endpoint": "test", "Headers": {}, "Method": "Get", "Body": "test", "RequestIdentifier": 1 }],
// 	"type": 1
//   }`), 0x1E)); err != nil {
// 	return nil
// }

// 	if err = client.Client.WriteMessage(websocket.TextMessage, append([]byte(`{
// 	"target": "DataFromClient",
// 	"arguments": [{"logId": "e80d6510-fb36-4de1-9478-397d80ac43d8", "kubeCommand": "test command", "endpoint": "test", "Headers": {}, "Method": "Get", "Body": "test", "RequestIdentifier": 1 }],
// 	"type": 1
//   }`), 0x1E)); err != nil {
// 		return nil
// 	}
// 	return nil
