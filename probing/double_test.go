package probing

import (
    "testing"
    "github.com/harlequix/quisper/internal/format"
    "fmt"
    "github.com/harlequix/quisper/timeslots"
)

func TestSum(t *testing.T) {
    testInput := []byte("A")
    fmt.Println(testInput)
    testBlock := format.NewBlock()
    testBlock.SetByte(testInput[0])
    fmt.Println(testBlock.String())
    unit := NewDoubleProbing()
    testPattern := unit.GenPattern(testBlock)
    fmt.Println(testPattern)
    fakeSlot := timeslots.NewTimeslot(uint64(0))
    testCids := testPattern.GetSenderCids(fakeSlot, uint64(0))
    testCids2 := testPattern.GetSenderCids(fakeSlot, uint64(1))
    fmt.Println(testCids)
    fmt.Println(testCids2)
    _ = unit
}
