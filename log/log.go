package log

import (
	"log"
	"os"
)

var (
	// Info logs a message at level Info.
	Info *log.Logger
	// Warning logs a message at level Warning.
	Warning *log.Logger
	// Error logs a message at level Error.
	Error *log.Logger
)

func Init() {
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
