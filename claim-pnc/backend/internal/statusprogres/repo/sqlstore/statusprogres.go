// Package sqlstore memenuhi seam statusprogres.Repo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/statusprogres"
)

//go:embed *.sql
var berkasKueri embed.FS

// kueri memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var kueri = muatSeluruhKueri()

// Repo membaca dan menulis POOLDATA.GCNM_MST_PROGRESS_KLAIM.
type Repo struct {
	db *sql.DB
}

// RepoBaru membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func RepoBaru(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca seluruh status progres.
func (r *Repo) Daftar(ctx context.Context) ([]statusprogres.StatusProgres, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("statusprogres_daftar"))
	if err != nil {
		return nil, fmt.Errorf("statusprogres/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []statusprogres.StatusProgres
	for baris.Next() {
		sp, err := pindaiBaris(baris)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, sp)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("statusprogres/sqlstore: menelusuri daftar: %w", err)
	}
	return hasil, nil
}

// Ambil membaca satu status progres berdasarkan ID-nya.
func (r *Repo) Ambil(ctx context.Context, id string) (statusprogres.StatusProgres, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("statusprogres_ambil"), id)

	sp, err := pindaiSatuBaris(baris)
	if errors.Is(err, sql.ErrNoRows) {
		return statusprogres.StatusProgres{}, statusprogres.ErrTidakDitemukan
	}
	if err != nil {
		return statusprogres.StatusProgres{}, fmt.Errorf("statusprogres/sqlstore: membaca %q: %w", id, err)
	}
	return sp, nil
}

// SisipBaru menurunkan ID dari isi tabel lalu menyisipkan barisnya.
//
// Keduanya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi:
// nomor baru diturunkan dari isi tabel itu sendiri, sehingga membaca dan menulisnya
// tidak dapat dipisahkan tanpa membuka kembali lubang balapan yang justru sedang
// ditutup. Alasannya dicatat di docs/keputusan-implementasi.md.
func (r *Repo) SisipBaru(ctx context.Context, isian statusprogres.Isian) (statusprogres.StatusProgres, error) {
	transaksi, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return statusprogres.StatusProgres{}, fmt.Errorf("statusprogres/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = transaksi.Rollback() }()

	terpakai, err := idTerpakai(ctx, transaksi)
	if err != nil {
		return statusprogres.StatusProgres{}, err
	}

	id := nomorBerikutnya(terpakai)
	if _, err := transaksi.ExecContext(ctx, ambilKueri("statusprogres_sisip"), id, isian.Nama, isian.KodePosisi); err != nil {
		return statusprogres.StatusProgres{}, fmt.Errorf("statusprogres/sqlstore: menyisipkan %q: %w", id, err)
	}
	if err := transaksi.Commit(); err != nil {
		return statusprogres.StatusProgres{}, fmt.Errorf("statusprogres/sqlstore: menutup transaksi sisip: %w", err)
	}

	return statusprogres.StatusProgres{ID: id, Nama: isian.Nama, KodePosisi: isian.KodePosisi}, nil
}

// Perbarui menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Perbarui(ctx context.Context, sp statusprogres.StatusProgres) error {
	hasil, err := r.db.ExecContext(ctx, ambilKueri("statusprogres_perbarui"), sp.Nama, sp.KodePosisi, sp.ID)
	if err != nil {
		return fmt.Errorf("statusprogres/sqlstore: memperbarui %q: %w", sp.ID, err)
	}

	terkena, err := hasil.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
		// kegagalan: pernyataannya sendiri sudah berhasil.
		return nil
	}
	if terkena == 0 {
		return statusprogres.ErrTidakDitemukan
	}
	return nil
}

// PeriksaTabel memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) PeriksaTabel(ctx context.Context) error {
	baris, err := r.db.QueryContext(ctx, ambilKueri("statusprogres_periksa_tabel"))
	if err != nil {
		return fmt.Errorf("statusprogres/sqlstore: POOLDATA.GCNM_MST_PROGRESS_KLAIM tidak dapat dibaca: %w", err)
	}
	defer func() { _ = baris.Close() }()
	return baris.Err()
}

