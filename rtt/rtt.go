package rtt

import (
    "time"
    prot "github.com/harlequix/quisper/protocol"
)
type RTT struct {
    Start time.Time
    End time.Time
    Result int
    Cid *prot.CID
}

func MeasureStart(cid *prot.CID) *RTT{
    return &RTT{
        Start: time.Now(),
        Cid:   cid,
        Result: -1,
    }
}

func MeasureEnd(cid *prot.CID, result int) *RTT{
    return &RTT{
        End: time.Now(),
        Cid:   cid,
        Result: result,
    }
}

func (self *RTT)Update(newRTT *RTT){
    if self.Start.IsZero(){
        self.Start = newRTT.Start
    }
    if self.End.IsZero(){
        self.End = newRTT.End
    }
    if self.Result == -1 {
        self.Result = newRTT.Result
    }
}
