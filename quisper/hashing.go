package quisper

import (
    // log "github.com/harlequix/quisper/log"
    prot "github.com/harlequix/quisper/protocol"
    // "golang.org/x/crypto/sha3"
)

func (self *Writer)hashCID(cid *prot.CID) *prot.CID {
    cidlength := 20
    cidbyte := make([]byte, cidlength)
    hasher := self.hashTemplate.Clone()
    hasher.Write(cid.Field)
    hasher.Read(cidbyte)
    out := prot.NewCID(cidbyte)
    return out
}
