package tools

import "sync"

type Stat struct {
	Store    map[string]int
	StatLock *sync.RWMutex
}

func (stat *Stat) Incre(key string, amount int) {
	stat.StatLock.Lock()
	stat.Store[key] += amount
	stat.StatLock.Unlock()
}

func (stat *Stat) Get(key string) (r int) {
	stat.StatLock.RLock()
	r = stat.Store[key]
	stat.StatLock.RUnlock()
	return r
}
