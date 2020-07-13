/*
Adds log level as a simple wrapper around standard library log package.
Default log level is INFO.

Usage:
   (optional) Set minimum log level for output
   log.SetLevel(log.DEBUG)

   Write a log message at the specified level
   log.Printf(log.WARN, "Flyrod askew")
*/
package log

import (
	"fmt"
	"log"
)

// Provides a common type for log level constants
type Level int

// Log levels
const (
	DEBUG Level = 10
	INFO  Level = 20
	WARN  Level = 30
	ERROR Level = 40
)

// Minimum log level for output
var minLevel Level = INFO

// Sets minimum log level to output
func SetLevel(level Level) {
	minLevel = level
}

// Always prints, so level not specified
func Fatal(v ...interface{}) {
	log.Fatal(v)
}

// Always prints, so level not specified
func Panic(v ...interface{}) {
	log.Panic(v)
}

func Printf(level Level, format string, v ...interface{}) {
	if level >= minLevel {
		log.Printf("%s %s", levelStr(level), fmt.Sprintf(format, v...))
	}
}

func Println(level Level, v ...interface{}) {
	if level >= minLevel {
		log.Printf("%s %s", levelStr(level), fmt.Sprintln(v...))
	}
}

func levelStr(level Level) (string) {
	switch level {
	case ERROR:
		return "ERR"
	case WARN:
		return "WRN"
	case DEBUG:
		return "DBG"
	default:
		return "INF"
	}
}
