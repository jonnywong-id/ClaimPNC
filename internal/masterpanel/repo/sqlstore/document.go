package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/masterpanel"
)

// dataIDSequenceWidth adalah lebar bagian nomor pada DATAID.
//
// `Database/SET_ATTACHMENT_64BIT.prc` merangkainya `tahun || lpad(runno, 10, '0')`.
// Sepuluh, bukan enam seperti ID panel — keduanya deret yang berbeda dengan bentuk yang
// berbeda, dan menyamakannya akan menghasilkan kunci yang tidak dikenali baris lama.
const dataIDSequenceWidth = 10

// composeDataID merangkai DATAID dari tahun dan nomor urut.
//
// Ia berada di adapter, bukan di domain, dan itu berbeda dari ComposeID milik panel.
// Alasannya: bentuk kunci PANEL adalah aturan domain — ia yang menentukan bagaimana sebuah
// panel dikenali, dan kedua adapter harus sepakat. DATAID sebaliknya milik tabel lampiran
// yang dipakai bersama banyak modul; bentuknya ditentukan tabel itu, bukan oleh Master
// Panel.
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

// SaveDocument mencatat metadata dokumen dan menautkannya ke panelnya, dalam SATU transaksi.
//
// Keduanya tidak boleh terpisah. Baris lampiran tanpa tautan adalah baris yatim yang tidak
// terlihat dari layar mana pun; tautan tanpa barisnya adalah panel yang menunjuk ke
// dokumen yang tidak ada. Sistem lama menempuh keduanya sebagai langkah terpisah di dalam
// activity — `D-68` memindahkan kepemilikan transaksi ke Go justru supaya hal seperti ini
// dapat dibungkus sekali jalan.
//
// Nomor urutnya diambil DI DALAM transaksi. Sequence Oracle tidak ikut mundur saat rollback,
// sehingga kegagalan meninggalkan lubang pada deret DATAID — dan itu memang benar: deret
// yang berlubang jauh lebih aman daripada deret yang berulang.
func (r *Repo) SaveDocument(
	ctx context.Context, doc masterpanel.PanelDocument,
) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: memulai transaksi dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sequence int64
	row := tx.QueryRowContext(ctx, getQuery("panel_document_next_sequence"))
	if err := row.Scan(&sequence); err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: mengambil nomor DATAID: %w", err)
	}

	at := time.Now()
	if doc.UploadedAt != nil {
		at = *doc.UploadedAt
	}
	dataID := composeDataID(at.Year(), sequence)

	if _, err := tx.ExecContext(ctx, getQuery("panel_document_insert"),
		dataID, doc.UploadedBy, doc.Name, doc.Note, doc.MimeType, doc.ImageID, doc.PanelID,
	); err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: menyisipkan dokumen %q: %w", dataID, err)
	}

	result, err := tx.ExecContext(ctx, getQuery("panel_document_link"), dataID, doc.PanelID)
	if err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: menautkan dokumen ke %q: %w", doc.PanelID, err)
	}
	// Panel yang tidak ditemukan harus menggagalkan transaksi, bukan lolos diam-diam.
	// Tanpa pemeriksaan ini, unggahan pada ID yang salah ketik akan melaporkan berhasil
	// sambil meninggalkan satu baris lampiran yatim.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return "", fmt.Errorf("%w: %q", masterpanel.ErrNotFound, doc.PanelID)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: menutup transaksi dokumen: %w", err)
	}
	return dataID, nil
}

// DocumentOf mengembalikan dokumen sebuah panel.
func (r *Repo) DocumentOf(
	ctx context.Context, panelID string,
) (masterpanel.PanelDocument, error) {
	row := r.db.QueryRowContext(ctx, getQuery("panel_document_get"), panelID)
	doc, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterpanel.PanelDocument{}, fmt.Errorf("%w: %q", masterpanel.ErrDocumentMissing, panelID)
	}
	if err != nil {
		return masterpanel.PanelDocument{}, fmt.Errorf(
			"masterpanel/sqlstore: membaca dokumen %q: %w", panelID, err)
	}
	return doc, nil
}

// scanDocument membaca satu baris dokumen.
//
// Urutan kolomnya mengikuti panel_document_get DAN panel_check_document_table — keduanya
// sengaja ditulis dengan daftar kolom yang sama persis, supaya `-periksa` benar-benar
// menguji kolom yang dibaca jalur produksi, bukan sekadar kolom yang kebetulan ada.
func scanDocument(p scanner) (masterpanel.PanelDocument, error) {
	var (
		doc      masterpanel.PanelDocument
		mime     sql.NullString
		note     sql.NullString
		by       sql.NullString
		at       sql.NullTime
		panelID  sql.NullString
		imageID  sql.NullString
		dataID   sql.NullString
		nameCell sql.NullString
	)
	if err := p.Scan(&dataID, &imageID, &nameCell, &mime, &note, &by, &at, &panelID); err != nil {
		return masterpanel.PanelDocument{}, err
	}
	doc.DataID = strings.TrimSpace(dataID.String)
	doc.ImageID = strings.TrimSpace(imageID.String)
	doc.Name = strings.TrimSpace(nameCell.String)
	doc.MimeType = strings.TrimSpace(mime.String)
	doc.Note = strings.TrimSpace(note.String)
	doc.UploadedBy = strings.TrimSpace(by.String)
	doc.PanelID = strings.TrimSpace(panelID.String)
	if at.Valid {
		moment := at.Time
		doc.UploadedAt = &moment
	}
	return doc, nil
}

// CheckDocumentTable membuktikan tabel metadata dokumen terjangkau beserta kolomnya.
func (r *Repo) CheckDocumentTable(ctx context.Context) error {
	return r.checkReadable(ctx, "panel_check_document_table", "POOLDATA.DATA_ATTACHFILE")
}

// CountDocumentLinked menghitung panel yang dokumennya ketemu.
func (r *Repo) CountDocumentLinked(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_document_linked")
}

// CountDocumentDangling menghitung panel ber-DOKUMENID yang barisnya TIDAK ketemu.
//
// Angka inilah yang dapat menggugurkan asumsi "DOKUMENID menyimpan DATAID". Lihat
// keterangan kueri yang sama namanya di berkas .sql.
func (r *Repo) CountDocumentDangling(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_document_dangling")
}

// LinkDocument menautkan dokumen yang sudah tercatat ke sebuah panel.
//
// Satu pernyataan, tanpa transaksi: barisnya sudah ada, dan yang tersisa hanya menunjuknya.
func (r *Repo) LinkDocument(ctx context.Context, panelID, dataID string) error {
	result, err := r.db.ExecContext(ctx, getQuery("panel_document_link"), dataID, panelID)
	if err != nil {
		return fmt.Errorf("masterpanel/sqlstore: menautkan dokumen ke %q: %w", panelID, err)
	}
	// Panel yang tidak ditemukan harus menggagalkan, bukan lolos diam-diam — baris lampiran
	// yang tidak ditunjuk siapa pun tidak terlihat dari layar mana pun.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return fmt.Errorf("%w: %q", masterpanel.ErrNotFound, panelID)
	}
	return nil
}
