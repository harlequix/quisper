package rtt

import (
    "context"
    prot "github.com/harlequix/quisper/protocol"
    "time"
    log "github.com/harlequix/quisper/log"
)

type RTTManager struct {
    measurements_succ int64
    measurements_fail int64
    rtt_success int64
    rtt_fail int64
    signalChan chan bool
    rttChan chan int64
    measurementChan chan *RTT
    measMap map[*prot.CID] *RTT
    adjustment int64
    logger *log.Logger
    signalMin chan bool
    giveMin chan int64
    minRTT int64

}

func NewRTTManager(con context.Context)*RTTManager{
    manager := &RTTManager{
        measurements_fail: 0,
        measurements_succ: 0,
        rtt_success: 0,
        rtt_fail: 0,
        signalChan: make(chan bool, 0),
        rttChan: make(chan int64, 1),
        measurementChan: make(chan *RTT, 128),
        measMap: make(map[*prot.CID]*RTT),
        adjustment: 200,
        logger: log.NewLogger("RTT"),
        signalMin: make(chan bool, 0),
        giveMin: make(chan int64, 1),
        minRTT: 1<<63 - 1,
    }
    go manager.Start(con)
    return manager
}

func movingAverage(old int64, new int64) int64 {
    alpha := 0.1
    var updated float64
    if old == 0 {
        updated = float64(new)
    } else {
        updated = (1.0 - alpha)*float64(old) + alpha * float64(new)
    }
    return int64(updated)
}

func (self *RTTManager)average(rtt *RTT){
    dur := (rtt.End.Sub(rtt.Start)).Nanoseconds()
    if dur < self.minRTT {
        self.minRTT = dur
    }
    if rtt.Result == 0 {
        saved := self.measurements_succ
        self.measurements_succ = movingAverage(self.measurements_succ, dur)
        self.logger.WithField("old", saved).WithField("new", self.measurements_succ).WithField("update", dur).Trace("update success RTT")
    } else {
        saved := self.measurements_fail
        self.measurements_fail = movingAverage(self.measurements_fail, dur)
        self.logger.WithField("old", saved).WithField("new", self.measurements_fail).WithField("update", dur).Trace("update fail RTT")
    }
}

func (self *RTTManager)Start(ctx context.Context){
    for {
        select {
        case <- ctx.Done():
            return
        case measurement := <- self.measurementChan:
            point, ok := self.measMap[measurement.Cid];if ok {
                point.Update(measurement)
                if !point.Start.IsZero() && !point.End.IsZero() {
                    self.average(point)
                }
            } else {
                self.measMap[measurement.Cid] = measurement
            }
        case <- self.signalChan:
            self.logger.Trace("dispatching new measurement")
            self.rttChan <- self.measurements_succ
        case <- self.signalMin:
            self.logger.Trace("Dispatch minimum")
            self.giveMin <- self.minRTT
    }
    }
}

func (self *RTTManager) GetMeasurement() time.Duration{
    self.logger.Trace("Requesting new measurement")
    self.signalChan <- true
    self.logger.Trace("Waiting for measurement")
    val := <- self.rttChan
    ret := time.Duration(val) * time.Nanosecond
    return ret
}

func (self *RTTManager)PlaceMeasurement(point *RTT){
    self.measurementChan <- point
}

func (self *RTTManager)GetMinRTT() time.Duration {
    self.signalMin <- true
    val := <- self.giveMin
    return time.Duration(val) * time.Nanosecond
}
