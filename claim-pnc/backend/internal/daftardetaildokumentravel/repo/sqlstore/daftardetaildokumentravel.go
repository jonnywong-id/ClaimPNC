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

// List membaca seluruh aturan dokumen TANPA coverage-nya.
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

// Get membaca satu aturan lengkap dengan daftar coverage-nya.
//
// Dua kueri, bukan satu JOIN. Sebabnya bentuk hasilnya: JOIN akan mengembalikan baris
// induk berulang sebanyak coverage-nya, dan menyusunnya kembali menjadi satu objek
// berarti menulis pengelompokan sendiri di Go. Untuk satu baris yang dibuka pengguna,
// dua kueri jauh lebih terbaca dan bedanya tidak terukur.
func (r *Repo) Get(ctx context.Context, id string) (daftardetaildokumentravel.Detail, error) {
	key := strings.TrimSpace(id)

	row := r.db.QueryRowContext(ctx, getQuery("detail_get"), key)
	detail, err := scanDetail(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftardetaildokumentravel.Detail{}, daftardetaildokumentravel.ErrNotFound
	case err != nil:
		return daftardetaildokumentravel.Detail{}, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca detail %q: %w", id, err)
	}

	coverages, err := r.coveragesOf(ctx, key)
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	detail.Coverages = coverages
	return detail, nil
}

// InsertNew menerbitkan ID lalu menyisipkan barisnya beserta seluruh coverage-nya.
//
// Seluruhnya dalam SATU transaksi. Ini bukan kehati-hatian berlebihan: sebuah aturan
// dokumen yang tersimpan tanpa pembatasan plan-nya berarti dokumen itu berlaku untuk
// SELURUH plan — kebalikan dari yang dimaksud petugas, dan tidak terlihat sebagai galat
// di layar mana pun. `D-68` menetapkan kepemilikan transaksi berpindah ke Go persis
// karena kelas kegagalan seperti ini.
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

		coverages, err := insertCoverages(ctx, tx, id, input)
		if err != nil {
			return err
		}

		saved = daftardetaildokumentravel.Detail{
			ID:           id,
			DocumentID:   input.DocumentID,
			DocumentName: input.DocumentName,
			Mandatory:    input.Mandatory,
			MinUpload:    input.MinUpload,
			Coverages:    coverages,
		}
		return nil
	})
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return saved, nil
}

// Update mengganti isi satu aturan beserta SELURUH daftar coverage-nya.
//
// Penggantian coverage dilakukan dengan menghapus lalu menyisip ulang, di dalam
// transaksi yang sama dengan perubahan barisnya. Kegagalan di tengah karena itu tidak
// dapat meninggalkan aturan dokumen yang kehilangan seluruh pembatasan plan-nya.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	key := strings.TrimSpace(id)
	var saved daftardetaildokumentravel.Detail

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, getQuery("detail_update"),
			input.DocumentID, input.DocumentName, mandatoryCode(input.Mandatory), input.MinUpload, key)
		if err != nil {
			return fmt.Errorf("daftardetaildokumentravel/sqlstore: memperbarui detail %q: %w", id, err)
		}

		// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak
		// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
		// atas perubahan yang tidak pernah terjadi — lalu MENGHAPUS coverage milik baris
		// yang tidak ada, yang kebetulan tidak berakibat apa-apa hanya karena barisnya
		// memang tidak ada.
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return daftardetaildokumentravel.ErrNotFound
		}

		if _, err := tx.ExecContext(ctx, getQuery("detail_coverage_delete_all"), key); err != nil {
			return fmt.Errorf("daftardetaildokumentravel/sqlstore: membuang coverage detail %q: %w", id, err)
		}

		coverages, err := insertCoverages(ctx, tx, key, input)
		if err != nil {
			return err
		}

		saved = daftardetaildokumentravel.Detail{
			ID:           key,
			DocumentID:   input.DocumentID,
			DocumentName: input.DocumentName,
			Mandatory:    input.Mandatory,
			MinUpload:    input.MinUpload,
			Coverages:    coverages,
		}
		return nil
	})
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return saved, nil
}

// CheckTable memastikan kedua view modul ini ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
//
// Yang TIDAK diperiksa di sini: tabel dasar dan urutan yang dipakai jalur tulis. Nama
// ketiganya belum terverifikasi (lihat kepala berkas .sql), dan memeriksa urutan berarti
// MENGHABISKAN satu nomor — efek samping yang tidak pantas dimiliki mode periksa.
// Verifikasinya ada di migrations/0006, dijalankan DBA sekali, bukan setiap kali
// aplikasi diperiksa.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, name := range []string{"detail_check_table", "detail_coverage_check_table"} {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			return fmt.Errorf("daftardetaildokumentravel/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, err)
		}
		closeErr := rows.Err()
		_ = rows.Close()
		if closeErr != nil {
			return fmt.Errorf("daftardetaildokumentravel/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, closeErr)
		}
	}
	return nil
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

