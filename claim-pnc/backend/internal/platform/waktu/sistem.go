// Berkas ini memenuhi seam Jam dengan jam sistem.
package waktu

import "time"

// JamSistem adalah jam yang mengikuti waktu mesin, selalu dikembalikan dalam UTC.
//
// Konversi ke WIB tidak terjadi di sini melainkan di tempat waktu ditampilkan. Tidak
// ada penambahan 7 jam manual di mana pun (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4).
type JamSistem struct{}

// Sekarang mengembalikan waktu saat ini dalam UTC.
func (JamSistem) Sekarang() time.Time { return time.Now().UTC() }
