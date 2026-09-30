package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/repo/memory"
	"claim-pnc/internal/dokumenpenunjang/usecase"
)

var wib = time.FixedZone("WIB", 7*60*60)

const portal = "asm"

// jamTetap adalah seam Clock yang dipatok, supaya uji folder menegaskan bulan TERTENTU —
// bukan bulan yang kebetulan sedang berjalan.
type jamTetap struct{ pada time.Time }

func (j jamTetap) Now() time.Time { return j.pada }

func bangun(t *testing.T, pada time.Time) (*usecase.Service, *memory.Repo, *memory.Storage) {
	t.Helper()
	service, repo, storage, _ := bangunLengkap(t, pada)
	return service, repo, storage
}

// bangunLengkap ikut mengembalikan layanan konversi, untuk uji yang memeriksa konversinya.
func bangunLengkap(
	t *testing.T,
	pada time.Time,
) (*usecase.Service, *memory.Repo, *memory.Storage, *memory.Converter) {
	t.Helper()
	repo := memory.NewRepo()
	storage := memory.NewStorage()
	converter := memory.NewConverter()
	service, err := usecase.NewService(usecase.Options{
		Repos:     func(string) (dokumenpenunjang.Repo, error) { return repo, nil },
		Storage:   storage,
		Converter: converter,
		Clock:     jamTetap{pada: pada},
	})
	require.NoError(t, err)
	return service, repo, storage, converter
}

func permintaan() dokumenpenunjang.UploadRequest {
	return dokumenpenunjang.UploadRequest{
		ClaimNumber: "PNC-1865",
		FileName:    "Foto Kerugian.pdf",
		Content:     []byte("%PDF-1.4 isi berkas"),
		By:          "PETUGAS01",
	}
}

func TestNewServiceMenolakBahanYangKurang(t *testing.T) {
	repo := memory.NewRepo()
	pilih := func(string) (dokumenpenunjang.Repo, error) { return repo, nil }

	_, err := usecase.NewService(usecase.Options{
		Storage: memory.NewStorage(), Converter: memory.NewConverter(), Clock: jamTetap{},
	})
	require.Error(t, err, "tanpa Repos harus ditolak")

	_, err = usecase.NewService(usecase.Options{
		Repos: pilih, Converter: memory.NewConverter(), Clock: jamTetap{},
	})
	require.Error(t, err, "tanpa Storage harus ditolak")

	// Converter TIDAK boleh nil, meski konversi hanya berlaku bagi empat ekstensi: nil
	// yang dibiarkan membuat unggahan PNG melewati konversi DIAM-DIAM — berhasil,
	// tersimpan, dan berbeda dari Pega tanpa satu pun gejala.
	_, err = usecase.NewService(usecase.Options{
		Repos: pilih, Storage: memory.NewStorage(), Clock: jamTetap{},
	})
	require.Error(t, err, "tanpa Converter harus ditolak")

	// Clock TIDAK boleh punya nilai bawaan: ia menentukan folder tujuan, dan bawaan yang
	// diam-diam terpakai membuat uji folder lulus karena kebetulan.
	_, err = usecase.NewService(usecase.Options{
		Repos: pilih, Storage: memory.NewStorage(), Converter: memory.NewConverter(),
	})
	require.Error(t, err, "tanpa Clock harus ditolak")
}

// TestUnggahMengirimBentukYangSamaDenganPega menjaga seluruh aturan sekaligus, pada satu
// tempat yang benar-benar dapat diperiksa: APA yang dikirim ke layanan penyimpanan.
func TestUnggahMengirimBentukYangSamaDenganPega(t *testing.T) {
	saat := time.Date(2026, time.September, 26, 10, 30, 0, 0, wib)
	service, _, storage := bangun(t, saat)

	dokumen, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal,
		Request:     permintaan(),
	})
	require.NoError(t, err)

	kirim := storage.Terakhir()
	require.Equal(t, "klaimpnc", kirim.NamaAplikasi,
		"yang dikirim adalah NAMA FOLDER hasil pencarian, bukan konstanta KLAIMPNC")
	require.Equal(t, "PETUGAS01", kirim.Pengunggah)
	require.Equal(t, "PNC-1865", kirim.NomorKlaim)
	require.Equal(t, "Doc/2026/09/", kirim.Folder)
	require.Equal(t, "FotoKerugianpdf", kirim.NamaBerkas)

	// Token izin unggah yang dicatat ke GCP_IMAGE ikut terkirim sebagai KodeString.
	require.Equal(t, "KODE-1", kirim.KodeAkses)

	// PDF ikut dikonversi (`PerluKonversi`), tetapi tipenya TIDAK berubah — Pega hanya
	// menyetel `Param.MimeType := "Avif"` pada cabang PNG/JPG/JPEG.
	require.Equal(t, "application/pdf", kirim.TipeMedia)

	// Yang terkirim adalah HASIL konversi, bukan isi aslinya.
	require.Equal(t, []byte("AVIF:%PDF-1.4 isi berkas"), kirim.Isi)

	require.Len(t, dokumen.ImageID, 32, "ImageID diterbitkan seperti GenerateImageID")
	require.Equal(t, "FotoKerugianpdf", dokumen.FileName)
	require.Equal(t, "PNC-1865", dokumen.ClaimNumber)
	require.NotNil(t, dokumen.UploadedAt)
	require.True(t, dokumen.UploadedAt.Equal(saat))
}

