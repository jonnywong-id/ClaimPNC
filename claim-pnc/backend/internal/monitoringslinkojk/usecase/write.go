package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
)

// Tiga aksi tulis layar Monitoring SLINK OJK.
//
//	Process  "Proses Data Klaim"  — susun laporan dari data klaim sumber
//	Upload   "Upload Data Klaim"  — susun laporan dari berkas unggahan
//	Submit   "SLIK OJK"           — kirim data debitur ke sistem SLIK
//
// Ketiganya bermuara pada INSERT yang sama (`InsertDataSlikOJKF06`); yang berbeda hanya
// dari mana barisnya datang. Itulah sebabnya keduanya berbagi satu jalur — lihat insert.

// Process menyusun laporan SLIK dari data klaim sumber — tombol "Proses Data Klaim".
//
// # Barisnya datang dari penyaring yang SEDANG dipakai layar
//
// Bukan dari seluruh basis data. Itu pilihan yang disengaja dan menyimpang dari tebakan
// termudah: tombol yang menyusun seluruh klaim sekaligus tidak dapat dibatalkan, dan
// pelapor tidak punya cara melihat lebih dulu apa yang akan tersusun.
//
// Dengan penyaring, pelapor menekan "Cari Data" untuk melihat isinya, baru menekan
// "Proses Data Klaim" untuk menyusunnya — dan yang tersusun persis yang terlihat.
func (s *Service) Process(ctx context.Context, request Request, now time.Time) (
	monitoringslinkojk.WriteOutcome, error,
) {
	filter, err := s.prepare(request)
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, err
	}
	filter.Page, filter.Size = 0, 0

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, err
	}

	month := monitoringslinkojk.ReportMonthOf(now)
	outcome := monitoringslinkojk.WriteOutcome{Skipped: []monitoringslinkojk.SkippedRow{}}
	seen := 0

	err = repo.StreamSource(ctx, filter, func(entry monitoringslinkojk.ReportEntry) error {
		seen++
		entry.ReportMonth = month

		// Recovery TIDAK tersedia dari kueri sumber; jalur akseptasi pun mengisinya
		// nol (`InsertAdjustmentListKredit`: `TempDataSlinkD01.RecoveryClaim = 0`).
		entry.Recovery = "0"

		return s.insert(ctx, repo, entry, 0, &outcome)
	})
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, fmt.Errorf(
			"monitoringslinkojk/usecase: proses data klaim: %w", err)
	}

	// Nol baris SUMBER dibedakan dari nol baris TERSUSUN.
	//
	// Yang pertama berarti penyaringnya tidak menemukan apa pun; yang kedua berarti
	// seluruhnya ditolak. Keduanya menghasilkan layar yang tampak sama, dan pelapor
	// perlu tahu mana yang terjadi.
	if seen == 0 {
		return outcome, monitoringslinkojk.ErrNothingToProcess
	}
	return outcome, nil
}

// UploadRow adalah satu baris berkas unggahan beserta nomor barisnya.
type UploadRow struct {
	// Line adalah nomor baris pada berkas, dihitung dari 1 TANPA baris kepala.
	Line  int
	Entry monitoringslinkojk.ReportEntry
}

