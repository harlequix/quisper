package encoding

import (
    _ "io"
    "fmt"
    "github.com/harlequix/quisper/timeslots"
    "strconv"
    log "github.com/harlequix/quisper/log"
)

type Decoder struct {
    field []byte
    timeslotMap map[uint64]uint64
    cap uint64
    complete int
    read int
    log *log.Logger
}

func NewDecoder() *Decoder {
    return &Decoder{
        field : make([]byte, 65535),
        timeslotMap : make(map[uint64]uint64),
        cap : 0,
        complete : 0,
        read: 0,
        log: log.NewLogger("Decoder"),
    }
}

func (self *Decoder) AddTimeslot(slot *timeslots.Timeslot, bitsSent uint64){
    self.log.WithField("cap", self.cap).WithField("bitsSent", bitsSent).Warn("WTF")
    self.timeslotMap[uint64(slot.Num)] = self.cap
    self.cap += bitsSent
    self.log.WithField("Timeslot", slot.Num).WithField("offset", self.timeslotMap[uint64(slot.Num)]).Warn("Register")
}

func (self *Decoder) SetCID (slotID uint64, CID uint64, value byte)  {
    slotOff := self.timeslotMap[slotID]
    self.field[slotOff + CID] = value
    self.log.WithField("Prefix", slotID).WithField("CID", CID).WithField("offset", slotOff).WithField("array stelle", slotOff + CID).Warn("Set CID")

    self.yield()
}

func (self *Decoder) yield() {
    blockLen := 8
    for index := self.complete; index < len(self.field); index++ {
        self.log.WithField("array", self.field[:self.complete]).Warn("current ready")
        if self.field[index] == 0 {
            self.complete = index
            break
        }
    }
    for index := self.read; index + blockLen < self.complete; index = index + blockLen {
        chunk := self.field[index:index+blockLen]
        bitString, _ := strconv.ParseInt(string(chunk), 2, blockLen)
        self.read = index + blockLen
        byteR := byte(bitString)
        fmt.Print(string(byteR))
    }
}
