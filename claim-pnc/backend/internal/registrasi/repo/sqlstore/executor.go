package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// executor adalah bagian *sql.DB dan *sql.Tx yang dipakai seluruh repo di paket ini.
//
// Dengan menyatakannya sebagai antarmuka, satu kode repo melayani dua keadaan: di luar
// transaksi ia berjalan atas *sql.DB, di dalam transaksi atas *sql.Tx. Tanpa itu setiap
// repo akan punya dua salinan — dan dua salinan yang harus dijaga tetap sama adalah cara
// paling pasti membuat yang satu tertinggal dari yang lain.
type executor interface {
	ExecContext(ctx context.Context, queries string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, queries string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, queries string, args ...any) *sql.Row
}

type contextKey string

const txKey contextKey = "registrasi_sqlstore_transaksi"

// UnitOfWork membuka transaksi basis data dan menitipkannya lewat context.
//
// # Kenapa lewat context, bukan lewat parameter
//
// Alternatifnya adalah menambahkan parameter transaksi pada setiap metode repo. Itu
// berarti seam `ClaimRepo` — yang dideklarasikan lapisan aturan — harus menyebut
// `*sql.Tx`, dan lapisan aturan menjadi tahu bahwa penyimpanannya berupa basis data SQL.
// Ketergantungan yang mengarah ke luar seperti itu tepat yang `ADR-0001` larang.
type UnitOfWork struct {
	db *sql.DB
}

// NewUnitOfWork membentuk unit kerja di atas sebuah koneksi.
func NewUnitOfWork(db *sql.DB) *UnitOfWork { return &UnitOfWork{db: db} }

// Run menjalankan kerja di dalam satu transaksi.
//
// Rollback dipanggil lewat defer dan diabaikan galatnya dengan sengaja: bila Commit
// sudah berhasil, Rollback mengembalikan sql.ErrTxDone yang bukan kegagalan. Yang
// berbahaya adalah kebalikannya — transaksi yang tidak pernah ditutup karena panic di
// tengah jalan — dan itulah yang defer ini cegah.
func (u *UnitOfWork) Run(ctx context.Context, work func(context.Context) error) error {
	// Transaksi bersarang tidak dibuka dua kali: bila context sudah membawa transaksi,
	// kerja ikut transaksi yang berjalan. Ini membuat usecase dapat memanggil usecase
	// lain tanpa saling membatalkan.
	if _, ok := txFrom(ctx); ok {
		return work(ctx)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membuka transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := work(context.WithValue(ctx, txKey, tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menutup transaksi: %w", err)
	}
	return nil
}

func txFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}

// executorFrom memilih transaksi berjalan bila ada, dan koneksi biasa bila tidak.
func executorFrom(ctx context.Context, db *sql.DB) executor {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return db
}

var _ registrasi.UnitOfWork = (*UnitOfWork)(nil)

// yesNo dan tidak menerjemahkan bool Go menjadi CHAR(1) Oracle.
//
// Oracle tidak punya tipe boolean di SQL, dan menyimpannya sebagai 0/1 menggoda orang
// menjumlahkannya. 'Y'/'N' tidak dapat dijumlahkan, dan terbaca sama oleh siapa pun yang
// membuka tabelnya langsung.
func yesNo(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

func fromYesNo(s string) bool { return s == "Y" || s == "y" }

// flagNOLL memetakan penanda Notice of Large Losses ke bentuk yang dipakai sistem lama.
//
// Ia sengaja TIDAK memakai "Y"/"N" seperti penanda lain di modul ini: sistem lama memakai
// `"1"` dan kosong (`Activity/SendEmailLargeLoss_act.xml` langkah 8 dan 11), dan kolom
// ini kelak dibaca berdampingan dengan data yang ditulis Pega selama masa paralel.
// Menulis "Y" akan membuat klaim yang sudah diberitahukan terbaca sebagai belum.
func flagNOLL(b bool) any {
	if b {
		return "1"
	}
	return nil
}

func fromFlagNOLL(s string) bool { return s == "1" }
