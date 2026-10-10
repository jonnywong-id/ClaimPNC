package inboxsurveyhttp

import (
	"time"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/usecase"
)

// Bentuk jawaban modul My Work.
//
// Nama field JSON berbahasa Indonesia karena ia KONTRAK yang dibaca layar, bukan nama
// internal (`D-80`); yang berbahasa Inggris hanyalah nama tipe dan nama field Go-nya.
//
// Judul kolom dan judul tab TIDAK diterjemahkan — keduanya teks yang dilihat pengguna, dan
// `D-13` menetapkan teks layar mengikuti Pega apa adanya. Perbedaan perlakuan itu disengaja:
// nama field adalah kontrak antar program, judul adalah kalimat untuk manusia.

// TaskDTO adalah satu baris antrean.
type TaskDTO struct {
	// SurveiID adalah kunci teknis objek survei (`T_SURVEYORLIST.CASEID`).
	//
	// TIDAK digambar sebagai kolom — isinya memuat nama kelas internal Pega — tetapi dikirim
	// karena ia kunci baris di tabel.
	SurveiID string `json:"survei_id"`

	// KlaimID adalah kunci klaim induknya, dipakai tautan baris membuka klaimnya.
	KlaimID string `json:"klaim_id"`

	// IndexSurvei adalah janji keberapa pada klaim yang sama.
	//
	// Tidak digambar. Tanpanya dua baris milik satu klaim tidak dapat dibedakan saat
	// menelusuri keluhan.
	IndexSurvei string `json:"index_survei"`

	// AppointmentNo dan ReferenceNo SELALU kosong hari ini.
	//
	// Kolom asalnya milik objek kerja `Work-SurveyClaim`, dan diminta ditambahkan ke
	// `POOLDATA.T_SURVEYORLIST`. Keduanya TETAP dikirim supaya kolomnya tetap
	// tergambar; keterangan `tersedia: false` pada metadata kolom yang menyatakan sebabnya,
	// sehingga sel kosong tidak terbaca sebagai "data belum diisi".
	//
	// `StatusASM` TIDAK lagi termasuk: `STS_SURVEY` terbukti membawa domain `ADJUSTERSTATUS_1`,
	// sehingga kolom itu kini terisi.
	AppointmentNo   string `json:"appointment_no"`
	ReferenceNo     string `json:"reference_no"`
	ClaimNo         string `json:"claim_no"`
	PolicyNo        string `json:"policy_no"`
	InsuredName     string `json:"insured_name"`
	COB             string `json:"cob"`
	CauseOfLoss     string `json:"cause_of_loss"`
	Location        string `json:"location"`
	PICASM          string `json:"pic_asm"`
	PICLossAdjuster string `json:"pic_loss_adjuster"`

	// DateOfLoss berbentuk YYYY-MM-DD, sudah dalam WIB.
	//
	// Dikonversi SERVER, bukan di peramban. Membiarkan tiap tempat di frontend memutuskan
	// sendiri cara mengubahnya ke WIB adalah cara paling cepat mengulang cacat `Set7Hours`
	// sistem lama, yang menambah tujuh jam manual di 118 titik pada 36 activity (`R-12`).
	DateOfLoss string `json:"date_of_loss"`

	// Aging adalah kolom "Aging" — DIHITUNG dari tanggal janji survei dicatat, bukan dibaca.
	//
	// Penunjuk, bukan angka: `null` berarti tanggal masuknya tidak ada sehingga umurnya tidak
	// dapat dihitung, dan itu berbeda dari nol hari. Menyamakannya akan menampilkan "0" pada
	// baris yang sebenarnya tidak punya angka.
	Aging *int `json:"aging"`

	StatusASM string `json:"status_asm"`

	// JenisSurveyor adalah `SURVEYORTYPE_1` — `"1"` internal, `"2"` loss adjuster.
	//
	// Tidak digambar sebagai kolom. Dikirim supaya antrean yang tampak salah isi dapat
	// ditelusuri tanpa membuka basis data.
	JenisSurveyor string `json:"jenis_surveyor"`
}

