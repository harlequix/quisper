package backends

import (
    // "fmt"
    "time"
    "crypto/tls"
    quic "github.com/lucas-clemente/quic-go"
    log "github.com/harlequix/quisper/log"
    // "github.com/sirupsen/logrus"
    "github.com/spf13/viper"
    )
var logger *log.Logger
func init() {
    logger = log.NewLogger("BackendNative")
}

func init() {
    viper.SetDefault("NativeTimeout", 2 * time.Second)
    viper.SetDefault("Protocols", []string{"echo", "h3-29"})
}

type NativeBackend struct {
    addr string
    tlsconfig *tls.Config
    config *quic.Config
}

type NativeBackendConfig struct {
    Protocols []string
    NativeTimeout time.Duration
}

func getConfig() *NativeBackendConfig {
    var config NativeBackendConfig
    viper.Unmarshal(&config)
    return &config
}

func defaultConfig() *quic.Config{
    return &quic.Config{
        HandshakeIdleTimeout: time.Second * 2,
        KeepAlive: true,
        MaxIdleTimeout: time.Second * 60,
    }
}

func NewNativeBackend(addr string, config *quic.Config) *NativeBackend {
    cfg := getConfig()
    tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		// NextProtos:         []string{"echo"},
        NextProtos:         cfg.Protocols,
	}
    if config == nil {
        config = &quic.Config {
            HandshakeIdleTimeout: cfg.NativeTimeout,
            KeepAlive: true,
            MaxIdleTimeout: time.Second * 60,
        }
    }
    logger.WithField("config", cfg).Info("creating new Backend")
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
