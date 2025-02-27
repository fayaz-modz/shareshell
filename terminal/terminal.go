package terminal

import (
	"os"
	"sync"
)

type Terminal struct {
	Ptmx         *os.File
	ResizeChan   chan os.Signal
	RemoteWidth  uint16
	RemoteHeight uint16
	Wg           sync.WaitGroup
}