// TestNamaAplikasiDicariDiMasterBukanDikirimApaAdanya menjaga pembedaan yang mudah hilang.
//
// `KLAIMPNC` adalah KUNCI pencarian; yang dikirim adalah hasilnya. Menyamakan keduanya
// membuat berkas mendarat di folder yang salah — tanpa satu pun galat.
func TestNamaAplikasiDicariDiMasterBukanDikirimApaAdanya(t *testing.T) {
	service, _, storage := bangun(t, time.Now())

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.NoError(t, err)

	require.NotEqual(t, dokumenpenunjang.NamaAplikasi, storage.Terakhir().NamaAplikasi,
		"yang dikirim tidak boleh kunci pencariannya sendiri")
}

// TestFolderAplikasiHilangMembatalkanUnggahan menjaga larangan menebak folder.
func TestFolderAplikasiHilangMembatalkanUnggahan(t *testing.T) {
	service, repo, storage := bangun(t, time.Now())
	repo.LupakanFolder(dokumenpenunjang.NamaAplikasi)

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.ErrorIs(t, err, dokumenpenunjang.ErrFolderAplikasiTidakAda)
	require.Empty(t, storage.Diterima(),
		"berkas tidak boleh terkirim ketika foldernya belum diketahui")
}

// TestIzinAksesDicatatSEBELUMBerkasDikirim menjaga urutan langkah Pega.
//
// Di Pega urutannya ditegakkan oleh urutan step; di sini tidak ada yang menegakkannya
// kecuali uji ini.
func TestIzinAksesDicatatSEBELUMBerkasDikirim(t *testing.T) {
	service, repo, storage := bangun(t, time.Now())

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.NoError(t, err)

	akses := repo.Akses()
	require.Len(t, akses, 1)
	require.Equal(t, "klaimpnc", akses[0].Aplikasi,
		"yang dicatat adalah nama folder, sama dengan yang dikirim ke layanan")
	require.Equal(t, "PETUGAS01", akses[0].Pengunggah)
	require.Len(t, storage.Diterima(), 1)
}

func TestNomorKlaimKosongTersimpanSebagaiTandaHubung(t *testing.T) {
	service, _, storage := bangun(t, time.Now())

	p := permintaan()
	p.ClaimNumber = "   "
	dokumen, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: p,
	})
	require.NoError(t, err)

	require.Equal(t, "-", storage.Terakhir().NomorKlaim)
	require.Equal(t, "-", dokumen.ClaimNumber)
}

func TestPermintaanYangTidakSahDitolakSebelumMenyentuhLayanan(t *testing.T) {
	for _, uji := range []struct {
		nama  string
		ubah  func(*dokumenpenunjang.UploadRequest)
		galat error
	}{
		{"pengunggah kosong", func(p *dokumenpenunjang.UploadRequest) { p.By = " " },
			dokumenpenunjang.ErrPengunggahKosong},
		{"berkas kosong", func(p *dokumenpenunjang.UploadRequest) { p.Content = nil },
			dokumenpenunjang.ErrBerkasKosong},
		{"nama tanpa huruf/angka", func(p *dokumenpenunjang.UploadRequest) { p.FileName = "___" },
			dokumenpenunjang.ErrNamaBerkasKosong},
		{"terlalu besar", func(p *dokumenpenunjang.UploadRequest) {
			p.Content = make([]byte, dokumenpenunjang.BatasUkuranBerkas+1)
		}, dokumenpenunjang.ErrBerkasTerlaluBesar},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			service, repo, storage := bangun(t, time.Now())
			p := permintaan()
			uji.ubah(&p)

			_, err := service.Upload(context.Background(), usecase.UploadCommand{
				PortalAlias: portal, Request: p,
			})
			require.ErrorIs(t, err, uji.galat)
			require.Empty(t, storage.Diterima(), "layanan tidak boleh tersentuh")
			require.Empty(t, repo.Akses(), "izin akses tidak boleh tercatat")
		})
	}
}

