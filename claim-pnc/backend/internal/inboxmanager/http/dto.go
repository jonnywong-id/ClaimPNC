package inboxmanagerhttp

import (
	"time"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/usecase"
)

// Nama field JSON di berkas ini berbahasa Indonesia, dan itu bukan ketidakkonsistenan: nama
// field JSON adalah KONTRAK yang dibaca layar, bukan nama internal (`D-80`).

// ErrorResponse adalah bentuk badan galat.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"rincian,omitempty"`
}

// ViolationDTO menunjuk satu isian yang bermasalah.
type ViolationDTO struct {
	Field   string `json:"isian"`
	Message string `json:"pesan"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`

	// Money menandai kolom nilai uang, supaya layar merata-kanankan dan memformatnya
	// sebagai rupiah alih-alih sebagai pencacah.
	Money bool `json:"uang,omitempty"`
}

// DecisionRuleDTO menyatakan apa yang boleh dilakukan pada sebuah antrean.
//
// Ia dikirim ke layar supaya tombol yang digambar selalu sesuai dengan yang akan diterima
// server. Layar TIDAK menyimpulkannya sendiri dari kode tab: aturan yang hidup di dua tempat
// akan menyimpang, dan yang menyimpang di sini adalah tombol yang tampak dapat ditekan tetapi
// selalu ditolak.
type DecisionRuleDTO struct {
	Decidable bool `json:"dapat_diputuskan"`

	// ApproveBlockedReason kosong berarti tombol Setujui boleh digambar aktif.
	ApproveBlockedReason string `json:"alasan_setuju_ditahan,omitempty"`

	ReasonRequiredOnReject bool   `json:"alasan_wajib_saat_menolak"`
	ReasonLabel            string `json:"label_alasan,omitempty"`
}

// TabDTO adalah satu tab beserta bentuk isinya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	// Kind bernilai "dashboard", "ringkasan", atau "antrean".
	Kind string `json:"jenis"`

	// Panels hanya terisi pada tab dashboard — bentuk gridnya, tanpa barisnya.
	Panels []PanelShapeDTO `json:"panel,omitempty"`

	// Columns hanya terisi pada tab antrean.
	Columns []ColumnDTO `json:"kolom,omitempty"`

	Decision DecisionRuleDTO `json:"keputusan"`

	// LineBusiness kosong berarti tab tidak dibatasi lini bisnis.
	LineBusiness string `json:"lini_bisnis,omitempty"`

	HasPeriodFilter bool `json:"punya_penyaring_periode"`
}

// PanelShapeDTO adalah bentuk satu grid dashboard, tanpa barisnya.
type PanelShapeDTO struct {
	Key     string      `json:"kunci"`
	Title   string      `json:"judul"`
	Columns []ColumnDTO `json:"kolom"`
}

// MetadataResponse adalah jawaban GET /api/inbox-manager/tab.
type MetadataResponse struct {
	Tabs         []TabDTO `json:"tab"`
	DefaultTab   string   `json:"tab_bawaan"`
	LineBusiness string   `json:"lini_bisnis_anda"`
}

// CounterDTO adalah satu pencacah di kepala layar.
type CounterDTO struct {
	TabCode string `json:"tab"`
	Label   string `json:"label"`
	Count   int    `json:"jumlah"`

	// Parent kosong pada pencacah tingkat atas.
	Parent string `json:"induk,omitempty"`

	// Unavailable terisi bila sumbernya sedang tidak dapat dibaca. Saat terisi, Count
	// TIDAK bermakna dan layar tidak boleh menggambarnya sebagai angka.
	Unavailable string `json:"tidak_tersedia,omitempty"`
}

// CountersResponse adalah jawaban GET /api/inbox-manager/ringkasan.
type CountersResponse struct {
	Counters []CounterDTO `json:"pencacah"`
}

