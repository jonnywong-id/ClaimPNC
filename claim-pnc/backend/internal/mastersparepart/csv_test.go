package mastersparepart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
)

// Header dibaca TANPA memandang huruf besar-kecil dan spasi di tepinya.
//
// Berkas nyata lahir dari Excel, dan Excel tidak menjaga keduanya. Menolak berkas hanya
// karena kapitalisasi akan menghasilkan kegagalan yang membingungkan — isinya benar, tetapi
// ditolak karena hal yang tidak dilihat pengguna.
func TestParseCSVHeaderTidakPekaHurufBesarKecil(t *testing.T) {
	berkas := " nama_spart , No_Spart ,KODE_SPART, harga_jual \n" +
		"FILTER OLI,NS-1,KD-1,250000\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "FILTER OLI", rows[0].Input.Name)
	require.Equal(t, "NS-1", rows[0].Input.Number)
	require.Equal(t, "KD-1", rows[0].Input.Code)
	require.Equal(t, "250000", rows[0].Input.SellingPrice)
}

// BOM yang ditulis Excel di depan header tidak boleh membuat kolom pertama tidak dikenal.
func TestParseCSVMembuangBOM(t *testing.T) {
	berkas := "\ufeffNAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n" +
		"SEAL KIT,NS-2,KD-2,90000\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "SEAL KIT", rows[0].Input.Name,
		"BOM di depan header membuat kolom pertama tidak terbaca")
}

// NAMA_SPART dihurufbesarkan — hanya pada jalur CSV, meniru
// `Activity/PNCUploadMasterSparepart_Act-Act.xml:91020` yang menyimpan hasil
// `@toUpperCase(TempSparepart.NAMA_SPART)` kembali ke properti yang sama.
//
// Uji ini ada karena perbedaannya dengan jalur form MUDAH tampak seperti cacat lalu
// "diperbaiki" menjadi seragam — dan keseragaman itulah yang justru menyimpang dari Pega.
func TestParseCSVMenghurufbesarkanNamaSparepart(t *testing.T) {
	berkas := "NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n" +
		"  filter Oli mesin  ,NS-9,KD-9,250000\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "FILTER OLI MESIN", rows[0].Input.Name)

	// Isian lain TIDAK ikut dihurufbesarkan — Pega hanya menyentuh NAMA_SPART.
	require.Equal(t, "NS-9", rows[0].Input.Number)
	require.Equal(t, "KD-9", rows[0].Input.Code)
}

// Nomor baris mengikuti nomor di BERKAS, dengan header sebagai baris 1.
//
// Tanpa itu, laporan "baris 3 gagal" menunjuk baris yang berbeda dari yang dilihat pengguna
// di penyunting teksnya.
func TestParseCSVNomorBarisIkutBerkas(t *testing.T) {
	berkas := "NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n" +
		"A,NS-1,KD-1,1\n" +
		"B,NS-2,KD-2,2\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Equal(t, 2, rows[0].Line)
	require.Equal(t, 3, rows[1].Line)
}

// Baris kosong di ujung berkas dilewati, bukan dilaporkan gagal.
func TestParseCSVMelewatiBarisKosong(t *testing.T) {
	berkas := "NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n" +
		"A,NS-1,KD-1,1\n" +
		",,,\n" +
		"\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1, "baris kosong tidak boleh ikut terbaca")
}

// Kolom yang ditetapkan server DIABAIKAN meski ada di berkas.
//
// Yang paling menentukan: APPROVAL. Menerimanya dari berkas membuat satu baris CSV dapat
// menyetujui dirinya sendiri, memintas seluruh antrean persetujuan.
func TestParseCSVMengabaikanKolomMilikServer(t *testing.T) {
	berkas := "ID,APPROVAL,USER_UPDATE,NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n" +
		"SP9999999999,1,SIAPAPUN,A,NS-1,KD-1,1\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Input memang tidak punya tempat untuk ketiganya — dan itulah pengamanannya.
	require.Equal(t, "A", rows[0].Input.Name)
	require.Equal(t, "NS-1", rows[0].Input.Number)
}

// Header wajib yang kurang ditolak DI MUKA, beserta nama kolomnya.
func TestParseCSVMenolakHeaderTidakLengkap(t *testing.T) {
	berkas := "NAMA_SPART,NO_SPART\nA,NS-1\n"

	_, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.ErrorIs(t, err, mastersparepart.ErrCSVHeaderMissing)
	require.Contains(t, err.Error(), "KODE_SPART")
	require.Contains(t, err.Error(), "HARGA_JUAL",
		"nama kolom yang kurang harus disebut; tanpa itu pengguna menebak di antara 19 kolom")
}

// Berkas tanpa satu baris data pun ditolak, bukan dilaporkan "0 baris berhasil".
func TestParseCSVMenolakBerkasKosong(t *testing.T) {
	_, err := mastersparepart.ParseCSV(strings.NewReader(
		"NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n"))
	require.ErrorIs(t, err, mastersparepart.ErrCSVEmpty)
}

// Batas baris ditegakkan, supaya satu berkas raksasa tidak menahan basis data.
func TestParseCSVMenolakTerlaluBanyakBaris(t *testing.T) {
	var b strings.Builder
	b.WriteString("NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n")
	for i := 0; i <= mastersparepart.MaxCSVRows; i++ {
		b.WriteString("A,NS,KD,1\n")
	}

	_, err := mastersparepart.ParseCSV(strings.NewReader(b.String()))
	require.ErrorIs(t, err, mastersparepart.ErrCSVTooManyRows)
}

// Baris yang kehilangan kolom terakhir dibaca apa adanya, bukan ditolak.
//
// Berkas nyata kerap begitu ketika nilai terakhirnya kosong.
func TestParseCSVMenerimaBarisKurangKolom(t *testing.T) {
	berkas := "NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL,SATUAN\n" +
		"A,NS-1,KD-1,1\n"

	rows, err := mastersparepart.ParseCSV(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "", rows[0].Input.Unit)
}
