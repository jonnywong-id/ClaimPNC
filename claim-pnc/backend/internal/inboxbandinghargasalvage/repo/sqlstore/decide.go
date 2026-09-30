package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Writer menulis keputusan banding harga ke SATU basis data entitas.
//
// Ia tipe TERSENDIRI, bukan method pada Repo, dan pemisahan itu mengikuti seam-nya: Repo
// dipakai seluruh permintaan baca, Writer hanya satu rute. Keduanya berbagi koneksi yang
// sama, tetapi tidak berbagi permukaan — sehingga pengisi Repo untuk uji baca tidak perlu
// memikul operasi tulis yang tidak dipakainya.
type Writer struct {
	db *sql.DB
}

// NewWriter membentuk penulis keputusan di atas satu koneksi.
func NewWriter(db *sql.DB) *Writer {
	return &Writer{db: db}
}

// Decide mencatat keputusan komite beserta ketiga langkah tambahannya.
//
// # Satu transaksi, dan kenapa itu berbeda dari Pega
//
// Di Pega keempat pernyataannya berdiri sendiri dan menyimpan seketika, sehingga kegagalan
// di tengah meninggalkan putusan tanpa harga — atau harga tanpa putusan. Ketiganya menyentuh
// nilai uang, dan keadaan setengah jalan pada nilai uang tidak dapat dibedakan dari nilai
// yang memang begitu.
//
// Kepemilikan transaksi ada di Go (`D-68`). Selisihnya disengaja dan dinyatakan lewat
// PlannedDifferences: saat gagal, sistem lama meninggalkan sebagian perubahan dan sistem ini
// tidak meninggalkan apa pun.
//
// # Urutan langkahnya mengikuti activity aslinya
//
// Putusan dicatat LEBIH DULU. Bila tidak ada baris yang cocok — sudah diputus, atau bukan
// milik komite ini — ketiga langkah berikutnya tidak dijalankan sama sekali, dan transaksinya
// dibatalkan. Menjalankan penerapan harga atas putusan yang tidak tercatat berarti mengubah
// nilai uang tanpa satu pun jejak yang menyatakan siapa memutuskannya.
func (w *Writer) Decide(
	ctx context.Context,
	command inboxbandinghargasalvage.DecisionCommand,
) (inboxbandinghargasalvage.DecisionResult, error) {
	var empty inboxbandinghargasalvage.DecisionResult

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return empty, fmt.Errorf("memulai transaksi keputusan banding: %w", err)
	}
	// Rollback yang dipanggil setelah Commit tidak berakibat apa pun; ia ada untuk jalur
	// gagal, termasuk jalur yang keluar lewat panic.
	defer func() { _ = tx.Rollback() }()

	recorded, err := w.record(ctx, tx, command)
	if err != nil {
		return empty, err
	}
	if !recorded {
		// Bukan galat teknis: barangnya sudah diputus, atau bukan milik komite ini.
		// Transaksinya dibatalkan oleh defer di atas.
		return empty, inboxbandinghargasalvage.ErrAlreadyDecided
	}

	result := inboxbandinghargasalvage.DecisionResult{Recorded: true}

	if command.CascadeTo != "" {
		if _, err := tx.ExecContext(ctx, query("decide_cascade"),
			command.Status,
			strings.ToUpper(strings.TrimSpace(command.CascadeTo)),
			command.DetailObject,
		); err != nil {
			return empty, fmt.Errorf("menutup baris komite berikutnya: %w", err)
		}
	}

	if command.MarkDocument {
		if _, err := tx.ExecContext(ctx, query("decide_mark_document"),
			command.DetailObject,
		); err != nil {
			return empty, fmt.Errorf("menandai dokumen banding: %w", err)
		}
		// Ditandai TERLAKSANA meski kemungkinan besar nol baris — penyaring kueri aslinya
		// tampak keliru dan ditiru apa adanya. Yang dinyatakan di sini adalah "langkahnya
		// dijalankan", bukan "ada baris yang berubah"; lihat catatan pada kuerinya.
		result.DocumentMarked = true
	}

	if command.ApplyPrice {
		if _, err := tx.ExecContext(ctx, query("decide_apply_price"),
			command.RequestPrice, command.SalvageID, command.DetailObject,
		); err != nil {
			return empty, fmt.Errorf("menerapkan harga banding: %w", err)
		}
		result.PriceApplied = true
	}

	if err := tx.Commit(); err != nil {
		return empty, fmt.Errorf("menyimpan keputusan banding: %w", err)
	}

	return result, nil
}

// record mencatat putusan dan menyatakan apakah ada baris yang benar-benar berubah.
//
// Jumlah baris dibaca, bukan diabaikan: `WHERE TGLAPPROVE IS NULL` membuat penekanan tombol
// kedua tidak mengubah apa pun, dan tanpa pemeriksaan ini pengguna akan melihat pesan
// "berhasil" yang kedua kalinya tidak berarti apa-apa.
func (w *Writer) record(
	ctx context.Context,
	tx *sql.Tx,
	command inboxbandinghargasalvage.DecisionCommand,
) (bool, error) {
	outcome, err := tx.ExecContext(ctx, query("decide_record"),
		command.Status,
		command.Note,
		strings.ToUpper(strings.TrimSpace(command.Reviewer.Name)),
		command.DetailObject,
	)
	if err != nil {
		return false, fmt.Errorf("mencatat keputusan banding: %w", err)
	}

	affected, err := outcome.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("membaca jumlah baris keputusan banding: %w", err)
	}

	return affected > 0, nil
}
