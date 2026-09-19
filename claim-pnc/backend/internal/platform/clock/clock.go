// Package clock mendeklarasikan seam Jam.
//
// Seluruh waktu di aplikasi ini disimpan sebagai UTC dan hanya dikonversi ke WIB di
// satu tempat. Tidak ada penambahan 7 jam manual di mana pun — itu tepat kegagalan
// yang diwarisi sistem lama (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.2).
//
// Modul Clock yang lengkap beserta hari libur dan jam kerja adalah milik F-5; yang ada
// di sini hanya bagian terkecil yang dibutuhkan sesi agar dapat diuji deterministik.
package clock

import "time"

// Clock adalah seam ke waktu berjalan.
type Clock interface {
	Now() time.Time
}

// FixedClock adalah implementasi untuk pengujian: waktunya tidak bergerak kecuali
// digeser dengan Maju.
type FixedClock struct {
	clock time.Time
}

// FixedClockAt membuat jam yang berhenti pada waktu tertentu.
func FixedClockAt(t time.Time) *FixedClock {
	return &FixedClock{clock: t.UTC()}
}

// Now mengembalikan waktu yang sedang dipegang jam ini.
func (j *FixedClock) Now() time.Time { return j.clock }

// Advance menggeser jam ke depan sebanyak d.
func (j *FixedClock) Advance(d time.Duration) { j.clock = j.clock.Add(d) }
