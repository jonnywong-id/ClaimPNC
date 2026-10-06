package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/daftardetaildokumentravel"
)

// sequenceDigits adalah lebar nomor urut pada ID baris.
//
// Harus sama dengan sequenceDigits pada repo/memory — kalau tidak, bentuk ID yang tampil
// saat pengembangan berbeda dari bentuk yang kelak datang dari Oracle, dan uji layar
// yang lulus tidak membuktikan apa-apa.
//
// Bentuk ID yang sebenarnya BELUM DIKETAHUI; alasan memilih lima digit berpadding nol
// ada di komentar repo/memory.
const sequenceDigits = 5

// Repo membaca dan menulis detail dokumen travel pada satu basis data entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh aturan dokumen.
func (r *Repo) List(ctx context.Context) ([]daftardetaildokumentravel.Detail, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_list"))
	if err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca daftar detail: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetaildokumentravel.Detail
	for rows.Next() {
		row, err := scanDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca baris detail: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menelusuri daftar detail: %w", err)
	}
	return result, nil
}

// Get membaca satu aturan dokumen.
func (r *Repo) Get(ctx context.Context, id string) (daftardetaildokumentravel.Detail, error) {
	row := r.db.QueryRowContext(ctx, getQuery("detail_get"), strings.TrimSpace(id))

	detail, err := scanDetail(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftardetaildokumentravel.Detail{}, daftardetaildokumentravel.ErrNotFound
	case err != nil:
		return daftardetaildokumentravel.Detail{}, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca detail %q: %w", id, err)
	}
	return detail, nil
}

// InsertNew menerbitkan ID lalu menyisipkan barisnya.
//
// Keduanya dalam SATU transaksi. Ini memperbaiki pola yang berulang di procedure warisan
// rumpun tabel ini: `DOCTRAVEL_CVG.prc:25` melakukan COMMIT sendiri di dalam cabang
// INSERT, sementara satu-satunya ROLLBACK-nya (`:53`) berada di handler terluar yang
// berjalan SESUDAH commit itu — sehingga tidak memulihkan apa pun. `D-68` menetapkan
// kepemilikan transaksi berpindah ke Go persis karena pola seperti itu.
func (r *Repo) InsertNew(
	ctx context.Context,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	var saved daftardetaildokumentravel.Detail

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		id, err := nextID(ctx, tx)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, getQuery("detail_insert"),
			id, input.DocumentID, input.DocumentName, mandatoryCode(input.Mandatory), input.MinUpload)
		if err != nil {
			return fmt.Errorf("daftardetaildokumentravel/sqlstore: menyisipkan detail %q: %w", id, err)
		}

		saved = daftardetaildokumentravel.Detail{
			ID:           id,
			DocumentID:   input.DocumentID,
			DocumentName: input.DocumentName,
			Mandatory:    input.Mandatory,
			MinUpload:    input.MinUpload,
		}
		return nil
	})
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return saved, nil
}

// Update mengganti isi satu aturan dokumen.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	key := strings.TrimSpace(id)

	result, err := r.db.ExecContext(ctx, getQuery("detail_update"),
		input.DocumentID, input.DocumentName, mandatoryCode(input.Mandatory), input.MinUpload, key)
	if err != nil {
		return daftardetaildokumentravel.Detail{}, fmt.Errorf("daftardetaildokumentravel/sqlstore: memperbarui detail %q: %w", id, err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi.
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return daftardetaildokumentravel.Detail{}, daftardetaildokumentravel.ErrNotFound
	}

	return daftardetaildokumentravel.Detail{
		ID:           key,
		DocumentID:   input.DocumentID,
		DocumentName: input.DocumentName,
		Mandatory:    input.Mandatory,
		MinUpload:    input.MinUpload,
	}, nil
}

