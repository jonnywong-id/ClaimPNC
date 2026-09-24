// Package inboxprogressclaimhttp adalah lapisan transport modul Inbox Progress Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: ClaimRow membawa Positions sebagai
// senarai, sedangkan grid menampilkannya sebagai satu teks bergabung koma.
package inboxprogressclaimhttp

import (
	"strings"
	"time"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/usecase"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// # Kenapa jamnya tidak dikirim, padahal sistem lama mengirimnya
//
// Karena jam yang dikirim sistem lama SALAH. `GET_POSISI_PROGRESS_PNC` memformat tenggat
// dengan `to_char(..., 'DD-MM-YYYY hh:mm:ss')`, dan pada Oracle `mm` di dalam bagian jam
// berarti BULAN — menitnya seharusnya `mi`. Yang selama ini tampil di layar karena itu
// berbunyi `jam:BULAN:detik`.
//
// Tidak ada satu pun kolom di ketiga grid yang membutuhkan jam untuk dibaca, sehingga yang
// dikirim tanggalnya saja. Ini selisih terencana pada uji kesetaraan, bukan cacat.
const dateLayout = "2006-01-02"

// joinSeparator memisahkan nilai antarposisi pada satu sel.
//
// Sistem lama memakai koma tanpa spasi (`temp_posisi || ',' || …`). Spasi ditambahkan di
// sini supaya teks panjang dapat dipotong barisnya oleh peramban; isinya tidak berubah.
const joinSeparator = ", "

// ClaimRowDTO adalah satu baris klaim pada region Outstanding dan Next Follow Up.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya menyebut ISI kolomnya, bukan alias Pega: `no_polis`, bukan
// `claim_no`.
//
// **Judul yang dibaca pengguna tetap memakai alias Pega**, dan itu keputusan terpisah —
// lihat ColumnDTO.Title. Kontrak dan judul memang tidak harus sama: yang satu dibaca mesin,
// yang lain dibaca orang.
type ClaimRowDTO struct {
	ClaimNumber  string  `json:"no_klaim"`
	PolicyNumber string  `json:"no_polis"`
	InsuredName  string  `json:"nama_tertanggung"`
	RegisterDate *string `json:"tanggal_registrasi"`
	LossDate     *string `json:"tanggal_kejadian"`
	LGBNote      string  `json:"catatan_lgb"`
	TechnicalPIC string  `json:"pic_klaim"`

	// Ketiga isian berikut digabung dari SELURUH posisi yang sedang berjalan, dipisah
	// koma — persis seperti keluaran `GET_POSISI_PROGRESS_PNC`.
	//
	// Padanannya positional: nilai ke-n pada `posisi` sepadan dengan nilai ke-n pada
	// `status_progres_1`, `status_progres_2`, dan `next_follow_up`. Itu bentuk sistem
	// lama, dan `jumlah_posisi` dikirim supaya layar dapat menyatakan berapa posisi yang
	// sebenarnya berjalan alih-alih menyuruh pengguna menghitung koma.
	Position        string `json:"posisi"`
	ProgressStatus1 string `json:"status_progres_1"`
	ProgressStatus2 string `json:"status_progres_2"`
	NextFollowUp    string `json:"next_follow_up"`

	// PositionCount adalah banyaknya posisi yang sedang berjalan.
	PositionCount int `json:"jumlah_posisi"`

	EarliestFollowUp *string `json:"follow_up_terawal"`
	ProcessDate      *string `json:"tanggal_proses"`
	ProdKe           string  `json:"prod_ke"`
}

// PICRowDTO adalah satu baris rekap per PIC.
type PICRowDTO struct {
	PIC           string `json:"pic"`
	ClaimCount    int    `json:"jumlah_klaim"`
	UpdateCount   int    `json:"jumlah_pembaruan"`
	DueTodayCount int    `json:"jatuh_tempo_hari_ini"`
	OnTimeCount   int    `json:"tepat_waktu"`
	LateCount     int    `json:"terlambat"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key adalah identitas kolom, unik di dalam satu region.
	//
	// Ia terpisah dari Field karena region Outstanding menggambar SATU isian di DUA kolom.
	Key string `json:"kunci"`

	// Field menyebut isian mana pada baris yang digambar.
	Field string `json:"isian"`

	// Title adalah judul yang dibaca pengguna — alias Pega apa adanya, keputusan Work
	// Owner 2026-09-21.
	Title string `json:"judul"`

	// Description menyatakan isi kolom yang sebenarnya.
	//
	// Ia dikirim KARENA judulnya memakai alias yang menyesatkan. Layar menggambarnya
	// sebagai keterangan kolom, sehingga "District" yang berisi nama tertanggung tetap
	// dapat dipahami tanpa membuka kode.
	Description string `json:"keterangan"`
}

// ViewDTO adalah satu region beserta bentuk gridnya.
type ViewDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	// Kind menyatakan bentuk barisnya: `klaim`, `pic`, atau `kosong`. Layar memilih cara
	// menggambar dari sini.
	Kind string `json:"bentuk"`

	Columns []ColumnDTO `json:"kolom"`

	// Keempat penanda berikut menentukan kontrol mana yang digambar layar. Ia dikirim
	// server, bukan ditentukan layar sendiri, supaya bentuk layar punya satu sumber
	// kebenaran.
	Paginated              bool `json:"pakai_paginasi"`
	SupportsSearch         bool `json:"pakai_pencarian"`
	SupportsDateRange      bool `json:"pakai_rentang_tanggal"`
	SupportsBusinessFilter bool `json:"pakai_lini_bisnis"`
	ScopedToCaller         bool `json:"hanya_milik_saya"`

	// DeadControls adalah kontrol yang digambar pada region ini tetapi tidak menyaring
	// apa pun.
	DeadControls []DeadControlDTO `json:"kontrol_mati"`
}

// DeadControlDTO adalah satu kontrol yang tampil tetapi tidak berpengaruh.
type DeadControlDTO struct {
	Name   string `json:"nama"`
	Reason string `json:"alasan"`
}

// BusinessLineDTO adalah satu pilihan pada dropdown lini bisnis.
type BusinessLineDTO struct {
	Code  string `json:"kode"`
	Label string `json:"label"`
}

// MetadataResponse adalah jawaban GET /api/inbox-progress-claim/bagian.
type MetadataResponse struct {
	Views       []ViewDTO `json:"bagian"`
	DefaultView string    `json:"bagian_bawaan"`

	BusinessLines []BusinessLineDTO `json:"lini_bisnis"`

	// PageSize adalah ukuran halaman bawaan pada region yang dipaginasi.
	PageSize int `json:"ukuran_halaman"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal
	// yang sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`

	// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta
	// alasannya, dalam kalimat yang dapat langsung ditampilkan ke pengguna.
	//
	// Ia dikirim sebagai data, bukan ditulis tetap di layar, supaya ia hilang dengan
	// sendirinya begitu penghalangnya hilang — tanpa menyunting frontend.
	Limitations []string `json:"keterbatasan"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// AppliedFilterDTO adalah penyaring yang BENAR-BENAR dipakai.
//
// Ia dikirim balik karena tidak selalu sama dengan yang diminta: region yang tidak
// mendukung pencarian mengembalikan kata kunci kosong, dan layar dapat membersihkan
// kotaknya alih-alih menampilkan kata kunci yang tampak aktif padahal tidak menyaring apa
// pun.
type AppliedFilterDTO struct {
	Keyword  string  `json:"cari"`
	Business string  `json:"bisnis"`
	From     *string `json:"dari"`
	To       *string `json:"sampai"`
}

// ListResponse adalah jawaban GET /api/inbox-progress-claim.
type ListResponse struct {
	View ViewDTO `json:"bagian"`

	// Rows berisi `[]ClaimRowDTO` atau `[]PICRowDTO`, mengikuti View.Kind.
	//
	// # Kenapa `any`, padahal standar koding melarangnya tanpa alasan tertulis
	//
	// Inilah alasan tertulisnya. Keempat region dilayani SATU endpoint karena layar tidak
	// tahu region mana yang tersedia sampai ia membaca metadata dari server. Bentuk
	// barisnya sendiri memang berbeda — yang satu klaim, yang lain petugas beserta
	// pencacahnya — dan menyatukannya menjadi satu struktur akan menghasilkan tipe yang
	// separuh isiannya selalu kosong.
	//
	// Pilihan lain yang dipertimbangkan dan ditolak: dua endpoint terpisah, yang memaksa
	// layar memilih endpoint sebelum ia tahu region mana yang diminta; dan satu struktur
	// gabungan, yang mengirim enam isian pencacah bernilai nol pada setiap baris klaim.
	//
	// Isinya SELALU berupa senarai, tidak pernah nil — lihat toListResponse.
	Rows any `json:"baris"`

	Pagination PaginationDTO    `json:"paginasi"`
	Filter     AppliedFilterDTO `json:"penyaring"`
	Portal     string           `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toClaimRowDTO mengubah satu baris klaim.
func toClaimRowDTO(item inboxprogressclaim.ClaimRow) ClaimRowDTO {
	return ClaimRowDTO{
		ClaimNumber:  item.ClaimNumber,
		PolicyNumber: item.PolicyNumber,
		InsuredName:  item.InsuredName,
		RegisterDate: toDateString(item.RegisterDate),
		LossDate:     toDateString(item.LossDate),
		LGBNote:      item.LGBNote,
		TechnicalPIC: item.TechnicalPIC,

		Position:        strings.Join(item.PositionNames(), joinSeparator),
		ProgressStatus1: strings.Join(item.Status1List(), joinSeparator),
		ProgressStatus2: strings.Join(item.Status2List(), joinSeparator),
		NextFollowUp:    joinFollowUps(item.Positions),
		PositionCount:   len(item.Positions),

		EarliestFollowUp: toDateString(item.EarliestFollowUp),
		ProcessDate:      toDateString(item.ProcessDate),
		ProdKe:           item.ProdKe,
	}
}

// joinFollowUps menggabungkan tenggat tindak lanjut seluruh posisi.
//
// Posisi yang belum punya tenggat menyumbang teks kosong, bukan dilewati: padanan
// antarkolom bersifat positional, sehingga melewatinya akan menggeser seluruh nilai
// sesudahnya dan memasangkan tenggat sebuah posisi dengan nama posisi yang lain.
func joinFollowUps(positions []inboxprogressclaim.Position) string {
	parts := make([]string, 0, len(positions))
	for _, position := range positions {
		if formatted := toDateString(position.NextFollowUp); formatted != nil {
			parts = append(parts, *formatted)
			continue
		}
		parts = append(parts, "")
	}
	return strings.Join(parts, joinSeparator)
}

// toClaimRowListDTO mengubah satu halaman baris klaim.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toClaimRowListDTO(items []inboxprogressclaim.ClaimRow) []ClaimRowDTO {
	result := make([]ClaimRowDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toClaimRowDTO(item))
	}
	return result
}

