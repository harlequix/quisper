package probing
import (
    "github.com/harlequix/quisper/internal/format"
)


type Strategy interface{
    GenPattern(*format.Block) *Pattern
    IsComplete(*Pattern) bool
    InterpretPattern(*Pattern)*format.Block
    PatternLen()int
    NewBit(*Pattern, int) bool
}
