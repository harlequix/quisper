package backends
import
(
    "testing"
    "github.com/harlequix/quisper/timeslots"
)


func TestScheduler(t *testing.T) {
    t.Log("foobar")
    backend := NewLSQuicBackend("127.0.0.1:12345")
    _ = backend
    var timeslotNum uint64 = 200000
    timeslot := timeslots.NewTimeslot(timeslotNum)
    cid := timeslot.GetGenCID(15)
    str := backend.patch(cid.Bytes())
    t.Log(str)

}
