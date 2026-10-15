package usecase_test

import (
	"context"
	"errors"
	"testing"

	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/detailpenyebab/repo/memory"
	"claim-pnc/internal/detailpenyebab/usecase"
)

const portalAlias = "utama"

// newService merakit layanan atas penyimpanan memori berisi baris contoh.
func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (detailpenyebab.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
	})
	if err != nil {
		t.Fatalf("merakit layanan: %v", err)
	}
	return service, repo
}

func TestPortalYangTidakDikenalDitolakAlihAlihDilayaniPortalUtama(t *testing.T) {
	// Inilah yang mencegah R-20: menulis data satu badan hukum ke basis data badan hukum
	// lain tanpa satu pun pesan galat.
	service, _ := newService(t)

	if _, err := service.List(context.Background(), "entitas-lain", detailpenyebab.Filter{}); err == nil {
		t.Fatal("portal yang tidak dikenal seharusnya ditolak, bukan dilayani")
	}
}

func TestDaftarTidakMenyaringBarisYangTidakAktif(t *testing.T) {
	// Rule pengisi grid Pega hilang dari export (R-16), dan bacaan yang dipilih adalah
	// TIDAK menyaring — alasannya pada banner paket detailpenyebab.
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias, detailpenyebab.Filter{})
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}

	var inactive int
	for _, one := range list {
		if !detailpenyebab.IsActive(one.Active) {
			inactive++
		}
	}
	if inactive == 0 {
		t.Fatal("daftar seharusnya memuat baris tidak aktif, tetapi seluruhnya aktif")
	}
}

func TestDaftarUrutMenurutDeskripsiKerugian(t *testing.T) {
	// Meniru `order by DESCRIPTION` pada QueryGetAllDataCauseOfLoss-SQL.xml:100.
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias, detailpenyebab.Filter{})
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	for index := 1; index < len(list); index++ {
		if list[index-1].Description > list[index].Description {
			t.Fatalf("daftar tidak urut: %q mendahului %q",
				list[index-1].Description, list[index].Description)
		}
	}
}

func TestDaftarTidakMemuatLiniBisnisKarenaGridTidakMenampilkannya(t *testing.T) {
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias, detailpenyebab.Filter{})
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	for _, one := range list {
		if len(one.Business) != 0 {
			t.Fatalf("daftar seharusnya tidak memuat lini bisnis, tetapi %q memuat %d",
				one.ID, len(one.Business))
		}
	}
}

func TestSatuBarisDimuatLengkapDenganLiniBisnisnya(t *testing.T) {
	service, _ := newService(t)

	found, err := service.Get(context.Background(), portalAlias, "990004")
	if err != nil {
		t.Fatalf("membaca satu baris: %v", err)
	}
	if len(found.Business) != 3 {
		t.Fatalf("baris 990004 seharusnya punya tiga lini bisnis, tetapi %d", len(found.Business))
	}
}

func TestSebutanIndukDiturunkanSaatBarisDibaca(t *testing.T) {
	service, _ := newService(t)

	found, err := service.Get(context.Background(), portalAlias, "990001")
	if err != nil {
		t.Fatalf("membaca satu baris: %v", err)
	}
	if found.MasterLabel != "Kebakaran" {
		t.Fatalf("sebutan induk seharusnya Kebakaran, tetapi %q", found.MasterLabel)
	}
}

func TestBarisYatimTetapTerbacaDenganSebutanIndukKosong(t *testing.T) {
	// Tidak ada foreign key yang diketahui (R-08), sehingga baris yatim mungkin ada — dan
	// menyembunyikannya akan membuatnya hilang dari layar tanpa satu pun tanda.
	service, _ := newService(t)

	found, err := service.Get(context.Background(), portalAlias, "990008")
	if err != nil {
		t.Fatalf("baris yatim seharusnya tetap terbaca, tetapi gagal: %v", err)
	}
	if found.MasterLabel != "" {
		t.Fatalf("sebutan induk baris yatim seharusnya kosong, tetapi %q", found.MasterLabel)
	}
}

func TestBarisBaruMenerimaIDBerpolaKodeSitusDanEmpatDigit(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, detailpenyebab.Input{
		Description: "Tanah longsor",
		Active:      detailpenyebab.ActiveYes,
	}, usecase.Actor{Login: "penguji"}, nil)
	if err != nil {
		t.Fatalf("menambah baris: %v", err)
	}

	// Kode situs tiruan "99" ditambah empat digit — lihat memory.siteCode.
	if len(saved.ID) < 6 {
		t.Fatalf("ID seharusnya sekurangnya enam karakter, tetapi %q", saved.ID)
	}
	if saved.ID[:2] != "99" {
		t.Fatalf("ID seharusnya diawali kode situs, tetapi %q", saved.ID)
	}
}

