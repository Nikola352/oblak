package vm

import "sync/atomic"

var nextCid atomic.Uint32

func init() {
	nextCid.Store(3)
}

func allocateCid() uint32 {
	return nextCid.Add(1) - 1
}
