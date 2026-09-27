package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/daftartipedokumen/repo/memory"
	"claim-pnc/internal/daftartipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"
)

// savedAt adalah jam tetap yang dipakai seluruh uji di berkas ini.
//
// Tetap, bukan `time.Now()`, supaya TGL_EDIT yang tersimpan dapat dibandingkan — itulah
// gunanya Clock menjadi seam dan bukan panggilan langsung di dalam repo.
var savedAt = time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)

// recordingRepo membungkus repo memori dan mencatat Editor yang diterimanya.
//
// Adapter memori sengaja TIDAK menyimpan jejak simpan — kolom itu tidak pernah dibaca
// kembali modul ini. Tetapi jejaknya tetap harus BENAR-BENAR SAMPAI ke repo, karena
// adapter SQL menuliskannya ke basis data. Pembungkus inilah yang membuktikannya.
type recordingRepo struct {
	*memory.Repo
	lastEditor daftartipedokumen.Editor
}

func (r *recordingRepo) InsertNew(
	ctx context.Context,
	input daftartipedokumen.Input,
	by daftartipedokumen.Editor,
) (daftartipedokumen.DocumentType, error) {
	r.lastEditor = by
	return r.Repo.InsertNew(ctx, input, by)
}

func (r *recordingRepo) Update(
	ctx context.Context,
	doc daftartipedokumen.DocumentType,
	by daftartipedokumen.Editor,
) error {
	r.lastEditor = by
	return r.Repo.Update(ctx, doc, by)
}

func newService(t *testing.T) (*usecase.Service, *recordingRepo) {
	t.Helper()

	repo := &recordingRepo{Repo: memory.NewRepo(memory.SampleList()...)}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (daftartipedokumen.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		Clock: clock.FixedAt(savedAt),
	})
	require.NoError(t, err)
	return service, repo
}

// Bahan yang tidak lengkap ditolak saat perakitan, bukan saat permintaan pertama datang.
func TestIncompleteOptionsAreRejectedAtAssemblyTime(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(savedAt)})
	require.Error(t, err, "RepoSelector wajib")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (daftartipedokumen.Repo, error) { return nil, nil },
	})
	require.Error(t, err, "Clock wajib")
}

// Portal yang belum siap menghasilkan galat, TIDAK dialihkan ke portal utama.
//
// Inilah jalur kegagalan `R-20`: mengembalikan repo portal utama sebagai jalan pintas
// berarti menulis data satu badan hukum ke basis data badan hukum lain tanpa satu pun pesan
// galat.
func TestAnUnavailablePortalIsNeverSilentlyReplacedByTheMainOne(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), "ASI")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(context.Background(), "ASI",
		daftartipedokumen.Input{Type: "Apa saja"}, "SOMEUSER")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// ID diterbitkan penyimpanan, tidak pernah datang dari pemanggil.
func TestNewRowGetsItsIDFromStorage(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), "ASM",
		daftartipedokumen.Input{Type: "Dokumen Investigasi", ProcessStatus: "Investigasi"}, "SOMEUSER")
	require.NoError(t, err)

	// Enam baris contoh berakhir di 10006, jadi yang berikutnya 10007.
	require.Equal(t, "10007", saved.ID)
	require.Equal(t, "Dokumen Investigasi", saved.Type)
	require.Equal(t, "Investigasi", saved.ProcessStatus)
}

// Jejak simpan disusun dari identitas pemanggil dan jam sistem, lalu benar-benar sampai ke
// repo — meniru `CNMInsertListDocumentType_act` yang menetapkan USER_EDIT dan TGL_EDIT
// sebelum menyimpan.
func TestSaveTrailReachesTheRepository(t *testing.T) {
	service, repo := newService(t)

	_, err := service.Create(context.Background(), "ASM",
		daftartipedokumen.Input{Type: "Dokumen Investigasi"}, "  SOMEUSER  ")
	require.NoError(t, err)

	require.Equal(t, "SOMEUSER", repo.lastEditor.Identity, "spasi tepi dipangkas")
	require.Equal(t, savedAt, repo.lastEditor.At, "waktunya datang dari seam jam, bukan time.Now()")
}

// Identitas yang kosong TIDAK menghentikan penyimpanan.
//
// Ia hanya mengisi kolom jejak, bukan menentukan apa yang tersimpan. Menolak penyimpanan
// karenanya akan membuat petugas kehilangan isian yang sudah diketik demi kolom yang tidak
// dilihat siapa pun di layar.
func TestSavingStillWorksWhenTheCallerIdentityIsUnknown(t *testing.T) {
	service, repo := newService(t)

	saved, err := service.Create(context.Background(), "ASM",
		daftartipedokumen.Input{Type: "Dokumen Tanpa Jejak"}, "")
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)
	require.Empty(t, repo.lastEditor.Identity)
}

// Isian kosong diterima — layar ini meniru Pega apa adanya (Work Owner 2026-09-21).
func TestEmptyInputIsAcceptedBecauseThisScreenHasNoValidation(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), "ASM", daftartipedokumen.Input{}, "SOMEUSER")
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID, "barisnya tetap terbit meski kedua isiannya kosong")
	require.Empty(t, saved.Type)
}

// Nama ganda diterima. Tabelnya tidak punya indeks unik, dan procedure lama tidak memeriksa
// apa pun — memeriksanya di sini akan menolak baris yang menurut layar lama sah.
func TestDuplicateNamesAreAcceptedLikeInPega(t *testing.T) {
	service, _ := newService(t)

	first, err := service.Create(context.Background(), "ASM",
		daftartipedokumen.Input{Type: "Dokumen Survey"}, "SOMEUSER")
	require.NoError(t, err)

	second, err := service.Create(context.Background(), "ASM",
		daftartipedokumen.Input{Type: "Dokumen Survey"}, "SOMEUSER")
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID, "keduanya baris berbeda dengan nama yang sama")
}

// Menyimpan hanya mengubah isian; ID tetap.
func TestUpdateKeepsTheRowKey(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Update(context.Background(), "ASM", "10002",
		daftartipedokumen.Input{Type: "Dokumen Survey Lapangan", ProcessStatus: "Survey"}, "SOMEUSER")
	require.NoError(t, err)

	require.Equal(t, "10002", saved.ID)
	require.Equal(t, "Dokumen Survey Lapangan", saved.Type)

	reread, err := service.Get(context.Background(), "ASM", "10002")
	require.NoError(t, err)
	require.Equal(t, saved, reread, "yang dikembalikan sama dengan yang benar-benar tersimpan")
}

// Mengubah baris yang sudah tidak ada dijawab ErrNotFound, bukan diam-diam berhasil.
func TestUpdatingARowThatIsNoLongerThereIsReportedAsNotFound(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(context.Background(), "ASM", "19999",
		daftartipedokumen.Input{Type: "Apa saja"}, "SOMEUSER")
	require.ErrorIs(t, err, daftartipedokumen.ErrNotFound)
}

// Kegagalan penyimpanan diteruskan apa adanya, tidak ditelan.
func TestStorageFailureIsPropagated(t *testing.T) {
	service, repo := newService(t)
	boom := errors.New("koneksi terputus")
	repo.SetError(boom)

	_, err := service.List(context.Background(), "ASM")
	require.ErrorIs(t, err, boom)
}
