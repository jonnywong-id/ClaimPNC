// Package sqlstore memenuhi seam daftarobjekdokumen.Repo dan BusinessRepo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (`ADR-0030` Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar `ListDocumentObject` adalah
// satu-satunya layar Pega yang menulis objek dokumen: pencarian di seluruh export
// menemukan hanya dua rule yang menyentuhnya sebagai penulis
// (`CNMInsertLstDocObj_act` dan `SetsLstDocObjValue_act`, keduanya dirujuk section layar
// ini saja). Memindahkan layarnya karena itu memindahkan kepemilikan tabelnya secara utuh.
//
// POOLDATA.BUSINESS TIDAK termasuk: ia milik GISFW dan hanya dibaca (`D-03`).
//
// # Nama objek tulisnya DUGAAN
//
// Seluruh nama objek ada di `daftarobjekdokumen.sql` beserta tanda mana yang terbaca dari
// export dan mana yang dugaan. Jangan menyebut nama tabel di berkas ini.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/daftarobjekdokumen"
)

// sequenceDigits adalah lebar nomor urut pada ID.
//
// Empat, mengikuti `Database/PEGA_LST_DOC_TYPE.prc:21` — `lpad(to_char(...), 4, '0')` —
// yang merupakan procedure tabel bersaudara pada rumpun LST_* yang sama. Ia BUKAN tiga
// seperti rumpun M_CAUSE_OF_LOSS; lebar berbeda per rumpun dan harus dibaca dari
// procedure-nya masing-masing, tidak pernah disalin dari modul tetangga.
const sequenceDigits = 4

// Repo membaca dan menulis objek dokumen beserta tabel pemetaan bisnisnya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh objek dokumen, tanpa pemetaan bisnisnya.
func (r *Repo) List(ctx context.Context) ([]daftarobjekdokumen.DocumentObject, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_object_list"))
	if err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftarobjekdokumen.DocumentObject
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca baris: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu objek dokumen LENGKAP dengan pemetaan bisnisnya.
func (r *Repo) Get(ctx context.Context, id string) (daftarobjekdokumen.DocumentObject, error) {
	row, err := scanRow(r.db.QueryRowContext(ctx, getQuery("document_object_get"), id))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftarobjekdokumen.DocumentObject{}, daftarobjekdokumen.ErrNotFound
	case err != nil:
		return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca %q: %w", id, err)
	}

	businesses, err := r.businessesOf(ctx, row.ID)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	row.Businesses = businesses
	return row, nil
}

// Insert menerbitkan ID baru lalu menyimpan barisnya beserta pemetaan bisnisnya.
//
// Seluruh langkahnya berada dalam SATU transaksi, mengikuti `D-68` yang memindahkan
// kepemilikan transaksi ke Go. Procedure sejenis pada rumpun ini menjalankan ROLLBACK di
// dalam handler galatnya sendiri setelah sebagian pekerjaan sudah dilakukan; di sini
// kegagalan di tengah tidak meninggalkan apa pun.
//
// Akibat yang harus disadari pada uji kesetaraan: bila penyimpanan gagal di tengah, sistem
// lama meninggalkan sebagian data sedangkan sistem baru tidak meninggalkan apa pun.
// Perbedaan itu DISENGAJA dan sudah dinyatakan di muka (`14-TESTING-STRATEGY.md` §6.4).
func (r *Repo) Insert(ctx context.Context, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	saved, err := r.inTransaction(ctx, func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error) {
		id, err := issueID(ctx, tx)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}

		if _, err := tx.ExecContext(ctx, getQuery("document_object_insert"), id, data.Description); err != nil {
			return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: menyisipkan %q: %w", id, err)
		}
		if err := replaceBusinesses(ctx, tx, id, data.Businesses); err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}

		return daftarobjekdokumen.DocumentObject{ID: id, Description: data.Description}, nil
	})
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang benar-benar tersimpan,
	// termasuk urutan pemetaan bisnisnya sebagaimana basis data mengembalikannya.
	return r.Get(ctx, saved.ID)
}

// Update menyimpan perubahan pada baris yang sudah ada beserta pemetaan bisnisnya.
func (r *Repo) Update(ctx context.Context, id string, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	_, err := r.inTransaction(ctx, func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error) {
		result, err := tx.ExecContext(ctx, getQuery("document_object_update"), data.Description, id)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: memperbarui %q: %w", id, err)
		}

		// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak
		// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
		// atas perubahan yang tidak pernah terjadi.
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return daftarobjekdokumen.DocumentObject{}, daftarobjekdokumen.ErrNotFound
		}

		if err := replaceBusinesses(ctx, tx, id, data.Businesses); err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}
		return daftarobjekdokumen.DocumentObject{}, nil
	})
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return r.Get(ctx, id)
}

