package inputreqprotection

import (
	"fmt"
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
	// ClaimNumber — "No Klaim" yang DIKETIK pengguna. Wajib.
	//
	// # Tidak ada field kedua untuk ClaimID, dan itu disengaja
	//
	// Sampai 2026-09-24 ada `ClaimReference` di sini, mengisi `ID_CLAIM`. Work Owner
	// menegaskan **keduanya kini berisi nilai yang sama**, yaitu `PNCN.YY.xxxx`.
	//
	// Dua isian yang WAJIB sama tetapi diketik terpisah akan berbeda cepat atau lambat, dan
	// perbedaannya tidak menghasilkan galat apa pun — hanya proteksi yang menunjuk dua
	// klaim berbeda. Karena itu ClaimID DITURUNKAN dari field ini di adapter, bukan
	// ditanyakan kedua kalinya.
	ClaimNumber string

	// Type — "Tipe Proteksi". Wajib.
	Type string

	// Note — "Keterangan". Wajib.
	Note string

	// Change memuat isian yang BENAR-BENAR DIPILIH pengguna pada panel Detail Perubahan.
	//
	// Nilai "sebelum" TIDAK ada di sini — lihat ChangeRequest.
	Change ChangeRequest
}

// ChangeRequest adalah bagian panel Detail Perubahan yang DIPILIH pengguna.
//
// # Kenapa hanya nilai "sesudah"
//
// Panel di Pega punya dua sisi, dan keduanya berasal dari tempat yang berbeda:
//
//	Current Date Of Loss   dari KLAIM   — `.ClaimDataProtect.BeforeDateOfLoss`
//	Next Date Of Loss      dari PENGGUNA
//	Cause Of Loss Dipilih  dari KLAIM
//	Next Cause Of Loss     dari PENGGUNA
//
// Sisi kiri disalin `Activity/OpenProtection-Act.xml` dari klaim yang ditemukan, dan ditimpa
// setiap kali klaim dicari ulang. Menerimanya dari klien berarti mempercayai pengirim untuk
// menyatakan keadaan klaim yang bukan miliknya.
//
// Akibatnya bukan sekadar merepotkan: permintaan "ubah DOL" dapat menyebut DOL sebelum yang
// tidak pernah menjadi DOL klaim itu, dan petugas akseptasi menyetujuinya tanpa cara
// mengetahuinya. Karena itu nilai "sebelum" diturunkan ULANG di server saat menyimpan —
// bukan sekadar dikunci di layar.
type ChangeRequest struct {
	// LossDateAfter — "Next Date Of Loss". Wajib untuk Type '7'.
	//
	// Pointer supaya "tidak diisi" dapat dibedakan dari "tanggal nol".
	LossDateAfter *time.Time

	// CauseOfLossAfter — "Next Cause Of Loss". Wajib untuk Type '8'.
	CauseOfLossAfter string
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
	d.ClaimNumber = strings.TrimSpace(d.ClaimNumber)
	d.Type = strings.TrimSpace(d.Type)
	d.Note = strings.TrimSpace(d.Note)

	d.Change.CauseOfLossAfter = strings.TrimSpace(d.Change.CauseOfLossAfter)

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
	FieldObjectName     = "nama_objek"
	FieldBranchName     = "nama_cabang"
)

