package quisper

import (
    quic "github.com/lucas-clemente/quic-go"
    "fmt"
)

type ConnectionManager struct {
    sessionMap map[uint64][]quic.Session
}

func NewConnectionManager() *ConnectionManager {
    sessionMap := make(map[uint64][]quic.Session)
    return &ConnectionManager{
        sessionMap: sessionMap,
    }
}

func (self *ConnectionManager) Hold (slot uint64, session quic.Session) {
    if session == nil {
        return
    }
    go handleSession(session)
    if slice, found := self.sessionMap[slot]; found {
        slice = append(slice, session)
    } else {
        self.sessionMap[slot] = []quic.Session{session}
    }

}

func (self *ConnectionManager) Retire (slot uint64) {
    if slice, found := self.sessionMap[slot]; found {
        for _, session := range slice {
            fmt.Println(session.ConnectionState())
        }

    }
}

func handleSession(session quic.Session){
    
}