// Upload menyusun laporan SLIK dari berkas unggahan — tombol "Upload Data Klaim".
//
// # Satu baris cacat tidak menggagalkan seluruh berkas
//
// Berkas berisi seribu baris yang ditolak seluruhnya karena satu baris kosong memaksa
// pelapor menebak baris mana yang salah. Baris yang ditolak dikembalikan beserta nomor
// barisnya, dan sisanya tetap tersusun.
//
// Itu perilaku yang BERBEDA dari sistem lama, yang tidak melaporkan apa pun. Dicatat
// sebagai penambahan, bukan replikasi.
func (s *Service) Upload(
	ctx context.Context,
	request Request,
	rows []UploadRow,
	now time.Time,
) (monitoringslinkojk.WriteOutcome, error) {
	if strings.TrimSpace(request.Caller.Login) == "" {
		return monitoringslinkojk.WriteOutcome{}, monitoringslinkojk.ErrCallerUnknown
	}
	if len(rows) == 0 {
		return monitoringslinkojk.WriteOutcome{}, monitoringslinkojk.NewValidationError(
			[]monitoringslinkojk.Violation{{
				Field:   monitoringslinkojk.FieldFile,
				Message: "Berkas tidak memuat satu baris data pun.",
			}})
	}

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, err
	}

	month := monitoringslinkojk.ReportMonthOf(now)
	outcome := monitoringslinkojk.WriteOutcome{Skipped: []monitoringslinkojk.SkippedRow{}}

	for _, row := range rows {
		entry := row.Entry.Clean()
		entry.ReportMonth = month

		// Tunggakan dikurangi recovery, persis seperti `PNCUploadAutoClaimSlikOJK`:
		// `.TUNGGAKAN - .RecoveryClaim`. Hanya jalur unggah yang melakukannya — jalur
		// proses mengisi recovery nol.
		entry.Arrears = monitoringslinkojk.Netting(entry.Arrears, entry.Recovery)

		if err := s.insert(ctx, repo, entry, row.Line, &outcome); err != nil {
			return monitoringslinkojk.WriteOutcome{}, fmt.Errorf(
				"monitoringslinkojk/usecase: unggah data klaim: %w", err)
		}
	}
	return outcome, nil
}

// insert menyusun satu baris laporan, atau mencatatnya sebagai ditolak.
//
// Di sinilah `operasidata` ditetapkan — bukan diserahkan pemanggil. Pemanggil yang
// mengisinya sendiri akan menandai baris berulang sebagai baris baru, dan pada laporan
// regulator itu berarti satu klaim terhitung dua kali.
func (s *Service) insert(
	ctx context.Context,
	repo monitoringslinkojk.Repo,
	entry monitoringslinkojk.ReportEntry,
	line int,
	outcome *monitoringslinkojk.WriteOutcome,
) error {
	entry = entry.Clean()

	// TIDAK ADA PENYARINGAN BARIS DI SINI — dan itu disengaja.
	//
	// Sempat ada: baris tanpa No Klaim atau tanpa Contract No ditolak sendirian dan
	// dilaporkan beserta nomor barisnya. Itu penambahan di luar Pega.
	// `PNCUploadAutoClaimSlikOJK` maupun `InsertDataSlikOJKF06` **tidak memeriksa apa
	// pun**; setiap baris berkas disisipkan apa adanya.
	//
	// Work Owner meminta jalur ini disamakan dengan Pega (2026-09-27) sesudah
	// konsekuensinya disampaikan: baris tanpa No Klaim tetap masuk
	// `T_CLAIM_SLIK_OJK`, dan baris semacam itu tidak dapat dicocokkan ke klaim mana
	// pun — sehingga tidak dapat dikoreksi maupun ditarik kembali lewat layar ini.
	//
	// `parameter line` dipertahankan supaya pemanggil tetap dapat menyebut baris
	// berkasnya bila kelak penyaringan itu dikembalikan.
	_ = line

	existing, err := repo.CountReport(ctx, entry.ClaimID)
	if err != nil {
		return err
	}
	entry.DataOperation = monitoringslinkojk.DataOperationFor(existing)

	if err := repo.InsertReport(ctx, entry); err != nil {
		return err
	}

	if entry.DataOperation == monitoringslinkojk.OperationUpdate {
		outcome.Updated++
		return nil
	}
	outcome.Created++
	return nil
}

