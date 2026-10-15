package sqlstore

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

var docCols = []string{
	"DATA_ID", "IMAGE_ID", "ATTACH_NAME", "ATTACH_MIME_TYPE",
	"ATTACH_NOTE", "INPUT_OPERATOR", "INPUT_DATE", "ID_PEGA",
}

func sampleDocument() masterpanel.PanelDocument {
	at := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	return masterpanel.PanelDocument{
		ImageID: "IMG000000123", Name: "panel.pdf", MimeType: "application/pdf",
		Note: "Foto panel", UploadedBy: "PETUGAS", UploadedAt: &at, PanelID: "01000001",
	}
}

// DATAID dirangkai dari TAHUN unggahan ditambah nomor urut berlebar sepuluh — bentuk yang
// sama dengan `Database/SET_ATTACHMENT_64BIT.prc`. Bentuk yang berbeda berarti kunci yang
// tidak dikenali baris lama maupun kueri modul lain yang membacanya.
func TestDataIDBerbentukTahunDanSepuluhDigit(t *testing.T) {
	repo, mock := newMock(t)
	doc := sampleDocument()

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_document_next_sequence")).
		WillReturnRows(sqlmock.NewRows(singleCol).AddRow(int64(42)))
	mock.ExpectExec(q("panel_document_insert")).
		WithArgs("20260000000042", "PETUGAS", "panel.pdf", "Foto panel",
			"application/pdf", "IMG000000123", "01000001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_document_link")).
		WithArgs("20260000000042", "01000001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	dataID, err := repo.SaveDocument(context.Background(), doc)
	require.NoError(t, err)
	require.Equal(t, "20260000000042", dataID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penyisipan dan penautan berada di SATU transaksi. Terpisah, kegagalan di antara keduanya
// meninggalkan baris lampiran yatim yang tidak terlihat dari layar mana pun — dan tidak
// ada yang akan membersihkannya.
func TestPenautanGagalMembatalkanPenyisipan(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_document_next_sequence")).
		WillReturnRows(sqlmock.NewRows(singleCol).AddRow(int64(7)))
	mock.ExpectExec(q("panel_document_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_document_link")).WillReturnError(errOracle)
	mock.ExpectRollback()

	_, err := repo.SaveDocument(context.Background(), sampleDocument())
	require.ErrorIs(t, err, errOracle)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Panel yang tidak ditemukan harus MENGGAGALKAN transaksi, bukan lolos diam-diam.
// Tanpa pemeriksaan baris terpengaruh, unggahan pada ID yang salah ketik akan melaporkan
// berhasil sambil meninggalkan satu baris lampiran yang tidak menempel ke mana pun.
func TestPanelTidakDitemukanMembatalkanSeluruhnya(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_document_next_sequence")).
		WillReturnRows(sqlmock.NewRows(singleCol).AddRow(int64(7)))
	mock.ExpectExec(q("panel_document_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_document_link")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	_, err := repo.SaveDocument(context.Background(), sampleDocument())
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentOfMemangkasSpasiKolom(t *testing.T) {
	repo, mock := newMock(t)
	at := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q("panel_document_get")).WithArgs("01000001").
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			"20260000000042 ", " IMG000000123", "panel.pdf ", " application/pdf",
			" Foto panel ", "PETUGAS ", at, " 01000001"))

	doc, err := repo.DocumentOf(context.Background(), "01000001")
	require.NoError(t, err)
	require.Equal(t, "20260000000042", doc.DataID)
	require.Equal(t, "IMG000000123", doc.ImageID)
	require.Equal(t, "application/pdf", doc.MimeType)
	require.Equal(t, "Foto panel", doc.Note)
	require.Equal(t, "01000001", doc.PanelID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentOfTanpaBarisMengembalikanErrDocumentMissing(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(q("panel_document_get")).WithArgs("01000001").
		WillReturnRows(sqlmock.NewRows(docCols))

	_, err := repo.DocumentOf(context.Background(), "01000001")
	require.ErrorIs(t, err, masterpanel.ErrDocumentMissing)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kolom yang DIBACA jalur produksi harus sama persis dengan kolom yang DIUJI `-periksa`.
// Bila keduanya berbeda, pemeriksaan itu akan melaporkan hijau untuk kolom yang tidak
// pernah dipakai — dan kolom yang dipakai baru ketahuan hilang saat pengguna mengunggah.
//
// scanDocument juga membaca berdasarkan POSISI, sehingga satu kolom yang bergeser akan
// menaruh catatan ke kolom nama berkas tanpa satu pun galat.
func TestKueriDokumenDanPemeriksaBerbagiUrutanKolom(t *testing.T) {
	reference := selectedColumns(t, getQuery("panel_document_get"))
	require.Len(t, reference, len(docCols), "kedelapan kolom dokumen harus dibaca")
	require.Equal(t, reference, selectedColumns(t, getQuery("panel_check_document_table")),
		"panel_check_document_table membaca kolom pada urutan yang berbeda")
}
