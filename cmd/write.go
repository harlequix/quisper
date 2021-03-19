package main

 import (
     "github.com/harlequix/quisper/quisper"
     "github.com/harlequix/quisper/internal/encoding"
     "bufio"
     "os"
     "context"
 )

 func main() {
     args := os.Args[1:]
     write(args)
 }

func write(args []string) {
    Writer := quisper.NewWriter(args[0], args[1])
    pipeline := make(chan byte, 64)
    waitFor := context.Background()
    go Writer.MainLoop(waitFor, pipeline)
    reader := bufio.NewReader(os.Stdin)
    for {
        text, _ := reader.ReadString('\n')
        messagebits := []byte(encoding.ToBinaryBytes(text))
        for i := range messagebits {
            pipeline <- messagebits[i]
        }
    }
}
