// Package config membaca konfigurasi aplikasi dari luar proses dan gagal keras saat
// start bila ada nilai wajib yang tidak terisi.
//
// Aturan yang mengikat paket ini:
//   - Nilai rahasia (kata sandi basis data, kredensial HCQ) hanya berasal dari variabel
//     lingkungan atau berkas .env, tidak pernah dari nilai baku di dalam kode (ADR-0025).
//   - Tidak ada nilai bisnis yang di-hardcode.
//   - Struct hasil pembacaan tidak pernah ditulis utuh ke log; lihat Ringkas().
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

// Lingkungan menyatakan tempat aplikasi berjalan. Nilainya menentukan adapter mana
// yang boleh hidup — provider identitas tiruan menolak berjalan di Produksi.
type Lingkungan string

const (
	Pengembangan Lingkungan = "development"
	Pengujian    Lingkungan = "test"
	Staging      Lingkungan = "staging"
	Produksi     Lingkungan = "production"
)

// Nama provider identitas yang dikenali.
const (
	AdapterIdentitasTiruan = "fake"
	AdapterIdentitasNyata  = "hcq"
)

// Nama penyimpanan yang dikenali.
const (
	PenyimpananOracle = "oracle"
	PenyimpananMemori = "memori"
)

// awalanPortal adalah awalan variabel lingkungan koneksi portal:
// POOLDATA_<ALIAS>_HOST, _PORT, _SERVICE, _PENGGUNA, _SANDI.
const awalanPortal = "POOLDATA_"

// masaBerlakuSesiBaku dipilih 60 menit: docs/Steering/11-SECURITY.md §2.2 menetapkan
// rentang 30–60 menit, dan form registrasi klaim tergolong panjang sehingga batas atas
// rentang itu yang dipakai. Nilai final menunggu Work Owner + Security (ADR-0024).
const masaBerlakuSesiBaku = 60 * time.Minute

// portalUtamaBaku mengikuti ADR-0030 yang menyebut Asuransi Sinar Mas sebagai entitas
// utama. Nilainya dapat diubah lewat PORTAL_UTAMA tanpa menyentuh kode.
const portalUtamaBaku = "ASM"

// Konfigurasi adalah seluruh nilai yang dibaca saat start.
type Konfigurasi struct {
	Lingkungan       Lingkungan
	Alamat           string
	AdapterIdentitas string
	Penyimpanan      string
	Sesi             Sesi
	HCQ              HCQ

	// PortalUtama adalah alias portal yang basis datanya melayani hal-hal yang
	// dibutuhkan SEBELUM pengguna memilih portal: daftar portal (M_PORTAL_PNC),
	// alamat layanan HCQ (GCNM_CONNECT_REST), login non-karyawan (M_LOGIN_PNC), dan
	// tabel sesi.
	//
	// Keputusan Work Owner 2026-09-16, ditandai **sementara**: kelak portal utama
	// mungkin ditentukan per login dari tabel. Karena itu nilainya dibaca dari
	// konfigurasi, bukan ditulis di kode.
	PortalUtama string

	// Portal memetakan alias portal ke parameter koneksinya. Isinya ditemukan dengan
	// memindai lingkungan, bukan dari daftar tetap.
	Portal map[string]Basisdata
}

// Sesi memuat parameter masa hidup sesi milik aplikasi.
type Sesi struct {
	MasaBerlaku time.Duration
}

// HCQ memuat kredensial Basic Auth ke API autentikasi HCC/HCQ.
//
// Alamat endpoint-nya TIDAK di sini: ia dibaca dari POOLDATA.GCNM_CONNECT_REST agar
// perpindahan endpoint menjadi perubahan data, bukan perubahan konfigurasi aplikasi.
type HCQ struct {
	Pengguna  string
	KataSandi string
	Batas     time.Duration
}

