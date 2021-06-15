package decoding

import (
    _ "io"
    "github.com/harlequix/quisper/timeslots"
    _ "strconv"
    log "github.com/harlequix/quisper/log"
    prot "github.com/harlequix/quisper/protocol"
    // "github.com/harlequix/quisper/internal/format"
    "github.com/harlequix/quisper/probing"
)


type Decoder struct {
    pattern []*probing.Pattern
    patternMap map[uint64]map[uint64]*probing.Pattern
    log *log.Logger
    maxRead int
    feedback chan(byte)
    interpreter probing.Strategy
}

func NewDecoder(feedback chan(byte), interpreter probing.Strategy) *Decoder {
    return &Decoder{
        pattern : []*probing.Pattern{},
        patternMap : make(map[uint64]map[uint64]*probing.Pattern),
        log: log.NewLogger("Decoder"),
        maxRead: 0,
        feedback: feedback,
        interpreter: interpreter,
    }
}

func (self *Decoder) AddTimeslot(slot *timeslots.Timeslot, blocksSent uint64){
    self.patternMap[slot.Num] = make(map[uint64]*probing.Pattern)
    for i := uint64(0); i < blocksSent; i++ {
        pattern := probing.EmptyPattern(self.interpreter.PatternLen())
        self.pattern = append(self.pattern, pattern)
        self.patternMap[slot.Num][uint64(i)] = pattern
    }
}

func (self *Decoder) SetCID (cid *prot.CID, value byte)  {
    prefix, suffix := cid.Cut()
    suffix = suffix - 24
    index := suffix/uint64(self.interpreter.PatternLen())
    offset := suffix % uint64(self.interpreter.PatternLen())
    self.patternMap[prefix][index].SetBit(int(offset), value)
    if self.interpreter.NewBit(self.patternMap[prefix][index], int(offset)){
        self.log.Trace("received a new Bit")
    }
    // self.log.WithField("Prefix", prefix).WithField("CID", cid.String()).WithField("offset", offset).WithField("array stelle", index).WithField("Block", self.timeslotMap[prefix][index].String()).WithField("status", self.timeslotMap[prefix][index].Ready()).WithField("val", value).Warn("Set CID")
    self.yield()
}

func (self *Decoder) yield() {
    for index := self.maxRead; index < len(self.pattern); index++ {
        pattern := self.pattern[index]
        // self.log.Debugf("%d, s", index, block.String(), block.Ready())
        // if block.Ready() {
        //     self.feedback <- block.GetValue()
        //     self.maxRead = index  + 1
        // } else {
        //     break
        // }
        if self.interpreter.IsComplete(pattern) {
            block := self.interpreter.InterpretPattern(pattern)
            self.maxRead = index + 1
            self.feedback <- block.GetValue()
        } else {
            break
        }
    }
}