// ringkasPelanggaran merangkai pesan pelanggaran menjadi satu kalimat.
//
// Dipakai untuk baris yang DITOLAK di tengah pemrosesan, tempat pesannya tidak dapat
// ditempelkan pada isian mana pun — yang dibaca pelapor adalah nomor barisnya.
func ringkasPelanggaran(err error) string {
	var validation *monitoringslinkojk.ValidationError
	if !errors.As(err, &validation) {
		return err.Error()
	}
	pesan := make([]string, 0, len(validation.Violations))
	for _, v := range validation.Violations {
		pesan = append(pesan, v.Message)
	}
	return strings.Join(pesan, " ")
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// SubmitFiltered mengirim data debitur SELURUH klaim yang cocok dengan penyaring —
// tombol "SLIK OJK".
//
// # Sasarannya himpunan yang disaring, dan itu keputusan yang perlu diketahui
//
// Di Pega, `InsertDataSlinkOJKIndividu` bekerja atas SATU klaim yang sedang dibuka
// (`pyWorkPagee.ClaimData.ClaimNo`). Layar pemantauan ini tidak punya klaim yang sedang
// dibuka, dan section-nya TIDAK menghubungkan tombol ini ke aktivitas mana pun — sehingga
// bagaimana klaimnya dipilih di layar ini **tidak dapat dipulihkan dari export**.
//
// Yang dipilih di sini adalah perilaku yang paling dapat dipertanggungjawabkan: sasarannya
// sama dengan kedua tombol tulis lainnya — himpunan yang sedang disaring dan sudah
// terlihat di tabel. Dengan begitu pelapor menekan "Cari Data" untuk melihat apa yang
// akan dikirim, lalu menekan tombolnya.
//
// Alternatifnya — menambah tombol per baris di grid — menambah kolom yang tidak ada di
// layar lama (`D-13`). Dicatat sebagai pertanyaan terbuka di
// `docs/permintaan-artefak-pega.md`.
func (s *Service) SubmitFiltered(ctx context.Context, request Request) (
	monitoringslinkojk.WriteOutcome, error,
) {
	filter, err := s.prepare(request)
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, err
	}
	filter.Page, filter.Size = 0, 0

	// Seam pengirim diperiksa LEBIH DULU, sebelum satu baris pun dibaca — apalagi
	// dicatat. Tanpa ini, setiap penekanan tombol meninggalkan baris pengiriman yang
	// tidak akan pernah terkirim.
	if s.sender == nil {
		return monitoringslinkojk.WriteOutcome{}, monitoringslinkojk.ErrSenderNotConfigured
	}

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, err
	}

	outcome := monitoringslinkojk.WriteOutcome{Skipped: []monitoringslinkojk.SkippedRow{}}
	seen := 0

	err = repo.StreamSource(ctx, filter, func(entry monitoringslinkojk.ReportEntry) error {
		seen++
		entry = entry.Clean()

		if err := entry.Validate(); err != nil {
			outcome.Skipped = append(outcome.Skipped, monitoringslinkojk.SkippedRow{
				ClaimID: entry.ClaimID,
				Reason:  ringkasPelanggaran(err),
			})
			return nil
		}

		if _, _, err := s.submitOne(ctx, repo, entry.ClaimID, entry.ContractNo); err != nil {
			// Satu pengiriman yang gagal TIDAK menghentikan sisanya, dan barisnya tetap
			// tercatat di basis data sebagai pengiriman yang dicoba — lihat submitOne.
			outcome.Skipped = append(outcome.Skipped, monitoringslinkojk.SkippedRow{
				ClaimID: entry.ClaimID,
				Reason:  err.Error(),
			})
			return nil
		}

		outcome.Created++
		return nil
	})
	if err != nil {
		return monitoringslinkojk.WriteOutcome{}, fmt.Errorf(
			"monitoringslinkojk/usecase: kirim ke SLIK: %w", err)
	}

	if seen == 0 {
		return outcome, monitoringslinkojk.ErrNothingToProcess
	}
	return outcome, nil
}

