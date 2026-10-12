package inboxrclpuclhttp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// TestJenisIsiDokumenBukanAkhiranTelanjang menjaga pemetaan jenis isi dokumen.
//
// # Kenapa uji ini ada
//
// Karena `DATA_ATTACHFILE.ATTACHMIMETYPE` TIDAK berisi jenis media. Ia berisi akhiran
// telanjang — `pdf`, `jpeg`, `PNG` — dan tidak satu pun dari 21 nilai berbedanya memuat
// tanda `/`.
//
// Versi pertama mengumumkannya apa adanya, sehingga peladen mengirim `Content-Type: pdf`.
// Itu bukan jenis media apa pun, dan peramban tidak menampilkan dokumennya. Cacatnya tidak
// menghasilkan satu pun galat di sisi peladen: permintaannya berhasil, isinya terkirim utuh,
// dan yang gagal hanya penggambarannya.
func TestJenisIsiDokumenBukanAkhiranTelanjang(t *testing.T) {
	kasus := []struct {
		nama     string
		tercatat string
		berkas   string
		mau      string
	}{
		{"akhiran pdf", "pdf", "Surat.pdf", "application/pdf"},
		{"akhiran huruf besar", "PNG", "Foto.PNG", "image/png"},
		{"jpeg dan jpg satu jenis", "jpg", "Foto.jpg", "image/jpeg"},
		{"jfif tidak dikenali tabel bawaan", "jfif", "Foto.jfif", "image/jpeg"},
		{"jenis media utuh dipakai apa adanya", "application/pdf", "x", "application/pdf"},
		{"kosong, jatuh ke nama berkas", "", "Lampiran.xlsx",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"tak dikenal sama sekali", "akhfsjkljdgf", "tanpa-akhiran",
			"application/octet-stream"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := documentContentType(inboxrclpucl.DocumentContent{
				Name:     k.berkas,
				MimeType: k.tercatat,
			})
			require.Equal(t, k.mau, hasil)

			// Tidak pernah mengumumkan akhiran telanjang, apa pun masukannya.
			require.Contains(t, hasil, "/",
				"jenis isi wajib berupa jenis media, bukan akhiran berkas")
		})
	}
}

// TestHanyaJenisAmanYangDigambarDiHalaman menjaga pemisahan inline dan attachment.
//
// # Kenapa ini bukan soal kerapian
//
// Berkas yang diunggah petugas disajikan dari ASAL YANG SAMA dengan aplikasi. Satu berkas
// `.html` atau `.svg` yang digambar sebagai halaman karena itu dapat menjalankan skrip atas
// nama petugas yang sedang masuk — dan nama berkasnyalah yang menentukan jenisnya, sehingga
// pemilihnya adalah pengunggah.
func TestHanyaJenisAmanYangDigambarDiHalaman(t *testing.T) {
	for _, jenis := range []string{
		"application/pdf", "image/jpeg", "image/png", "text/plain; charset=utf-8",
	} {
		require.True(t, jenisYangBolehTampil[jenis], "%s seharusnya boleh digambar", jenis)
	}

	for _, jenis := range []string{
		"text/html", "text/html; charset=utf-8", "image/svg+xml",
		"application/xhtml+xml", "application/octet-stream", "application/zip",
	} {
		require.False(t, jenisYangBolehTampil[jenis],
			"%s TIDAK boleh digambar di dalam halaman", jenis)
	}
}

// TestNamaBerkasTidakMerusakHeader menjaga pembersihan nama berkas.
//
// Namanya berasal dari ketikan petugas — kolom "Name" pada dialog unggah. Tanda kutip dan
// pergantian baris di dalamnya dapat menyudahi nilai header lebih awal lalu menyisipkan
// header lain.
func TestNamaBerkasTidakMerusakHeader(t *testing.T) {
	nakal := "Surat\".pdf\r\nX-Disuntikkan: ya"

	for _, header := range []string{inlineHeader(nakal), attachmentHeader(nakal)} {
		require.NotContains(t, header, "\r")
		require.NotContains(t, header, "\n")
		require.Equal(t, 2, countQuote(header), "tanda kutip hanya yang membungkus nama")
	}

	require.Equal(t, `inline; filename="dokumen"`, inlineHeader("   "))
	require.Equal(t, `attachment; filename="dokumen"`, attachmentHeader(""))
}

func countQuote(s string) int {
	n := 0
	for _, r := range s {
		if r == '"' {
			n++
		}
	}
	return n
}
