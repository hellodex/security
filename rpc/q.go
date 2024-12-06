package rpc

import (
	"sync"

	sys "github.com/hellodex/HelloSecurity/system"
)

var slotQueue = sys.NewSlotQueue()
var unqiueSlotMap = make(map[uint64]bool)
var slotMapLock = sync.Mutex{}

func ReadSloqQueue() *sys.SlotQueue {
	return slotQueue
}
