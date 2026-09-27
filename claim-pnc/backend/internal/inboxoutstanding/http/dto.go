// Package inboxoutstandinghttp adalah lapisan transport modul Inbox Outstanding.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4). Memakai
// tipe domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien, dan
// sebaliknya — penggantian nama field di domain menjadi perubahan yang merusak antarmuka.
package inboxoutstandinghttp

import (
	"time"

	"claim-pnc/internal/inboxoutstanding"
)

// claimDTO adalah satu baris pada layar.
//
// # Nama field berbahasa Indonesia
//
// Ia KONTRAK yang dibaca frontend, dan termasuk lima pengecualian `D-80` bersama komentar,
// nama kolom basis data, teks layar, dan variabel lingkungan. Yang berbahasa Inggris
// hanyalah nama tipe dan field Go-nya.
//
// # Yang dihitung di server, bukan di peramban
//
// `umur_hari` dan `status_tampil` diturunkan di sini. Menyerahkannya ke frontend berarti
// aturan bisnis hidup di dua tempat — dan yang di peramban tidak dapat diuji bersama
// aturan lainnya. `status_tampil` khususnya: pemetaannya berasal dari activity Pega dan
// merupakan perilaku yang harus setara saat `S-8` dijalankan.
type claimDTO struct {
	KlaimID      string `json:"klaim_id"`
	NomorKlaim   string `json:"nomor_klaim"`      // kolom "Claim no"
	NomorPolis   string `json:"nomor_polis"`      // kolom "Policy no"
	Tertanggung  string `json:"nama_tertanggung"` // kolom "Insured name"
	NamaBisnis   string `json:"nama_bisnis"`      // kolom "Business Name"
	SumberBisnis string `json:"sumber_bisnis"`    // kolom "Business source"
	NamaCabang   string `json:"nama_cabang"`      // kolom "Branch name"
	GroupPanel   string `json:"group_panel"`

	// Tanggal dikirim sebagai teks ISO-8601 tanggal saja (YYYY-MM-DD), bukan timestamp.
	//
	// Kolom yang ditampilkan adalah TANGGAL, dan mengirim timestamp UTC memaksa setiap
	// tempat di frontend memutuskan sendiri cara mengubahnya ke WIB — yaitu cara paling
	// mudah membuat tanggal bergeser satu hari tanpa ada yang menyadarinya (`R-12`).
	TanggalPendaftaran string `json:"tanggal_pendaftaran"` // kolom "Register Date"
	TanggalKejadian    string `json:"tanggal_kejadian"`    // kolom "Date of loss"
	TanggalLapor       string `json:"tanggal_lapor"`       // tidak ditampilkan; lihat domain

	StatusProses string `json:"status_proses"`
	StatusTampil string `json:"status_tampil"` // kolom "Claim status"
	StatusKlaim  string `json:"status_klaim"`  // kolom "Status ASM" — KODE, bukan label
	PICTeknik    string `json:"pic_teknik"`    // kolom "ASM PIC"
	AdminPNC     string `json:"admin_pnc"`     // kolom "Admin name"

	// UmurHari adalah kolom "Total Aging" — dihitung dari tanggal registrasi.
	UmurHari int `json:"umur_hari"`

	// AgingHari adalah kolom "Aging" — dibaca APA ADANYA dari kolom `AGING`.
	//
	// Bertipe pointer supaya "belum terisi" dapat dibedakan dari "nol hari". Keduanya
	// tampil berbeda di layar: yang pertama tanda hubung, yang kedua angka nol.
	AgingHari *int `json:"aging_hari"`

	// Tiga field berikut TIDAK ditampilkan sebagai kolom; section rujukan tidak punya
	// kolomnya. Keduanya tetap dikirim karena kueri menyediakannya dan berguna saat
	// menelusuri satu klaim.
	PosisiKlaim   string `json:"posisi_klaim"`
	TahapKini     string `json:"tahap_kini"`
	PemegangTugas string `json:"pemegang_tugas"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Klaim []claimDTO `json:"klaim"`
	Total int        `json:"total"`

	// Pemilik adalah operator yang pekerjaannya ditampilkan.
	//
	// Ia ada supaya layar dapat MENYATAKANNYA kepada pengguna. Daftar kosong pada layar
	// bernama "My Inbox" punya dua sebab yang tampak sama: memang tidak ada pekerjaan, atau
	// penyaringnya salah orang. Menyebut pemiliknya membedakan keduanya.
	//
	// Nilainya selalu identitas pemanggil — tidak pernah datang dari klien.
	Pemilik string `json:"pemilik"`
}

// statusDTO adalah satu tab status dokumen.
type statusDTO struct {
	// Kode dipakai klien untuk menyaring; ia stabil, sedangkan judulnya teks layar.
	Kode  string `json:"kode"`
	Judul string `json:"judul"`

	// Jumlah bernilai `null` bila tab ini BELUM dapat dihitung, dan itu berbeda dari nol.
	//
	// Pointer, bukan int, supaya perbedaannya sampai ke klien apa adanya. Layar tidak
	// menggambar lencana untuk yang null — konvensi yang sama dengan tab "Data rejected"
	// pada Inbox Laporan Klaim, karena lencana bertuliskan 0 menyatakan "tidak ada" dan
	// itu tidak benar.
	Jumlah *int `json:"jumlah"`

	// DapatDipilih menandai tab yang benar-benar dapat menyaring daftar.
	//
	// Yang belum dapat dihitung juga belum dapat menyaring — kuerinya belum ada. Ia tetap
	// DITAMPILKAN supaya petugas Pega mengenali layarnya, tetapi tidak dapat ditekan.
	DapatDipilih bool `json:"dapat_dipilih"`
}

// summaryResponse adalah badan respons ringkasan.
type summaryResponse struct {
	Status []statusDTO `json:"status"`

	// Total adalah seluruh pekerjaan pemanggil — baris "All" pada tabel di samping donut.
	//
	// Ia dikirim TERPISAH, bukan dijumlahkan di peramban: hari ini penjumlahannya kebetulan
	// benar, dan akan diam-diam salah begitu ada status yang tidak termasuk keduanya.
	Total int `json:"total"`

	// Pemilik sama artinya dengan pada daftar — donut kosong punya dua sebab yang tampak
	// sama, dan menyebut pemiliknya membedakan keduanya.
	Pemilik string `json:"pemilik"`
}

// toClaimDTO mengubah satu klaim menjadi bentuk kiriman.
func toClaimDTO(c inboxoutstanding.OutstandingClaim, now time.Time, loc *time.Location) claimDTO {
	return claimDTO{
		KlaimID:            c.ClaimID,
		NomorKlaim:         c.ClaimNumber,
		NomorPolis:         c.PolicyNumber,
		Tertanggung:        c.InsuredName,
		NamaBisnis:         c.BusinessName,
		SumberBisnis:       c.BusinessSource,
		NamaCabang:         c.BranchName,
		GroupPanel:         c.GroupPanel,
		TanggalPendaftaran: formatDate(&c.RegisteredAt, loc),
		TanggalKejadian:    formatDate(c.LossDate, loc),
		TanggalLapor:       formatDate(c.ReportDate, loc),
		StatusProses:       c.ProcessStatus,
		StatusTampil:       string(c.DisplayStatus()),
		StatusKlaim:        c.ClaimStatus,
		PICTeknik:          c.TechnicalPIC,
		AdminPNC:           c.RecordedBy,
		UmurHari:           c.AgeInDays(now, loc),
		AgingHari:          c.AgingDays,
		PosisiKlaim:        c.ProgressStatus,
		TahapKini:          c.CurrentStage,
		PemegangTugas:      c.CurrentHolder,
	}
}

// formatDate mengubah waktu UTC menjadi tanggal WIB berbentuk YYYY-MM-DD.
//
// Inilah SATU-SATUNYA tempat konversi zona waktu terjadi pada modul ini. Penyimpanan
// memakai UTC dan tampilan memakai WIB; menyebar konversinya akan mengulang persis cacat
// `Set7Hours` sistem lama, yang menambah tujuh jam manual di puluhan tempat sehingga satu
// tempat yang lupa menggeser tanggal tanpa terdeteksi.
func formatDate(t *time.Time, loc *time.Location) string {
	if t == nil || t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}
