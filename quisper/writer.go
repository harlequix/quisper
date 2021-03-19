package quisper

import
(
    log "github.com/harlequix/quisper/log"
    "github.com/harlequix/quisper/backends"
    "github.com/harlequix/quisper/timeslots"
    _ "strings"
    "fmt"
    _ "bytes"
    "strings"
    "time"
    "context"
    "github.com/sirupsen/logrus"
    "github.com/harlequix/quisper/internal/encoding"
)

type Backend interface {
    Dial (cid []byte) error
}


var logger *log.Logger
const RoleTX string = "tx"
const RoleRX string = "rx"
const TXOffset int64 = 2
const RXOffset int64 = 0
const RXReadyOffset int64 = TXOffset + 1

func init() {
    logger = log.NewLogger("Manager")
}

type DialResult struct {
    CID []byte
    Result int
}

type Writer struct {
    addr string
    secret string
    backend Backend
    timeslot *timeslots.Timeslot
    TimeslotScheduler *timeslots.TimeslotScheduler
    cid_length int
    logger *log.Logger
    role string
    offset int64
    dispatchChan chan([]byte)
    resultChan chan(*DialResult)
    timeslotChan chan *timeslots.Timeslot
}

func NewWriter(addr string, secret string) *Writer {
    backend := backends.NewNativeBackend(addr, nil) // TODO select backend
    timeslot := timeslots.NewTimeslotScheduler(10*time.Second) // TODO make duration configurable
    logger.Info("Create new backend ", addr)
    return &Writer{
        addr: addr,
        secret: secret,
        backend: backend,
        TimeslotScheduler: timeslot,
        timeslot: nil,
        cid_length: 16,
        logger: log.NewLogger(RoleTX + "-" + secret),
        role: RoleTX,
        offset: TXOffset,
        dispatchChan: make(chan []byte, 10),
        resultChan: make(chan *DialResult, 10),
        timeslotChan: make(chan *timeslots.Timeslot),
    }
}

func NewReader(addr string, secret string) *Writer {
    backend := backends.NewNativeBackend(addr, nil) // TODO select backend
    timeslot := timeslots.NewTimeslotScheduler(10*time.Second) // TODO make duration configurable
    logger.Info("Create new backend ", addr)
    return &Writer{
        addr: addr,
        secret: secret,
        backend: backend,
        TimeslotScheduler: timeslot,
        timeslot: nil,
        cid_length: 16,
        logger: log.NewLogger(RoleRX + "-" + secret),
        role: RoleRX,
        offset: RXOffset,
        dispatchChan: make(chan []byte, 10),
        resultChan: make(chan *DialResult, 10),
        timeslotChan: make(chan *timeslots.Timeslot),
    }
}

func (self *Writer)runDispatcher(ctx context.Context)  {
    cap := 100
    bucketQueue := make(chan bool, cap)
    for it := 0; it < cap; it++ {
        bucketQueue <- true
    }
    var overflow [][]byte
    _ = overflow // fuck you go
    for {
        select {
            case <- ctx.Done():
                    return
            case entry := <- self.dispatchChan:
                overflow = append(overflow, entry)
            case _ = <- bucketQueue:
                if len(overflow) > 0 {
                    var cid []byte
                    cid, overflow = overflow[0], overflow[1:]
                    go self.dispatchWrapper(cid, self.resultChan, bucketQueue)
                } else {
                    bucketQueue <- true // return bucket
                }
        }
    }
}

func (self *Writer) dispatchWrapper(cid []byte, feedback chan *DialResult, tokenBucket chan bool){
    self.dispatch(cid, feedback)
    tokenBucket <- true
}

func (self *Writer)runEncoder(ctx context.Context)  {
    log := self.logger.WithField("func", "encoder")
    _ = log
    decoder := encoding.NewDecoder()
    for {
        select {
            case <- ctx.Done():
                return
            case result := <- self.resultChan:
                self.logger.WithField("cid", result.CID).WithField("result", result.Result).Debug("received message")
                idSl, suffixSl := timeslots.Cut(result.CID)
                id := timeslots.SuffixToDec(idSl)
                suffix := timeslots.SuffixToDec(suffixSl) - 24
                var value byte
                if result.Result == 0 {
                    value = encoding.ZERO
                } else {
                    value = encoding.ONE
                }
                decoder.SetCID(id, suffix, value)
            case NewSlot := <- self.timeslotChan:
                // capB := NewSlot.GetHeaderValue("bitsSent")
                cap := NewSlot.Cnt
                decoder.AddTimeslot(NewSlot, cap)

        }
    }
}

