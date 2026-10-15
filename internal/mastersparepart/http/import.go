package masterspareparthttp

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/mastersparepart"
	usecase "claim-pnc/internal/mastersparepart/usecase"
	portalhttp "claim-pnc/internal/portal/http"
)

// Kode galat jalur unggah CSV.
//
// Dipisahkan dari kode jalur unggah dokumen: yang gagal di sini adalah BERKASNYA sebagai
// keseluruhan, bukan penyimpanannya, dan yang dapat dilakukan pengguna pun berbeda —
// memperbaiki berkasnya, bukan mengulang unggahannya.
const (
	// CodeCSVInvalid: berkasnya tidak dapat dibaca sebagai CSV master sparepart.
	CodeCSVInvalid = "csv_tidak_sah"
	// CodeCSVTooManyRows: barisnya melebihi batas satu unggahan.
	CodeCSVTooManyRows = "csv_terlalu_banyak_baris"
)

// ImportCSV menerima satu berkas CSV dan meng-upsert seluruh barisnya.
//
// # Jawabannya 200, bukan 201, bahkan saat ada baris yang gagal
//
// Permintaannya sendiri BERHASIL diproses: setiap baris dibaca, dan hasilnya dilaporkan satu
// per satu. Menjawab 4xx karena sebagian baris ditolak akan membuat layar memperlakukan
// seluruh unggahan sebagai gagal, padahal sebagian datanya sudah masuk — dan pengguna yang
// mengunggah ulang karenanya akan menimpa baris yang sudah benar.
//
// Yang menjadikannya 4xx hanyalah berkas yang tidak dapat dibaca sama sekali.
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists || strings.TrimSpace(active.Alias) == "" {
		h.writeModuleError(w, r, errors.New("mastersparepart/http: portal aktif tidak dikenali"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)

	file, _, err := r.FormFile("berkas")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, &mastersparepart.DocumentUploadError{
			Kind:    mastersparepart.UploadInvalid,
			Message: `Berkas tidak ditemukan pada permintaan. Sertakan bagian bernama "berkas".`,
			Err:     err,
		})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, err)
		return
	}

	by, known := h.actor(r)
	if !known {
		h.writeError(w, r, errors.New("mastersparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	report, err := h.service.ImportCSV(r.Context(), active.Alias, content, by, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ImportResponse{Data: toImportDTO(report)})
}

// mapCSVError memetakan kegagalan membaca berkas menjadi status dan badan respons.
func mapCSVError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, mastersparepart.ErrCSVEmpty):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeCSVInvalid,
			Message: "Berkas tidak memuat satu baris data pun.",
			Detail:  []ViolationDTO{{Field: "berkas", Message: "Berkas kosong."}},
		}, true

	case errors.Is(err, mastersparepart.ErrCSVHeaderMissing):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeCSVInvalid,
			// Nama header yang kurang ikut disebut: tanpa itu pengguna harus menebak
			// kolom mana yang salah ketik di antara sembilan belas.
			Message: "Header berkas tidak lengkap. " + headerHint(err),
			Detail:  []ViolationDTO{{Field: "berkas", Message: headerHint(err)}},
		}, true

	case errors.Is(err, mastersparepart.ErrCSVTooManyRows):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeCSVTooManyRows,
			Message: "Berkas memuat lebih dari " +
				strconv.Itoa(mastersparepart.MaxCSVRows) +
				" baris. Pecah berkasnya lalu unggah bergantian.",
			Detail: []ViolationDTO{{Field: "berkas", Message: "Terlalu banyak baris."}},
		}, true
	}
	return 0, ErrorResponse{}, false
}

// headerHint mengambil daftar header yang kurang dari pesan galatnya.
//
// Galat domainnya berbentuk `...: NAMA_SPART, HARGA_JUAL`, dan yang dibutuhkan layar hanya
// bagian setelah titik dua.
func headerHint(err error) string {
	pesan := err.Error()
	if potong := strings.LastIndex(pesan, ": "); potong >= 0 {
		return "Kolom yang kurang: " + pesan[potong+2:] + "."
	}
	return "Periksa baris pertama berkasnya."
}

// ImportRowDTO adalah hasil satu baris.
type ImportRowDTO struct {
	Baris int    `json:"baris"`
	Nomor string `json:"nomor_sparepart"`
	ID    string `json:"id_sparepart"`
	Hasil string `json:"hasil"`
	Pesan string `json:"pesan"`
}

// ImportReportDTO meringkas satu unggahan.
type ImportReportDTO struct {
	Total      int            `json:"total"`
	Baru       int            `json:"baru"`
	Diperbarui int            `json:"diperbarui"`
	Gagal      int            `json:"gagal"`
	Baris      []ImportRowDTO `json:"baris"`
}

// ImportResponse membungkus laporan unggahan.
type ImportResponse struct {
	Data ImportReportDTO `json:"data"`
}

// toImportDTO memindahkan laporan domain ke bentuk kawat.
func toImportDTO(report usecase.ImportReport) ImportReportDTO {
	baris := make([]ImportRowDTO, 0, len(report.Rows))
	for _, row := range report.Rows {
		baris = append(baris, ImportRowDTO{
			Baris: row.Line,
			Nomor: row.Number,
			ID:    row.ID,
			Hasil: string(row.Outcome),
			Pesan: row.Message,
		})
	}
	return ImportReportDTO{
		Total:      report.Total,
		Baru:       report.Created,
		Diperbarui: report.Updated,
		Gagal:      report.Failed,
		Baris:      baris,
	}
}
