//go:build darwin || linux || windows

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
	"time"
)

var (
	Logger           = zerolog.Logger{}
	globalFormatTime = "2006/01/02 15:04:05.000"
)

func SetGlobalLogger(log zerolog.Logger) {
	Logger = log
}

func New() zerolog.Logger {
	zerolog.TimeFieldFormat = globalFormatTime
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(time.Local)
	}

	consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr })
	consoleWriter.FormatLevel = func(i interface{}) string { return strings.ToUpper(fmt.Sprintf("%-6s", i)) }
	consoleWriter.FormatTimestamp = func(i interface{}) string {
		t, err := time.Parse(globalFormatTime, i.(string))
		if err != nil {
			return i.(string)
		}
		return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
	}

	consoleWriterLeveled := zerolog.MultiLevelWriter(consoleWriter)

	fileWriterInfo := &FilteredWriter{zerolog.MultiLevelWriter(newRollingFile(config.General.InfoLog)), zerolog.InfoLevel}
	fileWriterError := &FilteredWriter{zerolog.MultiLevelWriter(newRollingFile(config.General.ErrorLog)), zerolog.ErrorLevel}

	mw := zerolog.MultiLevelWriter(consoleWriterLeveled, fileWriterInfo, fileWriterError)
	if config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel | zerolog.ErrorLevel)
	}
	return zerolog.New(mw).With().
		Str("app", config.AppName).
		Int("pid", os.Getpid()).
		Timestamp().Logger()
}

func newRollingFile(logPath string) io.Writer {
	if err := os.MkdirAll(path.Dir(logPath), os.ModeDir); err != nil {
		log.Error().Err(err).Str("path", path.Dir(logPath)).Msg("can't create log directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename: logPath,
		MaxAge:   30, // days
	}
}

type FilteredWriter struct {
	w     zerolog.LevelWriter
	level zerolog.Level
}

func (w *FilteredWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}
func (w *FilteredWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level == w.level {
		return w.w.WriteLevel(level, p)
	}
	return len(p), nil
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

func Info(duration time.Duration) *zerolog.Event {
	return Logger.Info().Dur("duration", duration)
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
