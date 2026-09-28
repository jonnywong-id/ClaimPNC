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

	// Aging adalah kolom "Aging" — DIBACA dari kolom `AGING`, bukan dihitung.
	//
	// Penunjuk, bukan angka: `null` berarti belum dihitung, dan itu berbeda dari nol hari.
	// Menyamakannya akan menampilkan "0" pada baris yang sebenarnya tidak punya angka.
	Aging *int `json:"aging"`

	StatusASM string `json:"status_asm"`

	// JenisSurveyor adalah `SURVEYORTYPE_1` — `"1"` internal, `"2"` loss adjuster.
	//
	// Tidak digambar sebagai kolom. Dikirim supaya antrean yang tampak salah isi dapat
	// ditelusuri tanpa membuka basis data.
	JenisSurveyor string `json:"jenis_surveyor"`

	// StatusSurvei adalah `STS_SURVEY`, dibawa apa adanya dan TIDAK dipakai menyaring.
	//
	// Nilainya tidak terbaca dari export. Ia dikirim supaya domainnya terlihat dari data
	// nyata, dan tab Close dapat dikoreksi bila ternyata ia acuan yang benar.
	StatusSurvei string `json:"status_survei"`
}

// ColumnDTO adalah satu judul kolom.
type ColumnDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
}

// TabDTO adalah satu tab beserta judulnya.
type TabDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
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

	Kolom      []ColumnDTO `json:"kolom"`
	Tab        []TabDTO    `json:"tab"`
	KolomKPI   []ColumnDTO `json:"kolom_kpi"`
	TabBawaan  string      `json:"tab_bawaan"`
	JenisKPI   []string    `json:"jenis_kpi"`
	UkuranHala int         `json:"ukuran_halaman"`

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
	// Kelompok berisi nama adjuster, atau TAHUN pada ringkasan kuartal.
	Kelompok string `json:"kelompok"`

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

// KPIResponse adalah ringkasan KPI adjuster.
type KPIResponse struct {
	Portal   string      `json:"portal"`
	Identity IdentityDTO `json:"identitas"`

	Jenis    string `json:"jenis"`
	Kategori string `json:"kategori"`
	Tahun    string `json:"tahun"`

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
		columns = append(columns, ColumnDTO{Kunci: c.Key, Judul: c.Title, Keterangan: c.Note})
	}

	tabList := make([]TabDTO, 0, len(meta.Tabs))
	for _, t := range meta.Tabs {
		tabList = append(tabList, TabDTO{
			Kunci:      string(t.Key),
			Judul:      t.Title,
			Keterangan: t.Note,
		})
	}

	kpi := make([]ColumnDTO, 0, len(meta.KPIColumns))
	for _, c := range meta.KPIColumns {
		kpi = append(kpi, ColumnDTO{Kunci: c.Key, Judul: c.Title})
	}

	return MetadataResponse{
		Portal:     portalAlias,
		Kolom:      columns,
		Tab:        tabList,
		KolomKPI:   kpi,
		TabBawaan:  string(meta.DefaultTab),
		UkuranHala: meta.PageSize,
		JenisKPI: []string{
			string(inboxsurvey.KPIOutstanding),
			string(inboxsurvey.KPIFinal),
			string(inboxsurvey.KPIQuarterly),
		},
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
func toListResponse(listed usecase.Listed, portalAlias string, loc *time.Location) ListResponse {
	data := make([]TaskDTO, 0, len(listed.Page.Tasks))
	for _, task := range listed.Page.Tasks {
		data = append(data, toTaskDTO(task, loc))
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
func toTaskDTO(task inboxsurvey.SurveyTask, loc *time.Location) TaskDTO {
	return TaskDTO{
		SurveiID:        task.SurveyID,
		KlaimID:         task.ClaimID,
		IndexSurvei:     task.SurveyIndex,
		AppointmentNo:   task.AppointmentNumber,
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
		Aging:           task.AgingDays,
		StatusASM:       task.ASMStatus,
		JenisSurveyor:   task.SurveyorType,
		StatusSurvei:    task.SurveyStatus,
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

	return KPIResponse{
		Portal:   portalAlias,
		Identity: toIdentityDTO(scored.Identity),
		Jenis:    string(scored.Filter.Kind),
		Kategori: scored.Filter.Category,
		Tahun:    scored.Filter.Year,
		Data:     data,
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
