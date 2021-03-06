package encoding

import (
    "fmt"
    "strconv"
    "bytes"
    "encoding/binary"
)

type DialResult struct {
    CID []byte
    Result int
}

const ONE byte = 49
const ZERO byte = 48


func ByteToBit(input byte) []byte {
    var buffer bytes.Buffer
    fmt.Fprintf(&buffer, "%.8b", input)
    byteStr := fmt.Sprintf("%s", buffer.Bytes())
    return []byte(byteStr)
}

func StrToBinary(s string) []byte {

    var b []byte

    for _, c := range s {
        fmt.Println(c)
        fmt.Println(int64(c))
        fmt.Println(int64(int(c)))
        b = strconv.AppendInt(b, int64(c), 2)
    }

    return b
}

func Uint64ToByte(num uint64) []byte {
    prefix := make([]byte, 8)
    binary.LittleEndian.PutUint64(prefix, num)
    return prefix
}

func ByteToUint64(block []byte) uint64 {
    num := binary.LittleEndian.Uint64(block)
    return num
}

func ToBinaryBytes(s string) string {
	var buffer bytes.Buffer
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(&buffer, "%.8b", s[i])
	}
	return fmt.Sprintf("%s", buffer.Bytes())
}

//WHY DO I HAVE TO DO THAT MYSELF!
func Reverse(s string) string {
    runes := []rune(s)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return string(runes)
}

func EncodeSentHeader (num uint64) []byte {
    bitString := strconv.FormatUint(num, 2)
    bitString = Reverse(bitString) //change endianess
    bs := []byte(string(bitString))
    if len(bs) < 16 {
        tmp := make([]byte, 16)
        for i := 0; i < len(bs); i++ {
            tmp[i] = bs[i]
        }
        bs = tmp
    }
    return bs
}

func DecodeSentHeader(received []byte) uint64 {

    numberReceived, _ := strconv.ParseUint(Reverse(string(received)), 2, 16)
    return numberReceived
}

func DecodeMessage(received []byte) string {
    blockLen := 8 // length of character
    // maxChar := len(received) % blockLen
    bitMessage := make([]byte, len(received)/8)
    for index := 0;  index + blockLen <= len(received); index += blockLen {
        chunk := received[index:index+blockLen]
        char,_ := strconv.ParseInt(string(chunk), 2, blockLen)
        bitMessage[index/blockLen] = byte(char)
    }
    return string(bitMessage)
}
