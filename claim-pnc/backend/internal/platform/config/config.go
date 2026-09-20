// Package config membaca konfigurasi aplikasi dari luar proses dan gagal keras saat
// start bila ada nilai wajib yang tidak terisi.
//
// Aturan yang mengikat paket ini:
//   - Nilai rahasia (kata sandi basis data, kredensial HCQ) hanya berasal dari variabel
//     lingkungan atau berkas .env, tidak pernah dari nilai baku di dalam kode (ADR-0025).
//   - Tidak ada nilai bisnis yang di-hardcode.
//   - Struct hasil pembacaan tidak pernah ditulis utuh ke log; lihat Summary().
//   - **Daftar portal tidak ditulis di kode.** Alias portal ditemukan dengan memindai
//     variabel lingkungan, sesuai ADR-0030 yang menetapkan daftar portal adalah data.
package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Environment menyatakan tempat aplikasi berjalan. Nilainya menentukan adapter mana
// yang boleh hidup — provider identitas tiruan menolak berjalan di Production.
type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// Nama provider identitas yang dikenali.
const (
	IdentityAdapterFake = "fake"
	IdentityAdapterHCQ  = "hcq"
)

// Nama penyimpanan yang dikenali.
const (
	StorageOracle = "oracle"
	StorageMemory = "memori"
)

// portalPrefix adalah awalan variabel lingkungan koneksi portal:
// POOLDATA_<ALIAS>_HOST, _PORT, _SERVICE, _PENGGUNA, _SANDI.
const portalPrefix = "POOLDATA_"

// defaultSessionLifetime dipilih 60 menit: docs/Steering/11-SECURITY.md §2.2 menetapkan
// rentang 30–60 menit, dan form registrasi klaim tergolong panjang sehingga batas atas
// rentang itu yang dipakai. Nilai final menunggu Work Owner + Security (ADR-0024).
const defaultSessionLifetime = 60 * time.Minute

// defaultPrimaryPortal mengikuti ADR-0030 yang menyebut Asuransi Sinar Mas sebagai entitas
// utama. Nilainya dapat diubah lewat PORTAL_UTAMA tanpa menyentuh kode.
const defaultPrimaryPortal = "ASM"

// Config adalah seluruh nilai yang dibaca saat start.
type Config struct {
	Environment     Environment
	Address         string
	IdentityAdapter string
	Storage         string
	Session         Session
	HCQ             HCQ
	Cashier         Cashier
	SMTP            SMTP

	// PrimaryPortal adalah alias portal yang basis datanya melayani hal-hal yang
	// dibutuhkan SEBELUM pengguna memilih portal: daftar portal (M_PORTAL_PNC),
	// alamat layanan HCQ (GCNM_CONNECT_REST), login non-karyawan (M_LOGIN_PNC), dan
	// tabel sesi.
	//
	// Keputusan Work Owner 2026-09-16, ditandai **sementara**: kelak portal utama
	// mungkin ditentukan per login dari tabel. Karena itu nilainya dibaca dari
	// konfigurasi, bukan ditulis di kode.
	PrimaryPortal string

	// Portal memetakan alias portal ke parameter koneksinya. Isinya ditemukan dengan
	// memindai lingkungan, bukan dari daftar tetap.
	Portal map[string]Database
}

// Sesi memuat parameter masa hidup sesi milik aplikasi.
type Session struct {
	Lifetime time.Duration
}

// HCQ memuat kredensial Basic Auth ke API autentikasi HCC/HCQ.
//
// Alamat endpoint-nya TIDAK di sini: ia dibaca dari POOLDATA.GCNM_CONNECT_REST agar
// perpindahan endpoint menjadi perubahan data, bukan perubahan konfigurasi aplikasi.
type HCQ struct {
	User     string
	Password string
	Timeout  time.Duration
}

// Kasir memuat alamat dan kredensial layanan pendaftaran rekening ke sistem Kasir.
//
// Dipakai modul Master Rekening saat komite menyetujui sebuah rekening. Berbeda dari
// HCQ, alamatnya ADA di sini dan bukan di basis data: sistem lama menyimpannya di
// konfigurasi instans Pega, bukan di GCNM_CONNECT_REST, sehingga tidak ada tabel yang
// dapat dibaca untuk menemukannya.
//
// Seluruhnya boleh kosong. Bila kosong, modul Master Rekening tetap berjalan penuh dan
// hanya melewatkan langkah pendaftaran ke Kasir — itu yang membuat layar dapat dipakai
// sebelum kredensialnya tersedia dari Tim Infra.
type Cashier struct {
	RegisterURL string
	UpdateURL   string
	User        string
	Password    string
	Timeout     time.Duration
}

