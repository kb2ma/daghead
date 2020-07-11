/*
Adds log level as a simple wrapper around stand library log package.
Default log level is INFO.

Usage:
   log.ActiveLevel = DEBUG
   ...
   log.Printf(log.DEBUG, "Flyrod askew")
*/
package log

import (
	"log"
)

// Provides a common type for log level constants
type Level int

// Log levels
const (
	DEBUG Level = 10
	INFO Level = 20
	WARN Level = 30
	ERROR Level = 40
)

// Active log level
var ActiveLevel Level = INFO

// Always prints so level not specified
func Panic(v ...interface{}) {
	log.Panic(v)
}

func Printf(level Level, format string, v ...interface{}) {
	if level >= ActiveLevel {
		log.Printf(format, v)
	}
}

func Println(level Level, v ...interface{}) {
	if level >= ActiveLevel {
		log.Println(v)
	}
}
