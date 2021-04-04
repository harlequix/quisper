package protocol

import (
    "github.com/harlequix/quisper/internal/encoding"
    // "encoding/binary"
    // "fmt"
    "strconv"
)

const length int = 16
var suffix int = 8

type CID struct {
    Field []byte
}

func NewCID(field []byte) *CID {
        internal := make([]byte, length)
        copy(internal, field)
        return &CID{
            Field: internal,
        }
}

func NewCIDuint(integers ...uint64) *CID {
    cid := []byte{}
    for _, num := range integers {
        cid = append(cid, encoding.Uint64ToByte(num)...)
    }
    return NewCID(cid)
}

func (self *CID) String() string {
    out := ""
    blocksize := 8
    for index := 0; index + blocksize <= len(self.Field); index+=blocksize {
        block := self.Field[index:index+blocksize]
        num := encoding.ByteToUint64(block)
        str := strconv.FormatUint(num, 10)
        if out != "" {
            out += ":"
        }
        out += str
    }
    return out
}

func (self * CID) Bytes() []byte {
    return self.Field
}

func (self *CID) Cut() (uint64, uint64) {
    return encoding.ByteToUint64(self.Field[:suffix]), encoding.ByteToUint64(self.Field[suffix:])
}
