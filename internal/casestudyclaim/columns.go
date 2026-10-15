package casestudyclaim

// Kolom grid, dalam satu daftar.
//
// # Kenapa daftarnya di sini dan bukan di layar
//
// Ketiga pemakainya harus sepakat: grid di layar, judul kolom berkas CSV, dan uji
// kesetaraan `S-8` yang membandingkan keluaran ini dengan grid Pega. Selama daftarnya
// ditulis ulang di tiap tempat, "sepakat" adalah harapan — dan yang paling mudah
// tertinggal adalah berkas CSV, yang tidak dilihat siapa pun sampai ada yang mengunduhnya.
//
// Bentuknya satu tabel data, bukan rangkaian `switch`, dengan alasan yang sama seperti
// `statusOrder` pada modul Inbox Compliance: definisi baru dan grid lama dapat
// dibandingkan baris per baris.

// ColumnKind menyatakan cara sebuah kolom digambar dan diformat.
//
// Ia dikirim ke layar supaya layar tidak perlu menyimpan daftarnya sendiri untuk
// memutuskan mana yang rata kanan dan mana yang diformat sebagai rupiah.
type ColumnKind string

const (
	// KindText: teks apa adanya.
	KindText ColumnKind = "teks"

	// KindDate: tanggal `YYYY-MM-DD`, digambar layar sebagai `1 Juni 2026`.
	KindDate ColumnKind = "tanggal"

	// KindMoney: nilai uang dalam satuan terkecil (sen). Rata kanan.
	KindMoney ColumnKind = "uang"

	// KindPercent: persentase dikali 10.000. Rata kanan.
	KindPercent ColumnKind = "persen"

	// KindRemark: satu-satunya kolom yang dapat disunting.
	KindRemark ColumnKind = "catatan"
)

// Column adalah satu kolom grid.
type Column struct {
	// Key adalah nama field pada baris JSON. Ia stabil; judulnya tidak.
	Key string

	// Title adalah judul kolom PERSIS seperti di layar Pega (`D-13`).
	//
	// Termasuk huruf besar semua pada sebagian judul, dan termasuk salah ketik
	// "SUBROGARATION" — keduanya ada di
	// `Section/PNCStudyClaim-Section.xml` dan tidak diperbaiki di sini. Memperbaikinya
	// berarti layar baru menampilkan teks yang berbeda dari layar yang sedang dipakai,
	// dan itu muncul sebagai selisih pada uji kesetaraan.
	Title string

	Kind ColumnKind
}

// columnOrder adalah urutan kolom PERSIS seperti grid Pega.
//
// Dibaca dari `Section/PNCStudyClaim-Section.xml` berurutan: judul tebalnya pada `:5655`,
// `:5821`, `:5949`, `:6079`, `:6263`, `:6381`, `:6497`, `:6658`, `:6783`, `:6964`,
// `:7120`, `:7240`, `:7358`, `:7517`, `:7646`, `:7785`, `:7950`, `:8028`, `:8226`,
// `:8303`, `:8470`, `:8623`, `:8745`, `:8909` — dan properti yang mengisinya pada
// `:9154` sampai `:12814`, berpasangan satu-satu.
//
// Kolom aksi (tombol Save) TIDAK ada di daftar ini: ia bukan data melainkan kontrol, dan
// backend tidak tahu apa pun tentang tombol. Layar menambahkannya sendiri di ujung, sama
// seperti modul Inbox Compliance.
var columnOrder = []Column{
	{"nomor_klaim", "PNC Case ID", KindText},
	{"nomor_polis", "No Polis", KindText},
	{"nama_tertanggung", "Nama Tertanggung", KindText},
	{"cob", "COB", KindText},
	{"periode_polis", "Periode Polis", KindText},
	{"bulan_klaim", "Bulan Klaim", KindText},
	{"tanggal_kejadian", "Date of Loss", KindDate},
	{"sob", "SOB", KindText},
	{"posisi_reasuransi", "Leader/Member/Fac In", KindText},

	// Dua kolom berikut SELALU berisi nilai yang sama — keduanya `A.COL_DESC`. Lihat
	// CaseStudyRow.CauseOfLoss; kekembarannya disengaja dan dinyatakan di layar.
	{"nature_of_loss", "Nature of Loss", KindText},
	{"cause_of_loss", "Cause of Loss", KindText},

	{"tsi_100", "TSI (100%)", KindMoney},
	{"share_asm", "ASM SHARE", KindPercent},
	{"deductible", "Deductible", KindMoney},
	{"nilai_share_asm", "Nilai share ASM", KindMoney},
	{"nilai_klaim_100", "NILAI KLAIM 100%", KindMoney},
	{"adjuster_fee_100", "ADJUSTER FEE 100% SHARE", KindMoney},
	{"nilai_klaim_net_100", "NILAI KLAIM NET 100%", KindMoney},
	{"nilai_klaim_net_share_asm", "NILAI KLAIM NET ASM SHARE", KindMoney},
	{"lack_of_doc", "LACK OF DOC/SALVAGE / RECOVERY / SUBROGARATION", KindMoney},

	{"cabang", "Cabang", KindText},
	{"status_klaim", "Status", KindText},
	{"kronologi", "Kronologi", KindText},
	{"remark", "Remark", KindRemark},
}

// Columns mengembalikan seluruh kolom grid dalam urutan layar.
//
// Salinan dikembalikan, bukan irisan aslinya: pemanggil yang menyortir hasilnya akan
// menata ulang grid setiap layar sekaligus, dan kegagalan seperti itu tidak menghasilkan
// galat apa pun.
func Columns() []Column {
	result := make([]Column, len(columnOrder))
	copy(result, columnOrder)
	return result
}

// ColumnTitles mengembalikan judul kolom dalam urutan layar.
//
// Dipakai judul baris pertama berkas CSV, supaya berkas yang diunduh terbaca sebagai
// salinan apa yang dilihat pengguna — bukan susunan lain yang harus dicocokkan sendiri.
func ColumnTitles() []string {
	titles := make([]string, 0, len(columnOrder))
	for _, column := range columnOrder {
		titles = append(titles, column.Title)
	}
	return titles
}
