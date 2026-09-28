package inboxpladlapredla

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Berkas ini memuat operasi **"SEND"** — mengirim surat PLA/DLA beserta lampirannya ke
// reasuradur, lalu menandai dokumennya terkirim.
//
// Ia satu-satunya operasi di seluruh modul yang menyentuh dunia di luar basis data, dan
// satu-satunya yang tidak dapat ditarik kembali. Bagian terbesar berkas ini adalah
// pengaturan URUTAN dan pembatasan akibat — bukan penyusunan surat.
//
// # Apa yang digantikan
//
// `Activity/UpdateDetailPLA2-Act.xml`, yang di Pega menjalankan DUA BELAS langkah.
// Tujuh dibawa, lima tidak:
//
//	 1  validasi email reasuradur tidak kosong            DIBAWA
//	 2  susun subjek (Indonesia / Inggris menurut negara) DIBAWA
//	 3  susun badan surat                                 DIBAWA
//	 4  kumpulkan lampiran                                DIBAWA
//	 5  kirim surat + lampiran                            DIBAWA
//	 6  tandai dokumen terkirim                           DIBAWA
//	 7  perbarui master reasuransi (`UPDATEREAS`)         DIBAWA, ditulis ulang di Go
//	 8  buat operator Pega (`GCNMCreateOperator`)         TIDAK — tidak ada operator Pega
//	 9  log mitra (`PNCInsertMitraLog_Act`)               TIDAK — lihat di bawah
//	10  catat id dokumen (`InsertDokumenPLADLA`)          TIDAK — lihat di bawah
//	11  `PNCInsertPLADLA`                                 TIDAK — lihat di bawah
//	12  `Obj-Save` + `Commit` objek kerja Pega            TIDAK — tidak ada objek kerja
//
// Langkah 8 dan 12 menyentuh mekanisme Pega yang tidak punya padanan: operator Pega dan
// objek kerja Pega. Langkah 9, 10, dan 11 menulis tabel pencatatan yang **belum dianalisis
// isinya**, dan menulis tabel yang belum dipahami lebih buruk daripada tidak menulisnya —
// keduanya dicatat sebagai selisih terencana, bukan disamarkan.
//
// # URUTAN adalah bagian terpenting, dan ia berbeda dari Pega
//
// Pega menandai dokumen terkirim **sebelum** suratnya dikirim. Urutan itu tidak dibawa:
//
//	Pega     tandai terkirim -> … -> kirim surat
//	Di sini  kirim surat -> bila BERHASIL -> tandai terkirim
//
// Sebabnya satu kalimat: **baris yang hilang dari antrean tanpa surat yang sampai tidak
// meninggalkan jejak apa pun.** Tidak ada galat, tidak ada tanda, dan tidak ada yang
// mengetahui suratnya tidak pernah dikirim sampai reasuradur menanyakannya.
//
// Urutan di sini membalik arah kegagalannya: bila penandaan gagal setelah surat terkirim,
// barisnya TETAP di antrean dan dapat dikirim ulang. Pengiriman ganda merepotkan;
// pengiriman yang tidak pernah terjadi merugikan.

// SendOutcome menyatakan apa yang benar-benar terjadi pada satu pengiriman.
//
// Ia bukan sekadar berhasil-gagal. Surat yang terkirim tetapi gagal ditandai adalah
// keadaan yang HARUS dapat dibedakan — pengguna perlu tahu bahwa suratnya sudah sampai
// sebelum ia menekan tombolnya lagi.
type SendOutcome struct {
	// Recipients adalah alamat yang benar-benar dikirimi.
	Recipients []string

	// Attachments adalah jumlah lampiran yang ikut.
	Attachments int

	// Marked menyatakan dokumennya berhasil ditandai terkirim.
	//
	// `false` bersama galat `nil` tidak mungkin: penandaan yang gagal selalu disertai
	// galat. Ia ada supaya pemanggil tidak perlu menyimpulkannya dari ketiadaan galat.
	Marked bool
}

// Letter adalah satu surat yang siap dikirim.
type Letter struct {
	To      []string
	Subject string

	// HTMLBody adalah badan suratnya. Ia HTML karena surat Pega pun HTML.
	HTMLBody string

	Attachments []Attachment
}

// Attachment adalah satu berkas lampiran beserta isinya.
//
// Isinya dibawa di memori, dan itu batas yang disadari: surat PLA/DLA membawa dokumen
// pendukung satu klaim, bukan arsip. Bila kelak ada klaim yang lampirannya sangat besar,
// yang harus berubah adalah cara mengirimnya — bukan diam-diam memotongnya di sini.
type Attachment struct {
	Name     string
	MIMEType string
	Content  []byte
}

