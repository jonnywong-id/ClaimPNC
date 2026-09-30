package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inputacceptation"
)

// Repo membaca akseptasi klaim treaty non-proporsional dari SATU basis data entitas.
//
// Ia MEMBACA saja. Jalur tulisnya ada — lihat Save — tetapi menolak sampai kepemilikan
// tabelnya berpindah dari Pega (`P-1`, `D-63`).
type Repo struct {
	db *sql.DB

	// logger mencatat berapa isian dan grid yang jalurnya tidak ditemukan di dokumen.
	//
	// Boleh nil. Bila nil, hitungannya tidak ditulis dan tidak ada yang gagal karenanya —
	// tetapi jalur dokumen yang salah kehilangan satu-satunya tanda yang dimilikinya.
	logger *slog.Logger
}

// NewRepo membentuk pembaca akseptasi di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// WithLogger menyerahkan pencatat hitungan jalur dokumen.
//
// Ia terpisah dari NewRepo supaya pemanggil yang tidak peduli hitungannya — uji, misalnya —
// tidak perlu menyediakan apa pun.
func (r *Repo) WithLogger(logger *slog.Logger) *Repo {
	r.logger = logger
	return r
}

// Find mengambil satu rincian akseptasi.
//
// Klaim yang tidak ada menghasilkan inputacceptation.ErrNotFound. Klaim yang ADA tetapi
// dokumennya kosong TIDAK: ia klaim yang dapat dibuka dengan isian kosong, dan membedakan
// keduanya adalah inti gabungan LEFT JOIN di kuerinya.
func (r *Repo) Find(
	ctx context.Context, q inputacceptation.Query,
) (inputacceptation.Detail, error) {
	var (
		claimID, reference       sql.NullString
		statusWork, lastOperator sql.NullString
		rawDocument              sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query("find_claim"), q.ClaimID).Scan(
		&claimID, &reference, &statusWork, &lastOperator, &rawDocument,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return inputacceptation.Detail{}, inputacceptation.ErrNotFound
	}
	if err != nil {
		return inputacceptation.Detail{}, fmt.Errorf("membaca klaim %s: %w", q.ClaimID, err)
	}

	doc, err := parseDocument(rawDocument.String)
	if err != nil {
		return inputacceptation.Detail{}, fmt.Errorf("klaim %s: %w", q.ClaimID, err)
	}

	values, gridRows, stats := readDocument(doc)
	r.logStats(ctx, q.ClaimID, stats)

	detail, err := inputacceptation.NewDetail(
		claimID.String, reference.String, statusWork.String, lastOperator.String,
		values, gridRows,
	)
	if err != nil {
		// Galat di sini BUKAN galat data melainkan selisih antara pembaca dokumen dan
		// section.go — keduanya di dalam kode. Ia dibungkus apa adanya supaya kuncinya
		// terbaca di log, dan supaya uji menangkapnya.
		return inputacceptation.Detail{}, fmt.Errorf("klaim %s: %w", q.ClaimID, err)
	}
	return detail, nil
}

// Save menolak menulis selama tabelnya masih dimiliki Pega.
//
// # Kenapa ia ADA tetapi menolak, bukan tidak ada sama sekali
//
// Karena Work Owner memutuskan layar ini dibangun PENUH termasuk Submit (2026-09-30), dan
// jalur tulisnya memang dibangun: muatannya divalidasi penuh terhadap section.go — isian
// read-only ditolak, kolom read-only ditolak, kunci yang tidak dikenal ditolak — lalu
// permintaannya sampai ke sini.
//
// Yang TIDAK berubah oleh keputusan itu adalah `P-1`: selama masa paralel setiap tabel hanya
// boleh ditulis satu sistem, dan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` beserta
// `POOLDATA.JSON_KLAIM` masih ditulis Pega. Dua sistem yang menulisi nilai akseptasi yang sama
// tidak menghasilkan galat — keduanya hanya saling menimpa tanpa jejak, dan yang dipertaruhkan
// adalah Nomor Akseptasi beserta nilai uang yang menyertainya.
//
// Perpindahan kepemilikannya menempuh `D-63`: permintaan tertulis dari tim pengembang,
// persetujuan Work Owner, pelaksanaan DBA, lalu diuji dengan menjalankan Pega dan Go
// bersamaan. Begitu itu selesai, yang perlu diubah di sini hanyalah badan fungsi ini —
// kontraknya, validasinya, dan seluruh lapisan di atasnya sudah siap.
//
// Penolakannya memakai galat tersendiri, bukan 500, supaya pengguna membaca sebab dan jalan
// keluarnya alih-alih "terjadi kesalahan pada sistem".
func (r *Repo) Save(_ context.Context, cmd inputacceptation.SaveCommand) error {
	if r.logger != nil {
		r.logger.Warn("submit akseptasi ditolak karena kepemilikan tabel",
			slog.String("modul", "input-acceptation"),
			slog.String("no_klaim", cmd.Query.ClaimID),
			slog.String("pemanggil", cmd.Query.Caller.Login),
			slog.Int("isian_diubah", len(cmd.Values)),
			slog.Int("tabel_diubah", len(cmd.Grids)),
		)
	}
	return inputacceptation.ErrWriteNotOwned
}

// CheckTables memastikan kedua tabel yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah hak
// baca dan keberadaan tabelnya.
func (r *Repo) CheckTables(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK atau POOLDATA.JSON_KLAIM: %w", err)
	}
	return nil
}

// logStats mencatat berapa isian dan grid yang jalurnya tidak ditemukan.
//
// Ia yang membuat jalur dokumen yang salah punya tanda. Tanpa hitungan ini, layar berisi ~50
// isian kosong terbaca sama persis — entah karena klaimnya memang belum diisi, atau karena
// seluruh jalurnya salah.
func (r *Repo) logStats(ctx context.Context, claimID string, stats Stats) {
	if r.logger == nil {
		return
	}
	r.logger.InfoContext(ctx, "dokumen akseptasi klaim treaty non-prop dibaca",
		slog.String("modul", "input-acceptation"),
		slog.String("no_klaim", claimID),
		slog.Int("isian_ditemukan", stats.FieldsFound),
		slog.Int("isian_tidak_ada", stats.FieldsMissing),
		slog.Int("tabel_ditemukan", stats.GridsFound),
		slog.Int("tabel_tidak_ada", stats.GridsMissing),
	)
}
