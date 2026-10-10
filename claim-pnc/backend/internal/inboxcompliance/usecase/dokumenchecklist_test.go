package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
)

// travelStore adalah antrean Compliance berisi satu klaim TRAVEL.
func travelStore() *memory.Store {
	return memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now,
		Item: inboxcompliance.WorkItem{
			CaseID:     "PNC-1016",
			Reference:  claimKey,
			GroupPanel: inboxcompliance.GroupPanelTravel,
		},
	})
}

func checklistContoh() []inboxcompliance.DocumentChecklistItem {
	minimal := 1
	return []inboxcompliance.DocumentChecklistItem{
		{
			CategoryID:     "10064",
			CategoryName:   "Akte/Surat Keterangan Kematian (Copy)",
			MandatoryLabel: "Ya",
			MinUpload:      &minimal,
			UploadedCount:  0,
		},
		{
			CategoryID:   "10065",
			CategoryName: "Foto Penguburan (jika ada)",
			// Sengaja KOSONG — itu keadaan yang benar-benar terlihat di layar Pega.
			MandatoryLabel: "",
			MinUpload:      &minimal,
			UploadedCount:  2,
		},
	}
}

// Tab Dokumen HANYA dibaca untuk lini Travel.
func TestDaftarDokumenDibacaUntukTravel(t *testing.T) {
	t.Parallel()

	store := travelStore()
	store.SeedDocumentChecklist(claimKey, checklistContoh()...)
	service := newService(t, store)

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.NoError(t, opened.DocumentChecklistError)
	require.Len(t, opened.DocumentChecklist, 2)

	require.Equal(t, "Akte/Surat Keterangan Kematian (Copy)",
		opened.DocumentChecklist[0].CategoryName)
	require.Equal(t, "Ya", opened.DocumentChecklist[0].MandatoryLabel)
	require.Equal(t, 0, opened.DocumentChecklist[0].UploadedCount)
	require.Equal(t, 2, opened.DocumentChecklist[1].UploadedCount)
}

/*
Lini SELAIN Travel tidak membaca daftar periksa sama sekali.

Bukan sekadar "hasilnya kosong": kueri itu memang tidak boleh berjalan. Tabnya tidak
digambar di lini lain (`pyContainerVisibleWhen = IsTravel`), sehingga membacanya berarti
satu kueri setiap kali form dibuka yang hasilnya tidak pernah dipakai.

Diuji lewat store yang SUDAH disemai: kalau kuerinya tetap berjalan, senarainya akan
terisi — dan uji ini merah.
*/
func TestDaftarDokumenTidakDibacaDiLuarTravel(t *testing.T) {
	t.Parallel()

	for _, lini := range []string{
		inboxcompliance.GroupPanelPersonalAccident,
		"003",
		"004",
		"006",
		"009",
		"",
	} {
		t.Run("lini "+lini, func(t *testing.T) {
			t.Parallel()

			store := lineStoreWithPIC(lini)
			store.SeedDocumentChecklist(claimKey, checklistContoh()...)
			service := newService(t, store)

			opened, err := service.OpenChecker(context.Background(), portal, claimKey)
			require.NoError(t, err)
			require.Empty(t, opened.DocumentChecklist,
				"daftar periksa tidak boleh dibaca di luar lini Travel")
		})
	}
}

// Klaim Travel yang masternya belum diisi membuka form dengan tabel kosong — bukan galat.
func TestDaftarDokumenKosongBukanGalat(t *testing.T) {
	t.Parallel()

	service := newService(t, travelStore())

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.NoError(t, opened.DocumentChecklistError)
	require.Empty(t, opened.DocumentChecklist)
}

/*
Gagal membaca daftar periksa TIDAK menutup form, tetapi juga tidak diam.

Keduanya penting bersama. Menutup form akan menghentikan pekerjaan yang masih dapat
berjalan — keputusan Compliance tidak bergantung pada daftar ini. Tetapi gagal diam-diam
lebih buruk daripada form yang tertutup: tabel kosong terbaca sebagai "tidak ada dokumen
yang wajib", dan petugas dapat meneruskan klaim yang dokumennya belum lengkap.
*/
func TestDaftarDokumenGagalDibacaTidakMenutupForm(t *testing.T) {
	t.Parallel()

	// Service dirakit langsung, bukan lewat newService: helper itu menerima
	// `*memory.Store`, sedangkan di sini repo-nya dibungkus supaya satu pembacaan saja
	// yang gagal.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxcompliance.Repo, error) {
			return &repoChecklistRusak{Repo: travelStore()}, nil
		},
		Clock: fixedClock{at: now},
	})
	require.NoError(t, err)

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err, "form harus tetap terbuka")
	require.Error(t, opened.DocumentChecklistError, "galatnya harus sampai ke layar")
	require.Empty(t, opened.DocumentChecklist)

	// Dan sisa formnya tetap utuh.
	require.Equal(t, "PNC-1016", opened.Case.Claim.CaseID)
}

// repoChecklistRusak menggagalkan HANYA pembacaan daftar periksa.
type repoChecklistRusak struct {
	inboxcompliance.Repo
}

func (r *repoChecklistRusak) FindDocumentChecklist(
	context.Context, string,
) ([]inboxcompliance.DocumentChecklistItem, error) {
	return nil, errors.New("LST_TYPE_DOC_BUSINESS tidak terbaca")
}
