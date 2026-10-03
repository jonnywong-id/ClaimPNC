// Package usecase merangkai satu unggahan dokumen penunjang dari awal sampai tercatat.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/dokumenpenunjang"
)

// Service melayani kedua layar Open Protection.
//
// # Kenapa SATU service untuk dua layar, bukan dua
//
// Input Req Protection dan Inbox Accept Open Protection mengunggah hal yang SAMA — dokumen
// penunjang sebuah klaim — dengan aturan yang sama. Dua salinan akan berbeda diam-diam
// begitu salah satunya diperbaiki, dan perbedaannya baru terlihat sebagai berkas yang
// mendarat di folder berbeda.
//
// Modul pemanggilnya tetap tidak saling mengimpor: keduanya memanggil modul INI, bukan satu
// sama lain.
type Service struct {
	repos      dokumenpenunjang.RepoSelector
	storage    dokumenpenunjang.Storage
	converter  dokumenpenunjang.Converter
	clock      dokumenpenunjang.Clock
	skip       bool
	accessCode string

	// lastImageAt adalah waktu ImageID terakhir. GenerateImageID bergantung pada milidetik;
	// dua berkas satu Submit tidak boleh berbagi kunci, jadi waktunya dijaga selalu maju.
	mu          sync.Mutex
	lastImageAt time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	Repos     dokumenpenunjang.RepoSelector
	Storage   dokumenpenunjang.Storage
	Converter dokumenpenunjang.Converter
	Clock     dokumenpenunjang.Clock

	// SkipConversion melewati konversi AVIF (KONVERSI_GAMBAR_LEWATI): berkas diunggah apa
	// adanya. Converter tetap wajib diisi supaya menyalakan kembali konversi cukup dengan
	// mematikan penanda ini.
	SkipConversion bool

	// AccessCode adalah kode akses terdaftar (PENYIMPANAN_DOKUMEN_KODE_AKSES). Diisi: kode
	// itu yang dikirim sebagai KodeString dan tidak ada token sekali pakai yang dicatat —
	// aplikasi lain yang berhasil mengunggah (mis. folder Klaim MBU) tidak punya satu pun
	// baris GCP_IMAGE. Kosong: perilaku `GET_TOKEN_STORAGE`.
	AccessCode string
}

