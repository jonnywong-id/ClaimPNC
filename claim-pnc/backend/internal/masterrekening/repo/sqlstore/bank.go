package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// BankRepo membaca daftar bank.
//
// Tabel milik sistem lain; aplikasi ini hanya membaca (ADR-0004).
//
// # Dua sumber, bukan satu
//
// Sumber kanonikalnya `GENERAL.LST_BANK_GROUP`, sama dengan yang dibaca Report Definition
// `BrowseBankGroup` dan RDB List `SearchCodeBank_sql` pada sistem lama. Pada basis data
// pengembangan, skema GENERAL **tidak terlihat sama sekali** oleh pengguna POOLDATA, dan
// kueri itu gagal dengan ORA-00942. Yang tersedia di sana adalah view pendamping
// `POOLDATA.VH_GENERAL_LST_BANK_GROUP` dengan kolom yang sama persis.
//
// Keduanya dicoba berurutan, bukan dipilih lewat konfigurasi. Sakelar konfigurasi akan
// menuntut seseorang mengetahui jawabannya saat memasang aplikasi — dan jawabannya
// berbeda per basis data, bukan per lingkungan.
type BankRepo struct {
	db *sql.DB

	// Sumber yang terbukti bekerja diingat setelah percobaan pertama, sehingga kueri
	// yang gagal tidak dijalankan lagi pada setiap kali daftar bank dibuka.
	//
	// Dijaga mutex karena satu repo dipakai bersama oleh seluruh permintaan HTTP yang
	// masuk; tanpa penjagaan, dua permintaan bersamaan menulis dan membaca field yang
	// sama dan `go test -race` menolaknya.
	lock  sync.Mutex
	named string
}

// NewBankRepo membentuk repo; db wajib sudah terhubung.
func NewBankRepo(db *sql.DB) *BankRepo { return &BankRepo{db: db} }

// List membaca seluruh bank.
func (r *BankRepo) List(ctx context.Context) ([]masterrekening.Bank, error) {
	if name := r.remembered(); name != "" {
		return r.read(ctx, name)
	}

	result, err := r.read(ctx, "bank_list")
	if err == nil {
		r.remember("bank_list")
		return result, nil
	}

	// Hanya "objeknya tidak terlihat" yang berpindah ke sumber cadangan. Galat lain —
	// koneksi putus, waktu habis, hak baca dicabut di tengah jalan — DITERUSKAN apa
	// adanya: menyembunyikannya di balik percobaan kedua akan mengubah gangguan nyata
	// menjadi daftar bank yang diam-diam berasal dari tempat lain.
	if !missingObject(err) {
		return nil, err
	}

	fallback, errFallback := r.read(ctx, "bank_list_pooldata")
	if errFallback != nil {
		// Galat yang dilaporkan adalah galat sumber KANONIKAL, dengan kegagalan cadangan
		// menempel padanya. Melaporkan yang sebaliknya akan menyesatkan: yang seharusnya
		// dibaca aplikasi ini adalah GENERAL.LST_BANK_GROUP.
		return nil, fmt.Errorf("%w (sumber cadangan juga gagal: %v)", err, errFallback)
	}

	r.remember("bank_list_pooldata")
	slog.Warn("daftar bank dibaca dari sumber cadangan",
		slog.String("modul", "masterrekening"),
		slog.String("kanonikal", "GENERAL.LST_BANK_GROUP"),
		slog.String("dipakai", "POOLDATA.VH_GENERAL_LST_BANK_GROUP"),
		slog.String("sebab", err.Error()))
	return fallback, nil
}

func (r *BankRepo) read(ctx context.Context, query string) ([]masterrekening.Bank, error) {
	rows, err := r.db.QueryContext(ctx, getQuery(query))
	if err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterrekening.Bank
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("masterrekening/sqlstore: membaca baris bank: %w", err)
		}
		b := masterrekening.Bank{Code: text(code), Name: text(name)}
		// Baris tanpa kode tidak dapat dipilih pengguna dan hanya akan menghasilkan
		// rekening tanpa bank. Ia dilewati di sini, bukan dibiarkan muncul di layar.
		if b.Code == "" {
			continue
		}
		if b.Name == "" {
			b.Name = b.Code
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: menelusuri daftar bank: %w", err)
	}
	return result, nil
}

func (r *BankRepo) remembered() string {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.named
}

func (r *BankRepo) remember(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.named = name
}

// missingObject menyatakan apakah galat ini berarti "objeknya tidak terlihat".
//
// Ia mencocokkan TEKS galat, bukan kode galat terketik, karena `database/sql` tidak
// membuka kode galat Oracle tanpa mengikat paket ini ke satu driver tertentu. ORA-00942
// adalah satu-satunya yang dicocokkan, dan cocoknya pun harus tepat — pencocokan yang
// longgar akan menelan kegagalan yang seharusnya terlihat.
func missingObject(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ORA-00942")
}

var _ masterrekening.BankRepo = (*BankRepo)(nil)
