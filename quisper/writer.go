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
    "golang.org/x/crypto/sha3"
    "github.com/sirupsen/logrus"
    "github.com/harlequix/quisper/internal/encoding"
    "github.com/harlequix/quisper/internal/format"
    "github.com/harlequix/quisper/internal/decoding"
    "github.com/harlequix/quisper/probing"
    prot "github.com/harlequix/quisper/protocol"
    quic "github.com/lucas-clemente/quic-go"
    "math/rand"
    "github.com/spf13/viper"
    "github.com/harlequix/quisper/rtt"
)

type Backend interface {
    Dial (cid []byte) (quic.Session,error)
}

var firstTX = true

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
    OptimisticSync bool
    TimeslotOffset uint64
    Blocksize int
    AdaptiveRelease bool
    ConcurrentReads int
    ProbingStrategy string
    CCEnabled bool
    FCEnabled bool
    BlockWindowSize uint64
    Testing bool
    FCWindow string
}

func init() {
    viper.SetDefault("TimeslotLength", "10s")
    viper.SetDefault("Backend", "native")
    viper.SetDefault("OptimisticSync", false)
    viper.SetDefault("TimeslotOffset", 2)
    viper.SetDefault("Blocksize", 1)
    viper.SetDefault("AdaptiveRelease", false)
    viper.SetDefault("ConcurrentReads", 240)
    viper.SetDefault("ProbingStrategy", "single")
    viper.SetDefault("AdaptiveRelease", false)
    viper.SetDefault("CCEnabled", false)
    viper.SetDefault("FCEnabled", true)
    viper.SetDefault("BlockWindowSize", 128)
    viper.SetDefault("Testing", true)
    viper.SetDefault("FCWindow", "cubic")
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
    stratProbing probing.Strategy
    RTTManager rtt.Manager
    Debug DebugInterface
    CCManager CongestionController
    hashTemplate sha3.ShakeHash
    WindowManager WindowManagerInterface
}

func newInstance(addr string, secret string, config QuisperConfig) *Writer{
    backend := backends.NewNativeBackend(addr, nil) // TODO select backend
    timeslot := timeslots.NewTimeslotScheduler(config.TimeslotLength)
    var stratProbing probing.Strategy
    if config.ProbingStrategy == "single" {
        stratProbing = probing.NewSingleProbing()
    } else if config.ProbingStrategy == "double" {
        stratProbing = probing.NewDoubleProbing()
    } else {
        logger.WithField("InvalidValue", config.ProbingStrategy).Error("Do not know strategy for probing. Falling back to single strategy")
        stratProbing = probing.NewSingleProbing()
    }
    logger.Debug("Create new instance ", config)
    if config.Role != RoleTX && config.Role != RoleRX {
        panic("please configure a proper role")
    }
    var offset uint64
    if config.Role == RoleRX {
        offset = RXOffset
    } else {
        offset = config.TimeslotOffset
    }
    return &Writer{
        addr: addr,
        secret: secret,
        backend: backend,
        TimeslotScheduler: timeslot,
        timeslot: nil,
        cid_length: 16,
        logger: log.NewLogger(config.Role + "-" + secret),
        role: config.Role,
        offset: offset,
        dispatchChan: make(chan *prot.CID, 1024),
        resultChan: make(chan *DialResult, 1024),
        timeslotChan: make(chan *timeslots.Timeslot),
        ioChan: make(chan byte, 1024),
        config: config,
        stratProbing: stratProbing,
        Debug: NewDebugger(),
        hashTemplate: sha3.NewCShake256(nil, []byte(secret)),
    }
}