// CellDTO adalah satu sel grid dashboard.
//
// Ketiga isiannya saling meniadakan: yang terisi hanya satu, sesuai jenis kolomnya. Nilai
// uang dikirim sebagai TEKS presisi penuh — tidak pernah sebagai angka JSON, yang akan
// melewatkannya lewat bilangan pecahan biner (`I-12`).
type CellDTO struct {
	Count  int    `json:"jumlah,omitempty"`
	Amount string `json:"nilai,omitempty"`
	Text   string `json:"teks,omitempty"`
}

// PanelDTO adalah satu grid dashboard beserta isinya.
type PanelDTO struct {
	Key     string               `json:"kunci"`
	Title   string               `json:"judul"`
	Columns []ColumnDTO          `json:"kolom"`
	Rows    []map[string]CellDTO `json:"baris"`
}

// QueueRowDTO adalah satu baris antrean.
type QueueRowDTO struct {
	// Key dikirim dan diterima kembali APA ADANYA. Layar tidak pernah menyusunnya sendiri:
	// bentuknya berbeda tiap antrean, dan satu di antaranya gabungan empat kolom.
	Key   string            `json:"kunci"`
	Cells map[string]string `json:"sel"`
}

// PaginationDTO menyatakan halaman keberapa yang dijawab.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// PeriodDTO menyatakan periode yang BENAR-BENAR dipakai.
//
// Ia dikirim meski pemanggil tidak memilih apa pun, karena sebagian tab memakai nilai bawaan
// — dan periode bawaan yang tidak ditampilkan membuat angka di layar tidak dapat
// dipertanggungkan.
//
// Batas atasnya EKSKLUSIF di dalam kueri; yang dikirim ke layar adalah batas yang terbaca
// manusia, yaitu satu hari sebelumnya.
type PeriodDTO struct {
	From  string `json:"dari"`
	Until string `json:"sampai"`
}

// ListResponse adalah jawaban GET /api/inbox-manager.
type ListResponse struct {
	Tab TabDTO `json:"tab"`

	// Panels terisi pada tab dashboard.
	Panels []PanelDTO `json:"panel,omitempty"`

	// RefreshedAt terisi pada dashboard yang sumbernya tabel cuplikan. Ia BUKAN jam
	// aplikasi — lihat inboxmanager.DashboardView.
	RefreshedAt string `json:"disegarkan_pada,omitempty"`

	// Rows terisi pada tab antrean.
	Rows []QueueRowDTO `json:"baris,omitempty"`

	// Pagination terisi pada tab antrean.
	Pagination *PaginationDTO `json:"paginasi,omitempty"`

	// Period terisi pada tab yang punya penyaring periode.
	Period *PeriodDTO `json:"periode,omitempty"`
}

// DecisionRequest adalah badan POST /api/inbox-manager/keputusan.
type DecisionRequest struct {
	Tab     string   `json:"tab"`
	Verdict string   `json:"keputusan"`
	Keys    []string `json:"kunci"`
	Reason  string   `json:"alasan"`
}

// DecisionResponse adalah jawaban POST /api/inbox-manager/keputusan.
type DecisionResponse struct {
	Requested int `json:"diminta"`
	Changed   int `json:"berubah"`

	// Stale adalah jumlah baris yang TIDAK berubah karena sudah diputuskan lebih dulu.
	//
	// Ia dikirim meski nol. Mengirimkannya hanya saat lebih dari nol akan membuat layar
	// harus menebak selisihnya sendiri — dan menebak angka yang menyangkut keputusan orang
	// lain adalah hal terakhir yang boleh ditebak layar.
	Stale int `json:"tidak_berubah"`

	// Message adalah kalimat siap tampil yang menyatakan hasilnya apa adanya.
	Message string `json:"pesan"`
}

