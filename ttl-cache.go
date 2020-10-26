package ttl_cache

import (
	"sort"
	"time"
)

type key string
type cacheEntry struct {
	value interface{}
	key key
	exp uint32
}
type TTLCache struct {
	defaultTTL time.Duration
	cache map[key]*cacheEntry
	sweepTicker *time.Ticker
	ttlHK []*cacheEntry
}

func NewTTLCache(numSize uint, defaultTTL, sweepPeriod time.Duration) (*TTLCache, error) {
	if numSize <= 0 {
		return nil, newInvalidSizeErr(numSize)
	}

	if defaultTTL <= 0 * time.Second {
		return nil, newInvalidTTLErr(defaultTTL)
	}

	if sweepPeriod <= 0 * time.Second {
		return nil, newInvalidSweepPeriodErr(sweepPeriod)
	}

	return &TTLCache{
		defaultTTL: defaultTTL,
		cache: make(map[key]*cacheEntry, numSize),
		sweepTicker: time.NewTicker(sweepPeriod),
		ttlHK: make([]*cacheEntry, 0, numSize),
	}, nil
}

func newCacheEntry(key key, value interface{}, exp uint32) *cacheEntry {
	return &cacheEntry{
		key: key,
		value: value,
		exp: exp,
	}
}

func (c *TTLCache) Set(key key, value interface{}, optTTL ...time.Duration) error {
	ttl := c.defaultTTL
	if len(optTTL) > 0 && optTTL[0] > 0 {
		ttl = optTTL[0]
	}
	entry := newCacheEntry(key, value, getExp(ttl))

	if _, exists := c.cache[key]; exists {
		return c.updateCacheEntry(entry)
	}

	c.cache[entry.key] = entry
	c.insertNewHKEntry(entry)
	return nil
}

func (c *TTLCache) Get(key key) (interface{}, error) {
	entry, exists := c.cache[key]
	if !exists {
		return nil, newKeyNotFoundErr(key)
	}

	return entry.value, nil
}

//Possible optimization: use Sort.Search to find new location for entry; use multiple copy() to remove existing entry and re-add
//Benchmark it
func (c *TTLCache) updateCacheEntry(entry *cacheEntry) error {
	existingValue, exists := c.cache[entry.key]
	if !exists {
		return newBadUpdateRequestErr(entry.key)
	}

	existingValue.value = entry.value
	existingValue.exp = entry.exp

	sort.Slice(c.ttlHK, func(i, j int) bool {
		return c.ttlHK[i].exp >= c.ttlHK[i].exp
	})

	return nil
}

func (c *TTLCache) insertNewHKEntry(entry *cacheEntry) {
	i := sort.Search(len(c.ttlHK), func(i int) bool {
		return c.ttlHK[i].exp >= entry.exp
	})
	c.ttlHK = append(c.ttlHK, &cacheEntry{})
	copy(c.ttlHK[i+1:], c.ttlHK[i:])
	c.ttlHK[i] = entry
}

func getExp(ttl time.Duration) uint32 {
	return uint32(time.Now().Add(ttl).Unix())
}
