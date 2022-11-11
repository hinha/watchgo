package logger

import (
	"fmt"
	"github.com/hinha/watchgo/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"strings"
)

var (
	Logger            = zerolog.Logger{}
	defaultFormatTime = "2006-01-02T15:04:05+07:00"
	globalFormatTime  = "2006/01/02 15:04:05.000"
)

func SetGlobalLogger(log zerolog.Logger) {
	Logger = log
}

func New() zerolog.Logger {
	var writers []io.Writer

	zerolog.TimeFieldFormat = globalFormatTime
	console := zerolog.ConsoleWriter{Out: os.Stderr}
	console.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%-6s", i))
	}
	writers = append(writers, console)
	writers = append(writers, newRollingFile())
	mw := io.MultiWriter(writers...)
	if config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return zerolog.New(mw).With().
		Str("app", config.AppName).
		Int("pid", os.Getpid()).
		Timestamp().Logger()
}

func newRollingFile() io.Writer {
	if err := os.MkdirAll(path.Dir(config.General.InfoLog), 0744); err != nil {
		log.Error().Err(err).Str("path", path.Dir(config.General.InfoLog)).Msg("can't create log directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename: config.General.InfoLog,
		MaxAge:   30, // days
	}
}

func UpdateContext(update func(c zerolog.Context) zerolog.Context) {
	Logger.UpdateContext(update)
}
func Trace() *zerolog.Event {
	return Logger.Trace()
}

func Debug() *zerolog.Event {
	return Logger.Debug()
}

func Info() *zerolog.Event {
	return Logger.Info()
}

func Warn() *zerolog.Event {
	return Logger.Warn()
}

func Error() *zerolog.Event {
	return Logger.Error()
}

func Err(err error) *zerolog.Event {
	return Logger.Err(err)
}

func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

func Panic() *zerolog.Event {
	return Logger.Panic()
}

func WithLevel(level zerolog.Level) *zerolog.Event {
	return Logger.WithLevel(level)
}

func Log() *zerolog.Event {
	return Logger.Log()
}

func Print(v ...interface{}) {
	Logger.Print(v...)
}

func Printf(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}
