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
    CanExpand()bool
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
        alpha := 0.3
        beta := 0.6
        minRTT := float64(minRTTi)
        currentRTT := float64(currentRTTi)
        lower_limit := minRTT + minRTT * alpha
        upper_limit := minRTT + minRTT * beta
        self.logger.WithFields(logrus.Fields{
            "minRTT": minRTT,
            "currentRTT": currentRTT,
            "upper_limit": upper_limit,
            "lower_limit": lower_limit,
            "bucketSize": self.bucketSize,
            "increase": float64(currentRTT)/float64(minRTT),
        }).Trace("Adjusting bucketSize")
        if currentRTT < lower_limit {
            self.bucketSize++
            self.logger.WithFields(logrus.Fields{
                "minRTT": minRTT,
                "currentRTT": currentRTT,
                "upper_limit": upper_limit,
                "lower_limit": lower_limit,
                "bucketSize": self.bucketSize,
            }).Trace("Increasing bucketSize")
        }
        if currentRTT > upper_limit {
            if self.bucketSize > 1 {
                self.bucketSize--
                self.logger.WithFields(logrus.Fields{
                    "minRTT": minRTT,
                    "currentRTT": currentRTT,
                    "upper_limit": upper_limit,
                    "lower_limit": lower_limit,
                    "bucketSize": self.bucketSize,
                }).Trace("Decreasing bucketSize")
            }
        }
        return self.bucketSize
    }
}

func (self *CCVegas)CanExpand()bool{
    minRTTi := self.rtts.GetMinRTT().Nanoseconds()
    currentRTTi := self.rtts.GetMeasurement().Nanoseconds()
    alpha := 0.3
    minRTT := float64(minRTTi)
    currentRTT := float64(currentRTTi)
    lower_limit := minRTT + minRTT * alpha
    self.logger.WithFields(logrus.Fields{
        "minRTT": minRTT,
        "currentRTT": currentRTT,
        "lower_limit": lower_limit,
    }).Trace("Check if buckets can be expanded")
    return currentRTT < lower_limit
}
