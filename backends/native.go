package backends

import (
    // "fmt"
    // "time"
    "crypto/tls"
    quic "github.com/lucas-clemente/quic-go"
    log "github.com/harlequix/quisper/log"
    // "github.com/sirupsen/logrus"
    )
    const  addr = "127.0.0.1:4433"
var logger *log.Logger
func init() {
    logger = log.NewLogger("BackendNative")
}

type NativeBackend struct {
    addr string
    tlsconfig *tls.Config
    config *quic.Config
}

func NewNativeBackend(addr string, config *quic.Config) *NativeBackend {
    tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{},
	}
    return &NativeBackend {
        addr: addr,
        config: config,
        tlsconfig: tlsConf,
    }
}

func (self *NativeBackend) Dial (cid []byte) error {
    // fmt.Println("foobar2")
    newGen := quic.GenConnectionID(cid)
    // fmt.Println("foobar")
    session, err := quic.DialAddr(self.addr, self.tlsconfig, self.config, newGen)
    // fmt.Println("foobar3")
    _ = session
    return err
}
