package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/outstandingclaim"
)

// Repo membaca rincian klaim treaty dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Kedua tabel yang dibacanya milik sistem lama, dan
// selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB

	// logger mencatat berapa jalur dokumen yang TIDAK ditemukan.
	//
	// Ia di penyimpanan, bukan di usecase, karena hanya di sini bentuk dokumennya
	// diketahui. Boleh nil — lihat log.
	logger *slog.Logger
}

// NewRepo membentuk pembaca rincian di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// WithLogger menyalin Repo dengan pencatat tertentu.
//
// Ia terpisah dari NewRepo supaya pemanggil yang sudah ada tidak perlu berubah — dan supaya
// perintah `-periksa`, yang tidak punya logger, tetap dapat memakainya.
func (r *Repo) WithLogger(logger *slog.Logger) *Repo {
	clone := *r
	clone.logger = logger
	return &clone
}

// Find mengambil satu rincian klaim.
//
// Klaim yang tidak ada menghasilkan outstandingclaim.ErrNotFound, bukan Detail kosong: yang
// kedua terbaca di layar sebagai "klaim tanpa isi", padahal yang benar adalah "klaim tidak
// ditemukan di portal ini" — dan pada aplikasi yang melayani empat badan hukum, perbedaan itu
// yang memberi tahu pengguna bahwa ia mungkin salah portal.
func (r *Repo) Find(
	ctx context.Context,
	q outstandingclaim.Query,
) (outstandingclaim.Detail, error) {
	var (
		claimID, reference, statusWork, lastUpdateOperator sql.NullString
		rawDocument                                        sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query("find_claim"), q.ClaimID).Scan(
		&claimID, &reference, &statusWork, &lastUpdateOperator, &rawDocument,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return outstandingclaim.Detail{}, outstandingclaim.ErrNotFound
	case err != nil:
		return outstandingclaim.Detail{}, fmt.Errorf("membaca rincian klaim treaty: %w", err)
	}

	doc, err := parseDocument(rawDocument.String)
	if err != nil {
		// Dokumen yang ADA tetapi rusak TIDAK diturunkan menjadi rincian kosong. Rincian
		// kosong terbaca sebagai data yang belum diisi, dan kerusakan data yang tersaji
		// sebagai keadaan normal tidak pernah dilaporkan siapa pun.
		return outstandingclaim.Detail{}, fmt.Errorf("klaim %s: %w", q.ClaimID, err)
	}

	values, gridRows, stats := readDocument(doc)
	r.log(q.ClaimID, rawDocument.Valid, stats)

	return outstandingclaim.NewDetail(
		claimID.String,
		reference.String,
		statusWork.String,
		lastUpdateOperator.String,
		values,
		gridRows,
	), nil
}

// CheckTable memastikan kedua tabel yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah hak
// baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK atau POOLDATA.JSON_KLAIM: %w", err)
	}
	return nil
}

// log mencatat berapa jalur dokumen yang tidak ditemukan.
//
// # Kenapa ini dicatat, dan kenapa hanya saat ada yang hilang
//
// Bentuk dokumen klaim belum pernah diperiksa (`R-08`). Satu-satunya cara jalur yang salah
// terlihat adalah menghitungnya: layar yang seluruh isiannya kosong terbaca sama persis,
// entah karena klaimnya memang belum diisi atau karena seluruh jalurnya salah.
//
// Dicatat hanya saat ADA yang hilang supaya ia tidak menjadi satu baris log per permintaan
// yang normal — log yang selalu berbunyi adalah log yang berhenti dibaca.
func (r *Repo) log(claimID string, hasDocument bool, stats Stats) {
	if r.logger == nil {
		return
	}
	if stats.FieldsMissing == 0 && stats.GridsMissing == 0 {
		return
	}
	if !hasDocument {
		// Klaim tanpa baris di JSON_KLAIM: SELURUH jalur pasti hilang, dan itu keadaan
		// yang sudah dijelaskan gabungan LEFT JOIN. Mencatatnya sebagai jalur yang salah
		// akan menenggelamkan ketidakcocokan yang sebenarnya.
		return
	}

	r.logger.Warn("jalur dokumen klaim treaty tidak ditemukan",
		slog.String("modul", "outstandingclaim"),
		slog.String("no_klaim", claimID),
		slog.Int("isian_ketemu", stats.FieldsFound),
		slog.Int("isian_hilang", stats.FieldsMissing),
		slog.Int("grid_ketemu", stats.GridsFound),
		slog.Int("grid_hilang", stats.GridsMissing),
	)
}
