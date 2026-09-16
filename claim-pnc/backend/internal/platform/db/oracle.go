// Package db membuka koneksi basis data dan mengatur connection pool-nya.
//
// Satu aplikasi memegang **beberapa koneksi sekaligus** — satu per portal entitas
// (ADR-0030: satu database per entitas). Kumpulan itulah yang dikelola paket ini.
//
// Driver yang dipakai adalah go-ora (murni Go). docs/Steering/08-TECHNICAL-STRATEGY.md §1
// menetapkan godror; godror menuntut CGO dan Oracle Instant Client, keduanya tidak
// tersedia di lingkungan pengembangan ini. Kedua driver sama-sama berbicara
// database/sql sehingga pertukarannya menyentuh berkas ini saja. Penyimpangan ini
// dicatat di docs/keputusan-implementasi.md §3.2.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

// Parameter adalah bahan pembuka satu koneksi. KataSandi tidak pernah ikut tercetak.
type Parameter struct {
	Alias       string
	Host        string
	Port        int
	Service     string
	Pengguna    string
	KataSandi   string
	MaksKoneksi int
	MaksIdle    int
	UmurKoneksi time.Duration
}

// Kumpulan memegang koneksi seluruh portal yang berhasil dibuka.
//
// Portal yang kredensialnya belum diisi atau tidak dapat dihubungi **tidak membuat
// aplikasi gagal start** — ia hanya tidak tersedia, dan memilihnya menghasilkan galat
// yang menyebut portalnya. Pengisian kredensial tiap entitas berjalan bertahap, dan
// satu entitas yang belum siap tidak boleh menghalangi entitas yang sudah siap.
type Kumpulan struct {
	koneksi map[string]*sql.DB
	utama   string
}

// KumpulanBaru membuka koneksi untuk setiap parameter yang diberikan.
//
// Portal utama diperlakukan berbeda: kegagalannya **fatal**, karena basis datanya yang
// melayani daftar portal, alamat layanan HCQ, login non-karyawan, dan tabel sesi.
// Tanpa itu aplikasi tidak dapat melayani satu permintaan pun.
func KumpulanBaru(ctx context.Context, utama string, parameter []Parameter, catat func(alias string, err error)) (*Kumpulan, error) {
	k := &Kumpulan{koneksi: map[string]*sql.DB{}, utama: utama}

	for _, p := range parameter {
		koneksi, err := Buka(ctx, p)
		if err != nil {
			if p.Alias == utama {
				k.Tutup()
				return nil, fmt.Errorf("portal utama %q: %w", utama, err)
			}
			if catat != nil {
				catat(p.Alias, err)
			}
			continue
		}
		k.koneksi[p.Alias] = koneksi
	}

	if _, ada := k.koneksi[utama]; !ada {
		k.Tutup()
		return nil, fmt.Errorf("db: portal utama %q tidak ada di daftar koneksi", utama)
	}
	return k, nil
}

// Utama mengembalikan koneksi portal utama. Ia selalu ada bila Kumpulan terbentuk.
func (k *Kumpulan) Utama() *sql.DB { return k.koneksi[k.utama] }

// AliasUtama mengembalikan alias portal utama.
func (k *Kumpulan) AliasUtama() string { return k.utama }

// Untuk mengembalikan koneksi satu portal.
//
// Modul bisnis memanggil ini dengan portal yang sedang dipilih pengguna, sehingga
// kuerinya mengenai basis data entitas yang benar tanpa perlu menyaring per baris —
// pemisahan datanya ada di tingkat koneksi, bukan di tingkat kueri (ADR-0030 Opsi 1).
func (k *Kumpulan) Untuk(alias string) (*sql.DB, error) {
	koneksi, ada := k.koneksi[strings.ToUpper(strings.TrimSpace(alias))]
	if !ada {
		return nil, fmt.Errorf("db: portal %q tidak tersedia; yang tersedia: %s", alias, strings.Join(k.Tersedia(), ", "))
	}
	return koneksi, nil
}

// Tersedia menyebut alias portal yang koneksinya hidup, terurut.
func (k *Kumpulan) Tersedia() []string {
	alias := make([]string, 0, len(k.koneksi))
	for a := range k.koneksi {
		alias = append(alias, a)
	}
	sort.Strings(alias)
	return alias
}

// Tutup menutup seluruh koneksi.
func (k *Kumpulan) Tutup() {
	for _, koneksi := range k.koneksi {
		_ = koneksi.Close()
	}
}

// Buka membuka satu pool koneksi dan langsung mengujinya.
//
// Pengujian dilakukan saat start supaya parameter yang salah ketahuan saat itu juga —
// bukan berhasil start lalu gagal pada permintaan pengguna pertama.
func Buka(ctx context.Context, p Parameter) (*sql.DB, error) {
	koneksi, err := sql.Open("oracle", dsn(p))
	if err != nil {
		return nil, fmt.Errorf("db: membuka koneksi Oracle: %w", err)
	}

	// Batas pool dijaga rendah dengan sengaja. Selama masa paralel basis data dibagi
	// dengan Pega (ADR-0004); pool yang terlalu besar memakan koneksi yang dibutuhkan
	// Pega untuk melayani produksi. Menaikkannya harus dibicarakan dengan DBA.
	koneksi.SetMaxOpenConns(p.MaksKoneksi)
	koneksi.SetMaxIdleConns(p.MaksIdle)
	koneksi.SetConnMaxLifetime(p.UmurKoneksi)

	ctxUji, batal := context.WithTimeout(ctx, 10*time.Second)
	defer batal()
	if err := koneksi.PingContext(ctxUji); err != nil {
		_ = koneksi.Close()
		return nil, fmt.Errorf("db: tidak dapat menghubungi Oracle di %s:%d/%s sebagai %s: %w",
			p.Host, p.Port, p.Service, p.Pengguna, err)
	}
	return koneksi, nil
}

// dsn menyusun alamat koneksi. Kata sandi disandikan URL supaya karakter khusus di
// dalamnya tidak merusak alamat — dan hasilnya tidak pernah ditulis ke log.
func dsn(p Parameter) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(p.Pengguna), url.QueryEscape(p.KataSandi), p.Host, p.Port, p.Service)
}
