// Package casestudyclaimhttp adalah lapisan transport modul Case Study Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4). Memakai
// tipe domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien, dan
// sebaliknya — penggantian nama field di domain menjadi perubahan yang merusak antarmuka.
package casestudyclaimhttp

import (
	"time"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
)

// rowDTO adalah satu baris grid.
//
// # Nama field berbahasa Indonesia
//
// Ia KONTRAK yang dibaca frontend, dan termasuk lima pengecualian `D-80` bersama komentar,
// nama kolom basis data, teks layar, dan variabel lingkungan. Yang berbahasa Inggris
// hanyalah nama tipe dan field Go-nya.
//
// Nama field-nya sama persis dengan `Column.Key` pada `casestudyclaim.Columns()`. Itu
// bukan kebetulan: layar menyusun kolomnya dari daftar yang dikirim server lalu mengambil
// isinya dari baris menurut kunci yang sama. Mengganti salah satunya tanpa yang lain
// membuat kolomnya tampil kosong TANPA galat.
type rowDTO struct {
	NomorKlaim      string `json:"nomor_klaim"`
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`
	COB             string `json:"cob"`
	PeriodePolis    string `json:"periode_polis"`
	BulanKlaim      string `json:"bulan_klaim"`

	// TanggalKejadian dikirim sebagai teks ISO-8601 tanggal saja (YYYY-MM-DD), bukan
	// timestamp.
	//
	// Kolom yang ditampilkan adalah TANGGAL, dan mengirim timestamp UTC memaksa setiap
	// tempat di frontend memutuskan sendiri cara mengubahnya ke WIB — yaitu cara paling
	// mudah membuat tanggal bergeser satu hari tanpa ada yang menyadarinya (`R-12`).
	TanggalKejadian string `json:"tanggal_kejadian"`

	SOB              string `json:"sob"`
	PosisiReasuransi string `json:"posisi_reasuransi"`

	// Kedua field berikut SELALU berisi nilai yang sama.
	//
	// Di Pega kedua kolomnya terikat properti yang sama (`.Country`), dan kuerinya hanya
	// menyediakan satu nilai (`A.COL_DESC`). Keduanya tetap dikirim terpisah supaya grid
	// baru punya kolom yang sama persis dengan grid lama; layar menyatakan kekembarannya
	// apa adanya.
	NatureOfLoss string `json:"nature_of_loss"`
	CauseOfLoss  string `json:"cause_of_loss"`

	// Sembilan nilai berikut bertipe PENUNJUK, dan `null` BERBEDA dari nol.
	//
	// `SUM` atas himpunan kosong mengembalikan NULL. Layar menggambar `null` sebagai tanda
	// hubung dan berkas CSV sebagai sel kosong — bukan sebagai "Rp 0,00", yang pada
	// pengolah angka ikut terhitung dalam rata-rata dan penjumlahan.
	//
	// Satuannya SEN, yaitu rupiah dikali 100, dan bertipe bilangan bulat. Pecahan biner
	// tidak dapat mewakili rupiah dengan tepat — `0.1 + 0.2 !== 0.3` berlaku di JavaScript
	// persis seperti di Go — sehingga pembulatan hanya terjadi saat digambar (`I-12`).
	TSI                   *int64 `json:"tsi_100"`
	ShareASM              *int64 `json:"share_asm"` // persentase dikali 10.000 (`D-51`)
	Deductible            *int64 `json:"deductible"`
	NilaiShareASM         *int64 `json:"nilai_share_asm"`
	NilaiKlaim100         *int64 `json:"nilai_klaim_100"`
	AdjusterFee100        *int64 `json:"adjuster_fee_100"`
	NilaiKlaimNet100      *int64 `json:"nilai_klaim_net_100"`
	NilaiKlaimNetShareASM *int64 `json:"nilai_klaim_net_share_asm"`
	LackOfDoc             *int64 `json:"lack_of_doc"`

	Cabang      string `json:"cabang"`
	StatusKlaim string `json:"status_klaim"`
	Kronologi   string `json:"kronologi"`

	// Remark adalah satu-satunya isian yang dapat disunting di seluruh layar.
	Remark string `json:"remark"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Baris []rowDTO `json:"baris"`
	Total int      `json:"total"`

	// Periode menyebutkan rentang TAHUN yang benar-benar dipakai menyaring.
	//
	// Ia ada supaya layar dapat MENYATAKANNYA. Pengguna memilih dua tanggal, tetapi yang
	// dipakai hanya tahunnya — dan tanpa disebutkan, daftar yang memuat klaim di luar
	// bulan yang dipilih akan terbaca sebagai penyaring yang rusak.
	Periode periodDTO `json:"periode"`
}

// periodDTO adalah rentang tahun yang dipakai menyaring.
type periodDTO struct {
	TahunAwal  string `json:"tahun_awal"`
	TahunAkhir string `json:"tahun_akhir"`
}

