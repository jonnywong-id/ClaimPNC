// Package dashboardclaimhttp melayani layar Dashboard Claim lewat HTTP.
//
// Nama paketnya sengaja berbeda dari nama foldernya — folder `http`, paket
// `dashboardclaimhttp` — supaya ia tidak menutupi `net/http` di berkas yang mengimpor
// keduanya. Konvensi yang sama dipakai seluruh modul di sini.
//
// # Nama kunci JSON berbahasa Indonesia
//
// Ia KONTRAK yang dibaca frontend, bukan nama internal (`D-80`). Mengubahnya adalah
// perubahan yang merusak klien, bukan penggantian nama.
package dashboardclaimhttp

import (
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// cardDTO adalah satu kartu penghitung.
//
// Keempatnya dikirim sebagai LARIK, bukan sebagai empat field bernama, supaya URUTANNYA
// ditentukan server. Urutan kartu di layar ini bukan selera tata letak melainkan urutan yang
// sudah dikenal pengguna — `param.tipe` 0…3 pada sistem lama (`D-13`). Membiarkan frontend
// menyusunnya sendiri berarti urutan itu dapat berubah tanpa ada yang memutuskannya.
type cardDTO struct {
	Tile   string `json:"tile"`
	Judul  string `json:"judul"`
	Jumlah int    `json:"jumlah"`

	// Bentuk memberi tahu layar kolom mana yang digambar saat kartu ini ditelusuri.
	//
	// Ia dikirim server, bukan disimpulkan frontend dari nama tile: menebaknya dari nama
	// membuat penambahan tile kelak diam-diam salah gambar.
	Bentuk string `json:"bentuk"`
}

// summaryResponse adalah jawaban GET /dashboard-claim/ringkasan.
type summaryResponse struct {
	Kartu []cardDTO `json:"kartu"`

	// LiniBisnis adalah penyaring yang MENGHASILKAN keempat angka itu.
	//
	// Ia ikut dikirim supaya layar dapat menyatakannya. Empat angka nol punya dua sebab yang
	// tampak sama: memang tidak ada pekerjaan, atau penyaring mempersempitnya sampai habis.
	LiniBisnis string `json:"lini_bisnis"`

	// Portal ikut dikirim di setiap respons modul ini, sama seperti modul lain.
	//
	// Layar menampilkannya supaya pengguna yang memegang kewenangan di lebih dari satu
	// entitas dapat MELIHAT entitas mana yang sedang dibacanya. Angka ringkasan dari badan
	// hukum yang keliru tidak terlihat salah dengan cara lain apa pun (`R-20`).
	Portal string `json:"portal"`

	// SelisihTerencana menyebutkan perbedaan yang DISENGAJA terhadap sistem lama.
	//
	// Ia dikirim ke layar, bukan hanya ditulis di komentar kode, supaya penguji gerbang 1
	// membacanya saat membandingkan angka — bukan menemukannya sebagai kejutan lalu
	// melaporkannya sebagai cacat (`D-54`).
	SelisihTerencana []string `json:"selisih_terencana"`

	// CatatanWarisan menyebutkan perilaku yang SAMA dengan sistem lama tetapi mudah dibaca
	// sebagai cacat.
	//
	// Dipisahkan dari SelisihTerencana dengan sengaja. Yang di sana BERBEDA dari Pega dan
	// menuntut persetujuan; yang di sini SAMA dengan Pega dan hanya menuntut penjelasan.
	// Menggabungkan keduanya akan membuat penguji gerbang 1 mencari selisih yang tidak ada.
	CatatanWarisan []string `json:"catatan_warisan"`
}

// claimDTO adalah satu baris telusur bertipe klaim.
//
// Kolomnya mengikuti `Section/DashboardClaim_Section2-Section.xml`: No Klaim, No Polis, Nama
// Tertanggung, Nama Bisnis, Sumber Bisnis, Nama Cabang — ditambah PIC Teknik, Admin PNC, dan
// status yang dipakai layar sebagai keterangan baris.
type claimDTO struct {
	// KlaimID adalah kunci teknis Pega, dipakai membuka layar detail.
	//
	// Ia TIDAK digambar sebagai teks: isinya memuat nama kelas internal Pega yang bocor ke
	// data bisnis (utang teknis §4.1, dijawab `D-22`).
	KlaimID string `json:"klaim_id"`

	NomorKlaim   string `json:"nomor_klaim"`
	NomorPolis   string `json:"nomor_polis"`
	Tertanggung  string `json:"nama_tertanggung"`
	NamaBisnis   string `json:"nama_bisnis"`
	SumberBisnis string `json:"sumber_bisnis"`
	NamaCabang   string `json:"nama_cabang"`
	PICTeknik    string `json:"pic_teknik"`
	AdminPNC     string `json:"admin_pnc"`

	TanggalRegister string `json:"tanggal_register"`
	TanggalKejadian string `json:"tanggal_kejadian"`

	// StatusKlaimKode adalah KODE, bukan artinya.
	//
	// Artinya tidak disimpulkan di sini. Tiga arti kode yang pernah disimpulkan dari
	// pemakaiannya seluruhnya terbukti salah saat master diterima (`R-06`); pelabelannya
	// adalah urusan master status klaim.
	StatusKlaimKode string `json:"status_klaim_kode"`
	StatusProses    string `json:"status_proses"`
}

// surveyDTO adalah satu baris telusur bertipe survei.
//
// Kolomnya mengikuti `Section/DashboardClaim_Section1-Section.xml`: No Klaim, No Polis, Nama
// Tertanggung, Surveyor, Adjuster, Tanggal Survey, Status Survey, Status Register.
type surveyDTO struct {
	SurveiID string `json:"survei_id"`

	NomorSurvei    string `json:"nomor_survei"`
	NomorKlaim     string `json:"nomor_klaim"`
	NomorPolis     string `json:"nomor_polis"`
	Tertanggung    string `json:"nama_tertanggung"`
	NomorReferensi string `json:"nomor_referensi"`
	NamaSurveyor   string `json:"nama_surveyor"`
	PICTeknik      string `json:"pic_teknik"`
	PICAdjuster    string `json:"pic_adjuster"`
	LokasiSurvei   string `json:"lokasi_survei"`

	// TanggalSurvei hanya terisi pada tile Internal Surveyor.
	//
	// Kueri loss adjuster memang tidak mengambilnya, dan membiarkannya kosong di sana lebih
	// jujur daripada mengisinya dengan tanggal lain yang kebetulan ada.
	TanggalSurvei string `json:"tanggal_survei"`
	TanggalTugas  string `json:"tanggal_tugas"`

	StatusSurvei  string `json:"status_survei"`
	StatusProses  string `json:"status_proses"`
}

// listResponse adalah jawaban GET /dashboard-claim/{tile}.
//
// Hanya SATU dari Klaim dan Survei yang terisi, ditentukan Bentuk. Keduanya dinyatakan
// sebagai larik yang selalu ada — tidak pernah null — supaya frontend dapat memetakannya
// tanpa memeriksa nil lebih dulu.
type listResponse struct {
	Tile   string `json:"tile"`
	Judul  string `json:"judul"`
	Bentuk string `json:"bentuk"`

	Klaim  []claimDTO  `json:"klaim"`
	Survei []surveyDTO `json:"survei"`

	Halaman paginationDTO `json:"halaman"`

	LiniBisnis string `json:"lini_bisnis"`
	Portal     string `json:"portal"`
}

// paginationDTO adalah keterangan halaman.
//
// Bentuknya mengikuti modul lain — `halaman`, `ukuran`, `total`, `total_halaman` — supaya
// komponen tabel di frontend menerimanya tanpa pemetaan khusus per layar.
//
// Halaman dihitung mulai 1, bukan 0. Offset tidak ikut dikirim: ia bentuk yang dipakai SQL,
// dan mengirimkannya berarti dua cara menyatakan hal yang sama pada satu kontrak.
type paginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// pagination menyusun keterangan halaman dari penyaring yang sudah dinormalkan.
//
// TotalPages dibulatkan KE ATAS: 26 baris dengan ukuran 25 adalah dua halaman, bukan satu.
// Pembulatan ke bawah akan menyembunyikan baris terakhir tanpa satu pun tanda.
func pagination(total, limit, offset int) paginationDTO {
	if limit <= 0 {
		limit = 1
	}
	pages := (total + limit - 1) / limit
	if pages < 1 {
		pages = 1
	}
	return paginationDTO{
		Page:       offset/limit + 1,
		Size:       limit,
		Total:      total,
		TotalPages: pages,
	}
}

// metadataResponse adalah jawaban GET /dashboard-claim/penyaring.
//
// Ia membentuk LAYAR, bukan data: isi dropdown dan daftar kartu. Dipisahkan dari ringkasan
// supaya layar dapat menggambar kerangkanya sebelum angka apa pun selesai dihitung.
type metadataResponse struct {
	LiniBisnis []choiceDTO `json:"lini_bisnis"`
	Tile       []tileDTO   `json:"tile"`
	Portal     string      `json:"portal"`
}

// choiceDTO adalah satu pilihan pada dropdown.
type choiceDTO struct {
	Nilai string `json:"nilai"`
	Label string `json:"label"`
}

// tileDTO adalah keterangan satu kartu, tanpa angkanya.
type tileDTO struct {
	Tile   string `json:"tile"`
	Judul  string `json:"judul"`
	Bentuk string `json:"bentuk"`
}

// adaptClaim memetakan satu baris klaim menjadi DTO.
func adaptClaim(row dashboardclaim.ClaimRow, loc *time.Location) claimDTO {
	return claimDTO{
		KlaimID:         row.ClaimID,
		NomorKlaim:      row.ClaimNumber,
		NomorPolis:      row.PolicyNumber,
		Tertanggung:     row.InsuredName,
		NamaBisnis:      row.BusinessName,
		SumberBisnis:    row.BusinessSource,
		NamaCabang:      row.BranchName,
		PICTeknik:       row.TechnicalPIC,
		AdminPNC:        row.AdminPNC,
		TanggalRegister: formatDate(&row.RegisteredAt, loc),
		TanggalKejadian: formatDate(row.LossDate, loc),
		StatusKlaimKode: row.ClaimStatusCode,
		StatusProses:    row.ProcessStatus,
	}
}

// adaptSurvey memetakan satu baris survei menjadi DTO.
func adaptSurvey(row dashboardclaim.SurveyRow, loc *time.Location) surveyDTO {
	return surveyDTO{
		SurveiID:       row.SurveyID,
		NomorSurvei:    row.SurveyNumber,
		NomorKlaim:     row.ClaimNumber,
		NomorPolis:     row.PolicyNumber,
		Tertanggung:    row.InsuredName,
		NomorReferensi: row.ReferenceNumber,
		NamaSurveyor:   row.SurveyorName,
		PICTeknik:      row.TechnicalPIC,
		PICAdjuster:    row.AdjusterPIC,
		LokasiSurvei:   row.SurveyLocation,
		TanggalSurvei:  formatDate(row.ScheduledAt, loc),
		TanggalTugas:   formatDate(&row.AssignedAt, loc),
		StatusSurvei:   row.SurveyStatus,
		StatusProses:   row.ProcessStatus,
	}
}

// formatDate memformat tanggal menurut zona waktu tampilan.
//
// Waktu disimpan UTC dan dikonversi SATU KALI di sini (`F-5`). Tidak ada penambahan 7 jam
// manual di mana pun — itulah utang teknis §4.4 yang tidak dibawa.
//
// Kosong dan nol keduanya menghasilkan teks kosong: waktu nol akan tampil sebagai 1 Januari
// tahun 1, nilai yang tampak seperti data dan bukan seperti ketiadaan data.
func formatDate(t *time.Time, loc *time.Location) string {
	if t == nil || t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}
