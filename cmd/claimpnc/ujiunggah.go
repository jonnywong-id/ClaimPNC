package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/logging"
)

// Mode `-uji-unggah`: membuktikan jalur unggah dokumen dari ujung ke ujung.
//
// # Kenapa ia terpisah dari `-periksa`
//
// `-periksa` berjanji TIDAK menulis apa pun. Mode ini menulis: ia benar-benar mengirim satu
// berkas ke layanan penyimpanan, dan mencatat barisnya. Menggabungkan keduanya akan
// membatalkan janji itu bagi setiap orang yang menjalankan `-periksa` di produksi.
//
// # Kenapa ia ada sama sekali
//
// Jalur unggah menempuh TIGA sistem di luar aplikasi ini — konversi gambar, layanan
// penyimpanan, dan basis data metadata lewat DB link. Ketiganya dapat gagal sendiri-sendiri,
// dan dari layar ketiganya terbaca sama: "unggah gagal". Mode ini menyebut yang mana.
//
// Pada 2026-10-14 pertanyaannya bahkan lebih sempit: berkas terkirim ke folder yang benar
// atau tidak. `GENERAL.T_FOLDER_STORAGE` menjawab folder UJI, sementara dokumen produksi ada
// di folder lain — dan layanan menolak folder uji karena kode aksesnya belum didaftarkan.
// Baris pertama keluaran mode ini menjawab persis itu.

// berkasUji adalah PDF terkecil yang masih sah.
//
// Sengaja PDF, bukan teks: PDF menempuh jalur konversi, sehingga satu percobaan ini ikut
// membuktikan layanan konversi — atau membuktikan `KONVERSI_GAMBAR_LEWATI` memang bekerja.
var berkasUji = []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n")

// ujiUnggah menjalankan satu unggahan percobaan dan melaporkan apa yang terjadi.
func ujiUnggah(cfg config.Config, portalAlias string, out io.Writer) error {
	print := func(format string, args ...any) {
		fmt.Fprintf(out, format+"\n", args...)
	}

	print("")
	print("UJI UNGGAH DOKUMEN — satu berkas percobaan dikirim sungguhan")
	print("============================================================")
	print("")

	// Logger dibungkam ke tingkat Error: keluaran mode ini dibaca manusia, dan ratusan baris
	// log perakitan akan menenggelamkan tiga baris yang benar-benar menjawab.
	logger := logging.New(slog.LevelError)

	assembly, err := build(cfg, logger)
	if err != nil {
		return fmt.Errorf("aplikasi tidak dapat dirakit: %w", err)
	}
	defer assembly.close()

	if assembly.dokumenPenunjang == nil {
		print("  [GAGAL] layanan dokumen penunjang tidak ikut dirakit.")
		return errors.New("layanan dokumen penunjang tidak tersedia")
	}

	ctx, batal := context.WithTimeout(context.Background(), 3*time.Minute)
	defer batal()

	laporkanKonfigurasi(ctx, cfg, assembly, portalAlias, print)

	print("")
	print("Mengirim satu berkas percobaan…")
	print("")

	nama := fmt.Sprintf("UJI-MIGRASI-%s.pdf", time.Now().Format("20060102-150405"))
	dokumen, err := assembly.dokumenPenunjang.Upload(ctx, dokumenpenunjangusecase.UploadCommand{
		PortalAlias: portalAlias,
		Request: dokumenpenunjang.UploadRequest{
			FileName: nama,
			Content:  berkasUji,
			By:       "UJI-MIGRASI",
		},
	})
	if err != nil {
		laporkanKegagalan(err, print)
		return err
	}

	print("  [ok]    Berkas TERSIMPAN.")
	print("          Nama berkas : %s", nama)
	print("          IMAGEID     : %s", dokumen.ImageID)
	if strings.TrimSpace(dokumen.URL) != "" {
		print("          Alamat baca : terbit (%d karakter)", len(dokumen.URL))
	} else {
		print("          Alamat baca : KOSONG — berkasnya tersimpan tetapi belum dapat dibuka")
	}
	print("")
	print("  Jalur unggah terbukti bekerja dari ujung ke ujung.")
	print("  Berkas percobaan ini menempel pada klaim %q, bukan pada klaim sungguhan.",
		dokumenpenunjang.TanpaKlaim)
	return nil
}

