package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// PICReader adalah daftar PIC Teknik di memori, dipakai pengembangan lokal dan pengujian.
//
// Adapter kedua di balik seam `TechnicalPICReader`. Tanpa adapter kedua, seam itu hipotetis
// dan daftar PIC tidak dapat diuji tanpa Oracle.
type PICReader struct {
	Rows []dashboardclaim.TechnicalPICRow

	// Business memetakan OperatorID ke TYPE_BUSINESS-nya.
	//
	// Ia TIDAK lagi menyaring — lihat `ListTechnicalPIC`. Dibiarkan ada karena uji memakainya
	// untuk memastikan penyaring itu memang sudah tidak berlaku.
	Business map[string]string
}

// NewPICReader membangun pembaca berisi contoh.
func NewPICReader() *PICReader {
	rows, business := SamplePIC()
	return &PICReader{Rows: rows, Business: business}
}

// ListTechnicalPIC menyaring, mengurutkan, lalu memotong satu halaman.
//
// `filter.BusinessType` SENGAJA diabaikan, karena `picteknik.sql` pun mengabaikannya. Adapter
// memori yang menyaring lebih ketat daripada SQL-nya menghasilkan kelas kekeliruan terburuk
// di seam ini: daftar tampak benar saat dicoba tanpa Oracle, lalu berperilaku lain di Oracle.
func (r *PICReader) ListTechnicalPIC(
	_ context.Context,
	filter dashboardclaim.TechnicalPICFilter,
) (dashboardclaim.TechnicalPICPage, error) {
	filter = filter.Normalize()

	cocok := make([]dashboardclaim.TechnicalPICRow, 0, len(r.Rows))
	for _, row := range r.Rows {
		if filter.Search != "" && !mengandung(row, filter.Search) {
			continue
		}
		cocok = append(cocok, row)
	}

	// Urutan disamakan dengan SQL-nya: beban paling sedikit lebih dulu, lalu nama. Fake yang
	// mengurutkan berbeda membuat uji halaman lulus di memori dan gagal di Oracle.
	sort.SliceStable(cocok, func(i, j int) bool {
		if cocok[i].Workload != cocok[j].Workload {
			return cocok[i].Workload < cocok[j].Workload
		}
		return cocok[i].Name < cocok[j].Name
	})

	total := len(cocok)
	if filter.Offset >= total {
		return dashboardclaim.TechnicalPICPage{Total: total}, nil
	}

	akhir := filter.Offset + filter.Limit
	if akhir > total {
		akhir = total
	}

	return dashboardclaim.TechnicalPICPage{Rows: cocok[filter.Offset:akhir], Total: total}, nil
}

// mengandung meniru pencarian SQL: OPERATOR_ID atau MCL_NAME, tanpa membedakan huruf.
func mengandung(row dashboardclaim.TechnicalPICRow, cari string) bool {
	cari = strings.ToUpper(cari)
	return strings.Contains(strings.ToUpper(row.OperatorID), cari) ||
		strings.Contains(strings.ToUpper(row.Name), cari)
}
