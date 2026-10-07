package casestudyclaimhttp

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/casestudyclaim/usecase"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/platform/logging"
	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor sebagai tipe konkret,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, q usecase.Query) (casestudyclaim.Page, error)
	SaveRemark(ctx context.Context, portalAlias string, caller usecase.Caller, claimNumber, remark string) error
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Metadata menangani GET /api/case-study-claim/penyaring.
//
// Ia TIDAK menyentuh basis data sama sekali: seluruh isinya dibaca dari kode, tempat
// pembacaan export Pega tercatat. Karena itu ia tetap menjawab meski Oracle sedang tidak
// dapat dihubungi — layar tergambar lengkap, dan yang gagal hanyalah isinya.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	business := casestudyclaim.BusinessOptions()
	status := casestudyclaim.StatusOptions()
	columns := casestudyclaim.Columns()

	body := metadataResponse{
		Bisnis:           make([]optionDTO, 0, len(business)),
		Status:           make([]optionDTO, 0, len(status)),
		Kolom:            make([]columnDTO, 0, len(columns)),
		AmbangNilaiKlaim: casestudyclaim.LargeClaimThreshold.MinorUnits(),

		CatatanPeriode: "Dari kedua tanggal yang dipilih, yang dipakai menyaring hanya " +
			"TAHUNNYA. Memilih 1 Januari sampai 31 Januari 2024 menampilkan seluruh " +
			"klaim tahun 2024. Perilaku ini sama dengan layar lama.",

		CatatanKolomKembar: "Kolom \"Nature of Loss\" dan \"Cause of Loss\" selalu berisi " +
			"teks yang sama. Di layar lama keduanya pun terikat pada satu kolom yang " +
			"sama, dan itu dipertahankan apa adanya.",
	}
	for _, item := range business {
		body.Bisnis = append(body.Bisnis, optionDTO{Kode: item.Code, Label: item.Label})
	}
	for _, item := range status {
		body.Status = append(body.Status, optionDTO{Kode: item.Code, Label: item.Label})
	}
	for _, item := range columns {
		body.Kolom = append(body.Kolom, columnDTO{
			Kunci: item.Key,
			Judul: item.Title,
			Jenis: string(item.Kind),
		})
	}

	h.WriteJSON(w, r, http.StatusOK, body)
}

// List menangani GET /api/case-study-claim.
//
// Menggantikan `RDB List/BrowseClaimStudy-SQL.xml` beserta `Activity/StudyClaim_act`. Yang
// di sana menjadi penyisipan dua potongan SQL dari properti klipboard, di sini menjadi satu
// kueri berparameter.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.WriteJSON, w, r, err.field, err.message)
		return
	}

	page, listErr := h.Service.List(r.Context(), q)
	if listErr != nil {
		h.WriteError(w, r, listErr)
		return
	}

	rows := make([]rowDTO, 0, len(page.Rows))
	for _, row := range page.Rows {
		rows = append(rows, toRowDTO(row, h.Location))
	}

	h.WriteJSON(w, r, http.StatusOK, listResponse{
		Baris: rows,
		Total: page.Total,
		Periode: periodDTO{
			TahunAwal:  q.FromYear,
			TahunAkhir: q.ToYear,
		},
	})
}

// exportMaxRows membatasi banyaknya baris yang diekspor.
//
// # Kenapa ada batas, dan kenapa angkanya bukan warisan
//
// Sistem lama membatasi 500 baris lewat `pyMaxRecords` pada 54 dari 56 laporannya,
// sehingga kebutuhan export bervolume besar BELUM PERNAH benar-benar dilayani dan tidak
// ada data historis yang sahih untuk menentukan angkanya (`15-NFR` §3.2).
//
// Angka di bawah adalah batas pengaman yang dipilih sadar. Ia lebih longgar daripada yang
// dibutuhkan layar ini pada praktiknya: penyaring ambang Rp 5 miliar memangkas populasinya
// menjadi sangat kecil, dan hasil yang mendekati batas ini justru pertanda penyaringnya
// tidak bekerja.
//
// Berapa baris maksimum yang WAJIB dilayani adalah pertanyaan terbuka `ADR-0011`.
const exportMaxRows = 10000

