package log

import
(
    log "github.com/sirupsen/logrus"
    _ "io"
    // "io/ioutil"
    "os"
    _ "time"
    // "fmt"
)

type Logger struct {
    *log.Entry
}

func NewLogger(module string) *Logger {

    log.SetLevel(log.TraceLevel)

    base := log.New()

    base.SetFormatter(&log.JSONFormatter{
        TimestampFormat: "Jan _2 2006 15:04:05.000000",
        // DisableTimestamp: true,
    })
    // AddTracer(base)
    // output := os.Stdout
    // logfile := os.Getenv("LOG")
    // if logfile != "" {
    //     file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    //     if err == nil {
    //         output = file
    //     }
    // }
    // base.SetOutput(output)
    base.SetLevel(log.TraceLevel)
    // AddTracer(base, module)
    baselogger := base.WithFields(
        log.Fields{
            "name": module,
        })

    logger := &Logger{baselogger}
    return logger
}
func NewLoggerWithLogfile(module string, filePath string) *Logger{
    base := log.New()
    base.SetFormatter(&log.JSONFormatter{
        TimestampFormat: "Jan _2 2006 15:04:05.000000",
        // DisableTimestamp: true,
    })
    // AddTracer(base)
    output := os.Stdout

    if filePath != "" {
        file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err == nil {
            output = file
        }
    }
    base.SetOutput(output)
    base.SetLevel(log.TraceLevel)
    // AddTracer(base, module)
    baselogger := base.WithFields(
        log.Fields{
            "name": module,
        })

    logger := &Logger{baselogger}
    return logger
}