// idTerpakai mengunci baris yang ada lalu mengembalikan seluruh ID-nya.
func idTerpakai(ctx context.Context, transaksi *sql.Tx) ([]string, error) {
	baris, err := transaksi.QueryContext(ctx, ambilKueri("statusprogres_daftar_id_terkunci"))
	if err != nil {
		return nil, fmt.Errorf("statusprogres/sqlstore: mengunci daftar ID: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var terpakai []string
	for baris.Next() {
		var id sql.NullString
		if err := baris.Scan(&id); err != nil {
			return nil, fmt.Errorf("statusprogres/sqlstore: membaca ID: %w", err)
		}
		terpakai = append(terpakai, strings.TrimSpace(id.String))
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("statusprogres/sqlstore: menelusuri daftar ID: %w", err)
	}
	return terpakai, nil
}

// nomorBerikutnya menyusun ID baru yang belum dipakai.
//
// # Kenapa nomornya dihitung di Go, bukan dengan MAX di SQL
//
// Kueri lama memakai `NVL(MAX(A.ID_PROGRESS),0)+1`. Bila ID_PROGRESS bertipe VARCHAR2,
// MAX-nya adalah maksimum LEKSIKOGRAFIS — dan begitu tabel memuat "010", maksimumnya
// tetap "09" karena '9' > '1' pada karakter kedua. Nomor berikutnya kembali menjadi 10,
// dan ID "010" diterbitkan dua kali. Tipe kolomnya sendiri belum diketahui karena DDL
// tidak ada di export (R-08), sehingga cacat itu mungkin sudah aktif hari ini atau
// mungkin tidak — bergantung pada tipe yang dipilih DBA dahulu.
//
// Menghitungnya di Go menghindari pertanyaan itu seluruhnya: setiap ID ditafsirkan
// sebagai angka, diambil yang terbesar, lalu ditambah satu. Untuk rentang yang kedua
// cara sepakat — satu sampai sembilan baris — hasilnya sama persis dengan Pega,
// sehingga uji kesetaraan tidak melihat selisih. Sekaligus memenuhi aturan Steering
// bahwa pemformatan angka dilakukan di Go, bukan di SQL.
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
func nomorBerikutnya(terpakai []string) string {
	sudahAda := make(map[string]bool, len(terpakai))
	tertinggi := 0
	for _, id := range terpakai {
		sudahAda[id] = true
		if angka, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && angka > tertinggi {
			tertinggi = angka
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat ID berbentuk lain — misalnya "010" yang diterbitkan cacat
	// MAX leksikografis kueri lama — supaya baris baru tidak menabraknya.
	for nomor := tertinggi + 1; ; nomor++ {
		kandidat := statusprogres.FormatNomor(nomor)
		if !sudahAda[kandidat] {
			return kandidat
		}
	}
}

type pemindai interface {
	Scan(tujuan ...any) error
}

// pindaiBaris membaca satu baris hasil kueri menjadi StatusProgres.
//
// Ketiga kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom yang bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang
// diketahui (R-08).
func pindaiBaris(p pemindai) (statusprogres.StatusProgres, error) {
	var id, nama, kodePosisi sql.NullString
	if err := p.Scan(&id, &nama, &kodePosisi); err != nil {
		return statusprogres.StatusProgres{}, err
	}
	return statusprogres.StatusProgres{
		ID:         strings.TrimSpace(id.String),
		Nama:       strings.TrimSpace(nama.String),
		KodePosisi: strings.TrimSpace(kodePosisi.String),
	}, nil
}

func pindaiSatuBaris(baris *sql.Row) (statusprogres.StatusProgres, error) {
	return pindaiBaris(baris)
}

// ambilKueri mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
func ambilKueri(nama string) string {
	teks, ada := kueri[nama]
	if !ada {
		panic(fmt.Sprintf("statusprogres/sqlstore: kueri %q tidak ditemukan di berkas .sql", nama))
	}
	return teks
}

func muatSeluruhKueri() map[string]string {
	hasil := map[string]string{}
	daftar, err := berkasKueri.ReadDir(".")
	if err != nil {
		panic("statusprogres/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, berkas := range daftar {
		isi, err := berkasKueri.ReadFile(berkas.Name())
		if err != nil {
			panic("statusprogres/sqlstore: tidak dapat membaca " + berkas.Name() + ": " + err.Error())
		}
		for nama, teks := range pecahPerNama(string(isi)) {
			if _, bentrok := hasil[nama]; bentrok {
				panic("statusprogres/sqlstore: nama kueri ganda: " + nama)
			}
			hasil[nama] = teks
		}
	}
	return hasil
}

// pecahPerNama memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func pecahPerNama(isi string) map[string]string {
	const penanda = "-- name:"
	hasil := map[string]string{}
	nama := ""
	var badan []string

	simpan := func() {
		if nama == "" {
			return
		}
		var pernyataan []string
		for _, baris := range badan {
			if strings.HasPrefix(strings.TrimSpace(baris), "--") {
				continue
			}
			pernyataan = append(pernyataan, baris)
		}
		if teks := strings.TrimSpace(strings.Join(pernyataan, "\n")); teks != "" {
			hasil[nama] = teks
		}
	}

	for _, baris := range strings.Split(isi, "\n") {
		if potong := strings.TrimSpace(baris); strings.HasPrefix(potong, penanda) {
			simpan()
			nama = strings.TrimSpace(strings.TrimPrefix(potong, penanda))
			badan = nil
			continue
		}
		badan = append(badan, baris)
	}
	simpan()
	return hasil
}

var _ statusprogres.Repo = (*Repo)(nil)