// Batas panjang tiap isian, MENGIKUTI lebar kolom `POOLDATA.T_CLAIM_OPENPROTECTION`.
//
// # Kenapa dijaga di sini, padahal basis data juga menjaganya
//
// Yang dijaga bukan datanya melainkan PESANNYA. Tanpa pemeriksaan ini, isian yang terlalu
// panjang ditolak Oracle dengan `ORA-12899: value too large for column` — kalimat berbahasa
// Inggris yang menyebut nama kolom basis data, muncul SETELAH tombol simpan ditekan, dan
// tidak menunjuk field mana pun di layar.
//
// # Satu batas menyimpang dari kolomnya, dan itu disengaja
//
// `CLAIM_NO` hari ini `VARCHAR2(10)`, sedangkan nomor klaim `PNCN.YY.xxxx` (`D-71`) butuh
// dua belas. `MaxClaimNumberLength` di bawah memakai lebar SASARAN, **32**, yang ditetapkan
// Work Owner pada 2026-09-24 setelah keempat pilihan dihitung:
//
//	 8  PNC-1865        warisan, 118 dari 160 baris produksi
//	12  PNCN.26.0001    nomor klaim baru
//	16  baris warisan terpanjang di produksi, bukan pola PNC-
//	20  akan menyamakannya dengan OPEN_PROTECTION_ID, yang bentuk nomornya identik
//
// Enam belas sudah memuat seluruh data yang ada, dan dua puluh akan memberi nomor klaim
// ruang tumbuh yang sama dengan nomor proteksi. Work Owner tetap memilih 32; konsekuensinya
// diterima dan tidak merusak apa pun — hanya ruang yang tidak akan terpakai.
//
// Memakai SEPULUH di sini akan menolak seluruh nomor klaim sistem baru dengan pesan yang
// seolah-olah menyalahkan pengguna, padahal yang salah adalah lebar kolomnya. Dengan lebar
// sasaran, kegagalannya muncul sebagai `ORA-12899` yang menyebut nama kolom — dan itu memang
// tujuannya: ia menunjuk tempat yang benar.
//
// Pelebarannya menunggu DBA; pernyataan `ALTER`-nya di `docs/kolom-open-protection.md` §1.
const (
	MaxPolicyNumberLength = 20 // POLICY_NO

	// MaxClaimNumberLength mengikuti lebar SASARAN CLAIM_NO, bukan lebar hari ini.
	// Lihat catatan di atas.
	MaxClaimNumberLength = 32

	MaxTypeLength       = 2    // PROTECTION_TYPE_ID
	MaxNoteLength       = 2000 // NOTES
	MaxChangeDataLength = 400  // OLD_DATA / NEW_DATA
	MaxObjectNameLength = 500  // OBJECT_NAME
	MaxBranchNameLength = 100  // BRANCH_NAME
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

	// # No Polis TIDAK divalidasi di sini, karena ia bukan isian
	//
	// `Activity/InsertOpenProtectionCase-Act.xml:921` memang mensyaratkan `.PolicyNo != ""`,
	// tetapi `.PolicyNo` diisi `Activity/OpenProtection-Act.xml` dari klaim yang ditemukan —
	// bukan diketik. Syarat itu karena itu terpenuhi oleh DITEMUKANNYA klaim, dan yang
	// menjaganya adalah pencarian, bukan pemeriksaan isian di sini.
	//
	// # Pesan "Silakan Tulis dan Cari Ulang No Klaim" digantikan, bukan dihapus
	//
	// Di Pega pesan itu menegur pengguna yang mengetik nomor tanpa menekan tombol CARI. Di
	// sini pencariannya berjalan sendiri, sehingga keadaan itu tidak dapat terjadi.
	//
	// Yang menggantikannya adalah `ErrClaimNotFound`: nomornya tidak menunjuk klaim mana
	// pun. Pesannya berbeda karena SEBABNYA berbeda — bukan karena teksnya diterjemahkan
	// ulang.
	if d.ClaimNumber == "" {
		v.Add(FieldClaimNumber, "No Klaim wajib diisi.")
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
		if d.Change.LossDateAfter == nil {
			v.Add(FieldLossDateAfter, "Next Date Of Loss wajib diisi untuk permintaan perubahan DOL.")
		}
	case TypeChangeCauseOfLoss:
		// HANYA "Next Cause Of Loss" yang divalidasi di sini.
		//
		// "Cause Of Loss Dipilih" bertanda wajib di `:17827`, tetapi ia DITURUNKAN dari
		// klaim — bukan diisi pengguna. Memvalidasinya di sini akan menolak isian yang
		// pengguna tidak punya cara memperbaikinya; yang menjaganya adalah keberadaan
		// klaimnya, diperiksa saat pencarian.
		if d.Change.CauseOfLossAfter == "" {
			v.Add(FieldCauseOfLossMst, "Next Cause Of Loss wajib dipilih.")
		}
	}

	d.checkLengths(v)

	return v.OrNil()
}

// checkLengths memeriksa setiap isian terhadap lebar kolomnya.
//
// # Dihitung sebagai RUNE, bukan sebagai byte
//
// Kolomnya `VARCHAR2(n BYTE)`, sehingga Oracle menghitung BYTE — dan satu huruf beraksen
// memakan dua byte di UTF-8. Menghitung rune di sini karena itu LEBIH LONGGAR daripada
// Oracle, dan teks beraksen sepanjang batas masih dapat ditolak basis data.
//
// Itu pilihan yang disengaja: yang dilihat pengguna adalah jumlah huruf yang ia ketik,
// bukan jumlah byte. Pesan "maksimal 20 karakter" yang menolak masukan 15 huruf akan
// tampak seperti kerusakan. Keterangan berbahasa Indonesia — satu-satunya field panjang —
// nyaris tidak pernah beraksen, sehingga selisihnya praktis tidak pernah terpakai.
func (d Draft) checkLengths(v *ValidationError) {
	batas := []struct {
		field string
		label string
		nilai string
		max   int
	}{
		{FieldClaimNumber, "No Klaim", d.ClaimNumber, MaxClaimNumberLength},
		{FieldType, "Tipe Proteksi", d.Type, MaxTypeLength},
		{FieldNote, "Keterangan", d.Note, MaxNoteLength},
		{FieldCauseOfLossMst, "Next Cause Of Loss", d.Change.CauseOfLossAfter, MaxChangeDataLength},
	}

	// Nilai TURUNAN tidak diperiksa di sini — No Polis, Object Name, Branch Name, dan kedua
	// nilai "sebelum" berasal dari klaim, bukan dari isian. Yang menjaganya adalah lebar
	// kolom pada tabel klaim itu sendiri; memeriksanya lagi di sini akan menolak data yang
	// pengguna tidak punya cara memperbaikinya.

	for _, b := range batas {
		if n := len([]rune(b.nilai)); n > b.max {
			v.Add(b.field, fmt.Sprintf(
				"%s terlalu panjang: %d karakter, maksimal %d.", b.label, n, b.max))
		}
	}
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
// Nomor polis diserahkan TERPISAH dari isian, karena ia bukan isian: ia datang dari klaim
// yang ditemukan. Menerimanya lewat Draft akan mengembalikan persoalan yang baru saja
// ditutup — pemanggil menyatakan sendiri polis milik siapa proteksi ini.
func DuplicateKeyFor(
	policyNumber string,
	d Draft,
	at time.Time,
	location *time.Location,
) DuplicateKey {
	if location == nil {
		location = time.UTC
	}
	local := at.In(location)

	return DuplicateKey{
		PolicyNumber: strings.TrimSpace(policyNumber),
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
