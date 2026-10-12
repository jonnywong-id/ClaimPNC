package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcloseclaim/repo/memory"
	"claim-pnc/internal/inboxcloseclaim/usecase"
)

const portal = "asm"

// service membentuk layanan dengan gerbang kewenangan TERBUKA.
//
// Gerbang yang sesungguhnya — When rule `IsGCNMUser` — selalu salah, sehingga tanpa ini
// seluruh uji di bawah hanya akan membuktikan satu hal: bahwa gerbangnya menutup. Yang
// hendak dijaga justru sisanya: pemeriksaan klaim, pencatatan niat, dan penolakan permintaan
// ganda — kode yang menyala pada hari gerbang itu dibuka.
//
// Bahwa gerbangnya benar-benar menutup dijaga terpisah oleh
// TestRequestDitolakSaatGerbangTertutup, yang TIDAK mengisi seam-nya.
func service(t *testing.T, store *memory.Store) *usecase.Service {
	t.Helper()

	svc, err := usecase.NewService(usecase.Options{
		CanRequest: func() bool { return true },
		Claims: func(alias string) (inboxcloseclaim.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal")
			}
			return store, nil
		},
		Requests: func(alias string) (inboxcloseclaim.RequestRepo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal")
			}
			return store, nil
		},
		IDs:   memory.IDGenerator{},
		Clock: memory.FixedClock{At: time.Date(2026, 9, 23, 4, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)
	return svc
}

// TestNewServiceRequiresEverySeam menjaga perakitan yang tidak lengkap gagal saat start,
// bukan saat permintaan pertama datang.
func TestNewServiceRequiresEverySeam(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// TestUnknownPortalIsRejected menjaga `R-20`.
//
// Portal yang tidak dikenal menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama.
// Jatuh ke koneksi default berarti menampilkan klaim satu badan hukum di layar badan hukum
// lain tanpa satu pun pesan galat.
func TestUnknownPortalIsRejected(t *testing.T) {
	svc := service(t, memory.NewStoreWithSamples())

	_, err := svc.List(context.Background(), usecase.ListQuery{PortalAlias: "entah"})
	require.Error(t, err)
}

// TestListAttachesPendingRequests adalah alasan utama lapisan ini ada.
//
// Klaim TIDAK berubah saat tombolnya ditekan — yang tercatat baru permintaannya. Tanpa
// penanda ini, pengguna yang tidak melihat perubahan akan menekan tombolnya lagi.
func TestListAttachesPendingRequests(t *testing.T) {
	store := memory.NewStoreWithSamples()
	svc := service(t, store)
	ctx := context.Background()

	const klaim = "ASM-FW-GCNMFW-WORK PNCN.26.0001"

	_, err := svc.Request(ctx, usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     klaim,
		ActorLogin:  "BUDI",
	})
	require.NoError(t, err)

	result, err := svc.List(ctx, usecase.ListQuery{PortalAlias: portal})
	require.NoError(t, err)
	require.NoError(t, result.PendingLookupError)
	require.Len(t, result.Pending[klaim], 1)
	require.Equal(t, inboxcloseclaim.RequestReopen, result.Pending[klaim][0].Kind)
}

// TestReopenRecordsTheIntendedEffect menjaga niat ikut tersimpan pada barisnya.
//
// Supaya permintaan yang dijalankan bulan depan dijalankan menurut aturan yang berlaku SAAT
// IA DIAJUKAN — dan yang menjalankannya tidak perlu menebak aturan mana yang berlaku.
func TestReopenRecordsTheIntendedEffect(t *testing.T) {
	store := memory.NewStoreWithSamples()
	svc := service(t, store)

	request, err := svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		ActorLogin:  "budi",
		ActorName:   "Budi Santoso",
		Reason:      "dokumen susulan diterima",
	})
	require.NoError(t, err)

	require.Equal(t, inboxcloseclaim.EfekStatusKerjaReopen, request.EffectWorkStatus)
	require.Equal(t, inboxcloseclaim.EfekStatusKlaimReopen, request.EffectClaimStatus)
	require.Empty(t, request.CopyScope, "lingkup salin tidak berlaku pada reopen")

	require.Equal(t, "BUDI", request.ActorLogin, "login dinormalkan huruf besar")
	require.Equal(t, "PNCN.26.0001", request.ClaimNumber, "nomor klaim disalin, bukan dirujuk")
	require.Equal(t, inboxcloseclaim.RequestPending, request.Status)
	require.False(t, request.RequestedAt.IsZero())
}

