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
        adjustment: 2,
        logger: log.NewLogger("RTT"),
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
    dur := (rtt.End.Sub(rtt.Start)).Microseconds()
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
            self.rttChan <- self.measurements_succ * self.adjustment
    }
    }
}

func (self *RTTManager) GetMeasurement() time.Duration{
    self.signalChan <- true
    val := <- self.rttChan
    return time.Duration(val)
}

func (self *RTTManager)PlaceMeasurement(point *RTT){
    self.measurementChan <- point
}
