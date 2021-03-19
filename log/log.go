package log

import
(
    log "github.com/sirupsen/logrus"
    _ "io"
    "os"
    _ "time"
)

type Logger struct {
    *log.Entry
}

func NewLogger(module string) *Logger {

    log.SetLevel(log.WarnLevel)

    base := log.New()

    base.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		DisableTimestamp: false,
	})

    base.SetOutput(os.Stdout)
    base.SetLevel(log.ErrorLevel)

    base.Debug("warning")
    baselogger := base.WithFields(
        log.Fields{
            "name": module,
        })
    logger := &Logger{baselogger}
    return logger
}