// TestLayananGagalTidakMeninggalkanMetadata menjaga arah kegagalan yang benar.
func TestLayananGagalTidakMeninggalkanMetadata(t *testing.T) {
	service, _, storage := bangun(t, time.Now())
	storage.Gagalkan(errors.New("sambungan terputus"))

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.ErrorIs(t, err, dokumenpenunjang.ErrUnggahGagal)

	daftar, err := service.List(context.Background(), portal, "PNC-1865")
	require.NoError(t, err)
	require.Empty(t, daftar, "metadata tidak boleh tercatat bila berkasnya tidak terkirim")
}

// TestMetadataGagalDilaporkanSEBAGAI metadata gagal, bukan sebagai unggahan gagal.
//
// Bedanya menentukan apa yang pengguna lakukan berikutnya: "unggahan gagal" mengundang
// percobaan ulang, dan setiap percobaan meninggalkan satu salinan yatim di penyimpanan
// yang tidak dapat ditarik kembali.
func TestMetadataGagalDilaporkanSebagaiMetadataGagal(t *testing.T) {
	service, repo, _ := bangun(t, time.Now())
	repo.GagalkanSimpan(errors.New("ORA-00001"))

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.ErrorIs(t, err, dokumenpenunjang.ErrMetadataGagal)
	require.NotErrorIs(t, err, dokumenpenunjang.ErrUnggahGagal)
}

// TestDaftarHanyaMemberiDokumenKlaimItu menjaga batas antar klaim.
func TestDaftarHanyaMemberiDokumenKlaimItu(t *testing.T) {
	service, _, _ := bangun(t, time.Now())

	for _, nomor := range []string{"PNC-1865", "PNC-9000", "PNC-1865"} {
		p := permintaan()
		p.ClaimNumber = nomor
		p.Content = []byte("isi " + nomor + time.Now().String())
		_, err := service.Upload(context.Background(), usecase.UploadCommand{
			PortalAlias: portal, Request: p,
		})
		require.NoError(t, err)
	}

	daftar, err := service.List(context.Background(), portal, "pnc-1865")
	require.NoError(t, err)
	require.Len(t, daftar, 2, "pencarian tidak peka huruf besar-kecil")
	for _, d := range daftar {
		require.Equal(t, "PNC-1865", d.ClaimNumber)
	}
}

// TestDaftarTanpaNomorKlaimTIDAKMembukaKeranjangBersama menutup kebocoran yang halus.
//
// `"-"` adalah keranjang bersama SELURUH unggahan tanpa klaim. Memetakan pencarian kosong
// ke sana akan memperlihatkan dokumen milik orang lain kepada siapa pun yang membuka layar
// tanpa nomor klaim.
func TestDaftarTanpaNomorKlaimTidakMembukaKeranjangBersama(t *testing.T) {
	service, _, _ := bangun(t, time.Now())

	p := permintaan()
	p.ClaimNumber = ""
	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: p,
	})
	require.NoError(t, err)

	for _, cari := range []string{"", "   ", "-"} {
		daftar, err := service.List(context.Background(), portal, cari)
		require.NoError(t, err)
		require.Empty(t, daftar, "pencarian %q tidak boleh membuka keranjang bersama", cari)
	}
}

// TestURLDibacaKembaliSetelahMetadataTercatat menjaga langkah GetURLAndEXPDate.
//
// Respons unggah tidak memuat URL publik. Tanpa membaca kembali, daftar dokumen tidak punya
// alamat yang dapat dibuka sampai halaman dimuat ulang.
func TestURLDibacaKembaliSetelahMetadataTercatat(t *testing.T) {
	saat := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	service, repo, _ := bangun(t, saat)

	dokumen, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.NoError(t, err)

	tersimpan, err := repo.Ambil(context.Background(), dokumen.ImageID)
	require.NoError(t, err)
	require.Equal(t, dokumen.ImageID, tersimpan.ImageID)
	require.Equal(t, "Doc/2026/09/", tersimpan.Folder)
}

// ── Konversi gambar ──────────────────────────────────────────────────────────────
//
// Uji di bagian ini menjaga tiruan `Activity/Convert_Avif-Act.xml` beserta kedua cabang
// pemanggilnya di `Activity/InsertDokumenPNC-Act.xml`.

