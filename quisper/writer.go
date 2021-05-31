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
    "github.com/harlequix/quisper/internal/format"
    "github.com/harlequix/quisper/internal/decoding"
    prot "github.com/harlequix/quisper/protocol"
    quic "github.com/lucas-clemente/quic-go"
    "math/rand"
    "github.com/spf13/viper"
)

type Backend interface {
    Dial (cid []byte) (quic.Session,error)
}


var logger *log.Logger
const RoleTX string = "tx"
const RoleRX string = "rx"
const TXOffset uint64 = 2
const RXOffset uint64 = 0
const RXReadyOffset uint64 = TXOffset + 1

func init() {
    logger = log.NewLogger("Manager")
}

type DialResult struct {
    CID *prot.CID
    Result int
    Session quic.Session
}

type QuisperConfig struct {
    TimeslotLength time.Duration
    Backend string
    Logfile string
    Role string
}

func init() {
    viper.SetDefault("TimeslotLength", 3)
    viper.SetDefault("Backend", "native")
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
    offset uint64
    dispatchChan chan(*prot.CID)
    resultChan chan(*DialResult)
    timeslotChan chan *timeslots.Timeslot
    ioChan chan byte
    config QuisperConfig
}

func newInstance(addr string, secret string, config QuisperConfig) *Writer{
    backend := backends.NewNativeBackend(addr, nil) // TODO select backend
    timeslot := timeslots.NewTimeslotScheduler(config.TimeslotLength*time.Second) // TODO make duration configurable
    logger.Trace("Create new backend ", addr)
    if config.Role != RoleTX || config.Role != RoleRX {
        panic("please configure a proper role")
    }
    return &Writer{
        addr: addr,
        secret: secret,
        backend: backend,
        TimeslotScheduler: timeslot,
        timeslot: nil,
        cid_length: 16,
        logger: log.NewLogger(config.Role + "-" + secret),
        role: RoleTX,
        offset: TXOffset,
        dispatchChan: make(chan *prot.CID, 10),
        resultChan: make(chan *DialResult, 10),
        timeslotChan: make(chan *timeslots.Timeslot),
        ioChan: make(chan byte, 64),
        config: config,
    }
}

func NewWriter(addr string, secret string) *Writer {
    var config QuisperConfig
    err := viper.Unmarshal(&config)
    if err != nil {
        fmt.Println(err)
    }
    config.Role = RoleTX
    return newInstance(addr, secret, config)
}


func NewReader(addr string, secret string) *Writer {
    var config QuisperConfig
    err := viper.Unmarshal(&config)
    if err != nil {
        fmt.Println(err)
    }
    config.Role = RoleRX
    return newInstance(addr, secret, config)
}

func (self *Writer)Write(p []byte) (int, error){
    cnt := 0
    for _, b := range p {
        self.ioChan <- b
        cnt++
    }
    return cnt, nil
}

func (self *Writer)Read(p []byte) (int, error){
    cnt := 0
    for i := range p {
        p[i] = <- self.ioChan
        cnt++
    }
    return cnt, nil
}

func (self *Writer)Connect() (context.Context, error) {
    ctx := context.Background()
    go self.MainLoop(ctx, self.ioChan)
    return ctx, nil

}

func (self *Writer)runDispatcher(ctx context.Context)  {
    cap := 150
    bucketQueue := make(chan bool, cap)
    for it := 0; it < cap; it++ {
        bucketQueue <- true
    }
    var overflow []*prot.CID
    _ = overflow // fuck you go
    for {
        select {
            case <- ctx.Done():
                    return
            case entry := <- self.dispatchChan:
                go self.dispatchWrapper(entry, self.resultChan, bucketQueue)
        }
    }
}

func (self *Writer) dispatchWrapper(cid *prot.CID, feedback chan *DialResult, tokenBucket chan bool){
    <- tokenBucket
    self.dispatch(cid, feedback)
    tokenBucket <- true
}

func (self *Writer)runEncoder(ctx context.Context)  {
    log := self.logger.WithField("func", "encoder")
    _ = log
    decoder := decoding.NewDecoder(self.ioChan)
    for {
        select {
            case <- ctx.Done():
                return
            case result := <- self.resultChan:
                self.logger.WithField("cid", result.CID).WithField("result", result.Result).Trace("received message")
                var value byte
                if result.Result == 0 {
                    value = encoding.ZERO
                } else {
                    value = encoding.ONE
                }
                decoder.SetCID(result.CID, value)
            case NewSlot := <- self.timeslotChan:
                // capB := NewSlot.GetHeaderValue("bitsSent")
                cap := NewSlot.Cnt
                decoder.AddTimeslot(NewSlot, cap)

        }
    }
}

