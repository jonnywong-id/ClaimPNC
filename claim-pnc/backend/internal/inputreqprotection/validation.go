package inputreqprotection

import (
	"strings"
	"time"
)

// Draft adalah isian form Input Open Protection sebelum disimpan.
//
// Ia dipisahkan dari Protection dengan sengaja: Protection memuat hal-hal yang DITERBITKAN
// sistem — nomor, waktu pembuatan, pembuatnya — dan tidak satu pun boleh datang dari
// klien. Menerima Protection utuh dari badan permintaan akan membuat pemanggil dapat
// menentukan nomor proteksinya sendiri.
//
// Field di sini mengikuti `Section/InputProtectionSection-Section.xml`.
type Draft struct {
	// PolicyNumber — "No Polis". Wajib.
	PolicyNumber string

	// ClaimNumber — "No Klaim" yang DIKETIK pengguna. Wajib.
	ClaimNumber string

	// ClaimReference adalah klaim yang benar-benar DITEMUKAN saat pengguna menekan cari.
	//
	// Wajib, dan ketiadaannya punya pesan tersendiri di sistem lama. Lihat Validate.
	ClaimReference string

	// Type — "Tipe Proteksi". Wajib.
	Type string

	// Note — "Keterangan". Wajib.
	Note string

	// ChangeDetail wajib terisi untuk Type '7' dan '8'.
	ChangeDetail ChangeDetail
}

// Normalize merapikan isian sebelum divalidasi maupun disimpan.
//
// # Yang SENGAJA TIDAK dilakukan: membuang titik dari nomor klaim
//
// `Activity/ValidationInputProtection-Act.xml:288` menjalankan
//
//	.CaseID := @replaceAll(.CaseID, ".", "")
//
// yaitu membuang SELURUH titik dari nomor klaim sebelum menyimpannya, tanpa syarat.
//
// Itu tidak berbahaya ketika seluruh nomor klaim berbentuk `PNC-1865` — tidak ada titik di
// dalamnya, sehingga langkah itu tidak mengubah apa pun. Tetapi `D-71` menetapkan nomor
// klaim sistem baru berbentuk **`PNCN.YY.xxxx`**, yang justru dipisahkan titik. Terhadap
// nomor itu, langkah yang sama menghasilkan `PNCN260001` — dan tautan proteksi ke klaimnya
// PUTUS tanpa satu pun galat.
//
// Kegagalan seperti itu tidak punya gejala: penyimpanan berhasil, layar tampil normal,
// hanya nomor klaimnya yang tidak cocok dengan klaim mana pun. Karena itu langkah ini
// tidak dibawa, dan ketidakhadirannya adalah SELISIH TERENCANA yang harus dinyatakan di
// muka saat uji kesetaraan dijalankan (`P-5`).
//
// Yang dikerjakan sebagai gantinya hanyalah membuang spasi di ujung — yang memang
// dibutuhkan, karena nomor yang disalin-tempel kerap membawa spasi.
func (d Draft) Normalize() Draft {
	d.PolicyNumber = strings.TrimSpace(d.PolicyNumber)
	d.ClaimNumber = strings.TrimSpace(d.ClaimNumber)
	d.ClaimReference = strings.TrimSpace(d.ClaimReference)
	d.Type = strings.TrimSpace(d.Type)
	d.Note = strings.TrimSpace(d.Note)

	d.ChangeDetail.CauseOfLossID = strings.TrimSpace(d.ChangeDetail.CauseOfLossID)
	d.ChangeDetail.CauseOfLossMasterID = strings.TrimSpace(d.ChangeDetail.CauseOfLossMasterID)
	d.ChangeDetail.ObjectName = strings.TrimSpace(d.ChangeDetail.ObjectName)
	d.ChangeDetail.BranchName = strings.TrimSpace(d.ChangeDetail.BranchName)

	return d
}

// Nama field pada kontrak API. Dikumpulkan sebagai konstanta supaya pesan validasi dan
// bentuk JSON tidak dapat berbeda diam-diam.
const (
	FieldPolicyNumber   = "no_polis"
	FieldClaimNumber    = "no_klaim"
	FieldType           = "tipe_proteksi"
	FieldNote           = "keterangan"
	FieldLossDateAfter  = "dol_baru"
	FieldCauseOfLoss    = "penyebab_kerugian"
	FieldCauseOfLossMst = "penyebab_kerugian_master"
)

