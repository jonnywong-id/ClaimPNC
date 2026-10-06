// Package waktu mendeklarasikan seam Jam.
//
// Seluruh waktu di aplikasi ini disimpan sebagai UTC dan hanya dikonversi ke WIB di
// satu tempat. Tidak ada penambahan 7 jam manual di mana pun — itu tepat kegagalan
// yang diwarisi sistem lama (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.2).
//
// Modul Clock yang lengkap beserta hari libur dan jam kerja adalah milik F-5; yang ada
// di sini hanya bagian terkecil yang dibutuhkan sesi agar dapat diuji deterministik.
package waktu

import "time"

// Jam adalah seam ke waktu berjalan.
type Jam interface {
	Sekarang() time.Time
}

// JamTetap adalah implementasi untuk pengujian: waktunya tidak bergerak kecuali
// digeser dengan Maju.
type JamTetap struct {
	waktu time.Time
}

// JamTetapPada membuat jam yang berhenti pada waktu tertentu.
func JamTetapPada(t time.Time) *JamTetap {
	return &JamTetap{waktu: t.UTC()}
}

// Sekarang mengembalikan waktu yang sedang dipegang jam ini.
func (j *JamTetap) Sekarang() time.Time { return j.waktu }

// Maju menggeser jam ke depan sebanyak d.
func (j *JamTetap) Maju(d time.Duration) { j.waktu = j.waktu.Add(d) }