// TestCopyRecordsTheAgreedScope menjaga lingkup salin yang ditetapkan Work Owner.
func TestCopyRecordsTheAgreedScope(t *testing.T) {
	store := memory.NewStoreWithSamples()
	svc := service(t, store)

	request, err := svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestCopy,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		ActorLogin:  "BUDI",
	})
	require.NoError(t, err)

	require.Equal(t, inboxcloseclaim.LingkupSalinBaku, request.CopyScope)
	require.Empty(t, request.EffectWorkStatus, "efek status tidak berlaku pada salin")
}

// TestRequestRejectsClaimThatIsStillRunning menjaga gerbang yang paling penting pada modul
// ini.
//
// Kunci klaim datang dari peramban. Tanpa memeriksanya lebih dulu, permintaan ReOpen dapat
// diajukan atas klaim yang MASIH BERJALAN — yang justru tidak boleh dibuka kembali karena
// belum pernah tutup.
func TestRequestRejectsClaimThatIsStillRunning(t *testing.T) {
	store := memory.NewStoreWithSamples()
	store.AddClaim(inboxcloseclaim.ClosedClaim{
		ClaimID:       "ASM-FW-GCNMFW-WORK PNCN.26.7777",
		ClaimNumber:   "PNCN.26.7777",
		ProcessStatus: "New",
	})
	svc := service(t, store)

	_, err := svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.7777",
		ActorLogin:  "BUDI",
	})
	require.ErrorIs(t, err, inboxcloseclaim.ErrClaimNotFound)

	require.Empty(t, store.Requests(), "tidak ada baris yang boleh tercatat saat gerbangnya menolak")
}

// TestRequestRejectsUnknownClaim menjaga kunci yang tidak ada dijawab 404, bukan 500.
func TestRequestRejectsUnknownClaim(t *testing.T) {
	svc := service(t, memory.NewStoreWithSamples())

	_, err := svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK TIDAK-ADA",
		ActorLogin:  "BUDI",
	})
	require.ErrorIs(t, err, inboxcloseclaim.ErrClaimNotFound)
}

// TestSecondPendingRequestIsRejected menjaga tombol yang ditekan dua kali.
//
// Pada 'salin', dua baris yang lolos berarti DUA KLAIM BARU dari satu tombol.
func TestSecondPendingRequestIsRejected(t *testing.T) {
	svc := service(t, memory.NewStoreWithSamples())
	ctx := context.Background()

	cmd := usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestCopy,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		ActorLogin:  "BUDI",
	}

	_, err := svc.Request(ctx, cmd)
	require.NoError(t, err)

	_, err = svc.Request(ctx, cmd)
	require.ErrorIs(t, err, inboxcloseclaim.ErrRequestPending)
}

// TestRequestWithoutActorIsRejected menjaga jejak tidak pernah tanpa pelaku.
//
// `D-59` menghapus pemisahan tugas, sehingga jejak inilah satu-satunya kontrol pengimbang
// yang tersisa — dan jejak tanpa pelaku tidak mengimbangi apa pun.
func TestRequestWithoutActorIsRejected(t *testing.T) {
	svc := service(t, memory.NewStoreWithSamples())

	_, err := svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
	})

	var validation *inboxcloseclaim.ValidationError
	require.True(t, errors.As(err, &validation))
}