// optionDTO adalah satu pilihan dropdown.
type optionDTO struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`
}

// columnDTO adalah satu kolom grid.
type columnDTO struct {
	Kunci string `json:"kunci"`
	Judul string `json:"judul"`
	Jenis string `json:"jenis"`
}

// metadataResponse adalah keterangan layar: isi kedua dropdown dan susunan kolomnya.
//
// # Kenapa bentuk layar datang dari server
//
// Karena ketiga daftarnya adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
// tercatat adalah backend — `filter.go` untuk kedua dropdown, `columns.go` untuk kolom.
// Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
// tertinggal saat yang lain diperbaiki.
type metadataResponse struct {
	Bisnis []optionDTO `json:"bisnis"`
	Status []optionDTO `json:"status"`
	Kolom  []columnDTO `json:"kolom"`

	// AmbangNilaiKlaim adalah ambang yang membuat sebuah klaim masuk layar ini, dalam sen.
	//
	// Ia DIKIRIM ke layar, dan itu disengaja: layar menyebutkannya kepada pengguna,
	// sehingga "kenapa klaim saya tidak muncul" terjawab tanpa membuka kode. Begitu master
	// ambang (`F-4`) tiba dan angkanya dapat diubah tanpa deploy, layar ikut berubah tanpa
	// satu baris pun disunting.
	AmbangNilaiKlaim int64 `json:"ambang_nilai_klaim"`

	// CatatanPeriode menjelaskan bahwa hanya TAHUN dari kedua tanggal yang dipakai.
	//
	// Datang dari server, bukan ditulis tetap di layar, supaya penjelasan dan penyaringnya
	// tidak dapat menyimpang: keduanya berubah di satu tempat.
	CatatanPeriode string `json:"catatan_periode"`

	// CatatanKolomKembar menjelaskan kenapa "Nature of Loss" dan "Cause of Loss" selalu
	// sama.
	CatatanKolomKembar string `json:"catatan_kolom_kembar"`
}

// saveRemarkRequest adalah badan permintaan penyimpanan catatan telaah.
type saveRemarkRequest struct {
	// Catatan BOLEH kosong — mengosongkan kembali catatan yang salah ketik adalah hal yang
	// wajar, dan kolomnya nullable.
	Catatan string `json:"catatan"`
}

// saveRemarkResponse mengembalikan catatan sebagaimana tersimpan.
//
// Ia dikembalikan, bukan sekadar 204: layar memangkas spasi di ujung sebelum menyimpan,
// dan pengguna berhak melihat apa yang benar-benar tersimpan tanpa memuat ulang daftarnya.
type saveRemarkResponse struct {
	NomorKlaim string `json:"nomor_klaim"`
	Catatan    string `json:"catatan"`
}

// toRowDTO mengubah satu baris domain menjadi bentuk kiriman.
func toRowDTO(row casestudyclaim.CaseStudyRow, loc *time.Location) rowDTO {
	return rowDTO{
		NomorKlaim:      row.ClaimNumber,
		NomorPolis:      row.PolicyNumber,
		NamaTertanggung: row.InsuredName,
		COB:             row.BusinessName,
		PeriodePolis:    row.PolicyPeriod,
		BulanKlaim:      row.ClaimMonth,
		TanggalKejadian: formatDate(row.LossDate, loc),
		SOB:             row.BusinessSource,

		PosisiReasuransi: row.ReinsurerRole,

		// Satu nilai, dua kolom. Lihat catatan pada field-nya.
		NatureOfLoss: row.CauseOfLoss,
		CauseOfLoss:  row.CauseOfLoss,

		TSI:                   minorUnits(row.TSI),
		ShareASM:              row.ASMSharePercent,
		Deductible:            minorUnits(row.Deductible),
		NilaiShareASM:         minorUnits(row.ASMShareValue),
		NilaiKlaim100:         minorUnits(row.ClaimValue100),
		AdjusterFee100:        minorUnits(row.AdjusterFee),
		NilaiKlaimNet100:      minorUnits(row.NetClaim100),
		NilaiKlaimNetShareASM: minorUnits(row.NetClaimASM),
		LackOfDoc:             minorUnits(row.LackOfDoc),

		Cabang:      row.BranchName,
		StatusKlaim: row.ClaimStatus,
		Kronologi:   row.Chronology,
		Remark:      row.Remark,
	}
}

// minorUnits mengubah nilai uang menjadi bilangan bulat sen; nil tetap nil.
func minorUnits(amount *money.Money) *int64 {
	if amount == nil {
		return nil
	}
	value := amount.MinorUnits()
	return &value
}

// formatDate mengubah waktu UTC menjadi tanggal WIB berbentuk YYYY-MM-DD.
//
// Inilah SATU-SATUNYA tempat konversi zona waktu terjadi pada baris grid. Penyimpanan
// memakai UTC dan tampilan memakai WIB; menyebar konversinya akan mengulang persis cacat
// `Set7Hours` sistem lama, yang menambah tujuh jam manual di puluhan tempat sehingga satu
// tempat yang lupa menggeser tanggal tanpa terdeteksi.
func formatDate(at *time.Time, loc *time.Location) string {
	if at == nil || at.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return at.In(loc).Format("2006-01-02")
}