// toPICRowListDTO mengubah rekap per PIC.
func toPICRowListDTO(items []inboxprogressclaim.PICSummary) []PICRowDTO {
	result := make([]PICRowDTO, 0, len(items))
	for _, item := range items {
		result = append(result, PICRowDTO{
			PIC:           item.PIC,
			ClaimCount:    item.ClaimCount,
			UpdateCount:   item.UpdateCount,
			DueTodayCount: item.DueTodayCount,
			OnTimeCount:   item.OnTimeCount,
			LateCount:     item.LateCount,
		})
	}
	return result
}

// toViewDTO mengubah satu region.
func toViewDTO(view inboxprogressclaim.View) ViewDTO {
	columns := make([]ColumnDTO, 0, len(view.Columns))
	for _, column := range view.Columns {
		columns = append(columns, ColumnDTO{
			Key:         column.Key,
			Field:       column.Field,
			Title:       column.Title,
			Description: column.Description,
		})
	}

	dead := make([]DeadControlDTO, 0)
	for _, control := range inboxprogressclaim.DeadControlsFor(view.Code) {
		dead = append(dead, DeadControlDTO{Name: control.Name, Reason: control.Reason})
	}

	return ViewDTO{
		Code:                   view.Code,
		Name:                   view.Name,
		Description:            view.Description,
		Kind:                   string(view.Kind),
		Columns:                columns,
		Paginated:              view.Paginated,
		SupportsSearch:         view.SupportsSearch,
		SupportsDateRange:      view.SupportsDateRange,
		SupportsBusinessFilter: view.SupportsBusinessFilter,
		ScopedToCaller:         view.ScopedToCaller,
		DeadControls:           dead,
	}
}

