package memory

import "claim-pnc/internal/masterpicteknik"

// DaftarContoh adalah baris awal untuk menjalankan aplikasi tanpa basis data.
//
// # Isinya karangan, dan itu disengaja
//
// Berbeda dari Master Status Klaim yang 33 barisnya adalah isi master yang sungguhan,
// isi MST_USER_TEKNIK **tidak ada di dalam export** — tidak ada CSV, tidak ada dump.
// Yang terbaca dari export hanyalah bentuk tabelnya.
//
// Karena itu baris di bawah adalah contoh yang dikarang, bukan data nyata: nama, surel,
// dan ID operatornya tidak merujuk pegawai mana pun. Mengarang data yang berpura-pura
// nyata justru yang dilarang `docs/AGENTS.md` aturan 5; yang dilakukan di sini adalah
// sebaliknya — contoh yang jelas-jelas contoh, memakai domain `example.invalid` yang
// memang dicadangkan supaya tidak mungkin tertukar dengan alamat sungguhan.
//
// Struktur grup dan atasannya dibuat menyerupai keadaan nyata secukupnya untuk menguji
// layar: ada atasan, ada bawahan, ada yang nonaktif, dan ada yang kuotanya nol.
func SampleList() []masterpicteknik.PICTeknik {
	return []masterpicteknik.PICTeknik{
		{
			OperatorID: "PICTEKNIK01",
			Name:       "Contoh Kepala Teknik",
			Email:      "contoh.kepalateknik@example.invalid",
			BusinessLine: "NONMBU",
			Group:       "TEKNIK JAKARTA",
			Quota:      20,
			ExternalQuota:  0,
			Active:      true,
		},
		{
			OperatorID: "PICTEKNIK02",
			Name:       "Contoh Adjuster Madya",
			Email:      "contoh.adjuster@example.invalid",
			BusinessLine: "NONMBU",
			Group:       "TEKNIK JAKARTA",
			Supervisor:     "PICTEKNIK01",
			Quota:      15,
			ExternalQuota:  3,
			Active:      true,
		},
		{
			OperatorID: "PICTEKNIK03",
			Name:       "Contoh Adjuster Muda",
			Email:      "contoh.adjustermuda@example.invalid",
			BusinessLine: "NONMBU",
			Group:       "TEKNIK SURABAYA",
			Supervisor:     "PICTEKNIK01",
			Quota:      10,
			ExternalQuota:  0,
			Active:      true,
		},
		{
			// Nonaktif: tidak menerima penugasan baru, tetapi tetap terbaca karena
			// klaim lama merujuknya. Inilah yang menggantikan penghapusan.
			OperatorID: "PICTEKNIK04",
			Name:       "Contoh Petugas Mutasi",
			Email:      "contoh.mutasi@example.invalid",
			BusinessLine: "NONMBU",
			Group:       "TEKNIK SURABAYA",
			Supervisor:     "PICTEKNIK01",
			Quota:      0,
			ExternalQuota:  0,
			Active:      false,
		},
	}
}

// DirektoriContoh adalah direktori operator untuk lingkungan tanpa basis data.
//
// Ia memuat keempat contoh di atas DITAMBAH satu ID yang belum terdaftar di master —
// supaya alur "tambah petugas baru yang sudah ada di direktori" dapat dicoba, dan alur
// "ID yang tidak dikenal ditolak" dapat dicoba pula dengan ID mana pun di luar daftar.
func SampleDirectory() map[string]string {
	return map[string]string{
		"PICTEKNIK01": "Contoh Kepala Teknik",
		"PICTEKNIK02": "Contoh Adjuster Madya",
		"PICTEKNIK03": "Contoh Adjuster Muda",
		"PICTEKNIK04": "Contoh Petugas Mutasi",
		"PICTEKNIK05": "Contoh Petugas Baru",
	}
}
