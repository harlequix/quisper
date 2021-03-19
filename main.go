package main

import (
	"flag"
	"fmt"
    "time"
	"github.com/harlequix/quisper/version"
    "github.com/harlequix/quisper/backends"
)

func main() {


	versionFlag := flag.Bool("version", false, "Version")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Build Date:", version.BuildDate)
        fmt.Println("Git Commit:", version.GitCommit)
        fmt.Println("Version:", version.Version)
        fmt.Println("Go Version:", version.GoVersion)
        fmt.Println("OS / Arch:", version.OsArch)
		return
	}
	fmt.Println("Hello.")
    client := backends.NewNativeBackend("127.0.0.1:4433", nil)
    cid := make([]byte, 8)
    cid[0]=2
    fmt.Println(cid)
    try1 := make(chan error)
    try2 := make(chan error)
    go client.Dial(cid)
    time.Sleep(time.Second * 2)
    go client.Dial(cid)
    res1 := <- try1
    res2 := <- try2
    fmt.Println(res1)
    fmt.Println(res2)
}