// Validate memeriksa seluruh aturan yang dapat diperiksa TANPA menyentuh penyimpanan.
//
// Aturan yang menuntut pembacaan — proteksi ganda pada hari yang sama — tidak di sini;
// lihat CheckDuplicate.
//
// # Empat field wajib, dan asalnya
//
// `Activity/InsertOpenProtectionCase-Act.xml:921` memasang prakondisi
//
//	.TypeProtection != "" && .Keterangan != "" && .CaseID != "" && .PolicyNo != ""
//
// Keempatnya karena itu wajib, dan urutan pemeriksaannya di sini mengikuti urutan field di
// layar — bukan urutan pada prakondisi — supaya pesan yang muncul terbaca dari atas ke
// bawah sesuai yang dilihat pengguna.
func (d Draft) Validate() error {
	v := &ValidationError{}
	d = d.Normalize()

	if d.PolicyNumber == "" {
		v.Add(FieldPolicyNumber, "No Polis wajib diisi.")
	}

	// Nomor klaim punya DUA aturan, dan keduanya memberi pesan berbeda.
	//
	// Yang kedua disalin apa adanya dari `ValidationInputProtection-Act.xml`, termasuk
	// bunyinya: "Silakan Tulis dan Cari Ulang No Klaim". Mengetik nomor klaim tidak cukup —
	// klaimnya harus benar-benar ditemukan, dan hasil pencarian itulah yang disimpan
	// sebagai ClaimReference. Tanpa pembedaan ini, pengguna yang salah ketik akan melihat
	// pesan "wajib diisi" pada kolom yang jelas-jelas sudah ia isi.
	switch {
	case d.ClaimNumber == "":
		v.Add(FieldClaimNumber, "No Klaim wajib diisi.")
	case d.ClaimReference == "":
		v.Add(FieldClaimNumber, "Silakan Tulis dan Cari Ulang No Klaim.")
	}

	if d.Type == "" {
		v.Add(FieldType, "Tipe Proteksi wajib dipilih.")
	}
	if d.Note == "" {
		v.Add(FieldNote, "Keterangan wajib diisi.")
	}

	// Panel detail perubahan hanya muncul untuk dua tipe, dan isinya wajib ketika muncul.
	// Untuk tipe lain, isian yang kebetulan terkirim DIABAIKAN — bukan ditolak: layar tidak
	// menampilkan panelnya, sehingga pengguna tidak punya cara memperbaikinya.
	switch d.Type {
	case TypeChangeLossDate:
		// `Section/InputProtectionSection-Section.xml:8051` menandai Next Date Of Loss
		// sebagai wajib. Current Date Of Loss TIDAK wajib — ia keadaan sebelumnya, yang
		// boleh saja belum tercatat.
		if d.ChangeDetail.LossDateAfter == nil {
			v.Add(FieldLossDateAfter, "Next Date Of Loss wajib diisi untuk permintaan perubahan DOL.")
		}
	case TypeChangeCauseOfLoss:
		// Keduanya bertanda wajib di `:18459` dan `:17827`.
		if d.ChangeDetail.CauseOfLossID == "" {
			v.Add(FieldCauseOfLoss, "Cause Of Loss wajib dipilih untuk permintaan perubahan Cause Of Loss.")
		}
		if d.ChangeDetail.CauseOfLossMasterID == "" {
			v.Add(FieldCauseOfLossMst, "Cause Of Loss yang dituju wajib dipilih.")
		}
	}

	return v.OrNil()
}

// DuplicateKey adalah kunci pemeriksaan proteksi ganda.
//
// # Aturannya dari sistem lama, beserta bunyinya
//
// `Activity/ValidationInputProtection-Act.xml` menolak penyimpanan dengan pesan
//
//	"Sudah ada Open Protection dengan no polis dan tipe proteksi yang sama di hari ini"
//
// Pemeriksaannya menempuh Report Definition `BrowseCaseOPCList` dengan tiga parameter:
// `policyno`, `tipe`, dan `inputdate` yang diisi `@CurrentDate("dd MMM yyyy","WIB")`.
//
// # Kenapa TANGGAL, bukan rentang waktu
//
// Parameter tanggalnya diformat sebagai TANGGAL SAJA dalam zona WIB, sehingga "hari ini"
// berarti hari kalender WIB — bukan 24 jam terakhir. Dua permintaan pada polis dan tipe
// yang sama pukul 23.50 dan 00.10 adalah dua hari berbeda, dan keduanya sah.
//
// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4), sehingga konversinya harus terjadi
// sebelum perbandingan. Tanpa itu, seluruh permintaan antara pukul 00.00 dan 07.00 WIB
// akan dibandingkan terhadap hari sebelumnya.
type DuplicateKey struct {
	PolicyNumber string
	Type         string

	// Day adalah tanggal kalender WIB, bukan timestamp.
	Day time.Time
}

// DuplicateKeyFor menyusun kunci pemeriksaan ganda untuk sebuah isian pada satu waktu.
//
// Zona waktunya diserahkan pemanggil, bukan dibaca dari jam sistem, supaya aturan "hari
// ini" dapat diuji tanpa bergantung pada basis data zona waktu mesin yang menjalankan.
func DuplicateKeyFor(d Draft, at time.Time, location *time.Location) DuplicateKey {
	if location == nil {
		location = time.UTC
	}
	local := at.In(location)

	return DuplicateKey{
		PolicyNumber: strings.TrimSpace(d.PolicyNumber),
		Type:         strings.TrimSpace(d.Type),
		Day:          time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location),
	}
}

// DuplicateMessage adalah pesan yang dilihat pengguna saat proteksi ganda ditolak.
//
// Disalin APA ADANYA dari sistem lama, termasuk ejaannya. `D-13` menetapkan teks yang
// dilihat pengguna mengikuti layar Pega, dan pesan ini sudah dikenal petugas yang
// memakainya setiap hari.
const DuplicateMessage = "Sudah ada Open Protection dengan no polis dan tipe proteksi yang sama di hari ini"
