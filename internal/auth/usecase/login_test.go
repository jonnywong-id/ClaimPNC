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

type perkakas struct {
	service     *usecase.Service
	clock       *clock.Fixed
	userRepo    *memory.UserRepo
	sessionRepo *memory.SessionRepo
}

// prepare merakit layanan lengkap tanpa basis data dan tanpa jaringan: provider
// identitas tiruan dan penyimpanan di memori keduanya bekerja di dalam proses.
func prepare(t *testing.T) perkakas {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	testClock := clock.FixedAt(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC))
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

	return perkakas{service: service, clock: testClock, userRepo: userRepo, sessionRepo: sessionRepo}
}

func kredensialSah() auth.Credential {
	return auth.Credential{Username: "adminpnc", Password: "rahasia123"}
}

func TestLoginIssuesUsableSession(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	require.Equal(t, "90000001", result.User.Identity)
	require.Equal(t, p.clock.Now().Add(testLifetime), result.Session.ExpiresAt)

	baseCtx, err := p.service.Check(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, result.User.Identity, baseCtx.User.Identity)
}

// Token yang diterbitkan tidak memuat kredensial dan tidak memuat data nasabah —
// dibuktikan dengan membongkar isinya (TKT-F3-003).
func TestTokenCarriesNeitherCredentialNorIdentity(t *testing.T) {
	p := prepare(t)

	result, err := p.service.Login(context.Background(), kredensialSah())
	require.NoError(t, err)

	content, err := base64.RawURLEncoding.DecodeString(string(result.Token))
	require.NoError(t, err, "token harus berupa nilai acak yang dapat dibongkar, bukan teks bermakna")
	require.Len(t, content, 32, "token harus 32 byte acak")

	mentah := strings.ToLower(string(result.Token) + " " + string(content))
	for _, rahasia := range []string{"rahasia123", "adminpnc", "90000001", "contoh administrator", "example.invalid"} {
		require.NotContains(t, mentah, strings.ToLower(rahasia),
			"token tidak boleh memuat kredensial maupun identitas pengguna")
	}
}

// Yang tersimpan adalah sidik token, bukan tokennya. Bocornya isi tabel sesi tidak
// dengan sendirinya memberi orang lain sesi yang dapat dipakai.
func TestRawTokenIsNeverStored(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
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
func TestExpiredSessionDistinguishedFromInvalidToken(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
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
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)

	_, err = p.service.Check(ctx, result.Token)
	require.NoError(t, err)

	require.NoError(t, p.service.Logout(ctx, result.Token))

	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionRevoked)
	require.True(t, p.clock.Now().Before(result.Session.ExpiresAt),
		"sesi masih dalam masa berlaku; penolakannya harus karena pencabutan")
}

// Logout mencabut sesi di server, bukan menghapus barisnya (ADR-0012).
func TestLogoutDoesNotDeleteSessionRow(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
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
func TestSessionRecognisedByAnotherInstance(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)
	instansB, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        p.userRepo,
		SessionRepo:     p.sessionRepo,
		Clock:           p.clock,
		SessionLifetime: testLifetime,
	})
	require.NoError(t, err)

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)

	baseCtx, err := instansB.Check(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, result.User.Identity, baseCtx.User.Identity)

	require.NoError(t, instansB.Logout(ctx, result.Token))
	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionRevoked, "pencabutan di satu instans berlaku di instans lain")
}

func TestRenewShiftsExpiry(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)

	p.clock.Advance(20 * time.Minute)
	extended, err := p.service.Renew(ctx, result.Token)
	require.NoError(t, err)
	require.Equal(t, p.clock.Now().Add(testLifetime), extended.ExpiresAt)

	p.clock.Advance(testLifetime - time.Minute)
	_, err = p.service.Check(ctx, result.Token)
	require.NoError(t, err, "sesi yang sudah diperpanjang masih berlaku")
}

func TestExpiredSessionCannotBeRenewed(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)

	p.clock.Advance(testLifetime + time.Second)
	_, err = p.service.Renew(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrSessionExpired)
}

// Profil yang salah satu fieldnya kosong ditolak — pengguna yang berhasil masuk tetapi
// cabangnya kosong tidak dapat dicocokkan dengan data klaimnya sendiri.
func TestIncompleteProfileRejected(t *testing.T) {
	p := prepare(t)

	_, err := p.service.Login(context.Background(),
		auth.Credential{Username: "profilbolong", Password: "rahasia123"})
	require.Error(t, err)

	var issues *auth.IncompleteProfileError
	require.ErrorAs(t, err, &issues)
	require.Equal(t, []string{"nama"}, issues.EmptyFields)
	require.Equal(t, 0, p.sessionRepo.Count(), "tidak ada sesi yang boleh terbit")
}

func TestEmptyCredentialAnsweredSameAsWrongCredential(t *testing.T) {
	p := prepare(t)

	_, errEmpty := p.service.Login(context.Background(), auth.Credential{})
	_, errWrong := p.service.Login(context.Background(),
		auth.Credential{Username: "adminpnc", Password: "salah"})

	require.ErrorIs(t, errEmpty, auth.ErrWrongCredential)
	require.ErrorIs(t, errWrong, auth.ErrWrongCredential)
}

// Pengguna yang dinonaktifkan setelah masuk kehilangan aksesnya pada permintaan
// berikutnya — status aktif dibaca dari basis data, tidak tertanam di sesi.
func TestDeactivatedUserRejectedOnNextRequest(t *testing.T) {
	p := prepare(t)
	ctx := context.Background()

	result, err := p.service.Login(ctx, kredensialSah())
	require.NoError(t, err)

	p.userRepo.SetActive(result.User.Identity, false)

	_, err = p.service.Check(ctx, result.Token)
	require.ErrorIs(t, err, auth.ErrUserInactive)
}

func TestLogoutWithUnknownTokenIsNotAnError(t *testing.T) {
	p := prepare(t)
	require.NoError(t, p.service.Logout(context.Background(), auth.Token("bukan token siapa pun")))
}

func TestServiceRejectsIncompleteDeps(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}
