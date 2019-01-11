package counter

import (
	"github.com/streamrail/concurrent-map"
)

func NewConcurrentMap() cmap.ConcurrentMap {
	return cmap.New()
}
