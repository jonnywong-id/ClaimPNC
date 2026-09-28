// Package usecase mengorkestrasi modul My Work (MENU_ID 50).
//
// Empat operasi, dan keempatnya hanya MEMBACA:
//
//	Metadata  judul kolom, judul tab, selisih terencana, dan keterbatasan
//	List      satu halaman satu tab
//	Counts    jumlah baris ketujuh tab, untuk bilah tab
//	KPI       ringkasan KPI adjuster
//
// Tidak ada operasi yang menulis. Menerima penugasan, menjadwal ulang survei, dan mengunggah
// laporan seluruhnya menempuh `Surveyor_Flow` — sebuah flow yang TIDAK ADA di export — dan
// penugasan masih dimiliki Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxsurvey"
)

// Service melayani modul My Work.
type Service struct {
	repoSelector      inboxsurvey.RepoSelector
	directorySelector inboxsurvey.DirectorySelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector      inboxsurvey.RepoSelector
	DirectorySelector inboxsurvey.DirectorySelector
}

// NewService membentuk layanan modul My Work.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxsurvey/usecase: RepoSelector wajib diisi")
	}
	if o.DirectorySelector == nil {
		return nil, errors.New("inboxsurvey/usecase: DirectorySelector wajib diisi")
	}
	return &Service{
		repoSelector:      o.RepoSelector,
		directorySelector: o.DirectorySelector,
	}, nil
}

// Column adalah satu judul kolom pada layar.
//
// # Kenapa judul kolom datang dari server
//
// Karena ketiga belasnya adalah HASIL PEMBACAAN
// `Section/InboxSurvey_section-Section.xml`, dan tempat pembacaan itu tercatat adalah
// backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu
// akan tertinggal saat yang lain diperbaiki.
type Column struct {
	// Key adalah nama field pada baris JSON yang diisi kolom ini.
	Key string

	// Title adalah judul yang DILIHAT pengguna, apa adanya dari section (`D-13`).
	//
	// Ketiga belasnya berbahasa Inggris di layar lama, dan dibiarkan begitu. `D-80`
	// menetapkan teks yang dilihat pengguna mengikuti layar Pega apa adanya; menerjemahkan
	// "Appointment No" menjadi "Nomor Janji" akan membuat pengguna yang hafal layarnya harus
	// belajar ulang — hal yang justru dihindari `D-13`.
	Title string

	// Note adalah keterangan yang ditempelkan pada judul, kosong bila tidak ada.
	Note string
}

// columns adalah ketiga belas kolom layar, dalam urutan tampilnya.
//
// Urutannya mengikuti pembacaan grid pada `Section/InboxSurvey_section-Section.xml`, yang
// memuat ketiga belas judul itu berturut-turut pada ketiga varian gridnya.
var columns = []Column{
	{
		Key:   "appointment_no",
		Title: "Appointment No",
		Note: "Diambil dari kolom `ADJUSTERPIC_1`. Pemetaannya belum dikonfirmasi DBA — " +
			"nama kolomnya berbunyi \"PIC\", bukan nomor janji.",
	},
	{Key: "reference_no", Title: "Reference No"},
	{Key: "claim_no", Title: "Claim No"},
	{Key: "policy_no", Title: "Policy No"},
	{Key: "insured_name", Title: "Insured Name"},
	{Key: "cob", Title: "COB"},
	{
		Key:   "cause_of_loss",
		Title: "Cause Of Loss",
		Note: "Diambil dari kolom `LOSSTYPE`. Kueri Pega yang mengisinya hilang dari " +
			"export, sehingga pemetaannya belum dapat dibuktikan.",
	},
	{Key: "location", Title: "Location"},
	{Key: "pic_asm", Title: "PIC ASM"},
	{Key: "pic_loss_adjuster", Title: "PIC Loss Adjuster"},
	{Key: "date_of_loss", Title: "Date of Loss"},
	{
		Key:   "aging",
		Title: "Aging",
		Note:  "Dibaca dari kolom `AGING`, bukan dihitung. Kosong berarti belum dihitung.",
	},
	{Key: "status_asm", Title: "Status ASM"},
}

// Columns menyerahkan salinan daftar kolom.
func Columns() []Column {
	result := make([]Column, len(columns))
	copy(result, columns)
	return result
}

// TabInfo adalah satu tab beserta judul yang dilihat pengguna.
type TabInfo struct {
	Key   inboxsurvey.Tab
	Title string
	Note  string
}