func (self *Writer)  MainLoop(ctx context.Context, pipeline chan(byte)){
    logger := self.logger.WithField("component", "MainLoop")
    timeslotChn := make(chan int64)
    self.TimeslotScheduler.Logger = &log.Logger{self.logger.WithField("component", "scheduler")}
    go self.TimeslotScheduler.RunScheduler(ctx, timeslotChn)
    go self.runDispatcher(ctx)
    go self.runEncoder(ctx)
    controlWork, cancel := context.WithCancel(ctx)
    for {
        select {
            case timeslotNum := <- timeslotChn:
                var oldTimeslot int64 = 0
                if self.timeslot != nil {
                    oldTimeslot = self.timeslot.Num
                }
                cancel()
                controlWork, cancel = context.WithCancel(ctx)
                var bitsSent uint64 = 0
                if self.timeslot != nil {
                    bitsSent = self.timeslot.Cnt
                    if self.role == RoleTX {
                        self.writeSentBits(self.timeslot)
                    }
                }

                _ = bitsSent
                self.timeslot = timeslots.NewTimeslot(timeslotNum + self.offset)
                self.initTimeslot()
                logger.WithFields(logrus.Fields{
                    "From": oldTimeslot,
                    "To": self.timeslot.Num,
                    "BitSent": bitsSent,
                    }).Info("Switch to new timeslot")
                logger.WithField("Timeslot", self.timeslot.Num).WithField("Status", self.timeslot.Status).Info("Timeslot ready?")
                if self.timeslot.Status == true {
                    go self.work(controlWork, self.timeslot, pipeline)
                }
        }
    }
}

func (self *Writer) initTimeslot() {
    self.signalReadiness(self.timeslot)
    self.timeslot.Status = self.checkReadiness(self.timeslot)
}

func (self *Writer) work(ctx context.Context, timeslot *timeslots.Timeslot, pipeline chan(byte)) {
    if self.role == RoleTX {
        for {
            select {
                case <- ctx.Done():
                    return
                case bit := <- pipeline:
                    cid := timeslot.GetCID()
                    if bit == 49 {
                        err := self.backend.Dial(cid)
                        _ = err
                    }
            }
        }
    } else if self.role == RoleRX {
        num:= self.getBitsSent(timeslot)
        // timeslot.Cnt = num
        go self.handleTimeslot(num, timeslot)

    }
}

func (self *Writer) handleTimeslot(num uint64, slot *timeslots.Timeslot){
    slotCopy := timeslots.NewTimeslot(slot.Num)
    slotCopy.Cnt = num
    self.timeslotChan <- slotCopy
    for it := uint64(0); it < num; it++ {
        self.dispatchChan <- slot.GetCID()
    }
}

func (self *Writer) evalDial(cid []byte, err error) *DialResult {
    var result int
    if err == nil {
        result = 0
    } else if strings.HasPrefix(err.Error(), "CRYPTO_ERROR") {
        self.logger.WithField("cid", cid).Trace("Counting a connection success")
        result = 0
    } else {
        result = 1
    }
    return &DialResult{
        CID: cid,
        Result: result,
    }
}

func (self *Writer)dispatch(cid []byte, feedback chan(*DialResult))  {
    err := self.backend.Dial(cid)
    feedback <- self.evalDial(cid, err)
}

func (self *Writer) writeSentBits (slot *timeslots.Timeslot) {
    sentBits := slot.Cnt
    if sentBits == 0 {
        return
    }
    bs := encoding.EncodeSentHeader(sentBits)
    bs = bs[:16]
    self.logger.WithField("sendBuf", bs).WithField("Num", sentBits).WithField("Slot", slot.Num).Debug("Bitarray to write")
    hdrCids, _ := slot.GetHeader("BitsSent")
    for i, cid := range hdrCids {
        if bs[i] == encoding.ONE {
            self.backend.Dial(cid)
        }
    }
}




