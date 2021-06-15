package rtt

import (
    "time"
)


type Manager interface {
    GetMeasurement() time.Duration
    PlaceMeasurement(*RTT)
}
