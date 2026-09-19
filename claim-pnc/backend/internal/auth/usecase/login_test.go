package usecase_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/clock"
)

const testLifetime = 30 * time.Minute

type tools struct {
	service     *usecase.Service
	clock       *clock.FixedClock
	userRepo    *memory.UserRepo
	sessionRepo *memory.SessionRepo
}

// setup merakit layanan lengkap tanpa basis data dan tanpa jaringan: provider
// identitas tiruan dan penyimpanan di memori keduanya bekerja di dalam proses.
func setup(t *testing.T) tools {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	testClock := clock.FixedClockAt(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC))
	userRepo := memory.NewUserRepo()
	sessionRepo := memory.NewSessionRepo()

	service, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        userRepo,
		SessionRepo:     sessionRepo,
		Clock:           testClock,
		SessionLifetime: testLifetime,
	})
	require.NoError(t, err)

	return tools{service: service, clock: testClock, userRepo: userRepo, sessionRepo: sessionRepo}
}

func validCredential() auth.Credential {
	return auth.Credential{Username: "adminpnc", Password: "rahasia123"}
}

func TestLoginIssuesUsableSession(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	require.Equal(t, "90000001", result.User.Identity)
	require.Equal(t, p.clock.Now().Add(testLifetime), result.Session.ExpiresAt)

	sessionCtx, err := p.service.Check(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, result.User.Identity, sessionCtx.User.Identity)
}

// Token yang diterbitkan tidak memuat kredensial dan tidak memuat data nasabah —
// dibuktikan dengan membongkar isinya (TKT-F3-003).
func TestTokenCarriesNoCredentialOrIdentity(t *testing.T) {
	p := setup(t)

	result, err := p.service.Login(context.Background(), validCredential())
	require.NoError(t, err)

	body, err := base64.RawURLEncoding.DecodeString(string(result.Token))
	require.NoError(t, err, "token harus berupa nilai acak yang dapat dibongkar, bukan teks bermakna")
	require.Len(t, body, 32, "token harus 32 byte acak")

	raw := strings.ToLower(string(result.Token) + " " + string(body))
	for _, secret := range []string{"rahasia123", "adminpnc", "90000001", "contoh administrator", "example.invalid"} {
		require.NotContains(t, raw, strings.ToLower(secret),
			"token tidak boleh memuat kredensial maupun identitas pengguna")
	}
}

// Yang tersimpan adalah sidik token, bukan tokennya. Bocornya isi tabel sesi tidak
// dengan sendirinya memberi orang lain sesi yang dapat dipakai.
func TestRawTokenNeverStored(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	stored, err := p.sessionRepo.GetByTokenDigest(ctx, result.Token.Digest())
	require.NoError(t, err)
	require.NotEqual(t, string(result.Token), stored.TokenDigest)
	require.Len(t, stored.TokenDigest, 64, "sidik SHA-256 heksadesimal")
	require.NotEqual(t, stored.ID, stored.TokenDigest, "pengenal sesi tidak boleh diturunkan dari sidik token")
}

// Token kedaluwarsa ditolak dengan galat yang DAPAT DIBEDAKAN dari token tidak sah
// (TKT-F3-003). Frontend memakai pembedaan ini untuk menyelamatkan isian yang belum
// tersimpan, bukan sekadar melempar pengguna ke layar masuk.
func TestExpiredSessionDistinctFromInvalidToken(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	p.clock.Advance(testLifetime + time.Second)

	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionExpired)
	require.NotErrorIs(t, err, auth.ErrSessionNotFound)

	_, err = p.service.Check(ctx, auth.Token("token-yang-tidak-pernah-diterbitkan"))
	require.ErrorIs(t, err, auth.ErrSessionNotFound)
	require.NotErrorIs(t, err, auth.ErrSessionExpired)
}

// Mencabut sesi membuat permintaan berikutnya ditolak SEKETIKA — bukan menunggu masa
// berlaku habis (TKT-F3-003). Jam sengaja tidak digeser sedetik pun di uji ini.
func TestRevocationTakesEffectImmediately(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	_, err = p.service.Check(ctx, result.Token)
	require.NoError(t, err)

	require.NoError(t, p.service.Logout(ctx, result.Token))

	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionRevoked)
	require.True(t, p.clock.Now().Before(result.Session.ExpiresAt),
		"sesi masih dalam masa berlaku; penolakannya harus karena pencabutan")
}

