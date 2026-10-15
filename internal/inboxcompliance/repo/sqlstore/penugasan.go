package sqlstore

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// MoveAssignment menutup tahap lama dan membuka tahap baru, dalam SATU transaksi.
//
// # Kenapa transaksi, padahal modul ini di tempat lain menerimanya tanpa
//
// Karena di sini separuh hasil adalah keadaan yang TIDAK DAPAT DILIHAT siapa pun.
//
// Bandingkan dengan pasangan keputusan→Post Audit: bila yang kedua gagal, petugas melihat
// keputusannya tersimpan dan dapat mengulang, karena formnya masih terbuka. Di sini
// kebalikannya — bila penutupan tahap lama berhasil dan pembukaan tahap baru gagal, klaim
// **hilang dari antrean Compliance tanpa tiba di mana pun**. Tidak muncul di layar siapa
// pun, tidak ada galat, dan tidak ada yang mencarinya sampai seseorang menanyakan klaim
// yang tidak pernah selesai.
//
// Dua INSERT pada satu tabel dapat dibungkus satu transaksi tanpa menuntut kepemilikan
// transaksi di lapisan aplikasi — itulah sebabnya ia dikerjakan di sini dan sekarang,
// bukan ditunda bersama yang lain.
func (r *Repo) MoveAssignment(
	ctx context.Context, move inboxcompliance.AssignmentMove,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("memulai transaksi perpindahan penugasan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, a := range []inboxcompliance.Assignment{move.Closed, move.Opened} {
		if _, err := tx.ExecContext(
			ctx, query("insert_penugasan"),
			a.ID,
			a.Reference,
			a.Stage,
			a.Kind,
			nullableText(a.AssignedTo),
			nullableText(a.Workbasket),
			a.Status,
			a.CreatedAt,
		); err != nil {
			return fmt.Errorf(
				"menulis penugasan tahap %q (%s) ke POOLDATA.CPNC_PENUGASAN: %w",
				a.Stage, a.Status, storeMissing(err))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("menyelesaikan perpindahan penugasan: %w", err)
	}
	return nil
}

// CheckPenugasanWritable membuktikan tabel penugasan ada, terbaca, dan kolomnya sesuai.
//
// # Kenapa kegagalannya lebih berat daripada probe lain di modul ini
//
// Karena kueri DAFTAR ikut menyentuh tabel ini lewat `NOT EXISTS`. Tabel yang hilang
// tidak hanya menggagalkan perpindahan — ia **mengosongkan seluruh antrean Compliance**,
// dan layar yang kosong jauh lebih mudah disalahartikan sebagai "tidak ada pekerjaan"
// daripada sebagai kerusakan.
func (r *Repo) CheckPenugasanWritable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("check_table_penugasan")); err != nil {
		return fmt.Errorf(
			"tabel POOLDATA.CPNC_PENUGASAN tidak ada, tidak dapat diakses akun aplikasi, "+
				"atau kolomnya berbeda; jalankan migrations/0014_penugasan.up.sql di basis "+
				"data SETIAP entitas (DBA, `D-63`). Selama tabelnya belum ada, daftar "+
				"Inbox Compliance pun ikut kosong: %w",
			err)
	}
	return nil
}
