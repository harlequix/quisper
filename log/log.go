package log

import
(
    log "github.com/sirupsen/logrus"
    _ "io"
    "io/ioutil"
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
		DisableTimestamp: true,
	})
    // AddTracer(base)
    base.SetOutput(os.Stdout)
    base.SetOutput(ioutil.Discard)
    base.SetLevel(log.TraceLevel)
    AddTracer(base, module)
    baselogger := base.WithFields(
        log.Fields{
            "name": module,
        })

    logger := &Logger{baselogger}
    return logger
}
