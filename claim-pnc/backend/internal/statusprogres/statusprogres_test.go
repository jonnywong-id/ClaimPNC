package statusprogres_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/statusprogres"
)

// Nama isian wajib. Ini aturan pertama yang menolak isian pengguna, dan pesannya
// menyebut apa yang harus ia lakukan — bukan hanya bahwa ada yang salah.
func TestNamaWajibDiisi(t *testing.T) {
	isian := statusprogres.Isian{Nama: "   ", KodePosisi: "002"}.Bersihkan()

	err := isian.Periksa()
	require.Error(t, err)

	var validasi *statusprogres.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 1)
	require.Equal(t, "nama", validasi.Pelanggaran[0].Kolom)
}

// Seluruh pelanggaran dikembalikan sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku dengan Pega (P-5), bukan selera: `InputRegister_act`
// menampilkan semua pesan bersamaan, dan mengembalikan satu per satu akan membuat
// pengguna menebak isian mana lagi yang salah.
func TestSeluruhPelanggaranDikembalikanSekaligus(t *testing.T) {
	isian := statusprogres.Isian{Nama: "", KodePosisi: ""}.Bersihkan()

	err := isian.Periksa()
	var validasi *statusprogres.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 2, "nama kosong DAN posisi kosong, keduanya dilaporkan")

	kolom := map[string]bool{}
	for _, p := range validasi.Pelanggaran {
		kolom[p.Kolom] = true
	}
	require.True(t, kolom["nama"])
	require.True(t, kolom["kode_posisi"])
}

// Kode posisi di luar keempat yang dikenali ditolak.
//
// Di sistem lama kendali ini diberikan oleh dropdown. API dapat ditembak langsung tanpa
// melewati layar, sehingga kendalinya harus ada di server — kalau tidak, ia hilang.
func TestKodePosisiTidakDikenalDitolak(t *testing.T) {
	isian := statusprogres.Isian{Nama: "DOKUMEN DITERIMA", KodePosisi: "999"}.Bersihkan()

	err := isian.Periksa()
	var validasi *statusprogres.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 1)
	require.Equal(t, "kode_posisi", validasi.Pelanggaran[0].Kolom)
}

func TestIsianSahLolos(t *testing.T) {
	for _, posisi := range statusprogres.DaftarPosisi() {
		isian := statusprogres.Isian{Nama: "MENUNGGU DOKUMEN", KodePosisi: posisi.Kode}.Bersihkan()
		require.NoError(t, isian.Periksa(), "posisi %q seharusnya sah", posisi.Kode)
	}
}

func TestNamaTerlaluPanjangDitolak(t *testing.T) {
	panjang := strings.Repeat("A", statusprogres.BatasPanjangNama+1)
	isian := statusprogres.Isian{Nama: panjang, KodePosisi: "002"}.Bersihkan()

	err := isian.Periksa()
	var validasi *statusprogres.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Equal(t, "nama", validasi.Pelanggaran[0].Kolom)

	// Tepat di batas harus lolos. Kasus "tepat di batas" adalah tempat aturan seperti
	// ini paling sering salah (`14-TESTING-STRATEGY.md` §3.1).
	tepat := statusprogres.Isian{
		Nama:       strings.Repeat("A", statusprogres.BatasPanjangNama),
		KodePosisi: "002",
	}.Bersihkan()
	require.NoError(t, tepat.Periksa())
}

// Spasi di ujung isian dipangkas SEBELUM diperiksa, bukan sesudah — kalau tidak, nama
// berisi spasi saja akan lolos karena panjangnya bukan nol.
func TestBersihkanMemangkasDanMenyeragamkanHuruf(t *testing.T) {
	bersih := statusprogres.Isian{Nama: "  DOKUMEN DITERIMA  ", KodePosisi: " 002 "}.Bersihkan()

	require.Equal(t, "DOKUMEN DITERIMA", bersih.Nama)
	require.Equal(t, "002", bersih.KodePosisi)
}

