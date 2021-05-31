package main

import (

	"github.com/harlequix/quisper/cmd"
	// _ "net/http/pprof"
    // "net/http"
    // "log"
)

func main() {
    // go func() {
	// log.Println(http.ListenAndServe("localhost:6060", nil))
    // }()
    cmd.Execute()
}
