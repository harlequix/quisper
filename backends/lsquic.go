package backends

import (
    _ "github.com/harlequix/quisper/log"
    "os/exec"
    "os"
    "fmt"
    _ "path/filepath"
    "io"
    "strings"
    "bufio"
    "regexp"
    "errors"
)

const DefaultPath string = "/home/jack/Projects/ext/echo_client"
// const DefaultTmpPath string ="/tmp"
type LSQuicBackend struct {
    path string
    address string
}

type State string
type Transition struct {
    State State
    Trigger *regexp.Regexp
}
var transitionMachine map[State][]*Transition = make(map[State][]*Transition)

func init () {
    init2success := &Transition{
        State: success,
        Trigger: regexp.MustCompile(".*Got ACK frame, largest acked:.*"),
    }
    init2fail := &Transition{
        State: fail,
        Trigger: regexp.MustCompile(".*sendctl: retx timeout, mode RETX_MODE_HANDSHAKE.*"),
    }
    transitionMachine[initState] = []*Transition{init2success, init2fail}

}

const (
    initState State = "init"
    success State = "success"
    prefail1 State = "prefail1"
    prefail2 State = "prefail2"
    prefail3 State = "prefail3"
    fail State = "fail"
)


type LSQuicParser struct {
    state State
}

func NewLSQuicParser() *LSQuicParser{
    return &LSQuicParser{
        state: initState,
    }
}

func (self * LSQuicParser) Feed(input string) *bool {
    for _, transition := range transitionMachine[self.state] {
        if transition.Trigger.MatchString(input){
            fmt.Printf("Transition %s -> %s", self.state, transition.State)
            self.state = transition.State
            break
        }
    }
    out := new(bool)
    if self.state == success {
        *out = true
        return out
    } else if self.state == fail {
        *out = false
        return out
    } else {
        return nil
    }
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
    pipe, err := cmd.StderrPipe()
    if err != nil {
        panic(err)
    }
    cmd.Start()
    return self.parseOutput(pipe)
}


func (self *LSQuicBackend) parseOutput (pipe io.ReadCloser) error{
    reader := bufio.NewReader(pipe)
    engine := NewLSQuicParser()
    for {
        str, err := reader.ReadString('\n')
        if err != nil {
            break
            // log.Fatal(err)
        }
        fmt.Println(str)
        decision := engine.Feed(str)
        fmt.Println(decision)
        if decision != nil{
            fmt.Println(*decision)
            if *decision == true {
                return nil
            }
            return errors.New("cannot establish connection")
        }
    }
    return errors.New("cannot decide on connection")
}

func cidTocidstr(cid []byte) string {
    var builder strings.Builder
    for _, val := range cid {
        builder.WriteString(fmt.Sprint(int(val)))
        builder.WriteString(" ")
    }
    return builder.String()
}