// CheckTable memastikan view modul ini ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
//
// Yang TIDAK diperiksa di sini: tabel dasar dan urutan yang dipakai jalur tulis. Nama
// keduanya belum terverifikasi (lihat kepala berkas .sql), dan memeriksa urutan berarti
// MENGHABISKAN satu nomor — efek samping yang tidak pantas dimiliki mode periksa.
// Verifikasinya ada di migrations/0006, dijalankan DBA sekali, bukan setiap kali
// aplikasi diperiksa.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_check_table"))
	if err != nil {
		return fmt.Errorf("daftardetaildokumentravel/sqlstore: POOLDATA.V_LST_DOC_TRAVEL tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
func (r *Repo) inTransaction(ctx context.Context, work func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("daftardetaildokumentravel/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if err := work(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("daftardetaildokumentravel/sqlstore: menutup transaksi: %w", err)
	}
	return nil
}

// nextID mengambil nomor urut berikutnya dan memformatnya menjadi ID.
func nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("detail_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftardetaildokumentravel/sqlstore: mengambil nomor urut: %w", err)
	}

	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < sequenceDigits {
		digits = "0" + digits
	}
	return digits, nil
}

// mandatoryCode mengubah penanda wajib menjadi angka yang tersimpan di STSWAJIB.
//
// 1 dan 0, bukan 'Ya' dan 'Tidak'. Buktinya di `Activity/BrowseDocTravel-Act.xml`, yang
// precondition langkahnya berbunyi `.STSWAJIB==1` dan `.STSWAJIB==0`.
func mandatoryCode(mandatory bool) int {
	if mandatory {
		return 1
	}
	return 0
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan berbentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(target ...any) error }

// scanDetail membaca satu baris hasil kueri menjadi Detail.
//
// Seluruh kolom dibaca lewat tipe Null* lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena constraint tabelnya tidak diketahui (`R-08` — DDL-nya
// tidak ada di export).
//
// STSWAJIB dibaca sebagai ANGKA, bukan teks. Nilai selain 1 dijawab "tidak wajib",
// termasuk NULL — bukan ditolak sebagai galat. Baris lama dapat memuat apa saja, dan
// menggagalkan seluruh daftar karena satu baris berisi nilai tak terduga jauh lebih
// buruk daripada menampilkannya sebagai "Tidak".
func scanDetail(rows rowScanner) (daftardetaildokumentravel.Detail, error) {
	var id, documentID, documentName sql.NullString
	var mandatory, minUpload sql.NullInt64

	if err := rows.Scan(&id, &documentID, &documentName, &mandatory, &minUpload); err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return daftardetaildokumentravel.Detail{
		ID:           strings.TrimSpace(id.String),
		DocumentID:   strings.TrimSpace(documentID.String),
		DocumentName: strings.TrimSpace(documentName.String),
		Mandatory:    mandatory.Valid && mandatory.Int64 == 1,
		MinUpload:    int(minUpload.Int64),
	}, nil
}

// DocumentRepo membaca master dokumen travel sebagai daftar pilihan.
//
// Tipe tersendiri, bukan metode tambahan pada Repo, supaya batas kepemilikannya terbaca
// dari bentuknya: ia hanya punya List, dan tidak ada tempat untuk menambahkan operasi
// tulis tanpa sengaja (`P-1`).
type DocumentRepo struct {
	db *sql.DB
}

// NewDocumentRepo membentuk pembaca master dokumen travel.
func NewDocumentRepo(db *sql.DB) *DocumentRepo { return &DocumentRepo{db: db} }

// List membaca seluruh dokumen dari POOLDATA.M_DOCTRAVEL.
func (r *DocumentRepo) List(ctx context.Context) ([]daftardetaildokumentravel.Document, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_list"))
	if err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca master dokumen travel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetaildokumentravel.Document
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca baris master dokumen: %w", err)
		}
		result = append(result, daftardetaildokumentravel.Document{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menelusuri master dokumen travel: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.M_DOCTRAVEL dapat dibaca akun aplikasi.
func (r *DocumentRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("document_check_table"))
	if err != nil {
		return fmt.Errorf("daftardetaildokumentravel/sqlstore: POOLDATA.M_DOCTRAVEL tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

var (
	_ daftardetaildokumentravel.Repo         = (*Repo)(nil)
	_ daftardetaildokumentravel.DocumentRepo = (*DocumentRepo)(nil)
)
