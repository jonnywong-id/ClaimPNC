package dokumenpenunjang_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
)

var wib = time.FixedZone("WIB", 7*60*60)

// TestBersihkanNamaBerkasMenirutPega menjaga tiruan
// `@pxReplaceAllViaRegex(param.Filename,"[^a-zA-Z0-9]","")`.
//
// Baris ketiga adalah yang paling penting: titik ekstensinya IKUT hilang, dan itu memang
// perilaku Pega. Uji ini menyatakannya sebagai yang diharapkan, supaya orang berikutnya
// tidak "memperbaikinya" tanpa keputusan.
func TestBersihkanNamaBerkasMenirutPega(t *testing.T) {
	for _, uji := range []struct{ nama, masuk, keluar string }{
		{"spasi dibuang", "Foto Kerugian", "FotoKerugian"},
		{"tanda baca dibuang", "surat-tuntutan_v2", "surattuntutanv2"},
		{"titik ekstensi IKUT hilang", "Foto Kerugian.pdf", "FotoKerugianpdf"},
		{"angka dipertahankan", "IMG20260926", "IMG20260926"},
		{"huruf beraksen dibuang", "Kerugián.pdf", "Keruginpdf"},
		{"hanya tanda baca menjadi kosong", "___ ---", ""},
		{"kosong tetap kosong", "", ""},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			require.Equal(t, uji.keluar, dokumenpenunjang.BersihkanNamaBerkas(uji.masuk))
		})
	}
}

// TestEkstensiDiambilSebelumPembersihan menjaga urutan yang mudah terbalik.
//
// Membersihkan nama lebih dulu menghapus titiknya, sehingga ekstensi tidak lagi dapat
// dikenali dan SETIAP berkas jatuh ke tipe media bawaan. Terbaliknya tidak menghasilkan
// galat — hanya tipe media yang salah pada semuanya.
func TestEkstensiDiambilSebelumPembersihan(t *testing.T) {
	for _, uji := range []struct{ nama, masuk, keluar string }{
		{"biasa", "surat.pdf", "pdf"},
		{"huruf besar menjadi kecil", "FOTO.JPG", "jpg"},
		{"titik ganda ambil yang terakhir", "arsip.tar.gz", "gz"},
		{"tanpa titik", "surat", ""},
		{"titik di akhir", "surat.", ""},
		{"berspasi", "surat. pdf ", "pdf"},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			require.Equal(t, uji.keluar, dokumenpenunjang.EkstensiDari(uji.masuk))
		})
	}

	// Bila urutannya terbalik, yang ini yang gagal lebih dulu.
	require.Equal(t, "",
		dokumenpenunjang.EkstensiDari(dokumenpenunjang.BersihkanNamaBerkas("surat.pdf")),
		"nama yang SUDAH dibersihkan tidak lagi punya ekstensi")
}

// TestTipeMediaSalinanLimaBelasCabangPega menjaga peta
// `Activity/InsertDokumenPNC-Act.xml:3744` tetap utuh.
//
// Cabang terakhir ikut diuji: ekstensi tak dikenal menjadi `application/<ext>`, BUKAN
// `application/octet-stream`. Menggantinya adalah perubahan perilaku yang tidak diminta.
func TestTipeMediaSalinanLimaBelasCabangPega(t *testing.T) {
	for _, uji := range []struct{ ekstensi, mau string }{
		{"png", "image/png"},
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"avif", "image/avif"},
		{"txt", "text/plain"},
		{"doc", "application/msword"},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"pdf", "application/pdf"},
		{"eml", "message/rfc822"},
		{"rar", "application/vnd.rar"},
		{"zip", "application/zip"},
		{"csv", "text/csv"},
		{"xls", "application/vnd.ms-excel"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"ppt", "application/vnd.ms-powerpoint"},
		{"pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"PDF", "application/pdf"},
		{"heic", "application/heic"},
		{"", "application/"},
	} {
		t.Run(uji.ekstensi, func(t *testing.T) {
			require.Equal(t, uji.mau, dokumenpenunjang.TipeMedia(uji.ekstensi))
		})
	}
}

// TestFolderTanggalBerbentukDocTahunBulan menjaga tiruan
// `"Doc/"+substring(now,0,4)+"/"+substring(now,4,6)+"/"`.
func TestFolderTanggalBerbentukDocTahunBulan(t *testing.T) {
	require.Equal(t, "Doc/2026/09/",
		dokumenpenunjang.FolderTanggal(time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)))
	require.Equal(t, "Doc/2026/01/",
		dokumenpenunjang.FolderTanggal(time.Date(2026, time.January, 3, 0, 0, 0, 0, wib)),
		"bulan satu digit harus tetap dua karakter")
}

