package inboxadminhttp

import (
	"encoding/csv"
	"net/http"
	"time"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Tiga tombol ekspor CSV. Seluruhnya MEMBACA saja.
//
// # Kenapa seluruh baris dibaca lebih dulu, baru berkasnya ditulis
//
// Supaya galat basis data masih dapat dijawab sebagai JSON. Sesudah header CSV terkirim,
// galat tidak dapat lagi dijawab — yang sampai ke pengguna akan berupa berkas separuh jadi
// tanpa satu pun keterangan. Volumenya kecil: Branch Claim sudah ditarik seluruhnya oleh
// layarnya sendiri, dan tabel batch Auto Claim berisi ratusan baris.

var jakarta = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// ExportLOD menangani GET /api/inbox-admin/ekspor/lod — tombol "Export LOD".
//
// Pengganti activity `DownloadAllKlaimCabangLOD`. Judul kolom dan urutannya diambil dari
// `CSVPropHeaders` activity itu; DOL dan Tanggal Report diformat `dd/MM/yyyy` dalam WIB,
// sama seperti `@FormatDateTime(..., "Asia/Jakarta")` di sana.
func (h *Handler) ExportLOD(w http.ResponseWriter, r *http.Request) {
	alias, caller, ok := h.exportContext(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	rows, err := h.service.ExportLOD(r.Context(), alias, caller, inboxadmin.QueryInput{
		Business: query.Get("bisnis"),
		Keyword:  query.Get("cari"),
		Region:   query.Get("kanwil"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writer := startCSV(w, "Export Data LOD Reminder")
	_ = writer.Write([]string{
		"NOKLAIM", "No Polis", "Nama Tertanggung", "Nama Bisnis", "Sumbis",
		"Nama Cabang", "DOL", "Tanggal Report", "User Regist",
	})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.CaseID, row.PolicyNumber, row.InsuredName, row.BusinessName,
			row.BusinessSource, row.BranchName, wibDate(row.LossDate), wibDate(row.ReportDate),
			row.Creator,
		})
	}
	writer.Flush()
}

// ExportAutoClaim menangani GET /api/inbox-admin/ekspor/hasil-auto-claim — tombol
// "Export Hasil Auto Claim". Pengganti activity `PNCExportAutoClaimPNC_Act`.
func (h *Handler) ExportAutoClaim(w http.ResponseWriter, r *http.Request) {
	alias, caller, ok := h.exportContext(w, r)
	if !ok {
		return
	}
	rows, err := h.service.AutoClaimResults(r.Context(), alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writer := startCSV(w, "Laporan Hasil Auto Claim")
	_ = writer.Write([]string{
		"Inisial", "No Polis", "No Klaim", "No Aksep", "Nilai Klaim", "No Objek", "Keterangan",
	})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.Initial, row.PolicyNumber, row.ClaimID, row.AcceptanceNumber,
			row.ClaimValue, row.ObjectNumber, row.Message,
		})
	}
	writer.Flush()
}

// ExportAutoClaimFailures menangani GET /api/inbox-admin/ekspor/klaim-gagal — tombol
// "Export Klaim Gagal". Pengganti activity `PNCExportAutoClaimGagal_Act`.
//
// Judul kolomnya memang nama properti Pega (`NOPOLIS`, `CLIENTID`, ...): begitulah
// `CSVPropHeaders` activity lama, dan berkas itulah yang selama ini diterima pengguna.
func (h *Handler) ExportAutoClaimFailures(w http.ResponseWriter, r *http.Request) {
	alias, _, ok := h.exportContext(w, r)
	if !ok {
		return
	}
	rows, err := h.service.AutoClaimFailures(r.Context(), alias)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writer := startCSV(w, "Laporan Hasil Auto Claim Gagal")
	_ = writer.Write([]string{
		"NOPOLIS", "CLIENTID", "EDMNO", "EDMTYPE", "ACCUMCODE", "REGISTERID", "STATUSBUSINESS",
	})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.PolicyNumber, row.InsuranceNumber, row.ClaimValue, row.ClaimType,
			row.Currency, row.AgentID, row.Message,
		})
	}
	writer.Flush()
}

// exportContext membaca portal aktif dan pemanggil, atau menjawab galatnya.
func (h *Handler) exportContext(w http.ResponseWriter, r *http.Request) (string, inboxadmin.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return "", inboxadmin.Caller{}, false
	}
	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxadmin.ErrCallerUnknown)
		return "", inboxadmin.Caller{}, false
	}
	return active.Alias, caller, true
}

// startCSV menulis header jawaban lalu mengembalikan penulis CSV.
//
// Berkasnya diawali BOM UTF-8 supaya Excel membaca huruf beraksen pada nama tertanggung
// dengan benar.
func startCSV(w http.ResponseWriter, name string) *csv.Writer {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))
	return csv.NewWriter(w)
}

// wibDate menuliskan tanggal sebagai dd/MM/yyyy dalam WIB; kosong bila tidak ada.
func wibDate(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.In(jakarta).Format("02/01/2006")
}

// countDTO adalah satu baris daftar Status Register.
type countDTO struct {
	Code   string `json:"kode"`
	Name   string `json:"nama"`
	Count  int    `json:"jumlah"`
	Failed bool   `json:"gagal"`
}

// Counts menangani GET /api/inbox-admin/jumlah — daftar "Status Register" beserta
// jumlah baris setiap tab, dengan penyaring lini bisnis `bisnis`.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	alias, caller, ok := h.exportContext(w, r)
	if !ok {
		return
	}
	counts, err := h.service.Counts(r.Context(), alias, caller,
		r.URL.Query().Get("bisnis"), r.URL.Query().Get("kanwil"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	body := make([]countDTO, 0, len(counts))
	for _, c := range counts {
		body = append(body, countDTO{Code: c.Tab.Code, Name: c.Tab.Name, Count: c.Count, Failed: c.Failed})
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"jumlah": body, "portal": alias})
}

// Viewer menangani GET /api/inbox-admin/batas — batas data pemanggil dan isi dropdown
// "Pilih Kanwil". Layar memakainya untuk menggambar dropdown kanwil (manajer saja) dan
// menyebut cabang yang sedang membatasi antrean.
func (h *Handler) Viewer(w http.ResponseWriter, r *http.Request) {
	alias, caller, ok := h.exportContext(w, r)
	if !ok {
		return
	}
	info, err := h.service.Viewer(r.Context(), alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// Label "Kanwil N" mengikuti daftar tetap activity Pega `GetListKorwil`. Daftarnya
	// sendiri dibaca dari POOLDATA.BRANCH, bukan disalin: Pega hanya menulis Kanwil 1-5,
	// sedangkan basis data juga memakai nilai 6 — cabang-cabangnya tidak pernah dapat
	// dipilih di layar lama.
	regions := make([]map[string]string, 0, len(info.Regions))
	for _, code := range info.Regions {
		regions = append(regions, map[string]string{"kode": code, "label": "Kanwil " + code})
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{
		"manajer": info.Viewer.Manager,
		"cabang":  info.Viewer.BranchCode,
		"kanwil":  regions,
		"portal":  alias,
	})
}
