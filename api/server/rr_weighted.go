package server

import (
	"strconv"
	"strings"
	"sync"
)

type SchedulerRRW struct {
	entry map[string]*RRW
	lock  sync.RWMutex
}

func NewScheduler() *SchedulerRRW {
	return &SchedulerRRW{
		entry: make(map[string]*RRW),
		lock:  sync.RWMutex{},
	}
}

func (s *SchedulerRRW) Get(key string) (*RRW, bool) {
    s.lock.RLock()
    defer s.lock.RUnlock()
    rrw, ok := s.entry[key]
    return rrw, ok
}

func (s *SchedulerRRW) Set(key string, rrw *RRW) {
    s.lock.Lock()
    defer s.lock.Unlock()
    s.entry[key] = rrw
}

// rrWeighted is a wrapped weighted item that is used to implement LVS weighted round robin algorithm.
type rrWeighted struct {
	Item   interface{}
	Weight int
}

// RRW is a struct that contains weighted items implement LVS weighted round robin algorithm.
//
// http://kb.linuxvirtualitem.org/wiki/Weighted_Round-Robin_Scheduling
// http://zh.linuxvirtualitem.org/node/37
type RRW struct {
	items []*rrWeighted
	n     int
	gcd   int
	maxW  int
	i     int
	cw    int
	data  string
}

func (w *RRW) AddFrom(source string) error {
	for _, entry := range strings.Split(source, ",") {
		parts := strings.Split(entry, "=")
		key := strings.TrimSpace(parts[0])
		val, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return err
		}
		w.Add(key, val)
	}

	w.data = source
	return nil
}

// Add a weighted item.
func (w *RRW) Add(item interface{}, weight int) {
	weighted := &rrWeighted{Item: item, Weight: weight}
	if weight > 0 {
		if w.gcd == 0 {
			w.gcd = weight
			w.maxW = weight
			w.i = -1
			w.cw = 0
		} else {
			w.gcd = gcd(w.gcd, weight)
			if w.maxW < weight {
				w.maxW = weight
			}
		}
	}
	w.items = append(w.items, weighted)
	w.n++
}

// All returns all items.
func (w *RRW) All() map[interface{}]int {
	m := make(map[interface{}]int)
	for _, i := range w.items {
		m[i.Item] = i.Weight
	}
	return m
}

// RemoveAll removes all weighted items.
func (w *RRW) RemoveAll() {
	w.items = w.items[:0]
	w.n = 0
	w.gcd = 0
	w.maxW = 0
	w.i = -1
	w.cw = 0
}

//Reset resets all current weights.
func (w *RRW) Reset() {
	w.i = -1
	w.cw = 0
}

// Next returns next selected item.
func (w *RRW) Next() interface{} {
	if w.n == 0 {
		return nil
	}

	if w.n == 1 {
		return w.items[0].Item
	}

	for {
		w.i = (w.i + 1) % w.n
		if w.i == 0 {
			w.cw = w.cw - w.gcd
			if w.cw <= 0 {
				w.cw = w.maxW
				if w.cw == 0 {
					return nil
				}
			}
		}

		if w.items[w.i].Weight >= w.cw {
			return w.items[w.i].Item
		}
	}
}

func gcd(x, y int) int {
	var t int
	for {
		t = (x % y)
		if t > 0 {
			x = y
			y = t
		} else {
			return y
		}
	}
}