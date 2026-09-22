package memory

import "claim-pnc/internal/masterpicteknik"

// SampleList adalah baris awal untuk menjalankan aplikasi tanpa basis data.
//
// # Isinya karangan, dan itu dinyatakan terang-terangan
//
// Berbeda dari Master Status Klaim yang 33 barisnya adalah isi master sungguhan, isi
// MST_USER_TEKNIK **tidak ada di dalam export** — tidak ada CSV, tidak ada dump. Yang
// terbaca dari export hanyalah bentuk tabelnya.
//
// Karena itu baris di bawah adalah contoh yang dikarang: nama, surel, dan ID operatornya
// tidak merujuk pegawai mana pun. Mengarang data yang BERPURA-PURA nyata justru yang
// dilarang; yang dilakukan di sini sebaliknya — contoh yang jelas-jelas contoh, memakai
// domain `example.invalid` yang memang dicadangkan supaya tidak mungkin tertukar dengan
// alamat sungguhan.
//
// # Kenapa susunannya seperti ini
//
// Keempat baris dipilih supaya setiap jalur layar dapat dicoba tanpa basis data:
//
//	PICTEKNIK01  atasan, kuota besar          — muncul di daftar
//	PICTEKNIK02  bawahan, beban di bawah kuota — muncul di daftar
//	PICTEKNIK03  beban SAMA DENGAN kuota       — menguji tampilan "kuota penuh"
//	PICTEKNIK04  NONAKTIF                      — TIDAK muncul di daftar, tetapi dapat
//	                                             dibuka lewat ID dan diaktifkan kembali
//
// Baris keempat itulah yang membuat keputusan "daftar hanya menampilkan yang aktif" dapat
// diuji sungguhan, bukan hanya dipercaya.
func SampleList() []masterpicteknik.Technician {
	return []masterpicteknik.Technician{
		{
			OperatorID:   "PICTEKNIK01",
			Name:         "Contoh Kepala Teknik",
			Email:        "contoh.kepalateknik@example.invalid",
			BusinessLine: "NONMBU",
			Group:        "TEKNIK JAKARTA",
			Quota:        20,
			Workload:     6,
			Active:       true,
		},
		{
			OperatorID:   "PICTEKNIK02",
			Name:         "Contoh Adjuster Madya",
			Email:        "contoh.adjuster@example.invalid",
			BusinessLine: "NONMBU",
			Group:        "TEKNIK JAKARTA",
			Supervisor:   "PICTEKNIK01",
			Quota:        15,
			Workload:     9,
			Active:       true,
		},
		{
			OperatorID:    "PICTEKNIK03",
			Name:          "Contoh Petugas Teknik",
			Email:         "contoh.petugas@example.invalid",
			BusinessLine:  "NONMBU",
			Group:         "TEKNIK SURABAYA",
			Supervisor:    "PICTEKNIK01",
			Quota:         10,
			ExternalQuota: 3,
			Workload:      10,
			Active:        true,
		},
		{
			OperatorID:   "PICTEKNIK04",
			Name:         "Contoh Petugas Nonaktif",
			Email:        "contoh.nonaktif@example.invalid",
			BusinessLine: "NONMBU",
			Group:        "TEKNIK SURABAYA",
			Supervisor:   "PICTEKNIK01",
			Quota:        0,
			Workload:     0,
			Active:       false,
		},
	}
}
