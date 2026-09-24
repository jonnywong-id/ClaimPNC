// Package sqlstore memenuhi seam mastercolsimasonline.Repo dan BusinessRepo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (`ADR-0030` Opsi 1) — tidak ada satu pun kueri di sini yang
// menyaring berdasarkan entitas, dan memang tidak boleh ada.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel —
// bukan bahwa tabel lama tidak boleh ditulis sama sekali. Layar Simas Online adalah
// satu-satunya penulis M_CAUSE_OF_LOSS_ONLINE di sistem lama: jalur Simpan-nya
// (`Activity/Online_nsertCauseOfLoss_act-Act.xml`) memanggil `UpdateMCauseOfLoss_online`,
// pemanggil tunggal `PEGA_M_CAUSE_OF_LOSS_ONLINE`. Memindahkan layarnya karena itu
// memindahkan kepemilikan tabelnya secara utuh.
//
// POOLDATA.M_CAUSE_OF_LOSS — master COL biasa — TIDAK disentuh modul ini sama sekali. Ia
// milik layar yang lain, dan keduanya sempat tertukar karena nama kolomnya sama persis.
//
// POOLDATA.BUSINESS juga TIDAK termasuk: ia milik GISFW dan hanya dibaca (`D-03`).
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/mastercolsimasonline"
)

// Repo membaca dan menulis POOLDATA.M_CAUSE_OF_LOSS_ONLINE beserta tabel pemetaan
// bisnisnya, POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh penyebab kerugian, tanpa pemetaan bisnisnya.
func (r *Repo) List(ctx context.Context) ([]mastercolsimasonline.CauseOfLoss, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("cause_of_loss_list"))
	if err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastercolsimasonline.CauseOfLoss
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca baris: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu penyebab kerugian LENGKAP dengan pemetaan bisnisnya.
func (r *Repo) Get(ctx context.Context, code string) (mastercolsimasonline.CauseOfLoss, error) {
	row, err := scanRow(r.db.QueryRowContext(ctx, getQuery("cause_of_loss_get"), code))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return mastercolsimasonline.CauseOfLoss{}, mastercolsimasonline.ErrNotFound
	case err != nil:
		return mastercolsimasonline.CauseOfLoss{}, fmt.Errorf("mastercolsimasonline/sqlstore: membaca %q: %w", code, err)
	}

	businesses, err := r.businessesOf(ctx, row.Code)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	row.Businesses = businesses
	return row, nil
}

// Insert menerbitkan kode baru lalu menyimpan barisnya beserta pemetaan bisnisnya.
//
// Seluruh langkahnya berada dalam SATU transaksi. Ini memperbaiki cacat nyata sistem
// lama: `PEGA_M_CAUSE_OF_LOSS_ONLINE` menjalankan COMMIT sendiri di dalam cabang INSERT,
// sementara satu-satunya ROLLBACK-nya berada di handler terluar yang berjalan SESUDAH
// commit itu — sehingga tidak memulihkan apa pun. `D-68` menetapkan kepemilikan
// transaksi berpindah ke Go persis karena pola seperti itu.
//
// Akibat yang harus disadari pada uji kesetaraan: bila penyimpanan gagal di tengah,
// sistem lama meninggalkan sebagian data sedangkan sistem baru tidak meninggalkan apa
// pun. Perbedaan itu DISENGAJA dan sudah dinyatakan di muka
// (`14-TESTING-STRATEGY.md` §6.4 butir 2).
func (r *Repo) Insert(ctx context.Context, data mastercolsimasonline.SaveData) (mastercolsimasonline.CauseOfLoss, error) {
	saved, err := r.inTransaction(ctx, func(tx *sql.Tx) (mastercolsimasonline.CauseOfLoss, error) {
		code, err := issueCode(ctx, tx)
		if err != nil {
			return mastercolsimasonline.CauseOfLoss{}, err
		}

		// Deskripsi ditulis DUA KALI — ke NAME_M_COL_ID dan ke COL_DESC. Alasannya ada
		// pada kueri cause_of_loss_insert: keduanya sama persis pada seluruh baris yang
		// ada, procedure lama hanya mengisi yang pertama, dan grid membaca yang kedua.
		if _, err := tx.ExecContext(ctx, getQuery("cause_of_loss_insert"), code, data.Description, data.Description); err != nil {
			return mastercolsimasonline.CauseOfLoss{}, fmt.Errorf("mastercolsimasonline/sqlstore: menyisipkan %q: %w", code, err)
		}
		if err := saveBusinesses(ctx, tx, code, data.Businesses); err != nil {
			return mastercolsimasonline.CauseOfLoss{}, err
		}

		return mastercolsimasonline.CauseOfLoss{
			Code:        code,
			Description: data.Description,
		}, nil
	})
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang benar-benar tersimpan,
	// termasuk urutan pemetaan bisnisnya sebagaimana basis data mengembalikannya.
	return r.Get(ctx, saved.Code)
}

