package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/daftartipedokumen"
)

// Repo membaca dan menulis POOLDATA.LST_DOC_TYPE pada satu basis data entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh tipe dokumen.
func (r *Repo) List(ctx context.Context) ([]daftartipedokumen.DocumentType, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_type_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumen/sqlstore: membaca daftar tipe dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftartipedokumen.DocumentType
	for rows.Next() {
		doc, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("daftartipedokumen/sqlstore: membaca baris tipe dokumen: %w", err)
		}
		result = append(result, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftartipedokumen/sqlstore: menelusuri daftar tipe dokumen: %w", err)
	}
	return result, nil
}

// Get membaca satu tipe dokumen berdasarkan ID-nya.
func (r *Repo) Get(ctx context.Context, id string) (daftartipedokumen.DocumentType, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("document_type_get"), strings.TrimSpace(id))

	doc, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftartipedokumen.DocumentType{}, daftartipedokumen.ErrNotFound
	case err != nil:
		return daftartipedokumen.DocumentType{}, fmt.Errorf("daftartipedokumen/sqlstore: membaca tipe dokumen %q: %w", id, err)
	}
	return doc, nil
}

// InsertNew menerbitkan ID lalu menyisipkan barisnya.
//
// Ketiga langkahnya — membaca kode situs, mengambil nomor urut, menyisipkan — berada
// dalam SATU transaksi. Ini memperbaiki cacat nyata sistem lama: `PEGA_LST_DOC_TYPE.prc`
// tidak punya COMMIT sama sekali pada jalur berhasil sehingga bergantung pada commit
// pemanggilnya, sementara ketiga blok EXCEPTION-nya melakukan ROLLBACK atas transaksi
// yang bukan miliknya (`:17`, `:27`, `:40`) — membatalkan pekerjaan lain yang kebetulan
// belum ter-commit pada sesi yang sama. `D-68` menetapkan kepemilikan transaksi berpindah
// ke Go persis karena pola seperti itu.
func (r *Repo) InsertNew(
	ctx context.Context,
	input daftartipedokumen.Input,
	by daftartipedokumen.Editor,
) (daftartipedokumen.DocumentType, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return daftartipedokumen.DocumentType{}, fmt.Errorf("daftartipedokumen/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
	// Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan
	// menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	id, err := issueID(ctx, tx)
	if err != nil {
		return daftartipedokumen.DocumentType{}, err
	}

	if _, err := tx.ExecContext(ctx, getQuery("document_type_insert"),
		id, input.Type, input.ProcessStatus, by.Identity, by.At,
	); err != nil {
		return daftartipedokumen.DocumentType{}, fmt.Errorf("daftartipedokumen/sqlstore: menyisipkan tipe dokumen %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return daftartipedokumen.DocumentType{}, fmt.Errorf("daftartipedokumen/sqlstore: menutup transaksi sisip: %w", err)
	}

	return daftartipedokumen.DocumentType{
		ID:            id,
		Type:          input.Type,
		ProcessStatus: input.ProcessStatus,
	}, nil
}

// Update mengganti isi baris yang sudah ada.
func (r *Repo) Update(
	ctx context.Context,
	doc daftartipedokumen.DocumentType,
	by daftartipedokumen.Editor,
) error {
	result, err := r.db.ExecContext(ctx, getQuery("document_type_update"),
		doc.Type, doc.ProcessStatus, by.Identity, by.At, doc.ID,
	)
	if err != nil {
		return fmt.Errorf("daftartipedokumen/sqlstore: memperbarui tipe dokumen %q: %w", doc.ID, err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi.
	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
		// kegagalan: pernyataannya sendiri sudah berhasil.
		return nil
	}
	if affected == 0 {
		return daftartipedokumen.ErrNotFound
	}
	return nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("document_type_check_table"))
	if err != nil {
		return fmt.Errorf("daftartipedokumen/sqlstore: POOLDATA.LST_DOC_TYPE tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// issueID membentuk ID persis seperti `PEGA_LST_DOC_TYPE.prc` baris 12 dan 21: kode situs
// disambung nomor urut empat digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat kueri pada
// dialek Oracle.
func issueID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("document_type_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, ID tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu RETURN begitu saja
			// (`PEGA_LST_DOC_TYPE.prc:15-18`) — pemanggilnya tidak pernah tahu bahwa tidak
			// ada apa pun yang tersimpan. Di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("daftartipedokumen/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("daftartipedokumen/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("document_type_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftartipedokumen/sqlstore: mengambil nomor urut: %w", err)
	}

	return daftartipedokumen.FormatID(site, sequence), nil
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan berbentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(target ...any) error }

// scanRow membaca satu baris hasil kueri menjadi DocumentType.
//
// Ketiga kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui
// (`R-08` — DDL-nya tidak ada di export).
func scanRow(rows rowScanner) (daftartipedokumen.DocumentType, error) {
	var id, documentType, processStatus sql.NullString
	if err := rows.Scan(&id, &documentType, &processStatus); err != nil {
		return daftartipedokumen.DocumentType{}, err
	}
	return daftartipedokumen.DocumentType{
		ID:            strings.TrimSpace(id.String),
		Type:          strings.TrimSpace(documentType.String),
		ProcessStatus: strings.TrimSpace(processStatus.String),
	}, nil
}

var _ daftartipedokumen.Repo = (*Repo)(nil)
