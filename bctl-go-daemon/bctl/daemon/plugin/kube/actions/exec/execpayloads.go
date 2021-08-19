package exec

// const (
// 	// Default timeout for streams
// 	DefaultStreamCreationTimeout = 30 * time.Second

// 	// Enable stdin for remote command execution
// 	ExecStdinParam = "stdin"
// 	// Enable stdout for remote command execution
// 	ExecStdoutParam = "stdout"
// 	// Enable stderr for remote command execution
// 	ExecStderrParam = "stderr"
// 	// Enable TTY for remote command execution
// 	ExecTTYParam = "tty"
// 	// Command to run for remote command execution
// 	ExecCommandParam = "command"

// 	// Name of header that specifies stream type
// 	StreamType = "streamType"
// 	// Value for streamType header for stdin stream
// 	StreamTypeStdin = "stdin"
// 	// Value for streamType header for stdout stream
// 	StreamTypeStdout = "stdout"
// 	// Value for streamType header for stderr stream
// 	StreamTypeStderr = "stderr"
// 	// Value for streamType header for data stream
// 	StreamTypeData = "data"
// 	// Value for streamType header for error stream
// 	StreamTypeError = "error"
// 	// Value for streamType header for terminal resize stream
// 	StreamTypeResize = "resize"

// 	// Name of header that specifies the port being forwarded
// 	PortHeader = "port"
// 	// Name of header that specifies a request ID used to associate the error
// 	// and data streams for a single forwarded connection
// 	PortForwardRequestIDHeader = "requestID"
// )

// TerminalSize represents the width and height of a terminal.
type TerminalSize struct {
	Width  uint16
	Height uint16
}

type resizeCallback func(TerminalSize)

// TerminalSizeQueue is capable of returning terminal resize events as they occur.
type TerminalSizeQueue interface {
	// Next returns the new terminal size after the terminal has been resized. It returns nil when
	// monitoring has been stopped.
	Next() *TerminalSize
}

// type streamAndReply struct {
// 	httpstream.Stream
// 	replySent <-chan struct{}
// }

// type StatusError struct {
// 	ErrStatus metav1.Status
// }

// type remoteCommandProxy struct {
// 	conn         io.Closer
// 	stdinStream  io.ReadCloser
// 	stdoutStream io.WriteCloser
// 	stderrStream io.WriteCloser
// 	writeStatus  func(status *StatusError) error
// 	resizeStream io.ReadCloser
// 	tty          bool
// }

// type Options struct {
// 	Stdin           bool
// 	Stdout          bool
// 	Stderr          bool
// 	TTY             bool
// 	ExpectedStreams int
// 	Command         []string
// }