// tabs adalah ketujuh tab, dalam urutan tampilnya di section.
//
// Judulnya apa adanya dari `Section/InboxSurvey_section-Section.xml`, yang menuliskannya
// sebagai teks tebal: Outstanding · Invoice · Close · ALL · Not answered communication ·
// Not replied from ASM · Replied from ASM.
var tabs = []TabInfo{
	{
		Key:   inboxsurvey.TabOutstanding,
		Title: "Outstanding",
		Note:  "Penugasan yang belum dikonfirmasi adjuster.",
	},
	{
		Key:   inboxsurvey.TabInvoice,
		Title: "Invoice",
		Note:  "Sudah dikonfirmasi dan berstatus Invoice Fee.",
	},
	{
		Key:   inboxsurvey.TabClose,
		Title: "Close",
		Note: "Berstatus Close Case. Di layar lama tab ini memakai status ALUR KERJA objek " +
			"survei, yang tidak tersedia di tabel penggantinya — lihat selisih terencana.",
	},
	{
		Key:   inboxsurvey.TabAll,
		Title: "ALL",
		Note: "Sudah dikonfirmasi adjuster. Namanya \"ALL\" di layar lama, tetapi ia bukan " +
			"seluruh baris.",
	},
	{
		Key:   inboxsurvey.TabNotAnswered,
		Title: "Not answered communication",
		Note:  "Ada pesan terbuka dari pihak lain yang belum Anda jawab.",
	},
	{
		Key:   inboxsurvey.TabNotReplied,
		Title: "Not replied from ASM",
		Note:  "Anda sudah mengirim pesan, ASM belum membalas.",
	},
	{
		Key:   inboxsurvey.TabReplied,
		Title: "Replied from ASM",
		Note:  "Pesan Anda sudah dibalas ASM.",
	},
}

// TabsInfo menyerahkan salinan daftar tab.
func TabsInfo() []TabInfo {
	result := make([]TabInfo, len(tabs))
	copy(result, tabs)
	return result
}

// KPIColumn adalah satu judul kolom angka pada tab KPI.
type KPIColumn struct {
	Key   string
	Title string
}

// kpiColumns adalah kesembilan judul angka KPI, apa adanya dari section.
//
// Section menuliskannya dalam KAPITAL, dan bentuk itu dibawa: ia teks yang dilihat pengguna
// (`D-13`).
var kpiColumns = []KPIColumn{
	{Key: "penjadwalan_survey", Title: "PENJADWALAN SURVEY"},
	{Key: "immediate_advice", Title: "IMMEDIATE ADVICE"},
	{Key: "preliminary_advice", Title: "PRELIMINARY ADVICE"},
	{Key: "interim_report", Title: "INTERIM REPORT"},
	{Key: "update_progress", Title: "UPDATE PROGRESS"},
	{Key: "tanggapan_komunikasi", Title: "TANGGAPAN KOMUNIKASI"},
	{Key: "propose_adjustment", Title: "PROPOSE ADJUSTMENT"},
	{Key: "final_report", Title: "FINAL REPORT"},
	{Key: "nilai", Title: "NILAI"},
}

// KPIColumns menyerahkan salinan judul angka KPI.
func KPIColumns() []KPIColumn {
	result := make([]KPIColumn, len(kpiColumns))
	copy(result, kpiColumns)
	return result
}

// PlannedDifferences adalah perbedaan yang DISENGAJA terhadap layar Pega.
//
// Ia dikirim ke layar dan ditampilkan, bukan disimpan sebagai catatan teknis. `D-54`
// menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan Work Owner tertulis, dan
// menyatakannya di layar itulah yang membuat keputusan itu terlihat oleh orang yang memakai
// layarnya — bukan hanya oleh orang yang membaca kodenya.
func PlannedDifferences() []string {
	return []string{
		"Sumber datanya BERGESER. Layar lama membaca objek kerja `Work-SurveyClaim` di " +
			"tabel Pega; tabel penggantinya tidak memuat satu pun baris jenis itu, sehingga " +
			"baris di sini digerakkan `T_SURVEYORLIST` dan header klaimnya diambil dari " +
			"`T_CLAIMLIST_ADMIN`. Keputusan Work Owner 2026-09-28.",

		"Tab Close memakai status adjuster `Close Case`, bukan status alur kerja " +
			"`Resolved-Completed` milik objek survei — kolom itu tidak ada di tabel " +
			"penggantinya. Keduanya berkorelasi tetapi tidak sama: survei berstatus Close " +
			"Case yang objek kerjanya belum ditutup akan muncul di sini, sementara di layar " +
			"lama tidak.",

		"Status adjuster dan Reference No berlaku PER KLAIM, bukan per janji survei. Di " +
			"layar lama keduanya melekat pada tiap objek survei. Klaim dengan dua janji " +
			"survei berstatus berbeda karena itu menampilkan status yang sama pada kedua " +
			"barisnya.",

		"Halaman dipotong basis data, bukan setelah seluruh baris ditarik. Layar lama " +
			"menarik semuanya lalu menomori halamannya di memori dengan ukuran 15 baris; " +
			"di sini ukurannya 25 dan pemotongannya terjadi sebelum baris meninggalkan " +
			"basis data.",

		"Kotak cari Claim No dan Reference No dikerjakan SERVER. Layar lama menyaringnya " +
			"di klipboard, sehingga hasilnya hanya menyentuh halaman yang sedang terbuka.",
	}
}

