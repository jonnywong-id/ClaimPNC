// Berkas ini memenuhi seam Clock dengan jam sistem.
package clock

import "time"

// System adalah jam yang mengikuti waktu mesin, selalu dikembalikan dalam UTC.
//
// Konversi ke WIB tidak terjadi di sini melainkan di tempat waktu ditampilkan. Tidak
// ada penambahan 7 jam manual di mana pun (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4).
type System struct{}

// Now mengembalikan waktu saat ini dalam UTC.
func (System) Now() time.Time { return time.Now().UTC() }
