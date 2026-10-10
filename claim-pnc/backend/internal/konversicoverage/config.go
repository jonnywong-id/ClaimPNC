package konversicoverage

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Connection adalah satu koneksi Oracle yang dibaca dari variabel lingkungan
// KONVERSI_<PERAN>_HOST, _PORT, _SERVICE, _PENGGUNA, _SANDI.
type Connection struct {
	Role     string // LIVE | TEST
	Host     string
	Port     int
	Service  string
	User     string
	Password string
}

// Complete menyatakan apakah keempat isian wajibnya terisi.
func (c Connection) Complete() bool {
	return c.Host != "" && c.Service != "" && c.User != "" && c.Password != ""
}

// Missing menyebut nama variabel yang belum terisi.
func (c Connection) Missing() []string {
	var missing []string
	prefix := "KONVERSI_" + c.Role + "_"
	for name, value := range map[string]string{
		"HOST": c.Host, "SERVICE": c.Service, "PENGGUNA": c.User, "SANDI": c.Password,
	} {
		if value == "" {
			missing = append(missing, prefix+name)
		}
	}
	return missing
}

// Label adalah alamat koneksi tanpa kata sandi — aman ditampilkan di layar.
func (c Connection) Label() string {
	if c.Host == "" {
		return ""
	}
	return fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Service)
}

// Config adalah konfigurasi modul.
type Config struct {
	Live Connection
	Test Connection
	// Policies adalah daftar nomor polis bawaan dari KONVERSI_POLIS, untuk mengisi layar.
	Policies []string
}

// ErrSameDatabase berarti LIVE dan TEST menunjuk basis data dan pengguna yang sama.
var ErrSameDatabase = errors.New(
	"koneksi LIVE dan TEST menunjuk basis data dan pengguna yang sama; konversi ditolak supaya LIVE tidak tertulis")

// Validate memeriksa konfigurasi sebelum koneksi dibuka.
func (c Config) Validate() error {
	var missing []string
	missing = append(missing, c.Live.Missing()...)
	missing = append(missing, c.Test.Missing()...)
	if len(missing) > 0 {
		return fmt.Errorf("konfigurasi koneksi belum lengkap: %s", strings.Join(missing, ", "))
	}
	if strings.EqualFold(c.Live.Host, c.Test.Host) && c.Live.Port == c.Test.Port &&
		strings.EqualFold(c.Live.Service, c.Test.Service) && strings.EqualFold(c.Live.User, c.Test.User) {
		return ErrSameDatabase
	}
	return nil
}

// LoadConfig membaca konfigurasi dari lingkungan proses (berkas .env sudah dimuat
// cmd/claimpnc lebih dulu).
func LoadConfig() Config {
	return Config{
		Live:     loadConnection("LIVE"),
		Test:     loadConnection("TEST"),
		Policies: ParsePolicyList(os.Getenv("KONVERSI_POLIS")),
	}
}

func loadConnection(role string) Connection {
	prefix := "KONVERSI_" + role + "_"
	port, err := strconv.Atoi(strings.TrimSpace(os.Getenv(prefix + "PORT")))
	if err != nil || port <= 0 {
		port = 1521
	}
	return Connection{
		Role:     role,
		Host:     strings.TrimSpace(os.Getenv(prefix + "HOST")),
		Port:     port,
		Service:  strings.TrimSpace(os.Getenv(prefix + "SERVICE")),
		User:     strings.TrimSpace(os.Getenv(prefix + "PENGGUNA")),
		Password: os.Getenv(prefix + "SANDI"),
	}
}
