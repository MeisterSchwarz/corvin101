package zones

import "sync"

var (
	contributeMu   sync.RWMutex
	isContributing bool
)

// enables or disables contribution mode
func SetContributing(enabled bool) {
	contributeMu.Lock()
	isContributing = enabled
	contributeMu.Unlock()
}

// returns the current contribution state
func IsContributing() bool {
	contributeMu.RLock()
	defer contributeMu.RUnlock()
	return isContributing
}
