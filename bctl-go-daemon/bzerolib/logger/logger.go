package logger

import (
	"os"

	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

// Keep track of the different parts of our agent we can log
type Component string

const (
	Websocket      Component = "websocket"
	Datachannel    Component = "datachannel"
	Controlchannel Component = "controlchannel"
	Plugin         Component = "plugin"
	Action         Component = "action"
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

const (
	// TODO: Detect os and switch
	logFilePath = "/var/log/cwc/bctl-agent.log"
)

func NewLogger(component Component, debugLevel DebugLevel) *Logger {
	// Let's us display stack info on errors
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(debugLevel)

	// If the log file doesn't exist, create it, or append to the file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal()
	}

	return &Logger{
		logger: zerolog.New(logFile).With().Str(string(component), uuid.New().String()).Logger(),
	}
}

func (l *Logger) GetWebsocketSubLogger() *Logger {
	return &Logger{
		logger: l.logger.With().Str("websocket", uuid.New().String()).Logger(),
	}
}

func (l *Logger) GetPluginSubLogger(pluginName plgn.PluginName) *Logger {
	return &Logger{
		logger: l.logger.With().Str("plugin", string(pluginName)).Logger(),
	}
}

func (l *Logger) GetActionSubLogger(actionName string) *Logger {
	return &Logger{
		logger: l.logger.With().
			Str("action", actionName).
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
