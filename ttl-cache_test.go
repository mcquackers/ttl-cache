package ttl_cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type cacheSuite struct {
	size        uint
	defaultTTL  time.Duration
	sweepPeriod time.Duration
	cache       *TTLCache
	suite.Suite
}

func (cs *cacheSuite) SetupSuite() {
	var err error
	if cs.size == 0 {
		cs.size = 10
	}
	if cs.defaultTTL == 0 {
		cs.defaultTTL = 30 * time.Second
	}

	if cs.sweepPeriod == 0 {
		cs.sweepPeriod = 5 * time.Second
	}
	cs.cache, err = NewTTLCache(cs.size, cs.defaultTTL, cs.sweepPeriod)
	require.Nil(cs.T(), err)
}

//TestCases
//-Success
//--Normal Success
//
//-Error
//--Sweep Period = 0
//--TTL = 0
//--NumSize = 0?
func TestNewTTLCache_Creation(t *testing.T) {
	type tc struct {
		description   string
		numSize       uint
		defaultTTL    time.Duration
		sweepPeriod   time.Duration
		expectedCache *TTLCache
		expectedErr   error
	}

	tcs := []tc{
		{
			description: "normal success",
			numSize:     10,
			defaultTTL:  30 * time.Second,
			sweepPeriod: 5 * time.Second,
			expectedCache: &TTLCache{
				defaultTTL:  30 * time.Second,
				sweepTicker: time.NewTicker(5 * time.Second),
				cache:       make(map[key]*cacheEntry, 10),
				ttlHK:       make([]*cacheEntry, 0, 10),
			},
			expectedErr: nil,
		},
		{
			description:   "error - Sweep period <= 0s",
			numSize:       10,
			defaultTTL:    30 * time.Second,
			sweepPeriod:   0 * time.Second,
			expectedCache: nil,
			expectedErr:   newInvalidSweepPeriodErr(0 * time.Second),
		},
		{
			description:   "error - TTL <= 0s",
			numSize:       10,
			defaultTTL:    0 * time.Second,
			sweepPeriod:   5 * time.Second,
			expectedCache: nil,
			expectedErr:   newInvalidTTLErr(0 * time.Second),
		},
		{
			description:   "error - numSize <= 0",
			numSize:       0,
			defaultTTL:    30 * time.Second,
			sweepPeriod:   5 * time.Second,
			expectedCache: nil,
			expectedErr:   newInvalidSizeErr(0),
		},
	}

	for _, testCase := range tcs {
		t.Run(testCase.description, func(t *testing.T) {
			cache, err := NewTTLCache(testCase.numSize, testCase.defaultTTL, testCase.sweepPeriod)
			assertCachesAreEqual(t, testCase.expectedCache, cache)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

//TODO NewTTLCache_StartsTicker

func TestNewCacheEntry(t *testing.T) {
	type testVal struct {
		vals []int
	}

	testValInt := 5
	testValString := "string"
	testValStruct := &testVal{
		vals: []int{1, 2, 3},
	}
	testValPointer := &testVal{
		vals: []int{4, 5, 6},
	}

	type tc struct {
		description   string
		key           key
		value         interface{}
		exp           uint32
		expectedEntry *cacheEntry
	}

	tcs := []tc{
		{
			description: "int val",
			key:         key("int"),
			value:       testValInt,
			exp:         12345,
			expectedEntry: &cacheEntry{
				key:   key("int"),
				value: testValInt,
				exp:   12345,
			},
		},
		{
			description: "string val",
			key:         key("string"),
			value:       testValString,
			exp:         67890,
			expectedEntry: &cacheEntry{
				key:   key("string"),
				value: testValString,
				exp:   67890,
			},
		},
		{
			description: "struct val",
			key:         key("struct"),
			value:       testValStruct,
			exp:         45678,
			expectedEntry: &cacheEntry{
				key:   key("struct"),
				value: testValStruct,
				exp:   45678,
			},
		},
		{
			description: "pointer val",
			key:         key("struct"),
			value:       testValPointer,
			exp:         12390,
			expectedEntry: &cacheEntry{
				key:   key("struct"),
				value: testValPointer,
				exp:   12390,
			},
		},
	}

	for _, testCase := range tcs {
		t.Run(testCase.description, func(t *testing.T) {
			assert.Equal(t, testCase.expectedEntry, newCacheEntry(testCase.key, testCase.value, testCase.exp))
		})
	}
}

//TestCases
//-Success
//--New Entry correctly sorted
//--Existing Entry - Overwrite and update TTL
//--Full cache calls evict -- TODO
//--Add/Update to Cache is concurrent safe -- TODO
//
//-Error
//--Cache is full after evict-- TODO
func TestTTLCache_Set(t *testing.T) {
	css := new(setSuite)
	suite.Run(t, css)
}

type setSuite struct {
	cacheSuite
}

func (css *setSuite) SetupTest() {
	css.cacheSuite.SetupSuite()
}

func (css *setSuite) TestCache_Set_NewEntry() {
	expectedLen := 0
	keyOfEarlyExp := key("first")
	earlyExpVal := "first"

	err := css.cache.Set(keyOfEarlyExp, earlyExpVal)
	assert.Nil(css.T(), err)
	expectedLen++

	//Ensure new entry added to cache
	assert.Equal(css.T(), expectedLen, len(css.cache.cache))
	expectedEntry := newCacheEntry(keyOfEarlyExp, earlyExpVal, getExp(css.cache.defaultTTL))
	actualEntry, exists := css.cache.cache[keyOfEarlyExp]
	assert.True(css.T(), exists)
	assert.Equal(css.T(), expectedEntry, actualEntry)

	//Ensure new entry added to housekeeping slice
	assert.Equal(css.T(), expectedLen, len(css.cache.ttlHK))
	assert.Equal(css.T(), expectedEntry, css.cache.ttlHK[0])

	keyOfLaterExp := key("second")
	laterExpVal := "second"
	optTTL := 60 * time.Second
	err = css.cache.Set(keyOfLaterExp, laterExpVal, optTTL)
	assert.Nil(css.T(), err)
	expectedLen++

	//Ensure new entry added to cache with correct TTL
	assert.Equal(css.T(), expectedLen, len(css.cache.cache))
	expectedEntry = newCacheEntry(keyOfLaterExp, laterExpVal, getExp(optTTL))
	actualEntry, exists = css.cache.cache[keyOfLaterExp]
	assert.True(css.T(), exists)
	assert.Equal(css.T(), expectedEntry, actualEntry)

	//Ensure new entry added to housekeeping slice in correct place
	assert.Equal(css.T(), expectedLen, len(css.cache.ttlHK))
	assert.Equal(css.T(), expectedEntry, css.cache.ttlHK[1])
}

func (css *setSuite) TestCache_Set_OverwriteExisting() {
	key := key("key")
	initialValue := "string"
	expectedLen := 1

	//Set up existing entry
	err := css.cache.Set(key, initialValue)
	require.Nil(css.T(), err)
	assert.Equal(css.T(), expectedLen, len(css.cache.cache))
	assert.Equal(css.T(), expectedLen, len(css.cache.ttlHK))

	overwriteValue := 49
	//Overwrite existing value
	err = css.cache.Set(key, overwriteValue)
	require.Nil(css.T(), err)
	assert.Equal(css.T(), expectedLen, len(css.cache.cache))
	assert.Equal(css.T(), expectedLen, len(css.cache.ttlHK))
}

func TestCache_UpdateCache(t *testing.T) {
	uc := new(updateCacheSuite)
	suite.Run(t, uc)
}

type updateCacheSuite struct {
	e1 *cacheEntry
	e2 *cacheEntry
	cacheSuite
}

func (uc *updateCacheSuite) SetupTest() {
	uc.cacheSuite.SetupSuite()

	//Add two entries to cache
	uc.e1 = &cacheEntry{
		key:   key("key1"),
		value: "initialValue",
		exp:   12345,
	}
	uc.cache.cache[uc.e1.key] = uc.e1
	uc.cache.insertNewHKEntry(uc.e1)

	uc.e2 = &cacheEntry{
		key:   key("key2"),
		value: "initialValue",
		exp:   23456,
	}
	uc.cache.cache[uc.e2.key] = uc.e2
	uc.cache.insertNewHKEntry(uc.e2)

	//Ensure ttlHK is sorted by ascending entry.exp
	expectedLen := 2
	require.Equal(uc.T(), expectedLen, len(uc.cache.ttlHK))
	require.Equal(uc.T(), uc.e1, uc.cache.ttlHK[0])
	require.Equal(uc.T(), uc.e2, uc.cache.ttlHK[1])
}

func (uc *updateCacheSuite) TestUpdateCache_Success() {
	//update entry `e1`
	updateEntry := &cacheEntry{
		key:   uc.e1.key,
		value: 52,
		exp:   67890,
	}

	expectedLen := 2

	err := uc.cache.updateCacheEntry(updateEntry)
	assert.Nil(uc.T(), err)
	//ensure new entry not added
	assert.Equal(uc.T(), expectedLen, len(uc.cache.ttlHK))
	//ensure updated e1 with later exp is now after e2 in ttlHK
	assert.Equal(uc.T(), uc.e1, uc.cache.ttlHK[1])
	assert.Equal(uc.T(), uc.e2, uc.cache.ttlHK[0])
}

func (uc *updateCacheSuite) TestUpdateCache_InvalidRequest() {
	updateEntry := &cacheEntry{
		key:   key("invalid key"),
		value: 23,
		exp:   getExp(uc.cache.defaultTTL),
	}

	err := uc.cache.updateCacheEntry(updateEntry)
	assert.NotNil(uc.T(), err)
	assert.Equal(uc.T(), newBadUpdateRequestErr(updateEntry.key), err)
}

//TestCases
//-Success
//--Successfully found
//
//-Error
//--Not found
func TestCache_Get(t *testing.T) {
	gc := new(getCacheSuite)
	suite.Run(t, gc)
}

type getCacheSuite struct {
	key   key
	value interface{}
	entry *cacheEntry
	cacheSuite
}

func (gc *getCacheSuite) SetupSuite() {
	gc.cacheSuite.SetupSuite()
	gc.key = key("exists")
	gc.value = "value"
	gc.entry = newCacheEntry(gc.key, gc.value, getExp(gc.defaultTTL))
	gc.cache.cache[gc.key] = gc.entry
}

func (gc *getCacheSuite) TestCache_Get_Success() {
	value, err := gc.cache.Get(gc.key)
	assert.Nil(gc.T(), err)
	assert.Equal(gc.T(), gc.value, value)
}

func (gc *getCacheSuite) TestCache_Get_NotFound() {
	nonexistentKey := key("doesn't exist")
	value, err := gc.cache.Get(nonexistentKey)
	assert.Nil(gc.T(), value)
	assert.NotNil(gc.T(), err)
	assert.Equal(gc.T(), newKeyNotFoundErr(nonexistentKey), err)
}

//TODO
//Eviction

//prospective: Export Manual Eviction

func assertCachesAreEqual(t *testing.T, expected, actual *TTLCache) {
	if expected == nil || actual == nil {
		assert.Equal(t, expected, actual)
		return
	}
	//assert.Equal(t, expected.sweepTicker, actual.sweepTicker)
	assert.Equal(t, expected.defaultTTL, actual.defaultTTL)
	assert.Equal(t, len(expected.cache), len(actual.cache))
	assert.Equal(t, len(expected.ttlHK), len(actual.ttlHK))
	assert.Equal(t, cap(expected.ttlHK), cap(actual.ttlHK))
}
