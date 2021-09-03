package quisper
import(
    log "github.com/harlequix/quisper/log"
    "context"
)

type DialManager struct {
    overflow []*DialTask
    workersCtrl chan bool
    CCManager CongestionController
    numWorkers int
    maxWorkers int
    logger *log.Logger
    DispatchChan chan *DialResult
    app *Writer
}

func NewDialManager(app *Writer, maxWorkers int) *DialManager {
    out := &DialManager{
        workersCtrl: make(chan bool, maxWorkers),
        CCManager: app.CCManager,
        logger: log.NewLogger("DialManager"),
        maxWorkers: maxWorkers,
        app: app,
        DispatchChan: make(chan *DialResult, 1),
    }
    return out
}

func (self *DialManager)AddWorkers(num int){
    for num > 0{
        if(self.numWorkers < self.maxWorkers){
            go self.app.DispatchWorker(self.workersCtrl, self.numWorkers)
            self.logger.WithField("ID:", self.numWorkers).Trace("added one worker")
            self.numWorkers++
        }else{
            break
        }
    }
}


func (self *DialManager)Start(ctx context.Context){
    for {
    select {
    case <-ctx.Done():
        for i:= 0; i < self.numWorkers; i++{
            self.workersCtrl <- true
        }
    case item := <- self.DispatchChan:
        self.logger.Debug("received result")
        if item.Result == 0 {
            if self.numWorkers < self.maxWorkers{
                newW := self.CCManager.CanAdjust()
                self.logger.WithField("new",newW).Debug("Adjusting workers")
                if newW > 0 && self.app.Backlog() > 1{
                    self.AddWorkers(1)
                    self.logger.WithField("num",self.numWorkers).Debug("Adjusting workers")

                }
                if newW < 0 && self.numWorkers > 4{
                    self.workersCtrl <- true
                    self.numWorkers--
                    self.logger.WithField("num", self.numWorkers).Debug("Adjusting workers")

                }
            }
        }
    }
    }
}
