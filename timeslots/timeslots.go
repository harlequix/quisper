package timeslots
import (
    log "github.com/harlequix/quisper/log"

    "time"
    "context"
    "encoding/binary"
    // "fmt"
)
type Timeslot struct {
    Prefix []byte
    Cnt uint64
    Num int64
    adminOffset uint64
    Status bool
    Header map[string]*Header

}
const prefixLength int = 8
type Header struct {
    Len uint64
    From uint64
    To uint64
    Name string
    Value []byte
}

func NewTimeslot (num int64) *Timeslot {
    prefix := make([]byte, 8)
    binary.LittleEndian.PutUint64(prefix, uint64(num))
    slot := &Timeslot {
        Prefix: prefix,
        Cnt: 0,
        Num: num,
        Status: false,
        adminOffset: 0,
        Header: make(map[string]*Header),
    }
    slot.addHeader("WriterRDY", 4)
    slot.addHeader("ReaderRDY", 4)
    slot.addHeader("BitsSent", 16)
    return slot

}

var logger *log.Logger = log.NewLogger("timeslots")

func (self *Timeslot) addHeader(name string, len uint64){
    header := &Header {
        Name: name,
        Len: len,
        From: self.adminOffset,
        To: self.adminOffset + len,
        Value: make([]byte, len),
    }
    self.Header[name] = header
    self.adminOffset += len
}

func (self *Timeslot) GetHeader(name string) ([][]byte, error) {
    if hdr, ok := self.Header[name]; ok {
        out := make([][]byte, hdr.Len)
        for index := uint64(0); index < hdr.Len; index++ {
                out[index] = self.GetGenCID(hdr.From + index)
        }
        return out, nil
    }
    return nil, nil

}

func (self *Timeslot) SetHeaderValue (cid []byte, val byte){
    _, cid = Cut(cid)
    index := SuffixToDec(cid)
    for _, element := range self.Header {
        if index >= element.From && index < element.To {
            element.Value[index - element.From] = val
        }
    }
}

func (self *Timeslot) GetHeaderValue(name string) []byte {
    if hdr, ok := self.Header[name]; ok {
        return hdr.Value
    }
    return nil
}

func Cut(cid []byte) ([]byte, []byte){
    return cid[:prefixLength], cid[prefixLength:]
}

func SuffixToDec(suffix []byte) uint64 {
    logger.WithField("suffix", suffix).Warn("TODO")
    return binary.LittleEndian.Uint64(suffix)
}

func (self *Timeslot) GetGenCID(numb uint64) []byte{
    suffix := make([]byte, 8)
    binary.LittleEndian.PutUint64(suffix, numb)
    cid := append(self.Prefix, suffix...)
    return cid
}

func (self *Timeslot) GetCID() []byte{
    suffix := make([]byte, 8)
    binary.LittleEndian.PutUint64(suffix, self.Cnt+self.adminOffset)
    self.Cnt = self.Cnt + 1
    cid := append(self.Prefix, suffix...)
    return cid
}

type TimeslotScheduler struct {
    Duration time.Duration
    Logger *log.Logger
}

func NewTimeslotScheduler (duration time.Duration) *TimeslotScheduler{
    return &TimeslotScheduler {
        Duration: duration,
        Logger: log.NewLogger("timeslot"),
    }
}

func (self *TimeslotScheduler) RunScheduler(ctx context.Context, feedback chan(int64)){
    current := time.Now().UnixNano()
    duration := self.Duration.Nanoseconds()
    timeslot := current / duration
    timeToNext := timeslot * duration + duration - current
    self.Logger.WithField("Time", timeToNext).Trace("waiting time")
    // log.Info("")
    time.Sleep(time.Duration(timeToNext))
    ticker := time.NewTicker(self.Duration)
    timeslot = timeslot + 1
    feedback <- timeslot
    for {
        select {
        case <- ticker.C:
            timeslot = timeslot + 1
            select { //consume old timeslot or do nothing
            case dbg := <- feedback:
                    self.Logger.WithField("old", dbg).Debug("Throw away old data")
                default:
            }
            feedback <- timeslot
        case <- ctx.Done():
            return
        }
    }
}