// Update menyimpan perubahan pada baris yang sudah ada beserta pemetaan bisnisnya.
func (r *Repo) Update(ctx context.Context, code string, data mastercolsimasonline.SaveData) (mastercolsimasonline.CauseOfLoss, error) {
	_, err := r.inTransaction(ctx, func(tx *sql.Tx) (mastercolsimasonline.CauseOfLoss, error) {
		result, err := tx.ExecContext(ctx, getQuery("cause_of_loss_update"), data.Description, data.Description, code)
		if err != nil {
			return mastercolsimasonline.CauseOfLoss{}, fmt.Errorf("mastercolsimasonline/sqlstore: memperbarui %q: %w", code, err)
		}

		// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap kode yang
		// tidak ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan
		// "tersimpan" atas perubahan yang tidak pernah terjadi.
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return mastercolsimasonline.CauseOfLoss{}, mastercolsimasonline.ErrNotFound
		}

		if err := saveBusinesses(ctx, tx, code, data.Businesses); err != nil {
			return mastercolsimasonline.CauseOfLoss{}, err
		}
		return mastercolsimasonline.CauseOfLoss{}, nil
	})
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	return r.Get(ctx, code)
}

// CheckTable memastikan kedua tabel ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, check := range []struct {
		queryName string
		object    string
	}{
		{"cause_of_loss_check_table", "POOLDATA.M_CAUSE_OF_LOSS_ONLINE"},
		{"cause_of_loss_business_check_table", "POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL"},
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(check.queryName))
		if err != nil {
			return fmt.Errorf("mastercolsimasonline/sqlstore: %s tidak dapat dibaca: %w", check.object, err)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return fmt.Errorf("mastercolsimasonline/sqlstore: %s tidak dapat dibaca: %w", check.object, err)
		}
	}
	return nil
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
//
// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
// Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan
// menahan kunci baris sampai koneksinya didaur ulang.
func (r *Repo) inTransaction(
	ctx context.Context,
	work func(tx *sql.Tx) (mastercolsimasonline.CauseOfLoss, error),
) (mastercolsimasonline.CauseOfLoss, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, fmt.Errorf("mastercolsimasonline/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := work(tx)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	if err := tx.Commit(); err != nil {
		return mastercolsimasonline.CauseOfLoss{}, fmt.Errorf("mastercolsimasonline/sqlstore: menutup transaksi: %w", err)
	}
	return result, nil
}

// businessesOf membaca pemetaan bisnis satu penyebab kerugian.
func (r *Repo) businessesOf(ctx context.Context, code string) ([]mastercolsimasonline.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("cause_of_loss_business_list"), code)
	if err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca pemetaan bisnis %q: %w", code, err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastercolsimasonline.Business, 0)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca baris pemetaan bisnis: %w", err)
		}
		result = append(result, mastercolsimasonline.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: menelusuri pemetaan bisnis: %w", err)
	}
	return result, nil
}