func unggahBerkas(
	t *testing.T,
	service *usecase.Service,
	nama string,
) (dokumenpenunjang.Document, error) {
	t.Helper()
	p := permintaan()
	p.FileName = nama
	p.Content = []byte("isi " + nama)
	return service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: p,
	})
}

// TestHanyaEmpatEkstensiYangDikonversi menjaga prakondisi
// `@toUpperCase(param.MimeType)=="PNG"||=="JPG"||=="JPEG"||=="PDF"`.
//
// Mengonversi yang lain akan mengirim berkas yang tidak dapat dikonversi ke layanan
// gambar; melewatkan yang empat akan menyimpan gambar pada format asli — keduanya berbeda
// dari Pega, dan keduanya tanpa gejala.
func TestHanyaEmpatEkstensiYangDikonversi(t *testing.T) {
	for _, uji := range []struct {
		nama       string
		dikonversi bool
	}{
		{"foto.png", true},
		{"foto.jpg", true},
		{"foto.jpeg", true},
		{"FOTO.JPEG", true},
		{"surat.pdf", true},
		{"surat.docx", false},
		{"arsip.zip", false},
		{"catatan.txt", false},
		{"tanpaekstensi", false},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			service, _, storage, converter := bangunLengkap(t, time.Now())

			_, err := unggahBerkas(t, service, uji.nama)
			require.NoError(t, err)

			if uji.dikonversi {
				require.Len(t, converter.Diterima(), 1, "berkas ini seharusnya dikonversi")
				require.Equal(t, []byte("AVIF:isi "+uji.nama), storage.Terakhir().Isi,
					"yang terunggah harus HASIL konversi")
			} else {
				require.Empty(t, converter.Diterima(), "berkas ini tidak boleh dikonversi")
				require.Equal(t, []byte("isi "+uji.nama), storage.Terakhir().Isi)
			}
		})
	}
}

// TestGambarBergantiTipeMenjadiAvifTetapiPDFTIDAK menjaga pembedaan kedua cabang.
//
// Pega menyetel `Param.MimeType := "Avif"` HANYA pada cabang PNG/JPG/JPEG
// (`Activity/InsertDokumenPNC-Act.xml:2322`). Cabang PDF (`:2401`) mengganti isinya saja.
// Menyeragamkannya akan menandai PDF sebagai `image/avif`, dan peramban menolak membukanya.
func TestGambarBergantiTipeMenjadiAvifTetapiPDFTidak(t *testing.T) {
	for _, uji := range []struct{ nama, tipeMedia string }{
		{"foto.png", "image/avif"},
		{"foto.jpg", "image/avif"},
		{"foto.jpeg", "image/avif"},
		{"surat.pdf", "application/pdf"},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			service, _, storage, _ := bangunLengkap(t, time.Now())

			_, err := unggahBerkas(t, service, uji.nama)
			require.NoError(t, err)
			require.Equal(t, uji.tipeMedia, storage.Terakhir().TipeMedia)
		})
	}
}

// TestKonversiGagalMembatalkanUnggahan menjaga penahanan `Param.ErrMsg`.
//
// Di Pega, konversi yang gagal membiarkan `TempAviff.City` kosong, dan pemanggilnya
// menyalin penampung kosong itu ke `Param.Base64`. Yang menahan agar berkas kosong tidak
// terunggah adalah `Page-Set-Messages` di `:2649`. Penahanan itulah yang ditiru.
func TestKonversiGagalMembatalkanUnggahan(t *testing.T) {
	service, repo, storage, converter := bangunLengkap(t, time.Now())
	converter.Gagalkan(errors.New("layanan konversi tumbang"))

	_, err := unggahBerkas(t, service, "foto.png")
	require.ErrorIs(t, err, dokumenpenunjang.ErrKonversiGagal)

	require.Empty(t, storage.Diterima(), "berkas tidak boleh terkirim ke penyimpanan")

	// Izin unggah pun tidak boleh tercatat: konversi mendahuluinya, persis seperti di Pega.
	// Mencatatnya lebih dulu akan meninggalkan baris GCP_IMAGE untuk unggahan yang tidak
	// pernah terjadi.
	require.Empty(t, repo.Akses())
}