func (self *Writer) getBitsSent (slot *timeslots.Timeslot) uint64{
    log := self.logger.WithField("func", "getsBitsSent").WithField("slot", slot.Num)
    hdrCids, _ := slot.GetHeader("BitsSent")
    feedback := make(chan *DialResult, len(hdrCids))
    for _, cid := range hdrCids {
        go self.dispatch(cid, feedback)
    }
    for _ = range hdrCids {
        result := <- feedback
        var value byte
        if result.Result == 0 {
            value = encoding.ZERO
        } else {
            value = encoding.ONE
        }
        slot.SetHeaderValue(result.CID, value)
    }
    _ = log
    headerBits := slot.GetHeaderValue("BitsSent")
    numberReceived := encoding.DecodeSentHeader(headerBits)
    log.WithField("BitsSent", numberReceived).WithField("Bits", headerBits).Debug("Received sentBits")
    return numberReceived

}

func (self *Writer) signalReadiness(slot *timeslots.Timeslot) {
    feedback := make(chan *DialResult, 4)
    var admin_bits [4]uint64
    var readyTimeSlot *timeslots.Timeslot
    if self.role == RoleTX {
        admin_bits = [4] uint64{0,1,2,3}
        readyTimeSlot = slot

    } else {
        admin_bits = [4] uint64{4,5,6,7}
        readyTimeSlot = timeslots.NewTimeslot(slot.Num + RXReadyOffset)
    }
    self.logger.WithField("timeslot", readyTimeSlot.Num).Debug("Setting readiness on Timeslot")
    for _, bit := range admin_bits {
        cid := readyTimeSlot.GetGenCID(bit)
        go self.dispatch(cid, feedback)
    }
    for range admin_bits {
        _ = <- feedback // wait for request to finish
    }
}

func (self *Writer) checkReadiness(slot *timeslots.Timeslot) bool {
    feedback := make(chan *DialResult, 4)
    var admin_bits [4]uint64
    if self.role == RoleTX {
        admin_bits = [4] uint64{4,5,6,7}
    } else {
        admin_bits = [4] uint64{0,1,2,3}
    }
    self.logger.WithField("timeslot", slot.Num).Debug("checking readiness on Timeslot")
    for _, bit := range admin_bits {
        cid := slot.GetGenCID(bit)
        go self.dispatch(cid, feedback)
    }
    for range admin_bits {
        rdy := <- feedback
        self.logger.WithField("result", rdy.Result).Trace("Result from check")
        if rdy.Result == 0 {
            self.logger.Trace("Slot not ready")
            return false
        }
    }
    return true
}

// func (self *Writer) Read(from uint64, to uint64) []byte {
//     //timeslot:= make([]byte, 8)
//     message := make([]byte, to+1)
//     for i := from; i <= to; i++ {
//         cid := make([]byte, self.cid_length)
//         binary.LittleEndian.PutUint64(cid, i)
//         result := self.backend.Dial(cid)
//         if result != nil {
//             if strings.HasPrefix(result.Error(), "CRYPTO_ERROR") {
//                 logger.Trace("Counting a connection success")
//                 message[i] = 0
//             } else {
//                 logger.Trace("Counting a connection failure")
//                 message[i] = 1
//             }
//         }
//     }
//     return message
// }


func main(){
    Writer := NewWriter("192.168.0.2:12345", "secret")
    Reader := NewReader("192.168.0.2:12345", "secret")
    message := "Hello World"
    messagebits_tmp := encoding.ToBinaryBytes(message)
    messagebits := []byte(messagebits_tmp)
    fmt.Println(len(messagebits))
    logger.WithField("message", messagebits).WithField("bitstring", string(messagebits_tmp)).Info("Message to sent")
    pipeline := make(chan byte, 64)
    _ = messagebits
    waitFor := context.Background()
    Reading := context.Background()
    go Writer.MainLoop(waitFor, pipeline)
    go Reader.MainLoop(Reading, nil)
    fmt.Println("################################")
    for i := range messagebits {
        pipeline <- messagebits[i]
    }
    <- waitFor.Done()
}