func SetConfig(configFile string){
    if configFile != "" {
        fmt.Println(viper.AllSettings())
        viper.SetConfigFile(configFile)
        // viper.ReadInConfig()
        err := viper.ReadInConfig() // Find and read the config file
        if err != nil { // Handle errors reading the config file
        	panic(fmt.Errorf("Fatal error config file: %s \n", err))
        }
        fmt.Println(viper.AllSettings())
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

func (self *Writer)Connect() (context.CancelFunc, error) {
    parent := context.Background()
    ctx, cancel := context.WithCancel(parent)
    self.RTTManager = rtt.NewRTTManager(ctx)
    self.CCManager = NewVegasCC(1, self.RTTManager)
    if self.config.FCWindow == "static"{
        self.WindowManager = NewStaticWindowManager(self.config.BlockWindowSize)
    } else if self.config.FCWindow == "cubic" {
        self.WindowManager = NewCubicWindowManager(self.config.BlockWindowSize)
    } else {
        self.logger.Fatal("Unknown FC Window manager")
    }

    go self.MainLoop(ctx, self.ioChan)
    return cancel, nil

}

func (self *Writer)addDispatcher(ctx context.Context, cap int)  {
    for it := 0; it < cap; it++ {
        go self.dispatchWorker(ctx, it)
    }

}

func (self *Writer) dispatchWorker(ctx context.Context, num int) {
    logger := self.logger.WithField("workerID", num)
    logger.Trace("Added worker")
    for {
        select {
        case <-ctx.Done():
            return
        case entry := <- self.dispatchChan:
            logger.WithField("CID", entry.String()).Trace("Dispatching CID")
            self.dispatch(entry, []chan*DialResult{self.resultChan})
        }
    }
}

func (self *Writer) dispatchWrapper(cid *prot.CID, feedback []chan *DialResult, tokenBucket chan bool){
    <- tokenBucket
    if self.config.AdaptiveRelease {
        adapChan := make(chan *DialResult, 1)
        adaptiveTimeout := time.NewTimer(self.RTTManager.GetMeasurement())
        feedback = append(feedback, adapChan)
        go self.dispatch(cid, feedback)
        select {
        case <- adapChan:
            self.logger.WithField("cid", cid.String()).Trace("Request finished, releasing token")
            <- tokenBucket
        case <- adaptiveTimeout.C:
            self.logger.WithField("cid", cid.String()).Trace("Request timed out softly, releasing token early")
            <- tokenBucket
        }
    } else {
        self.dispatch(cid, feedback)
        tokenBucket <- true
    }
}

func (self *Writer)runEncoder(ctx context.Context)  {
    log := self.logger.WithField("func", "encoder")
    _ = log
    decoder := decoding.NewDecoder(self.ioChan, self.stratProbing)
    for {
        select {
            case <- ctx.Done():
                self.logger.Info("shutting down encoder")
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
    logger.Warn("Mainloop started")
    timeslotChn := make(chan uint64)
    timeslotStatusChn := make(chan bool)
    startWork := make(chan bool, 5)
    _ = startWork
    reportChn := make(chan uint64, 1)
    logger.Trace("about to place something into the reportChn")
    reportChn <- uint64(0)
    logger.Trace("placed into reportChn")
    var sync bool = false
    _ = sync
    self.TimeslotScheduler.Logger = &log.Logger{self.logger.WithField("component", "scheduler")}
    go self.TimeslotScheduler.RunScheduler(ctx, timeslotChn)
    self.addDispatcher(ctx, self.config.ConcurrentReads)
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
                }
                if self.role == RoleTX && self.config.FCEnabled == true{
                    cwnd := self.WindowManager.GetSendingWindow()
                    if cwnd == bitsSent {
                        newCwnd := self.WindowManager.AdjustWindowSize()
                        self.logger.WithField("Timeslot", self.timeslot.Num).WithField("new window", newCwnd).WithField("old window", cwnd).Debug("expanding flow control window")
                    }

                }


                desync := false
                // desync_thresh := 0
                self.timeslot = timeslots.NewTimeslot(timeslotNum + self.offset)
                if self.role == RoleRX {
                    leftover := len(self.dispatchChan)
                    if leftover  > 0 {
                        self.logger.WithField("Timeslot", self.timeslot.Num).WithField("leftover", leftover).Debug("Requests leftover, consider stopping TX")
                        if self.config.FCEnabled == true {
                            canexpand := self.CCManager.CanExpand()
                            self.logger.WithField("Timeslot", self.timeslot.Num).WithField("canExpand", canexpand).Debug("canexpand")
                            if canexpand {
                                if leftover > self.config.ConcurrentReads{
                                    self.addDispatcher(ctx, self.config.ConcurrentReads)
                                    self.logger.WithField("Timeslot", self.timeslot.Num).WithField("addedWorker", self.config.ConcurrentReads).Trace("Adding Workers")
                                } else {
                                    self.addDispatcher(ctx, leftover)
                                    self.logger.WithField("Timeslot", self.timeslot.Num).WithField("addedWorker", leftover).Trace("Adding Workers")

                                }
                            }
                            if leftover > self.config.ConcurrentReads{
                                desync = true
                                self.logger.WithField("Timeslot", self.timeslot.Num).Warning("Too many cids are unread, desyncing to catch up")
                            }
                        }
                    }
                }
                if desync == false {
                    self.signalReadiness(self.timeslot)
                } else {
                    self.logger.WithField("Timeslot", self.timeslot.Num).Debug("Skipping synchronization")
                    // self.logger.WithField("Timeslot", self.timeslot.Num).Debug("expanding workers to catch up")
                    // self.addDispatcher(ctx, self.config.ConcurrentReads)
                }
                timeslotStatusChn = make(chan bool)
                go self.checkReadiness(self.timeslot, timeslotStatusChn)
                logger.WithFields(logrus.Fields{
                    "From": oldTimeslot,
                    "To": self.timeslot.Num,
                    "BitSent": bitsSent,
                    }).Trace("Switch to new timeslot")
                logger.WithField("Timeslot", self.timeslot.Num).WithField("Status", self.timeslot.Status).Trace("Timeslot ready?")
                startWork = make(chan bool, 5)
                go func(){
                    logger.Trace("Wait for RDY")
                    select {
                    case <- startWork:
                        logger.Trace("Start Working")
                        self.work(controlWork, self.timeslot, pipeline, reportChn)
                    case <- controlWork.Done():
                        reportChn <- 0
                    }
                }()
                // if sync == true {
                //     startWork <- true
                // }

            case status := <- timeslotStatusChn:
                self.WindowManager.PlaceSyncStatus(self.timeslot.Num, status)
                if status == false {
                        if sync == true {
                            logger.Trace("Desync detected, stop worker")
                        }
                        cancel()
                        //TODO go Back N
                } else {
                    logger.Trace("Starting work")
                    startWork <- true
                }
                self.timeslot.Status = status
            case <- ctx.Done():
                self.logger.Info("Shutting down manager")
                return
        }
    }
}


func (self *Writer) work(ctx context.Context, timeslot *timeslots.Timeslot, pipeline chan(byte), report chan(uint64)) {
    var sent uint64 = 0
    maxBlocks := self.WindowManager.GetSendingWindow()
    self.logger.WithField("timeslot",timeslot.Num).WithField("maxBlocks", maxBlocks).Trace("current window size")
    var ccqueue chan(*DialResult)
    if self.role == RoleTX {
        for (maxBlocks > 0){
            select {
                case <- ctx.Done():
                    self.logger.WithField("timeslot", timeslot.Num).Trace("Cancelling work")
                    report <- sent
                    return
                case Byte := <- pipeline: //TODO change pipeline type to block
                    if firstTX {
                        firstTX = false
                        self.Debug.Emit("START_TRANSMISSION", "Start to transmit blocks")
                    }
                    if ccqueue == nil {
                        ccqueue = self.CCManager.GetWindowBucket(timeslot.Num)
                    }
                    block := format.NewBlock()
                    block.SetByte(Byte)
                    pattern := self.stratProbing.GenPattern(block)
                    cids := pattern.GetSenderCids (timeslot, sent)
                    // bits := block.GetBits()
                    sent++
                    maxBlocks--
                    self.logger.WithField("timeslot", timeslot.Num).WithField("SentBlocks", sent).Trace("Sending Block")
                    for index, val := range cids {
                        self.logger.WithField("CID", cids[index].String()).Trace("Sending bit")
                        if self.config.CCEnabled {
                            self.dispatchControlled(val, ccqueue)
                        } else {
                            self.dispatch(val, nil)
                        }
                    }
                    select {
                        case <- ctx.Done():
                            self.logger.WithField("timeslot", timeslot.Num).Trace("Cancelling work")
                            report <- sent
                            return
                        default:
                            // nothing to do
                    }
            }
        }
    } else if self.role == RoleRX {
        num:= self.getBitsSent(timeslot)
        // timeslot.Cnt = num
        self.logger.WithField("num", num).Trace("Bits to probe")
        go self.handleTimeslot(num, timeslot)
    }
    self.logger.WithField("timeslot", timeslot.Num)
    report <- sent
}

func (self *Writer) handleTimeslot(num uint64, slot *timeslots.Timeslot){
    if num > 0 {
        self.Debug.Emit("RECEIVE", "receive bytes")
    }
    slotCopy := timeslots.NewTimeslot(slot.Num)
    slotCopy.Cnt = num
    self.timeslotChan <- slotCopy
    for it := uint64(0); it < num*uint64(self.stratProbing.PatternLen()); it++ {
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

func (self *Writer)dispatch(logcid *prot.CID, feedback []chan(*DialResult))  {
    var cid *prot.CID
    if self.config.Testing == false {
        cid = self.hashCID(logcid)
    } else {
        cid = logcid
    }
    self.logger.WithField("cid", logcid.String()).WithField("actualcid", cid.String()).Trace("Start Request")
    self.RTTManager.PlaceMeasurement(rtt.MeasureStart(cid))
    session, err := self.backend.Dial(cid.Bytes())
    ret := self.evalDial(logcid, session, err)
    self.RTTManager.PlaceMeasurement(rtt.MeasureEnd(cid, ret.Result))
    self.logger.WithField("cid", logcid.String()).WithField("result", ret.Result).WithField("actualcid", cid.String()).Trace("Finished Request")
    if feedback != nil {
        for _,fchan := range feedback {
            fchan <- ret
        }
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
        go self.dispatch(cid, []chan*DialResult{feedback})
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
        readyTimeSlot := timeslots.NewTimeslot(slot.Num + self.config.TimeslotOffset + 1)
        cids, _ = readyTimeSlot.GetHeader("ReaderRDY")
    }
    for _, cid := range cids {
        go self.dispatch(cid, nil)
    }

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
        go self.dispatch(cid, []chan*DialResult{feedback})
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
        go self.dispatch(cid, []chan*DialResult{feedback})
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
        self.dispatch(cid, []chan*DialResult{hold})
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
        go self.dispatch(cid, []chan*DialResult{feedback})
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
