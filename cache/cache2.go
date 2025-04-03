package cache

import (
	"strings"
	"sync"
	"time"
)

type MemCache struct {
	mutex    sync.RWMutex
	duration time.Duration
	datamap  map[string]map[string]*time.Timer
}

func NewMemCache(dur time.Duration) *MemCache {
	return &MemCache{
		duration: dur,
		datamap:  make(map[string]map[string]*time.Timer),
	}
}

func (mc *MemCache) InsertOrRefreshDD(prefix, suffix string, maxCount int) bool {
	return mc.InsertOrRefresh(prefix, suffix, mc.duration, maxCount)
}

func (mc *MemCache) InsertOrRefresh(prefix, suffix string, duration time.Duration, maxCount int) bool {
	if prefix == "" || suffix == "" {
		return false
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	smap, ok := mc.datamap[prefix]

	if !ok {
		smap = make(map[string]*time.Timer)
		mc.datamap[prefix] = smap
	}

	tmr, ok := smap[suffix]
	if ok {
		tmr.Reset(duration)
		return true
	}

	if len(smap) >= maxCount {
		return false
	}

	smap[suffix] = time.AfterFunc(duration, func() { mc.delete(prefix, suffix) })

	return true
}

func (mc *MemCache) GetWords() []byte {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var sb strings.Builder
	for k1, v1 := range mc.datamap {
		sb.WriteString(k1 + "\r\n")
		for k2 := range v1 {
			sb.WriteString("\t" + k2 + "\r\n")
		}
	}

	return []byte(sb.String())
}

func (mc *MemCache) Delete(prefix, suffix string) bool {
	return mc.delete(prefix, suffix)
}

func (mc *MemCache) delete(prefix, suffix string) bool {
	if prefix == "" || suffix == "" {
		return false
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if smap, ok1 := mc.datamap[prefix]; ok1 {
		if tmr, ok2 := smap[suffix]; ok2 {
			tmr.Stop()
			delete(smap, suffix)
			if len(smap) == 0 {
				delete(mc.datamap, prefix)
			}
			return true
		}
	}

	return false
}
