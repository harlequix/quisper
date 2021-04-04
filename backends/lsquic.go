package backends

import (
    _ "github.com/harlequix/quisper/log"
    "os/exec"
    "os"
    "fmt"
    _ "path/filepath"
    "io"
    "strings"
)

const DefaultPath string = "/home/jack/Projects/ext/echo_client"
// const DefaultTmpPath string ="/tmp"
type LSQuicBackend struct {
    path string
    address string
}



func NewLSQuicBackend(address string) *LSQuicBackend {
    //TODO check if DefaultPath exists
    return &LSQuicBackend{
        path: DefaultPath,
        address: address,
    }

}

func (self *LSQuicBackend) Dial(cid []byte) error {
    // tmpfn := filepath.Join(self.tmpdir, "quisper")
    cidStr := cidTocidstr(cid)
    cmd := exec.Command(self.path, "-H", self.address, "-l", "sendctl=debug", "-l", "handshake=debug")
    cidEnv := fmt.Sprintf("LSQUIC_CID='%s'", cidStr)
    cmd.Env = append(os.Environ(), cidEnv)
    cmd.Start()
    pipe, _ := cmd.StderrPipe()
    return self.parseOutput(pipe)
}


func (self *LSQuicBackend) parseOutput (pipe io.ReadCloser) error{
    return nil
}

func cidTocidstr(cid []byte) string {
    var builder strings.Builder
    for _, val := range cid {
        builder.WriteString(string(int(val)))
        builder.WriteString(" ")
    }
    return builder.String()
}
