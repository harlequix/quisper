package log

import
(
    log "github.com/sirupsen/logrus"
    _ "io"
    "os"
    "fmt"
    _ "time"
)

type Logger struct {
    *log.Entry
}

func NewLogger(module string) *Logger {

    log.SetLevel(log.WarnLevel)

    base := log.New()
    var file, err = os.OpenFile(module+".log", os.O_RDWR|os.O_CREATE, 0666)
    if err != nil {
        fmt.Println("Could Not Open Log File : " + err.Error())
    }

    base.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		DisableTimestamp: false,
	})
    _ = file
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
