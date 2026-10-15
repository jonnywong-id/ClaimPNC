package directory

import (
	"context"
	"strings"

	"claim-pnc/internal/masterpicteknik"
)

// Fake adalah direktori pegawai di dalam memori.
//
// Ia adapter kedua yang membuat seam EmployeeDirectory nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3) — dan tanpa itu seluruh jalur simpan tidak
// dapat diuji sama sekali, karena setiap penyimpanan menuntut pencarian nama lebih dulu.
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle. Itu membuat layar Master PIC
// Teknik dapat dicoba utuh — termasuk penolakan "ID operator tidak terdaftar", yang justru
// jalur paling mudah terlewat bila hanya dicoba dengan ID yang benar.
type Fake struct {
	rows map[string]masterpicteknik.Employee
	err  error
}

// NewFake membentuk direktori berisi pegawai yang diberikan.
func NewFake(list ...masterpicteknik.Employee) *Fake {
	f := &Fake{rows: make(map[string]masterpicteknik.Employee, len(list))}
	for _, employee := range list {
		f.rows[masterpicteknik.IDKey(employee.OperatorID)] = employee
	}
	return f
}

// SetError membuat direktori menjawab dengan galat, untuk menguji jalur gagal.
func (f *Fake) SetError(err error) { f.err = err }

// Lookup mencari pegawai. portalAlias diabaikan: direktori tiruan tidak berbeda antar
// entitas, dan membedakannya hanya akan menambah bahan pengujian tanpa menambah jaminan.
func (f *Fake) Lookup(_ context.Context, _, operatorID string) (masterpicteknik.Employee, error) {
	if f.err != nil {
		return masterpicteknik.Employee{}, f.err
	}
	if strings.TrimSpace(operatorID) == "" {
		return masterpicteknik.Employee{}, masterpicteknik.ErrEmployeeUnknown
	}

	employee, existing := f.rows[masterpicteknik.IDKey(operatorID)]
	if !existing {
		return masterpicteknik.Employee{}, masterpicteknik.ErrEmployeeUnknown
	}
	return employee, nil
}

// SampleEmployees adalah direktori tiruan yang berpasangan dengan repo/memory.SampleList.
//
// Keempat ID-nya sengaja sama dengan yang ada di contoh master, ditambah DUA yang BELUM
// terdaftar sebagai PIC — tanpa itu, jalur "tambah petugas baru" tidak dapat dicoba sama
// sekali, karena setiap ID yang dikenal direktori sudah ada di master.
//
// Namanya dikarang dan memakai domain `example.invalid` yang memang dicadangkan, sehingga
// tidak mungkin tertukar dengan pegawai sungguhan.
func SampleEmployees() []masterpicteknik.Employee {
	return []masterpicteknik.Employee{
		{
			OperatorID: "PICTEKNIK01",
			Name:       "Contoh Kepala Teknik",
			Email:      "contoh.kepalateknik@example.invalid",
		},
		{
			OperatorID:     "PICTEKNIK02",
			Name:           "Contoh Adjuster Madya",
			Email:          "contoh.adjuster@example.invalid",
			SupervisorID:   "PICTEKNIK01",
			SupervisorName: "Contoh Kepala Teknik",
		},
		{
			OperatorID:     "PICTEKNIK03",
			Name:           "Contoh Petugas Teknik",
			Email:          "contoh.petugas@example.invalid",
			SupervisorID:   "PICTEKNIK01",
			SupervisorName: "Contoh Kepala Teknik",
		},
		{
			OperatorID:     "PICTEKNIK04",
			Name:           "Contoh Petugas Nonaktif",
			Email:          "contoh.nonaktif@example.invalid",
			SupervisorID:   "PICTEKNIK01",
			SupervisorName: "Contoh Kepala Teknik",
		},
		{
			OperatorID:     "PICTEKNIK05",
			Name:           "Contoh Petugas Baru",
			Email:          "contoh.baru@example.invalid",
			SupervisorID:   "PICTEKNIK01",
			SupervisorName: "Contoh Kepala Teknik",
		},
		{
			OperatorID:     "PICTEKNIK06",
			Name:           "Contoh Surveyor Lapangan",
			Email:          "contoh.surveyor@example.invalid",
			SupervisorID:   "PICTEKNIK02",
			SupervisorName: "Contoh Adjuster Madya",
		},
	}
}

var _ masterpicteknik.EmployeeDirectory = (*Fake)(nil)
