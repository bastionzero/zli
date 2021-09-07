package logger

import (
	"os"

	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

// This is here for translation, so that the rest of the program doesn't need to care or know
// about zerolog
type DebugLevel = zerolog.Level

const (
	Debug DebugLevel = zerolog.DebugLevel
	Info  DebugLevel = zerolog.InfoLevel
	Error DebugLevel = zerolog.ErrorLevel
	Trace DebugLevel = zerolog.TraceLevel
)

type Logger struct {
	logger zerolog.Logger
}

func NewLogger(debugLevel DebugLevel, logFilePath string, stdout bool) (*Logger, error) {
	// Let's us display stack info on errors
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(debugLevel)

	// If the log file doesn't exist, create it, or append to the file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("error: %s", err)
		return &Logger{}, err
	}

	if stdout {
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
		multi := zerolog.MultiLevelWriter(consoleWriter, logFile)

		return &Logger{
			logger: zerolog.New(multi).With().Timestamp().Logger(),
		}, nil
	} else {
		return &Logger{
			logger: zerolog.New(logFile).With().Timestamp().Logger(),
		}, nil
	}
}

func (l *Logger) AddAgentVersion(version string) {
	l.logger = l.logger.With().Str("agentVersion", version).Logger()
}

func (l *Logger) AddDaemonVersion(version string) {
	l.logger = l.logger.With().Str("daemonVersion", version).Logger()
}

// TODO: instead of assigning random uuid, the control channel and data channel should both use
// their respective connectionIds
func (l *Logger) GetControlchannelLogger() *Logger {
	return &Logger{
		logger: l.logger.With().Str("controlchannel", uuid.New().String()).Logger(),
	}
}

func (l *Logger) GetDatachannelLogger() *Logger {
	return &Logger{
		logger: l.logger.With().Str("datachannel", uuid.New().String()).Logger(),
	}
}

func (l *Logger) GetWebsocketLogger() *Logger {
	return &Logger{
		logger: l.logger.With().Str("websocket", uuid.New().String()).Logger(),
	}
}

func (l *Logger) GetPluginLogger(pluginName plgn.PluginName) *Logger {
	return &Logger{
		logger: l.logger.With().Str("plugin", string(pluginName)).Logger(),
	}
}

func (l *Logger) GetActionLogger(actionName string) *Logger {
	return &Logger{
		logger: l.logger.With().
			Str("action", actionName).
			Logger(),
	}
}

func (l *Logger) GetComponentLogger(component string) *Logger {
	return &Logger{
		logger: l.logger.With().
			Str("component", component).
			Logger(),
	}
}

func (l *Logger) AddRequestId(rid string) {
	l.logger = l.logger.With().Str("requestId", rid).Logger()
}

func (l *Logger) AddField(key string, value string) {
	l.logger = l.logger.With().Str(key, value).Logger()
}

func (l *Logger) Info(msg string) {
	l.logger.Info().
		Msg(msg)
}

func (l *Logger) Debug(msg string) {
	l.logger.Debug().
		Msg(msg)
}

func (l *Logger) Error(err error) {
	l.logger.Error().
		Stack(). // stack trace for errors woot
		Msg(err.Error())
}

func (l *Logger) Trace(msg string) {
	l.logger.Trace().
		Msg(msg)
}

// ??
// func (l *Logger) Fatal(err error) {
// 	l.logger.Fatal().
// 		Msg(err.Error())
// }
