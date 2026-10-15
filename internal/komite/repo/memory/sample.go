package memory

import (
	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// SampleThresholds mengembalikan isi master ambang komite sebagaimana yang berlaku hari
// ini.
//
// # Dari mana angkanya
//
// Diturunkan baris per baris dari `Database/emailkomite.csv` — berkas yang diserahkan
// Work Owner. Ia BUKAN data karangan: setiap batas, jenjang, dan penanda status di bawah
// ada di berkas itu, dan itulah yang membuat ketujuh kasus jumlah penyetuju pada
// `docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md` dapat diuji tanpa basis data
// sama sekali.
//
// # Isinya 29 baris, bukan 30 — dan kenapa itu sempat salah dibaca
//
// Berkasnya 31 baris fisik: satu judul dan 30 baris isi. Tetapi salah satu record
// membentang di DUA baris fisik, karena OPERATOR_ID pada baris ID 4 diakhiri BARIS BARU
// di dalam tanda kutip. Dibaca per baris, record itu terbelah dan menghasilkan dua baris
// palsu — sempat memunculkan "TYPE_BUSINESS" bernilai `NJOMANSUDARTHA` dan `1`, yang
// keduanya sebenarnya pecahan kolom lain.
//
// Dibaca dengan pengurai CSV yang benar, isinya **29 record**, dan **15 di antaranya
// jenjang persetujuan aktif** — tepat sebanyak yang ada di bawah.
//
// # Yang dibawa ke sini: seluruh jenjang, sebagian yang bukan
//
// Kelima belas baris jenjang persetujuan dibawa SELURUHNYA — itu yang menentukan hasil
// perhitungan. Dari 14 baris sisanya, hanya EMPAT yang dibawa sebagai contoh: baris tidak
// aktif dan baris pemberitahuan registrasi, masing-masing dua, supaya pengujian dapat
// membuktikan keduanya memang tidak pernah ikut menyetujui. Sisanya tidak menambah satu
// pun kasus uji yang belum tercakup.
//
// # Yang sengaja TIDAK dibawa: alamat surel
//
// Master aslinya memuat kolom EMAIL dan CC. Keduanya tidak dibaca modul ini dan tidak
// disalin ke sini, karena dua alasan yang saling menguatkan:
//
//   - Modul ini menghitung SIAPA yang menyetujui, bukan ke mana pemberitahuan dikirim;
//     yang terakhir adalah `S-3`.
//   - `D-69` mewajibkan alamat surel disamarkan di seluruh artefak yang di-commit, dan
//     `D-67` menetapkan alamat pribadi pada master lama — sekurang-kurangnya enam akun
//     Gmail di jalur produksi — TIDAK dibawa ke sistem baru sama sekali.
//
// Menyalin kolomnya ke berkas yang akan di-commit hanya akan memindahkan masalah yang
// sudah diputuskan untuk dihapus.
//
// Nama orang dan Operator ID ditulis lengkap, dan itu memang dibolehkan `D-69`: tanpa
// keduanya, hasil penjenjangan tidak dapat ditelusuri kembali ke baris masternya.
func SampleThresholds() []komite.Threshold {
	rp := money.FromRupiah

	return []komite.Threshold{
		// ── Non-MBU, pita 1 — klaim sampai Rp 100.000.000 ────────────────────────────
		//
		// Perhatikan kedua baris ini ber-DEGREE SAMA (1). Itu bukan salah salin: di
		// master pun demikian, dan itulah sebab penanda AmbiguousOrder ada.
		{
			ID: "7", Name: "ELLENSUPRIYATI", OperatorID: "ELLENSUPRIYATI",
			BusinessLine: "NONMBU", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			ID: "1", Name: "INDRA", OperatorID: "INDRAGUNAWAN",
			BusinessLine: "NONMBU", CommitteeType: "1",
			LowerBound: rp(50_000_001), UpperBound: rp(100_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},

		// ── Non-MBU, pita 2 — klaim di atas Rp 100.000.000 ───────────────────────────
		{
			ID: "2", Name: "BAMBANG", OperatorID: "BAMBANGSETIADJIGUNAWAN",
			BusinessLine: "NONMBU", CommitteeType: "2",
			LowerBound: rp(100_000_001), UpperBound: rp(500_000_000), Tier: 2,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			ID: "3", Name: "DANIELLISWANDI", OperatorID: "DANIELLISWANDI",
			BusinessLine: "NONMBU", CommitteeType: "2",
			LowerBound: rp(500_000_001), UpperBound: rp(1_000_000_000), Tier: 3,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			// OPERATOR_ID baris ini di berkas asli diakhiri BARIS BARU —
			// "MARTENPETRUSLALAMENTIK_1\n". Di sini sudah bersih; perapiannya dikerjakan
			// Threshold.Normalized supaya data dari Oracle pun ikut terlindungi.
			ID: "4", Name: "MARTENPETRUSLALAMENTIK", OperatorID: "MARTENPETRUSLALAMENTIK_1",
			BusinessLine: "NONMBU", CommitteeType: "2",
			LowerBound: rp(1_000_000_001), UpperBound: rp(100_000_000_000), Tier: 4,
			Active: true, ForAdjustment: true,
		},

		// ── Baris Non-MBU yang BUKAN jenjang persetujuan ─────────────────────────────
		//
		// Inilah jawaban atas pertanyaan terbuka "baris DEGREE=0 maksudnya apa?" pada
		// TKT-B07-001, dan jawabannya terbaca dari datanya sendiri: ia aktif, tetapi
		// STS_ADJ-nya KOSONG dan STS_REG-nya menyala. Ia penerima pemberitahuan saat
		// registrasi, bukan jenjang — dan penyaring STS_ADJ sudah mengeluarkannya tanpa
		// perlu aturan khusus tentang DEGREE.
		//
		// Ia sengaja dibawa ke berkas contoh supaya pengujian membuktikan baris seperti
		// ini benar-benar tidak pernah terhitung sebagai penyetuju.
		{
			ID: "9", Name: "KLAIMNONMBU", OperatorID: "",
			BusinessLine: "NONMBU", CommitteeType: "",
			LowerBound: rp(0), UpperBound: rp(0), Tier: 0,
			Active: true, ForAdjustment: false, ForRegistration: true,
		},

		// ── Non-MBU AB dan C — masing-masing satu jenjang ────────────────────────────
		{
			ID: "5", Name: "ELLENSUPRIYATI", OperatorID: "ELLENSUPRIYATI",
			BusinessLine: "NONMBUAB", CommitteeType: "0",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},
		{
			ID: "6", Name: "YOHANES RAYMOND ADIKARTA", OperatorID: "ELLENSUPRIYATI",
			BusinessLine: "NONMBUC", CommitteeType: "0",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},

		// ── Baris tidak aktif — dibawa supaya pengujian membuktikan ia diabaikan ─────
		{
			ID: "8", Name: "INDRA", OperatorID: "INDRAGUNAWAN",
			BusinessLine: "NONMBUAB", CommitteeType: "0",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: false, ForAdjustment: true,
		},
		{
			ID: "12", Name: "INDRA", OperatorID: "INDRAGUNAWAN",
			BusinessLine: "NONMBU", CommitteeType: "2",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: false, ForAdjustment: true, ForRejection: true,
		},

		// ── Travel — tiga jenjang, TANPA pita ────────────────────────────────────────
		//
		// Seluruh jenjang aktifnya ber-TYPE_KOMITE "1", termasuk yang di atas
		// Rp 100.000.000. Bila pita diberlakukan di sini, klaim Travel Rp 150.000.000
		// akan kehilangan SELURUH penyetujunya.
		{
			ID: "29", Name: "RATNA GUSTINA SARI", OperatorID: "RATNAGUSNITASARI",
			BusinessLine: "TRAVEL", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true, ForRegistration: true, ForRejection: true,
		},
		{
			ID: "30", Name: "RUDY WIDJAYA", OperatorID: "RUDYWIDJAJACHARISSA",
			BusinessLine: "TRAVEL", CommitteeType: "1",
			LowerBound: rp(50_000_001), UpperBound: rp(100_000_000), Tier: 2,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			ID: "31", Name: "DUMASI", OperatorID: "DUMASIMMSAMOSIR",
			BusinessLine: "TRAVEL", CommitteeType: "1",
			LowerBound: rp(100_000_001), UpperBound: rp(200_000_000), Tier: 3,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			ID: "28", Name: "IFANI", OperatorID: "IFANIOKTAVIANI",
			BusinessLine: "TRAVEL", CommitteeType: "",
			LowerBound: rp(0), UpperBound: rp(0), Tier: 1,
			Active: true, ForAdjustment: false, ForRegistration: true,
		},

		// ── Personal Accident — empat jenjang, TANPA pita ────────────────────────────
		//
		// TYPE_KOMITE di sini berselang-seling 2 · 1 · 1 · 2 menaiki tangga. Itulah
		// bukti paling jelas bahwa kolom tersebut BUKAN pita nilai pada lini ini,
		// melainkan pembeda PA reguler dari PA TKI (`D-70`).
		{
			ID: "41", Name: "Dr. Wahyu", OperatorID: "WAHYUKRISTANTI",
			BusinessLine: "PA", CommitteeType: "2",
			LowerBound: rp(0), UpperBound: rp(10_000_000), Tier: 1,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
		{
			ID: "42", Name: "Dr. Rossa", OperatorID: "MARGARETHAROSAGUNAWAN",
			BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(10_000_001), UpperBound: rp(50_000_000), Tier: 2,
			Active: true, ForAdjustment: true,
		},
		{
			ID: "43", Name: "Dr. Rudy", OperatorID: "RUDYWIDJAJACHARISSA",
			BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(50_000_001), UpperBound: rp(100_000_000), Tier: 3,
			Active: true, ForAdjustment: true,
		},
		{
			ID: "44", Name: "Dumasi", OperatorID: "DUMASIMMSAMOSIR",
			BusinessLine: "PA", CommitteeType: "2",
			LowerBound: rp(100_000_001), UpperBound: rp(200_000_000), Tier: 4,
			Active: true, ForAdjustment: true,
		},

		// ── Bonding — satu jenjang, batasnya 0/0 ─────────────────────────────────────
		//
		// Kueri Bonding di sistem lama (`EmailKomiteBerjenjangBonding_sql`) memang TIDAK
		// menyaring LIMIT sama sekali, dan isi masternya sejalan dengan itu.
		{
			ID: "74", Name: "RIZALGREATLIN", OperatorID: "RIZALGREATLIN",
			BusinessLine: "BONDING", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(0), Tier: 1,
			Active: true, ForAdjustment: true, ForRejection: true,
		},
	}
}
