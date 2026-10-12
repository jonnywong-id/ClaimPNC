package monitoringslinkojkhttp

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/usecase"
)

// Tiga aksi tulis layar Monitoring SLINK OJK.

// WriteOutcomeDTO adalah hasil satu aksi tulis.
type WriteOutcomeDTO struct {
	// Created dan Updated dipisah karena artinya berbeda bagi pelapor: `Updated` adalah
	// klaim yang SUDAH pernah dilaporkan dan kini dilaporkan lagi — dan itulah yang
	// patut diperiksa sebelum berkasnya dikirim ke OJK.
	Created int `json:"baru"`
	Updated int `json:"diperbarui"`
	Total   int `json:"total"`

	Skipped []SkippedRowDTO `json:"ditolak"`
}

// SkippedRowDTO adalah satu baris yang tidak dapat disusun.
type SkippedRowDTO struct {
	// Line adalah nomor baris pada berkas unggahan; 0 untuk baris yang bukan dari berkas.
	Line    int    `json:"baris,omitempty"`
	ClaimID string `json:"no_klaim,omitempty"`
	Reason  string `json:"alasan"`
}

func toWriteOutcomeDTO(outcome monitoringslinkojk.WriteOutcome) WriteOutcomeDTO {
	skipped := make([]SkippedRowDTO, 0, len(outcome.Skipped))
	for _, row := range outcome.Skipped {
		skipped = append(skipped, SkippedRowDTO{
			Line:    row.Line,
			ClaimID: row.ClaimID,
			Reason:  row.Reason,
		})
	}
	return WriteOutcomeDTO{
		Created: outcome.Created,
		Updated: outcome.Updated,
		Total:   outcome.Total(),
		Skipped: skipped,
	}
}

// Process menangani POST /monitoring-slink-ojk/proses — tombol "Proses Data Klaim".
//
// Penyaringnya dibaca dari query string, SAMA seperti "Cari Data". Itu disengaja: yang
// tersusun harus persis yang terlihat, dan satu-satunya cara menjaminnya adalah kedua
// tombol membaca penyaring yang sama.
func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	request, err := readRequest(r, alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	outcome, err := h.service.Process(r.Context(), request, h.now())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toWriteOutcomeDTO(outcome))
}

// Upload menangani POST /monitoring-slink-ojk/unggah — tombol "Upload Data Klaim".
//
// Berkasnya berbentuk CSV dengan kepala kolom yang sama seperti berkas "Format File".
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	file, err := h.readUpload(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	rows, err := parseUpload(file)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	request := usecase.Request{PortalAlias: alias, Caller: caller}
	outcome, err := h.service.Upload(r.Context(), request, rows, h.now())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toWriteOutcomeDTO(outcome))
}

// maxUploadBytes membatasi ukuran berkas unggahan.
//
// Ia penjaga terhadap permintaan yang menghabiskan memori, bukan aturan bisnis. Berkas
// laporan SLIK berisi satu baris per fasilitas kredit; 16 MiB jauh di atas kebutuhan
// wajarnya, dan tetap jauh di bawah ukuran yang dapat menjatuhkan proses.
const maxUploadBytes = 16 << 20

// readUpload mengambil berkas dari permintaan multipart.
func (h *Handler) readUpload(r *http.Request) (io.Reader, error) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return nil, monitoringslinkojk.NewValidationError([]monitoringslinkojk.Violation{{
			Field:   monitoringslinkojk.FieldFile,
			Message: "Berkas tidak dapat dibaca. Pastikan ukurannya wajar lalu coba lagi.",
		}})
	}

	file, _, err := r.FormFile("berkas")
	if err != nil {
		return nil, monitoringslinkojk.NewValidationError([]monitoringslinkojk.Violation{{
			Field:   monitoringslinkojk.FieldFile,
			Message: "Berkas belum dipilih.",
		}})
	}
	return file, nil
}

