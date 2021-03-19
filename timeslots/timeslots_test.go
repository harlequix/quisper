package timeslots
import
(
    "context"
    "time"
    "testing"
    "fmt"
)


func TestScheduler(t *testing.T) {
    feedback1 := make(chan int64)
    ctx1 := context.Background()

    feedback2 := make(chan int64)
    ctx2 := context.Background()

    fmt.Println("fmt")
    scheduler := NewTimeslotScheduler(10*time.Second)
    go scheduler.RunScheduler(ctx1, feedback1)
    go scheduler.RunScheduler(ctx2, feedback2)
    t.Log("Foooo")
    for {
        select {
        case new := <- feedback1:
            fmt.Println("scheduler1")
            fmt.Println(new)
        case new2 := <- feedback2:
            fmt.Println("scheduler2")
            fmt.Println(new2)
        }
    }


}