func (self *Writer)  MainLoop(ctx context.Context, pipeline chan(byte)){
    logger := self.logger.WithField("component", "MainLoop")
    logger.Trace("Mainloop started")
    timeslotChn := make(chan uint64)
    timeslotStatusChn := make(chan bool)
    reportChn := make(chan uint64, 1)
    logger.Trace("about to place something into the reportChn")
    reportChn <- uint64(0)
    logger.Trace("placed into reportChn")
    var sync bool = false
    _ = sync
    self.TimeslotScheduler.Logger = &log.Logger{self.logger.WithField("component", "scheduler")}
    go self.TimeslotScheduler.RunScheduler(ctx, timeslotChn)
    go self.runDispatcher(ctx)
    go self.runEncoder(ctx)
    controlWork, cancel := context.WithCancel(ctx)
    for {
        select {
            case timeslotNum := <- timeslotChn:
                logger.WithField("New", timeslotNum + self.offset).Trace("Setup new Timeslot")
                var oldTimeslot uint64 = 0
                if self.timeslot != nil {
                    oldTimeslot = self.timeslot.Num
                }
                cancel()
                controlWork, cancel = context.WithCancel(ctx)
                _ = controlWork

                bitsSent := <- reportChn
                reportChn = make(chan uint64, 1)
                if self.timeslot != nil {
                    if self.role == RoleTX {
                        self.writeSentBits(self.timeslot, bitsSent)
                    }
                    sync = self.timeslot.Status
                    // if bitsSent != self.timeslot.Cnt {
                    //     // fmt.Println("RACERACERACE")
                    // }
                }



                self.timeslot = timeslots.NewTimeslot(timeslotNum + self.offset)
                // self.initTimeslot()
                self.signalReadiness(self.timeslot)
                timeslotStatusChn = make(chan bool)
                go self.checkReadiness(self.timeslot, timeslotStatusChn)
                logger.WithFields(logrus.Fields{
                    "From": oldTimeslot,
                    "To": self.timeslot.Num,
                    "BitSent": bitsSent,
                    }).Trace("Switch to new timeslot")
                logger.WithField("Timeslot", self.timeslot.Num).WithField("Status", self.timeslot.Status).Trace("Timeslot ready?")
                if sync == true {
                    logger.Trace("Start Working")
                    go self.work(controlWork, self.timeslot, pipeline, reportChn)
                } else {
                    reportChn <- 0
                }
                // if sync == false && self.timeslot.Status == true {
                //     fmt.Println("Clients synchronized")
                //     logger.Trace("Client in Sync")
                // }
                // if sync == true && self.timeslot.Status == false {
                //     fmt.Println("Synchronization lost")
                //     logger.Trace("Synchronization lost")
                // }
                // sync = self.timeslot.Status
                // if self.timeslot.Status == true {
                //     go self.work(controlWork, self.timeslot, pipeline)
                // }
            case status := <- timeslotStatusChn:
                if status == false {
                        cancel()
                        //TODO go Back N
                }
                self.timeslot.Status = status
        }
    }
}

func (self *Writer) initTimeslot() {
    self.signalReadiness(self.timeslot)
    // self.timeslot.Status = self.checkReadiness(self.timeslot)
}

func (self *Writer) work(ctx context.Context, timeslot *timeslots.Timeslot, pipeline chan(byte), report chan(uint64)) {
    var sent uint64 = 0
    if self.role == RoleTX {
        for {
            select {
                case <- ctx.Done():
                    self.logger.WithField("timeslot", timeslot.Num).Trace("Cancelling work")
                    report <- sent
                    return
                case Byte := <- pipeline:
                    block := format.NewBlock()
                    block.SetByte(Byte)
                    cids := block.GetCIDs(timeslot, sent)
                    bits := block.GetBits()
                    sent++
                    self.logger.WithField("Byte", Byte).WithField("bits", bits).Trace("Convert")
                    for index := range bits {
                        self.logger.WithField("CID", cids[index].String()).WithField("Bit", bits[index]).Trace("Sending bit")
                        if bits[index] == encoding.ONE {
                            self.dispatch(cids[index], nil)
                        }
                    }
            }
        }
    } else if self.role == RoleRX {
        num:= self.getBitsSent(timeslot)
        // timeslot.Cnt = num
        self.logger.WithField("num", num).Trace("Bits to probe")
        go self.handleTimeslot(num, timeslot)
        report <- 0

    }
}

func (self *Writer) handleTimeslot(num uint64, slot *timeslots.Timeslot){
    slotCopy := timeslots.NewTimeslot(slot.Num)
    slotCopy.Cnt = num
    self.timeslotChan <- slotCopy
    for it := uint64(0); it < num*uint64(prot.BlockLen); it++ {
        self.dispatchChan <- slot.GetCID()
    }
}

func (self *Writer) evalDial(cid *prot.CID, session quic.Session, err error) *DialResult {
    var result int
    if err == nil {
        result = 0
    } else if strings.HasPrefix(err.Error(), "CRYPTO_ERROR") {
        // self.logger.WithField("cid", cid).Trace("Counting a connection success")
        result = 0
    } else {
        result = 1
    }
    return &DialResult{
        CID: cid,
        Result: result,
        Session: session,
    }
}

