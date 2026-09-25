package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxreceivetka/notification"
	"claim-pnc/internal/inboxreceivetka/repo/memory"
	"claim-pnc/internal/inboxreceivetka/usecase"
	"claim-pnc/internal/portal"
)

const portalAlias = "ASM"

func completedAt() time.Time {
	return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
}

// build merakit layanan dengan repo contoh dan perekam pemberitahuan.
//
// notifier nil berarti pemberitahuan memang tidak dipasang — keadaan yang wajar di
// lingkungan pengembangan, dan yang harus dapat dibedakan dari gagal kirim.
func build(t *testing.T, notifier inboxreceivetka.Notifier) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxreceivetka.Repo, error) {
			if alias != portalAlias {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		Notifier: notifier,
	})
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceRejectsMissingRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err, "rakitan setengah jadi harus gagal saat start, bukan saat dipakai")
}

// Jalur lengkap: tanggal tersimpan, barisnya hilang, dan surel terkirim satu kali.
func TestCompleteSavesAndNotifies(t *testing.T) {
	recorder := notification.NewRecorder()
	service, _ := build(t, recorder)

	result, err := service.Complete(context.Background(), portalAlias,
		inboxreceivetka.Completion{ClaimNumber: "PNC-1546", CompletedAt: completedAt()})
	require.NoError(t, err)

	require.True(t, result.NotificationAttempted)
	require.True(t, result.NotificationSent)
	require.Equal(t, 1, recorder.Attempts())

	// Keenam nilai surel berasal dari baris yang dibaca DI DALAM transaksi, bukan dari
	// permintaan — sehingga surel tidak dapat memuat keadaan yang sudah basi.
	notices := recorder.Notices()
	require.Len(t, notices, 1)
	require.Equal(t, "PNC-1546", notices[0].ClaimNumber)
	require.Equal(t, "12000000000002", notices[0].PolicyNumber)
	require.Equal(t, "PT Contoh Sejahtera Abadi", notices[0].InsuredName)
	require.Equal(t, "Peserta Contoh Satu", notices[0].ParticipantName)
	require.Equal(t, completedAt(), notices[0].CompletedAt)
}

// # Uji yang paling penting di berkas ini
//
// Surel yang gagal TIDAK membatalkan penyimpanan. Itu perbedaan yang disengaja terhadap
// sistem lama, yang mengirim surel SEBELUM `Commit` sehingga kegagalannya membuang seluruh
// pekerjaan pengguna.
//
// Yang dibuktikan di sini bukan sekadar bahwa galatnya ditelan, melainkan bahwa TANGGALNYA
// BENAR-BENAR TERSIMPAN — barisnya harus hilang dari daftar meski surelnya gagal.
func TestCompleteKeepsSaveWhenNotificationFails(t *testing.T) {
	recorder := notification.NewRecorder()
	recorder.SetError(errors.New("server surel tidak dapat dihubungi"))
	service, repo := build(t, recorder)

	result, err := service.Complete(context.Background(), portalAlias,
		inboxreceivetka.Completion{ClaimNumber: "PNC-1546", CompletedAt: completedAt()})
	require.NoError(t, err, "kegagalan surel bukan kegagalan penyimpanan")

	require.True(t, result.NotificationAttempted, "pengirimannya memang dicoba")
	require.False(t, result.NotificationSent)
	require.Equal(t, 1, recorder.Attempts())

	page, err := repo.List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	for _, one := range page.Tasks {
		require.NotEqual(t, "PNC-1546", one.ClaimNumber,
			"tanggalnya harus tetap tersimpan meski surelnya gagal")
	}
}

// Notifier yang memang tidak dipasang DIBEDAKAN dari yang dipasang lalu gagal.
//
// Menyatukan keduanya akan membuat lingkungan pengembangan yang tidak punya server surel
// terus-menerus menampilkan peringatan gagal — dan peringatan yang selalu muncul berhenti
// dibaca.
func TestCompleteWithoutNotifierIsNotAFailure(t *testing.T) {
	service, _ := build(t, nil)
	require.False(t, service.NotificationEnabled())

	result, err := service.Complete(context.Background(), portalAlias,
		inboxreceivetka.Completion{ClaimNumber: "PNC-1546", CompletedAt: completedAt()})
	require.NoError(t, err)

	require.False(t, result.NotificationAttempted)
	require.False(t, result.NotificationSent)
}

// Validasi berjalan SEBELUM basis data disentuh, dan galatnya dikembalikan apa adanya
// supaya transport dapat memetakannya dengan errors.Is.
func TestCompleteRejectsEmptyDateBeforeTouchingRepo(t *testing.T) {
	recorder := notification.NewRecorder()
	service, repo := build(t, recorder)

	_, err := service.Complete(context.Background(), portalAlias,
		inboxreceivetka.Completion{ClaimNumber: "PNC-1546"})
	require.ErrorIs(t, err, inboxreceivetka.ErrDateRequired)

	require.Equal(t, 0, recorder.Attempts(), "surel tidak boleh dicoba saat validasi gagal")

	page, err := repo.List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	require.Len(t, page.Tasks, 6, "tidak satu baris pun boleh berubah")
}

// Galat domain dari repo naik apa adanya, tidak dibungkus menjadi galat teknis.
//
// Bila dibungkus, transport tidak dapat mengenalinya dan seluruhnya jatuh menjadi 500 —
// pengguna akan melihat "kesalahan sistem" atas baris yang sebenarnya hanya sudah
// dikerjakan orang lain.
func TestCompletePropagatesDomainErrors(t *testing.T) {
	recorder := notification.NewRecorder()
	service, _ := build(t, recorder)

	cases := []struct {
		name  string
		claim string
		want  error
	}{
		{name: "tidak ada", claim: "PNC-000000", want: inboxreceivetka.ErrTaskNotFound},
		{name: "klaim yatim", claim: "PNC-1977", want: inboxreceivetka.ErrClaimMissing},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			_, err := service.Complete(context.Background(), portalAlias,
				inboxreceivetka.Completion{
					ClaimNumber: one.claim,
					CompletedAt: completedAt(),
				})
			require.ErrorIs(t, err, one.want)
		})
	}

	require.Equal(t, 0, recorder.Attempts(),
		"surel tidak boleh dikirim atas pengisian yang ditolak")
}

// Portal dipilih LEBIH DULU, sebelum satu baris pun disentuh — pada jalur baca maupun tulis.
//
// Tanpa itu, kegagalan portal akan terbaca sebagai daftar kosong atau sebagai kegagalan
// penyimpanan, dan keduanya menyesatkan (`R-20`).
func TestPortalIsCheckedFirst(t *testing.T) {
	recorder := notification.NewRecorder()
	service, _ := build(t, recorder)

	t.Run("baca", func(t *testing.T) {
		_, err := service.List(context.Background(), "SMAS", "")
		require.ErrorIs(t, err, portal.ErrNotReady)
	})

	t.Run("tulis", func(t *testing.T) {
		_, err := service.Complete(context.Background(), "SMAS",
			inboxreceivetka.Completion{ClaimNumber: "PNC-1546", CompletedAt: completedAt()})
		require.ErrorIs(t, err, portal.ErrNotReady)
		require.Equal(t, 0, recorder.Attempts())
	})

	t.Run("EnsurePortalReady", func(t *testing.T) {
		require.NoError(t, service.EnsurePortalReady(portalAlias))
		require.Error(t, service.EnsurePortalReady("SMAS"))
	})
}
