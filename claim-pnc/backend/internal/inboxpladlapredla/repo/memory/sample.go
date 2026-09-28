package memory

import (
	"time"

	"claim-pnc/internal/inboxpladlapredla"
)

// pegaWorkKeyPrefix adalah awalan kunci objek kerja Pega pada `T_CLAIM_PNC.CLAIMID`.
//
// Nama kelas internal Pega tertanam di dalam kunci data bisnis — utang teknis §4.1 yang
// `D-22` dan `D-71` hapus untuk klaim baru. Ia ditiru di data contoh karena pencarian di
// layar ini mencocokkan KUNCI itu, bukan nomor klaim: data contoh yang kuncinya berbentuk
// lain akan membuat uji pencarian lolos karena alasan yang salah.
//
// Perhatikan SPASI di ujungnya. Ia bagian dari nilainya.
const pegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// workKey menyusun kunci objek kerja dari nomor klaim.
func workKey(claimNo string) string { return pegaWorkKeyPrefix + claimNo }

// day membentuk tanggal kalender polos, tanpa jam.
//
// UTC, bukan WIB — sama dengan Query.From dan dengan kolom tanggal di basis data lama,
// yang disimpan tanpa zona waktu. Lihat calendarDate pada query.go.
func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

// NewSampleStore membentuk penyimpanan berisi data contoh.
//
// # Janji yang dipegang sepuluh klaim di bawah
//
// SETIAP penyaring punya baris yang cocok MAUPUN yang tidak. Yang membuktikan sebuah
// penyaring bekerja bukan baris yang muncul, melainkan baris yang seharusnya TIDAK muncul
// dan memang tidak muncul — dan di layar ini ada tujuh penyaring yang berbeda-beda antar
// tab.
//
// Dua klaim punya tugas tambahan, dan keduanya menjaga kejanggalan yang sengaja dibawa
// dari Pega tetap tertiru:
//
//	PNC-1008  seluruh PLA-nya ber-ISKIRIM='0' -> MUNCUL di daftar, tanggal advice KOSONG
//	PNC-1006  cabangnya kosong (NULL)        -> TERSARING KELUAR dari tab DLA
//
// Tanpa keduanya, kedua kejanggalan itu dapat "diperbaiki" tanpa satu pun uji gagal.
func NewSampleStore() *Store {
	store := NewStore()
	store.Seed(sampleClaims(), sampleAdvices())
	return store
}

