package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// FindDocumentChecklist membaca daftar periksa kelengkapan dokumen satu klaim.
//
// Lihat kueri `find_document_checklist` untuk asal setiap kolomnya, dan
// `inboxcompliance.DocumentChecklistStage` untuk satu-satunya asumsi yang tersisa.
func (r *Repo) FindDocumentChecklist(
	ctx context.Context, reference string,
) ([]inboxcompliance.DocumentChecklistItem, error) {
	rows, err := r.db.QueryContext(
		ctx, query("find_document_checklist"), reference, reference)
	if err != nil {
		return nil, fmt.Errorf(
			"membaca daftar periksa dokumen dari POOLDATA.LST_TYPE_DOC_BUSINESS: %w", err)
	}
	defer rows.Close()

	// Senarai kosong, bukan nil — alasan yang sama dengan FindSurveyResults: lini yang
	// masternya belum diisi adalah keadaan normal, dan nil memaksa setiap pemanggil
	// mengingat perbedaan yang tidak berarti apa-apa.
	items := []inboxcompliance.DocumentChecklistItem{}

	for rows.Next() {
		var (
			categoryID     sql.NullString
			categoryName   sql.NullString
			mandatoryLabel sql.NullString
			minUpload      sql.NullInt64
			uploadedCount  int
		)
		if err := rows.Scan(
			&categoryID, &categoryName, &mandatoryLabel, &minUpload, &uploadedCount,
		); err != nil {
			return nil, fmt.Errorf("membaca baris daftar periksa dokumen: %w", err)
		}

		item := inboxcompliance.DocumentChecklistItem{
			CategoryID: categoryID.String,

			CategoryName: categoryName.String,

			// NULL menjadi teks kosong, dan itu BUKAN kehilangan arti — ia memang
			// tergambar kosong di Pega. Lihat catatan pada MandatoryLabel.
			MandatoryLabel: mandatoryLabel.String,

			UploadedCount: uploadedCount,
		}
		if minUpload.Valid {
			nilai := int(minUpload.Int64)
			item.MinUpload = &nilai
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri daftar periksa dokumen: %w", err)
	}

	return items, nil
}

// CheckDocumentChecklist melaporkan kesiapan ketiga objek tab Dokumen.
//
// ExecContext, BUKAN QueryRow().Scan() — alasan yang sama dengan CheckTable: kueri
// pemeriksanya ber-`WHERE 1 = 0` sehingga mengembalikan nol baris, dan `Scan` atas nol
// baris memulangkan `sql.ErrNoRows` yang akan dilaporkan sebagai kegagalan padahal
// pemeriksaannya justru berhasil.
//
// Yang diuji PARSING, bukan isi: Oracle memvalidasi seluruh nama kolom saat mem-parse,
// sehingga kolom yang salah nama gagal di sini — jauh sebelum petugas membuka tabnya.
func (r *Repo) CheckDocumentChecklist(ctx context.Context) error {
	if _, err := r.db.ExecContext(
		ctx, query("check_table_document_checklist"),
	); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.LST_TYPE_DOC_BUSINESS, POOLDATA.V_LST_DOC_TYPE, "+
				"dan POOLDATA.DATA_ATTACHFILE: %w", err)
	}
	return nil
}

// DocumentStageCounts mencacah kategori dokumen per TAHAP.
//
// # Ia bukan pemeriksa kesiapan — ia menjawab satu pertanyaan terbuka
//
// Tahap yang disaring tab Dokumen (`DocumentChecklistStage`) masih ASUMSI, karena Report
// Definition aslinya hilang dari export. Keluaran fungsi ini dibandingkan dengan jumlah
// baris grid di layar Pega: tahap yang angkanya cocok adalah jawabannya.
//
// Dipasang di perintah `check` supaya pertanyaannya terjawab oleh perkakas yang memang
// dijalankan sebelum rilis — bukan oleh seseorang yang kebetulan ingat untuk menanyakannya.
func (r *Repo) DocumentStageCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, query("count_document_stages"))
	if err != nil {
		return nil, fmt.Errorf("mencacah kategori dokumen per tahap: %w", err)
	}
	defer rows.Close()

	hasil := map[string]int{}
	for rows.Next() {
		var (
			tahap  sql.NullString
			jumlah int
		)
		if err := rows.Scan(&tahap, &jumlah); err != nil {
			return nil, fmt.Errorf("membaca cacahan tahap dokumen: %w", err)
		}
		hasil[tahap.String] = jumlah
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri cacahan tahap dokumen: %w", err)
	}
	return hasil, nil
}
