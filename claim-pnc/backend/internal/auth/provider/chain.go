package provider

import (
	"context"
	"errors"

	"claim-pnc/internal/auth"
)

// Chain mencoba beberapa sumber identitas berurutan.
//
// # Urutannya adalah aturan bisnis, bukan detail teknis
//
// Ditetapkan Work Owner 2026-09-16:
//
//  1. Kredensial dicoba ke API HCC/HCQ. Bila `pyErrorCode` = "200", pengguna masuk
//     sebagai karyawan dan namanya diambil dari `EmpResponse.Person.Name`.
//  2. Bila langkah 1 gagal — apa pun sebabnya — kata sandi disidik dengan SHA-256 dan
//     dicocokkan ke POOLDATA.M_LOGIN_PNC untuk pengguna non-karyawan (broker dan
//     surveyor independen).
//
// Urutan ini tidak boleh dibalik: mendahulukan tabel lokal berarti kata sandi karyawan
// ikut disidik dan dicocokkan ke tabel yang bukan tempatnya.
type Chain struct {
	mataRantai []Link
}

// Link adalah satu sumber identitas beserta namanya untuk keperluan galat.
type Link struct {
	Name   string
	Sumber auth.Identity
}

// NewChain membentuk rantai. Urutan slice adalah urutan percobaan.
func NewChain(mataRantai ...Link) (*Chain, error) {
	if len(mataRantai) == 0 {
		return nil, errors.New("provider: rantai identitas tidak boleh kosong")
	}
	for _, m := range mataRantai {
		if m.Sumber == nil {
			return nil, errors.New("provider: ada mata rantai tanpa sumber identitas")
		}
	}
	return &Chain{mataRantai: mataRantai}, nil
}

// Verify mencoba setiap sumber berurutan sampai ada yang menerima.
//
// # Galat mana yang dilaporkan bila semuanya gagal
//
// Bukan sekadar galat terakhir. Bila ADA satu sumber yang tidak dapat dihubungi, yang
// dilaporkan adalah ErrIdentitySystemUnreachable — karena kita memang **tidak tahu** apakah
// kredensialnya salah; bisa jadi ia benar tetapi sistem yang memverifikasinya sedang
// mati. Menyebutnya "kata sandi salah" dalam keadaan itu menyuruh pengguna mengetik
// ulang sesuatu yang sebenarnya sudah benar.
//
// Pengguna yang tidak aktif dilaporkan apa adanya dan menghentikan rantai: mencoba
// sumber berikutnya untuk akun yang sengaja dinonaktifkan tidak masuk akal.
func (b *Chain) Verify(ctx context.Context, k auth.Credential) (auth.Profile, error) {
	var (
		anyDown bool
		lastErr error
	)

	for _, mata := range b.mataRantai {
		profile, err := mata.Sumber.Verify(ctx, k)
		if err == nil {
			return profile, nil
		}

		switch {
		case errors.Is(err, auth.ErrUserInactive):
			return auth.Profile{}, err

		case errors.Is(err, auth.ErrIdentitySystemUnreachable):
			anyDown = true
			lastErr = err

		case errors.Is(err, auth.ErrWrongCredential):
			if lastErr == nil {
				lastErr = err
			}

		default:
			// Galat yang tidak dikenali — misalnya profil tidak lengkap dari sumber
			// yang menjawab berhasil — tidak ditelan. Ia menghentikan rantai supaya
			// tidak tersamarkan menjadi "kredensial salah".
			return auth.Profile{}, err
		}
	}

	if anyDown {
		return auth.Profile{}, lastErr
	}
	if lastErr != nil {
		return auth.Profile{}, lastErr
	}
	return auth.Profile{}, auth.ErrWrongCredential
}