// parseUpload membaca berkas CSV menjadi baris laporan.
//
// # Kepala kolomnya DIBACA menurut nama — dan begitu pula Pega
//
// Ini sempat saya catat sebagai penambahan di luar Pega. Itu KELIRU.
// `PNCUploadAutoClaimSlikOJK` memanggil `pxUploadCSVResults`, lalu membaca tiap baris
// lewat NAMA PROPERTI (`.ContractNo`, `.Keterangan`, dan seterusnya) — bukan lewat nomor
// kolom. Pemetaan menurut nama karena itu adalah perilaku Pega, bukan penyimpangan
// darinya.
//
// Kepala kolom dicocokkan tanpa membedakan huruf besar-kecil dan tanpa spasi tepi.
func parseUpload(source io.Reader) ([]usecase.UploadRow, error) {
	reader := csv.NewReader(source)
	// Panjang baris yang berbeda ditangani di sini, bukan ditolak pembaca CSV: berkas
	// lembar kerja sering menyimpan baris terakhir tanpa koma penutup.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, monitoringslinkojk.NewValidationError([]monitoringslinkojk.Violation{{
			Field:   monitoringslinkojk.FieldFile,
			Message: "Berkas bukan CSV yang sah.",
		}})
	}
	if len(records) < 2 {
		return nil, monitoringslinkojk.NewValidationError([]monitoringslinkojk.Violation{{
			Field:   monitoringslinkojk.FieldFile,
			Message: "Berkas tidak memuat satu baris data pun di bawah kepala kolomnya.",
		}})
	}

	index := map[string]int{}
	for i, header := range records[0] {
		index[normalizeHeader(header)] = i
	}

	if _, exists := index[normalizeHeader("ClaimID")]; !exists {
		if _, alt := index[normalizeHeader("No Klaim")]; !alt {
			return nil, monitoringslinkojk.NewValidationError(
				[]monitoringslinkojk.Violation{{
					Field: monitoringslinkojk.FieldFile,
					Message: "Kepala kolom tidak dikenali. Unduh berkas lewat tombol " +
						"\"Format File\", isi, lalu unggah kembali.",
				}})
		}
	}

	rows := make([]usecase.UploadRow, 0, len(records)-1)
	for i, record := range records[1:] {
		cell := func(names ...string) string {
			for _, name := range names {
				position, exists := index[normalizeHeader(name)]
				if exists && position < len(record) {
					return strings.TrimSpace(record[position])
				}
			}
			return ""
		}

		rows = append(rows, usecase.UploadRow{
			Line: i + 1,
			Entry: monitoringslinkojk.ReportEntry{
				ClaimID:               cell("ClaimID", "No Klaim"),
				ContractNo:            cell("ContractNo", "Contract No"),
				FacilityAccountNo:     cell("NOMORREKENINGFASILITAS"),
				DebtorCIF:             cell("NOMORCIFDEBITUR"),
				FacilityTypeCode:      cell("KodeJenisFasilitas"),
				FundSource:            cell("SumberDana"),
				PolicyStart:           cell("AwalPolis"),
				PolicyEnd:             cell("AkhirPolis"),
				InterestRate:          cell("SukuBunga"),
				CurrencyCode:          cell("KODEVALUTA"),
				OriginalCurrencyValue: cell("NilaiMataUangAsal"),
				Obligation:            cell("JUMLAHKEWAJIBAN", "ClaimAmount"),
				CollectibilityCode:    cell("KodeKolektibilitas"),
				DefaultDate:           cell("TanggalMacet"),
				DefaultReasonCode:     cell("KodeSebabMacet"),
				Arrears:               cell("TUNGGAKAN"),
				ArrearsDays:           cell("JumlahHariTunggakan"),
				ConditionDate:         cell("TanggalKondisi", "TanggalBayarKlaim"),
				ConditionCode:         cell("KodeKondisi"),
				BranchCode:            cell("KodeKantorCabang"),
				Remark:                cell("Keterangan"),
				IDCardNo:              cell("NoKTP"),
				CompanyNPWP:           cell("NPWPPerusahaan"),

				// Recovery TIDAK ada di berkas format. Ia dibaca bila pelapor
				// menambahkan kolomnya sendiri, dan kosong berarti nol — sehingga
				// tunggakan tidak berkurang.
				Recovery: cell("RecoveryClaim"),

				// PolicyNo dan ClientID sengaja TIDAK diisi di sini.
				//
				// `PNCUploadAutoClaimSlikOJK` menetapkan 27 properti pada
				// `TempDataSlinkD01`, dan kedua ini tidak termasuk — sehingga
				// `nopolis` dan `clientid` masuk kosong pada jalur unggah, meski
				// kolomnya ada di INSERT dan terisi pada jalur "Proses Data Klaim".
				//
				// Berkas contoh memang memuat kolom `PolicyNo`, jadi nilainya
				// tersedia. Ia tetap tidak dipakai karena Work Owner meminta jalur
				// ini disamakan dengan Pega apa adanya (2026-09-27).
			},
		})
	}
	return rows, nil
}

