package probing

import (
    "github.com/harlequix/quisper/internal/format"
    "github.com/harlequix/quisper/internal/encoding"
    prot "github.com/harlequix/quisper/protocol"
)

type StrategySingle struct {
    patternLen int
}

func NewSingleProbing () *StrategySingle {
    return &StrategySingle{
        patternLen: prot.BlockLen,
    }
}


func (self *StrategySingle) GenPattern(block *format.Block) *Pattern {
    pattern := BlankPattern()
    pattern.Pattern = make([]byte, self.patternLen)
    sentCids := 0
    for pos, value := range block.GetBits(){
        if value == encoding.ONE {
            pattern.Pattern[pos] = encoding.ONE
            sentCids++
        } else {
            pattern.Pattern[pos] = encoding.ZERO
        }
    }
    pattern.SentCids = sentCids
    return pattern
}

func (self *StrategySingle) IsComplete(pattern *Pattern) bool {
    for _, value := range pattern.Pattern {
        if value == 0 {
            return false
        }
    }
    return true
}

func (self *StrategySingle) InterpretPattern(pattern *Pattern) *format.Block {
    block := format.NewBlock()
    for pos := range pattern.Pattern {
        block.SetBit(pos, pattern.Pattern[pos])
    }
    return block
}

func (self *StrategySingle) PatternLen() int {
    return self.patternLen
}

func (self *StrategySingle) NewBit(pattern *Pattern, offset int) bool {
    return true
}
