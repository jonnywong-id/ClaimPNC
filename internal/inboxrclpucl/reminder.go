package inboxrclpucl

import (
	"context"
	"errors"
	"strings"
)

// Reminder PUCL — pengingat kelengkapan dokumen ke tertanggung.
//
// # Dari mana bentuknya
//
// Tombol "Reminder PUCL" pada antrean Kelengkapan Dokumen
// (`Section/InboxKelengkapanDocPUCLRCL_Section-Section.xml:1156`) membuka modal berisi
// daftar klaim yang dapat dicentang, lalu mengirim surel lewat
// `Activity/ReminderPUCLPA-Act.xml`.
//
// # Yang terbaca utuh dari export, dan yang TIDAK
//
//	Terbaca   penerima    `.ClaimData.EmailLOD` — `T_CLAIM_PNC.EMAIL_LOD` (`:1350`)
//	Terbaca   subjek      lihat ReminderSubject (`:1415`)
//	Terbaca   akun surel  `"Admin-PNC"` (`:3629`)
//	HILANG    badan surel rule HTML `SendReminderPUCLKeTertanggungPA` (`:3521`)
//
// Rule HTML-nya tidak ada di export sama sekali — folder `HTML/` memuat 42 berkas dan tidak
// satu pun bernama mirip itu. Ia bagian dari `R-16`. Badan surelnya karena itu DISUSUN,
// bukan disalin, dan selisihnya dinyatakan di PlannedDifferences. Preseden yang sama sudah
// ditempuh modul Salvage untuk alasan yang sama persis.
//
// # Kredensial SMTP di export TIDAK dibawa
//
// Langkah pengirimnya memasang host, pengguna, dan KATA SANDI langsung di dalam activity.
// Itu `R-17`, dan `D-40` masih terbuka. Tidak satu pun nilainya direproduksi di berkas yang
// di-commit (`D-69`); konfigurasinya diambil dari `POOLDATA.M_EMAIL_SERVER_PNC` lewat
// `platform/emailserver`, akun `Admin-PNC` — jalur yang sudah dipakai modul lain.

// MaxReminderRecipients membatasi jumlah klaim dalam satu permintaan.
//
// Batas ini BUKAN kerapian. Satu permintaan menerbitkan satu surel per klaim ke tertanggung
// — tindakan yang keluar dari sistem dan tidak dapat ditarik kembali. Permintaan yang
// menyapu seluruh antrean karena satu klik keliru adalah kegagalan yang jauh lebih mahal
// daripada menyuruh petugas mengirim dua kali.
//
// Angkanya sama dengan satu halaman penuh grid (`pyPageSize = 50`), sehingga petugas yang
// memilih seluruh baris yang TERLIHAT tidak pernah tertolak.
const MaxReminderRecipients = 50

// ErrNoReminderRecipients menyatakan permintaan tidak memuat satu pun klaim.
var ErrNoReminderRecipients = errors.New("inboxrclpucl: tidak ada klaim yang dipilih")

// ErrTooManyReminderRecipients menyatakan permintaan melampaui MaxReminderRecipients.
var ErrTooManyReminderRecipients = errors.New("inboxrclpucl: klaim yang dipilih terlalu banyak")

// Dua sebab kegagalan pengiriman yang WAJIB dibedakan.
//
// "Ditolak" dan "tidak terjangkau" menuntut tindakan yang berbeda, dan menyebut keduanya
// dengan kalimat yang sama menyuruh orang mencari di tempat yang keliru. Itu benar-benar
// terjadi 2026-10-12: layar berbunyi "Pengiriman ditolak peladen surel" padahal nama host
// peladennya tidak teresolusi sama sekali — peladennya tidak pernah dihubungi, apalagi
// menolak. Yang dicari orang menjadi akun surel, bukan nama host.
var (
	// ErrReminderUnreachable: peladen surel tidak dapat DIHUBUNGI — nama host tidak
	// teresolusi, jaringan tertutup, atau tidak menjawab. Perbaikannya urusan konfigurasi
	// dan jaringan, bukan akun surel.
	ErrReminderUnreachable = errors.New("inboxrclpucl: peladen surel tidak dapat dihubungi")

	// ErrReminderRejected: peladen MENJAWAB dan menolak — kredensial, alamat pengirim,
	// atau alamat tujuan. Perbaikannya urusan akun surel dan isian alamat.
	ErrReminderRejected = errors.New("inboxrclpucl: peladen surel menolak pengiriman")
)

// ReminderRecipient adalah satu klaim beserta alamat surel tertanggungnya.
//
// Ia dibaca ULANG di peladen dari nomor klaim yang dikirim layar — bukan diterima dari
// layar. Alamat surel yang datang dari permintaan dapat diubah siapa pun yang dapat
// memanggil API, dan ini tindakan yang MENGIRIM sesuatu keluar.
type ReminderRecipient struct {
	// ClaimNumber adalah nomor case, mis. `PNC-2067`.
	ClaimNumber string

	// InsuredName adalah `QQ_NAME` — dipakai menyusun subjek.
	InsuredName string

	// Email adalah `T_CLAIM_PNC.EMAIL_LOD`. Boleh KOSONG: kolomnya baru mulai diisi, dan
	// klaim lama belum tentu punya. Klaim tanpa alamat dilewati dan dilaporkan, bukan
	// menggagalkan seluruh permintaan.
	Email string
}

