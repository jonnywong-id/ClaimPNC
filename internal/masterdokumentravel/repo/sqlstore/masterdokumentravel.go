package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterdokumentravel"
)

// Repo membaca dan menulis POOLDATA.M_DOCTRAVEL pada satu basis data entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh dokumen travel.
func (r *Repo) List(ctx context.Context) ([]masterdokumentravel.TravelDocument, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("travel_document_list"))
	if err != nil {
		return nil, fmt.Errorf("masterdokumentravel/sqlstore: membaca daftar dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterdokumentravel.TravelDocument
	for rows.Next() {
		doc, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterdokumentravel/sqlstore: membaca baris dokumen: %w", err)
		}
		result = append(result, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterdokumentravel/sqlstore: menelusuri daftar dokumen: %w", err)
	}
	return result, nil
}

// Get membaca satu dokumen travel berdasarkan DOCID-nya.
func (r *Repo) Get(ctx context.Context, id string) (masterdokumentravel.TravelDocument, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("travel_document_get"), strings.TrimSpace(id))

	doc, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterdokumentravel.TravelDocument{}, masterdokumentravel.ErrNotFound
	case err != nil:
		return masterdokumentravel.TravelDocument{}, fmt.Errorf("masterdokumentravel/sqlstore: membaca dokumen %q: %w", id, err)
	}
	return doc, nil
}

// InsertNew menerbitkan DOCID lalu menyisipkan barisnya.
//
// Ketiga langkahnya — membaca kode situs, mengambil nomor urut, menyisipkan — berada
// dalam SATU transaksi. Ini memperbaiki cacat nyata sistem lama: `DOCTRAVEL_CVG.prc:25`
// melakukan COMMIT sendiri di dalam cabang INSERT, sementara satu-satunya ROLLBACK-nya
// (`:53`) berada di handler terluar yang berjalan SESUDAH commit itu — sehingga tidak
// memulihkan apa pun. `D-68` menetapkan kepemilikan transaksi berpindah ke Go persis
// karena pola seperti itu.
func (r *Repo) InsertNew(ctx context.Context, input masterdokumentravel.Input) (masterdokumentravel.TravelDocument, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterdokumentravel.TravelDocument{}, fmt.Errorf("masterdokumentravel/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	id, err := issueID(ctx, tx)
	if err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}

	if _, err := tx.ExecContext(ctx, getQuery("travel_document_insert"), id, input.Name); err != nil {
		return masterdokumentravel.TravelDocument{}, fmt.Errorf("masterdokumentravel/sqlstore: menyisipkan dokumen %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return masterdokumentravel.TravelDocument{}, fmt.Errorf("masterdokumentravel/sqlstore: menutup transaksi sisip: %w", err)
	}

	return masterdokumentravel.TravelDocument{ID: id, Name: input.Name}, nil
}

// Update mengganti judul dokumen yang sudah ada.
func (r *Repo) Update(ctx context.Context, doc masterdokumentravel.TravelDocument) error {
	result, err := r.db.ExecContext(ctx, getQuery("travel_document_update"), doc.Name, doc.ID)
	if err != nil {
		return fmt.Errorf("masterdokumentravel/sqlstore: memperbarui dokumen %q: %w", doc.ID, err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap DOCID yang tidak
	// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
	// atas perubahan yang tidak pernah terjadi.
	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
		// kegagalan: pernyataannya sendiri sudah berhasil.
		return nil
	}
	if affected == 0 {
		return masterdokumentravel.ErrNotFound
	}
	return nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("travel_document_check_table"))
	if err != nil {
		return fmt.Errorf("masterdokumentravel/sqlstore: POOLDATA.M_DOCTRAVEL tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// issueID membentuk DOCID persis seperti `DOCTRAVEL_CVG.prc` baris 12 dan 20: kode situs
// disambung nomor urut lima digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat kueri
// pada dialek Oracle.
func issueID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("travel_document_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, DOCID tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu RETURN begitu saja
			// (`DOCTRAVEL_CVG.prc:15-17`) — pemanggilnya tidak pernah tahu bahwa tidak
			// ada apa pun yang tersimpan. Di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("masterdokumentravel/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("masterdokumentravel/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("travel_document_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("masterdokumentravel/sqlstore: mengambil nomor urut: %w", err)
	}

	return masterdokumentravel.FormatID(site, sequence), nil
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan berbentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(target ...any) error }

// scanRow membaca satu baris hasil kueri menjadi TravelDocument.
//
// Kedua kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui
// (`R-08` — DDL-nya tidak ada di export).
func scanRow(rows rowScanner) (masterdokumentravel.TravelDocument, error) {
	var id, name sql.NullString
	if err := rows.Scan(&id, &name); err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}
	return masterdokumentravel.TravelDocument{
		ID:   strings.TrimSpace(id.String),
		Name: strings.TrimSpace(name.String),
	}, nil
}

var _ masterdokumentravel.Repo = (*Repo)(nil)