// Aktif menyatakan konfigurasi ini cukup untuk menghubungi Kasir.
func (k Cashier) Active() bool {
	return strings.TrimSpace(k.RegisterURL) != "" && strings.TrimSpace(k.UpdateURL) != ""
}

// SMTP memuat parameter server surel keluar.
//
// Dipakai modul Master Rekening untuk memberi tahu komite bila rekening yang baru
// disetujuinya gagal didaftarkan ke Kasir.
//
// User dan Password boleh kosong: relay SMTP internal sering menerima pengirim
// dari jaringan tepercaya tanpa autentikasi. Bila keduanya diisi, kredensialnya HANYA
// dikirim setelah STARTTLS berhasil — penolakannya ada di kode, bukan di konfigurasi.
type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string

	// AlertRecipients adalah mailbox Tim IT yang menerima peringatan kegagalan
	// integrasi. Ditulis sebagai daftar dipisah koma di SMTP_PENERIMA_PERINGATAN.
	//
	// Keputusan Work Owner 2026-09-17: peringatan ditujukan ke pihak yang dapat
	// MEMPERBAIKI kegagalan, bukan ke pengguna yang kebetulan memicunya.
	AlertRecipients []string

	// XOLCommitteeRecipients adalah mailbox komite yang menerima pemberitahuan pengajuan
	// Master XOL. Ditulis sebagai daftar dipisah koma di XOL_PENERIMA_KOMITE.
	//
	// # Kenapa ia konfigurasi, dan kenapa hanya SEMENTARA
	//
	// `Activity/SendDataMasterXOLToKomites-Act.xml` menuliskan dua alamat PERORANGAN
	// langsung di dalam activity-nya, dan yang kedua menimpa yang pertama ketika berjalan
	// di host dev. Pola itu persis yang `D-15` larang dibawa, dan `D-67` menegaskan tidak
	// ada akun pribadi yang ikut ke sistem baru.
	//
	// Tempat yang benar bagi daftar ini adalah master **Penerima Notifikasi** (`F-4`),
	// dan master itu belum dibangun. Sampai ia ada, daftarnya ditaruh di konfigurasi —
	// tetap dapat diubah tanpa menyentuh kode, tetapi belum dapat diubah pengguna bisnis
	// sendiri. Dicatat terbuka di docs/keputusan-implementasi.md.
	//
	// Kosong berarti pemberitahuan TIDAK dikirim: cmd memasang tiruan yang mencatat, dan
	// penyimpanan Master XOL tetap berhasil.
	XOLCommitteeRecipients []string

	Timeout time.Duration
}

// Aktif menyatakan konfigurasi ini cukup untuk mengirim surel.
//
// Penerima ikut disyaratkan: pengirim surel tanpa tujuan bukan setengah aktif, ia
// tidak aktif — dan menyatakannya aktif akan menyembunyikan konfigurasi yang belum
// selesai di balik pengiriman yang tidak pernah sampai ke siapa pun.
func (s SMTP) Active() bool {
	return strings.TrimSpace(s.Host) != "" &&
		s.Port > 0 &&
		strings.TrimSpace(s.From) != "" &&
		len(s.AlertRecipients) > 0
}