// Limitations adalah keterbatasan yang berlaku hari ini dan akan hilang dengan sendirinya.
//
// Bedanya dengan PlannedDifferences: yang di atas adalah pilihan yang sudah diputuskan, yang
// di bawah adalah penghalang yang masih menunggu pihak lain. Keduanya dipisah supaya
// keterbatasan yang selesai dapat dihapus tanpa menyentuh keputusan yang masih berlaku.
func Limitations() []string {
	return []string{
		"Empat kueri tab layar lama HILANG dari export — `BrowseOSLostAdjuster`, " +
			"`BrowseConfirmLostAdjuster`, `BrowseCommunicationLostAdjuster`, dan " +
			"`BrowseCloseLostAdjuster`. Penyaring ketujuh tab dipulihkan dari " +
			"`CountOSLostAdjuster` yang menghitung keranjang yang sama; daftar kolom dan " +
			"urutannya mengikuti `BrowseLossAdjuster`.",

		"Dua pemetaan kolom menunggu konfirmasi DBA: \"Appointment No\" ke `ADJUSTERPIC_1` " +
			"dan \"Cause Of Loss\" ke `LOSSTYPE`.",

		"Membuka baris untuk mengerjakan surveinya belum tersedia. Di layar lama tautannya " +
			"membuka penugasan `Surveyor_Flow`, dan flow itu tidak ada di export — " +
			"`Flow/` hanya memuat empat, dan itu bukan salah satunya. Selama masa paralel " +
			"penugasan tetap dikerjakan di Pega.",

		"Pemeriksaan kewenangan menu belum ada (`TKT-F3-005`). Yang menjaga layar ini " +
			"sekarang adalah sesi, portal aktif, dan penyaring identitas surveyor.",
	}
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	Columns            []Column
	Tabs               []TabInfo
	KPIColumns         []KPIColumn
	PlannedDifferences []string
	Limitations        []string

	// DefaultTab adalah tab yang terbuka saat layar dibuka pertama kali.
	DefaultTab inboxsurvey.Tab

	// PageSize adalah ukuran halaman bawaan.
	PageSize int
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: judul kolom dan
// judul tab sama di seluruh entitas, karena keduanya bentuk layar — bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Columns:            Columns(),
		Tabs:               TabsInfo(),
		KPIColumns:         KPIColumns(),
		PlannedDifferences: PlannedDifferences(),
		Limitations:        Limitations(),
		DefaultTab:         inboxsurvey.DefaultTab,
		PageSize:           inboxsurvey.DefaultLimit,
	}
}

// Listed adalah satu halaman antrean beserta penyaring yang benar-benar dipakai.
type Listed struct {
	// Identity adalah identitas surveyor pemanggil.
	//
	// Dikirim balik ke layar, dan itu bukan kelebihan: seorang leader melihat pekerjaan
	// anggotanya, sehingga tanpa menyebut cakupannya pengguna tidak dapat menjelaskan kenapa
	// ia melihat baris atas nama orang lain.
	Identity inboxsurvey.SurveyorIdentity

	// Filter adalah penyaring setelah dinormalkan.
	//
	// Layar menggambar keadaan kotak cari, tab aktif, dan bilah halaman dari sini — bukan
	// dari isian yang ia kirim. Permintaan `batas=5000` dipangkas menjadi 100, dan tab yang
	// tidak dikenal dijatuhkan ke Outstanding.
	Filter inboxsurvey.Filter

	Page inboxsurvey.Page
}

// List mengambil satu halaman satu tab.
//
// # Urutannya: identitas, portal, jembatan, baru antrean
//
// Identitas diperiksa lebih dulu karena tanpanya tidak ada antrean yang dapat dibentuk sama
// sekali, dan memilih repo untuk permintaan yang pasti ditolak hanyalah satu perjalanan yang
// terbuang. Yang lebih penting: urutan ini membuat kegagalan sesi terbaca sebagai kegagalan
// sesi, bukan sebagai antrean kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
	filter inboxsurvey.Filter,
) (Listed, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Listed{}, err
	}

	clean := filter.Normalize()

	page, err := repo.List(ctx, identity, clean)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil antrean survei: %w", err)
	}

	return Listed{Identity: identity, Filter: clean, Page: page}, nil
}

