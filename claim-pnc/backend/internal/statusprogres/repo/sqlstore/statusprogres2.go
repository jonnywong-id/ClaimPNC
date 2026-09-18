package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/statusprogres"
)

// Repo2 membaca dan menulis POOLDATA.GCNM_MST_PROGRESS — Master Status Progres 2.
//
// Seperti Repo tingkat 1, satu instans terikat pada SATU koneksi basis data, yaitu satu
// portal entitas. Tidak ada satu pun kueri di sini yang menyaring berdasarkan entitas,
// dan memang tidak boleh ada (ADR-0030 Opsi 1).
type Repo2 struct {
	db *sql.DB
}

// Repo2Baru membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func Repo2Baru(db *sql.DB) *Repo2 { return &Repo2{db: db} }

// Daftar membaca seluruh status progres tingkat 2.
func (r *Repo2) Daftar(ctx context.Context) ([]statusprogres.StatusProgres2, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("statusprogres2_daftar"))
	if err != nil {
		return nil, fmt.Errorf("statusprogres2/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []statusprogres.StatusProgres2
	for baris.Next() {
		sp, err := pindaiBaris2(baris)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, sp)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("statusprogres2/sqlstore: menelusuri daftar: %w", err)
	}
	return hasil, nil
}

// Ambil membaca satu status progres tingkat 2 berdasarkan ID_MST-nya.
func (r *Repo2) Ambil(ctx context.Context, id string) (statusprogres.StatusProgres2, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("statusprogres2_ambil"), id)

	sp, err := pindaiBaris2(baris)
	if errors.Is(err, sql.ErrNoRows) {
		return statusprogres.StatusProgres2{}, statusprogres.ErrTidakDitemukan
	}
	if err != nil {
		return statusprogres.StatusProgres2{}, fmt.Errorf("statusprogres2/sqlstore: membaca %q: %w", id, err)
	}
	return sp, nil
}

// SisipBaru membaca induknya, menurunkan ID_MST, lalu menyisipkan barisnya.
//
// Ketiganya berjalan di dalam SATU transaksi — pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5, dengan alasan yang sama seperti tingkat 1: nomor baru
// diturunkan dari isi tabel itu sendiri.
//
// Urutannya mengikuti Activity/InsertMstStatusProgress2_act apa adanya:
//
//	1. cari status progress 1      -> baca baris induk
//	2. set status progress 1       -> salin namanya
//	3. CARI MAKS ID STATUS PROGRESS 2
//	4. SET KE LOCAL DAN TEMP
//	5. INSERT
//
// Induk dibaca LEBIH DULU, sebelum baris mana pun dikunci. Bila induknya tidak ada,
// penambahan ditolak tanpa sempat menahan kunci atas tabel tingkat 2 — kegagalan yang
// paling mungkin terjadi diletakkan paling awal, supaya ia paling murah.
func (r *Repo2) SisipBaru(ctx context.Context, isian statusprogres.Isian2) (statusprogres.StatusProgres2, error) {
	transaksi, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return statusprogres.StatusProgres2{}, fmt.Errorf("statusprogres2/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = transaksi.Rollback() }()

	induk, err := ambilInduk(ctx, transaksi, isian.IDInduk)
	if err != nil {
		return statusprogres.StatusProgres2{}, err
	}

	terpakai, err := idTerpakai2(ctx, transaksi)
	if err != nil {
		return statusprogres.StatusProgres2{}, err
	}

	baru := statusprogres.StatusProgres2{
		ID:      nomorBerikutnya2(terpakai),
		Nama:    isian.Nama,
		IDInduk: induk.ID,
		// Nama induk DISALIN ke kolom STS_PROGRESS1, bukan dibiarkan kosong dan bukan
		// dibaca lewat join saat menampilkan. Itu perilaku sistem lama yang
		// dipertahankan atas keputusan Work Owner 2026-09-18; konsekuensinya — salinan
		// yang dapat basi bila induknya diganti nama — dicatat pada statusprogres.Repo2.
		NamaInduk: induk.Nama,
	}

	// TIPE tidak ikut ditulis; lihat statusprogres2_sisip pada berkas .sql.
	if _, err := transaksi.ExecContext(ctx, ambilKueri("statusprogres2_sisip"),
		baru.ID, baru.NamaInduk, baru.Nama, baru.IDInduk,
	); err != nil {
		return statusprogres.StatusProgres2{}, fmt.Errorf("statusprogres2/sqlstore: menyisipkan %q: %w", baru.ID, err)
	}
	if err := transaksi.Commit(); err != nil {
		return statusprogres.StatusProgres2{}, fmt.Errorf("statusprogres2/sqlstore: menutup transaksi sisip: %w", err)
	}

	return baru, nil
}

// PeriksaTabel memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo2) PeriksaTabel(ctx context.Context) error {
	baris, err := r.db.QueryContext(ctx, ambilKueri("statusprogres2_periksa_tabel"))
	if err != nil {
		return fmt.Errorf("statusprogres2/sqlstore: POOLDATA.GCNM_MST_PROGRESS tidak dapat dibaca: %w", err)
	}
	defer func() { _ = baris.Close() }()
	return baris.Err()
}

