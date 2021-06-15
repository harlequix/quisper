package probing

import(
    prot "github.com/harlequix/quisper/protocol"
    "github.com/harlequix/quisper/timeslots"
    "github.com/harlequix/quisper/internal/encoding"
)

type Pattern struct {
    Pattern []byte
    Cids []*prot.CID
    SentCids int
}

func BlankPattern() *Pattern {
    return &Pattern{}
}

func EmptyPattern(len int) *Pattern {
    return &Pattern{
        Pattern: make([]byte, len),
    }
}

func (self *Pattern) GetCids(slot *timeslots.Timeslot, offset uint64)[]*prot.CID{
    out := make([]*prot.CID, len(self.Pattern))
    for i := range self.Pattern {
        out[i] = slot.GetBodyCID(offset*uint64(prot.BlockLen)+uint64(i))
    }
    return out
}

func (self *Pattern) GetSenderCids(slot *timeslots.Timeslot, offset uint64)[]*prot.CID{
    out := make([]*prot.CID, self.SentCids)
    pos := 0
    for i, val := range self.Pattern {
        if val == encoding.ONE {
            out[pos] = slot.GetBodyCID(offset*uint64(len(self.Pattern))+uint64(i))
            pos++
        }
    }
    return out
}

func (self *Pattern) SetBit(pos int, val byte){
    self.Pattern[pos] = val
}
