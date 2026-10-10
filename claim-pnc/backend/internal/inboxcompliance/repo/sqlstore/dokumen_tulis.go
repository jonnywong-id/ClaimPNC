package sqlstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// SaveDocument menulis satu baris lampiran, meniru `SET_ATTACHMENT_64BIT` tanpa
// memanggilnya (`D-02`).
//
// # Tiga penulisan dalam SATU transaksi
//
//  1. ambil nomor urut   ATTACHFILE_SEQ.NEXTVAL
//  2. baris pendamping   C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO)
//  3. baris lampiran     DATA_ATTACHFILE
//
// Ketiganya dibungkus satu transaksi karena langkah 2 dan 3 harus berdiri atau gugur
// bersama: `DATAID` dibentuk dari baris pendamping, sehingga lampiran tanpa pendamping
// adalah baris yang asalnya tidak dapat ditelusuri.
//
// Nomor urut diambil di dalam transaksi yang sama, tetapi sequence TIDAK ikut rollback —
// itu sifat sequence Oracle, bukan kelalaian. Akibatnya kegagalan menyisakan lubang pada
// deretan nomor, dan itu tidak merugikan siapa pun.
func (r *Repo) SaveDocument(
	ctx context.Context, document inboxcompliance.Document,
) (inboxcompliance.Document, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("memulai transaksi penyimpanan lampiran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var runno int64
	if err := tx.QueryRowContext(
		ctx, query("next_attachment_runno"),
	).Scan(&runno); err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("mengambil nomor urut lampiran: %w", err)
	}

	// `new_uuid` tidak ada di export; ia jelas pembangkit UUID, dan nilainya tidak
	// pernah dibaca manusia — ia hanya kunci baris pendamping.
	kunci, err := kunciAcak()
	if err != nil {
		return inboxcompliance.Document{}, err
	}

	// `to_char(sysdate,'yy')` pada procedure. Tahunnya diambil dari jam aplikasi, bukan
	// dari basis data, supaya ia mengikuti Clock seperti seluruh waktu lain (`F-5`).
	tahun := time.Now().Format("06")

	if _, err := tx.ExecContext(
		ctx, query("insert_attachment_counter"), kunci, tahun, runno,
	); err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("menulis baris pendamping lampiran: %w", err)
	}

	// `year || lpad(runno,10,'0')` — `SET_ATTACHMENT_64BIT.prc:25-27`.
	dataID := fmt.Sprintf("%s%010d", tahun, runno)

	if _, err := tx.ExecContext(
		ctx, query("insert_attachment"),
		dataID,
		nullableText(document.UploadedBy),
		nullableText(document.Name),
		nullableText(document.Note),
		nullableText(document.MimeType),
		nullableText(document.StorageID),
		nullableText(document.Category),
		nullableText(document.SubCategory),
		document.ClaimReference,
	); err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("menulis baris lampiran: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("menyelesaikan penyimpanan lampiran: %w", err)
	}

	document.ID = dataID
	return document, nil
}

// DeleteDocument menghapus satu baris lampiran MILIK satu klaim.
//
// Penyaring `IDPEGA` adalah pengetatan yang disengaja — procedure Pega menghapus hanya
// dengan `DATAID`. Lihat kueri `delete_attachment`.
func (r *Repo) DeleteDocument(
	ctx context.Context, reference, documentID string,
) (bool, error) {
	hasil, err := r.db.ExecContext(
		ctx, query("delete_attachment"), documentID, reference)
	if err != nil {
		return false, fmt.Errorf("menghapus baris lampiran: %w", err)
	}

	terhapus, err := hasil.RowsAffected()
	if err != nil {
		// Driver yang tidak melaporkan jumlah baris membuat kita tidak dapat
		// membedakan "terhapus" dari "tidak ada yang cocok". Dilaporkan sebagai galat,
		// bukan ditebak sebagai berhasil — karena pemanggil memakai nilai itu untuk
		// memutuskan apakah berkasnya ikut dihapus dari penyimpanan.
		return false, fmt.Errorf("membaca jumlah baris lampiran terhapus: %w", err)
	}
	return terhapus > 0, nil
}

// kunciAcak membangkitkan kunci 32 heksadesimal untuk baris pendamping.
//
// Bukan UUID berformat, karena bentuknya tidak pernah dibaca siapa pun — yang dituntut
// hanyalah keunikan. `crypto/rand`, bukan `math/rand`: kunci yang dapat ditebak membuat
// baris pendamping dapat ditabrak dengan sengaja.
func kunciAcak() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("membangkitkan kunci baris pendamping lampiran: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