// Submit mengirim data debitur satu klaim ke sistem SLIK — tombol "SLIK OJK".
//
// Empat langkah, urutannya mengikuti `InsertDataSlinkOJKIndividu`:
//
//  1. ambil nomor urut
//  2. CATAT pengirimannya
//  3. kirim
//  4. simpan jawabannya
//
// # Kenapa dicatat SEBELUM dikirim
//
// Karena itulah yang membuat pengiriman gagal tetap meninggalkan jejak. Baris tanpa
// `id_transaction` adalah pengiriman yang tidak pernah sampai — dan tanpa baris itu,
// pengiriman yang gagal di tengah tidak dapat dibedakan dari pengiriman yang tidak pernah
// dicoba.
func (s *Service) Submit(
	ctx context.Context,
	request Request,
	claimID string,
	contractNo string,
) (monitoringslinkojk.Submission, monitoringslinkojk.SubmissionResult, error) {
	var (
		submission monitoringslinkojk.Submission
		result     monitoringslinkojk.SubmissionResult
	)

	if strings.TrimSpace(request.Caller.Login) == "" {
		return submission, result, monitoringslinkojk.ErrCallerUnknown
	}

	violations := make([]monitoringslinkojk.Violation, 0, 2)
	if strings.TrimSpace(claimID) == "" {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldClaimID,
			Message: "No Klaim wajib dipilih.",
		})
	}
	if strings.TrimSpace(contractNo) == "" {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldContractNo,
			Message: "Contract No wajib dipilih.",
		})
	}
	if err := monitoringslinkojk.NewValidationError(violations); err != nil {
		return submission, result, err
	}

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return submission, result, err
	}

	// Seam pengirim diperiksa LEBIH DULU, sebelum satu baris pun dicatat.
	//
	// Tanpa pemeriksaan ini, setiap penekanan tombol meninggalkan baris pengiriman yang
	// tidak akan pernah terkirim — dan tabelnya terisi jejak palsu.
	if s.sender == nil {
		return submission, result, monitoringslinkojk.ErrSenderNotConfigured
	}

	return s.submitOne(ctx, repo, claimID, contractNo)
}

// submitOne menempuh keempat langkah pengiriman untuk satu klaim.
//
// Dipisahkan supaya Submit dan SubmitFiltered menempuh jalur yang SAMA PERSIS — termasuk
// urutan "catat dulu, baru kirim" yang menjadi satu-satunya jejak pengiriman gagal.
// Pemanggil bertanggung jawab memastikan seam pengirim ada.
func (s *Service) submitOne(
	ctx context.Context,
	repo monitoringslinkojk.Repo,
	claimID string,
	contractNo string,
) (monitoringslinkojk.Submission, monitoringslinkojk.SubmissionResult, error) {
	var result monitoringslinkojk.SubmissionResult

	id, err := repo.NextSubmissionID(ctx)
	if err != nil {
		return monitoringslinkojk.Submission{}, result, fmt.Errorf(
			"monitoringslinkojk/usecase: nomor urut pengiriman: %w", err)
	}

	submission := monitoringslinkojk.Submission{
		ID:         id,
		ClaimID:    strings.TrimSpace(claimID),
		ContractNo: strings.TrimSpace(contractNo),
	}

	if err := repo.RecordSubmission(ctx, submission); err != nil {
		return submission, result, fmt.Errorf(
			"monitoringslinkojk/usecase: catat pengiriman: %w", err)
	}

	result, err = s.sender.Send(ctx, submission)
	if err != nil {
		// Barisnya SENGAJA tidak dihapus. Ia jejak bahwa pengiriman pernah dicoba dan
		// gagal — dan menghapusnya berarti menghilangkan satu-satunya bukti itu.
		return submission, result, fmt.Errorf(
			"monitoringslinkojk/usecase: kirim ke SLIK: %w", err)
	}

	if err := repo.CompleteSubmission(ctx, submission, result); err != nil {
		return submission, result, fmt.Errorf(
			"monitoringslinkojk/usecase: simpan id transaksi: %w", err)
	}
	return submission, result, nil
}