// TestFolderMemakaiTanggalWIB menyatakan SELISIH terencana dari Pega.
//
// Pega memakai GMT, sehingga unggahan pukul 01:00 WIB tanggal 1 Oktober masih tercatat
// 30 September dan mendarat di folder bulan sebelumnya. `F-5` menetapkan tanggal yang
// dilihat manusia dihitung terhadap WIB — dan folder ini dilihat manusia.
func TestFolderMemakaiTanggalWIB(t *testing.T) {
	dini := time.Date(2026, time.October, 1, 1, 0, 0, 0, wib)

	require.Equal(t, "Doc/2026/10/", dokumenpenunjang.FolderTanggal(dini))
	require.Equal(t, "Doc/2026/09/", dokumenpenunjang.FolderTanggal(dini.UTC()),
		"inilah folder yang Pega pakai — dicatat sebagai selisih, bukan ditiru")
}

// TestNomorKlaimKosongMenjadiTandaHubung menjaga tambalan kolom NOT NULL.
func TestNomorKlaimKosongMenjadiTandaHubung(t *testing.T) {
	require.Equal(t, "-", dokumenpenunjang.NomorKlaimUntukPenyimpanan(""))
	require.Equal(t, "-", dokumenpenunjang.NomorKlaimUntukPenyimpanan("   "))
	require.Equal(t, "PNC-1865", dokumenpenunjang.NomorKlaimUntukPenyimpanan("  PNC-1865 "))
}

// TestURLTanpaMasaBerlakuDianggapMasihBerlaku menjaga 128.379 baris warisan tetap terbuka.
func TestURLTanpaMasaBerlakuDianggapMasihBerlaku(t *testing.T) {
	saat := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	lampau := saat.Add(-time.Hour)
	depan := saat.Add(time.Hour)

	require.False(t, dokumenpenunjang.Document{}.Kedaluwarsa(saat),
		"tanpa EXPDATE harus dianggap masih berlaku")
	require.True(t, dokumenpenunjang.Document{ExpiresAt: &lampau}.Kedaluwarsa(saat))
	require.False(t, dokumenpenunjang.Document{ExpiresAt: &depan}.Kedaluwarsa(saat))
}

// TestPerluKonversiHanyaEmpatEkstensi menjaga prakondisi
// `Activity/Convert_Avif-Act.xml:672`.
func TestPerluKonversiHanyaEmpatEkstensi(t *testing.T) {
	for _, ekstensi := range []string{"png", "jpg", "jpeg", "pdf", "PNG", "JPEG", " pdf "} {
		require.True(t, dokumenpenunjang.PerluKonversi(ekstensi),
			"%q seharusnya dikonversi", ekstensi)
	}
	for _, ekstensi := range []string{"docx", "zip", "txt", "avif", "xlsx", ""} {
		require.False(t, dokumenpenunjang.PerluKonversi(ekstensi),
			"%q tidak boleh dikonversi", ekstensi)
	}
}

// TestEkstensiSetelahKonversiHanyaMengubahGambar menjaga pembedaan kedua cabang Pega.
//
// `Param.MimeType := "Avif"` HANYA ada di cabang PNG/JPG/JPEG. Menyeragamkannya akan
// menandai PDF sebagai `image/avif`, dan peramban menolak membukanya.
func TestEkstensiSetelahKonversiHanyaMengubahGambar(t *testing.T) {
	for _, ekstensi := range []string{"png", "jpg", "jpeg", "PNG"} {
		require.Equal(t, dokumenpenunjang.EkstensiHasilKonversi,
			dokumenpenunjang.EkstensiSetelahKonversi(ekstensi))
	}

	// PDF melewati konversi tetapi tipenya TIDAK berubah.
	require.Equal(t, "pdf", dokumenpenunjang.EkstensiSetelahKonversi("pdf"))
	require.Equal(t, "docx", dokumenpenunjang.EkstensiSetelahKonversi("docx"))
}

// TestEkstensiHasilKonversiDipetakanKeTipeAvif menutup rantainya.
//
// `"Avif"` bukan tipe media; ia ekstensi yang masih harus melewati peta. Bila peta itu
// tidak mengenalinya, hasilnya `application/Avif` — dan gambar yang benar-benar AVIF
// tersimpan dengan tipe yang salah.
func TestEkstensiHasilKonversiDipetakanKeTipeAvif(t *testing.T) {
	require.Equal(t, "image/avif",
		dokumenpenunjang.TipeMedia(dokumenpenunjang.EkstensiHasilKonversi))
}
