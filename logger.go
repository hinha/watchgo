package main

import (
	"log"
	"os"
	"strings"
)

const bufSize = 5000

// LogConfig
type LogConfig struct {
	AppName string
	Debug   bool
	LogFile string

	logChannel chan []byte
}

// New return https://godoc.org/log
func NewLogger(lw *LogConfig) *log.Logger {
	lw.logChannel = make(chan []byte, bufSize)
	go lw.loop()

	prefix := "<" + lw.AppName + "> - "
	return log.New(lw, prefix, log.LstdFlags)
}

// Write implement io/writer
func (lw *LogConfig) Write(p []byte) (int, error) {
	lw.logChannel <- p
	return len(p), nil
}

// loop append file worker
func (lw *LogConfig) loop() {
	logfile, err := os.OpenFile(lw.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err)
		close(lw.logChannel)
		return
	}
	defer logfile.Close()
	builder := new(strings.Builder)

	for {
		data := <-lw.logChannel
		if lw.Debug {
			builder.Write(data)
			textLog := builder.String()
			builder.Reset()
			log.Println(textLog)
		}
		logfile.Write(data)
	}
}
