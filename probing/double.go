package probing

import (
    "github.com/harlequix/quisper/internal/format"
    "github.com/harlequix/quisper/internal/encoding"
    prot "github.com/harlequix/quisper/protocol"
)


type StrategyDouble struct {
    patternLen int
}

func NewDoubleProbing () *StrategyDouble {
    return &StrategyDouble{
        patternLen: 2*prot.BlockLen,
    }
}

func (self *StrategyDouble) GenPattern(block *format.Block) *Pattern {
    pattern := EmptyPattern(2*block.Len())
    for index, value := range block.GetBits(){
        if value == encoding.ONE {
            pattern.Pattern[2*index+1] = encoding.ONE
        } else {
            pattern.Pattern[2*index+0] = encoding.ONE
        }
    }
    pattern.SentCids = block.Len()
    return pattern
}

func (self *StrategyDouble) IsComplete(pattern *Pattern) bool{ //TODO doesnt work
    for index := 0; index < self.patternLen; index+=2 {
        if !isSet(pattern.Pattern, index) && !isSet(pattern.Pattern, index+1){
            return false
        }
    }
    return true
}

func (self* StrategyDouble) InterpretPattern(pattern *Pattern)*format.Block { //TODO doesnt work
    block := format.NewBlock()
    for index := 0; index < block.Len(); index++ {
        if pattern.Pattern[2*index] == encoding.ZERO {
            block.SetBit(index, encoding.ONE)
        } else if pattern.Pattern[2*index+1] == encoding.ZERO {
            block.SetBit(index, encoding.ZERO)
        } else if pattern.Pattern[2*index] == encoding.ONE {
            block.SetBit(index, encoding.ZERO)
        } else if pattern.Pattern[2*index+1] == encoding.ONE {
            block.SetBit(index, encoding.ONE)
        } else {
            //TODO this should not happen
        }
    }
    return block
}
func (self *StrategyDouble) PatternLen()int {
    return self.patternLen
}
func (self *StrategyDouble) NewBit(pattern *Pattern, pos int) bool { //TODO doesnt work
    var partnerPos int
    if pos%2 == 0 {
        partnerPos = pos + 1
    } else {
        partnerPos = pos - 1
    }
    if isSet(pattern.Pattern, partnerPos){
        return false
    } else {
        return true
    }
}


func isSet(field []byte, offset int) bool {
    return ( field[offset] == encoding.ONE || field[offset] == encoding.ZERO)
}
