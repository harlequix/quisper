package quisper

import(
    prot "github.com/harlequix/quisper/protocol"
)

func (self *Writer) dispatchControlled(cid *prot.CID, controlQueue chan(*DialResult))  {
    select {
        case <-controlQueue:
            self.logger.WithField("CID", cid).Trace("can continue immediately")
        default:
            self.logger.WithField("CID", cid).Trace("Wait for queue ticket")
            <- controlQueue
    }
    // <- controlQueue
    go self.dispatch(cid, []chan(*DialResult){controlQueue})
}
