module github.com/harlequix/quisper

go 1.15

require (
	github.com/jinzhu/copier v0.2.8
	github.com/lucas-clemente/quic-go v0.19.3
	github.com/sirupsen/logrus v1.4.1
)

replace github.com/lucas-clemente/quic-go => github.com/harlequix/quic-go v0.7.1-0.20210309095737-56573e04b15e
