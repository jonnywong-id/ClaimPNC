package sqlstore

import (
	"context"
	"fmt"
)

// UpdateDocumentCategory memindahkan satu lampiran ke kategori lain.
//
// Mengembalikan false bila tidak ada baris yang terpengaruh — dokumennya tidak ada, atau
// ia milik klaim lain. Keduanya TIDAK dibedakan di sini; pemanggil menerjemahkannya
// menjadi satu galat yang sama, supaya jawaban tidak memberi tahu bahwa sebuah DATAID itu
// sah dan hanya milik orang lain.
func (r *Repo) UpdateDocumentCategory(
	ctx context.Context, reference, documentID, category string,
) (bool, error) {
	hasil, err := r.db.ExecContext(
		ctx, query("update_document_category"), category, documentID, reference)
	if err != nil {
		return false, fmt.Errorf(
			"memindahkan kategori lampiran di POOLDATA.DATA_ATTACHFILE: %w", err)
	}

	// RowsAffected dipakai, bukan diabaikan: ia satu-satunya cara membedakan "berhasil
	// dipindahkan" dari "tidak ada yang cocok". Tanpa itu, permintaan atas dokumen milik
	// klaim lain dijawab berhasil — dan petugas mengira kategorinya sudah berubah.
	jumlah, err := hasil.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("membaca jumlah baris yang dipindahkan: %w", err)
	}
	return jumlah > 0, nil
}
