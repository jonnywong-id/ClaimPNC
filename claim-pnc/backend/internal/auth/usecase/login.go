// Package usecase mengorkestrasi alur modul auth: masuk, pemeriksaan sesi,
// perpanjangan, dan keluar.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket auth dan tidak memuat
// aturan modul sendiri. Ia juga tidak tahu apa pun soal HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/platform/clock"
)

// Service menyatukan sistem identitas, catatan pengguna lokal, dan penyimpanan sesi.
type Service struct {
	identity        auth.Identity
	userRepo        auth.UserRepo
	sessionRepo     auth.SessionRepo
	clock           clock.Clock
	sessionLifetime time.Duration
	sumberAcak      io.Reader
}

// Options adalah bahan pembentuk Service. Seluruhnya wajib kecuali SumberAcak.
type Options struct {
	Identity        auth.Identity
	UserRepo        auth.UserRepo
	SessionRepo     auth.SessionRepo
	Clock           clock.Clock
	SessionLifetime time.Duration
	SumberAcak      io.Reader
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalan di
// sini terjadi saat start, bukan saat pengguna pertama mencoba masuk.
func NewService(o Options) (*Service, error) {
	switch {
	case o.Identity == nil:
		return nil, errors.New("usecase: seam identitas wajib diisi")
	case o.UserRepo == nil:
		return nil, errors.New("usecase: penyimpanan pengguna wajib diisi")
	case o.SessionRepo == nil:
		return nil, errors.New("usecase: penyimpanan sesi wajib diisi")
	case o.Clock == nil:
		return nil, errors.New("usecase: seam clock wajib diisi")
	case o.SessionLifetime <= 0:
		return nil, errors.New("usecase: masa berlaku sesi harus lebih besar dari nol")
	}
	return &Service{
		identity:        o.Identity,
		userRepo:        o.UserRepo,
		sessionRepo:     o.SessionRepo,
		clock:           o.Clock,
		sessionLifetime: o.SessionLifetime,
		sumberAcak:      o.SumberAcak,
	}, nil
}

// Result adalah yang didapat pemanggil setelah berhasil masuk. Token di dalamnya adalah
// satu-satunya kesempatan membacanya — setelah ini yang tersimpan hanyalah sidiknya.
type Result struct {
	Token   auth.Token
	Session auth.Session
	User    auth.User
}

// Caller adalah identitas pemanggil yang sudah terbukti, dipakai seluruh lapisan lain.
type Caller struct {
	User    auth.User
	Session auth.Session
}

// Login memverifikasi kredensial ke sistem identitas, menyegarkan catatan pengguna
// lokal, lalu menerbitkan sesi milik aplikasi.
//
// Urutannya disengaja: pemanggilan sistem luar selesai lebih dulu, sebelum satu pun
// baris basis data disentuh — kegagalan jaringan tidak boleh menahan kunci baris.
func (l *Service) Login(ctx context.Context, k auth.Credential) (Result, error) {
	if !k.Complete() {
		// Kredensial kosong dijawab sama dengan kredensial salah. Membedakannya
		// memberi tahu penyerang bahwa formatnya sudah benar.
		return Result{}, auth.ErrWrongCredential
	}

	profile, err := l.identity.Verify(ctx, k)
	if err != nil {
		return Result{}, err
	}
	if err := profile.Check(); err != nil {
		return Result{}, err
	}

	p, err := l.refreshUser(ctx, profile)
	if err != nil {
		return Result{}, err
	}
	if !p.Active {
		return Result{}, auth.ErrUserInactive
	}

	token, err := auth.IssueToken(l.sumberAcak)
	if err != nil {
		return Result{}, fmt.Errorf("usecase: menerbitkan token: %w", err)
	}
	sessionID, err := auth.NewSessionID(l.sumberAcak)
	if err != nil {
		return Result{}, fmt.Errorf("usecase: menerbitkan pengenal sesi: %w", err)
	}
	sekarang := l.clock.Now().UTC()
	s := auth.Session{
		ID:          sessionID,
		TokenDigest: token.Digest(),
		Identity:    p.Identity,
		IssuedAt:    sekarang,
		ExpiresAt:   sekarang.Add(l.sessionLifetime),
	}
	if err := l.sessionRepo.Save(ctx, s); err != nil {
		return Result{}, fmt.Errorf("usecase: menyimpan sesi: %w", err)
	}
	return Result{Token: token, Session: s, User: p}, nil
}

// Check menguji token yang dibawa permintaan dan mengembalikan identitas pemiliknya.
func (l *Service) Check(ctx context.Context, token auth.Token) (Caller, error) {
	if strings.TrimSpace(string(token)) == "" {
		return Caller{}, auth.ErrSessionNotFound
	}
	s, err := l.sessionRepo.GetByTokenDigest(ctx, token.Digest())
	if err != nil {
		return Caller{}, err
	}
	if err := s.Check(l.clock.Now().UTC()); err != nil {
		return Caller{}, err
	}
	p, err := l.userRepo.GetByIdentity(ctx, s.Identity)
	if err != nil {
		return Caller{}, fmt.Errorf("usecase: memuat pengguna sesi: %w", err)
	}
	if !p.Active {
		return Caller{}, auth.ErrUserInactive
	}
	return Caller{User: p, Session: s}, nil
}

// Renew menggeser batas berlaku sesi yang masih hidup. Sesi yang sudah kedaluwarsa
// atau dicabut tidak dapat dihidupkan kembali — pengguna harus masuk ulang.
func (l *Service) Renew(ctx context.Context, token auth.Token) (auth.Session, error) {
	baseCtx, err := l.Check(ctx, token)
	if err != nil {
		return auth.Session{}, err
	}
	newLimit := l.clock.Now().UTC().Add(l.sessionLifetime)
	if err := l.sessionRepo.Renew(ctx, baseCtx.Session.ID, newLimit); err != nil {
		return auth.Session{}, fmt.Errorf("usecase: memperpanjang sesi: %w", err)
	}
	baseCtx.Session.ExpiresAt = newLimit
	return baseCtx.Session, nil
}

// Logout mencabut sesi di server. Mencabut sesi yang sudah tidak berlaku bukan galat:
// hasil akhirnya sama, dan pengguna tidak perlu diberi tahu bedanya.
func (l *Service) Logout(ctx context.Context, token auth.Token) error {
	s, err := l.sessionRepo.GetByTokenDigest(ctx, token.Digest())
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	if s.RevokedAt != nil {
		return nil
	}
	if err := l.sessionRepo.Revoke(ctx, s.ID, l.clock.Now().UTC()); err != nil {
		return fmt.Errorf("usecase: mencabut sesi: %w", err)
	}
	return nil
}

// refreshUser menyalin profil terbaru dari sistem identitas ke catatan lokal.
//
// Catatan: pembaruan catatan pengguna dan penyimpanan sesi belum berada dalam satu
// transaksi — kepemilikan transaksi di lapisan aplikasi adalah lingkup TKT-F2-003 yang
// belum dikerjakan. Dampak bila langkah kedua gagal terbatas: catatan pengguna
// tersegarkan tanpa sesi terbit, dan penyegaran itu idempoten.
func (l *Service) refreshUser(ctx context.Context, profile auth.Profile) (auth.User, error) {
	sekarang := l.clock.Now().UTC()

	p, err := l.userRepo.GetByIdentity(ctx, profile.Identity)
	switch {
	case errors.Is(err, auth.ErrUserNotFound):
		p = auth.FromProfile(profile, sekarang)
	case err != nil:
		return auth.User{}, fmt.Errorf("usecase: memuat pengguna: %w", err)
	default:
		// Status aktif dan OperatorID tidak ikut tersegarkan — keduanya dimiliki
		// administrator aplikasi ini, bukan sistem identitas luar.
		p.RefreshFrom(profile, sekarang)
	}

	if err := l.userRepo.Save(ctx, p); err != nil {
		return auth.User{}, fmt.Errorf("usecase: menyimpan pengguna: %w", err)
	}
	return p, nil
}
