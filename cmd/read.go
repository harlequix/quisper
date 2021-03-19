package main

 import (
     "github.com/harlequix/quisper/quisper"
     "os"
     "context"
 )

 func main() {
     args := os.Args[1:]
     read(args)
 }

func read(args []string) {
    Reader := quisper.NewReader(args[0], args[1])
    waitFor := context.Background()
    go Reader.MainLoop(waitFor, nil)
    for {

    }
}
