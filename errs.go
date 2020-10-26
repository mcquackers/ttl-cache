package ttl_cache

import (
	"fmt"
	"time"
)

func newInvalidSweepPeriodErr(invalidDur time.Duration) error {
	return fmt.Errorf("invalid sweep period %s; must be > 0s", invalidDur)
}

func newInvalidTTLErr(invalidTTL time.Duration) error {
	return fmt.Errorf("invalid TTL %s; must be > 0s", invalidTTL)
}

func newInvalidSizeErr(invalidSize uint) error {
	return fmt.Errorf("invalid cache size %d; must be > 0", invalidSize)
}

func newBadUpdateRequestErr(invalidKey key) error {
	return fmt.Errorf("invalid key for update request %s", invalidKey)
}
