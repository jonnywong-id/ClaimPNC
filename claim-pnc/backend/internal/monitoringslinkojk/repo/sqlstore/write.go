package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/monitoringslinkojk"
)

// Aksi tulis modul Monitoring SLINK OJK.
//
// Kepemilikan tabelnya dan keputusan yang mendasarinya ada di kepala
// `monitoringslinkojk/write.go` dan `write.sql`; tidak diulang di sini.

// CountReport menghitung baris laporan yang sudah ada untuk satu klaim.
func (r *Repo) CountReport(ctx context.Context, claimID string) (int, error) {
	var total sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query("report_count"), claimID).Scan(&total); err != nil {
		return 0, fmt.Errorf("monitoringslinkojk/sqlstore: report_count: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

// InsertReport menyusun satu baris laporan.
//
// Kedua puluh delapan argumennya disusun di SATU tempat — reportArgs — supaya urutannya
// dapat dibaca berdampingan dengan berkas .sql. Satu argumen yang tertukar di sini tidak
// menghasilkan galat apa pun; ia hanya menulis nilai ke kolom yang salah pada laporan
// regulator. query_test.go memeriksa jumlahnya.
func (r *Repo) InsertReport(ctx context.Context, entry monitoringslinkojk.ReportEntry) error {
	if _, err := r.db.ExecContext(ctx, query("report_insert"), reportArgs(entry)...); err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: report_insert: %w", err)
	}
	return nil
}

// reportArgs menyusun kedua puluh delapan nilai INSERT, sesuai urutan kolom di .sql.
func reportArgs(entry monitoringslinkojk.ReportEntry) []any {
	return []any{
		entry.FacilityAccountNo,     // :1  NOREKFASILITAS
		entry.DebtorCIF,             // :2  NOCIFDEBITUR
		entry.FacilityTypeCode,      // :3  KODEJENISFASILITAS
		entry.FundSource,            // :4  SUMBERDANA
		entry.PolicyStart,           // :5  TANGGALMULAI
		entry.PolicyEnd,             // :6  TANGGALAKHIR
		entry.InterestRate,          // :7  SUKUBUNGA
		entry.CurrencyCode,          // :8  KODEVALUTA
		entry.Obligation,            // :9  NOMINAL
		entry.OriginalCurrencyValue, // :10 NILAIMATAUANGASAL
		entry.CollectibilityCode,    // :11 KODEKOLEKTABILITAS
		entry.DefaultDate,           // :12 TANGGALMACET
		entry.DefaultReasonCode,     // :13 KODESEBABMACET
		entry.Arrears,               // :14 TUNGGAKAN
		entry.ArrearsDays,           // :15 JUMLAHHARITUNGGAKAN
		entry.ConditionDate,         // :16 TANGGALKONDISI
		entry.ConditionCode,         // :17 KODEKONDISI
		entry.BranchCode,            // :18 KODEKANTORCABANG
		entry.DataOperation,         // :19 OPERASIDATA
		entry.Remark,                // :20 KETERANGAN
		entry.ReportMonth,           // :21 BULANLAPOR
		entry.ClaimID,               // :22 CLAIMID
		entry.ContractNo,            // :23 CONTRACTNO
		entry.Recovery,              // :24 RECOVERYCLAIM
		entry.IDCardNo,              // :25 NOKTP
		entry.CompanyNPWP,           // :26 NPWPPERUSAHAAN
		entry.PolicyNo,              // :27 NOPOLIS
		entry.ClientID,              // :28 CLIENTID
	}
}

// StreamSource membaca data klaim sumber yang siap disusun menjadi laporan.
//
// Kolom yang TIDAK tersedia dari kueri ini dibiarkan kosong pada hasilnya, bukan diisi
// tebakan — lihat catatan "DUA KOLOM YANG TIDAK DAPAT DIISI DARI SINI" di write.sql.
func (r *Repo) StreamSource(
	ctx context.Context,
	filter monitoringslinkojk.Filter,
	emit func(monitoringslinkojk.ReportEntry) error,
) error {
	// Penyaringnya sama dengan grid D01 — tanggal DAN "Business Name" — karena tombol
	// "Proses Data Klaim" hanya ada di segmen itu, dan yang tersusun harus persis yang
	// terlihat. Kotak pencarian sudah tidak ada; lihat Filter.
	plan := segmentPlan{takesBusinessScope: true}
	args := append([]any{r.branchCode}, plan.filterArgs(filter)...)

	rows, err := r.db.QueryContext(ctx, query("source_rows"), args...)
	if err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: source_rows: %w", err)
	}
	defer rows.Close()

	names, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]any, len(names))
	pointers := make([]any, len(names))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		for i := range values {
			values[i] = nil
		}
		if err := rows.Scan(pointers...); err != nil {
			return err
		}

		raw := make(map[string]string, len(names))
		for i, name := range names {
			raw[strings.ToUpper(name)] = text(values[i])
		}

		if err := emit(sourceEntry(raw)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// sourceEntry memetakan hasil kueri sumber menjadi satu baris laporan.
//
// Empat kolom sengaja TIDAK diisi di sini:
//
//   - ArrearsDays — di Pega ia `rownum`, nomor urut baris yang tidak ada hubungannya
//     dengan tunggakan. Tidak dibawa.
//   - IDCardNo, CompanyNPWP — tidak tersedia dari kueri ini maupun dari tabel objek.
//   - ReportMonth, DataOperation, Recovery, ClientID — ditetapkan lapisan aplikasi,
//     bukan dibaca dari basis data.
func sourceEntry(raw map[string]string) monitoringslinkojk.ReportEntry {
	return monitoringslinkojk.ReportEntry{
		ClaimID:           raw["CLAIM_ID"],
		ContractNo:        raw["CONTRACT_NO"],
		FacilityAccountNo: raw["NOREK_FASILITAS"],
		DebtorCIF:         raw["NOCIF_DEBITUR"],
		FacilityTypeCode:  raw["KODE_JENIS_FASILITAS"],
		FundSource:        raw["SUMBER_DANA"],
		PolicyStart:       raw["TANGGAL_MULAI"],
		PolicyEnd:         raw["TANGGAL_AKHIR"],
		InterestRate:      raw["SUKU_BUNGA"],
		CurrencyCode:      raw["KODE_VALUTA"],
		Obligation:        raw["JUMLAH_KEWAJIBAN"],

		// Di Pega, `NilaiMataUangAsal` dirangkai `CURRENCY || ' ' || SUMTSI`. Keduanya
		// sudah tersedia terpisah di sini, dan dirangkai ulang di lapisan aplikasi
		// supaya pemformatannya berada di satu tempat.
		OriginalCurrencyValue: joinCurrency(raw["KODE_VALUTA"], raw["JUMLAH_KEWAJIBAN"]),

		CollectibilityCode: raw["KODE_KOLEKTIBILITAS"],
		DefaultDate:        raw["TANGGAL_MACET"],
		DefaultReasonCode:  raw["KODE_SEBAB_MACET"],
		Arrears:            raw["TUNGGAKAN"],

		// Di Pega, `TanggalKondisi` dan `TanggalMacet` sama-sama diisi tanggal
		// registrasi klaim. Direplikasi.
		ConditionDate: raw["TANGGAL_MACET"],
		ConditionCode: raw["KODE_KONDISI"],

		BranchCode: raw["KODE_KANTOR_CABANG"],
		Remark:     raw["KETERANGAN"],
		PolicyNo:   raw["NO_POLIS"],
	}
}

// joinCurrency merangkai kode valuta dan nilainya, seperti `CURRENCY || ' ' || SUMTSI`.
//
// Bila salah satunya kosong, yang terisi dikembalikan sendirian — bukan dengan spasi
// menggantung di ujung, yang akan terbawa ke berkas laporan.
func joinCurrency(currency, amount string) string {
	switch {
	case currency == "":
		return amount
	case amount == "":
		return currency
	default:
		return currency + " " + amount
	}
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// NextSubmissionID mengembalikan nomor urut pengiriman berikutnya.
func (r *Repo) NextSubmissionID(ctx context.Context) (int64, error) {
	var next sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query("submission_next")).Scan(&next); err != nil {
		return 0, fmt.Errorf("monitoringslinkojk/sqlstore: submission_next: %w", err)
	}
	if !next.Valid {
		return 1, nil
	}
	return next.Int64, nil
}

// RecordSubmission mencatat satu pengiriman sebelum dikirim.
func (r *Repo) RecordSubmission(
	ctx context.Context,
	submission monitoringslinkojk.Submission,
) error {
	_, err := r.db.ExecContext(ctx, query("submission_insert"),
		submission.ClaimID, submission.ContractNo, submission.ID)
	if err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: submission_insert: %w", err)
	}
	return nil
}

// CompleteSubmission menyimpan jawaban sistem SLIK ke baris pengirimannya.
func (r *Repo) CompleteSubmission(
	ctx context.Context,
	submission monitoringslinkojk.Submission,
	result monitoringslinkojk.SubmissionResult,
) error {
	_, err := r.db.ExecContext(ctx, query("submission_done"),
		result.ClientID, result.TransactionID, submission.ID, submission.ClaimID)
	if err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: submission_done: %w", err)
	}
	return nil
}
