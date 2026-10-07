// Package businessline memuat pilihan "Lini Bisnis" yang dipakai penyaring layar Dashboard
// Claim dan Inbox Close Claim. Kedua layar membaca daftar yang SAMA dari sistem lama, sehingga
// daftarnya tinggal di satu tempat dan tidak dapat berselisih tanpa ketahuan.
package businessline

import "strings"

// Line adalah satu pilihan lini bisnis.
type Line string

// Pilihan lini bisnis, dalam urutan tampilnya di layar.
const (
	All     Line = "ALL"
	NonMBU  Line = "NONMBU"
	Bonding Line = "BONDING"
	PA      Line = "PA"
	Travel  Line = "TRAVEL"
)

var lines = []Line{All, NonMBU, Bonding, PA, Travel}

// Lines mengembalikan salinan seluruh pilihan, berurutan.
func Lines() []Line {
	result := make([]Line, len(lines))
	copy(result, lines)
	return result
}

// Label adalah teks pilihan yang dibaca pengguna.
func (b Line) Label() string {
	switch b {
	case All:
		return "Semua Lini Bisnis"
	case NonMBU:
		return "Non-MBU"
	case Bonding:
		return "Bonding"
	case PA:
		return "Personal Accident"
	case Travel:
		return "Travel"
	default:
		return string(b)
	}
}

// Parse membaca pilihan dari teks permintaan; kosong berarti All. Nilai kedua false bila
// teksnya bukan salah satu pilihan.
func Parse(raw string) (Line, bool) {
	value := Line(strings.ToUpper(strings.TrimSpace(raw)))
	if value == "" {
		return All, true
	}
	for _, known := range lines {
		if known == value {
			return value, true
		}
	}
	return "", false
}