// CheckTable memastikan ketiga objek yang dipakai modul ini ada dan dapat dibaca akun
// aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
//
// Ketiganya diperiksa TERPISAH dengan sengaja: yang pertama namanya pasti, yang kedua dan
// ketiga dugaan. Membedakan mana yang gagal adalah satu-satunya cara mengetahui apakah
// dugaan itu benar — sebelum pengguna pertama menekan Simpan, bukan sesudahnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, check := range []struct {
		queryName string
		object    string
	}{
		{"document_object_check_table", "POOLDATA.V_LST_DOC_OBJ"},
		{"document_object_write_check_table", "POOLDATA.LST_DOC_OBJ"},
		{"document_object_business_check_table", "POOLDATA.LST_DOC_OBJ_BUSINESS"},
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(check.queryName))
		if err != nil {
			return fmt.Errorf("daftarobjekdokumen/sqlstore: %s tidak dapat dibaca: %w", check.object, err)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return fmt.Errorf("daftarobjekdokumen/sqlstore: %s tidak dapat dibaca: %w", check.object, err)
		}
	}
	return nil
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
//
// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
// Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
// kunci baris sampai koneksinya didaur ulang.
func (r *Repo) inTransaction(
	ctx context.Context,
	work func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error),
) (daftarobjekdokumen.DocumentObject, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := work(tx)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	if err := tx.Commit(); err != nil {
		return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: menutup transaksi: %w", err)
	}
	return result, nil
}

