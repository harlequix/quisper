module github.com/harlequix/quisper

go 1.15

require (
	github.com/jinzhu/copier v0.2.8
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/lucas-clemente/quic-go v0.19.3
	github.com/pkg/profile v1.6.0 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
)

replace github.com/lucas-clemente/quic-go => github.com/harlequix/quic-go v0.7.1-0.20210404152617-2377252de3cf