// Export menangani GET /api/case-study-claim/unduh.
//
// # Kenapa CSV
//
// Tombol di layar lama berbunyi "Export Data", dan yang dipanggilnya adalah
// `Activity/ExportDataCaseStudyClaim-Act.xml` yang seluruh isinya `pxConvertResultsToCSV` —
// keluarannya CSV. Memakai `encoding/csv` dari pustaka standar karena itu SETARA dengan
// sistem lama, bukan penyederhanaan, dan tidak menambah satu pun dependensi.
//
// # Kenapa penyaringnya SAMA PERSIS dengan daftar
//
// Karena di Pega pun demikian: activity export membaca halaman hasil yang sudah diisi
// `StudyClaim_act`, bukan menjalankan kueri lain. Berkas yang diunduh karena itu wajib
// berisi klaim yang sama dengan yang baru dilihat pengguna — hanya tanpa batas halaman.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.WriteJSON, w, r, err.field, err.message)
		return
	}

	// Halaman diabaikan saat mengekspor: yang diminta adalah seluruh hasil yang cocok,
	// bukan halaman yang sedang dilihat.
	q.Offset = 0
	q.Limit = casestudyclaim.MaxExportBatch

	first, listErr := h.Service.List(r.Context(), q)
	if listErr != nil {
		h.WriteError(w, r, listErr)
		return
	}

	// Header ditulis SEBELUM baris pertama dikirim. Setelah badan respons mulai mengalir,
	// status HTTP tidak dapat diubah lagi — sehingga galat yang terjadi di tengah tidak
	// dapat dijawab dengan 500. Itu diterima, dan alasannya ada di bawah.
	filename := "case-study-claim-" + h.Now().In(h.Location).Format("20060102-150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(casestudyclaim.ColumnTitles()); err != nil {
		h.logExportInterrupted(r, err)
		return
	}

	written := 0
	page := first

	for {
		for _, row := range page.Rows {
			if written >= exportMaxRows {
				return
			}
			if err := writer.Write(exportRow(toRowDTO(row, h.Location))); err != nil {
				// Sambungan putus di tengah unduhan adalah kejadian biasa — pengguna
				// menutup tab. Ia dicatat sebagai peringatan, bukan galat, dan tidak dapat
				// diberitahukan ke klien karena badan respons sudah mengalir.
				h.logExportInterrupted(r, err)
				return
			}
			written++
		}

		// Baris yang diterima lebih sedikit dari yang diminta berarti sudah habis.
		if len(page.Rows) < q.Limit || written >= page.Total || written >= exportMaxRows {
			return
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportInterrupted(r, err)
			return
		}

		q.Offset = written
		page, listErr = h.Service.List(r.Context(), q)
		if listErr != nil {
			h.logExportInterrupted(r, listErr)
			return
		}
	}
}

// exportRow menyusun satu baris CSV.
//
// Urutannya mengikuti `casestudyclaim.Columns()`, yaitu urutan kolom di layar — berkas yang
// diunduh harus terbaca sebagai salinan apa yang dilihat pengguna, bukan susunan lain yang
// harus dicocokkan sendiri.
//
// Nilai uang ditulis sebagai angka berdesimal titik, bukan "Rp 1.000.000,00": berkas ini
// dibuka di pengolah angka, dan teks bersatuan tidak dapat dijumlahkan di sana.
func exportRow(row rowDTO) []string {
	return []string{
		row.NomorKlaim,
		row.NomorPolis,
		row.NamaTertanggung,
		row.COB,
		row.PeriodePolis,
		row.BulanKlaim,
		row.TanggalKejadian,
		row.SOB,
		row.PosisiReasuransi,
		row.NatureOfLoss,
		row.CauseOfLoss,
		decimalText(row.TSI, 2),
		decimalText(row.ShareASM, 4),
		decimalText(row.Deductible, 2),
		decimalText(row.NilaiShareASM, 2),
		decimalText(row.NilaiKlaim100, 2),
		decimalText(row.AdjusterFee100, 2),
		decimalText(row.NilaiKlaimNet100, 2),
		decimalText(row.NilaiKlaimNetShareASM, 2),
		decimalText(row.LackOfDoc, 2),
		row.Cabang,
		row.StatusKlaim,
		row.Kronologi,
		row.Remark,
	}
}