// splitAddress memecah daftar alamat yang dipisah koma dan membuang yang kosong.
func splitAddress(list string) []string {
	var result []string
	for _, part := range strings.Split(list, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Database memuat parameter koneksi satu portal. Password tidak pernah ikut tercetak.
type Database struct {
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

// Complete menyatakan apakah seluruh parameter wajib koneksi ini terisi.
//
// Portal yang belum lengkap bukan galat: pengisian kredensial tiap entitas berjalan
// bertahap. Portal seperti itu ditandai tidak tersedia, dan memilihnya menghasilkan
// pesan yang menyebut variabel mana yang missing.
func (b Database) Complete() bool {
	return b.Host != "" && b.Service != "" && b.User != "" && b.Password != ""
}

// Missing menyebut variabel lingkungan yang belum terisi untuk portal ini.
func (b Database) Missing() []string {
	var missing []string
	for name, value := range map[string]string{
		portalPrefix + b.Alias + "_HOST":     b.Host,
		portalPrefix + b.Alias + "_SERVICE":  b.Service,
		portalPrefix + b.Alias + "_PENGGUNA": b.User,
		portalPrefix + b.Alias + "_SANDI":    b.Password,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

// Load membaca seluruh konfigurasi. Seluruh kesalahan dikumpulkan, tidak berhenti pada
// yang pertama, supaya operator melihat semua yang missing dalam satu kali jalan.
func Load() (Config, error) {
	var issues []error

	env := Environment(strings.TrimSpace(get("APP_ENV", string(Development))))
	if !knownEnvironments(env) {
		issues = append(issues, fmt.Errorf("APP_ENV %q tidak dikenal; pilihan: development, test, staging, production", env))
	}

	adapter := strings.TrimSpace(get("IDENTITAS_ADAPTER", IdentityAdapterFake))
	if adapter != IdentityAdapterFake && adapter != IdentityAdapterHCQ {
		issues = append(issues, fmt.Errorf("IDENTITAS_ADAPTER %q tidak dikenal; pilihan: fake, hcq", adapter))
	}

	storage := strings.TrimSpace(get("PENYIMPANAN", defaultStorage(env)))
	if storage != StorageOracle && storage != StorageMemory {
		issues = append(issues, fmt.Errorf("PENYIMPANAN %q tidak dikenal; pilihan: oracle, memori", storage))
	}

	masaBerlaku, err := getDuration("SESI_MASA_BERLAKU", defaultSessionLifetime)
	if err != nil {
		issues = append(issues, err)
	}
	hcqTimeout, err := getDuration("HCQ_LOGIN_BATAS_WAKTU", 15*time.Second)
	if err != nil {
		issues = append(issues, err)
	}
	cashierTimeout, err := getDuration("KASIR_BATAS_WAKTU", 30*time.Second)
	if err != nil {
		issues = append(issues, err)
	}
	portSMTP, err := getInt("SMTP_PORT", 0)
	if err != nil {
		issues = append(issues, err)
	}
	smtpTimeout, err := getDuration("SMTP_BATAS_WAKTU", 20*time.Second)
	if err != nil {
		issues = append(issues, err)
	}

	primaryPortal := strings.ToUpper(strings.TrimSpace(get("PORTAL_UTAMA", defaultPrimaryPortal)))
	portal, portalErrs := loadPortals()
	issues = append(issues, portalErrs...)

	k := Config{
		Environment:     env,
		Address:         get("APP_ALAMAT", ":8080"),
		IdentityAdapter: adapter,
		Storage:         storage,
		Session:         Session{Lifetime: masaBerlaku},
		HCQ: HCQ{
			User:     strings.TrimSpace(os.Getenv("HCQ_LOGIN_USER")),
			Password: os.Getenv("HCQ_LOGIN_PASSWORD"),
			Timeout:  hcqTimeout,
		},
		Cashier: Cashier{
			RegisterURL: strings.TrimSpace(os.Getenv("KASIR_URL_DAFTAR_REKENING")),
			UpdateURL:   strings.TrimSpace(os.Getenv("KASIR_URL_PERBARUI_REKENING")),
			User:        strings.TrimSpace(os.Getenv("KASIR_USER")),
			Password:    os.Getenv("KASIR_PASSWORD"),
			Timeout:     cashierTimeout,
		},
		SMTP: SMTP{
			Host:            strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:            portSMTP,
			User:            strings.TrimSpace(os.Getenv("SMTP_USER")),
			Password:        os.Getenv("SMTP_PASSWORD"),
			From:            strings.TrimSpace(os.Getenv("SMTP_DARI")),
			AlertRecipients: splitAddress(os.Getenv("SMTP_PENERIMA_PERINGATAN")),

			XOLCommitteeRecipients: splitAddress(os.Getenv("XOL_PENERIMA_KOMITE")),

			Timeout: smtpTimeout,
		},
		PrimaryPortal: primaryPortal,
		Portal:        portal,
	}

	issues = append(issues, checkDependencies(k)...)

	if len(issues) > 0 {
		return Config{}, fmt.Errorf("konfigurasi tidak sah:\n  - %s", strings.Join(errorMessage(issues), "\n  - "))
	}
	return k, nil
}

// checkDependencies memeriksa syarat yang baru dapat dinilai setelah seluruh nilai
// terbaca — misalnya portal utama harus benar-benar ada bila penyimpanannya Oracle.
//
// Pesannya sengaja menyebut **cara memperbaikinya**, bukan hanya apa yang missing.
// Galat saat start dibaca orang yang sedang terhenti; menyebut variabel yang hilang
// tanpa menyebut langkah berikutnya hanya memindahkan pekerjaan menebak kepadanya.
func checkDependencies(k Config) []error {
	var issues []error

	// Koneksi portal dibutuhkan bila tabel CPNC_ hidup di Oracle ATAU bila identitas
	// nyata dipakai — yang kedua membaca GCNM_CONNECT_REST dan M_LOGIN_PNC dari sana.
	if k.Storage == StorageOracle || k.IdentityAdapter == IdentityAdapterHCQ {
		primary, existing := k.Portal[k.PrimaryPortal]
		switch {
		case !existing:
			issues = append(issues, fmt.Errorf(
				"PORTAL_UTAMA %q tidak punya satu pun variabel %s%s_*; portal yang terbaca: %s.\n%s",
				k.PrimaryPortal, portalPrefix, k.PrimaryPortal, aliasList(k.Portal),
				portalFixHint(k.PrimaryPortal)))
		case !primary.Complete():
			issues = append(issues, fmt.Errorf(
				"portal utama %q belum lengkap; yang missing: %s.\n%s",
				k.PrimaryPortal, strings.Join(primary.Missing(), ", "),
				portalFixHint(k.PrimaryPortal)))
		}
	}

	if k.IdentityAdapter == IdentityAdapterHCQ {
		if k.HCQ.User == "" {
			issues = append(issues, fmt.Errorf("HCQ_LOGIN_USER wajib diisi bila IDENTITAS_ADAPTER=hcq"))
		}
		if k.HCQ.Password == "" {
			issues = append(issues, fmt.Errorf("HCQ_LOGIN_PASSWORD wajib diisi bila IDENTITAS_ADAPTER=hcq"))
		}
	}
	return issues
}

// portalFixHint menyebut dua jalan keluar beserta jebakan yang paling sering
// terjadi: berkas .env dibaca relatif terhadap direktori kerja, bukan letak binary.
func portalFixHint(alias string) string {
	return "    Perbaikan — salah satu dari:\n" +
		"      (a) salin .env.example menjadi .env di folder backend/, lalu isi " +
		portalPrefix + alias + "_HOST, _SERVICE, _PENGGUNA, _SANDI\n" +
		"      (b) jalankan tanpa basis data: PENYIMPANAN=memori + IDENTITAS_ADAPTER=fake (hanya di luar produksi)\n" +
		"    Note: .env dibaca relatif terhadap direktori kerja, jadi jalankan dari dalam folder backend/."
}

// loadPortals menemukan alias portal dengan memindai lingkungan.
//
// Alias diambil dari setiap variabel berbentuk POOLDATA_<ALIAS>_HOST. Cara ini dipilih
// supaya menambah portal cukup dengan menambah lima baris di .env — tanpa menyentuh
// kode sama sekali (ADR-0030: daftar portal adalah data, bukan konstanta).
func loadPortals() (map[string]Database, []error) {
	var issues []error
	result := map[string]Database{}

	for _, alias := range aliasesFromEnvironment() {
		port, err := getInt(portalPrefix+alias+"_PORT", 1521)
		if err != nil {
			issues = append(issues, err)
		}
		maxConnections, err := getInt(portalPrefix+alias+"_MAKS_KONEKSI", 20)
		if err != nil {
			issues = append(issues, err)
		}
		maksIdle, err := getInt(portalPrefix+alias+"_MAKS_IDLE", 5)
		if err != nil {
			issues = append(issues, err)
		}
		umur, err := getDuration(portalPrefix+alias+"_UMUR_KONEKSI", 30*time.Minute)
		if err != nil {
			issues = append(issues, err)
		}

		result[alias] = Database{
			Alias:              alias,
			Host:               strings.TrimSpace(os.Getenv(portalPrefix + alias + "_HOST")),
			Port:               port,
			Service:            strings.TrimSpace(os.Getenv(portalPrefix + alias + "_SERVICE")),
			User:               strings.TrimSpace(os.Getenv(portalPrefix + alias + "_PENGGUNA")),
			Password:           os.Getenv(portalPrefix + alias + "_SANDI"),
			MaxConnections:     maxConnections,
			MaxIdle:            maksIdle,
			ConnectionLifetime: umur,
		}
	}
	return result, issues
}

func aliasesFromEnvironment() []string {
	ditemukan := map[string]bool{}
	for _, rows := range os.Environ() {
		name, _, _ := strings.Cut(rows, "=")
		if !strings.HasPrefix(name, portalPrefix) || !strings.HasSuffix(name, "_HOST") {
			continue
		}
		alias := strings.TrimSuffix(strings.TrimPrefix(name, portalPrefix), "_HOST")
		if alias != "" && !strings.Contains(alias, "_") {
			ditemukan[strings.ToUpper(alias)] = true
		}
	}
	alias := make([]string, 0, len(ditemukan))
	for a := range ditemukan {
		alias = append(alias, a)
	}
	sort.Strings(alias)
	return alias
}

// AvailableAliases mengembalikan alias portal yang koneksinya lengkap, terurut.
func (k Config) AvailableAliases() []string {
	var alias []string
	for a, b := range k.Portal {
		if b.Complete() {
			alias = append(alias, a)
		}
	}
	sort.Strings(alias)
	return alias
}

// Summary mengembalikan bentuk konfigurasi yang aman ditulis ke log: kata sandi basis
// data dan kredensial HCQ tidak pernah ikut, bahkan sebagiannya.
func (k Config) Summary() map[string]any {
	return map[string]any{
		"lingkungan":         string(k.Environment),
		"alamat":             k.Address,
		"adapter_identitas":  k.IdentityAdapter,
		"penyimpanan":        k.Storage,
		"sesi_masa_berlaku":  k.Session.Lifetime.String(),
		"portal_utama":       k.PrimaryPortal,
		"portal_terbaca":     aliasList(k.Portal),
		"portal_tersedia":    strings.Join(k.AvailableAliases(), ","),
		"hcq_pengguna_diisi": k.HCQ.User != "",
		"hcq_sandi_diisi":    k.HCQ.Password != "",
		"kasir_aktif":        k.Cashier.Active(),
		"smtp_aktif":         k.SMTP.Active(),
	}
}

func aliasList(portal map[string]Database) string {
	if len(portal) == 0 {
		return "(tidak ada)"
	}
	alias := make([]string, 0, len(portal))
	for a := range portal {
		alias = append(alias, a)
	}
	sort.Strings(alias)
	return strings.Join(alias, ",")
}

func knownEnvironments(l Environment) bool {
	switch l {
	case Development, Test, Staging, Production:
		return true
	default:
		return false
	}
}

func get(name, fallback string) string {
	if value, existing := os.LookupEnv(name); existing && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getInt(name string, fallback int) (int, error) {
	mentah, existing := os.LookupEnv(name)
	if !existing || strings.TrimSpace(mentah) == "" {
		return fallback, nil
	}
	angka, err := strconv.Atoi(strings.TrimSpace(mentah))
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka, terbaca %q", name, mentah)
	}
	return angka, nil
}

func getDuration(name string, fallback time.Duration) (time.Duration, error) {
	mentah, existing := os.LookupEnv(name)
	if !existing || strings.TrimSpace(mentah) == "" {
		return fallback, nil
	}
	durasi, err := time.ParseDuration(strings.TrimSpace(mentah))
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa durasi seperti 45m atau 1h, terbaca %q", name, mentah)
	}
	if durasi <= 0 {
		return 0, fmt.Errorf("%s harus lebih besar dari nol, terbaca %q", name, mentah)
	}
	return durasi, nil
}

// errorMessage mengubah daftar galat menjadi daftar teks, supaya seluruhnya tercetak
// sebagai butir terpisah dan operator melihat semua yang missing sekaligus.
func errorMessage(issues []error) []string {
	message := make([]string, 0, len(issues))
	for _, g := range issues {
		message = append(message, g.Error())
	}
	return message
}

// defaultStorage memilih penyimpanan yang masuk akal bila PENYIMPANAN tidak disetel.
//
// Di pengembangan dan pengujian: **memori**, supaya `go run` pada clone yang baru
// langsung jalan tanpa satu pun kredensial basis data. Di staging dan produksi:
// **oracle**, dan bila kredensialnya missing aplikasi gagal start dengan pesan yang
// menyebut apa yang harus diisi.
//
// Nilai baku ini aman karena penolakannya berlapis: penyimpanan memori **menolak
// berjalan di produksi** dari dalam kode (lihat cmd/claimpnc), bukan hanya lewat nilai
// baku ini. Jadi lingkungan produksi yang lupa menyetel PENYIMPANAN tidak mungkin
// diam-diam menyimpan sesi di memori.
//
// Penyimpanan yang benar-benar dipakai selalu tercetak di baris log "konfigurasi
// terbaca" saat start, sehingga tidak ada yang perlu menebak.
func defaultStorage(l Environment) string {
	switch l {
	case Staging, Production:
		return StorageOracle
	default:
		return StorageMemory
	}
}
