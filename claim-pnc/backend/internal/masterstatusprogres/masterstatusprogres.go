// Package masterstatusprogres adalah inti modul Master Status Progres.
//
// # Apa yang dimodelkan di sini
//
// Satu klaim berjalan melewati beberapa **Posisi** (Register, Survey, Komite,
// Akseptasi). Di dalam setiap posisi, petugas mencatat **Status Progres** — keterangan
// sudah sampai mana pekerjaan pada posisi itu. Daftar status itulah yang dikelola modul
// ini, dan di sistem lama ia tinggal di POOLDATA.GCNM_MST_PROGRESS_KLAIM.
//
// Status Progres berjenjang dua tingkat:
//
//	Status Progres 1  POOLDATA.GCNM_MST_PROGRESS_KLAIM  <- yang dikerjakan paket ini
//	  └─ Status Progres 2  POOLDATA.GCNM_MST_PROGRESS    <- anak, merujuk ID_PROGRESS
//
// Tingkat 2 belum dibangun. Ia menempel pada modul yang sama karena kuncinya merujuk
// tingkat 1 (`GCNM_MST_PROGRESS.ID_PROGRESS`), bukan berdiri sendiri.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/StatusProgress-Harness.xml            layar "Master Status Progress 1"
//	Section/MasterStatusProgress-Section.xml       judul layar, tombol Tambah & Refresh
//	Section/BrowseStatusProgress-Section.xml       grid 3 kolom, form modal 3 isian
//	RDB List/BrowseStatusProgress-SQL.xml          daftar, ORDER BY ID_PROGRESS ASC
//	RDB List/InsertStatusProgress1-SQL.xml         sisip
//	RDB List/UpdateStatusProgress1_sql-SQL.xml     perbarui
//	RDB List/UpdateStatusProgress1-SQL.xml         ambil satu baris untuk disunting
//	RDB List/BrowseIDStatusProgress-SQL.xml        MAX(ID_PROGRESS)+1
//	Activity/InsertMstStatusProgress1_act-Act.xml  urutan langkah sisip
//	Activity/UpdateStatusProgress1_act-Act.xml     urutan langkah sunting
//
// # Penamaan ulang yang disengaja (D-19)
//
// Kueri lama mengaliaskan kolomnya ke nama yang tidak mencerminkan isi — persis utang
// teknis §4.2 `03-CURRENT-ARCHITECTURE.md`. Alias itu TIDAK dibawa:
//
//	ID_PROGRESS    AS "CaseID"  -> ID          (bukan nomor klaim sama sekali)
//	STS_PROGRESS1  AS "City"    -> Nama        (bukan nama kota)
//	STATUS         AS "CityID"  -> KodePosisi  (bukan kode kota; lihat posisi.go)
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterstatusprogres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ProgressStatus adalah satu baris master status progres tingkat 1.
type ProgressStatus struct {
	// ID adalah kunci baris, kolom ID_PROGRESS. Bentuknya warisan: lihat FormatID.
	ID string

	// Nama adalah keterangan status yang dibaca petugas, kolom STS_PROGRESS1.
	Name string

	// KodePosisi menyebut posisi klaim tempat status ini berlaku, kolom STATUS.
	// Nilainya salah satu Kode pada ListPositions().
	PositionCode string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Ia sengaja terpisah dari ProgressStatus: ID tidak pernah berasal dari pengguna pada
// penambahan — ia diturunkan dari isi tabel (lihat FormatID).
type Input struct {
	Name         string
	PositionCode string
}

// MaxNameLength adalah panjang maksimum kolom STS_PROGRESS1.
//
// Ditetapkan Work Owner 2026-09-17. Ia BUKAN lagi asumsi — versi sebelumnya memakai 200
// sebagai penjaga longgar karena DDL tabelnya tidak ada di export (R-08).
//
// Angka yang sama diulang di frontend (`FormStatusProgres.tsx`) supaya pengguna tahu
// sebelum mengirim. Server tetap yang berwenang; pemeriksaan di layar hanya kenyamanan.
// Bila angka ini berubah, KEDUA tempat harus ikut berubah — itu utang yang disadari dari
// menduplikasi sebuah angka, dan uji di bawah `masterstatusprogres_test.go` yang menjaganya
// tetap terlihat.
const MaxNameLength = 100

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterstatusprogres: status progres tidak ditemukan")

	// ErrIDTaken: ID yang akan disisipkan sudah dipakai baris lain.
	ErrIDTaken = errors.New("masterstatusprogres: ID status progres sudah dipakai")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Kolom adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (P-5). `InputRegister_act` sistem lama memeriksa
// belasan aturan lalu menampilkan semuanya bersamaan; mengembalikan satu galat per
// percobaan akan membuat pengguna menebak-nebak isian mana lagi yang salah
// (`11-CROSSCUTTING.md` §1.2 aturan 1).
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterstatusprogres: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas
// — bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	return Input{
		Name:         strings.TrimSpace(i.Name),
		PositionCode: strings.ToUpper(strings.TrimSpace(i.PositionCode)),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan semua pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
func (i Input) Check() error {
	var violation []Violation

	switch {
	case i.Name == "":
		violation = append(violation, Violation{
			Field:   "nama",
			Message: "Nama status progres wajib diisi.",
		})
	case len(i.Name) > MaxNameLength:
		violation = append(violation, Violation{
			Field:   "nama",
			Message: fmt.Sprintf("Nama status progres paling panjang %d karakter.", MaxNameLength),
		})
	}

	if i.PositionCode == "" {
		violation = append(violation, Violation{
			Field:   "kode_posisi",
			Message: "Posisi klaim wajib dipilih.",
		})
	} else if _, known := FindPosition(i.PositionCode); !known {
		// Menolak kode yang tidak dikenal adalah kendali yang di sistem lama diberikan
		// oleh dropdown. Kalau di sini diterima apa adanya, kendali itu hilang begitu
		// permintaan datang dari luar layar — dan API memang dapat ditembak langsung.
		violation = append(violation, Violation{
			Field:   "kode_posisi",
			Message: "Posisi klaim tidak dikenal.",
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// FormatID menyusun ID_PROGRESS dari nomor urut berikutnya.
//
// Bentuknya "0" diikuti nomor urut, direplikasi apa adanya dari
// `Activity/InsertMstStatusProgress1_act-Act.xml` yang menyusun
// `Local.idprogress1 := "0" + .City` atas hasil
// `SELECT NVL(MAX(ID_PROGRESS),0)+1` (`BrowseIDStatusProgress-SQL.xml`).
//
// KENAPA DIREPLIKASI, BUKAN DIPERBAIKI. Kolom ini adalah kunci yang dirujuk
// `GCNM_MST_PROGRESS.ID_PROGRESS` dan `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1`, dan baris
// lamanya sudah memakai bentuk ini. Mengganti formatnya akan memutus baris baru dari
// data historis. P-5 berlaku: perilaku dipertahankan lebih dulu, dan bentuk ini tidak
// ada di daftar 13 perbaikan eksplisit `D-49`.
//
// CACAT YANG IKUT TERBAWA, dicatat supaya tidak dianggap rancangan: karena "0" hanya
// disisipkan di depan tanpa pemadatan lebar, urutan teksnya tidak sama dengan urutan
// penerbitannya — "010" mendahului "09". Persoalan yang sama sudah dikenali pada nomor
// klaim `PNCN.YY.xxxx` (`D-71` butir 2). Daftar di layar karena itu diurutkan di basis
// data mengikuti kueri lama, bukan diurutkan ulang sebagai teks di frontend.
func FormatID(sequence int) string {
	return "0" + strconv.Itoa(sequence)
}

// Repo adalah seam ke penyimpanan master status progres SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memori. Satu instans Repo selalu terikat
// pada satu basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan
// di tingkat kueri (ADR-0030 Opsi 1). Yang memilih instans mana yang dipakai satu
// permintaan adalah RepoSelector.
type Repo interface {
	// List mengembalikan seluruh baris, terurut seperti kueri lama.
	List(ctx context.Context) ([]ProgressStatus, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (ProgressStatus, error)

	// InsertNew menurunkan ID dari isi tabel lalu menyisipkan barisnya, dan
	// mengembalikan baris yang benar-benar tersimpan beserta ID-nya.
	//
	// Penurunan ID berada di dalam satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi, karena nomornya diturunkan dari isi
	// tabel itu sendiri: memecahnya melebarkan jarak antara membaca MAX dan menyisipkan
	// — dan jarak itulah yang membuat dua penambahan bersamaan bertabrakan.
	//
	// Input sudah harus bersih dan lolos Check.
	InsertNew(ctx context.Context, input Input) (ProgressStatus, error)

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila
	// barisnya hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, sp ProgressStatus) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada
// saat permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
type RepoSelector func(portalAlias string) (Repo, error)