// decimalText menuliskan bilangan bulat berskala menjadi teks berdesimal.
//
// `decimalText(123456, 2)` menghasilkan `"1234.56"`, dan `decimalText(400000, 4)`
// menghasilkan `"40"` — empat desimal nol di belakang dipangkas supaya persentase bulat
// tidak tampil sebagai "40.0000".
//
// Nilai nil menghasilkan sel KOSONG, bukan "0". Pada berkas yang dibuka di pengolah angka,
// nol adalah angka yang ikut terhitung dalam rata-rata dan penjumlahan; "belum ada
// nilainya" bukan nol.
func decimalText(value *int64, scale int) string {
	if value == nil {
		return ""
	}

	factor := int64(1)
	for i := 0; i < scale; i++ {
		factor *= 10
	}

	amount := *value
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}

	whole := amount / factor
	fraction := amount % factor
	if fraction == 0 {
		return sign + strconv.FormatInt(whole, 10)
	}

	text := fmt.Sprintf("%0*d", scale, fraction)
	text = strings.TrimRight(text, "0")
	return sign + strconv.FormatInt(whole, 10) + "." + text
}

// SaveRemark menangani PUT /api/case-study-claim/{nomor}/catatan.
//
// Menggantikan tombol **Save** pada setiap baris grid
// (`Activity/SaveRemarksRecommendation_act`).
func (h *Handler) SaveRemark(w http.ResponseWriter, r *http.Request) {
	caller, found := h.Caller(r.Context())
	if !found || strings.TrimSpace(caller.Login) == "" {
		writeBadRequest(h.WriteJSON, w, r, "", "Identitas pemanggil tidak dikenali.")
		return
	}

	activePortal, portalFound := portalhttp.ActivePortalFrom(r.Context())
	if !portalFound {
		writeBadRequest(h.WriteJSON, w, r, "", "Portal aktif tidak dikenali.")
		return
	}

	number := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if number == "" {
		writeBadRequest(h.WriteJSON, w, r, "nomor_klaim", "Nomor klaim tidak disebutkan.")
		return
	}

	var body saveRemarkRequest
	decoder := json.NewDecoder(r.Body)

	// Field asing DITOLAK, tidak diabaikan. Klien yang mengirim `{"remark": …}` alih-alih
	// `{"catatan": …}` akan mengira catatannya tersimpan padahal yang tersimpan adalah teks
	// kosong — dan kosong di sini berarti MENGHAPUS catatan yang sudah ada.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeBadRequest(h.WriteJSON, w, r, "catatan", "Badan permintaan tidak dapat dibaca.")
		return
	}

	remark := strings.TrimSpace(body.Catatan)
	if err := h.Service.SaveRemark(
		r.Context(), activePortal.Alias, usecase.Caller{Login: caller.Login}, number, remark,
	); err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, saveRemarkResponse{
		NomorKlaim: number,
		Catatan:    remark,
	})
}

// paramError adalah kesalahan pembacaan query string, beserta isian yang menyebabkannya.
type paramError struct {
	field   string
	message string
}

func (e *paramError) Error() string { return e.message }

