package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/masterstatusprogres"
)

// Repo2 membaca dan menulis POOLDATA.GCNM_MST_PROGRESS — Master Status Progres 2.
//
// Seperti Repo tingkat 1, satu instans terikat pada SATU koneksi basis data, yaitu satu
// portal entitas. Tidak ada satu pun kueri di sini yang menyaring berdasarkan entitas,
// dan memang tidak boleh ada (ADR-0030 Opsi 1).
type Repo2 struct {
	db *sql.DB
}

// NewRepo2 membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo2(db *sql.DB) *Repo2 { return &Repo2{db: db} }

// List membaca seluruh status progres tingkat 2.
func (r *Repo2) List(ctx context.Context) ([]masterstatusprogres.ProgressStatus2, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("progress_status2_list"))
	if err != nil {
		return nil, fmt.Errorf("masterstatusprogres2/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterstatusprogres.ProgressStatus2
	for rows.Next() {
		sp, err := scanRow2(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterstatusprogres2/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu status progres tingkat 2 berdasarkan ID_MST-nya.
func (r *Repo2) Get(ctx context.Context, id string) (masterstatusprogres.ProgressStatus2, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("progress_status2_get"), id)

	sp, err := scanRow2(rows)
	if errors.Is(err, sql.ErrNoRows) {
		return masterstatusprogres.ProgressStatus2{}, masterstatusprogres.ErrNotFound
	}
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("masterstatusprogres2/sqlstore: membaca %q: %w", id, err)
	}
	return sp, nil
}

// InsertNew membaca induknya, menurunkan ID_MST, lalu menyisipkan barisnya.
//
// Ketiganya berjalan di dalam SATU transaksi — pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5, dengan alasan yang sama seperti tingkat 1: nomor baru
// diturunkan dari isi tabel itu sendiri.
//
// Urutannya mengikuti Activity/InsertMstStatusProgress2_act apa adanya:
//
//  1. cari status progress 1      -> baca baris induk
//  2. set status progress 1       -> salin namanya
//  3. CARI MAKS ID STATUS PROGRESS 2
//  4. SET KE LOCAL DAN TEMP
//  5. INSERT
//
// Induk dibaca LEBIH DULU, sebelum baris mana pun dikunci. Bila induknya tidak ada,
// penambahan ditolak tanpa sempat menahan kunci atas tabel tingkat 2 — kegagalan yang
// paling mungkin terjadi diletakkan paling awal, supaya ia paling murah.
func (r *Repo2) InsertNew(ctx context.Context, input masterstatusprogres.Input2) (masterstatusprogres.ProgressStatus2, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("masterstatusprogres2/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	parent, err := getParent(ctx, tx, input.ParentID)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}

	used, err := usedIDs2(ctx, tx)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}

	fresh := masterstatusprogres.ProgressStatus2{
		ID:       nextID2(used),
		Name:     input.Name,
		ParentID: parent.ID,
		// Nama induk DISALIN ke kolom STS_PROGRESS1, bukan dibiarkan kosong dan bukan
		// dibaca lewat join saat menampilkan. Itu perilaku sistem lama yang
		// dipertahankan atas keputusan Work Owner 2026-09-18; konsekuensinya — salinan
		// yang dapat basi bila induknya diganti nama — dicatat pada masterstatusprogres.Repo2.
		ParentName: parent.Name,
	}

	// TIPE tidak ikut ditulis; lihat progress_status2_insert pada berkas .sql.
	if _, err := tx.ExecContext(ctx, getQuery("progress_status2_insert"),
		fresh.ID, fresh.ParentName, fresh.Name, fresh.ParentID,
	); err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("masterstatusprogres2/sqlstore: menyisipkan %q: %w", fresh.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("masterstatusprogres2/sqlstore: menutup transaksi sisip: %w", err)
	}

	return fresh, nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo2) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("progress_status2_check_table"))
	if err != nil {
		return fmt.Errorf("masterstatusprogres2/sqlstore: POOLDATA.GCNM_MST_PROGRESS tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// getParent membaca satu baris Status Progres 1 di dalam transaksi yang sedang berjalan.
//
// Ia memakai kueri milik tingkat 1 (progress_status_get), bukan kueri sendiri, supaya
// hanya ada SATU tempat yang tahu cara membaca tabel induk — termasuk alasan TRIM pada
// penyaringnya. Menyalinnya ke berkas tingkat 2 berarti dua kueri yang harus diingat
// bersamaan setiap kali tipe kolomnya berubah.
func getParent(ctx context.Context, tx *sql.Tx, parentID string) (masterstatusprogres.ProgressStatus, error) {
	rows := tx.QueryRowContext(ctx, getQuery("progress_status_get"), parentID)

	parent, err := scanRow(rows)
	if errors.Is(err, sql.ErrNoRows) {
		// Galat yang KHUSUS, bukan ErrNotFound: yang hilang bukan baris yang
		// diminta pengguna, melainkan induk yang ia pilih dari dropdown. Layar
		// menanganinya berbeda — yang satu berarti "muat ulang daftar", yang lain berarti
		// "pilih induk lain".
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("%w: %q", masterstatusprogres.ErrParentNotFound, parentID)
	}
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("masterstatusprogres2/sqlstore: membaca induk %q: %w", parentID, err)
	}
	return parent, nil
}

// usedIDs2 mengunci baris yang ada lalu mengembalikan seluruh ID_MST-nya.
func usedIDs2(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQuery("progress_status2_list_id_locked"))
	if err != nil {
		return nil, fmt.Errorf("masterstatusprogres2/sqlstore: mengunci daftar ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var used []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterstatusprogres2/sqlstore: membaca ID: %w", err)
		}
		used = append(used, strings.TrimSpace(id.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterstatusprogres2/sqlstore: menelusuri daftar ID: %w", err)
	}
	return used, nil
}

// nextID2 menyusun ID_MST baru yang belum dipakai.
//
// Alasannya dihitung di Go — bukan dengan MAX di SQL — sama persis dengan
// nomorBerikutnya tingkat 1: bila kolomnya bertipe teks, MAX-nya adalah maksimum
// LEKSIKOGRAFIS, dan begitu tabel memuat "10" maksimumnya tetap "9". Tipe kolomnya
// belum diketahui (R-08), sehingga cacat itu mungkin sudah aktif hari ini atau mungkin
// tidak.
//
// Yang BERBEDA dari tingkat 1 hanyalah bentuk hasilnya: tanpa awalan "0"
// (masterstatusprogres.FormatID2).
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
func nextID2(used []string) string {
	taken := make(map[string]bool, len(used))
	highest := 0
	for _, id := range used {
		taken[id] = true
		if number, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && number > highest {
			highest = number
		}
	}

	for number := highest + 1; ; number++ {
		candidate := masterstatusprogres.FormatID2(number)
		if !taken[candidate] {
			return candidate
		}
	}
}

// scanRow2 membaca satu baris hasil kueri menjadi ProgressStatus2.
//
// Kelima kolom dibaca lewat sql.NullString lalu dipangkas, dengan dua sebab yang sama
// seperti tingkat 1: kolom bertipe CHAR berlebar tetap memadatkan nilainya dengan spasi
// tanpa memberi tanda apa pun, dan baris lama dapat memuat NULL karena tabel ini tidak
// punya constraint NOT NULL yang diketahui (R-08).
//
// Urutan kolomnya mengikuti berkas .sql: ID_MST, STS_PROGRESS2, ID_PROGRESS,
// STS_PROGRESS1, TIPE. Ia sengaja TIDAK mengikuti urutan pada kueri Pega, yang menaruh
// salinan nama induk di posisi kedua — urutan di sini menempatkan baris ini sendiri lebih
// dulu, lalu induknya, sehingga terbaca sebagai "siapa ini, lalu anak siapa".
func scanRow2(p scanner) (masterstatusprogres.ProgressStatus2, error) {
	var id, name, parentID, parentName, tipe sql.NullString
	if err := p.Scan(&id, &name, &parentID, &parentName, &tipe); err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}
	return masterstatusprogres.ProgressStatus2{
		ID:         strings.TrimSpace(id.String),
		Name:       strings.TrimSpace(name.String),
		ParentID:   strings.TrimSpace(parentID.String),
		ParentName: strings.TrimSpace(parentName.String),
		Kind:       strings.TrimSpace(tipe.String),
	}, nil
}

var _ masterstatusprogres.Repo2 = (*Repo2)(nil)