// Notifier adalah seam ke pengiriman surat.
//
// Dua pengisi nyata: SMTP, dan pengisi palsu yang merekam surat tanpa mengirimnya. Yang
// kedua bukan sekadar alat uji — ia yang membuat seluruh alur Send dapat dijalankan
// berulang kali tanpa satu pun surat sampai ke reasuradur sungguhan.
type Notifier interface {
	// SendAdvice mengirim satu surat. Galat berarti surat TIDAK terkirim.
	//
	// Kontrak itu mengikat, dan ia yang menopang seluruh urutan di berkas ini: penandaan
	// terkirim hanya dijalankan ketika metode ini mengembalikan `nil`. Pengisi yang
	// mengembalikan `nil` pada pengiriman yang gagal akan menghasilkan baris yang hilang
	// dari antrean tanpa surat.
	SendAdvice(ctx context.Context, letter Letter) error
}

// SendableAdvice adalah satu baris PLA atau DLA beserta keterangan reasuradurnya.
type SendableAdvice struct {
	AdviceNo   string
	AdviceType string

	// Reinsurer adalah nama reasuradur pada baris dokumennya
	// (`T_PLALIST.PLAREINSURER` / `T_DLALIST.DLAREINSURER`).
	Reinsurer string

	// ReinsurerID adalah `REINSCODE` — kunci ke `POOLDATA.T_REINSURER`.
	ReinsurerID string

	// Email adalah alamat tujuan (`EMAILPLA` / `EMAILDLA`).
	//
	// KOSONG menghentikan pengiriman. Pega memeriksanya pula dan menjawab
	// "Email Reinsurer Kosong".
	Email string

	// Sent adalah `ISKIRIM`. Dokumen yang sudah terkirim tidak dikirim ulang.
	Sent string

	// Login dan Country berasal dari `T_REINSURER`, dicocokkan dengan `REINSCODE`
	// ditambah **huruf pertama nomor dokumen** sebagai `TYPE`.
	//
	// Aturan huruf pertama itu bukan tebakan: `RDB List/GetDataPreDLA-SQL.xml` memakai
	// `substr(a.NODLA,0,1) = TYPE` pada sub-kueri yang membaca `LOGIN` dan `COUNTRY` dari
	// tabel yang sama.
	Login   string
	Country string
}

// ClaimSummary adalah keterangan klaim yang muncul di subjek surat.
type ClaimSummary struct {
	ClaimNo  string
	PolicyNo string
	Insured  string
	Business string

	// LossDate sudah berbentuk teks `YYYY-MM-DD`; subjeknya menuliskannya apa adanya.
	LossDate string
}

// ErrReinsurerEmailEmpty menyatakan dokumen itu belum punya alamat reasuradur.
//
// Pega menjawabnya dengan "Email Reinsurer Kosong". Ia bukan kegagalan sistem melainkan
// data yang belum lengkap, dan pesannya menyebut apa yang harus dilengkapi.
var ErrReinsurerEmailEmpty = errors.New(
	"inboxpladlapredla: alamat surel reasuradur belum diisi")

// ErrAdviceAlreadySent menyatakan dokumen itu sudah terkirim.
var ErrAdviceAlreadySent = errors.New(
	"inboxpladlapredla: dokumen ini sudah terkirim")

// ErrLetterNotSent menyatakan suratnya gagal dikirim, sehingga TIDAK ada yang ditandai.
//
// Ia dibungkus di sekitar galat aslinya supaya lapisan transport dapat membedakannya dari
// kegagalan basis data — keduanya menghasilkan tindakan pengguna yang berbeda.
var ErrLetterNotSent = errors.New(
	"inboxpladlapredla: surat tidak terkirim")

// ErrNotifierUnavailable menyatakan pengiriman surat belum dapat dijalankan sama sekali.
//
// Berbeda dari ErrLetterNotSent: yang ini berarti sambungan surelnya belum disiapkan,
// bukan bahwa pengirimannya dicoba lalu gagal. Pengguna tidak dapat menolongnya dengan
// mencoba lagi.
var ErrNotifierUnavailable = errors.New(
	"inboxpladlapredla: pengiriman surat belum disiapkan")

