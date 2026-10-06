package inboxrclhttp

import (
	"time"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/usecase"
)

// Bentuk jawaban modul Inbox RCL. Nama field JSON berbahasa Indonesia karena ia KONTRAK
// (`D-80`).

// TaskDTO adalah satu baris antrean.
type TaskDTO struct {
	// KlaimID sama dengan NomorCase (`TC_PNC_PUCL.CLAIMID`). Dipakai sebagai kunci baris dan
	// kunci layar kerja.
	KlaimID string `json:"klaim_id"`

	NomorCase       string `json:"nomor_case"`
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`

	// TanggalMasukInbox berbentuk "YYYY-MM-DD HH:mm", sudah dalam WIB (`R-12`).
	TanggalMasukInbox string `json:"tanggal_masuk_inbox"`

	// DeskripsiAnalyst adalah `KOMENTAR_ANALISATOR`.
	DeskripsiAnalyst string `json:"deskripsi_analyst"`

	// Mode, StatusProses, dan OperatorPenerima tidak digambar; dikirim untuk penelusuran.
	Mode             string `json:"mode"`
	StatusProses     string `json:"status_proses"`
	OperatorPenerima string `json:"operator_penerima"`
}

// ColumnDTO adalah satu judul kolom.
type ColumnDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Portal           string      `json:"portal"`
	Kolom            []ColumnDTO `json:"kolom"`
	SelisihTerencana []string    `json:"selisih_terencana"`
	Keterbatasan     []string    `json:"keterbatasan"`
	UkuranHalaman    int         `json:"ukuran_halaman"`
}

// ListResponse adalah satu halaman antrean.
type ListResponse struct {
	Portal string    `json:"portal"`
	Data   []TaskDTO `json:"data"`
	Total  int       `json:"total"`
	Lewati int       `json:"lewati"`
	Batas  int       `json:"batas"`
	Cari   string    `json:"cari"`

	// PenggunaDitemukan menyatakan login pemanggil ada dan aktif di POOLDATA.M_LOGIN_PNC.
	// Tidak digambar layar; dikirim supaya antrean kosong dapat ditelusuri dari jawaban API.
	PenggunaDitemukan bool `json:"pengguna_ditemukan"`
}

// DetailResponse adalah isi layar kerja `RCLDokter` satu klaim, seluruhnya dari TC_PNC_PUCL.
type DetailResponse struct {
	Portal string `json:"portal"`

	NomorCase       string `json:"nomor_case"`
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`

	// Mode adalah `RCL_PUCL`: "1" RCL (Setuju / Tidak Setuju), "3" MSIG (Back / Submit).
	Mode string `json:"mode"`

	// CatatanAnalyst — "Catatan dari Analyst" <- KOMENTAR_ANALISATOR.
	CatatanAnalyst string `json:"catatan_analyst"`

	// Alasan — "Alasan Klaim Ditolak/RCL" (mode RCL) atau "Alasan Klaim MSIG" (mode MSIG)
	// <- KETERANGAN2.
	Alasan string `json:"alasan"`

	// AlasanDokter — "Alasan Dokter" <- ALASAN_DOKTER_REJECT_RCL.
	AlasanDokter string `json:"alasan_dokter"`

	StatusKlaim       string `json:"status_klaim"`
	StatusProses      string `json:"status_proses"`
	OperatorPenerima  string `json:"operator_penerima"`
	TanggalMasukInbox string `json:"tanggal_masuk_inbox"`
}

// ErrorResponse adalah bentuk galat yang dibaca klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	columns := make([]ColumnDTO, 0, len(meta.Columns))
	for _, c := range meta.Columns {
		columns = append(columns, ColumnDTO{Kunci: c.Key, Judul: c.Title, Keterangan: c.Note})
	}

	return MetadataResponse{
		Portal:           portalAlias,
		Kolom:            columns,
		SelisihTerencana: meta.PlannedDifferences,
		Keterbatasan:     meta.Limitations,
		UkuranHalaman:    meta.PageSize,
	}
}

func toListResponse(listed usecase.Listed, portalAlias string, loc *time.Location) ListResponse {
	data := make([]TaskDTO, 0, len(listed.Page.Tasks))
	for _, task := range listed.Page.Tasks {
		data = append(data, toTaskDTO(task, loc))
	}

	return ListResponse{
		Portal:            portalAlias,
		Data:              data,
		Total:             listed.Page.Total,
		Lewati:            listed.Filter.Offset,
		Batas:             listed.Filter.Limit,
		Cari:              listed.Filter.Search,
		PenggunaDitemukan: listed.IdentityFound,
	}
}

func toTaskDTO(task inboxrcl.RCLTask, loc *time.Location) TaskDTO {
	return TaskDTO{
		KlaimID:           task.ClaimNumber,
		NomorCase:         task.ClaimNumber,
		NomorPolis:        task.PolicyNumber,
		NamaTertanggung:   task.InsuredName,
		TanggalMasukInbox: formatDateTime(task.SentToRCLAt, loc),
		DeskripsiAnalyst:  task.AnalystNote,
		Mode:              string(task.Mode),
		StatusProses:      task.ProcessStatus,
		OperatorPenerima:  task.AssignedOperator,
	}
}

func toDetailResponse(d inboxrcl.RCLDetail, portalAlias string, loc *time.Location) DetailResponse {
	return DetailResponse{
		Portal:            portalAlias,
		NomorCase:         d.ClaimNumber,
		NomorPolis:        d.PolicyNumber,
		NamaTertanggung:   d.InsuredName,
		Mode:              string(d.Mode),
		CatatanAnalyst:    d.AnalystNote,
		Alasan:            d.Reason,
		AlasanDokter:      d.DoctorReason,
		StatusKlaim:       d.StatusClaim,
		StatusProses:      d.ProcessStatus,
		OperatorPenerima:  d.AssignedOperator,
		TanggalMasukInbox: formatDateTime(d.SentToRCLAt, loc),
	}
}

// formatDateTime mengubah waktu UTC menjadi "YYYY-MM-DD HH:mm" WIB — SATU-SATUNYA tempat
// konversi zona waktu pada modul ini.
func formatDateTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}