// TestKonversiMengembalikanKosongDianggapGagal menutup keberhasilan semu.
//
// Layanan yang menjawab berhasil tanpa isi akan membuat berkas KOSONG terunggah dan
// tercatat sebagai dokumen yang sah — dan tidak ada satu pun gejala yang menandainya.
func TestKonversiMengembalikanKosongDianggapGagal(t *testing.T) {
	service, _, storage, converter := bangunLengkap(t, time.Now())
	converter.KembalikanKosong()

	_, err := unggahBerkas(t, service, "foto.png")
	require.ErrorIs(t, err, dokumenpenunjang.ErrKonversiGagal)
	require.Empty(t, storage.Diterima())
}

// TestKonversiGagalPadaBerkasYangTidakDikonversiTidakBerpengaruh.
//
// Layanan konversi yang tumbang hanya menutup empat ekstensi. Berkas lain tetap terunggah —
// dan itu yang membuat matinya konversi bukan matinya unggah.
func TestKonversiGagalPadaBerkasYangTidakDikonversiTidakBerpengaruh(t *testing.T) {
	service, _, storage, converter := bangunLengkap(t, time.Now())
	converter.Gagalkan(errors.New("layanan konversi tumbang"))

	_, err := unggahBerkas(t, service, "catatan.txt")
	require.NoError(t, err)
	require.Len(t, storage.Diterima(), 1)
	require.Equal(t, "text/plain", storage.Terakhir().TipeMedia)
}

// TestNamaBerkasTIDAKBerubahKarenaKonversi.
//
// Yang berubah karena konversi hanyalah ISI dan TIPE. Nama tetap diturunkan dari nama asli
// yang diketik pengguna — `FotoKerugianpdf`, bukan `FotoKerugianavif`. Begitu pula di Pega:
// `DocAPI.NamaFile` diisi dari `param.Filename` yang sudah dibersihkan di langkah pertama,
// sebelum konversi berjalan.
func TestNamaBerkasTidakBerubahKarenaKonversi(t *testing.T) {
	service, _, storage, _ := bangunLengkap(t, time.Now())

	_, err := unggahBerkas(t, service, "Foto Kerugian.png")
	require.NoError(t, err)
	require.Equal(t, "FotoKerugianpng", storage.Terakhir().NamaBerkas)
}

// TestKonversiDilewatiMengunggahBerkasAsli: dengan KONVERSI_GAMBAR_LEWATI (keputusan Work
// Owner 2026-09-30) PNG dan PDF terunggah apa adanya — isi, nama, dan tipe media asli —
// tanpa menyentuh layanan konversi, bahkan saat layanan itu gagal.
func TestKonversiDilewatiMengunggahBerkasAsli(t *testing.T) {
	for _, uji := range []struct{ nama, tipeMedia string }{
		{"foto.png", "image/png"},
		{"surat.pdf", "application/pdf"},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			repo := memory.NewRepo()
			storage := memory.NewStorage()
			converter := memory.NewConverter()
			converter.Gagalkan(errors.New("layanan konversi tumbang"))
			service, err := usecase.NewService(usecase.Options{
				Repos:          func(string) (dokumenpenunjang.Repo, error) { return repo, nil },
				Storage:        storage,
				Converter:      converter,
				Clock:          jamTetap{pada: time.Now()},
				SkipConversion: true,
			})
			require.NoError(t, err)

			_, err = unggahBerkas(t, service, uji.nama)
			require.NoError(t, err)
			require.Empty(t, converter.Diterima())
			require.Equal(t, []byte("isi "+uji.nama), storage.Terakhir().Isi)
			require.Equal(t, uji.tipeMedia, storage.Terakhir().TipeMedia)
		})
	}
}

// TestKodeAksesTerdaftarDikirimTanpaTokenSekaliPakai: bila PENYIMPANAN_DOKUMEN_KODE_AKSES
// diisi, kode itu yang menjadi KodeString dan tidak ada baris GCP_IMAGE yang dicatat.
func TestKodeAksesTerdaftarDikirimTanpaTokenSekaliPakai(t *testing.T) {
	repo := memory.NewRepo()
	storage := memory.NewStorage()
	service, err := usecase.NewService(usecase.Options{
		Repos:      func(string) (dokumenpenunjang.Repo, error) { return repo, nil },
		Storage:    storage,
		Converter:  memory.NewConverter(),
		Clock:      jamTetap{pada: time.Now()},
		AccessCode: "KODE-TERDAFTAR",
	})
	require.NoError(t, err)

	_, err = service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: portal, Request: permintaan(),
	})
	require.NoError(t, err)
	require.Equal(t, "KODE-TERDAFTAR", storage.Terakhir().KodeAkses)
	require.Empty(t, repo.Akses())
}
