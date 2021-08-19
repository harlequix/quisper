package quisper

import
(
    log "github.com/harlequix/quisper/log"
    "github.com/harlequix/quisper/rtt"
    "time"
    "github.com/sirupsen/logrus"
)

type CongestionController interface {
    GetWindowBucket(uint64) chan(*DialResult)
}

type CCVegas struct {
    logger *log.Logger
    rtts rtt.Manager
    bucketSize int
    rrt0Est time.Duration
    lastTimeSlot uint64
}

func NewVegasCC(initialSize int, rtts rtt.Manager) *CCVegas  {
    ccontroller := &CCVegas{
        logger: log.NewLogger("CCVegas"),
        rtts: rtts,
        bucketSize: initialSize,
    }
    return ccontroller
}

func (self *CCVegas)GetWindowBucket(lastTimeSlot uint64) chan(*DialResult)  {
    bucketSize := self.adjust(lastTimeSlot)
    queue := make(chan *DialResult, bucketSize)
    self.logger.WithField("BucketSize", bucketSize).Trace("Issuing new bucket")
    for i := 0; i < bucketSize; i++{
        queue <- &DialResult{}
    }
    return queue
}

func (self *CCVegas) adjust (lastTimeSlot uint64) int{
    last := self.lastTimeSlot
    self.lastTimeSlot = lastTimeSlot
    if last == 0 {
        self.lastTimeSlot = lastTimeSlot
        return self.bucketSize
    }
    if lastTimeSlot - last > 1 {
        return self.bucketSize
    } else {
        minRTTi := self.rtts.GetMinRTT().Nanoseconds()
        currentRTTi := self.rtts.GetMeasurement().Nanoseconds()
        alpha := float64(1)
        beta := float64(3)
        minRTT := float64(minRTTi)
        currentRTT := float64(currentRTTi)
        cwnd := float64(self.bucketSize)
        expected := cwnd/minRTT
        actual := cwnd/currentRTT

        diff := (expected - actual)*currentRTT
        // lower_limit := minRTT + minRTT * alpha
        // upper_limit := minRTT + minRTT * beta
        self.logger.WithFields(logrus.Fields{
            "minRTT": minRTT,
            "currentRTT": currentRTT,
            "expected": expected,
            "actual": actual,
            "diff": diff,
            "bucketSize": self.bucketSize,
        }).Trace("Adjusting bucketSize")
        if diff < alpha {
            self.bucketSize++
            self.logger.WithFields(logrus.Fields{
                "minRTT": minRTT,
                "currentRTT": currentRTT,
                "expected": expected,
                "actual": actual,
                "diff": diff,
                "bucketSize": self.bucketSize,
            }).Trace("Increasing bucketSize")
        }
        if diff > beta {
            if self.bucketSize > 1 {
                self.bucketSize--
                self.logger.WithFields(logrus.Fields{
                    "minRTT": minRTT,
                    "currentRTT": currentRTT,
                    "expected": expected,
                    "actual": actual,
                    "diff": diff,
                    "bucketSize": self.bucketSize,
                }).Trace("Decreasing bucketSize")
            }
        }
        return self.bucketSize
    }
}

// type CCTargetControl struct {
//     logger *log.Logger
//     rtts rtt.Manager
//     bucketSize int
//     rrt0Est time.Duration
//     lastTimeSlot uint64
// }
//
// func NewCCTargetControl(initialSize int, rtts rtt.Manager)*CCTargetControl{
//     ccontroller := &CCTargetControl{
//         logger: log.NewLogger("CCVegas"),
//         rtts: rtts,
//         bucketSize: initialSize,
//     }
//     return ccontroller
// }
//
// func (self *CCTargetControl) CanExpand(timeslot uint64) bool{
//     last := self.lastTimeSlot
//     self.lastTimeSlot = lastTimeSlot
//     if last == 0 {
//         self.lastTimeSlot = lastTimeSlot
//         return self.bucketSize
//     }
//     if lastTimeSlot - last > 1 {
//         return self.bucketSize
//     } else {
//         minRTTi := self.rtts.GetMinRTT()
//         smoothRtt := self.rtts.GetMeasurement()
//         target := self.calculateTarget(minRTTi)
//         self.logger.WithFields(logrus.Fields{
//             "minrtt":minRTTi,
//             "smoothRtt": smoothRtt,
//             "target": target,
//             }).Debug("Calculating new buckets")
//     }
// }
//
// func (self *CCTargetControl) calculateTarget(minrtt time.Duration) time.Duration {
//     stepHigher := time.Duration(200 * time.Milliseconds())
    // while (stepHigher.Nanoseconds() < minrtt.Nanoseconds()){
    //     // stepHigher = stepHigher * 2
    // }
    // return stepHigher - time.Duration(100 * time.Milliseconds())
// }