// ColumnDTO adalah satu judul kolom.
type ColumnDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`

	// Tersedia menyatakan kolom ini benar-benar terisi dari data.
	//
	// Ia dikirim untuk SETIAP kolom, termasuk yang bernilai `true` — bukan `omitempty`.
	// Dengan `omitempty`, kolom tersedia dan kolom yang field-nya lupa diisi akan terbaca
	// sama oleh layar, dan layar akan menandai seluruh kolom sebagai belum tersedia.
	Tersedia bool `json:"tersedia"`

	// Pengganti menyatakan kolom ini terisi, tetapi dari kolom yang BERBEDA dari Pega.
	//
	// Berbeda dari Tersedia, ia memakai `omitempty`: kolom yang setara adalah keadaan
	// normal, dan menuliskannya pada sepuluh dari tiga belas kolom hanya menambah derau.
	Pengganti bool `json:"pengganti,omitempty"`
}

// TabDTO adalah satu tab beserta judulnya.
type TabDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`

	// Tersedia menyatakan tab ini dapat dihitung dari data yang ada hari ini.
	//
	// Tab yang TIDAK tersedia tetap digambar — `D-13` menetapkan bentuk layar mengikuti Pega.
	// Yang berubah: layar menandainya dan menyebut sebabnya, alih-alih menampilkan daftar
	// kosong yang terbaca sebagai "tidak ada pekerjaan".
	Tersedia bool `json:"tersedia"`

	// AlasanTakTersedia menyebut sebabnya, kosong bila tabnya tersedia.
	AlasanTakTersedia string `json:"alasan_tak_tersedia,omitempty"`
}

// IdentityDTO adalah identitas surveyor pemanggil.
//
// # Kenapa cakupannya ikut dikirim
//
// Karena seorang leader melihat pekerjaan anggotanya. Tanpa menyebut cakupannya, pengguna
// yang melihat baris atas nama orang lain tidak punya cara menjelaskan kenapa — dan yang
// pertama kali terpikir adalah "layarnya bocor".
type IdentityDTO struct {
	Login     string   `json:"login"`
	Nama      string   `json:"nama"`
	Leader    bool     `json:"leader"`
	Cakupan   []string `json:"cakupan"`
	JumlahTim int      `json:"jumlah_tim"`
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Portal string `json:"portal"`

	Kolom        []ColumnDTO `json:"kolom"`
	Tab          []TabDTO    `json:"tab"`
	KolomKPI     []ColumnDTO `json:"kolom_kpi"`
	TabBawaan    string      `json:"tab_bawaan"`
	StatusSurvei []string    `json:"status_survei"`
	TipeReport   []string    `json:"tipe_report"`
	Kuartal      []string    `json:"kuartal"`
	UkuranHala   int         `json:"ukuran_halaman"`

	// SelisihTerencana adalah perbedaan yang DISENGAJA terhadap layar Pega (`D-54`).
	SelisihTerencana []string `json:"selisih_terencana"`

	// Keterbatasan adalah penghalang yang masih menunggu pihak lain, dan akan hilang dengan
	// sendirinya begitu penghalangnya hilang.
	Keterbatasan []string `json:"keterbatasan"`
}

// ListResponse adalah satu halaman antrean.
type ListResponse struct {
	Portal   string      `json:"portal"`
	Identity IdentityDTO `json:"identitas"`

	Data []TaskDTO `json:"data"`

	// Total adalah jumlah SELURUH baris yang cocok, bukan jumlah baris di halaman ini.
	Total int `json:"total"`

	// Lewati dan Batas adalah paginasi yang BENAR-BENAR dipakai, bukan yang diminta.
	Lewati int `json:"lewati"`
	Batas  int `json:"batas"`

	// Tab adalah tab yang BENAR-BENAR dipakai. Tab tak dikenal dijatuhkan ke Outstanding,
	// dan tanpa mengembalikan yang dipakai, bilah tab di layar akan menyorot tab yang salah.
	Tab  string `json:"tab"`
	Cari string `json:"cari"`
}

// TabCountDTO adalah jumlah baris satu tab.
type TabCountDTO struct {
	Kunci string `json:"kunci"`
	Total int    `json:"total"`
}

// CountResponse adalah jumlah baris ketujuh tab.
type CountResponse struct {
	Portal   string        `json:"portal"`
	Identity IdentityDTO   `json:"identitas"`
	Tab      []TabCountDTO `json:"tab"`
}

// KPIRowDTO adalah satu baris ringkasan KPI.
type KPIRowDTO struct {
	// Kelompok berisi nama adjuster, atau TAHUN pada bentuk berkuartal.
	Kelompok string `json:"kelompok"`

	// Ketiganya terisi HANYA pada bentuk yang memakainya; pada bentuk lain selalu kosong.
	// Mana yang berlaku dinyatakan KolomAwal pada jawaban, bukan ditebak layar dari isinya.
	Status  string `json:"status"`
	Kuartal string `json:"kuartal"`
	Bulan   string `json:"bulan"`
	CaseID  string `json:"case_id"`

	PenjadwalanSurvey   float64 `json:"penjadwalan_survey"`
	ImmediateAdvice     float64 `json:"immediate_advice"`
	PreliminaryAdvice   float64 `json:"preliminary_advice"`
	InterimReport       float64 `json:"interim_report"`
	UpdateProgress      float64 `json:"update_progress"`
	TanggapanKomunikasi float64 `json:"tanggapan_komunikasi"`
	ProposeAdjustment   float64 `json:"propose_adjustment"`
	FinalReport         float64 `json:"final_report"`
	Nilai               float64 `json:"nilai"`
}