// saveBusinesses menyimpan pemetaan bisnis satu penyebab kerugian.
//
// Barisnya dikenali menurut NAMANYA, bukan menurut ID maupun posisinya di grid: ID boleh
// kosong pada nama yang diketik bebas, dan POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL tidak
// punya kolom urutan. Alasan lengkapnya pada kueri cause_of_loss_business_update.
//
// # Namanya BUKAN replaceBusinesses, dan itu disengaja
//
// Ia tidak menggantikan: bisnis yang dicabut pengguna dari grid TIDAK terhapus. Tabelnya
// tidak punya penanda aktif, dan `D-66` melarang penghapusan fisik — sehingga pencabutan
// belum dapat disimpan sama sekali. Satu kolom `STS_AKTIF` dari DBA menutupnya; lihat
// peringatan pada kueri cause_of_loss_business_insert.
//
// Nama fungsinya menyatakan itu supaya pemanggil berikutnya tidak menyangka pencabutan
// sudah tertangani.
func saveBusinesses(ctx context.Context, tx *sql.Tx, code string, businesses []mastercolsimasonline.Business) error {
	for _, business := range businesses {
		result, err := tx.ExecContext(ctx, getQuery("cause_of_loss_business_update"),
			nullIfEmpty(business.ID), code, business.Name)
		if err != nil {
			return fmt.Errorf("mastercolsimasonline/sqlstore: memperbarui pemetaan bisnis %q: %w", business.Name, err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			// Driver yang tidak dapat melaporkan jumlah baris membuat upsert ini tidak
			// dapat memutuskan apa pun. Menyisipkan secara membabi buta berisiko baris
			// ganda, jadi kegagalannya dinyatakan terang-terangan.
			return fmt.Errorf("mastercolsimasonline/sqlstore: jumlah baris pemetaan bisnis tidak terbaca: %w", err)
		}
		if affected > 0 {
			continue
		}

		if _, err := tx.ExecContext(ctx, getQuery("cause_of_loss_business_insert"),
			code, nullIfEmpty(business.ID), business.Name); err != nil {
			return fmt.Errorf("mastercolsimasonline/sqlstore: menyisipkan pemetaan bisnis %q: %w", business.Name, err)
		}
	}
	return nil
}

// issueCode membentuk M_COL_ID persis seperti `Database/PEGA_M_CAUSE_OF_LOSS_ONLINE`:
// kode situs disambung nomor urut tiga digit, memakai urutan M_CAUSE_SEQ_ONLINE.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat kueri
// pada dialek Oracle.
func issueCode(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("cause_of_loss_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, kode tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("mastercolsimasonline/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("mastercolsimasonline/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("cause_of_loss_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("mastercolsimasonline/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + ThreeDigits(sequence), nil
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0') pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan kode
// ke-1000 karena itu menjadi LIMA karakter. Bila kolom M_COL_ID berlebar tetap empat
// seperti LSC_ID pada M_STS_CLAIM, penyisipannya akan DITOLAK basis data (ORA-12899),
// bukan diterima dengan kode aneh. Lebar M_COL_ID sendiri BELUM DIKETAHUI — DDL-nya
// belum ada (`R-08`) — dan itu ikut diminta bersama migrasi 0004.
//
// Dipotong menjadi tiga digit? Tidak. Itu akan menghasilkan kode GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas. Perilakunya dibiarkan apa
// adanya, persis seperti masterstatus.ThreeDigits.
//
// Diekspor supaya perilaku ini dapat diuji.
func ThreeDigits(n int64) string {
	digits := strconv.FormatInt(n, 10)
	for len(digits) < 3 {
		digits = "0" + digits
	}
	return digits
}

// nullIfEmpty menyimpan NULL, bukan teks kosong, untuk isian opsional yang tidak diisi.
//
// Keduanya berbeda di basis data, dan membiarkan keduanya masuk berarti dua bentuk
// "tidak diisi" yang harus sama-sama diingat setiap kueri sesudahnya.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

// scanRow membaca satu baris hasil kueri menjadi CauseOfLoss.
//
// Kedua kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom yang bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// keduanya nullable di katalog — M_COL_ID sekalipun, karena tabel ini tidak punya satu
// pun constraint PK, UK, maupun FK.
func scanRow(rows rowScanner) (mastercolsimasonline.CauseOfLoss, error) {
	var code, description sql.NullString
	if err := rows.Scan(&code, &description); err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	return mastercolsimasonline.CauseOfLoss{
		Code:        strings.TrimSpace(code.String),
		Description: strings.TrimSpace(description.String),
	}, nil
}

// BusinessRepo membaca POOLDATA.BUSINESS milik GISFW.
//
// Tipe terpisah, bukan method tambahan pada Repo, supaya "modul ini hanya membaca
// bisnis" terbaca dari bentuknya — dan supaya hak akses yang dibutuhkan keduanya dapat
// diminta terpisah ke DBA.
type BusinessRepo struct {
	db *sql.DB
}

// NewBusinessRepo membentuk repo master bisnis.
func NewBusinessRepo(db *sql.DB) *BusinessRepo { return &BusinessRepo{db: db} }

// List membaca seluruh bisnis, terurut menurut namanya.
func (r *BusinessRepo) List(ctx context.Context) ([]mastercolsimasonline.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("business_list"))
	if err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca daftar bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastercolsimasonline.Business, 0)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastercolsimasonline/sqlstore: membaca baris bisnis: %w", err)
		}
		result = append(result, mastercolsimasonline.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastercolsimasonline/sqlstore: menelusuri daftar bisnis: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.BUSINESS dapat dibaca akun aplikasi.
func (r *BusinessRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("business_check_table"))
	if err != nil {
		return fmt.Errorf("mastercolsimasonline/sqlstore: POOLDATA.BUSINESS tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

var (
	_ mastercolsimasonline.Repo         = (*Repo)(nil)
	_ mastercolsimasonline.BusinessRepo = (*BusinessRepo)(nil)
)