func TestBarisBaruTidakMenabrakIDContoh(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, detailpenyebab.Input{
		Description: "Tanah longsor",
	}, usecase.Actor{Login: "penguji"}, nil)
	if err != nil {
		t.Fatalf("menambah baris: %v", err)
	}
	// Baris contoh terbesar 990008; yang baru harus melanjutkannya.
	if saved.ID != "990009" {
		t.Fatalf("ID baris baru seharusnya melanjutkan deret contoh (990009), tetapi %q", saved.ID)
	}
}

func TestIDWarisanTidakHilangSaatLayarTidakMengirimkannya(t *testing.T) {
	// OLD_D_COL_ID tidak digambar di form mana pun, tetapi dimuat dan disimpan ulang oleh
	// sistem lama. Klien yang tidak mengirimnya tidak boleh menghapusnya — lihat
	// Service.Save.
	service, _ := newService(t)

	saved, err := service.Save(context.Background(), portalAlias, "990001", detailpenyebab.Input{
		MasterID:    "9001",
		Description: "Kebakaran akibat hubungan arus pendek",
		LossCode:    "FIRE-01",
		Active:      detailpenyebab.ActiveYes,
		// IDLama sengaja TIDAK diisi.
	}, usecase.Actor{Login: "penguji"}, nil)
	if err != nil {
		t.Fatalf("menyimpan: %v", err)
	}

	if saved.LegacyID != "COL-0001" {
		t.Fatalf("ID warisan seharusnya dipertahankan, tetapi menjadi %q", saved.LegacyID)
	}
}

func TestIDWarisanYangDikirimLayarTetapDihormati(t *testing.T) {
	// Yang dicegah hanyalah penghapusan yang tidak disengaja; koreksi tetap mungkin.
	service, _ := newService(t)

	saved, err := service.Save(context.Background(), portalAlias, "990001", detailpenyebab.Input{
		LegacyID:    "COL-9999",
		Description: "Kebakaran akibat hubungan arus pendek",
	}, usecase.Actor{Login: "penguji"}, nil)
	if err != nil {
		t.Fatalf("menyimpan: %v", err)
	}

	if saved.LegacyID != "COL-9999" {
		t.Fatalf("ID warisan yang dikirim seharusnya dipakai, tetapi menjadi %q", saved.LegacyID)
	}
}

func TestMenyimpanBarisYangSudahTidakAdaDitolakSebagaiTidakDitemukan(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Save(context.Background(), portalAlias, "tidak-ada", detailpenyebab.Input{
		Description: "apa saja",
	}, usecase.Actor{Login: "penguji"}, nil)

	if !errors.Is(err, detailpenyebab.ErrNotFound) {
		t.Fatalf("galat seharusnya ErrNotFound, tetapi %v", err)
	}
}

func TestPenyaringLiniBisnisMempersempitDaftar(t *testing.T) {
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias,
		detailpenyebab.Filter{BusinessID: "004"})
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("penyaring lini bisnis 004 seharusnya menemukan sesuatu")
	}
	for _, one := range list {
		if one.ID != "990004" && one.ID != "990006" {
			t.Fatalf("baris %q seharusnya tidak lolos penyaring lini bisnis 004", one.ID)
		}
	}
}

func TestPenyaringIndukMempersempitDaftar(t *testing.T) {
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias,
		detailpenyebab.Filter{MasterID: "9001"})
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("induk 9001 seharusnya punya dua detail, tetapi %d", len(list))
	}
}

func TestPencarianMenjangkauKolomYangTerlihatDiGrid(t *testing.T) {
	// Pencarian yang tidak menjangkau kolom yang terlihat akan terbaca sebagai cacat.
	service, _ := newService(t)

	for _, keyword := range []string{"petir", "FIRE-02", "990002"} {
		list, err := service.List(context.Background(), portalAlias,
			detailpenyebab.Filter{Keyword: keyword})
		if err != nil {
			t.Fatalf("mencari %q: %v", keyword, err)
		}
		if len(list) != 1 || list[0].ID != "990002" {
			t.Fatalf("kata kunci %q seharusnya menemukan tepat baris 990002, tetapi %d baris",
				keyword, len(list))
		}
	}
}

func TestKataKunciPencarianTerlaluPendekMenghasilkanDaftarKosongBukanGalat(t *testing.T) {
	// Autocomplete memanggilnya pada setiap ketukan; menjawab galat pada huruf pertama akan
	// menampilkan pesan merah di bawah isian yang sedang diketik.
	service, _ := newService(t)

	list, err := service.SearchBusiness(context.Background(), portalAlias, "A")
	if err != nil {
		t.Fatalf("kata kunci pendek seharusnya tidak menghasilkan galat, tetapi %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("kata kunci pendek seharusnya menghasilkan daftar kosong, tetapi %d", len(list))
	}
}

func TestPencarianIndukMenemukanMenurutSebutannya(t *testing.T) {
	service, _ := newService(t)

	list, err := service.SearchMaster(context.Background(), portalAlias, "Keba")
	if err != nil {
		t.Fatalf("mencari induk: %v", err)
	}
	if len(list) != 1 || list[0].ID != "9001" {
		t.Fatalf("pencarian induk seharusnya menemukan 9001, tetapi %+v", list)
	}
}
