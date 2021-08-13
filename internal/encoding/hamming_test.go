package encoding

import (
    "testing"
    "fmt"
)

func TestBasic(t *testing.T) {
    fmt.Println("HelloWorld")
}

func TestHappy(t *testing.T) {
    payload := []byte{49,49,49,48,48,48,48,48,48,48,49}
    encoded, _ :=EncodeHamming4(payload)
    fmt.Println(encoded)
    flip(encoded, 11)
    decodes, err := DecodeHamming4(encoded)
    fmt.Println(err)
    fmt.Println(decodes)
}

func flip(b []byte, index int){
    if b[index] == ZERO{
        b[index] = ONE
    } else {
        b[index] = ZERO
    }
}