// Keluar mencabut sesi di server, bukan menghapus barisnya (ADR-0012).
func TestLogoutDoesNotDeleteSessionRow(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)
	require.NoError(t, p.service.Logout(ctx, result.Token))

	require.Equal(t, 1, p.sessionRepo.Count())
	stored, err := p.sessionRepo.GetByTokenDigest(ctx, result.Token.Digest())
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
}

// Dua instans aplikasi mengenali sesi yang sama: sesi diterbitkan instans A dan dipakai
// di instans B. Keduanya berbagi penyimpanan, persis seperti dua instans di belakang
// load balancer berbagi satu basis data (D-27).
//
// CATATAN KETERBATASAN: uji ini memakai penyimpanan di memori yang dibagi dua layanan.
// Ia membuktikan bahwa layanan tidak menyimpan state di dirinya sendiri, TETAPI belum
// membuktikan perilaku yang sama terhadap Oracle. Pembuktian itu menunggu basis data
// pengembangan tersedia.
func TestSessionRecognizedByOtherInstance(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)
	instanceB, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        p.userRepo,
		SessionRepo:     p.sessionRepo,
		Clock:           p.clock,
		SessionLifetime: testLifetime,
	})
	require.NoError(t, err)

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	sessionCtx, err := instanceB.Check(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, result.User.Identity, sessionCtx.User.Identity)

	require.NoError(t, instanceB.Logout(ctx, result.Token))
	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionRevoked, "pencabutan di satu instans berlaku di instans lain")
}

func TestExtendMovesDeadline(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	p.clock.Advance(20 * time.Minute)
	extended, err := p.service.Extend(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, p.clock.Now().Add(testLifetime), extended.ExpiresAt)

	p.clock.Advance(testLifetime - time.Minute)
	_, err = p.service.Check(ctx, result.Token)
	require.NoError(t, err, "sesi yang sudah diperpanjang masih berlaku")
}

func TestExpiredSessionCannotBeExtended(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	p.clock.Advance(testLifetime + time.Second)
	_, err = p.service.Extend(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionExpired)
}

// Profil yang salah satu fieldnya kosong ditolak — pengguna yang berhasil masuk tetapi
// cabangnya kosong tidak dapat dicocokkan dengan data klaimnya sendiri.
func TestIncompleteProfileRejected(t *testing.T) {
	p := setup(t)

	_, err := p.service.Login(context.Background(),
		auth.Credential{Username: "profilbolong", Password: "rahasia123"})
	require.Error(t, err)

	var failure *auth.IncompleteProfileError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, []string{"nama"}, failure.EmptyFields)
	require.Equal(t, 0, p.sessionRepo.Count(), "tidak ada sesi yang boleh terbit")
}

func TestEmptyCredentialAnsweredLikeWrongCredential(t *testing.T) {
	p := setup(t)

	_, errEmpty := p.service.Login(context.Background(), auth.Credential{})
	_, errWrong := p.service.Login(context.Background(),
		auth.Credential{Username: "adminpnc", Password: "salah"})

	require.ErrorIs(t, errEmpty, auth.ErrWrongCredential)
	require.ErrorIs(t, errWrong, auth.ErrWrongCredential)
}

// Pengguna yang dinonaktifkan setelah masuk kehilangan aksesnya pada permintaan
// berikutnya — status aktif dibaca dari basis data, tidak tertanam di sesi.
func TestDeactivatedUserRejectedOnNextRequest(t *testing.T) {
	p := setup(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, validCredential())
	require.NoError(t, err)

	p.userRepo.SetActive(result.User.Identity, false)

	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrUserInactive)
}

func TestLogoutWithUnknownTokenIsNotAnError(t *testing.T) {
	p := setup(t)
	require.NoError(t, p.service.Logout(context.Background(), auth.Token("bukan token siapa pun")))
}

func TestServiceRejectsIncompleteParts(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}