// limitations adalah keterbatasan modul ini yang perlu diketahui pengguna.
//
// Seluruhnya bukan cacat, melainkan akibat keputusan yang sudah diambil atau penghalang
// yang pemiliknya di luar tim ini. Menuliskannya di layar membuat pengguna tidak
// melaporkannya berulang kali sebagai kerusakan.
var limitations = []string{
	"Penyaring Cabang belum aktif, sehingga daftar ini belum dibatasi cabang Anda. " +
		"Di sistem lama ia dibaca lewat sambungan ke basis data HRD yang belum punya " +
		"API pengganti.",
	"Kotak cari menelusuri No Klaim, No Polis, dan nama PIC sekaligus — sama seperti di " +
		"sistem lama.",
	"Pada Progress Klaim per PIC, lini bisnis harus dipilih sendiri. Di sistem lama ia " +
		"diambil dari jabatan pada catatan operator Pega, dan nilai itu belum tersedia " +
		"di sistem baru.",
	"Bagian \"Approval Progress Klaim\" dan tombol \"Input Progress Claim\" belum " +
		"dibawa. Keduanya menulis ke tabel yang selama masa berjalan paralel masih " +
		"dimiliki sistem lama.",
	"Tenggat tindak lanjut ditampilkan tanggalnya saja. Jam yang ditampilkan sistem " +
		"lama keliru — bagian menitnya sebenarnya berisi bulan.",
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	views := make([]ViewDTO, 0, len(meta.Views))
	for _, view := range meta.Views {
		views = append(views, toViewDTO(view))
	}

	lines := make([]BusinessLineDTO, 0, len(meta.BusinessLines))
	for _, line := range meta.BusinessLines {
		lines = append(lines, BusinessLineDTO{Code: string(line), Label: line.Label()})
	}

	return MetadataResponse{
		Views:         views,
		DefaultView:   meta.DefaultView,
		BusinessLines: lines,
		PageSize:      meta.PageSize,
		Portal:        portalAlias,
		Limitations:   limitations,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxprogressclaim.ClaimPage) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// toListResponse merakit jawaban isi satu region.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	response := ListResponse{
		View: toViewDTO(listed.Query.View),
		Filter: AppliedFilterDTO{
			Keyword:  listed.Query.Keyword,
			Business: string(listed.Query.Business),
			From:     toDateString(listed.Query.From),
			To:       toDateString(listed.Query.To),
		},
		Portal: portalAlias,
	}

	switch listed.Query.View.Kind {
	case inboxprogressclaim.KindClaim:
		response.Rows = toClaimRowListDTO(listed.Claims.Items)
		response.Pagination = toPaginationDTO(listed.Claims)

	case inboxprogressclaim.KindPIC:
		rows := toPICRowListDTO(listed.PICRows)
		response.Rows = rows
		// Rekap ini tidak dipaginasi. Keterangan halamannya tetap diisi supaya layar tidak
		// perlu bercabang saat menampilkan jumlah baris — satu halaman berisi semuanya.
		response.Pagination = PaginationDTO{
			Page: 1, Size: len(rows), Total: len(rows), TotalPages: 1,
		}

	default:
		// Region Evaluasi: kosong di sistem lama, dan kosong di sini. Senarai kosong,
		// bukan nil.
		response.Rows = []ClaimRowDTO{}
		response.Pagination = PaginationDTO{Page: 1, Size: 0, Total: 0, TotalPages: 1}
	}

	return response
}

// toDateString memformat tanggal, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}