// Status satu klaim pada hasil pengiriman.
const (
	// ReminderSent — surelnya diserahkan ke peladen SMTP.
	ReminderSent = "terkirim"

	// ReminderNoEmail — klaimnya tidak punya alamat surel tertanggung.
	ReminderNoEmail = "tanpa-email"

	// ReminderNotEligible — nomor klaimnya tidak ada di antrean Kelengkapan Dokumen.
	//
	// Ia bukan sekadar "tidak ditemukan": nomor yang tidak memenuhi penyaring antrean tidak
	// boleh dikirimi pengingat meski klaimnya ada, karena pengingat kelengkapan dokumen
	// hanya bermakna bagi klaim yang memang sedang menunggu dokumen.
	ReminderNotEligible = "tidak-memenuhi-syarat"

	// ReminderFailed — pengirimannya ditolak peladen SMTP.
	ReminderFailed = "gagal"
)

// ReminderOutcome adalah hasil satu klaim.
type ReminderOutcome struct {
	ClaimNumber string
	Status      string

	// Reason diisi hanya pada ReminderFailed, dan isinya pesan yang layak dibaca petugas —
	// bukan galat mentah peladen SMTP, yang dapat memuat nama host internal.
	Reason string
}

// ReminderResult adalah hasil seluruh permintaan.
//
// Ia selalu melaporkan PER KLAIM, tidak pernah satu kalimat untuk semuanya. Pengiriman
// sebagian adalah hasil yang lazim di sini — sebagian klaim punya alamat, sebagian belum —
// dan satu kalimat "berhasil" atau "gagal" menyembunyikan tepat hal yang perlu ditindak.
type ReminderResult struct {
	Outcomes []ReminderOutcome
}

// Count menghitung klaim berstatus tertentu.
func (r ReminderResult) Count(status string) int {
	n := 0
	for _, outcome := range r.Outcomes {
		if outcome.Status == status {
			n++
		}
	}
	return n
}

// ReminderSubject menyusun subjek surel, mengikuti Pega apa adanya.
//
// Ekspresi aslinya (`Activity/ReminderPUCLPA-Act.xml:1415`):
//
//	"REMINDER PROSES ULANG KLAIM PERSONAL AACCIDENT - A/N "
//	  + @whatComesBeforeFirst(pyWorkPage.Policy.QQName, 'Q')
//
// # Dua hal yang SENGAJA ditiru apa adanya
//
// **Salah ketik "AACCIDENT"** — huruf A ganda. Ia ada di Pega dan sampai ke kotak masuk
// tertanggung hari ini. `P-5` menetapkan perilaku dipertahankan lebih dulu, dan memperbaiki
// teks yang dibaca pihak LUAR adalah perubahan yang layak diputuskan Work Owner, bukan
// diselipkan. Dicatat di PlannedDifferences sebagai calon perbaikan.
//
// **Pemenggalan pada huruf `Q` pertama**, bukan pada `" QQ "`. Konvensi nama tertanggung di
// sini memakai `QQ` sebagai pemisah ("PT.SUMMIT OTO FINANCE QQ YAZIER …"), dan Pega
// memenggalnya pada huruf Q mana pun yang muncul lebih dulu. Nama yang memuat Q lebih awal
// karena itu terpotong terlalu dini — perilaku yang ditiru, bukan diperbaiki.
func ReminderSubject(insuredName string) string {
	name := insuredName
	if i := strings.IndexByte(name, 'Q'); i >= 0 {
		name = name[:i]
	}
	return "REMINDER PROSES ULANG KLAIM PERSONAL AACCIDENT - A/N " + strings.TrimSpace(name)
}

// ReminderNotifier adalah seam pengirim pengingat.
//
// Satu klaim satu pemanggilan, bukan satu pemanggilan untuk seluruh daftar: kegagalan satu
// alamat tidak boleh membatalkan sisanya, dan hasilnya wajib dilaporkan per klaim.
type ReminderNotifier interface {
	SendReminder(ctx context.Context, recipient ReminderRecipient) error
}

// NormalizeReminderRequest merapikan dan memvalidasi daftar nomor klaim yang dipilih.
//
// Ia membuang spasi, membuang yang kosong, dan membuang KEMBARAN — dua baris terpilih yang
// bernomor sama akan menghasilkan dua surel ke orang yang sama.
func NormalizeReminderRequest(claimNumbers []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(claimNumbers))

	for _, number := range claimNumbers {
		trimmed := strings.TrimSpace(number)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}

	if len(out) == 0 {
		return nil, ErrNoReminderRecipients
	}
	if len(out) > MaxReminderRecipients {
		return nil, ErrTooManyReminderRecipients
	}
	return out, nil
}
