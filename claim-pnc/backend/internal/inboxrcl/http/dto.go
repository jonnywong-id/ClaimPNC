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
	// KlaimID adalah `PZINSKEY`. Tidak digambar; dipakai sebagai kunci baris.
	KlaimID string `json:"klaim_id"`

	NomorCase       string `json:"nomor_case"`
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`

	// TanggalMasukInbox berbentuk "YYYY-MM-DD HH:mm", sudah dalam WIB.
	//
	// Propertinya `DateTime` (`pyDataType` pada Report Definition) dan section menggambarnya
	// dengan kontrol `pxDateTime`, sehingga jamnya ikut ditampilkan. Konversi ke WIB
	// dikerjakan SERVER, bukan peramban (`R-12`).
	TanggalMasukInbox string `json:"tanggal_masuk_inbox"`

	// DeskripsiAnalyst adalah `.ClaimData.PUCLStatus.KomentarAnalisator`.
	DeskripsiAnalyst string `json:"deskripsi_analyst"`

	// DokterRCL, StatusProses, dan OperatorPenerima tidak digambar; dikirim supaya jawaban
	// dapat ditelusuri tanpa membuka basis data.
	DokterRCL        string `json:"dokter_rcl"`
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

	// IdentitasLamaDitemukan menyatakan identitas lama pemanggil (`TempOperator.City`)
	// ditemukan. `false` berarti antrean kosong karena BELUM DIKETAHUI pekerjaan siapa —
	// bukan karena tidak ada pekerjaan. Layar menggambar keduanya berbeda.
	IdentitasLamaDitemukan bool `json:"identitas_lama_ditemukan"`
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
		Portal:                 portalAlias,
		Data:                   data,
		Total:                  listed.Page.Total,
		Lewati:                 listed.Filter.Offset,
		Batas:                  listed.Filter.Limit,
		Cari:                   listed.Filter.Search,
		IdentitasLamaDitemukan: listed.LegacyIdentityFound,
	}
}

func toTaskDTO(task inboxrcl.RCLTask, loc *time.Location) TaskDTO {
	return TaskDTO{
		KlaimID:           task.ClaimID,
		NomorCase:         task.ClaimNumber,
		NomorPolis:        task.PolicyNumber,
		NamaTertanggung:   task.InsuredName,
		TanggalMasukInbox: formatDateTime(task.SentToRCLAt, loc),
		DeskripsiAnalyst:  task.AnalystNote,
		DokterRCL:         task.RCLDoctor,
		StatusProses:      task.ProcessStatus,
		OperatorPenerima:  task.AssignedOperator,
	}
}

// formatDateTime mengubah waktu UTC menjadi "YYYY-MM-DD HH:mm" WIB — SATU-SATUNYA tempat
// konversi zona waktu pada modul ini, supaya cacat `Set7Hours` sistem lama tidak terulang.
func formatDateTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}