func (self *Writer)dispatch(cid *prot.CID, feedback chan(*DialResult))  {
    self.logger.WithField("cid", cid.String()).Trace("Start Request")
    session, err := self.backend.Dial(cid.Bytes())
    ret := self.evalDial(cid, session, err)
    self.logger.WithField("cid", cid.String()).WithField("result", ret.Result).Trace("Finished Request")
    if feedback != nil {
        feedback <- ret
    }
}

func (self *Writer) writeSentBits (slot *timeslots.Timeslot, sentBits uint64) {
    //sentBits := slot.Cnt
    if sentBits == 0 {
        return
    }
    bs := encoding.EncodeSentHeader(sentBits)
    bs = bs[:16]
    self.logger.WithField("sendBuf", bs).WithField("Num", sentBits).WithField("Slot", slot.Num).Debug("Bitarray to write")
    hdrCids, _ := slot.GetHeader("BitsSent")
    for i, cid := range hdrCids {
        if bs[i] == encoding.ONE {
            self.dispatch(cid, nil)
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
    var cids []*prot.CID
    if self.role == RoleTX {
        cids, _ = slot.GetHeader("WriterRDY")
    } else {
        readyTimeSlot := timeslots.NewTimeslot(slot.Num + RXReadyOffset)
        cids, _ = readyTimeSlot.GetHeader("ReaderRDY")
    }
    for _, cid := range cids {
        self.dispatch(cid, nil)
    }
    // feedback := make(chan *DialResult, 4)
    // var admin_bits [4]uint64
    // var readyTimeSlot *timeslots.Timeslot
    // if self.role == RoleTX {
    //     admin_bits = [4] uint64{0,1,2,3}
    //     readyTimeSlot = slot
    //
    // } else {
    //     admin_bits = [4] uint64{4,5,6,7}
    //     readyTimeSlot = timeslots.NewTimeslot(slot.Num + RXReadyOffset)
    // }
    // self.logger.WithField("timeslot", readyTimeSlot.Num).Debug("Setting readiness on Timeslot")
    // for _, bit := range admin_bits {
    //     cid := readyTimeSlot.GetGenCID(bit)
    //     go self.dispatch(cid, feedback)
    // }
    // for range admin_bits {
    //     _ = <- feedback // wait for request to finish
    // }
}

func (self *Writer) checkReadiness(slot *timeslots.Timeslot, resp chan(bool)) {
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
        // self.logger.WithField("result", rdy.Result).Trace("Result from check")
        if rdy.Result == 0 {
            // self.logger.Trace("Slot not ready")
            resp <- false
            return
        }
    }
    resp <- true
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
func (self *Writer) TestServer(){
    var timeslotNum uint64 = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
    test_bits := []uint64{0}
    feedback := make(chan *DialResult, len(test_bits))
    timeslot := timeslots.NewTimeslot(timeslotNum)
    for _, bit := range test_bits {
        cid := timeslot.GetGenCID(bit)
        self.dispatch(cid, nil)
    }
    for _, bit := range test_bits {
        cid := timeslot.GetGenCID(bit)
        go self.dispatch(cid, feedback)
    }
    var resultNum int = 0
    for _ = range test_bits {
        result := <- feedback
        resultNum += result.Result
    }
    if resultNum == len(test_bits){
        fmt.Println("Server is quisper ready")
    } else if resultNum == 0 {
        fmt.Println("Server is not quisper ready")
    } else {
        fmt.Printf("The server only succeded in %d out of %d requests", resultNum, len(test_bits))
    }
}

func (self *Writer) TestLongServer() {
    var timeslotNum uint64 = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
    test_bits := []uint64{0}
    feedback := make(chan *DialResult, len(test_bits))
    hold := make(chan *DialResult, len(test_bits))
    timeslot := timeslots.NewTimeslot(timeslotNum)
    for _, bit := range test_bits {
        cid := timeslot.GetGenCID(bit)
        self.dispatch(cid, hold)
    }
    holder := NewConnectionManager()
    for _ = range test_bits {
        result := <- hold
        if result.Session == nil {
            fmt.Println("cannot connect")
        } else {
            holder.Hold(timeslot.Num, result.Session)
        }
    }
    fmt.Println("Dispatched first group, now delaying")
    time.Sleep(10 * time.Second)
    fmt.Println("Continuing")
    var resultNum int = 0
    for _, bit := range test_bits {
        cid := timeslot.GetGenCID(bit)
        go self.dispatch(cid, feedback)
    }
    for _ = range test_bits {
        result := <- feedback
        resultNum += result.Result
    }
    holder.Retire(timeslot.Num)
    fmt.Println("Long test")
    if resultNum == len(test_bits){
        fmt.Println("Server is quisper ready")
    } else if resultNum == 0 {
        fmt.Println("Server is not quisper ready")
    } else {
        fmt.Printf("The server only succeded in %d out of %d requests", resultNum, len(test_bits))
    }
}

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
