package logger

import (
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime)
}

// Info emits info severity log message
func Info(msg interface{}) {
	log.Printf("[INFO] %s\n", msg)
}

// Warning emits warning severity log message
func Warning(msg interface{}) {
	log.Printf("[WARNING] %s\n", msg)
}

// Error emits error severity log message
func Error(msg interface{}) {
	log.Printf("[ERROR] %s\n", msg)
}
