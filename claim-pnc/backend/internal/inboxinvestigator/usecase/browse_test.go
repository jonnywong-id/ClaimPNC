package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxinvestigator/repo/memory"
	"claim-pnc/internal/inboxinvestigator/usecase"
)

func mustTime(t *testing.T, text string) time.Time {
	t.Helper()
	moment, err := time.Parse(time.RFC3339, text)
	require.NoError(t, err)
	return moment
}

// serviceWith merakit layanan atas repo memori satu portal.
func serviceWith(t *testing.T, repo inboxinvestigator.Repo) *usecase.Service {
	t.Helper()
	service, _ := serviceAndForms(t, repo)
	return service
}

// serviceAndForms merakit layanan beserta penyimpanan hasil investigasinya.
//
// Dikembalikan berpasangan supaya uji formulir dapat memeriksa apa yang BENAR-BENAR
// tersimpan — termasuk perpindahan klaimnya — bukan hanya apa yang dijawab layanan.
func serviceAndForms(
	t *testing.T,
	repo inboxinvestigator.Repo,
) (*usecase.Service, *memory.InvestigationRepo) {
	t.Helper()

	forms := memory.NewInvestigationRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxinvestigator.Repo, error) {
			if alias != "asm" {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return repo, nil
		},
		InvestigationSelector: func(
			alias string,
		) (inboxinvestigator.InvestigationRepo, error) {
			if alias != "asm" {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return forms, nil
		},
		Clock: func() time.Time { return formAt },
	})
	require.NoError(t, err)
	return service, forms
}

// formAt adalah jam tetap layanan ini.
//
// Modul ini sempat tidak punya jam sama sekali, dan itu benar selama ia hanya membaca.
// Formulir investigasi mengubahnya: tiga nilai lahir dari waktu kini.
var formAt = time.Date(2026, 10, 5, 7, 30, 0, 0, time.UTC)

func TestLayananMenolakDirakitTanpaPemilihRepo(t *testing.T) {
	// Rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestDaftarMengembalikanTanggalSurveiApaAdanya(t *testing.T) {
	// Kolom kesembilan grid bercaption "Lama Masuk Inbox", tetapi isinya TANGGAL SURVEI —
	// terbukti dari pemasangan caption-ke-sel pada Section/InputInvestigator_Section.
	//
	// Uji ini mengunci pembacaan itu. Bila kelak seseorang mengubahnya kembali menjadi
	// durasi yang dihitung, inilah yang gagal lebih dulu.
	survey := mustTime(t, "2026-09-22T01:00:00Z")
	repo := memory.NewRepo(memory.Options{Tasks: []inboxinvestigator.Task{
		{Reference: "ref-1", CaseNumber: "PNC-1", SurveyDate: &survey},
	}})

	page, err := serviceWith(t, repo).List(context.Background(), "asm", "")

	require.NoError(t, err)
	require.Len(t, page.Tasks, 1)
	require.NotNil(t, page.Tasks[0].SurveyDate)
	require.True(t, page.Tasks[0].SurveyDate.Equal(survey))
}

func TestTugasTanpaBarisSurveiTetapTampilDenganTanggalKosong(t *testing.T) {
	// Klaim yang belum disurvei tetap menunggu di antrean — ia tidak boleh hilang dari
	// daftar hanya karena satu kolomnya kosong.
	repo := memory.NewRepo(memory.Options{Tasks: []inboxinvestigator.Task{
		{Reference: "ref-2", CaseNumber: "PNC-2", SurveyDate: nil},
	}})

	page, err := serviceWith(t, repo).List(context.Background(), "asm", "")

	require.NoError(t, err)
	require.Len(t, page.Tasks, 1)
	require.Nil(t, page.Tasks[0].SurveyDate)
}

func TestPortalYangTidakDikenalDitolakSebelumSatuBarisPunDibaca(t *testing.T) {
	// Antrean satu badan hukum bukan antrean badan hukum lain. Portal yang tidak dapat
	// dilayani WAJIB menghasilkan galat, bukan jatuh ke portal utama (`R-20`).
	_, err := serviceWith(t, memory.NewSampleRepo()).
		List(context.Background(), "entitas-lain", "")

	require.Error(t, err)
}

func TestKegagalanPenyimpananDikembalikanSebagaiGalat(t *testing.T) {
	// Kegagalan membaca antrean TIDAK boleh terbaca sebagai antrean kosong: petugas akan
	// menyimpulkan tidak ada pekerjaan, lalu pulang.
	repo := memory.NewSampleRepo()
	repo.SetError(errors.New("koneksi terputus"))

	_, err := serviceWith(t, repo).List(context.Background(), "asm", "")

	require.Error(t, err)
}

func TestPenyaringKataKunciDipangkasSebelumDipakai(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Tasks: []inboxinvestigator.Task{
		{Reference: "ref-1", CaseNumber: "PNC-100241"},
		{Reference: "ref-2", CaseNumber: "PNC-100238"},
	}})

	page, err := serviceWith(t, repo).List(context.Background(), "asm", "  100241  ")

	require.NoError(t, err)
	require.Len(t, page.Tasks, 1)
	require.Equal(t, "PNC-100241", page.Tasks[0].CaseNumber)
}

func TestAntreanKosongBukanGalat(t *testing.T) {
	// Inbox yang bersih adalah keadaan yang DIHARAPKAN, bukan kegagalan.
	page, err := serviceWith(t, memory.NewRepo(memory.Options{})).
		List(context.Background(), "asm", "")

	require.NoError(t, err)
	require.Empty(t, page.Tasks)
	require.False(t, page.Truncated)
}