// ambilInduk membaca satu baris Status Progres 1 di dalam transaksi yang sedang berjalan.
//
// Ia memakai kueri milik tingkat 1 (statusprogres_ambil), bukan kueri sendiri, supaya
// hanya ada SATU tempat yang tahu cara membaca tabel induk — termasuk alasan TRIM pada
// penyaringnya. Menyalinnya ke berkas tingkat 2 berarti dua kueri yang harus diingat
// bersamaan setiap kali tipe kolomnya berubah.
func ambilInduk(ctx context.Context, transaksi *sql.Tx, idInduk string) (statusprogres.StatusProgres, error) {
	baris := transaksi.QueryRowContext(ctx, ambilKueri("statusprogres_ambil"), idInduk)

	induk, err := pindaiBaris(baris)
	if errors.Is(err, sql.ErrNoRows) {
		// Galat yang KHUSUS, bukan ErrTidakDitemukan: yang hilang bukan baris yang
		// diminta pengguna, melainkan induk yang ia pilih dari dropdown. Layar
		// menanganinya berbeda — yang satu berarti "muat ulang daftar", yang lain berarti
		// "pilih induk lain".
		return statusprogres.StatusProgres{}, fmt.Errorf("%w: %q", statusprogres.ErrIndukTidakDitemukan, idInduk)
	}
	if err != nil {
		return statusprogres.StatusProgres{}, fmt.Errorf("statusprogres2/sqlstore: membaca induk %q: %w", idInduk, err)
	}
	return induk, nil
}

// idTerpakai2 mengunci baris yang ada lalu mengembalikan seluruh ID_MST-nya.
func idTerpakai2(ctx context.Context, transaksi *sql.Tx) ([]string, error) {
	baris, err := transaksi.QueryContext(ctx, ambilKueri("statusprogres2_daftar_id_terkunci"))
	if err != nil {
		return nil, fmt.Errorf("statusprogres2/sqlstore: mengunci daftar ID: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var terpakai []string
	for baris.Next() {
		var id sql.NullString
		if err := baris.Scan(&id); err != nil {
			return nil, fmt.Errorf("statusprogres2/sqlstore: membaca ID: %w", err)
		}
		terpakai = append(terpakai, strings.TrimSpace(id.String))
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("statusprogres2/sqlstore: menelusuri daftar ID: %w", err)
	}
	return terpakai, nil
}

// nomorBerikutnya2 menyusun ID_MST baru yang belum dipakai.
//
// Alasannya dihitung di Go — bukan dengan MAX di SQL — sama persis dengan
// nomorBerikutnya tingkat 1: bila kolomnya bertipe teks, MAX-nya adalah maksimum
// LEKSIKOGRAFIS, dan begitu tabel memuat "10" maksimumnya tetap "9". Tipe kolomnya
// belum diketahui (R-08), sehingga cacat itu mungkin sudah aktif hari ini atau mungkin
// tidak.
//
// Yang BERBEDA dari tingkat 1 hanyalah bentuk hasilnya: tanpa awalan "0"
// (statusprogres.FormatNomor2).
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
func nomorBerikutnya2(terpakai []string) string {
	sudahAda := make(map[string]bool, len(terpakai))
	tertinggi := 0
	for _, id := range terpakai {
		sudahAda[id] = true
		if angka, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && angka > tertinggi {
			tertinggi = angka
		}
	}

	for nomor := tertinggi + 1; ; nomor++ {
		kandidat := statusprogres.FormatNomor2(nomor)
		if !sudahAda[kandidat] {
			return kandidat
		}
	}
}

// pindaiBaris2 membaca satu baris hasil kueri menjadi StatusProgres2.
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
func pindaiBaris2(p pemindai) (statusprogres.StatusProgres2, error) {
	var id, nama, idInduk, namaInduk, tipe sql.NullString
	if err := p.Scan(&id, &nama, &idInduk, &namaInduk, &tipe); err != nil {
		return statusprogres.StatusProgres2{}, err
	}
	return statusprogres.StatusProgres2{
		ID:        strings.TrimSpace(id.String),
		Nama:      strings.TrimSpace(nama.String),
		IDInduk:   strings.TrimSpace(idInduk.String),
		NamaInduk: strings.TrimSpace(namaInduk.String),
		Tipe:      strings.TrimSpace(tipe.String),
	}, nil
}

var _ statusprogres.Repo2 = (*Repo2)(nil)