// TahunKPIResponse adalah isi dropdown "Tahun Kuartal".
type TahunKPIResponse struct {
	Portal string   `json:"portal"`
	Tahun  []string `json:"tahun"`
}

// KPIResponse adalah ringkasan KPI adjuster.
type KPIResponse struct {
	Portal   string      `json:"portal"`
	Identity IdentityDTO `json:"identitas"`

	StatusSurvei string `json:"status_survei"`
	TipeReport   string `json:"tipe_report"`
	Kuartal      string `json:"kuartal"`
	Tahun        string `json:"tahun"`

	// Bentuk menyatakan APA yang menjadi satu baris — lihat inboxsurvey.KPIShape.
	Bentuk string `json:"bentuk"`

	// KolomAwal adalah kolom kunci di depan kesembilan angka, sesuai Bentuk.
	KolomAwal []ColumnDTO `json:"kolom_awal"`

	Data []KPIRowDTO `json:"data"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Isian string `json:"isian"`
	Pesan string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dibaca klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toMetadataResponse menyusun keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	columns := make([]ColumnDTO, 0, len(meta.Columns))
	for _, c := range meta.Columns {
		columns = append(columns, ColumnDTO{
			Kunci:      c.Key,
			Judul:      c.Title,
			Keterangan: c.Note,
			Tersedia:   c.Available,
			Pengganti:  c.Substitute,
		})
	}

	tabList := make([]TabDTO, 0, len(meta.Tabs))
	for _, t := range meta.Tabs {
		tabList = append(tabList, TabDTO{
			Kunci:             string(t.Key),
			Judul:             t.Title,
			Keterangan:        t.Note,
			Tersedia:          t.Available,
			AlasanTakTersedia: t.UnavailableReason,
		})
	}

	// Kolom KPI seluruhnya tersedia — tabel `DETAIL_KPI_ADJUSTER` memuat kesembilan angkanya.
	// Bila tabelnya sendiri tidak ada, yang kosong adalah datanya, bukan kolomnya.
	kpi := make([]ColumnDTO, 0, len(meta.KPIColumns))
	for _, c := range meta.KPIColumns {
		kpi = append(kpi, ColumnDTO{Kunci: c.Key, Judul: c.Title, Tersedia: true})
	}

	return MetadataResponse{
		Portal:           portalAlias,
		Kolom:            columns,
		Tab:              tabList,
		KolomKPI:         kpi,
		TabBawaan:        string(meta.DefaultTab),
		UkuranHala:       meta.PageSize,
		StatusSurvei:     statusValues(),
		TipeReport:       reportValues(),
		Kuartal:          inboxsurvey.Quarters(),
		SelisihTerencana: meta.PlannedDifferences,
		Keterbatasan:     meta.Limitations,
	}
}

// toIdentityDTO menyusun identitas surveyor pemanggil.
//
// JumlahTim menghitung ANGGOTA, bukan seluruh cakupan: cakupan selalu memuat pemanggil
// sendiri, sehingga seorang surveyor tanpa anggota akan tampak "punya 1 anggota" bila yang
// dikirim adalah panjang cakupannya.
func toIdentityDTO(identity inboxsurvey.SurveyorIdentity) IdentityDTO {
	scope := make([]string, len(identity.Scope))
	copy(scope, identity.Scope)

	members := len(scope) - 1
	if members < 0 {
		members = 0
	}

	return IdentityDTO{
		Login:     identity.Login,
		Nama:      identity.Name,
		Leader:    identity.IsLeader,
		Cakupan:   scope,
		JumlahTim: members,
	}
}

// toListResponse menyusun satu halaman antrean.
//
// `loc` diserahkan pemanggil, bukan dibaca dari jam sistem di sini: ia yang menentukan
// tanggal yang dilihat pengguna, dan memusatkannya adalah cara menghindari cacat `Set7Hours`.
func toListResponse(
	listed usecase.Listed,
	portalAlias string,
	now time.Time,
	loc *time.Location,
) ListResponse {
	data := make([]TaskDTO, 0, len(listed.Page.Tasks))
	for _, task := range listed.Page.Tasks {
		data = append(data, toTaskDTO(task, now, loc))
	}

	return ListResponse{
		Portal:   portalAlias,
		Identity: toIdentityDTO(listed.Identity),
		Data:     data,
		Total:    listed.Page.Total,
		Lewati:   listed.Filter.Offset,
		Batas:    listed.Filter.Limit,
		Tab:      string(listed.Filter.Tab),
		Cari:     listed.Filter.Search,
	}
}

// toTaskDTO mengubah satu tugas menjadi bentuk yang dibaca layar.
//
// `now` dan `loc` keduanya diserahkan pemanggil, bukan dibaca dari jam sistem di sini. Kolom
// Aging dihitung terhadap TANGGAL WIB, sehingga keduanya menentukan angka yang dilihat
// pengguna — dan angka yang bergantung pada jam mesin tidak dapat diuji (`F-5`).
func toTaskDTO(task inboxsurvey.SurveyTask, now time.Time, loc *time.Location) TaskDTO {
	return TaskDTO{
		SurveiID:        task.SurveyID,
		KlaimID:         task.ClaimID,
		IndexSurvei:     task.SurveyIndex,
		AppointmentNo:   task.AppointmentNo(),
		ReferenceNo:     task.ReferenceNumber,
		ClaimNo:         task.ClaimNumber,
		PolicyNo:        task.PolicyNumber,
		InsuredName:     task.InsuredName,
		COB:             task.ClassOfBusiness,
		CauseOfLoss:     task.CauseOfLoss,
		Location:        task.Location,
		PICASM:          task.TechnicalPIC,
		PICLossAdjuster: task.AdjusterPIC,
		DateOfLoss:      formatDate(task.DateOfLoss, loc),
		Aging:           task.AgingDays(now, loc),
		StatusASM:       task.ASMStatus,
		JenisSurveyor:   task.SurveyorType,
	}
}

// toCountResponse menyusun jumlah baris ketujuh tab.
func toCountResponse(counted usecase.Counted, portalAlias string) CountResponse {
	tabList := make([]TabCountDTO, 0, len(counted.Counts))
	for _, c := range counted.Counts {
		tabList = append(tabList, TabCountDTO{Kunci: string(c.Tab), Total: c.Total})
	}

	return CountResponse{
		Portal:   portalAlias,
		Identity: toIdentityDTO(counted.Identity),
		Tab:      tabList,
	}
}

// toKPIResponse menyusun ringkasan KPI.
func toKPIResponse(scored usecase.Scored, portalAlias string) KPIResponse {
	data := make([]KPIRowDTO, 0, len(scored.Rows))
	for _, row := range scored.Rows {
		data = append(data, KPIRowDTO{
			Kelompok:            row.Group,
			Status:              row.Status,
			Kuartal:             row.Quarter,
			Bulan:               row.Month,
			CaseID:              row.CaseID,
			PenjadwalanSurvey:   row.SurveyScheduling,
			ImmediateAdvice:     row.ImmediateAdvice,
			PreliminaryAdvice:   row.PreliminaryAdvice,
			InterimReport:       row.InterimReport,
			UpdateProgress:      row.ProgressUpdate,
			TanggapanKomunikasi: row.CommunicationResponse,
			ProposeAdjustment:   row.ProposeAdjustment,
			FinalReport:         row.FinalReport,
			Nilai:               row.Value,
		})
	}

	shape := scored.Filter.Shape()

	awal := make([]ColumnDTO, 0, 4)
	for _, c := range usecase.KPILeadingColumns(shape) {
		awal = append(awal, ColumnDTO{Kunci: c.Key, Judul: c.Title, Tersedia: true})
	}

	return KPIResponse{
		Portal:       portalAlias,
		Identity:     toIdentityDTO(scored.Identity),
		StatusSurvei: string(scored.Filter.Status),
		TipeReport:   string(scored.Filter.Report),
		Kuartal:      scored.Filter.Quarter,
		Tahun:        scored.Filter.Year,
		Bentuk:       string(shape),
		KolomAwal:    awal,
		Data:         data,
	}
}

// formatDate mengubah waktu UTC menjadi tanggal WIB berbentuk YYYY-MM-DD.
//
// Inilah SATU-SATUNYA tempat konversi zona waktu terjadi pada modul ini. Menyebarkannya akan
// mengulang persis cacat `Set7Hours` sistem lama, yang menambah tujuh jam manual di 118 titik
// pada 36 activity sehingga satu tempat yang lupa menggeser tanggal tanpa terdeteksi.
func formatDate(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}

// statusValues dan reportValues mengubah pilihan domain menjadi teks untuk layar.
//
// Nilainya dikirim APA ADANYA — "ALL", "OUTSTANDING", "FINAL", "DATA SUMMARY",
// "DATA DETAIL" — persis seperti isi `TipeData.pxResults().DESCRIPTION` dan
// `TipeExport.pxResults().DESCRIPTION` di Pega. Menerjemahkannya menjadi kode pendek akan
// membuat nilai yang dikirim layar berbeda dari yang dibandingkan kuerinya.
func statusValues() []string {
	out := make([]string, 0, 3)
	for _, s := range inboxsurvey.SurveyStatuses() {
		out = append(out, string(s))
	}
	return out
}

func reportValues() []string {
	out := make([]string, 0, 2)
	for _, r := range inboxsurvey.ReportTypes() {
		out = append(out, string(r))
	}
	return out
}
