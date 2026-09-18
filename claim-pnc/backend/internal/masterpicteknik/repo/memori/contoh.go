package memori

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
func DaftarContoh() []masterpicteknik.PICTeknik {
	return []masterpicteknik.PICTeknik{
		{
			IDOperator: "PICTEKNIK01",
			Nama:       "Contoh Kepala Teknik",
			Email:      "contoh.kepalateknik@example.invalid",
			LiniBisnis: "NONMBU",
			Grup:       "TEKNIK JAKARTA",
			Kuota:      20,
			KuotaLuar:  0,
			Aktif:      true,
		},
		{
			IDOperator: "PICTEKNIK02",
			Nama:       "Contoh Adjuster Madya",
			Email:      "contoh.adjuster@example.invalid",
			LiniBisnis: "NONMBU",
			Grup:       "TEKNIK JAKARTA",
			Atasan:     "PICTEKNIK01",
			Kuota:      15,
			KuotaLuar:  3,
			Aktif:      true,
		},
		{
			IDOperator: "PICTEKNIK03",
			Nama:       "Contoh Adjuster Muda",
			Email:      "contoh.adjustermuda@example.invalid",
			LiniBisnis: "NONMBU",
			Grup:       "TEKNIK SURABAYA",
			Atasan:     "PICTEKNIK01",
			Kuota:      10,
			KuotaLuar:  0,
			Aktif:      true,
		},
		{
			// Nonaktif: tidak menerima penugasan baru, tetapi tetap terbaca karena
			// klaim lama merujuknya. Inilah yang menggantikan penghapusan.
			IDOperator: "PICTEKNIK04",
			Nama:       "Contoh Petugas Mutasi",
			Email:      "contoh.mutasi@example.invalid",
			LiniBisnis: "NONMBU",
			Grup:       "TEKNIK SURABAYA",
			Atasan:     "PICTEKNIK01",
			Kuota:      0,
			KuotaLuar:  0,
			Aktif:      false,
		},
	}
}

// DirektoriContoh adalah direktori operator untuk lingkungan tanpa basis data.
//
// Ia memuat keempat contoh di atas DITAMBAH satu ID yang belum terdaftar di master —
// supaya alur "tambah petugas baru yang sudah ada di direktori" dapat dicoba, dan alur
// "ID yang tidak dikenal ditolak" dapat dicoba pula dengan ID mana pun di luar daftar.
func DirektoriContoh() map[string]string {
	return map[string]string{
		"PICTEKNIK01": "Contoh Kepala Teknik",
		"PICTEKNIK02": "Contoh Adjuster Madya",
		"PICTEKNIK03": "Contoh Adjuster Muda",
		"PICTEKNIK04": "Contoh Petugas Mutasi",
		"PICTEKNIK05": "Contoh Petugas Baru",
	}
}