// ErrSentButNotMarked menyatakan suratnya TERKIRIM tetapi dokumennya gagal ditandai.
//
// Ia keadaan terburuk yang dapat terjadi pada operasi ini, dan justru karena itu ia punya
// galatnya sendiri: pengguna yang membaca "gagal" akan menekan tombolnya lagi, dan surat
// kedua akan sampai ke reasuradur.
//
// Barisnya tetap di antrean — itu benar, karena catatannya memang belum ada. Yang harus
// diperbaiki adalah catatannya, dan itu tidak dapat dilakukan dengan mengirim ulang.
var ErrSentButNotMarked = errors.New(
	"inboxpladlapredla: surat terkirim tetapi dokumennya gagal ditandai")

// CanBeSent memeriksa satu dokumen sebelum suratnya disusun.
//
// Pemeriksaan ini ada di DOMAIN, bukan di adapter, karena keduanya aturan bisnis: alamat
// yang kosong dan dokumen yang sudah terkirim sama-sama menghentikan pengiriman, dan
// keduanya harus menghasilkan pesan yang berbeda.
func (a SendableAdvice) CanBeSent() error {
	if strings.TrimSpace(a.Email) == "" {
		return ErrReinsurerEmailEmpty
	}
	if strings.TrimSpace(a.Sent) == "1" {
		return ErrAdviceAlreadySent
	}
	return nil
}

// Recipients memecah isian alamat menjadi daftar penerima.
//
// Kolomnya memuat SATU alamat pada sebagian besar baris, tetapi tidak selalu: isian yang
// diketik manusia kerap memuat beberapa alamat dipisahkan koma atau titik koma. Mengirim
// ke teks gabungan itu apa adanya akan ditolak relay SMTP, dan penolakannya terbaca
// sebagai gangguan jaringan.
func (a SendableAdvice) Recipients() []string {
	pisah := func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	}

	hasil := []string{}
	terlihat := map[string]bool{}

	for _, bagian := range strings.FieldsFunc(a.Email, pisah) {
		alamat := strings.TrimSpace(bagian)
		if alamat == "" || !strings.Contains(alamat, "@") {
			continue
		}
		kunci := strings.ToLower(alamat)
		if terlihat[kunci] {
			continue
		}
		terlihat[kunci] = true
		hasil = append(hasil, alamat)
	}

	return hasil
}

// InIndonesian menyatakan suratnya memakai bahasa Indonesia.
//
// Pega memilihnya dengan `@if(@contains(param.country,"INDONESIA"), …)`
// (`UpdateDetailPLA2-Act.xml:426250`) — pencocokan teks pada nama negara reasuradur,
// bukan pada kode negara. Aturannya ditiru apa adanya, termasuk kelonggarannya: negara
// yang namanya memuat kata itu ikut terhitung.
//
// Negara yang KOSONG menghasilkan bahasa Inggris. Itu pilihan yang lebih aman: surat
// berbahasa Inggris kepada reasuradur Indonesia tetap terbaca, sedangkan sebaliknya belum
// tentu.
func (a SendableAdvice) InIndonesian() bool {
	return strings.Contains(strings.ToUpper(a.Country), "INDONESIA")
}

// Subject menyusun subjek surat.
//
// Bentuknya mengikuti `UpdateDetailPLA2-Act.xml:275170` apa adanya:
//
//	PLA Supporting Document Claim {bisnis} -- {no klaim} a/n. {tertanggung}
//	  -- Policy : {no polis} -- Dol : {tanggal kejadian}
//
// Satu hal yang TIDAK dibawa: Pega menambahkan tipe treaty ke subjek pada jalur treaty
// (`:278239`). Percabangan itu belum ditelusuri sampai ke sumber nilainya, dan menebaknya
// akan menghasilkan subjek yang salah pada sebagian surat — lebih buruk daripada subjek
// yang seragam.
func Subject(tab Tab, claim ClaimSummary) string {
	jenis := "PLA"
	if tab.Kind == KindDLA {
		jenis = "DLA"
	}

	return fmt.Sprintf(
		"%s Supporting Document Claim %s -- %s a/n. %s -- Policy : %s -- Dol : %s",
		jenis,
		strings.TrimSpace(claim.Business),
		strings.TrimSpace(claim.ClaimNo),
		strings.TrimSpace(claim.Insured),
		strings.TrimSpace(claim.PolicyNo),
		strings.TrimSpace(claim.LossDate),
	)
}

// AttachmentCategory adalah kategori lampiran yang ikut dikirim.
//
// `UpdateDetailPLA2-Act.xml:251646` memakai `"PLA"`; jalur DLA memakai `"DLA"`. Kategori
// itu pula yang dipakai panel "Print Pre DLA" pada kuerinya sendiri.
func AttachmentCategory(tab Tab) string {
	if tab.Kind == KindDLA {
		return "DLA"
	}
	return "PLA"
}