// Basisdata memuat parameter koneksi satu portal. KataSandi tidak pernah ikut tercetak.
type Basisdata struct {
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

// Lengkap menyatakan apakah seluruh parameter wajib koneksi ini terisi.
//
// Portal yang belum lengkap bukan galat: pengisian kredensial tiap entitas berjalan
// bertahap. Portal seperti itu ditandai tidak tersedia, dan memilihnya menghasilkan
// pesan yang menyebut variabel mana yang kurang.
func (b Basisdata) Lengkap() bool {
	return b.Host != "" && b.Service != "" && b.Pengguna != "" && b.KataSandi != ""
}

// YangKurang menyebut variabel lingkungan yang belum terisi untuk portal ini.
func (b Basisdata) YangKurang() []string {
	var kurang []string
	for nama, nilai := range map[string]string{
		awalanPortal + b.Alias + "_HOST":     b.Host,
		awalanPortal + b.Alias + "_SERVICE":  b.Service,
		awalanPortal + b.Alias + "_PENGGUNA": b.Pengguna,
		awalanPortal + b.Alias + "_SANDI":    b.KataSandi,
	} {
		if nilai == "" {
			kurang = append(kurang, nama)
		}
	}
	sort.Strings(kurang)
	return kurang
}

// Muat membaca seluruh konfigurasi. Seluruh kesalahan dikumpulkan, tidak berhenti pada
// yang pertama, supaya operator melihat semua yang kurang dalam satu kali jalan.
func Muat() (Konfigurasi, error) {
	var galat []error

	ling := Lingkungan(strings.TrimSpace(ambil("APP_ENV", string(Pengembangan))))
	if !lingkunganDikenal(ling) {
		galat = append(galat, fmt.Errorf("APP_ENV %q tidak dikenal; pilihan: development, test, staging, production", ling))
	}

	adapter := strings.TrimSpace(ambil("IDENTITAS_ADAPTER", AdapterIdentitasTiruan))
	if adapter != AdapterIdentitasTiruan && adapter != AdapterIdentitasNyata {
		galat = append(galat, fmt.Errorf("IDENTITAS_ADAPTER %q tidak dikenal; pilihan: fake, hcq", adapter))
	}

	penyimpanan := strings.TrimSpace(ambil("PENYIMPANAN", penyimpananBaku(ling)))
	if penyimpanan != PenyimpananOracle && penyimpanan != PenyimpananMemori {
		galat = append(galat, fmt.Errorf("PENYIMPANAN %q tidak dikenal; pilihan: oracle, memori", penyimpanan))
	}

	masaBerlaku, err := ambilDurasi("SESI_MASA_BERLAKU", masaBerlakuSesiBaku)
	if err != nil {
		galat = append(galat, err)
	}
	batasHCQ, err := ambilDurasi("HCQ_LOGIN_BATAS_WAKTU", 15*time.Second)
	if err != nil {
		galat = append(galat, err)
	}

	portalUtama := strings.ToUpper(strings.TrimSpace(ambil("PORTAL_UTAMA", portalUtamaBaku)))
	portal, galatPortal := muatPortal()
	galat = append(galat, galatPortal...)

	k := Konfigurasi{
		Lingkungan:       ling,
		Alamat:           ambil("APP_ALAMAT", ":8080"),
		AdapterIdentitas: adapter,
		Penyimpanan:      penyimpanan,
		Sesi:             Sesi{MasaBerlaku: masaBerlaku},
		HCQ: HCQ{
			Pengguna:  strings.TrimSpace(os.Getenv("HCQ_LOGIN_USER")),
			KataSandi: os.Getenv("HCQ_LOGIN_PASSWORD"),
			Batas:     batasHCQ,
		},
		PortalUtama: portalUtama,
		Portal:      portal,
	}

	galat = append(galat, periksaKetergantungan(k)...)

	if len(galat) > 0 {
		return Konfigurasi{}, fmt.Errorf("konfigurasi tidak sah:\n  - %s", strings.Join(pesanGalat(galat), "\n  - "))
	}
	return k, nil
}

// periksaKetergantungan memeriksa syarat yang baru dapat dinilai setelah seluruh nilai
// terbaca — misalnya portal utama harus benar-benar ada bila penyimpanannya Oracle.
//
// Pesannya sengaja menyebut **cara memperbaikinya**, bukan hanya apa yang kurang.
// Galat saat start dibaca orang yang sedang terhenti; menyebut variabel yang hilang
// tanpa menyebut langkah berikutnya hanya memindahkan pekerjaan menebak kepadanya.
func periksaKetergantungan(k Konfigurasi) []error {
	var galat []error

	// Koneksi portal dibutuhkan bila tabel CPNC_ hidup di Oracle ATAU bila identitas
	// nyata dipakai — yang kedua membaca GCNM_CONNECT_REST dan M_LOGIN_PNC dari sana.
	if k.Penyimpanan == PenyimpananOracle || k.AdapterIdentitas == AdapterIdentitasNyata {
		utama, ada := k.Portal[k.PortalUtama]
		switch {
		case !ada:
			galat = append(galat, fmt.Errorf(
				"PORTAL_UTAMA %q tidak punya satu pun variabel %s%s_*; portal yang terbaca: %s.\n%s",
				k.PortalUtama, awalanPortal, k.PortalUtama, daftarAlias(k.Portal),
				saranPerbaikanPortal(k.PortalUtama)))
		case !utama.Lengkap():
			galat = append(galat, fmt.Errorf(
				"portal utama %q belum lengkap; yang kurang: %s.\n%s",
				k.PortalUtama, strings.Join(utama.YangKurang(), ", "),
				saranPerbaikanPortal(k.PortalUtama)))
		}
	}

	if k.AdapterIdentitas == AdapterIdentitasNyata {
		if k.HCQ.Pengguna == "" {
			galat = append(galat, fmt.Errorf("HCQ_LOGIN_USER wajib diisi bila IDENTITAS_ADAPTER=hcq"))
		}
		if k.HCQ.KataSandi == "" {
			galat = append(galat, fmt.Errorf("HCQ_LOGIN_PASSWORD wajib diisi bila IDENTITAS_ADAPTER=hcq"))
		}
	}
	return galat
}

// saranPerbaikanPortal menyebut dua jalan keluar beserta jebakan yang paling sering
// terjadi: berkas .env dibaca relatif terhadap direktori kerja, bukan letak binary.
func saranPerbaikanPortal(alias string) string {
	return "    Perbaikan — salah satu dari:\n" +
		"      (a) salin .env.example menjadi .env di folder backend/, lalu isi " +
		awalanPortal + alias + "_HOST, _SERVICE, _PENGGUNA, _SANDI\n" +
		"      (b) jalankan tanpa basis data: PENYIMPANAN=memori + IDENTITAS_ADAPTER=fake (hanya di luar produksi)\n" +
		"    Catatan: .env dibaca relatif terhadap direktori kerja, jadi jalankan dari dalam folder backend/."
}

// muatPortal menemukan alias portal dengan memindai lingkungan.
//
// Alias diambil dari setiap variabel berbentuk POOLDATA_<ALIAS>_HOST. Cara ini dipilih
// supaya menambah portal cukup dengan menambah lima baris di .env — tanpa menyentuh
// kode sama sekali (ADR-0030: daftar portal adalah data, bukan konstanta).
func muatPortal() (map[string]Basisdata, []error) {
	var galat []error
	hasil := map[string]Basisdata{}

	for _, alias := range aliasDariLingkungan() {
		port, err := ambilAngka(awalanPortal+alias+"_PORT", 1521)
		if err != nil {
			galat = append(galat, err)
		}
		maksKoneksi, err := ambilAngka(awalanPortal+alias+"_MAKS_KONEKSI", 20)
		if err != nil {
			galat = append(galat, err)
		}
		maksIdle, err := ambilAngka(awalanPortal+alias+"_MAKS_IDLE", 5)
		if err != nil {
			galat = append(galat, err)
		}
		umur, err := ambilDurasi(awalanPortal+alias+"_UMUR_KONEKSI", 30*time.Minute)
		if err != nil {
			galat = append(galat, err)
		}

		hasil[alias] = Basisdata{
			Alias:       alias,
			Host:        strings.TrimSpace(os.Getenv(awalanPortal + alias + "_HOST")),
			Port:        port,
			Service:     strings.TrimSpace(os.Getenv(awalanPortal + alias + "_SERVICE")),
			Pengguna:    strings.TrimSpace(os.Getenv(awalanPortal + alias + "_PENGGUNA")),
			KataSandi:   os.Getenv(awalanPortal + alias + "_SANDI"),
			MaksKoneksi: maksKoneksi,
			MaksIdle:    maksIdle,
			UmurKoneksi: umur,
		}
	}
	return hasil, galat
}

func aliasDariLingkungan() []string {
	ditemukan := map[string]bool{}
	for _, baris := range os.Environ() {
		nama, _, _ := strings.Cut(baris, "=")
		if !strings.HasPrefix(nama, awalanPortal) || !strings.HasSuffix(nama, "_HOST") {
			continue
		}
		alias := strings.TrimSuffix(strings.TrimPrefix(nama, awalanPortal), "_HOST")
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

// AliasTersedia mengembalikan alias portal yang koneksinya lengkap, terurut.
func (k Konfigurasi) AliasTersedia() []string {
	var alias []string
	for a, b := range k.Portal {
		if b.Lengkap() {
			alias = append(alias, a)
		}
	}
	sort.Strings(alias)
	return alias
}

// Ringkas mengembalikan bentuk konfigurasi yang aman ditulis ke log: kata sandi basis
// data dan kredensial HCQ tidak pernah ikut, bahkan sebagiannya.
func (k Konfigurasi) Ringkas() map[string]any {
	return map[string]any{
		"lingkungan":         string(k.Lingkungan),
		"alamat":             k.Alamat,
		"adapter_identitas":  k.AdapterIdentitas,
		"penyimpanan":        k.Penyimpanan,
		"sesi_masa_berlaku":  k.Sesi.MasaBerlaku.String(),
		"portal_utama":       k.PortalUtama,
		"portal_terbaca":     daftarAlias(k.Portal),
		"portal_tersedia":    strings.Join(k.AliasTersedia(), ","),
		"hcq_pengguna_diisi": k.HCQ.Pengguna != "",
		"hcq_sandi_diisi":    k.HCQ.KataSandi != "",
	}
}

func daftarAlias(portal map[string]Basisdata) string {
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

func lingkunganDikenal(l Lingkungan) bool {
	switch l {
	case Pengembangan, Pengujian, Staging, Produksi:
		return true
	default:
		return false
	}
}

func ambil(nama, baku string) string {
	if nilai, ada := os.LookupEnv(nama); ada && strings.TrimSpace(nilai) != "" {
		return nilai
	}
	return baku
}

func ambilAngka(nama string, baku int) (int, error) {
	mentah, ada := os.LookupEnv(nama)
	if !ada || strings.TrimSpace(mentah) == "" {
		return baku, nil
	}
	angka, err := strconv.Atoi(strings.TrimSpace(mentah))
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka, terbaca %q", nama, mentah)
	}
	return angka, nil
}

func ambilDurasi(nama string, baku time.Duration) (time.Duration, error) {
	mentah, ada := os.LookupEnv(nama)
	if !ada || strings.TrimSpace(mentah) == "" {
		return baku, nil
	}
	durasi, err := time.ParseDuration(strings.TrimSpace(mentah))
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa durasi seperti 45m atau 1h, terbaca %q", nama, mentah)
	}
	if durasi <= 0 {
		return 0, fmt.Errorf("%s harus lebih besar dari nol, terbaca %q", nama, mentah)
	}
	return durasi, nil
}

// pesanGalat mengubah daftar galat menjadi daftar teks, supaya seluruhnya tercetak
// sebagai butir terpisah dan operator melihat semua yang kurang sekaligus.
func pesanGalat(galat []error) []string {
	pesan := make([]string, 0, len(galat))
	for _, g := range galat {
		pesan = append(pesan, g.Error())
	}
	return pesan
}

// penyimpananBaku memilih penyimpanan yang masuk akal bila PENYIMPANAN tidak disetel.
//
// Di pengembangan dan pengujian: **memori**, supaya `go run` pada clone yang baru
// langsung jalan tanpa satu pun kredensial basis data. Di staging dan produksi:
// **oracle**, dan bila kredensialnya kurang aplikasi gagal start dengan pesan yang
// menyebut apa yang harus diisi.
//
// Nilai baku ini aman karena penolakannya berlapis: penyimpanan memori **menolak
// berjalan di produksi** dari dalam kode (lihat cmd/claimpnc), bukan hanya lewat nilai
// baku ini. Jadi lingkungan produksi yang lupa menyetel PENYIMPANAN tidak mungkin
// diam-diam menyimpan sesi di memori.
//
// Penyimpanan yang benar-benar dipakai selalu tercetak di baris log "konfigurasi
// terbaca" saat start, sehingga tidak ada yang perlu menebak.
func penyimpananBaku(l Lingkungan) string {
	switch l {
	case Staging, Produksi:
		return PenyimpananOracle
	default:
		return PenyimpananMemori
	}
}
