package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// DocumentReader memenuhi seam inboxbandinghargasalvage.DocumentReader.
//
// Ia tipe tersendiri, bukan method pada Repo, mengikuti pemisahan seam-nya di domain.
type DocumentReader struct {
	db *sql.DB
}

// NewDocumentReader membentuk pembaca dokumen banding di atas satu koneksi.
func NewDocumentReader(db *sql.DB) *DocumentReader {
	return &DocumentReader{db: db}
}

// ListDocuments menyerahkan dokumen banding satu barang.
func (r *DocumentReader) ListDocuments(
	ctx context.Context,
	q inboxbandinghargasalvage.DocumentQuery,
) ([]inboxbandinghargasalvage.DocumentRow, error) {
	rows, err := r.db.QueryContext(ctx, query("list_documents"),
		q.DetailObject,
		q.SalvageID,
		strings.ToUpper(strings.TrimSpace(q.Reviewer.Name)),
	)
	if err != nil {
		return nil, fmt.Errorf("membaca dokumen banding: %w", err)
	}
	defer rows.Close()

	// Senarai KOSONG, bukan nil: pemanggil menyerahkannya apa adanya ke JSON, dan `nil`
	// tergambar sebagai `null` sementara senarai kosong tergambar sebagai `[]`. Layar
	// membedakan keduanya — yang kedua berarti "tidak ada dokumen", yang pertama tidak
	// berarti apa-apa.
	documents := []inboxbandinghargasalvage.DocumentRow{}

	for rows.Next() {
		var (
			document inboxbandinghargasalvage.DocumentRow
			name     sql.NullString
			uploaded sql.NullTime
		)

		if err := rows.Scan(&document.ID, &name, &uploaded); err != nil {
			return nil, fmt.Errorf("memindai dokumen banding: %w", err)
		}

		document.Name = strings.TrimSpace(name.String)
		document.UploadedAt = timeOrNil(uploaded)
		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca dokumen banding: %w", err)
	}

	return documents, nil
}

// DocumentContent menyerahkan isi satu dokumen.
//
// Nol baris dijawab ErrDocumentNotFound, dan itu menutup DUA keadaan sekaligus: dokumennya
// memang tidak ada, atau ia bukan milik banding yang ditangani komite pemanggil. Keduanya
// sengaja tidak dibedakan — lihat ErrDocumentNotFound.
func (r *DocumentReader) DocumentContent(
	ctx context.Context,
	documentID string,
	q inboxbandinghargasalvage.DocumentQuery,
) (inboxbandinghargasalvage.DocumentContent, error) {
	var (
		empty    inboxbandinghargasalvage.DocumentContent
		name     sql.NullString
		mimeType sql.NullString
		content  []byte
	)

	err := r.db.QueryRowContext(ctx, query("document_content"),
		documentID,
		q.DetailObject,
		q.SalvageID,
		strings.ToUpper(strings.TrimSpace(q.Reviewer.Name)),
	).Scan(&name, &mimeType, &content)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return empty, inboxbandinghargasalvage.ErrDocumentNotFound
	case err != nil:
		return empty, fmt.Errorf("membaca isi dokumen banding: %w", err)
	}

	return inboxbandinghargasalvage.DocumentContent{
		Name:     strings.TrimSpace(name.String),
		MIMEType: strings.TrimSpace(mimeType.String),
		Content:  content,
	}, nil
}