// sampleClaims adalah sepuluh baris `T_CLAIM_PNC` contoh.
//
// Nama tertanggung dan nomor polis di sini KARANGAN. Data nasabah tidak pernah disalin ke
// dalam berkas yang di-commit (`D-69`), dan yang dibutuhkan uji hanyalah nilai yang dapat
// dibedakan satu sama lain.
func sampleClaims() []Claim {
	return []Claim{
		{
			// Lengkap: punya PLA, DLA, dan Pre-DLA yang ketiganya memenuhi syarat.
			Key: workKey("PNC-1001"), No: "PNC-1001", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0001", Insured: "PT Contoh Satu",
			RegisterDate: day(2026, time.January, 5),
			LossDate:     day(2026, time.January, 2),
			PICTeknik:    "BUDI", GroupPanel: "003",
			BranchName: "JAKARTA", BusinessGroupID: "10001",
		},
		{
			// PLA-nya belum menunjuk reasuradur -> TIDAK masuk tab PLA.
			// DLA-nya sudah terkirim            -> TIDAK masuk tab DLA.
			Key: workKey("PNC-1002"), No: "PNC-1002", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0002", Insured: "PT Contoh Dua",
			RegisterDate: day(2026, time.January, 6),
			LossDate:     day(2026, time.January, 3),
			PICTeknik:    "SITI", GroupPanel: "004",
			BranchName: "SURABAYA", BusinessGroupID: "10002",
		},
		{
			// Personal Accident. Dikecualikan KETIGA tab meski dokumennya lengkap.
			Key: workKey("PNC-1003"), No: "PNC-1003", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0003", Insured: "PT Contoh Tiga",
			RegisterDate: day(2026, time.January, 7),
			LossDate:     day(2026, time.January, 4),
			PICTeknik:    "BUDI", GroupPanel: "002",
			BranchName: "JAKARTA", BusinessGroupID: "10001",
		},
		{
			// Travel. Dikecualikan ketiga tab.
			Key: workKey("PNC-1004"), No: "PNC-1004", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0004", Insured: "PT Contoh Empat",
			RegisterDate: day(2026, time.January, 8),
			LossDate:     day(2026, time.January, 5),
			PICTeknik:    "SITI", GroupPanel: "005",
			BranchName: "JAKARTA", BusinessGroupID: "10001",
		},
		{
			// Cabang ASNET. Masuk tab PLA dan Pre DLA, TIDAK masuk tab DLA.
			Key: workKey("PNC-1005"), No: "PNC-1005", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0005", Insured: "PT Contoh Lima",
			RegisterDate: day(2026, time.February, 2),
			LossDate:     day(2026, time.January, 20),
			PICTeknik:    "AGUS", GroupPanel: "006",
			BranchName: "ASNET", BusinessGroupID: "10003",
		},
		{
			// Cabang KOSONG — mewakili NULL di Oracle.
			//
			// Ia TIDAK masuk tab DLA, karena `NULL <> 'ASNET'` menghasilkan UNKNOWN,
			// bukan TRUE. Itu perilaku Pega, dan klaim inilah yang menjaganya tertiru.
			Key: workKey("PNC-1006"), No: "PNC-1006", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0006", Insured: "PT Contoh Enam",
			RegisterDate: day(2026, time.February, 3),
			LossDate:     day(2026, time.January, 21),
			PICTeknik:    "AGUS", GroupPanel: "009",
			BranchName: "", BusinessGroupID: "10004",
		},
		{
			// Kelompok bisnis 10008. Dikecualikan tab DLA saja.
			Key: workKey("PNC-1007"), No: "PNC-1007", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0007", Insured: "PT Contoh Tujuh",
			RegisterDate: day(2026, time.February, 4),
			LossDate:     day(2026, time.January, 22),
			PICTeknik:    "RINA", GroupPanel: "003",
			BranchName: "BANDUNG", BusinessGroupID: "10008",
		},
		{
			// SELURUH PLA-nya ber-ISKIRIM='0'.
			//
			// Ia MUNCUL di tab PLA — penyaring keanggotaan menerima '0' — tetapi kolom
			// "Tanggal PLA"-nya KOSONG, karena sub-kueri tanggalnya hanya menerima NULL.
			// Kejanggalan Pega yang sengaja dibawa (`P-5`).
			Key: workKey("PNC-1008"), No: "PNC-1008", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0008", Insured: "PT Contoh Delapan",
			RegisterDate: day(2026, time.February, 5),
			LossDate:     day(2026, time.January, 23),
			PICTeknik:    "RINA", GroupPanel: "003",
			BranchName: "MEDAN", BusinessGroupID: "10005",
		},
		{
			// Pre-DLA-nya sudah ber-Nomor Akseptasi -> TIDAK masuk tab Pre DLA.
			Key: workKey("PNC-1009"), No: "PNC-1009", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0009", Insured: "PT Contoh Sembilan",
			RegisterDate: day(2026, time.February, 6),
			LossDate:     day(2026, time.January, 24),
			PICTeknik:    "BUDI", GroupPanel: "003",
			BranchName: "SEMARANG", BusinessGroupID: "10006",
		},
		{
			// Tanpa dokumen sama sekali.
			//
			// Ia tidak muncul di tab mana pun, tetapi ADA — sehingga permintaan
			// rinciannya dijawab daftar kosong, bukan "tidak ditemukan". Itulah satu
			// keadaan yang membedakan kedua jawaban itu.
			Key: workKey("PNC-1010"), No: "PNC-1010", BusinessName: "Marine Cargo",
			PolicyNo: "POL-2026-0010", Insured: "PT Contoh Sepuluh",
			RegisterDate: day(2026, time.February, 7),
			LossDate:     day(2026, time.January, 25),
			PICTeknik:    "SITI", GroupPanel: "003",
			BranchName: "DENPASAR", BusinessGroupID: "10007",
		},
	}
}