// TestListSurvivesMissingRequestTable menjaga perilaku yang NYATA hari ini.
//
// Migrasi `0006` belum dijalankan DBA di lingkungan mana pun, sehingga pembacaan permintaan
// akan gagal di setiap portal. Daftarnya TIDAK boleh ikut mati — mengembalikan galat berarti
// layar ini tidak dapat dipakai sama sekali sampai perubahan skema selesai.
func TestListSurvivesMissingRequestTable(t *testing.T) {
	store := memory.NewStoreWithSamples()

	svc, err := usecase.NewService(usecase.Options{
		Claims: func(string) (inboxcloseclaim.Repo, error) { return store, nil },
		Requests: func(string) (inboxcloseclaim.RequestRepo, error) {
			return nil, errors.New("ORA-00942: table or view does not exist")
		},
		IDs:   memory.IDGenerator{},
		Clock: memory.FixedClock{At: time.Now().UTC()},
	})
	require.NoError(t, err)

	result, err := svc.List(context.Background(), usecase.ListQuery{PortalAlias: portal})
	require.NoError(t, err, "daftar tetap tampil meski tabel permintaan belum ada")
	require.NotEmpty(t, result.Page.Claims)
	require.Error(t, result.PendingLookupError, "kegagalannya tetap dilaporkan, bukan ditelan diam-diam")
}

// TestRequestDitolakSaatGerbangTertutup menjaga keputusan Work Owner 2026-09-23.
//
// Ia SENGAJA tidak mengisi seam CanRequest, sehingga yang berlaku adalah aturan yang
// sesungguhnya — When rule `IsGCNMUser`, yang isinya `1 = 2`.
//
// Inilah satu-satunya uji di berkas ini yang memakai gerbang bawaan. Bila kelak Work Owner
// membuka gerbangnya, uji INI yang gagal lebih dulu — dan kegagalannya itulah pengingat
// bahwa keputusannya berubah, bukan bahwa kodenya rusak.
func TestRequestDitolakSaatGerbangTertutup(t *testing.T) {
	store := memory.NewStoreWithSamples()

	svc, err := usecase.NewService(usecase.Options{
		Claims:   func(string) (inboxcloseclaim.Repo, error) { return store, nil },
		Requests: func(string) (inboxcloseclaim.RequestRepo, error) { return store, nil },
		IDs:      memory.IDGenerator{},
		Clock:    memory.FixedClock{At: time.Now().UTC()},
		// CanRequest sengaja TIDAK diisi.
	})
	require.NoError(t, err)

	require.False(t, svc.CanRequest(), "gerbang bawaan wajib tertutup")

	_, err = svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		ActorLogin:  "BUDI",
	})
	require.ErrorIs(t, err, inboxcloseclaim.ErrRequestNotAllowed)

	require.Empty(t, store.Requests(),
		"tidak satu pun baris boleh tercatat saat gerbangnya menutup")
}

// TestGerbangDiperiksaSebelumKlaimDicari menjaga urutan yang mencegah penebakan nomor klaim.
//
// Bila kewenangan diperiksa BELAKANGAN, pemanggil yang tidak berwenang tetap dapat
// menanyakan keberadaan sebuah klaim lewat perbedaan galat yang ia terima: "tidak ditemukan"
// untuk kunci yang salah, "tidak berwenang" untuk kunci yang benar. Perbedaan itu cukup
// untuk menebak nomor klaim satu per satu.
//
// Yang diuji: kunci yang JELAS TIDAK ADA pun dijawab ErrRequestNotAllowed, bukan
// ErrClaimNotFound.
func TestGerbangDiperiksaSebelumKlaimDicari(t *testing.T) {
	store := memory.NewStoreWithSamples()

	svc, err := usecase.NewService(usecase.Options{
		Claims:   func(string) (inboxcloseclaim.Repo, error) { return store, nil },
		Requests: func(string) (inboxcloseclaim.RequestRepo, error) { return store, nil },
		IDs:      memory.IDGenerator{},
		Clock:    memory.FixedClock{At: time.Now().UTC()},
	})
	require.NoError(t, err)

	_, err = svc.Request(context.Background(), usecase.RequestCommand{
		PortalAlias: portal,
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK KLAIM-YANG-TIDAK-PERNAH-ADA",
		ActorLogin:  "BUDI",
	})

	require.ErrorIs(t, err, inboxcloseclaim.ErrRequestNotAllowed)
	require.NotErrorIs(t, err, inboxcloseclaim.ErrClaimNotFound,
		"keberadaan klaim tidak boleh bocor lewat perbedaan galat")
}
