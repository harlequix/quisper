package quisper

import(
    "math"
    log "github.com/sirupsen/logrus"
)

type WindowManagerInterface interface {
    PlaceSyncStatus(uint64, bool)
    GetSendingWindow()uint64
    AdjustWindowSize()uint64
}

type StaticWindowManager struct {
    maxBlocks uint64
}

func NewStaticWindowManager(maxBlocks uint64) *StaticWindowManager{
    return &StaticWindowManager{
        maxBlocks: maxBlocks,
    }
}

func (self *StaticWindowManager)PlaceSyncStatus(timeslotnum uint64, status bool){
    //Do nothing
}

func (self *StaticWindowManager)GetSendingWindow()uint64{
    return self.maxBlocks
}

func (self *StaticWindowManager)AdjustWindowSize()uint64{
    return self.maxBlocks
}

type CubicWindowManager struct {
    maxBlocks uint64
    statusDuration int
    lastStatus bool
    lastTimeslot uint64
    lastDuration int
    lastBase uint64
}

func NewCubicWindowManager(maxBlocks uint64) *CubicWindowManager{
    return &CubicWindowManager{
        maxBlocks: maxBlocks,
        lastBase: maxBlocks,
    }
}

func (self *CubicWindowManager) PlaceSyncStatus(timeslotnum uint64, status bool){
    if timeslotnum < self.lastTimeslot{
        // this should not happen
    }
    self.lastTimeslot = timeslotnum
    if self.lastStatus == status {
        self.statusDuration++
    } else {
        self.lastStatus = status
        self.lastDuration = self.statusDuration
        self.statusDuration = 0
    }
}

func (self *CubicWindowManager)GetSendingWindow()uint64{
    return self.maxBlocks
}

func (self* CubicWindowManager)AdjustWindowSize()uint64{
    c := 0.4
    beta := 0.7
    log.WithField("status", self.lastStatus).WithField("duration", self.statusDuration).Trace("Adjusting Windowsize")
    if self.lastStatus == true {
        if self.statusDuration > 4 {
            log.Trace("increasing window")
            K := math.Cbrt((float64(self.lastBase)*(1.0 - beta))/c)
            cwnd := math.Pow(float64(self.statusDuration - 4) - K, 3)
            cwnd = c * cwnd
            self.maxBlocks = uint64(cwnd)
        }
    if self.lastStatus == false {
        if self.statusDuration == 2 && self.lastDuration > 0 {
            self.lastBase = uint64(float64(self.maxBlocks) * beta)
            self.maxBlocks = self.lastBase
        }
    }
    }
    return self.maxBlocks
}
