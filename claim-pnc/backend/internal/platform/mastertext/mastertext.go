// Package mastertext memuat aturan bersama master data berbentuk "satu keterangan + daftar
// nama bisnis": perapian isian dan penjaga panjangnya.
//
// Penjaga panjang itu BUKAN aturan bisnis melainkan penjaga teknis: nilai yang melampaui lebar
// kolom akan ditolak basis data dengan ORA-12899, galat yang muncul di layar sebagai 500.
// Panjang dihitung dalam rune, bukan byte, supaya huruf beraksen tidak membuat batasnya
// terasa berubah-ubah.
package mastertext

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"claim-pnc/internal/platform/validation"
)

// Clean merapikan keterangan dan membuang nama bisnis yang kosong.
func Clean(description string, businessNames []string) (string, []string) {
	names := make([]string, 0, len(businessNames))
	for _, name := range businessNames {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return strings.TrimSpace(description), names
}

// Limits adalah batas panjang beserta nama isian yang ditandai pelanggarannya.
type Limits struct {
	DescriptionField string
	DescriptionLabel string
	MaxDescription   int
	BusinessField    string
	MaxBusinessName  int
}

// CheckLengths memeriksa panjang keterangan dan nama bisnis. Nama bisnis yang terlalu panjang
// dilaporkan SEKALI — yang pertama ditemukan — dengan menyebut namanya.
func CheckLengths(description string, businessNames []string, l Limits) []validation.Violation {
	var violation []validation.Violation
	if utf8.RuneCountInString(description) > l.MaxDescription {
		violation = append(violation, validation.Violation{
			Field:   l.DescriptionField,
			Message: l.DescriptionLabel + " paling panjang " + strconv.Itoa(l.MaxDescription) + " karakter.",
		})
	}
	for _, name := range businessNames {
		if utf8.RuneCountInString(name) > l.MaxBusinessName {
			violation = append(violation, validation.Violation{
				Field:   l.BusinessField,
				Message: "Nama bisnis paling panjang " + strconv.Itoa(l.MaxBusinessName) + " karakter: " + name,
			})
			break
		}
	}
	return violation
}

// NormalizeBusinessName menyeragamkan nama bisnis untuk dicocokkan: huruf besar, tanpa spasi
// tepi.
func NormalizeBusinessName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
