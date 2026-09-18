// Package clock mendeklarasikan seam Clock.
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

// Fixed adalah implementasi untuk pengujian: waktunya tidak bergerak kecuali digeser
// dengan Advance.
type Fixed struct {
	now time.Time
}

// FixedAt membuat jam yang berhenti pada waktu tertentu.
func FixedAt(t time.Time) *Fixed {
	return &Fixed{now: t.UTC()}
}

// Now mengembalikan waktu yang sedang dipegang jam ini.
func (f *Fixed) Now() time.Time { return f.now }

// Advance menggeser jam ke depan sebanyak d.
func (f *Fixed) Advance(d time.Duration) { f.now = f.now.Add(d) }