// toTabDTO menyusun satu tab untuk dikirim ke layar.
func toTabDTO(tab inboxmanager.Tab) TabDTO {
	dto := TabDTO{
		Code:            tab.Code,
		Name:            tab.Name,
		Description:     tab.Description,
		Kind:            string(tab.Kind),
		LineBusiness:    tab.LineBusiness,
		HasPeriodFilter: tab.HasPeriodFilter,
		Decision: DecisionRuleDTO{
			Decidable:              tab.Decision.Decidable,
			ApproveBlockedReason:   tab.Decision.ApproveBlockedReason,
			ReasonRequiredOnReject: tab.Decision.ReasonRequiredOnReject,
			ReasonLabel:            tab.Decision.ReasonLabel,
		},
	}

	for _, column := range tab.Columns {
		dto.Columns = append(dto.Columns, toColumnDTO(column))
	}
	for _, panel := range tab.Panels {
		shape := PanelShapeDTO{Key: panel.Key, Title: panel.Title}
		for _, column := range panel.Columns {
			shape.Columns = append(shape.Columns, toColumnDTO(column))
		}
		dto.Panels = append(dto.Panels, shape)
	}

	return dto
}

func toColumnDTO(column inboxmanager.Column) ColumnDTO {
	return ColumnDTO{Key: column.Key, Title: column.Title, Money: column.Money}
}

// toMetadataResponse menyusun jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata) MetadataResponse {
	response := MetadataResponse{
		DefaultTab:         meta.DefaultTab,
		LineBusiness:       meta.LineBusiness,
		Tabs:               []TabDTO{},
	}
	for _, tab := range meta.Tabs {
		response.Tabs = append(response.Tabs, toTabDTO(tab))
	}
	return response
}

// toCountersResponse menyusun jawaban pencacah.
func toCountersResponse(counters []inboxmanager.Counter) CountersResponse {
	response := CountersResponse{Counters: []CounterDTO{}}
	for _, counter := range counters {
		response.Counters = append(response.Counters, CounterDTO{
			TabCode:     counter.TabCode,
			Label:       counter.Label,
			Count:       counter.Count,
			Parent:      counter.Parent,
			Unavailable: counter.Unavailable,
		})
	}
	return response
}

// toListResponse menyusun jawaban isi satu tab.
func toListResponse(view usecase.View) ListResponse {
	response := ListResponse{Tab: toTabDTO(view.Tab)}

	switch view.Tab.Kind {
	case inboxmanager.KindDashboard:
		response.Panels = []PanelDTO{}
		for _, panel := range view.Dashboard.Panels {
			response.Panels = append(response.Panels, toPanelDTO(panel))
		}
		if view.Dashboard.RefreshedAt != nil {
			response.RefreshedAt = view.Dashboard.RefreshedAt.Format(time.RFC3339)
		}

	case inboxmanager.KindQueue:
		response.Rows = []QueueRowDTO{}
		for _, row := range view.Queue.Rows {
			response.Rows = append(response.Rows, QueueRowDTO{Key: row.Key, Cells: row.Cells})
		}
		response.Pagination = &PaginationDTO{
			Page:       view.Queue.Pagination.Page,
			Size:       view.Queue.Pagination.Size,
			Total:      view.Queue.Total,
			TotalPages: view.Queue.TotalPages(),
		}
	}

	if view.Tab.HasPeriodFilter && !view.Period.Empty() {
		response.Period = &PeriodDTO{
			From: view.Period.From.Format("2006-01-02"),

			// Batas atas kueri eksklusif; yang dibaca manusia adalah hari terakhir yang
			// ikut terhitung.
			Until: view.Period.Until.AddDate(0, 0, -1).Format("2006-01-02"),
		}
	}

	return response
}

func toPanelDTO(panel inboxmanager.Panel) PanelDTO {
	dto := PanelDTO{Key: panel.Key, Title: panel.Title, Rows: []map[string]CellDTO{}}
	for _, column := range panel.Columns {
		dto.Columns = append(dto.Columns, toColumnDTO(column))
	}
	for _, row := range panel.Rows {
		cells := map[string]CellDTO{}
		for key, cell := range row.Cells {
			cells[key] = CellDTO{Count: cell.Count, Amount: cell.Amount, Text: cell.Text}
		}
		dto.Rows = append(dto.Rows, cells)
	}
	return dto
}
