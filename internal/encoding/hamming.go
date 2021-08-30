package encoding

import (
    "errors"
    "fmt"
)

var places map[int][]int


func init(){
    places = make(map[int][]int)
    places[0] = []int{0,2,4,8,10,12,14}
    places[1] = []int{1,2,5,6,9,10,13,14}
    places[3] = []int{3,4,5,6,11,12,13,14}
    places[7] = []int{7,8,9,10,11,12,13,14}
}

func EncodeHamming4 (b []byte) ([]byte,error) {
    if len(b) > 11 {
        return nil, errors.New("Bit slice is too long, cannot encode")
    }
    bitfield := make([]byte, 16)
    newit := 0
    for bitit := range bitfield {
        if bitit == 0 || bitit == 1 || bitit == 3 || bitit == 7 {
            bitfield[bitit] = 1
            continue
        }
        if newit < len(b) {
            bitfield[bitit] = b[newit]
        } else {
            bitfield[bitit] = ZERO
        }
        newit++
    }
    for key := range places {
        par := calculateParity(bitfield, places[key])
        if par {
            bitfield[key] = ONE
        } else {
            bitfield[key] = ZERO
        }
    }
    return bitfield,nil
}

func DecodeHamming4(b []byte) ([]byte, error){
    checkNull := true
    for place := range b {
        if b[place] == ONE {
            checkNull = false
        }
    }
    if checkNull == true{
        out := make(byte, 12)
        for place := range out {
            out[place] = ZERO
        }
        return out
    }
    errorPlace := 0
    foundError := false
    for par := range places{
        check := calculateParity(b, places[par])
        if check {
            foundError = true
            errorPlace += par+1
        }
    }
    if foundError {
        if b[errorPlace-1] == ONE {
            b[errorPlace-1] = ZERO
        } else {
            b[errorPlace-1] = ONE
        }
    }
    if foundError {
        return stripCode(b), errors.New(fmt.Sprintf("found error in %d", errorPlace - 1))
    }
    return stripCode(b), nil
}

func stripCode(b []byte) []byte{
    out := make([]byte, 12)
    bIt := 0
    for i := range b {
        _, ok := places[i]
        if !ok {
            out[bIt] = b[i]
            bIt++
        }
    }
    return out
}

func calculateParity(bits []byte, places []int) bool{
    count := 0
    for _, place := range places {
        if bits[place] == ONE{
            count++
        }
    }
    if count % 2 == 0 {
        return true
    }
    return false
}
