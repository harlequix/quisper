package backends
import
(
    "testing"
    "github.com/harlequix/quisper/timeslots"
)


func TestScheduler(t *testing.T) {
    t.Log("foobar")
    backend := NewLSQuicBackend("192.168.0.2:12345")
    _ = backend
    var timeslotNum uint64 = 200000
    timeslot := timeslots.NewTimeslot(timeslotNum)
    cid := timeslot.GetGenCID(15)
    str := backend.Dial(cid.Bytes())
    t.Log(str)

}
