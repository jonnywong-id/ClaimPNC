package memory

import "claim-pnc/internal/dashboardclaim"

// SamplePIC adalah daftar PIC Teknik **fiktif** untuk pengembangan lokal.
//
// Nama dan Operator ID di sini dikarang. Aturan penulisan `D-69` melarang menyalin data
// nasabah; nama petugas bukan data nasabah, tetapi daftar contoh tidak perlu memakai nama
// orang sungguhan untuk membuktikan layarnya bekerja.
func SamplePIC() ([]dashboardclaim.TechnicalPICRow, map[string]string) {
	rows := []dashboardclaim.TechnicalPICRow{
		{OperatorID: "PICCONTOH01", Name: "PIC Contoh Satu", Email: "pic01@contoh.invalid", TeamGroup: "Tim A", Workload: 3},
		{OperatorID: "PICCONTOH02", Name: "PIC Contoh Dua", Email: "pic02@contoh.invalid", TeamGroup: "Tim A", Workload: 7},
		{OperatorID: "PICCONTOH03", Name: "PIC Contoh Tiga", Email: "pic03@contoh.invalid", TeamGroup: "Tim B", Workload: 1},
		{OperatorID: "PICCONTOH04", Name: "PIC Contoh Empat", Email: "pic04@contoh.invalid", TeamGroup: "Tim B", Workload: 5},
	}

	// Dua lini bisnis supaya penyaringnya benar-benar teruji menyaring.
	business := map[string]string{
		"PICCONTOH01": "NONMBU",
		"PICCONTOH02": "NONMBU",
		"PICCONTOH03": "PA",
		"PICCONTOH04": "NONMBU",
	}

	return rows, business
}
