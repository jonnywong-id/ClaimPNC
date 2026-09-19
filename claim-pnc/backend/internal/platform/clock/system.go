// Berkas ini memenuhi seam Jam dengan jam sistem.
package clock

import "time"

// SystemClock adalah jam yang mengikuti waktu mesin, selalu dikembalikan dalam UTC.
//
// Konversi ke WIB tidak terjadi di sini melainkan di tempat waktu ditampilkan. Tidak
// ada penambahan 7 jam manual di mana pun (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4).
type SystemClock struct{}

// Now mengembalikan waktu saat ini dalam UTC.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
