package quisper

import (
    // log "github.com/harlequix/quisper/log"
)

type DebugInterface interface {
    Emit(string, string)
    Subscribe(string, chan string)
}

type Debugger struct {
    eventMap map[string][]chan string
}

func NewDebugger()*Debugger{
    return &Debugger{
        eventMap: make(map[string][]chan string),
    }
}

func (self *Debugger)Emit(event string, msg string)  {
    if channels, ok := self.eventMap[event]; ok {
        for _, ch := range channels {
            select {
            case ch <- msg:
            default:
        }
        }
    }
}

func (self *Debugger) Subscribe (event string, callback chan string){
    if _, ok := self.eventMap[event]; !ok {
        self.eventMap[event] = make([]chan string, 0)
    }
    self.eventMap[event] = append(self.eventMap[event], callback)
}
