package inboxsalvage

import (
	"context"
	"strings"
)

// SubmissionNotice adalah PERISTIWA "pengajuan salvage tersimpan", bukan perintah kirim
// surel.
//
// # Kenapa bentuknya peristiwa, bukan `SendEmail(to, subject, body)`
//
// `04-FUTURE-ARCHITECTURE.md` §3.6 menetapkan seam Notifier bicara dalam peristiwa domain,
// supaya SIAPA penerimanya menjadi urusan konfigurasi (`D-15`) alih-alih urusan pemanggil.
// Itu bukan kerapian: di sistem lama penerimanya ditulis langsung di dalam activity —
// `Activity/SetStsSalvagePNC_act-Act.xml:1757-1760` memasang dua alamat tetap, salah
// satunya akun Gmail pribadi di jalur produksi. Selama pemanggil yang menentukan penerima,
// pola itu selalu dapat kembali.
//
// # Yang TIDAK ada di sini, dan sebabnya
//
// Isi badan surel. Rule HTML pemasoknya di sistem lama, `SendEmailRejectedApprovetochecker`
// (dirujuk `:10195`), TIDAK ADA di export — ia bagian dari `R-16`. Yang terbaca utuh hanya
// subjeknya (`:10212`) dan daftar penerimanya. Badan surel karena itu disusun pengisi seam
// dari isian di bawah, dan selisihnya dinyatakan di PlannedDifferences — bukan ditebak lalu
// disamarkan sebagai salinan.
type SubmissionNotice struct {
	// SalvageID adalah nomor pengajuan yang baru terbit.
	SalvageID string

	// ClaimNo adalah nomor klaimnya.
	//
	// Ia IKUT ke dalam surel — penerimanya memang perlu tahu klaim mana — tetapi TIDAK
	// ikut ke dalam log (`D-69`). Keduanya tempat yang berbeda dengan aturan yang berbeda.
	ClaimNo string

	// InsuredName adalah nama tertanggung, yang di sistem lama menjadi ekor subjek:
	// `"Pengajuan Salvage an " + pyWorkPage.Policy.QQName` (`:10212`).
	//
	// Boleh kosong. Subjeknya tetap terbentuk, hanya tanpa ekornya.
	InsuredName string

	SalvageType string
	Location    string
	Currency    string

	// MinimumValue dan OfferValue dibawa sebagai TEKS, apa adanya seperti yang tersimpan.
	//
	// Mengubahnya menjadi angka di sini berarti memutuskan pembulatan di tempat yang tidak
	// berwenang memutuskannya — `I-12` menetapkan nilai uang disimpan presisi penuh dan
	// dibulatkan hanya saat ditampilkan.
	MinimumValue string
	OfferValue   string

	Remark string

	// Submitter adalah Operator ID orang yang menekan Submit.
	Submitter string

	// ItemCount adalah jumlah baris Detail Item Salvage yang ikut tersimpan.
	ItemCount int

	// ExtraRecipients adalah isian "Email" pada form Tambah.
	//
	// Ia DITAMBAHKAN kepada penerima tetap dari konfigurasi, tidak menggantikannya. Alasan
	// menambahkan: isian itu ada di form lama dan diisi petugas dengan alamat pihak yang
	// perlu tahu pengajuan ini. Alasan tidak menggantikan: kalau ia menggantikan, satu
	// isian yang dikosongkan membuat surel tidak sampai ke siapa pun tanpa satu pun tanda.
	ExtraRecipients []string
}

// Subject menyusun subjek surel, mengikuti `SetStsSalvagePNC_act:10212` apa adanya.
//
// Ia di DOMAIN, bukan di pengisi seam, karena bunyinya adalah perilaku yang harus setara
// dengan Pega — bukan rincian SMTP. Pengisi seam yang berbeda tetap menghasilkan subjek
// yang sama, dan uji kesetaraannya cukup satu.
func (n SubmissionNotice) Subject() string {
	insured := strings.TrimSpace(n.InsuredName)
	if insured == "" {
		return "Pengajuan Salvage"
	}
	return "Pengajuan Salvage an " + insured
}

// Notifier adalah seam ke pemberitahuan.
//
// Dua pengisi nyata: pengirim SMTP, dan tiruan perekam untuk pengujian — syarat "dua
// adapter" pada `04-FUTURE-ARCHITECTURE.md` §3 karena itu terpenuhi.
//
// # Nil berarti TIDAK DIPASANG, dan itu keadaan yang sah
//
// Aplikasi melayani puluhan layar lain. Satu server surel yang belum dikonfigurasi tidak
// boleh membuat Submit gagal — yang terjadi sebagai gantinya adalah jawaban yang menyebut
// surel tidak terkirim, sehingga petugas tahu ia perlu mengabari orang lain sendiri.
type Notifier interface {
	NotifySalvageSubmitted(ctx context.Context, notice SubmissionNotice) error
}