// normalizeHeader menyamakan bentuk kepala kolom sebelum dicocokkan.
func normalizeHeader(header string) string {
	return strings.ToUpper(strings.Join(strings.Fields(header), ""))
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// SubmitRequest adalah badan permintaan tombol "SLIK OJK".
type SubmitRequest struct {
	ClaimID    string `json:"no_klaim"`
	ContractNo string `json:"contract_no"`
}

// SubmitResponse adalah jawaban pengiriman yang berhasil.
type SubmitResponse struct {
	SubmissionID  int64  `json:"id_pengiriman"`
	ClaimID       string `json:"no_klaim"`
	ContractNo    string `json:"contract_no"`
	ClientID      string `json:"client_id,omitempty"`
	TransactionID string `json:"id_transaksi,omitempty"`
}

// Submit menangani POST /monitoring-slink-ojk/kirim — tombol "SLIK OJK".
//
// # Dua bentuk permintaan, dan keduanya disengaja
//
// Badan BERISI `no_klaim` dan `contract_no` mengirim satu klaim itu saja. Badan KOSONG
// mengirim seluruh klaim yang cocok dengan penyaring di query string — sasaran yang sama
// dengan "Proses Data Klaim", dan itulah yang dipakai tombolnya di layar.
//
// Keduanya ada karena cara Pega memilih klaimnya di layar ini **tidak dapat dipulihkan**:
// sectionnya tidak menghubungkan tombol ini ke aktivitas mana pun. Bentuk satu-klaim
// dipertahankan supaya pemanggil lain — perkakas, pengujian, atau layar detail kelak —
// dapat mengirim satu klaim tanpa menunggu keputusan itu diambil.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	var body SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		h.writeError(w, r, monitoringslinkojk.NewValidationError(
			[]monitoringslinkojk.Violation{{
				Field:   monitoringslinkojk.FieldClaimID,
				Message: "Permintaan tidak dapat dibaca.",
			}}))
		return
	}

	// Badan kosong berarti "kirim yang sedang tampil".
	if strings.TrimSpace(body.ClaimID) == "" && strings.TrimSpace(body.ContractNo) == "" {
		request, err := readRequest(r, alias, caller)
		if err != nil {
			h.writeError(w, r, err)
			return
		}

		outcome, err := h.service.SubmitFiltered(r.Context(), request)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeJSON(w, r, http.StatusOK, toWriteOutcomeDTO(outcome))
		return
	}

	request := usecase.Request{PortalAlias: alias, Caller: caller}
	submission, result, err := h.service.Submit(
		r.Context(), request, body.ClaimID, body.ContractNo)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SubmitResponse{
		SubmissionID:  submission.ID,
		ClaimID:       submission.ClaimID,
		ContractNo:    submission.ContractNo,
		ClientID:      result.ClientID,
		TransactionID: result.TransactionID,
	})
}