// sampleAdvices adalah baris dokumen contoh untuk ketiga tabel.
func sampleAdvices() []Advice {
	pla := inboxpladlapredla.KindPLA
	dla := inboxpladlapredla.KindDLA
	pre := inboxpladlapredla.KindPreDLA

	return []Advice{
		// PNC-1001 — lengkap di ketiga tab.
		{
			ClaimKey: workKey("PNC-1001"), Kind: pla,
			No: "PLA/2026/0001", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Revision: "0", Date: day(2026, time.January, 10),
			Sent: "", ReinsCode: "R001", Notes: "Estimasi awal",
			// Alamat dan negara diisi supaya tombol "SEND" dapat dijalankan sampai
			// tuntas di lingkungan pengembangan. Negaranya INDONESIA, sehingga
			// suratnya berbahasa Indonesia.
			Email:          "reas-a@contoh.example",
			ReinsurerLogin: "REASA", ReinsurerCountry: "INDONESIA",
		},
		{
			ClaimKey: workKey("PNC-1001"), Kind: pla,
			No: "PLA/2026/0002", Reinsurer: "Reasuransi Contoh B", Type: "ORS",
			Revision: "1", Date: day(2026, time.January, 12),
			Sent: "1", SentDate: day(2026, time.January, 13),
			ReceivedDate: day(2026, time.January, 14),
			ReinsCode:    "R002", Email: "reas-b@contoh.example",
			Notes: "Revisi pertama, sudah dikirim",
		},
		{
			ClaimKey: workKey("PNC-1001"), Kind: dla,
			No: "DLA/2026/0001", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Date: day(2026, time.February, 1), Sent: "",
			AcceptanceNo: "AKS-2026-0001", ReinsCode: "R001",
			Notes: "Nilai akseptasi final",
			// Negaranya BUKAN Indonesia -> suratnya berbahasa Inggris. Pasangan ini
			// sengaja ada supaya kedua bahasa terwakili di data contoh.
			Email:          "reas-a-sg@contoh.example",
			ReinsurerLogin: "REASASG", ReinsurerCountry: "SINGAPORE",
		},
		{
			ClaimKey: workKey("PNC-1001"), Kind: pre,
			No: "PRE/2026/0001", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Date: day(2026, time.January, 25), Sent: "",
			AcceptanceNo: "", ReinsCode: "R001",
			// Lampirannya sudah ada -> baris ini MUNCUL di panel "Print Pre DLA".
			AttachmentKey: "ATT-PNC-1001-0001",
		},
		{
			// Pre-DLA kedua pada klaim yang SAMA, tetapi TANPA lampiran.
			//
			// Ia muncul di tab Pre DLA — penyaring tabnya `NOAKSEP IS NULL`, dan kolom
			// itu kosong di sini — tetapi TIDAK muncul di panel "Print Pre DLA", karena
			// gabungan ke tabel lampiran menyingkirkannya.
			//
			// Pasangan ini sengaja ada pada satu klaim: tanpanya, selisih jumlah baris
			// antara tab dan panelnya tidak pernah terwakili di data contoh, dan
			// penyaring lampiran dapat dihapus tanpa satu pun uji gagal.
			ClaimKey: workKey("PNC-1001"), Kind: pre,
			No: "PRE/2026/0003", Reinsurer: "Reasuransi Contoh B", Type: "ORS",
			Date: day(2026, time.January, 27), Sent: "",
			AcceptanceNo: "", ReinsCode: "R002",
			AttachmentKey: "",
		},

		// PNC-1002 — dua penolakan sekaligus.
		{
			// REINSCODE kosong -> tidak lolos penyaring tab PLA.
			ClaimKey: workKey("PNC-1002"), Kind: pla,
			No: "PLA/2026/0003", Reinsurer: "", Type: "OR",
			Revision: "0", Date: day(2026, time.January, 11),
			Sent: "", ReinsCode: "",
		},
		{
			// Sudah terkirim -> tidak lolos penyaring tab DLA.
			ClaimKey: workKey("PNC-1002"), Kind: dla,
			No: "DLA/2026/0002", Reinsurer: "Reasuransi Contoh C", Type: "OR",
			Date: day(2026, time.February, 2), Sent: "1",
			SentDate: day(2026, time.February, 3), ReinsCode: "R003",
			Email: "reas-c@contoh.example",
		},

		// PNC-1003 dan PNC-1004 — dokumennya memenuhi syarat, klaimnya yang
		// dikecualikan. Keduanya ada supaya penyaring lini bisnis benar-benar teruji:
		// tanpa dokumen, klaim ini tidak akan muncul walaupun penyaringnya dilepas.
		{
			ClaimKey: workKey("PNC-1003"), Kind: pla,
			No: "PLA/2026/0004", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Revision: "0", Date: day(2026, time.January, 15),
			Sent: "", ReinsCode: "R001",
		},
		{
			ClaimKey: workKey("PNC-1004"), Kind: dla,
			No: "DLA/2026/0003", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Date: day(2026, time.January, 16), Sent: "", ReinsCode: "R001",
		},

		// PNC-1005 (ASNET), PNC-1006 (cabang kosong), PNC-1007 (bisnis 10008).
		// Ketiganya punya PLA dan DLA yang memenuhi syarat dokumen, sehingga yang
		// menentukan munculnya hanyalah penyaring di sisi klaim.
		{
			ClaimKey: workKey("PNC-1005"), Kind: pla,
			No: "PLA/2026/0005", Reinsurer: "Reasuransi Contoh D", Type: "OR",
			Revision: "0", Date: day(2026, time.February, 10),
			Sent: "", ReinsCode: "R004",
		},
		{
			ClaimKey: workKey("PNC-1005"), Kind: dla,
			No: "DLA/2026/0004", Reinsurer: "Reasuransi Contoh D", Type: "OR",
			Date: day(2026, time.February, 11), Sent: "", ReinsCode: "R004",
		},
		{
			ClaimKey: workKey("PNC-1006"), Kind: pla,
			No: "PLA/2026/0006", Reinsurer: "Reasuransi Contoh D", Type: "OR",
			Revision: "0", Date: day(2026, time.February, 12),
			Sent: "", ReinsCode: "R004",
		},
		{
			ClaimKey: workKey("PNC-1006"), Kind: dla,
			No: "DLA/2026/0005", Reinsurer: "Reasuransi Contoh D", Type: "OR",
			Date: day(2026, time.February, 13), Sent: "", ReinsCode: "R004",
		},
		{
			ClaimKey: workKey("PNC-1007"), Kind: pla,
			No: "PLA/2026/0007", Reinsurer: "Reasuransi Contoh E", Type: "OR",
			Revision: "0", Date: day(2026, time.February, 14),
			Sent: "", ReinsCode: "R005",
		},
		{
			ClaimKey: workKey("PNC-1007"), Kind: dla,
			No: "DLA/2026/0006", Reinsurer: "Reasuransi Contoh E", Type: "OR",
			Date: day(2026, time.February, 15), Sent: "", ReinsCode: "R005",
		},

		// PNC-1008 — SELURUH PLA-nya ber-ISKIRIM='0'.
		//
		// Dua baris, bukan satu, supaya jelas bahwa yang mengosongkan kolom tanggal
		// bukanlah ketiadaan dokumen melainkan nilai `'0'` itu sendiri.
		{
			ClaimKey: workKey("PNC-1008"), Kind: pla,
			No: "PLA/2026/0008", Reinsurer: "Reasuransi Contoh F", Type: "OR",
			Revision: "0", Date: day(2026, time.February, 16),
			Sent: "0", ReinsCode: "R006",
		},
		{
			ClaimKey: workKey("PNC-1008"), Kind: pla,
			No: "PLA/2026/0009", Reinsurer: "Reasuransi Contoh F", Type: "ORS",
			Revision: "1", Date: day(2026, time.February, 17),
			Sent: "0", ReinsCode: "R006",
		},

		// PNC-1009 — Pre-DLA sudah ber-Nomor Akseptasi, sehingga keluar dari antrean.
		{
			ClaimKey: workKey("PNC-1009"), Kind: pre,
			No: "PRE/2026/0002", Reinsurer: "Reasuransi Contoh A", Type: "OR",
			Date: day(2026, time.February, 18), Sent: "",
			AcceptanceNo: "AKS-2026-0002", ReinsCode: "R001",
			// Sudah ber-Nomor Akseptasi -> keluar dari tab Pre DLA. Lampirannya tetap
			// diisi supaya terlihat bahwa yang menyingkirkannya adalah akseptasinya,
			// bukan ketiadaan lampiran.
			AttachmentKey: "ATT-PNC-1009-0001",
		},
	}
}