// NewService membentuk Service; ketiga bahannya WAJIB.
//
// Tidak ada nilai bawaan untuk Clock, meski `time.Now` tampak tidak berbahaya: waktu unggah
// menentukan folder tujuan, dan bawaan yang diam-diam terpakai membuat uji folder lulus
// karena kebetulan, bukan karena benar.
func NewService(o Options) (*Service, error) {
	if o.Repos == nil {
		return nil, errors.New("dokumenpenunjang/usecase: Repos wajib diisi")
	}
	if o.Storage == nil {
		return nil, errors.New("dokumenpenunjang/usecase: Storage wajib diisi")
	}
	// Converter WAJIB, meski konversi hanya berlaku bagi empat ekstensi.
	//
	// Nil yang dibiarkan akan membuat unggahan PNG melewati konversi DIAM-DIAM — berhasil,
	// tersimpan, dan berbeda dari Pega tanpa satu pun gejala. Lingkungan yang memang tidak
	// punya layanan konversi memasang penolak bernama, bukan nil.
	if o.Converter == nil {
		return nil, errors.New("dokumenpenunjang/usecase: Converter wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("dokumenpenunjang/usecase: Clock wajib diisi")
	}
	return &Service{
		repos:      o.Repos,
		storage:    o.Storage,
		converter:  o.Converter,
		clock:      o.Clock,
		skip:       o.SkipConversion,
		accessCode: strings.TrimSpace(o.AccessCode),
	}, nil
}

// UploadCommand adalah permintaan unggah dari transport.
type UploadCommand struct {
	PortalAlias string
	Request     dokumenpenunjang.UploadRequest
}

// Upload menjalankan satu unggahan penuh.
//
// # Urutannya mengikuti Pega, dan urutan itu BERMAKNA
//
//	Activity/InsertDokumenPNC-Act.xml
//	  1  Convert_Avif             konversi PNG/JPG/JPEG/PDF
//	  2  GetAppFolder             cari nama folder aplikasi
//	  3  GenerateTokenPNCDokumen  catat izin akses
//	  4  UploadDokumenPNC         kirim berkasnya
//	  5  InsertDataPNCStorage     catat metadatanya
//	  6  GetURLAndEXPDate         baca kembali URL dan masa berlakunya
//
// Langkah 1 mendahului karena ia mengubah ISI dan TIPE berkasnya, dan kegagalannya
// membatalkan seluruh unggahan. Langkah 2 mendahului karena namanya ikut terkirim. Langkah
// 3 mendahului langkah 4 karena ia yang mencatat siapa yang mengunggah. Langkah 5 menyusul
// langkah 4 karena `IMAGEID` diterbitkan sesudah unggah (`GenerateImageID`), dan hanya
// bila layanan menjawab dengan URLImage.
//
// # Langkah 5 dijalankan, dan alasannya bukan sekadar meniru
//
// Respons unggah memuat `ImageID` dan `exp`, tetapi TIDAK memuat `URLPUBLIC`. Tanpa
// membaca kembali, daftar dokumen tidak punya alamat yang dapat dibuka sampai halaman
// dimuat ulang.
func (s *Service) Upload(
	ctx context.Context,
	perintah UploadCommand,
) (dokumenpenunjang.Document, error) {
	permintaan := perintah.Request

	if err := periksa(permintaan); err != nil {
		return dokumenpenunjang.Document{}, err
	}

	repo, err := s.repos(perintah.PortalAlias)
	if err != nil {
		return dokumenpenunjang.Document{}, err
	}

	// KONVERSI LEBIH DULU, sebelum folder dicari dan sebelum izin dicatat.
	//
	// Urutan itu disalin dari Pega — `Call Convert_Avif` ada di langkah ~2177, sedangkan
	// `GetAppFolder` dan `GenerateTokenPNCDokumen` menyusul di ~3003 dan ~3442 — dan ia
	// memang yang benar: konversi yang gagal membatalkan seluruh unggahan, sehingga
	// menjalankannya belakangan akan meninggalkan satu baris izin `GCP_IMAGE` untuk
	// unggahan yang tidak pernah terjadi.
	isi := permintaan.Content
	ekstensi := dokumenpenunjang.EkstensiDari(permintaan.FileName)

	if dokumenpenunjang.PerluKonversi(ekstensi) && !s.skip {
		dikonversi, err := s.converter.Convert(ctx, isi)
		if err != nil {
			return dokumenpenunjang.Document{}, fmt.Errorf("%w: %v",
				dokumenpenunjang.ErrKonversiGagal, err)
		}
		if len(dikonversi) == 0 {
			// Layanan menjawab berhasil tanpa memberi isi. Meneruskannya mengunggah berkas
			// KOSONG yang tercatat sebagai dokumen sah — persis yang terjadi di Pega bila
			// penahanan `Param.ErrMsg` dilewati.
			return dokumenpenunjang.Document{}, fmt.Errorf(
				"%w: layanan konversi tidak mengembalikan isi berkas",
				dokumenpenunjang.ErrKonversiGagal)
		}
		isi = dikonversi
		ekstensi = dokumenpenunjang.EkstensiSetelahKonversi(ekstensi)
	}

	folderAplikasi, err := repo.NamaFolderAplikasi(ctx, dokumenpenunjang.NamaAplikasi)
	if err != nil {
		return dokumenpenunjang.Document{}, err
	}

	pengunggah := strings.TrimSpace(permintaan.By)
	kodeAkses := s.accessCode
	if kodeAkses == "" {
		kodeAkses, err = repo.CatatAksesUnggah(ctx, folderAplikasi, pengunggah)
		if err != nil {
			return dokumenpenunjang.Document{}, err
		}
	}

	saat := s.clock.Now()
	nomorKlaim := dokumenpenunjang.NomorKlaimUntukPenyimpanan(permintaan.ClaimNumber)
	namaBersih := dokumenpenunjang.BersihkanNamaBerkas(permintaan.FileName)
	tipeMedia := dokumenpenunjang.TipeMedia(ekstensi)
	folder := dokumenpenunjang.FolderTanggal(saat)

	hasil, err := s.storage.Upload(ctx, dokumenpenunjang.PerintahUnggah{
		NamaAplikasi: folderAplikasi,
		Pengunggah:   pengunggah,
		KodeAkses:    kodeAkses,
		NomorKlaim:   nomorKlaim,
		Folder:       folder,
		NamaBerkas:   namaBersih,
		TipeMedia:    tipeMedia,
		Isi:          isi,
	})
	if err != nil {
		return dokumenpenunjang.Document{}, fmt.Errorf("%w: %v",
			dokumenpenunjang.ErrUnggahGagal, err)
	}
	// Kuncinya diterbitkan di sini, sesudah unggah — `GenerateImageID` pada Pega
	// (`InsertDokumenPNC` :4216, :4448). Respons layanan tidak dibaca untuk itu.
	imageID := dokumenpenunjang.NewImageID(s.imageTime())

	dokumen := dokumenpenunjang.Document{
		ImageID:      imageID,
		FileName:     namaBersih,
		URL:          strings.TrimSpace(hasil.URL),
		ExpiresAt:    hasil.ExpiresAt,
		Folder:       pilihFolder(hasil.Folder, folder),
		ClaimNumber:  nomorKlaim,
		DocumentType: strings.TrimSpace(permintaan.DocumentType),
		UploadedAt:   &saat,
	}

	if err := repo.Simpan(ctx, dokumen); err != nil {
		// Berkasnya SUDAH di penyimpanan dan tidak dapat ditarik kembali. Galatnya dibungkus
		// ErrMetadataGagal supaya pesan ke pengguna menyatakan itu, alih-alih menyuruhnya
		// mencoba lagi dan meninggalkan satu salinan yatim per percobaan.
		return dokumenpenunjang.Document{}, fmt.Errorf("%w: %v",
			dokumenpenunjang.ErrMetadataGagal, err)
	}

	// Baca kembali: URL publik lahir saat metadata tercatat, bukan saat berkas terkirim.
	// Gagalnya TIDAK membatalkan apa pun — dokumennya sudah sah, hanya alamatnya yang
	// belum terisi, dan itu terisi sendiri saat daftar dimuat.
	if tersimpan, err := repo.Ambil(ctx, dokumen.ImageID); err == nil {
		if tersimpan.URL != "" {
			dokumen.URL = tersimpan.URL
		}
		if tersimpan.ExpiresAt != nil {
			dokumen.ExpiresAt = tersimpan.ExpiresAt
		}
		if tersimpan.Folder != "" {
			dokumen.Folder = tersimpan.Folder
		}
	}

	return dokumen, nil
}

// imageTime memberi waktu penerbit ImageID yang selalu maju minimal satu milidetik.
func (s *Service) imageTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := s.clock.Now().Truncate(time.Millisecond)
	if !at.After(s.lastImageAt) {
		at = s.lastImageAt.Add(time.Millisecond)
	}
	s.lastImageAt = at
	return at
}

