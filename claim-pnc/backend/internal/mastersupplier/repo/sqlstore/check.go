package sqlstore

import (
	"context"
	"fmt"

	"claim-pnc/internal/mastersupplier"
)

// Berkas ini hanya dipakai mode periksa (`claimpnc -periksa`).
//
// Tidak satu pun fungsinya menulis, dan tidak satu pun mengambil baris berisi data
// nasabah maupun pihak ketiga — seluruhnya aman dijalankan terhadap produksi.

// CheckTable memastikan M_SUPPLIER ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("supplier_check_table"))
	if err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: M_SUPPLIER tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountAll menghitung seluruh baris M_SUPPLIER.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("supplier_count_all")).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastersupplier/sqlstore: menghitung baris: %w", err)
	}
	return total, nil
}

// CountReadable menghitung baris yang kunci NAMA-nya benar-benar terbaca JSON_VALUE.
//
// # Kenapa pemeriksaan ini menentukan, dan kenapa hanya ada di modul ini
//
// Seluruh isi supplier tinggal di SATU kolom JSONDATA. Bila kolomnya ternyata bukan JSON
// yang sah, atau kunci-kuncinya dinamai lain dari yang dibaca
// `RDB List/GetDataEditMasterSupller-SQL.xml`, maka `JSON_VALUE` menjawab **NULL alih-alih
// gagal** — dan layar akan menampilkan sederet baris berisi kolom kosong TANPA satu pun
// pesan galat.
//
// Itu kelas kegagalan yang tidak dimiliki modul master lain, yang membaca kolom bernama
// dan gagal keras ketika kolomnya tidak ada. Membandingkan angka ini dengan CountAll
// adalah cara termurah menangkapnya tanpa DDL (`R-08`).
func (r *Repo) CountReadable(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("supplier_check_document")).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastersupplier/sqlstore: menghitung baris terbaca: %w", err)
	}
	return total, nil
}

// CheckApprovalTable memastikan POOLDATA.PROTEKSI_KLAIMMBU dapat dibaca.
//
// Ia penting justru karena tabelnya milik proses lain: penambahan supplier menuntut
// penyisipan ke sana, dan supplier yang tersimpan tanpa baris permintaannya akan tertahan
// selamanya tanpa satu pun tanda di layar.
//
// Pemeriksaannya lewat pembacaan, bukan penulisan — mode periksa tidak menulis apa pun.
// Hak baca yang ada tidak menjamin hak tulis; itu tetap harus dipastikan DBA.
func (r *Repo) CheckApprovalTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("approval_check_table"))
	if err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: POOLDATA.PROTEKSI_KLAIMMBU tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountApprovalPending menghitung permintaan supplier yang masih di posisi awal.
func (r *Repo) CountApprovalPending(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("approval_count_pending"),
		mastersupplier.PositionRequested).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastersupplier/sqlstore: menghitung permintaan menunggu: %w", err)
	}
	return total, nil
}
