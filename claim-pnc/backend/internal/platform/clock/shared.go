package clock

import (
	"time"
)

// Jakarta adalah zona Asia/Jakarta; bila basis data zona tidak tersedia, dipakai zona tetap
// WIB (UTC+7).
func Jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}
