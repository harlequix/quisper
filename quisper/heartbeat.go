package quisper

import(
    "time"
    "context"
    "math/rand"
    prot "github.com/harlequix/quisper/protocol"
)

type HeartBeatMonitor interface {
    Start(context.Context, *Writer, float64)
    Aquire() bool
}

const cap int = 50
const interval = 200 * time.Millisecond

type HBMonitor struct {
    interval time.Duration
    thresh float64
    waiting chan int
    waitingQueue int
    feedback chan bool
    app *Writer
}

func NewHeartBeatMonitor() *HBMonitor {
    out := &HBMonitor{
        interval: interval,
        waiting: make(chan int, 0),
        feedback: make(chan bool),
    }
    return out
}

func (self *HBMonitor)Start(con context.Context, app *Writer, thresh float64){
    self.app = app
    self.thresh = thresh
    text, _ := context.WithCancel(con)
    go self.run(text)
}

func (self *HBMonitor) run(con context.Context){
    ticker := time.NewTicker(self.interval)
    status := true
    results := make(chan *DialResult, 10)
    failed := 0
    waitingQueue:= 0
    _ = waitingQueue //fuck you go
    rr:= make([]*DialResult, cap)
    rrIt:= 0
    rand.Seed(time.Now().UnixNano())
    for{
        if self.waitingQueue > 0 && status == true {
            for ; self.waitingQueue > 0; self.waitingQueue-- {
                self.feedback <- true
            }
        }
        select {
        case <- con.Done():
            return
        case <- self.waiting:
            self.waitingQueue++
        case <- ticker.C:
            cidB := make([]byte, 16)
            rand.Read(cidB)
            cid := prot.NewCID(cidB)
            self.app.Dispatch(cid, []chan *DialResult{results})
        case req := <- results:
            place := rrIt
            rrIt = (rrIt + 1) % cap
            if rr[place] != nil && rr[place].Result == 1 {
                failed = failed - 1
            }
            rr[place] = req
            if req.Result == 1 {
                failed++
            }
            if float64(failed)/float64(cap) > self.thresh{
                status = false
            } else {
                status = true
            }
        }
    }
}

func (self *HBMonitor) Aquire() bool {
    self.waiting <- 1
    return <- self.feedback
}