// businessesOf membaca pemetaan bisnis satu objek dokumen.
func (r *Repo) businessesOf(ctx context.Context, id string) ([]daftarobjekdokumen.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_object_business_list"), id)
	if err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca pemetaan bisnis %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]daftarobjekdokumen.Business, 0)
	for rows.Next() {
		var businessID, name sql.NullString
		if err := rows.Scan(&businessID, &name); err != nil {
			return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca baris pemetaan bisnis: %w", err)
		}
		result = append(result, daftarobjekdokumen.Business{
			ID:   strings.TrimSpace(businessID.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: menelusuri pemetaan bisnis: %w", err)
	}
	return result, nil
}

// replaceBusinesses menggantikan SELURUH pemetaan bisnis satu objek dokumen.
//
// # Namanya memang replace, dan di sini itu benar
//
// Berbeda dari saveBusinesses pada modul Master COL Simas Online, yang namanya sengaja
// BUKAN replace karena tabelnya tidak punya penanda aktif sehingga pencabutan tidak dapat
// disimpan sama sekali. Tabel modul ini dirancang sejak awal dengan STS_AKTIF, sehingga
// pencabutan benar-benar tersimpan — sebagai penandaan, bukan penghapusan (`D-66`).
//
// Langkahnya dua, dan urutannya mengikat:
//
//  1. seluruh baris milik objek dokumen ini ditandai TIDAK AKTIF
//  2. baris pada setiap posisi dihidupkan kembali dengan isi yang baru
//
// Barisnya dikenali menurut POSISINYA di grid, bukan menurut namanya. Dengan begitu urutan
// yang disusun pengguna terjaga, dan dua baris bernama sama tetap dua baris — keduanya sah,
// karena grid Pega tidak punya satu pun penanda keunikan.
func replaceBusinesses(ctx context.Context, tx *sql.Tx, id string, businesses []daftarobjekdokumen.Business) error {
	if _, err := tx.ExecContext(ctx, getQuery("document_object_business_deactivate"), id); err != nil {
		return fmt.Errorf("daftarobjekdokumen/sqlstore: menonaktifkan pemetaan bisnis %q: %w", id, err)
	}

	for index, business := range businesses {
		// Posisi dihitung mulai dari 1, mengikuti nomor baris yang dilihat pengguna di
		// layar — bukan indeks slice yang mulai dari 0.
		position := index + 1

		result, err := tx.ExecContext(ctx, getQuery("document_object_business_activate"),
			nullIfEmpty(business.ID), business.Name, id, position)
		if err != nil {
			return fmt.Errorf("daftarobjekdokumen/sqlstore: memperbarui pemetaan bisnis baris %d: %w", position, err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			// Driver yang tidak dapat melaporkan jumlah baris membuat upsert ini tidak
			// dapat memutuskan apa pun. Menyisipkan secara membabi buta berisiko baris
			// ganda, jadi kegagalannya dinyatakan terang-terangan.
			return fmt.Errorf("daftarobjekdokumen/sqlstore: jumlah baris pemetaan bisnis tidak terbaca: %w", err)
		}
		if affected > 0 {
			continue
		}

		if _, err := tx.ExecContext(ctx, getQuery("document_object_business_insert"),
			id, position, nullIfEmpty(business.ID), business.Name); err != nil {
			return fmt.Errorf("daftarobjekdokumen/sqlstore: menyisipkan pemetaan bisnis baris %d: %w", position, err)
		}
	}
	return nil
}

// issueID membentuk ID persis seperti `Database/PEGA_LST_DOC_TYPE.prc`: kode situs
// disambung nomor urut empat digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR termasuk
// yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat kueri pada dialek
// Oracle.
func issueID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("document_object_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, ID tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("daftarobjekdokumen/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("daftarobjekdokumen/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("document_object_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftarobjekdokumen/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + FourDigits(sequence), nil
}

// FourDigits meniru lpad(to_char(seq), 4, '0') pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 9999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan ID
// ke-10000 karena itu menjadi ENAM karakter. Bila kolom ID berlebar tetap lima, penyisipan
// berikutnya akan DITOLAK basis data (ORA-12899), bukan diterima dengan ID aneh. Lebar ID
// sendiri BELUM DIKETAHUI — DDL-nya belum ada (`R-08`) — dan itu ikut ditanyakan di migrasi
// 0008.
//
// Dipotong menjadi empat digit? Tidak. Itu akan menghasilkan ID GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas. Perilakunya dibiarkan apa adanya,
// persis seperti masterstatus.ThreeDigits.
//
// Diekspor supaya perilaku ini dapat diuji.
func FourDigits(n int64) string {
	digits := strconv.FormatInt(n, 10)
	for len(digits) < sequenceDigits {
		digits = "0" + digits
	}
	return digits
}

// nullIfEmpty menyimpan NULL, bukan teks kosong, untuk isian opsional yang tidak diisi.
//
// Keduanya berbeda di basis data, dan membiarkan keduanya masuk berarti dua bentuk "tidak
// diisi" yang harus sama-sama diingat setiap kueri sesudahnya.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

// scanRow membaca satu baris hasil kueri menjadi DocumentObject.
//
// Ketiga kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan ketiganya
// nullable — OLD_ID pasti, dan kedua lainnya belum dapat dipastikan karena DDL-nya belum
// diterima (`R-08`).
func scanRow(rows rowScanner) (daftarobjekdokumen.DocumentObject, error) {
	var id, description, oldID sql.NullString
	if err := rows.Scan(&id, &description, &oldID); err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return daftarobjekdokumen.DocumentObject{
		ID:          strings.TrimSpace(id.String),
		Description: strings.TrimSpace(description.String),
		OldID:       strings.TrimSpace(oldID.String),
	}, nil
}

// BusinessRepo membaca POOLDATA.BUSINESS milik GISFW.
//
// Tipe terpisah, bukan method tambahan pada Repo, supaya "modul ini hanya membaca bisnis"
// terbaca dari bentuknya — dan supaya hak akses yang dibutuhkan keduanya dapat diminta
// terpisah ke DBA.
type BusinessRepo struct {
	db *sql.DB
}

// NewBusinessRepo membentuk repo master bisnis.
func NewBusinessRepo(db *sql.DB) *BusinessRepo { return &BusinessRepo{db: db} }

// List membaca seluruh bisnis, terurut menurut namanya.
func (r *BusinessRepo) List(ctx context.Context) ([]daftarobjekdokumen.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("business_list"))
	if err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca daftar bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]daftarobjekdokumen.Business, 0)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca baris bisnis: %w", err)
		}
		result = append(result, daftarobjekdokumen.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: menelusuri daftar bisnis: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.BUSINESS dapat dibaca akun aplikasi.
func (r *BusinessRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("business_check_table"))
	if err != nil {
		return fmt.Errorf("daftarobjekdokumen/sqlstore: POOLDATA.BUSINESS tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

var (
	_ daftarobjekdokumen.Repo         = (*Repo)(nil)
	_ daftarobjekdokumen.BusinessRepo = (*BusinessRepo)(nil)
)
