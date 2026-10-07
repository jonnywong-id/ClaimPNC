package validation

import (
	"fmt"
	"strings"
)

// Length menolak isian yang lebih panjang dari max karakter.
func Length(field, label, value string, max int) []Violation {
	if len(value) > max {
		return []Violation{{
			Field:   field,
			Message: fmt.Sprintf("%s paling panjang %d karakter.", label, max),
		}}
	}
	return nil
}

// Required menolak isian kosong, lalu memeriksa panjangnya seperti Length.
func Required(field, label, value string, max int) []Violation {
	if value == "" {
		return []Violation{{Field: field, Message: label + " wajib diisi."}}
	}
	return Length(field, label, value, max)
}

// EmailLooksValid memeriksa bentuk alamat surel secara longgar: satu @, bagian lokal dan domain
// tidak kosong, dan domain memuat titik yang tidak di ujung.
func EmailLooksValid(address string) bool {
	address = strings.TrimSpace(address)
	i := strings.IndexByte(address, '@')
	if i <= 0 || i == len(address)-1 {
		return false
	}
	// Tidak boleh ada "@" kedua, dan bagian setelahnya harus memuat titik di tengah.
	domain := address[i+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	j := strings.IndexByte(domain, '.')
	return j > 0 && j < len(domain)-1
}
