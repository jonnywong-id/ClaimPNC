package memori

import (
	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/uang"
)

// AmbangContoh mengembalikan isi master ambang komite sebagaimana yang berlaku hari ini.
//
// # Dari mana angkanya
//
// Diturunkan baris per baris dari `Database/emailkomite.csv` — berkas yang diserahkan
// Work Owner, 30 baris dan 21 kolom. Ia BUKAN data karangan: setiap batas, jenjang, dan
// penanda status di bawah ada di berkas itu, dan itulah yang membuat ketujuh kasus
// jumlah penyetuju pada `docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md` dapat
// diuji tanpa basis data sama sekali.
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
func AmbangContoh() []komite.Ambang {
	rp := uang.DariRupiah

	return []komite.Ambang{
		// ── Non-MBU, pita 1 — klaim sampai Rp 100.000.000 ────────────────────────────
		//
		// Perhatikan kedua baris ini ber-DEGREE SAMA (1). Itu bukan salah salin: di
		// master pun demikian, dan itulah sebab penanda UrutanTidakPasti ada.
		{
			ID: "7", Nama: "ELLENSUPRIYATI", OperatorID: "ELLENSUPRIYATI",
			Lini: "NONMBU", JenisKomite: "1",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			ID: "1", Nama: "INDRA", OperatorID: "INDRAGUNAWAN",
			Lini: "NONMBU", JenisKomite: "1",
			BatasBawah: rp(50_000_001), BatasAtas: rp(100_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true,
		},

		// ── Non-MBU, pita 2 — klaim di atas Rp 100.000.000 ───────────────────────────
		{
			ID: "2", Nama: "BAMBANG", OperatorID: "BAMBANGSETIADJIGUNAWAN",
			Lini: "NONMBU", JenisKomite: "2",
			BatasBawah: rp(100_000_001), BatasAtas: rp(500_000_000), Jenjang: 2,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			ID: "3", Nama: "DANIELLISWANDI", OperatorID: "DANIELLISWANDI",
			Lini: "NONMBU", JenisKomite: "2",
			BatasBawah: rp(500_000_001), BatasAtas: rp(1_000_000_000), Jenjang: 3,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			// OPERATOR_ID baris ini di berkas asli diakhiri BARIS BARU —
			// "MARTENPETRUSLALAMENTIK_1\n". Di sini sudah bersih; perapiannya
			// dikerjakan Ambang.Bersih supaya data dari Oracle pun ikut terlindungi.
			ID: "4", Nama: "MARTENPETRUSLALAMENTIK", OperatorID: "MARTENPETRUSLALAMENTIK_1",
			Lini: "NONMBU", JenisKomite: "2",
			BatasBawah: rp(1_000_000_001), BatasAtas: rp(100_000_000_000), Jenjang: 4,
			Aktif: true, UntukAdjustment: true,
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
			ID: "9", Nama: "KLAIMNONMBU", OperatorID: "",
			Lini: "NONMBU", JenisKomite: "",
			BatasBawah: rp(0), BatasAtas: rp(0), Jenjang: 0,
			Aktif: true, UntukAdjustment: false, UntukRegistrasi: true,
		},

		// ── Non-MBU AB dan C — masing-masing satu jenjang ────────────────────────────
		{
			ID: "5", Nama: "ELLENSUPRIYATI", OperatorID: "ELLENSUPRIYATI",
			Lini: "NONMBUAB", JenisKomite: "0",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true,
		},
		{
			ID: "6", Nama: "YOHANES RAYMOND ADIKARTA", OperatorID: "ELLENSUPRIYATI",
			Lini: "NONMBUC", JenisKomite: "0",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true,
		},

		// ── Baris tidak aktif — dibawa supaya pengujian membuktikan ia diabaikan ─────
		{
			ID: "8", Nama: "INDRA", OperatorID: "INDRAGUNAWAN",
			Lini: "NONMBUAB", JenisKomite: "0",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: false, UntukAdjustment: true,
		},
		{
			ID: "12", Nama: "INDRA", OperatorID: "INDRAGUNAWAN",
			Lini: "NONMBU", JenisKomite: "2",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: false, UntukAdjustment: true, UntukPenolakan: true,
		},

		// ── Travel — tiga jenjang, TANPA pita ────────────────────────────────────────
		//
		// Seluruh jenjang aktifnya ber-TYPE_KOMITE "1", termasuk yang di atas
		// Rp 100.000.000. Bila pita diberlakukan di sini, klaim Travel Rp 150.000.000
		// akan kehilangan SELURUH penyetujunya.
		{
			ID: "29", Nama: "RATNA GUSTINA SARI", OperatorID: "RATNAGUSNITASARI",
			Lini: "TRAVEL", JenisKomite: "1",
			BatasBawah: rp(0), BatasAtas: rp(50_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true, UntukRegistrasi: true, UntukPenolakan: true,
		},
		{
			ID: "30", Nama: "RUDY WIDJAYA", OperatorID: "RUDYWIDJAJACHARISSA",
			Lini: "TRAVEL", JenisKomite: "1",
			BatasBawah: rp(50_000_001), BatasAtas: rp(100_000_000), Jenjang: 2,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			ID: "31", Nama: "DUMASI", OperatorID: "DUMASIMMSAMOSIR",
			Lini: "TRAVEL", JenisKomite: "1",
			BatasBawah: rp(100_000_001), BatasAtas: rp(200_000_000), Jenjang: 3,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			ID: "28", Nama: "IFANI", OperatorID: "IFANIOKTAVIANI",
			Lini: "TRAVEL", JenisKomite: "",
			BatasBawah: rp(0), BatasAtas: rp(0), Jenjang: 1,
			Aktif: true, UntukAdjustment: false, UntukRegistrasi: true,
		},

		// ── Personal Accident — empat jenjang, TANPA pita ────────────────────────────
		//
		// TYPE_KOMITE di sini berselang-seling 2 · 1 · 1 · 2 menaiki tangga. Itulah
		// bukti paling jelas bahwa kolom tersebut BUKAN pita nilai pada lini ini,
		// melainkan pembeda PA reguler dari PA TKI (`D-70`).
		{
			ID: "41", Nama: "Dr. Wahyu", OperatorID: "WAHYUKRISTANTI",
			Lini: "PA", JenisKomite: "2",
			BatasBawah: rp(0), BatasAtas: rp(10_000_000), Jenjang: 1,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
		{
			ID: "42", Nama: "Dr. Rossa", OperatorID: "MARGARETHAROSAGUNAWAN",
			Lini: "PA", JenisKomite: "1",
			BatasBawah: rp(10_000_001), BatasAtas: rp(50_000_000), Jenjang: 2,
			Aktif: true, UntukAdjustment: true,
		},
		{
			ID: "43", Nama: "Dr. Rudy", OperatorID: "RUDYWIDJAJACHARISSA",
			Lini: "PA", JenisKomite: "1",
			BatasBawah: rp(50_000_001), BatasAtas: rp(100_000_000), Jenjang: 3,
			Aktif: true, UntukAdjustment: true,
		},
		{
			ID: "44", Nama: "Dumasi", OperatorID: "DUMASIMMSAMOSIR",
			Lini: "PA", JenisKomite: "2",
			BatasBawah: rp(100_000_001), BatasAtas: rp(200_000_000), Jenjang: 4,
			Aktif: true, UntukAdjustment: true,
		},

		// ── Bonding — satu jenjang, batasnya 0/0 ─────────────────────────────────────
		//
		// Kueri Bonding di sistem lama (`EmailKomiteBerjenjangBonding_sql`) memang TIDAK
		// menyaring LIMIT sama sekali, dan isi masternya sejalan dengan itu.
		{
			ID: "74", Nama: "RIZALGREATLIN", OperatorID: "RIZALGREATLIN",
			Lini: "BONDING", JenisKomite: "1",
			BatasBawah: rp(0), BatasAtas: rp(0), Jenjang: 1,
			Aktif: true, UntukAdjustment: true, UntukPenolakan: true,
		},
	}
}
