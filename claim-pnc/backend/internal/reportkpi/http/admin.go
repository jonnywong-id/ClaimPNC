package reportkpihttp

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini adalah lapisan transport tab **KPI Admin**.

// AdminGroupDTO adalah satu pilihan dropdown "Pilih Data KPI".
type AdminGroupDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`
	Note  string `json:"keterangan,omitempty"`
}

// MetricDTO adalah satu baris terukur pada kartu skor.
type MetricDTO struct {
	Code  string `json:"kode"`
	Label string `json:"judul"`

	// Value `null` berarti metriknya tidak dapat dihitung — misalnya karena tidak ada satu
	// pun klaim pada periode yang dipilih, sehingga pembaginya nol. BUKAN berarti nol.
	Value *float64 `json:"nilai"`

	// Format menyatakan cara angkanya digambar: cacah, persen, nilai, atau desimal.
	//
	// Ia dikirim sebagai DATA, bukan disimpulkan layar dari nama metriknya: satu kartu
	// memuat kelima bentuk sekaligus, dan menebaknya dari nama akan salah pada metrik
	// berikutnya yang ditambahkan.
	Format string `json:"bentuk"`
}

// AdminIdentityDTO adalah kepala kartu skor — siapa yang dinilai.
type AdminIdentityDTO struct {
	Category    string `json:"kategori"`
	Coordinator string `json:"nama_koordinator"`
	NIK         string `json:"nik"`
	WorkUnit    string `json:"unit_kerja"`
}

// AdminFilterDTO adalah penyaring yang BENAR-BENAR dipakai menjawab permintaan.
type AdminFilterDTO struct {
	Group string `json:"kelompok"`
	From  string `json:"dari"`
	To    string `json:"sampai"`
}

// ScorecardResponse adalah jawaban GET /api/report-kpi/admin/kartu-skor.
type ScorecardResponse struct {
	Identity AdminIdentityDTO `json:"identitas"`

	// EffectiveOn adalah TANGGAL EFEKTIF — periode yang dinilai, sebagai teks.
	EffectiveOn string `json:"tanggal_efektif"`

	Metrics []MetricDTO `json:"metrik"`

	// Achievement kosong pada kelompok PA: kartu skornya memang tidak punya baris
	// kesimpulan, dan mengarangnya akan menampilkan penilaian yang tidak pernah dibuat.
	Achievement string `json:"achievement,omitempty"`

	Filter AdminFilterDTO `json:"penyaring"`
	Portal string         `json:"portal"`
}

// AdminDetailRowDTO adalah satu baris grid rincian tab KPI Admin.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang
// digambar dari daftar kolom pada metadata — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang.
type AdminDetailRowDTO struct {
	ClaimNumber  string `json:"no_klaim"`
	PolicyNumber string `json:"no_polis"`
	BusinessName string `json:"business"`
	RegisterDate string `json:"tgl_regist_klaim"`
	TransferDate string `json:"tgl_terima_dokumen"`
	TeamFlag     string `json:"flag"`

	RegisterAging *float64 `json:"aging_regist_klaim"`

	ReceiveDate    string   `json:"tgl_terima_dokumen_pa"`
	LODReceiveDate string   `json:"tgl_terima_lod"`
	AcceptanceDate string   `json:"tgl_pembayaran"`
	PaymentAging   *float64 `json:"aging_pembayaran_klaim"`
	RegisterSLA    string   `json:"status_sla_regist"`
	PaymentSLA     string   `json:"status_sla_pembayaran"`
}

// AdminDetailResponse adalah jawaban GET /api/report-kpi/admin.
type AdminDetailResponse struct {
	Rows       []AdminDetailRowDTO `json:"baris"`
	Pagination PaginationDTO       `json:"paginasi"`
	Filter     AdminFilterDTO      `json:"penyaring"`
	Portal     string              `json:"portal"`
}

// Scorecard menangani GET /api/report-kpi/admin/kartu-skor.
func (h *Handler) Scorecard(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	result, err := h.service.AdminScorecard(
		r.Context(), active.Alias, caller, readAdminFilter(r.URL.Query()))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	card := result.Card
	metrics := make([]MetricDTO, 0, len(card.Metrics))
	for _, metric := range card.Metrics {
		metrics = append(metrics, MetricDTO{
			Code:   metric.Code,
			Label:  metric.Label,
			Value:  optionalNumber(metric.Value),
			Format: string(metric.Format),
		})
	}

	h.writeJSON(w, r, http.StatusOK, ScorecardResponse{
		Identity: AdminIdentityDTO{
			Category:    card.Identity.Category,
			Coordinator: card.Identity.Coordinator,
			NIK:         card.Identity.NIK,
			WorkUnit:    card.Identity.WorkUnit,
		},
		EffectiveOn: card.EffectiveOn,
		Metrics:     metrics,
		Achievement: card.Achievement,
		Filter:      toAdminFilterDTO(result.Query),
		Portal:      active.Alias,
	})
}

// AdminDetail menangani GET /api/report-kpi/admin.
func (h *Handler) AdminDetail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	page := reportkpi.Pagination{
		Page: positiveNumber(query.Get("halaman")),
		Size: positiveNumber(query.Get("ukuran")),
	}

	result, err := h.service.AdminDetail(
		r.Context(), active.Alias, caller, readAdminFilter(query), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, AdminDetailResponse{
		Rows:       toAdminDetailRows(result.Page.Rows),
		Pagination: toPaginationDTO(page, result.Page.Total),
		Filter:     toAdminFilterDTO(result.Query),
		Portal:     active.Alias,
	})
}

// AdminExport menangani GET /api/report-kpi/admin/ekspor.
//
// Ia mengunduh GRID RINCIAN, bukan kartu skornya: kartu skor satu "baris" dan lebih
// berguna dibaca di layar. Bila kelak diminta, ia menempuh parameter `grid` seperti tab
// Adjuster.
func (h *Handler) AdminExport(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter := readAdminFilter(r.URL.Query())
	page := reportkpi.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan.
	first, err := h.service.AdminDetail(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	columns := reportkpi.AdminGridFor(reportkpi.GridAdminDetail).
		ColumnsForGroup(first.Query.Group)

	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}

	h.beginDownload(w, adminExportFilename(first.Query))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, row := range current.Page.Rows {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(header), current.Page.Total))
				return
			}

			cells := make([]string, 0, len(columns))
			for _, column := range columns {
				cells = append(cells, adminCell(row, column.Key))
			}
			if err := writer.Write(cells); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		if !h.flush(w, writer, r) {
			return
		}
		if written >= current.Page.Total || len(current.Page.Rows) == 0 {
			return
		}

		page.Page++
		next, err := h.service.AdminDetail(r.Context(), active.Alias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
}

// readAdminFilter membaca isian penyaring tab KPI Admin.
//
// Dikumpulkan sebagai satu fungsi supaya kartu skor, rincian, dan ekspor membaca parameter
// yang SAMA PERSIS — berkas ekspor yang penyaringnya berbeda dari kartu skornya tidak dapat
// dicocokkan dengan apa pun di layar.
func readAdminFilter(query url.Values) reportkpi.AdminQueryInput {
	return reportkpi.AdminQueryInput{
		Group: query.Get("kelompok"),
		From:  query.Get("dari"),
		To:    query.Get("sampai"),
	}
}

// toAdminFilterDTO menyusun keterangan penyaring yang dipakai.
func toAdminFilterDTO(query reportkpi.AdminQuery) AdminFilterDTO {
	return AdminFilterDTO{
		Group: string(query.Group),
		From:  query.Range.From,
		To:    query.Range.To,
	}
}

// toAdminDetailRows mengubah baris rincian.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien.
func toAdminDetailRows(rows []reportkpi.AdminDetailRow) []AdminDetailRowDTO {
	result := make([]AdminDetailRowDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, AdminDetailRowDTO{
			ClaimNumber:    row.ClaimNumber,
			PolicyNumber:   row.PolicyNumber,
			BusinessName:   row.BusinessName,
			RegisterDate:   row.RegisterDate,
			TransferDate:   row.TransferDate,
			TeamFlag:       row.TeamFlag,
			RegisterAging:  optionalNumber(row.RegisterAging),
			ReceiveDate:    row.ReceiveDate,
			LODReceiveDate: row.LODReceiveDate,
			AcceptanceDate: row.AcceptanceDate,
			PaymentAging:   optionalNumber(row.PaymentAging),
			RegisterSLA:    row.RegisterSLA,
			PaymentSLA:     row.PaymentSLA,
		})
	}
	return result
}

// adminCell mengambil isi satu sel grid rincian menurut nama kolomnya.
//
// Ia memakai konstanta `Field…` yang sama dengan screen.go dan dengan nama field JSON di
// atas — ketiganya merujuk nama yang sama, sehingga tidak dapat berselisih.
func adminCell(row reportkpi.AdminDetailRow, key string) string {
	switch key {
	case reportkpi.FieldClaimNumber:
		return row.ClaimNumber
	case reportkpi.FieldPolicyNumber:
		return row.PolicyNumber
	case reportkpi.FieldBusinessName:
		return row.BusinessName
	case reportkpi.FieldRegisterDate:
		return row.RegisterDate
	case reportkpi.FieldTransferDate:
		return row.TransferDate
	case reportkpi.FieldTeamFlag:
		return row.TeamFlag
	case reportkpi.FieldRegisterAging:
		return numberText(row.RegisterAging)
	case reportkpi.FieldReceiveDate:
		return row.ReceiveDate
	case reportkpi.FieldLODReceiveDate:
		return row.LODReceiveDate
	case reportkpi.FieldAcceptanceDate:
		return row.AcceptanceDate
	case reportkpi.FieldPaymentAging:
		return numberText(row.PaymentAging)
	case reportkpi.FieldRegisterSLA:
		return row.RegisterSLA
	case reportkpi.FieldPaymentSLA:
		return row.PaymentSLA
	default:
		return ""
	}
}

// adminExportFilename menyusun nama berkas beserta penyaring yang menghasilkannya.
func adminExportFilename(query reportkpi.AdminQuery) string {
	name := "rincian-kpi-admin-" + string(query.Group)
	if query.Range.From != "" && query.Range.To != "" {
		name += "-" + query.Range.From + "-sd-" + query.Range.To
	}
	return name + ".csv"
}

// optionalNumber mengubah nilai domain menjadi angka yang boleh `null`.
//
// Yang tidak ada menjadi `null`, BUKAN 0 — keduanya berbeda artinya pada kartu skor.
func optionalNumber(value reportkpi.Score) *float64 {
	if !value.Present {
		return nil
	}
	number := value.Value
	return &number
}

// numberText menggambar satu angka untuk berkas ekspor.
//
// Yang tidak ada menjadi sel KOSONG, bukan "0". Berkas ekspor dibaca ulang di Excel dan
// sering dijumlahkan di sana; nol yang dikarang akan ikut terhitung.
func numberText(value reportkpi.Score) string {
	if !value.Present {
		return ""
	}
	return strconv.FormatFloat(value.Value, 'f', -1, 64)
}
