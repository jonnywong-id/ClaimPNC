package provider

import (
	"context"
	"errors"

	"claim-pnc/internal/auth"
)

// Berantai mencoba beberapa sumber identitas berurutan.
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
type Berantai struct {
	mataRantai []MataRantai
}

// MataRantai adalah satu sumber identitas beserta namanya untuk keperluan galat.
type MataRantai struct {
	Nama   string
	Sumber auth.Identitas
}

// BerantaiBaru membentuk rantai. Urutan slice adalah urutan percobaan.
func BerantaiBaru(mataRantai ...MataRantai) (*Berantai, error) {
	if len(mataRantai) == 0 {
		return nil, errors.New("provider: rantai identitas tidak boleh kosong")
	}
	for _, m := range mataRantai {
		if m.Sumber == nil {
			return nil, errors.New("provider: ada mata rantai tanpa sumber identitas")
		}
	}
	return &Berantai{mataRantai: mataRantai}, nil
}

// Verifikasi mencoba setiap sumber berurutan sampai ada yang menerima.
//
// # Galat mana yang dilaporkan bila semuanya gagal
//
// Bukan sekadar galat terakhir. Bila ADA satu sumber yang tidak dapat dihubungi, yang
// dilaporkan adalah ErrSistemTidakTerhubung — karena kita memang **tidak tahu** apakah
// kredensialnya salah; bisa jadi ia benar tetapi sistem yang memverifikasinya sedang
// mati. Menyebutnya "kata sandi salah" dalam keadaan itu menyuruh pengguna mengetik
// ulang sesuatu yang sebenarnya sudah benar.
//
// Pengguna yang tidak aktif dilaporkan apa adanya dan menghentikan rantai: mencoba
// sumber berikutnya untuk akun yang sengaja dinonaktifkan tidak masuk akal.
func (b *Berantai) Verifikasi(ctx context.Context, k auth.Kredensial) (auth.Profil, error) {
	var (
		adaYangPutus  bool
		galatTerakhir error
	)

	for _, mata := range b.mataRantai {
		profil, err := mata.Sumber.Verifikasi(ctx, k)
		if err == nil {
			return profil, nil
		}

		switch {
		case errors.Is(err, auth.ErrPenggunaTidakAktif):
			return auth.Profil{}, err

		case errors.Is(err, auth.ErrSistemTidakTerhubung):
			adaYangPutus = true
			galatTerakhir = err

		case errors.Is(err, auth.ErrKredensialSalah):
			if galatTerakhir == nil {
				galatTerakhir = err
			}

		default:
			// Galat yang tidak dikenali — misalnya profil tidak lengkap dari sumber
			// yang menjawab berhasil — tidak ditelan. Ia menghentikan rantai supaya
			// tidak tersamarkan menjadi "kredensial salah".
			return auth.Profil{}, err
		}
	}

	if adaYangPutus {
		return auth.Profil{}, galatTerakhir
	}
	if galatTerakhir != nil {
		return auth.Profil{}, galatTerakhir
	}
	return auth.Profil{}, auth.ErrKredensialSalah
}
