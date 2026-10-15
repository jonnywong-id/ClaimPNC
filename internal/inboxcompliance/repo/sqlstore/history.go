package sqlstore

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// AppendHistory menulis satu baris riwayat keputusan Compliance.
//
// Satu INSERT, tanpa transaksi — tidak ada yang perlu dibungkus bersama.
func (r *Repo) AppendHistory(
	ctx context.Context, entry inboxcompliance.HistoryEntry,
) error {
	if _, err := r.db.ExecContext(
		ctx, query("insert_history_claim"),
		entry.Reference,
		entry.RecordedAt,
		entry.Note,
		nullableText(entry.By),
	); err != nil {
		return fmt.Errorf(
			"menulis baris riwayat ke POOLDATA.LIST_HISTORY_CLAIM_PNC: %w",
			storeMissing(err))
	}
	return nil
}

// CheckHistoryWritable membuktikan tabel riwayat ada, terbaca, dan punya keempat kolomnya.
//
// # Kenapa ia diperiksa terpisah, dan kenapa itu penting di sini
//
// SKEMA tabelnya adalah KESIMPULAN, bukan bacaan: `PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc`
// menyebutnya tanpa skema, sehingga ia resolve ke pemilik procedure — yang disimpulkan
// POOLDATA. Satu-satunya tabel di modul ini yang alamatnya belum pernah dibaca langsung.
//
// Bila kesimpulan itu keliru, `-periksa` mengatakannya saat start — bukan petugas yang
// menemukannya setelah menekan Simpan pada keputusan yang menyangkut uang.
func (r *Repo) CheckHistoryWritable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("check_table_history")); err != nil {
		return fmt.Errorf(
			"tabel POOLDATA.LIST_HISTORY_CLAIM_PNC tidak ada, tidak dapat diakses akun "+
				"aplikasi, atau kolomnya berbeda dari yang ditulis "+
				"(CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE). Skemanya DISIMPULKAN "+
				"POOLDATA karena procedure aslinya menyebut tabelnya tanpa skema — bila "+
				"ia sebenarnya di skema lain, kueri insert_history_claim dan "+
				"check_table_history yang disesuaikan: %w",
			err)
	}
	return nil
}
