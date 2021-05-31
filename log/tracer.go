package log

import (
    log "github.com/sirupsen/logrus"
    "fmt"
    "github.com/rifflock/lfshook"
)

type Tracer struct {
}

func AddTracer (logger *log.Logger, path string){
    pathMap := lfshook.PathMap{
		log.TraceLevel: path + ".trace",
        log.WarnLevel: path + ".warn",
	}
    hook := lfshook.NewHook(
        pathMap,
        &log.JSONFormatter{
            TimestampFormat: "Jan _2 2006 15:04:05.000000",
            // DisableTimestamp: true,
        },
    )
    logger.Hooks.Add(hook)
}

func (tr *Tracer) Levels() []log.Level {
    return []log.Level{log.TraceLevel}
}

func (tr *Tracer) Fire(event *log.Entry) error {
    fmt.Println(event.Message)
    return nil
}
