package encoding

import (
    "testing"
    "fmt"
)

func TestHeader(t *testing.T) {
    var blocks uint64 = 355
    test,err := EncodeSentHeader(blocks, H4)
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(test)
    // flip(test, 8)
    // flip(test, 4)
    out, err := DecodeSentHeader(test, H4)
    fmt.Println(out)
    fmt.Println(out == blocks)

}
