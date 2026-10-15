package memory

import (
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// SampleRows adalah baris contoh untuk pengembangan lokal dan uji.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun nomor polis, nama tertanggung, nomor klaim, maupun nama petugas di bawah
// berasal dari data nyata. `D-69` melarang menulis data nasabah ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian untuk "data contoh".
//
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "PT Contoh" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Sembilan baris, dipilih supaya setiap penyaring modul ini punya baris yang LOLOS dan
// baris yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan
// penyaringnya bekerja.
//
//	baris  membuktikan
//	-----  ----------------------------------------------------------------------------
//	1, 2   tab Non-MBU berisi penugasan pada unit organisasi AdminPNC
//	3, 4   tab PA berisi penugasan pada unit organisasi AdminPA
//	5      tab Travel berisi penugasan pada unit organisasi AdminTRAVEL
//	6      klaim SELESAI tidak muncul di tab mana pun
//	7      klaim DITOLAK tidak muncul di tab mana pun
//	8      berkas penerimaan dokumen (kelas berbeda) tidak bocor ke tab mana pun
//	9      penugasan pada unit organisasi LAIN tidak muncul di tab mana pun
//
// Baris 2 sengaja punya tanggal pendaftaran KOSONG. Ia membuktikan dua hal sekaligus:
// pengurutan tidak panik pada tanggal kosong, dan kolom Lama Waktu Klaim tampil kosong
// alih-alih berbunyi "0 minutes ago" untuk baris yang tanggalnya memang belum ada.
func SampleRows() []Row {
	// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
	// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
	// menyentuh kode.
	base := time.Date(2026, time.September, 26, 8, 0, 0, 0, time.UTC)

	at := func(days int) *time.Time {
		moment := base.AddDate(0, 0, -days)
		return &moment
	}

	return []Row{
		{
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitNonMBU,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:      "ASM-FW-GCNMFW-WORK PNC-9001",
				CaseID:         "PNC-9001",
				PolicyNumber:   "00.000.2026.00001",
				InsuredName:    "PT Contoh Sejahtera",
				BusinessName:   "Property All Risk",
				BusinessSource: "Broker Contoh",
				RegisteredAt:   at(3),
				AdminName:      "Admin Contoh Satu",
			},
		},
		{
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitNonMBU,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-9002",
				CaseID:       "PNC-9002",
				PolicyNumber: "00.000.2026.00002",
				InsuredName:  "PT Contoh Abadi",
				BusinessName: "Marine Cargo",

				// Sumber bisnis dan tanggal pendaftaran sengaja KOSONG: keduanya memang
				// dapat kosong pada klaim yang snapshot polisnya belum lengkap, dan layar
				// harus menampilkannya sebagai tanda pisah — bukan sebagai kolom yang
				// gagal dimuat.
				BusinessSource: "",
				RegisteredAt:   nil,

				AdminName: "Admin Contoh Satu",
			},
		},
		{
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitPA,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:      "ASM-FW-GCNMFW-WORK PNC-9101",
				CaseID:         "PNC-9101",
				PolicyNumber:   "00.002.2026.00011",
				InsuredName:    "Tertanggung Contoh A",
				BusinessName:   "Personal Accident",
				BusinessSource: "Keagenan Contoh",
				RegisteredAt:   at(1),
				AdminName:      "Admin Contoh Dua",
			},
		},
		{
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitPA,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:      "ASM-FW-GCNMFW-WORK PNC-9102",
				CaseID:         "PNC-9102",
				PolicyNumber:   "00.002.2026.00012",
				InsuredName:    "Tertanggung Contoh B",
				BusinessName:   "Personal Accident",
				BusinessSource: "Bancassurance Contoh",
				RegisteredAt:   at(400),
				AdminName:      "Admin Contoh Dua",
			},
		},
		{
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitTravel,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:      "ASM-FW-GCNMFW-WORK PNC-9201",
				CaseID:         "PNC-9201",
				PolicyNumber:   "00.005.2026.00021",
				InsuredName:    "Tertanggung Contoh C",
				BusinessName:   "Travel",
				BusinessSource: "Digital Contoh",
				RegisteredAt:   at(10),
				AdminName:      "Admin Contoh Tiga",
			},
		},
		{
			// Klaim SELESAI. Unit organisasinya Non-MBU, sehingga bila penyaring status
			// kerja terlewat ia akan muncul di tab pertama — dan uji akan menangkapnya.
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitNonMBU,
			WorkStatus: StatusCompleted,
			Item: inboxmanageradmin.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-9003",
				CaseID:       "PNC-9003",
				PolicyNumber: "00.000.2026.00003",
				InsuredName:  "PT Contoh Selesai",
				RegisteredAt: at(30),
				AdminName:    "Admin Contoh Satu",
			},
		},
		{
			// Klaim DITOLAK, dengan alasan yang sama seperti baris di atas.
			WorkClass:  WorkClassPNC,
			OrgUnit:    inboxmanageradmin.OrgUnitPA,
			WorkStatus: StatusRejected,
			Item: inboxmanageradmin.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-9103",
				CaseID:       "PNC-9103",
				PolicyNumber: "00.002.2026.00013",
				InsuredName:  "Tertanggung Contoh D",
				RegisteredAt: at(45),
				AdminName:    "Admin Contoh Dua",
			},
		},
		{
			// Berkas penerimaan dokumen — kelas objek kerja BERBEDA, unit organisasinya
			// sama. Ia membuktikan penyaring kelas benar-benar dipakai.
			WorkClass:  "ASM-FW-GCNMFW-Work-ReceiveDocument",
			OrgUnit:    inboxmanageradmin.OrgUnitNonMBU,
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK RCV-7001",
				CaseID:       "RCV-7001",
				PolicyNumber: "00.000.2026.00004",
				InsuredName:  "PT Contoh Dokumen",
				RegisteredAt: at(2),
				AdminName:    "Petugas Terima Contoh",
			},
		},
		{
			// Unit organisasi LAIN. Ia membuktikan ketiga tab tidak menampilkan penugasan
			// unit mana pun di luar ketiganya — termasuk unit yang namanya mirip.
			WorkClass:  WorkClassPNC,
			OrgUnit:    "AdminLain",
			WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-9301",
				CaseID:       "PNC-9301",
				PolicyNumber: "00.000.2026.00005",
				InsuredName:  "PT Contoh Unit Lain",
				RegisteredAt: at(5),
				AdminName:    "Admin Contoh Empat",
			},
		},
	}
}