// Keempat posisi adalah yang benar-benar ada di
// `Activity/ViewStatusProgress_act-Act.xml`. Bila daftarnya kelak pindah menjadi master
// data `F-4`, uji ini yang mengingatkan bahwa nilainya pernah ditetapkan di sini.
func TestDaftarPosisiSesuaiSistemLama(t *testing.T) {
	daftar := statusprogres.DaftarPosisi()
	require.Len(t, daftar, 4)

	require.Equal(t, "002", daftar[0].Kode)
	require.Equal(t, "REGISTER", daftar[0].Nama)
	require.Equal(t, "004", daftar[1].Kode)
	require.Equal(t, "SURVEY", daftar[1].Nama)
	require.Equal(t, "006", daftar[2].Kode)
	require.Equal(t, "KOMITE", daftar[2].Nama)
	require.Equal(t, "007", daftar[3].Kode)
	require.Equal(t, "AKSEPTASI", daftar[3].Nama)
}

// Kode 003 dan 005 tidak dipakai jalur ini; keduanya harus tetap tidak dikenal.
func TestCariPosisi(t *testing.T) {
	p, dikenal := statusprogres.CariPosisi("006")
	require.True(t, dikenal)
	require.Equal(t, "KOMITE", p.Nama)

	// Kolom STATUS bisa bertipe CHAR berlebar tetap yang memadatkan nilainya dengan
	// spasi tanpa memberi tanda apa pun (R-08: DDL belum ada).
	p, dikenal = statusprogres.CariPosisi("  006  ")
	require.True(t, dikenal)
	require.Equal(t, "KOMITE", p.Nama)

	_, dikenal = statusprogres.CariPosisi("003")
	require.False(t, dikenal)
	_, dikenal = statusprogres.CariPosisi("")
	require.False(t, dikenal)
}

// Kode yang tidak dikenal ditampilkan APA ADANYA, tidak disembunyikan.
//
// Baris lama dapat memuat kode di luar keempat yang dikenali — kolom STATUS tidak punya
// constraint yang membatasinya — dan menyembunyikannya membuat baris tampak kosong
// tanpa sebab. Pengguna perlu melihat apa yang benar-benar tersimpan.
func TestNamaPosisiTidakMenyembunyikanKodeAsing(t *testing.T) {
	require.Equal(t, "REGISTER", statusprogres.NamaPosisi("002"))
	require.Equal(t, "999", statusprogres.NamaPosisi("999"))
	require.Equal(t, "", statusprogres.NamaPosisi("   "))
}

// Bentuk nomor direplikasi apa adanya dari
// `Activity/InsertMstStatusProgress1_act-Act.xml`: `"0" + nomor urut`.
func TestFormatNomorMengikutiSistemLama(t *testing.T) {
	require.Equal(t, "01", statusprogres.FormatNomor(1))
	require.Equal(t, "09", statusprogres.FormatNomor(9))

	// Cacat yang ikut terbawa, dicatat sebagai uji supaya tidak disangka rancangan:
	// lebarnya tidak dipadatkan, sehingga urutan teksnya tidak sama dengan urutan
	// penerbitannya.
	require.Equal(t, "010", statusprogres.FormatNomor(10))
	require.True(t, statusprogres.FormatNomor(10) < statusprogres.FormatNomor(9),
		"sebagai teks, 010 mendahului 09 — inilah sebab daftar diurutkan di basis data")
}

// DaftarPosisi mengembalikan salinan; pemanggil yang mengubahnya tidak boleh merusak
// daftar bagi pemanggil berikutnya.
func TestDaftarPosisiTidakDapatDirusakPemanggil(t *testing.T) {
	pertama := statusprogres.DaftarPosisi()
	pertama[0].Nama = "DIRUSAK"

	kedua := statusprogres.DaftarPosisi()
	require.Equal(t, "REGISTER", kedua[0].Nama)
}
