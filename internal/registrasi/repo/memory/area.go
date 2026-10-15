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

	// Causes adalah pilihan Penyebab Kerugian per kode bisnis polis. Isinya karangan.
	Causes map[string][]registrasi.CauseOfLossOption
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
	}, Causes: map[string][]registrasi.CauseOfLossOption{
		"10013": {
			{ID: "11997", Name: "FIRE - OPEN FLAME"},
			{ID: "12010", Name: "FIRE - SHORT CIRCUIT"},
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

// CauseOfLossOptions memenuhi seam CauseOfLossDirectory.
func (d *AreaDirectory) CauseOfLossOptions(_ context.Context, businessCode string) ([]registrasi.CauseOfLossOption, error) {
	code := strings.TrimSpace(businessCode)
	if code == "" {
		return []registrasi.CauseOfLossOption{}, nil
	}
	return append([]registrasi.CauseOfLossOption{}, d.Causes[code]...), nil
}

var (
	_ registrasi.AreaDirectory        = (*AreaDirectory)(nil)
	_ registrasi.CauseOfLossDirectory = (*AreaDirectory)(nil)
)
