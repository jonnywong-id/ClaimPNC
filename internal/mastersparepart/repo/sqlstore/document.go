package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/mastersparepart"
)

// dataIDSequenceWidth adalah lebar bagian nomor pada DATAID.
//
// `Database/SET_ATTACHMENT_64BIT.prc` merangkainya `tahun || lpad(runno, 10, '0')`.
//
// Angkanya kebetulan sama dengan sequenceWidth milik ID sparepart, dan kebetulan itu TIDAK
// dipakai: keduanya tetap dua tetapan terpisah karena mereka deret yang berbeda milik tabel
// yang berbeda. Menyatukannya akan membuat perubahan pada salah satunya diam-diam mengubah
// yang lain.
const dataIDSequenceWidth = 10

// composeDataID merangkai DATAID dari tahun dan nomor urut.
//
// Ia berada di adapter, bukan di domain, dan itu berbeda dari ComposeID milik sparepart.
// Alasannya: bentuk kunci SPAREPART adalah aturan domain — ia yang menentukan bagaimana
// sebuah sparepart dikenali, dan kedua adapter harus sepakat. DATAID sebaliknya milik tabel
// lampiran yang dipakai bersama banyak modul; bentuknya ditentukan tabel itu, bukan oleh
// Master Sparepart.
//
// Nomor yang lebih panjang dari lebarnya TIDAK dipotong, mengikuti perilaku LPAD Oracle
// yang justru memotong — perbedaan yang disengaja, karena memotong kunci berarti
// menerbitkan kunci yang bertabrakan dengan kunci lain. Bila deretnya benar-benar melewati
// sepuluh digit, kunci yang lebih panjang jauh lebih baik daripada kunci ganda.
func composeDataID(year int, sequence int64) string {
	number := strconv.FormatInt(sequence, 10)
	if len(number) < dataIDSequenceWidth {
		number = strings.Repeat("0", dataIDSequenceWidth-len(number)) + number
	}
	return strconv.Itoa(year) + number
}

// SaveDocument mencatat metadata dokumen dan menautkannya ke sparepart-nya, dalam SATU
// transaksi.
//
// Keduanya tidak boleh terpisah. Baris lampiran tanpa tautan adalah baris yatim yang tidak
// terlihat dari layar mana pun; tautan tanpa barisnya adalah sparepart yang menunjuk ke
// dokumen yang tidak ada. Sistem lama menempuh keduanya sebagai langkah terpisah di dalam
// activity — `D-68` memindahkan kepemilikan transaksi ke Go justru supaya hal seperti ini
// dapat dibungkus sekali jalan.
//
// Nomor urutnya diambil DI DALAM transaksi. Sequence Oracle tidak ikut mundur saat
// rollback, sehingga kegagalan meninggalkan lubang pada deret DATAID — dan itu memang
// benar: deret yang berlubang jauh lebih aman daripada deret yang berulang.
func (r *Repo) SaveDocument(
	ctx context.Context, doc mastersparepart.SparepartDocument,
) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("mastersparepart/sqlstore: memulai transaksi dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sequence int64
	row := tx.QueryRowContext(ctx, getQuery("sparepart_document_next_sequence"))
	if err := row.Scan(&sequence); err != nil {
		return "", fmt.Errorf("mastersparepart/sqlstore: mengambil nomor DATAID: %w", err)
	}

	at := time.Now()
	if doc.UploadedAt != nil {
		at = *doc.UploadedAt
	}
	dataID := composeDataID(at.Year(), sequence)

	if _, err := tx.ExecContext(ctx, getQuery("sparepart_document_insert"),
		dataID, doc.UploadedBy, doc.Name, doc.Note, doc.MimeType, doc.ImageID, doc.SparepartID,
	); err != nil {
		return "", fmt.Errorf("mastersparepart/sqlstore: menyisipkan dokumen %q: %w", dataID, err)
	}

	result, err := tx.ExecContext(ctx, getQuery("sparepart_document_link"), dataID, doc.SparepartID)
	if err != nil {
		return "", fmt.Errorf(
			"mastersparepart/sqlstore: menautkan dokumen ke %q: %w", doc.SparepartID, err)
	}
	// Sparepart yang tidak ditemukan harus menggagalkan transaksi, bukan lolos diam-diam.
	// Tanpa pemeriksaan ini, unggahan pada ID yang salah ketik akan melaporkan berhasil
	// sambil meninggalkan satu baris lampiran yatim.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return "", fmt.Errorf("%w: %q", mastersparepart.ErrNotFound, doc.SparepartID)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("mastersparepart/sqlstore: menutup transaksi dokumen: %w", err)
	}
	return dataID, nil
}

// DocumentOf mengembalikan dokumen sebuah sparepart.
func (r *Repo) DocumentOf(
	ctx context.Context, sparepartID string,
) (mastersparepart.SparepartDocument, error) {
	row := r.db.QueryRowContext(ctx, getQuery("sparepart_document_get"), sparepartID)
	doc, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastersparepart.SparepartDocument{}, fmt.Errorf(
			"%w: %q", mastersparepart.ErrDocumentMissing, sparepartID)
	}
	if err != nil {
		return mastersparepart.SparepartDocument{}, fmt.Errorf(
			"mastersparepart/sqlstore: membaca dokumen %q: %w", sparepartID, err)
	}
	return doc, nil
}

// scanDocument membaca satu baris dokumen.
//
// Urutan kolomnya mengikuti sparepart_document_get. Seluruh kolomnya dibaca sebagai
// Null* karena tak satu pun dijamin terisi: `SET_ATTACHMENT_64BIT.prc` mengisi sebagian
// saja, dan baris lama yang ditulis Pega memuat kombinasi yang berbeda-beda.
func scanDocument(p scanner) (mastersparepart.SparepartDocument, error) {
	var (
		doc         mastersparepart.SparepartDocument
		mime        sql.NullString
		note        sql.NullString
		by          sql.NullString
		at          sql.NullTime
		sparepartID sql.NullString
		imageID     sql.NullString
		dataID      sql.NullString
		nameCell    sql.NullString
	)
	if err := p.Scan(&dataID, &imageID, &nameCell, &mime, &note, &by, &at, &sparepartID); err != nil {
		return mastersparepart.SparepartDocument{}, err
	}
	doc.DataID = strings.TrimSpace(dataID.String)
	doc.ImageID = strings.TrimSpace(imageID.String)
	doc.Name = strings.TrimSpace(nameCell.String)
	doc.MimeType = strings.TrimSpace(mime.String)
	doc.Note = strings.TrimSpace(note.String)
	doc.UploadedBy = strings.TrimSpace(by.String)
	doc.SparepartID = strings.TrimSpace(sparepartID.String)
	if at.Valid {
		moment := at.Time
		doc.UploadedAt = &moment
	}
	return doc, nil
}

var _ mastersparepart.DocumentRepo = (*Repo)(nil)
