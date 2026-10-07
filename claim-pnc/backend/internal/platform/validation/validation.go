// Package validation memuat bentuk pelanggaran isian yang dipakai bersama modul domain.
//
// Setiap modul tetap memiliki tipe ValidationError-nya sendiri — ia bagian dari API domain
// modul dan pesan galatnya berawalan nama modul. Yang dipusatkan di sini hanyalah satu
// pelanggaran (Violation) dan cara pelanggaran-pelanggaran itu dirangkai menjadi teks galat.
package validation

import "strings"

// Violation adalah satu isian yang tidak lolos aturan beserta pesannya untuk pengguna.
type Violation struct {
	Field   string
	Message string
}

// Format merangkai pelanggaran menjadi teks galat:
//
//	prefix + field₁ + fieldSep + message₁ + joinSep + field₂ + … + suffix
func Format(list []Violation, prefix, fieldSep, joinSep, suffix string) string {
	parts := make([]string, 0, len(list))
	for _, v := range list {
		parts = append(parts, v.Field+fieldSep+v.Message)
	}
	return prefix + strings.Join(parts, joinSep) + suffix
}