// laporkanKonfigurasi menyebutkan ke mana berkas akan dikirim, SEBELUM dikirim.
//
// Urutannya disengaja: bila unggahannya gagal, pembaca sudah melihat folder tujuannya di
// layar yang sama — tidak perlu menjalankan kueri terpisah untuk menduga sebabnya.
func laporkanKonfigurasi(
	ctx context.Context,
	cfg config.Config,
	assembly assembly,
	portalAlias string,
	print func(string, ...any),
) {
	print("Tujuan:")
	print("  Portal                : %s", portalAlias)
	print("  Layanan penyimpanan   : %s", alamatAman(cfg.DocumentStorage.BaseURL))
	print("  Layanan konversi      : %s", alamatAman(cfg.DocumentStorage.ConverterURL))

	if cfg.DocumentStorage.SkipConversion {
		print("  Konversi AVIF         : DILEWATI (KONVERSI_GAMBAR_LEWATI)")
	} else {
		print("  Konversi AVIF         : aktif")
	}

	if cfg.DocumentStorage.AccessCode != "" {
		print("  Kode akses            : terisi (%d karakter) — dikirim sebagai KodeString",
			len(cfg.DocumentStorage.AccessCode))
	} else {
		print("  Kode akses            : KOSONG — token sekali pakai dibuat per unggahan")
	}

	// Folder adalah inti persoalannya, jadi ia dilaporkan paling rinci: yang tertulis di
	// tabel, yang dipakai, dan apakah keduanya berbeda.
	dariTabel := folderDariTabel(ctx, assembly, portalAlias)
	penimpa := strings.TrimSpace(cfg.DocumentStorage.FolderName)

	print("")
	print("Folder penyimpanan:")
	print("  T_FOLDER_STORAGE      : %s", dariTabel)

	switch {
	case penimpa == "":
		print("  Dipakai               : %s (dari tabel, seperti Pega)", dariTabel)
	case penimpa == dariTabel:
		print("  Dipakai               : %s (penimpa aktif, nilainya sama)", penimpa)
	default:
		print("  Dipakai               : %s  <-- PENIMPA AKTIF", penimpa)
		print("")
		print("  PERHATIAN: berkas TIDAK mendarat di folder yang dibaca Pega.")
		print("             Kosongkan PENYIMPANAN_DOKUMEN_FOLDER untuk mengembalikannya.")
	}
}

// folderDariTabel membaca nama folder apa adanya dari metadata, tanpa penimpa.
//
// Kegagalannya dilaporkan sebagai teks, bukan dinaikkan sebagai galat: ia keterangan
// pendukung, dan uji unggahnya tetap layak dijalankan meski baris ini tidak terbaca.
func folderDariTabel(ctx context.Context, assembly assembly, portalAlias string) string {
	if assembly.dokumenPenunjangRepos == nil {
		return "(tidak dapat dibaca: metadata tidak dirakit)"
	}
	repo, err := assembly.dokumenPenunjangRepos(portalAlias)
	if err != nil {
		return fmt.Sprintf("(tidak dapat dibaca: %v)", err)
	}
	nama, err := repo.NamaFolderAplikasi(ctx, dokumenpenunjang.NamaAplikasi)
	if err != nil {
		return fmt.Sprintf("(tidak dapat dibaca: %v)", err)
	}
	return nama
}

// laporkanKegagalan menerjemahkan galat menjadi sebab DAN tindakan.
//
// Tanpa ini keluarannya hanya pesan galat mentah, dan pembaca tetap harus menebak pihak mana
// yang harus dihubungi — persis masalah yang mode ini hendak tutup.
func laporkanKegagalan(err error, print func(string, ...any)) {
	print("  [GAGAL] Berkas TIDAK tersimpan.")
	print("          Galat: %v", err)
	print("")

	switch {
	case errors.Is(err, dokumenpenunjang.ErrFolderAplikasiTidakAda):
		print("  Sebab   : folder penyimpanan untuk aplikasi %q tidak ada di",
			dokumenpenunjang.NamaAplikasi)
		print("            GENERAL.T_FOLDER_STORAGE.")
		print("  Tindakan: minta DBA memeriksa baris itu, atau isi PENYIMPANAN_DOKUMEN_FOLDER.")

	case errors.Is(err, dokumenpenunjang.ErrUnggahGagal):
		print("  Sebab   : layanan penyimpanan menolak atau tidak dapat dihubungi.")
		print("            Bila pesannya memuat \"invalid kodestring data\", kode akses untuk")
		print("            folder tujuan BELUM didaftarkan pemilik layanan.")
		print("  Tindakan: tunjukkan keluaran ini ke tim pemilik layanan penyimpanan, dan")
		print("            tanyakan kode akses yang berlaku untuk folder di atas.")

	case errors.Is(err, dokumenpenunjang.ErrKonversiGagal):
		print("  Sebab   : layanan konversi gambar tidak dapat dihubungi.")
		print("  Tindakan: periksa jalur jaringan ke layanan konversi, atau setel")
		print("            KONVERSI_GAMBAR_LEWATI=true untuk melewatinya sementara.")

	case errors.Is(err, dokumenpenunjang.ErrLinkTakTerjangkau):
		print("  Sebab   : basis data metadata (DB link ASMD) tidak dapat dihubungi.")
		print("  Tindakan: laporkan ke DBA.")

	case errors.Is(err, dokumenpenunjang.ErrMetadataGagal):
		print("  Sebab   : berkas SUDAH terkirim ke penyimpanan, catatannya yang gagal.")
		print("  Tindakan: JANGAN ulangi berkali-kali — setiap pengulangan menambah satu")
		print("            berkas di penyimpanan yang tidak tertaut ke mana pun.")

	default:
		print("  Sebab   : tidak dikenali. Keluaran galat di atas adalah jawaban mentahnya.")
	}
}

// alamatAman menjaga hostname produksi tidak tertulis utuh ke layar yang bisa disalin ke
// dokumen (`D-69`). Yang dibutuhkan pembaca hanya "terisi atau tidak, dan yang mana".
func alamatAman(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return "KOSONG"
	}
	tanpaSkema := url
	for _, skema := range []string{"https://", "http://"} {
		tanpaSkema = strings.TrimPrefix(tanpaSkema, skema)
	}
	if host, _, ada := strings.Cut(tanpaSkema, "."); ada {
		return host + ".…"
	}
	return tanpaSkema
}