// Counted adalah jumlah baris ketujuh tab beserta identitas yang dipakai menghitungnya.
type Counted struct {
	Identity inboxsurvey.SurveyorIdentity
	Counts   []inboxsurvey.TabCount
}

// Counts menghitung isi ketujuh tab.
//
// Ia rute TERSENDIRI, bukan bagian dari jawaban List. Alasannya: bilah tab tidak berubah saat
// pengguna berpindah halaman, sehingga menempelkannya pada setiap permintaan daftar akan
// menjalankan tujuh penjumlahan setiap kali tombol halaman ditekan.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
) (Counted, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Counted{}, err
	}

	counts, err := repo.Counts(ctx, identity)
	if err != nil {
		return Counted{}, fmt.Errorf("menghitung isi tab antrean survei: %w", err)
	}

	return Counted{Identity: identity, Counts: counts}, nil
}

// Scored adalah ringkasan KPI beserta penyaring yang benar-benar dipakai.
type Scored struct {
	Identity inboxsurvey.SurveyorIdentity
	Filter   inboxsurvey.KPIFilter
	Rows     []inboxsurvey.KPIRow
}

// KPI mengambil ringkasan KPI adjuster.
//
// # Ia disaring cakupan yang SAMA dengan antrean
//
// Itu keputusan, bukan kebetulan. `DETAIL_KPI_ADJUSTER` memuat penilaian SELURUH adjuster,
// dan di Pega layar KPI ini berada di dalam harness yang sama dengan antrean — yang berarti
// pemakainya sama. Menampilkan penilaian adjuster lain akan mengubah layar kerja menjadi
// papan peringkat yang tidak pernah diminta siapa pun.
func (s *Service) KPI(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
	filter inboxsurvey.KPIFilter,
) (Scored, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Scored{}, err
	}

	clean := filter.Normalize()

	rows, err := repo.KPI(ctx, identity, clean)
	if err != nil {
		return Scored{}, fmt.Errorf("mengambil ringkasan KPI adjuster: %w", err)
	}

	return Scored{Identity: identity, Filter: clean, Rows: rows}, nil
}

// prepare menjalankan tiga langkah yang sama bagi ketiga operasi pembaca.
//
// Ia ada supaya urutan pemeriksaannya TIDAK dapat berbeda antar operasi. Urutan yang berbeda
// menghasilkan galat yang berbeda untuk keadaan yang sama — dan layar yang menerima "sesi
// tidak terbaca" pada satu rute lalu "portal tidak dinyatakan" pada rute lain, untuk satu
// permintaan yang sama, tidak dapat menjelaskan apa pun kepada penggunanya.
func (s *Service) prepare(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
) (inboxsurvey.SurveyorIdentity, inboxsurvey.Repo, error) {
	if caller.Login == "" {
		return inboxsurvey.SurveyorIdentity{}, nil, inboxsurvey.ErrCallerUnknown
	}

	directory, err := s.directorySelector(portalAlias)
	if err != nil {
		return inboxsurvey.SurveyorIdentity{}, nil, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxsurvey.SurveyorIdentity{}, nil, err
	}

	identity, err := directory.ResolveSurveyor(ctx, caller.Login)
	if err != nil {
		// ErrNotSurveyor diteruskan APA ADANYA, tidak dibungkus menjadi galat lain dan tidak
		// diubah menjadi antrean kosong. Lapisan transport yang menerjemahkannya menjadi
		// jawaban yang dapat dibaca pengguna.
		if errors.Is(err, inboxsurvey.ErrNotSurveyor) {
			return inboxsurvey.SurveyorIdentity{}, nil, err
		}
		return inboxsurvey.SurveyorIdentity{}, nil,
			fmt.Errorf("menerjemahkan identitas surveyor: %w", err)
	}

	if len(identity.Scope) == 0 {
		// Identitas terbaca tetapi cakupannya kosong. Kueri antrean akan mengembalikan nol
		// baris, dan nol baris terbaca sebagai "tidak ada pekerjaan" — bukan sebagai data
		// master yang belum lengkap.
		return inboxsurvey.SurveyorIdentity{}, nil, inboxsurvey.ErrNotSurveyor
	}

	return identity, repo, nil
}
