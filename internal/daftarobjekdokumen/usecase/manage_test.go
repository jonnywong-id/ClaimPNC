package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/repo/memory"
	"claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/portal"
)

// newService merakit layanan atas dua portal: ASM berisi contoh, ASI kosong.
//
// ASI yang kosong itu yang membuktikan pemisahan antarentitas benar-benar berlaku, bukan
// hanya dinyatakan.
func newService(t *testing.T) (*usecase.Service, *memory.Repo, *memory.BusinessRepo) {
	t.Helper()

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()
	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (daftarobjekdokumen.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		BusinessSelector: func(alias string) (daftarobjekdokumen.BusinessRepo, error) {
			switch alias {
			case "ASM", "ASI":
				return business, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)
	return service, asm, business
}

// Bahan yang tidak lengkap ditolak saat perakitan, bukan saat permintaan pertama datang:
// rakitan setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func TestNewServiceMenolakBahanTidakLengkap(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (daftarobjekdokumen.Repo, error) { return nil, nil },
	})
	require.ErrorContains(t, err, "BusinessSelector wajib diisi")
}

// Portal yang tidak dikenal DITOLAK, bukan diam-diam dilayani portal utama. Itu jalur
// kegagalan `R-20` yang paling mudah terjadi tanpa disadari.
func TestPortalTidakDikenalDitolak(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.List(context.Background(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotReady)

	require.ErrorIs(t, service.EnsurePortalReady("TIDAKADA"), portal.ErrNotReady)
	require.NoError(t, service.EnsurePortalReady("ASM"))
}

// Dua entitas TIDAK berbagi data, meski dilayani satu aplikasi.
func TestEntitasTerpisah(t *testing.T) {
	service, _, _ := newService(t)

	asm, err := service.List(context.Background(), "ASM")
	require.NoError(t, err)
	require.NotEmpty(t, asm)

	asi, err := service.List(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, asi)
}

// Daftar sengaja TIDAK membawa pemetaan bisnis; hanya pengambilan satu baris yang
// membawanya.
//
// Bila ini berubah tanpa disadari, layar akan menampilkan bisnis pada grid yang tidak punya
// kolomnya — dan yang lebih buruk, satu kueri tambahan berjalan untuk setiap baris tanpa ada
// yang melihat hasilnya.
func TestDaftarTidakMembawaPemetaanBisnis(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), "ASM")
	require.NoError(t, err)
	for _, row := range list {
		require.Empty(t, row.Businesses, "baris %q tidak boleh membawa pemetaan bisnis pada daftar", row.ID)
	}

	one, err := service.Get(context.Background(), "ASM", list[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, one.Businesses)
}

// Nama bisnis yang cocok dengan master diselesaikan menjadi ID, dan EJAANNYA diambil dari
// master — bukan dari yang diketik pengguna.
//
// Dengan begitu "aneka" yang diketik huruf kecil tersimpan sebagai "ANEKA", dan daftar di
// layar tidak menampilkan satu bisnis dalam dua ejaan.
func TestNamaBisnisDiselesaikanKeMaster(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description:   "Surat Kuasa",
		BusinessNames: []string{"  aneka  "},
	})
	require.NoError(t, err)
	require.Len(t, saved.Businesses, 1)
	require.Equal(t, "10013", saved.Businesses[0].ID)
	require.Equal(t, "ANEKA", saved.Businesses[0].Name)
}

// Nama bisnis DI LUAR master DITOLAK, dan pesannya menyebut nama mana.
//
// Ini berbeda dari modul Master COL Simas Online, dan perbedaannya dipaksa penyimpanan:
// pemetaan di sini hanya menyimpan ID — di dalam dokumen JSON baris induknya — sehingga
// nama tanpa ID tidak punya tempat sama sekali.
//
// Menerimanya diam-diam adalah perilaku terburuk yang mungkin: pengguna menekan Simpan,
// melihat "berhasil", lalu menemukan barisnya hilang saat form dibuka lagi.
func TestNamaBisnisDiLuarMasterDitolak(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description:   "Nota Dinas",
		BusinessNames: []string{"BISNIS YANG TIDAK ADA"},
	})

	var validationError *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, daftarobjekdokumen.FieldBusiness, validationError.Violation[0].Field)
	require.Contains(t, validationError.Violation[0].Message, "BISNIS YANG TIDAK ADA")
}

