package backends

import (
    // "fmt"
    "time"
    "crypto/tls"
    quic "github.com/lucas-clemente/quic-go"
    log "github.com/harlequix/quisper/log"
    // "github.com/sirupsen/logrus"
    )
var logger *log.Logger
func init() {
    logger = log.NewLogger("BackendNative")
}

type NativeBackend struct {
    addr string
    tlsconfig *tls.Config
    config *quic.Config
}

func defaultConfig() *quic.Config{
    return &quic.Config{
        HandshakeIdleTimeout: time.Second * 2,
        KeepAlive: true,
        MaxIdleTimeout: time.Second * 60,
    }
}

func NewNativeBackend(addr string, config *quic.Config) *NativeBackend {
    tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		// NextProtos:         []string{"echo"},
        NextProtos:         []string{"echo", "h3-29"},
	}
    if config == nil {
        config = defaultConfig()
    }
    return &NativeBackend {
        addr: addr,
        config: config,
        tlsconfig: tlsConf,
    }
}

func (self *NativeBackend) Dial (cid []byte) (quic.Session, error) {
    // fmt.Println("foobar2")
    newGen := quic.GenConnectionID(cid)
    // fmt.Println("foobar")
    session, err := quic.DialAddr(self.addr, self.tlsconfig, self.config, newGen)
    // fmt.Println(err)
    // _ = session
    return session, err
}
