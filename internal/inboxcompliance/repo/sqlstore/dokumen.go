package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// FindDocuments membaca dokumen satu klaim dari `POOLDATA.DATA_ATTACHFILE`.
//
// Tabel ini TIDAK diperiksa `-periksa`, sama alasannya dengan T_SURVEYORLIST: ia tabel
// produksi yang sudah terisi dan hanya dibaca di sini, bukan tabel baru yang menunggu DBA.
func (r *Repo) FindDocuments(
	ctx context.Context, reference string,
) ([]inboxcompliance.Document, error) {
	rows, err := r.db.QueryContext(ctx, query("find_claim_documents"), reference)
	if err != nil {
		return nil, fmt.Errorf(
			"membaca dokumen klaim dari POOLDATA.DATA_ATTACHFILE: %w", err)
	}
	defer rows.Close()

	documents := []inboxcompliance.Document{}

	for rows.Next() {
		var (
			id          sql.NullString
			name        sql.NullString
			mimeType    sql.NullString
			category    sql.NullString
			subCategory sql.NullString
			storageID   sql.NullString
			uploadedAt  sql.NullTime
		)
		if err := rows.Scan(
			&id, &name, &mimeType, &category, &subCategory, &storageID, &uploadedAt,
		); err != nil {
			return nil, fmt.Errorf("membaca baris dokumen klaim: %w", err)
		}

		document := inboxcompliance.Document{
			ID:          id.String,
			Name:        name.String,
			MimeType:    mimeType.String,
			Category:    category.String,
			SubCategory: subCategory.String,
			StorageID:   storageID.String,
		}
		if uploadedAt.Valid {
			at := uploadedAt.Time
			document.UploadedAt = &at
		}
		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri dokumen klaim: %w", err)
	}
	return documents, nil
}