// queryFrom membaca penyaring dan paginasi dari query string.
//
// # Di sinilah tanggal menjadi TAHUN
//
// Pengguna memilih dua tanggal; yang dipakai menyaring hanya tahunnya, persis seperti
// kueri lama yang membungkus keduanya dengan `TO_CHAR(TO_DATE(…),'yyyy')`. Pengambilan
// tahunnya terjadi DI SINI, sekali, memakai zona waktu tampilan — sehingga 1 Januari yang
// dipilih pengguna WIB tidak terbaca sebagai 31 Desember tahun sebelumnya (`R-12`).
func (h *Handler) queryFrom(r *http.Request) (usecase.Query, *paramError) {
	activePortal, portalFound := portalhttp.ActivePortalFrom(r.Context())
	if !portalFound {
		// Portal aktif sudah diperiksa middleware; ketiadaannya di sini berarti rute
		// dipasang di luar middleware itu — cacat perakitan, bukan kesalahan pengguna. Ia
		// ditolak, tidak pernah dilayani portal utama sebagai cadangan (`R-20`).
		return usecase.Query{}, &paramError{message: "Portal aktif tidak dikenali."}
	}

	values := r.URL.Query()

	from, err := dateParam(values.Get("dari"), h.Location)
	if err != nil {
		return usecase.Query{}, &paramError{
			field:   "dari",
			message: "Tanggal Awal tidak sah; formatnya YYYY-MM-DD.",
		}
	}
	to, err := dateParam(values.Get("sampai"), h.Location)
	if err != nil {
		return usecase.Query{}, &paramError{
			field:   "sampai",
			message: "Tanggal Akhir tidak sah; formatnya YYYY-MM-DD.",
		}
	}

	query := usecase.Query{PortalAlias: activePortal.Alias}
	if from != nil {
		query.FromYear = casestudyclaim.YearOf(*from, h.Location)
	}
	if to != nil {
		query.ToYear = casestudyclaim.YearOf(*to, h.Location)
	}

	business, known := casestudyclaim.FindBusinessScope(values.Get("bisnis"))
	if !known {
		return usecase.Query{}, &paramError{
			field:   "bisnis",
			message: "Pilihan Bisnis tidak dikenal.",
		}
	}
	query.Business = business

	status, known := casestudyclaim.FindClaimStatus(values.Get("status"))
	if !known {
		return usecase.Query{}, &paramError{
			field:   "status",
			message: "Pilihan Status tidak dikenal.",
		}
	}
	query.Status = status

	limit, err := positiveInt(values.Get("batas"), casestudyclaim.DefaultLimit)
	if err != nil {
		return usecase.Query{}, &paramError{field: "batas", message: "Parameter batas tidak sah."}
	}
	if limit > casestudyclaim.MaxLimit {
		// Ditolak, bukan dipangkas diam-diam (`10-API-STRATEGY.md` §4): klien yang meminta
		// seribu baris lalu menerima seratus tanpa diberi tahu akan menampilkan daftar yang
		// ia kira lengkap.
		return usecase.Query{}, &paramError{
			field:   "batas",
			message: fmt.Sprintf("Parameter batas melebihi maksimum %d.", casestudyclaim.MaxLimit),
		}
	}
	query.Limit = limit

	offset, err := positiveInt(values.Get("lewati"), 0)
	if err != nil {
		return usecase.Query{}, &paramError{field: "lewati", message: "Parameter lewati tidak sah."}
	}
	query.Offset = offset

	return query, nil
}

// dateParam membaca tanggal YYYY-MM-DD sebagai awal hari pada zona tampilan.
//
// Kosong berarti tidak diisi, bukan galat bentuk — yang menolak isian kosong adalah
// pemeriksaan periode di domain, dengan pesan yang menjelaskan sebabnya.
func dateParam(raw string, loc *time.Location) (*time.Time, error) { return httpquery.Date(raw, loc) }

// positiveInt membaca bilangan bulat tak negatif; kosong berarti nilai baku.
func positiveInt(raw string, fallback int) (int, error) {
	return httpquery.NonNegativeOr(raw, fallback)
}

// logExportInterrupted mencatat export yang berhenti di tengah.
func (h *Handler) logExportInterrupted(r *http.Request, err error) {
	logging.From(r.Context(), h.Logger).Warn("export berhenti sebelum selesai",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