// List mengembalikan dokumen sebuah klaim.
//
// Nomor klaim kosong TIDAK dipetakan ke TanpaKlaim di sini: `"-"` adalah keranjang bersama
// seluruh unggahan tanpa klaim, dan menampilkannya kepada seseorang yang kebetulan membuka
// klaim tanpa nomor akan memperlihatkan dokumen milik orang lain.
func (s *Service) List(
	ctx context.Context,
	portalAlias, nomorKlaim string,
) ([]dokumenpenunjang.Document, error) {
	rapi := strings.TrimSpace(nomorKlaim)
	if rapi == "" || rapi == dokumenpenunjang.TanpaKlaim {
		return nil, nil
	}

	repo, err := s.repos(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.PerKlaim(ctx, rapi)
}

// periksa menegakkan aturan yang tidak boleh sampai ke layanan penyimpanan.
func periksa(p dokumenpenunjang.UploadRequest) error {
	if strings.TrimSpace(p.By) == "" {
		return dokumenpenunjang.ErrPengunggahKosong
	}
	if len(p.Content) == 0 {
		return dokumenpenunjang.ErrBerkasKosong
	}
	if len(p.Content) > dokumenpenunjang.BatasUkuranBerkas {
		return dokumenpenunjang.ErrBerkasTerlaluBesar
	}
	if dokumenpenunjang.BersihkanNamaBerkas(p.FileName) == "" {
		return dokumenpenunjang.ErrNamaBerkasKosong
	}
	return nil
}

// pilihFolder mendahulukan folder yang dilaporkan layanan.
//
// Layanan boleh menaruh berkas di tempat lain dari yang diminta; yang benar adalah yang
// dilaporkannya. Menyimpan folder yang KITA minta akan membuat metadata menunjuk tempat
// yang berkasnya tidak ada di sana.
func pilihFolder(dariLayanan, diminta string) string {
	if rapi := strings.TrimSpace(dariLayanan); rapi != "" {
		return rapi
	}
	return diminta
}