// Master bisnis yang GAGAL dibaca MENGGAGALKAN penyimpanan.
//
// Juga berbeda dari modul Master COL Simas Online, dan dengan alasan yang sama: di sana
// master hanya MELENGKAPI ID, sehingga gagal membacanya cukup kehilangan ID. Di sini master
// MENENTUKAN apa yang disimpan — tanpa dapat membacanya, satu-satunya yang dapat ditulis
// adalah pemetaan kosong, yang berarti menghapus seluruh pemetaan yang sudah ada tanpa ada
// yang memintanya.
func TestMasterBisnisGagalMenggagalkanPenyimpanan(t *testing.T) {
	service, asm, business := newService(t)
	business.SetError(errors.New("koneksi putus"))

	before, err := asm.Get(context.Background(), "100002")
	require.NoError(t, err)
	require.Len(t, before.Businesses, 2)

	_, err = service.Update(context.Background(), "ASM", "100002", daftarobjekdokumen.Input{
		Description:   "Polis Asli",
		BusinessNames: []string{"ANEKA"},
	})
	require.Error(t, err)

	after, err := asm.Get(context.Background(), "100002")
	require.NoError(t, err)
	require.Len(t, after.Businesses, 2, "pemetaan yang sudah ada tidak boleh ikut hilang")
}

// Objek dokumen TANPA bisnis tetap dapat disimpan, dan master tidak perlu dibaca sama
// sekali — sehingga master yang sedang gagal pun tidak menghalanginya.
func TestTanpaBisnisTidakMembacaMaster(t *testing.T) {
	service, _, business := newService(t)
	business.SetError(errors.New("koneksi putus"))

	saved, err := service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description: "Kwitansi",
	})
	require.NoError(t, err)
	require.Empty(t, saved.Businesses)
}

// ID diterbitkan penyimpanan, bukan diterima dari pemanggil, dan bentuknya kode situs
// ditambah LIMA digit — mengikuti `POOLDATA.PEGA_LST_DOC_OBJ`.
func TestIDDiterbitkanPenyimpanan(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description: "Berita Acara",
	})
	require.NoError(t, err)
	require.Len(t, saved.ID, 6, "kode situs satu digit ditambah lima digit nomor urut")
	require.Equal(t, "100005", saved.ID, "melanjutkan deret contoh yang berakhir di 100004")
}

// Penyuntingan MENGGANTI seluruh pemetaan bisnis, bukan menggabungkannya.
//
// Layar mengirim keadaan akhir grid apa adanya, sehingga baris yang dicabut pengguna memang
// harus hilang. Menggabungkannya akan membuat baris yang baru saja dicabut muncul kembali
// saat form dibuka lagi — dan pengguna akan mengira pencabutannya gagal.
func TestPenyuntinganMenggantiSeluruhPemetaanBisnis(t *testing.T) {
	service, _, _ := newService(t)

	// 100002 punya DUA bisnis pada data contoh.
	before, err := service.Get(context.Background(), "ASM", "100002")
	require.NoError(t, err)
	require.Len(t, before.Businesses, 2)

	after, err := service.Update(context.Background(), "ASM", "100002", daftarobjekdokumen.Input{
		Description:   before.Description,
		BusinessNames: []string{"TRAVEL"},
	})
	require.NoError(t, err)
	require.Len(t, after.Businesses, 1)
	require.Equal(t, "TRAVEL", after.Businesses[0].Name)
}

// ID tidak pernah ikut berubah saat penyuntingan.
//
// Ia dirujuk LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan, sehingga
// mengubahnya akan memutus setiap baris yang bernaung di bawahnya.
func TestPenyuntinganTidakMengubahID(t *testing.T) {
	service, _, _ := newService(t)

	after, err := service.Update(context.Background(), "ASM", "100001", daftarobjekdokumen.Input{
		Description: "Nama Baru",
	})
	require.NoError(t, err)
	require.Equal(t, "100001", after.ID)
	require.Equal(t, "Nama Baru", after.Description)
}

// Baris yang tidak ada menghasilkan ErrNotFound, bukan galat penyimpanan mentah —
// transport bergantung padanya untuk menjawab 404.
func TestBarisTidakAda(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Get(context.Background(), "ASM", "999999")
	require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)

	_, err = service.Update(context.Background(), "ASM", "999999", daftarobjekdokumen.Input{})
	require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
}

// Isian yang melanggar aturan ditolak SEBELUM menyentuh penyimpanan.
func TestIsianTidakSahDitolakSebelumDisimpan(t *testing.T) {
	service, asm, _ := newService(t)

	before, err := asm.List(context.Background())
	require.NoError(t, err)

	terlaluPanjang := make([]byte, daftarobjekdokumen.MaxDescriptionLength+1)
	for i := range terlaluPanjang {
		terlaluPanjang[i] = 'A'
	}

	_, err = service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description: string(terlaluPanjang),
	})
	var validationError *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validationError)

	after, err := asm.List(context.Background())
	require.NoError(t, err)
	require.Len(t, after, len(before), "tidak ada baris yang tersimpan")
}

// Daftar bisnis dapat dibaca lewat layanan ini, meski rutenya tidak didaftarkan modul ini.
// Layar memakainya lewat rute milik Master COL Simas Online; method ini tetap ada karena
// layanan inilah yang tahu cara membaca master milik portal aktif.
func TestListBusiness(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.ListBusiness(context.Background(), "ASM")
	require.NoError(t, err)
	require.NotEmpty(t, list)

	_, err = service.ListBusiness(context.Background(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotReady)
}
