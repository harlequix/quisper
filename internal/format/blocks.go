package format

import (
    "github.com/harlequix/quisper/internal/encoding"
    "github.com/harlequix/quisper/timeslots"
    prot "github.com/harlequix/quisper/protocol"
    "strconv"
)

type Block struct {
    field []byte
    blockLen int
}

func NewBlock() *Block {
    field := make([]byte, prot.BlockLen)
    return &Block{
        field:field,
        blockLen: prot.BlockLen,
    }
}

func (self *Block) SetByte (value byte) {
    bitString := encoding.ByteToBit(value)
    copy(self.field, bitString)
}

func (self *Block) GetCIDs(slot *timeslots.Timeslot, offset uint64) []*prot.CID {
    out := make([]*prot.CID, len(self.field))
    for i := range self.field {
        out[i] = slot.GetBodyCID(offset*uint64(prot.BlockLen)+uint64(i))
    }
    return out
}

func (self *Block) SetBit(offset int, value byte){
    self.field[offset] = value
}

func (self *Block) Len()int {
    return len(self.field)
}
func (self *Block) Ready() bool {
    for _, val := range self.field {
        if val == 0 {
            return false
        }
    }
    return true
}

func (self *Block) GetValue() byte {
    val,_ := strconv.ParseInt(string(self.field), 2, self.blockLen)
    return byte(val)
}

func (self *Block) GetBits () []byte {
    return self.field
}

func (self *Block) String() string {
    out := ""
    for _, val := range self.field {
        if val == encoding.ONE {
            out += "1"
        } else if val == encoding.ZERO {
            out += "0"
        } else {
            out += "_"
        }
    }
    return out
}