// coveragesOf membaca pembatasan plan dan jaminan milik satu baris detail.
func (r *Repo) coveragesOf(ctx context.Context, id string) ([]daftardetaildokumentravel.Coverage, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_coverage_list"), id)
	if err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca coverage detail %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetaildokumentravel.Coverage
	for rows.Next() {
		var coverageID, planID, planName, itemID, itemName sql.NullString
		if err := rows.Scan(&coverageID, &planID, &planName, &itemID, &itemName); err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca baris coverage: %w", err)
		}
		result = append(result, daftardetaildokumentravel.Coverage{
			ID:           strings.TrimSpace(coverageID.String),
			PlanID:       strings.TrimSpace(planID.String),
			PlanName:     strings.TrimSpace(planName.String),
			CoverageID:   strings.TrimSpace(itemID.String),
			CoverageName: strings.TrimSpace(itemName.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menelusuri coverage detail %q: %w", id, err)
	}
	return result, nil
}

// insertCoverages menyisipkan seluruh baris coverage milik satu detail.
func insertCoverages(
	ctx context.Context,
	tx *sql.Tx,
	parentID string,
	input daftardetaildokumentravel.Input,
) ([]daftardetaildokumentravel.Coverage, error) {
	if len(input.Coverages) == 0 {
		return nil, nil
	}

	result := make([]daftardetaildokumentravel.Coverage, 0, len(input.Coverages))
	for _, coverage := range input.Coverages {
		id, err := nextID(ctx, tx)
		if err != nil {
			return nil, err
		}

		// DOCID, DOCUMENTNAME, dan STSWAJIB ikut diisi meski sudah ada di baris induknya
		// — alasannya ada di komentar `detail_coverage_insert` pada berkas .sql.
		_, err = tx.ExecContext(ctx, getQuery("detail_coverage_insert"),
			id, parentID, input.DocumentID, input.DocumentName, mandatoryCode(input.Mandatory),
			coverage.PlanID, coverage.PlanName, coverage.CoverageID, coverage.CoverageName)
		if err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menyisipkan coverage detail %q: %w", parentID, err)
		}

		result = append(result, daftardetaildokumentravel.Coverage{
			ID:           id,
			PlanID:       coverage.PlanID,
			PlanName:     coverage.PlanName,
			CoverageID:   coverage.CoverageID,
			CoverageName: coverage.CoverageName,
		})
	}
	return result, nil
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

// PlanRepo membaca master plan dan jaminan Travel milik GISFW.
type PlanRepo struct {
	db *sql.DB
}

// NewPlanRepo membentuk pembaca master plan dan jaminan.
func NewPlanRepo(db *sql.DB) *PlanRepo { return &PlanRepo{db: db} }

// ListPlans membaca plan yang dapat dipilih.
func (r *PlanRepo) ListPlans(ctx context.Context) ([]daftardetaildokumentravel.Plan, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("plan_list"))
	if err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca master plan travel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetaildokumentravel.Plan
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca baris plan travel: %w", err)
		}
		result = append(result, daftardetaildokumentravel.Plan{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menelusuri master plan travel: %w", err)
	}
	return result, nil
}

// ListCoverages membaca jaminan yang dapat dipilih beserta plan pemiliknya.
func (r *PlanRepo) ListCoverages(ctx context.Context) ([]daftardetaildokumentravel.CoverageOption, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("coverage_list"))
	if err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca master jaminan travel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetaildokumentravel.CoverageOption
	for rows.Next() {
		var id, name, planID sql.NullString
		if err := rows.Scan(&id, &name, &planID); err != nil {
			return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: membaca baris jaminan travel: %w", err)
		}
		result = append(result, daftardetaildokumentravel.CoverageOption{
			ID:     strings.TrimSpace(id.String),
			Name:   strings.TrimSpace(name.String),
			PlanID: strings.TrimSpace(planID.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetaildokumentravel/sqlstore: menelusuri master jaminan travel: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.M_PLANTRAVEL dapat dibaca akun aplikasi.
func (r *PlanRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("plan_check_table"))
	if err != nil {
		return fmt.Errorf("daftardetaildokumentravel/sqlstore: POOLDATA.M_PLANTRAVEL tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

var (
	_ daftardetaildokumentravel.Repo         = (*Repo)(nil)
	_ daftardetaildokumentravel.DocumentRepo = (*DocumentRepo)(nil)
	_ daftardetaildokumentravel.PlanRepo     = (*PlanRepo)(nil)
)
