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

// Parameter adalah bahan pembuka satu koneksi. Password tidak pernah ikut tercetak.
type Parameter struct {
	Alias              string
	Host               string
	Port               int
	Service            string
	User               string
	Password           string
	MaxConnections     int
	MaxIdle            int
	ConnectionLifetime time.Duration
}

// Pool memegang koneksi seluruh portal yang berhasil dibuka.
//
// Portal yang kredensialnya belum diisi atau tidak dapat dihubungi **tidak membuat
// aplikasi gagal start** — ia hanya tidak tersedia, dan memilihnya menghasilkan galat
// yang menyebut portalnya. Pengisian kredensial tiap entitas berjalan bertahap, dan
// satu entitas yang belum siap tidak boleh menghalangi entitas yang sudah siap.
type Pool struct {
	connections map[string]*sql.DB
	primary     string
}

// NewPool membuka koneksi untuk setiap parameter yang diberikan.
//
// Portal utama diperlakukan berbeda: kegagalannya **fatal**, karena basis datanya yang
// melayani daftar portal, alamat layanan HCQ, login non-karyawan, dan tabel sesi.
// Tanpa itu aplikasi tidak dapat melayani satu permintaan pun.
func NewPool(ctx context.Context, primary string, parameter []Parameter, record func(alias string, err error)) (*Pool, error) {
	k := &Pool{connections: map[string]*sql.DB{}, primary: primary}

	for _, p := range parameter {
		connections, err := Open(ctx, p)
		if err != nil {
			if p.Alias == primary {
				k.Close()
				return nil, fmt.Errorf("portal utama %q: %w", primary, err)
			}
			if record != nil {
				record(p.Alias, err)
			}
			continue
		}
		k.connections[p.Alias] = connections
	}

	if _, existing := k.connections[primary]; !existing {
		k.Close()
		return nil, fmt.Errorf("db: portal utama %q tidak ada di daftar koneksi", primary)
	}
	return k, nil
}

// Primary mengembalikan koneksi portal utama. Ia selalu ada bila Pool terbentuk.
func (k *Pool) Primary() *sql.DB { return k.connections[k.primary] }

// PrimaryAlias mengembalikan alias portal utama.
func (k *Pool) PrimaryAlias() string { return k.primary }

// For mengembalikan koneksi satu portal.
//
// Modul bisnis memanggil ini dengan portal yang sedang dipilih pengguna, sehingga
// kuerinya mengenai basis data entitas yang benar tanpa perlu menyaring per baris —
// pemisahan datanya ada di tingkat koneksi, bukan di tingkat kueri (ADR-0030 Opsi 1).
func (k *Pool) For(alias string) (*sql.DB, error) {
	connections, existing := k.connections[strings.ToUpper(strings.TrimSpace(alias))]
	if !existing {
		return nil, fmt.Errorf("db: portal %q tidak tersedia; yang tersedia: %s", alias, strings.Join(k.Available(), ", "))
	}
	return connections, nil
}

// Available menyebut alias portal yang koneksinya hidup, terurut.
func (k *Pool) Available() []string {
	alias := make([]string, 0, len(k.connections))
	for a := range k.connections {
		alias = append(alias, a)
	}
	sort.Strings(alias)
	return alias
}

// Close menutup seluruh koneksi.
func (k *Pool) Close() {
	for _, connections := range k.connections {
		_ = connections.Close()
	}
}

// Open membuka satu pool koneksi dan langsung mengujinya.
//
// Pengujian dilakukan saat start supaya parameter yang salah ketahuan saat itu juga —
// bukan berhasil start lalu gagal pada permintaan pengguna pertama.
func Open(ctx context.Context, p Parameter) (*sql.DB, error) {
	connections, err := sql.Open("oracle", dsn(p))
	if err != nil {
		return nil, fmt.Errorf("db: membuka koneksi Oracle: %w", err)
	}

	// Batas waktu pool dijaga rendah dengan sengaja. Selama masa paralel basis data dibagi
	// dengan Pega (ADR-0004); pool yang terlalu besar memakan koneksi yang dibutuhkan
	// Pega untuk melayani produksi. Menaikkannya harus dibicarakan dengan DBA.
	connections.SetMaxOpenConns(p.MaxConnections)
	connections.SetMaxIdleConns(p.MaxIdle)
	connections.SetConnMaxLifetime(p.ConnectionLifetime)

	testCtx, batal := context.WithTimeout(ctx, 10*time.Second)
	defer batal()
	if err := connections.PingContext(testCtx); err != nil {
		_ = connections.Close()
		return nil, fmt.Errorf("db: tidak dapat menghubungi Oracle di %s:%d/%s sebagai %s: %w",
			p.Host, p.Port, p.Service, p.User, err)
	}
	return connections, nil
}

// dsn menyusun alamat koneksi. Kata sandi disandikan URL supaya karakter khusus di
// dalamnya tidak merusak alamat — dan hasilnya tidak pernah ditulis ke log.
func dsn(p Parameter) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(p.User), url.QueryEscape(p.Password), p.Host, p.Port, p.Service)
}
