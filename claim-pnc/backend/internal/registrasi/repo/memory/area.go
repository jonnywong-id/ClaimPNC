package memory

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// AreaDirectory adalah master wilayah kecil untuk uji dan untuk berjalan tanpa Oracle.
//
// Isinya satu rantai nyata dari layar Pega — DI YOGYAKARTA sampai KEL. CATURTUNGGAL —
// dengan kode yang sama dengan master POOLDATA, supaya layar yang diuji di memori
// memperlihatkan hal yang sama dengan yang akan dilihat petugas.
type AreaDirectory struct {
	option map[string][]registrasi.AreaOption
}

// NewAreaDirectory membentuk master wilayah contoh.
func NewAreaDirectory() *AreaDirectory {
	return &AreaDirectory{option: map[string][]registrasi.AreaOption{
		key(registrasi.AreaCountry, ""): {
			{ID: "100009", Name: registrasi.CountryIndonesia},
			{ID: "100001", Name: "TIMOR LESTE"},
		},
		key(registrasi.AreaProvince, registrasi.CountryIndonesia): {
			{ID: "10012", Name: "DI YOGYAKARTA"},
		},
		key(registrasi.AreaCity, "10012"): {
			{ID: "10259", Name: "KAB. SLEMAN"},
			{ID: "10138", Name: "KOTA SLEMAN"},
		},
		key(registrasi.AreaDistrict, "10259"): {
			{ID: "10000925", Name: "KEC. DEPOK"},
		},
		key(registrasi.AreaVillage, "10000925"): {
			{ID: "10004326", Name: "KEL. CATURTUNGGAL", PostalCode: "55281"},
		},
	}}
}

func key(level registrasi.AreaLevel, parent string) string {
	return string(level) + "|" + strings.ToUpper(strings.TrimSpace(parent))
}

// Options memenuhi seam AreaDirectory.
func (d *AreaDirectory) Options(_ context.Context, level registrasi.AreaLevel, parent string) ([]registrasi.AreaOption, error) {
	switch level {
	case registrasi.AreaCountry, registrasi.AreaProvince, registrasi.AreaCity,
		registrasi.AreaDistrict, registrasi.AreaVillage:
	default:
		return nil, fmt.Errorf("%w: tingkat wilayah %q", registrasi.ErrUnknownAreaLevel, level)
	}
	if level == registrasi.AreaCountry {
		parent = ""
	}
	out := append([]registrasi.AreaOption{}, d.option[key(level, parent)]...)
	return out, nil
}

var _ registrasi.AreaDirectory = (*AreaDirectory)(nil)
