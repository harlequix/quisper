package decoding

import (
    _ "io"
    "fmt"
    "github.com/harlequix/quisper/timeslots"
    _ "strconv"
    log "github.com/harlequix/quisper/log"
    prot "github.com/harlequix/quisper/protocol"
    "github.com/harlequix/quisper/internal/format"
)


type Decoder struct {
    field []*format.Block
    timeslotMap map[uint64]map[uint64]*format.Block
    log *log.Logger
}

func NewDecoder() *Decoder {
    return &Decoder{
        field : []*format.Block{},
        timeslotMap : make(map[uint64]map[uint64]*format.Block),
        log: log.NewLogger("Decoder"),
    }
}

func (self *Decoder) AddTimeslot(slot *timeslots.Timeslot, blocksSent uint64){
    self.timeslotMap[slot.Num] = make(map[uint64]*format.Block)
    for i := uint64(0); i < blocksSent; i++ {
        block := format.NewBlock()
        self.field = append(self.field, block)
        self.timeslotMap[slot.Num][uint64(i)] = block
    }
}

func (self *Decoder) SetCID (cid *prot.CID, value byte)  {
    prefix, suffix := cid.Cut()
    suffix = suffix - 24
    index := suffix/uint64(prot.BlockLen)
    offset := suffix % uint64(prot.BlockLen)
    self.timeslotMap[prefix][index].SetBit(int(offset), value)
    self.log.WithField("Prefix", prefix).WithField("CID", cid.String()).WithField("offset", offset).WithField("array stelle", index).WithField("Block", self.timeslotMap[prefix][index].String()).WithField("status", self.timeslotMap[prefix][index].Ready()).WithField("val", value).Warn("Set CID")
    self.yield()
}

func (self *Decoder) yield() {
    str := ""
    for  index , block := range self.field{
        fmt.Println("%d, s", index, block.String(), block.Ready())
        if block.Ready() {
            str += string(block.GetValue())
        } else {
            str += "_"
        }
    }
    fmt.Println(str)
}
