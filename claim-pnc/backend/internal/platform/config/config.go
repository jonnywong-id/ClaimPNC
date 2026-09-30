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

// anekaPrefix adalah awalan variabel lingkungan koneksi KEDUA milik tiap portal:
// ANEKA_<ALIAS>_HOST, _PORT, _SERVICE, _PENGGUNA, _SANDI.
//
// # Kenapa ada koneksi kedua sama sekali
//
// Sejumlah kueri laporan membaca basis data lain lewat DB Link `@ASMD` — kalender libur
// `GENERAL.HRD_LBR`, jam kerja `DATAMINING.GET_WORKING_HOURS`, dan master mitra
// `GENERAL.LST_MITRA`. `D-25` mengganti seluruh DB Link dengan pemanggilan API, dan API
// penggantinya belum ada (`R-03`).
//
// Work Owner memutuskan 2026-09-24: **yang berupa sub-query tetap memakai DB Link; selain
// itu memakai koneksi langsung** ke basis data yang bersangkutan. Koneksi langsung itulah
// yang dikonfigurasi di sini.
//
// # Kenapa per portal, bukan satu untuk semua
//
// Alasannya sama dengan koneksi portal itu sendiri: satu badan hukum bukan badan hukum
// lain (`ADR-0030`, `R-20`). Koneksi kedua milik portal ASI tidak boleh dipakai melayani
// permintaan portal ASM — kalau boleh, pemisahan yang dijaga di tingkat koneksi bocor
// lewat pintu belakang.
//
// # Ia OPSIONAL, dan ketiadaannya bukan galat
//
// Portal yang belum punya blok ANEKA tetap berjalan penuh; yang hilang hanyalah kolom
// laporan yang memang membutuhkannya, dan kolom itu dikosongkan serta ditandai. Ini
// perlakuan yang sama dengan blok Kasir dan SMTP di .env.example.
const anekaPrefix = "ANEKA_"

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
	VirtualAccount  VirtualAccount
	SMTP            SMTP
	DocumentStorage DocumentStorage

	// PrimaryPortal adalah alias portal yang basis datanya melayani hal-hal yang
	// dibutuhkan SEBELUM pengguna memilih portal: daftar portal (M_PORTAL_PNC),
	// alamat layanan HCQ (GCNM_CONNECT_REST), login non-karyawan (M_LOGIN_PNC), dan
	// tabel sesi.
	//
	// Keputusan Work Owner 2026-09-16, ditandai **sementara**: kelak portal utama
	// mungkin ditentukan per login dari tabel. Karena itu nilainya dibaca dari
	// konfigurasi, bukan ditulis di kode.
	PrimaryPortal string

	// DevelopmentReinsurerLogin mendaftarkan SATU login sebagai mitra reasuransi pada
	// data contoh layar Inbox PLA DLA (`MENU_ID 45`).
	//
	// # Kenapa isian ini ada
	//
	// Layar itu menyaring klaim menurut `POOLDATA.T_REINSURER.LOGIN`, dan penolakannya
	// terhadap login yang tidak terdaftar adalah PERILAKU YANG BENAR — bukan kerusakan.
	// Akibatnya, di lingkungan pengembangan layar itu tidak dapat dilihat sama sekali:
	// login pengembang adalah pegawai internal, dan pegawai internal memang ditolak.
	//
	// Isian ini menjawabnya tanpa melemahkan penyaringnya: ia hanya mengganti login pada
	// DATA CONTOH, bukan melonggarkan aturan.
	//
	// # Dua jalur, karena ada DUA sumber data
	//
	// Yang menentukan jalur mana yang berlaku bukan `PENYIMPANAN` sendirian, melainkan
	// syarat yang sama dengan `needsOracle` di `cmd/claimpnc`: `IDENTITAS_ADAPTER=hcq`
	// menarik SELURUH modul ke Oracle, termasuk modul ini, walau `PENYIMPANAN=memori`.
	//
	//	data contoh  → `PENYIMPANAN != oracle` DAN `IDENTITAS_ADAPTER != hcq`
	//	Oracle nyata → selain itu
	//
	// Pada data contoh, isian ini cukup sendirian: ia mengganti login mitra pada data
	// contoh, tanpa melonggarkan penyaringnya.
	//
	// Pada Oracle nyata ia TIDAK cukup, dan sebabnya bukan selera: kueri layar menyaring
	// dengan `T_REINSURER.LOGIN` secara langsung. Memberi login ini sekadar "izin masuk"
	// akan membuka layar yang SELURUH tabnya kosong — keadaan yang tidak dapat dibedakan
	// dari penyaring yang rusak. Karena itu Oracle menuntut
	// `DevelopmentReinsurerPartner`.
	//
	// Kosong berarti data contoh memakai login bawaannya, dan layar menolak login lain.
	DevelopmentReinsurerLogin string

	// DevelopmentReinsurerPartner adalah login mitra NYATA yang dipinjam
	// `DevelopmentReinsurerLogin` saat modul ini membaca Oracle.
	//
	// Selama dipinjam, layar berjalan PERSIS sebagai mitra itu: penyaringnya tidak
	// disentuh sama sekali, dan yang berpindah hanyalah login yang dicocokkan. Itulah
	// sebabnya ia menguji jalur yang sama dengan yang dipakai mitra sungguhan — berbeda
	// dari sekadar melonggarkan gerbangnya.
	//
	// # Hanya di `APP_ENV=development`
	//
	// Bukan karena kebocoran baca, melainkan karena layar ini MENULIS: balasan komunikasi
	// mengubah `POOLDATA.M_KOMUNIKASI_PNC` — tabel milik Pega (`P-1`) — dan di bawah
	// peminjaman, balasan itu akan tercatat atas nama mitra yang dipinjam. Jejak audit
	// adalah satu-satunya kontrol pengimbang yang tersisa (`D-59`), sehingga memalsukan
	// pelakunya adalah hal terakhir yang boleh terjadi di luar lingkungan pengembangan.
	DevelopmentReinsurerPartner string

	// KomiteTanpaPenyaringOperator mematikan penyaring pemilik pada layar Inbox Komite.
	//
	// # Kenapa isian ini ada
	//
	// Inbox Komite menyaring `PXASSIGNEDOPERATORID` terhadap login pemanggil, dan
	// penyaring itu BENAR — inbox adalah daftar pekerjaan seseorang. Tetapi pemetaan
	// identitas HCC/HCQ ke `OPERATOR_ID` belum ada (`ADR-0024`), sehingga login pengembang
	// tidak cocok dengan satu pun operator di data warisan dan layarnya kosong untuk semua
	// orang — tanpa satu pun galat yang menjelaskannya.
	//
	// Diminta Work Owner 2026-09-29 supaya isi Inbox Outstanding dapat dilihat lebih dulu.
	//
	// # Apa yang ia lakukan, dinyatakan terang
	//
	// Ia TIDAK meminjam identitas orang lain seperti `DevelopmentReinsurerPartner`. Ia
	// mematikan penyaringnya seluruhnya: daftarnya menjadi SELURUH antrean komite
	// perusahaan, beserta nama tertanggung dan nomor polis milik pekerjaan orang lain.
	//
	// Karena itu ia lebih keras dijaga, bukan lebih longgar:
	//
	//   - MENOLAK berjalan di luar `APP_ENV=development`.
	//   - Respons daftar membawa penandanya, dan layar WAJIB menyatakannya.
	//   - Di lapisan domain ia tetap harus diminta lewat `InboxFilter.AllOperators`;
	//     operator yang kebetulan kosong tidak pernah berarti "semua".
	//
	// Kosong atau `false` berarti penyaring pemilik berlaku seperti biasa.
	KomiteTanpaPenyaringOperator bool

	// Portal memetakan alias portal ke parameter koneksinya. Isinya ditemukan dengan
	// memindai lingkungan, bukan dari daftar tetap.
	Portal map[string]Database

	// Aneka memetakan alias portal ke parameter koneksi KEDUA-nya, bila ada.
	//
	// Kuncinya alias portal yang sama dengan Portal di atas — koneksi kedua selalu
	// MILIK sebuah portal, tidak pernah berdiri sendiri. Alias yang muncul di sini
	// tetapi tidak di Portal adalah salah ketik di .env, dan Validate menyebutkannya.
	//
	// Kosong berarti tidak ada portal yang punya koneksi kedua; itu keadaan yang sah.
	// Lihat anekaPrefix.
	Aneka map[string]Database

	// BulkSelectExcludedBusinesses adalah kode lini bisnis yang TIDAK ikut terpilih oleh
	// tombol "Pilih semua" pada layar Daftar Tipe Dokumen Bisnis.
	//
	// Di Pega kelimanya ditulis langsung di dalam rule
	// (`Activity/SetAllBusiness-Act.xml:984`) sebagai syarat yang mengeluarkan baris dari
	// perulangan — kelimanya lini MBU, yang aturan dokumennya tidak dikelola layar ini.
	//
	// `D-15` melarang nilai bisnis di dalam kode, sehingga daftarnya pindah ke sini:
	// perilakunya sama persis dengan Pega, tetapi kelima kodenya dapat diubah tanpa
	// menyentuh kode. Ia BELUM dapat diubah pengguna bisnis sendiri — masternya belum
	// ada — dan itu keadaan yang sama dengan XOLCommitteeRecipients di atas.
	//
	// Ditulis sebagai daftar dipisah koma di BISNIS_DIKECUALIKAN_PILIH_SEMUA. Kosong
	// berarti "Pilih semua" benar-benar memilih semuanya.
	BulkSelectExcludedBusinesses []string
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

// Dua nilai sah untuk VIRTUAL_ACCOUNT_ADAPTER.
//
// Bawaannya SELALU `tiruan`, termasuk di produksi. Ini satu-satunya adapter di aplikasi
// yang bawaannya tertutup meski seluruh kredensialnya sudah terisi — lihat VirtualAccount.
const (
	VirtualAccountAdapterFake = "tiruan"
	VirtualAccountAdapterPega = "pega"
)

// VirtualAccount memuat sakelar dan kredensial penerbit rekening virtual Master Recovery.
//
// # Kenapa ia punya sakelar sendiri, tidak mengikuti PENYIMPANAN seperti adapter lain
//
// Karena menyalakannya MENERBITKAN REKENING SUNGGUHAN. Adapter lain yang keliru menyala
// paling jauh membaca data yang salah; yang ini meninggalkan rekening bank nyata yang
// tidak diminta siapa pun, pada sistem yang dipakai orang lain.
//
// Semula pilihannya mengikuti ada-tidaknya koneksi Oracle — dan itu berbahaya: begitu
// aplikasi dijalankan dengan `PENYIMPANAN=oracle`, penerbitan sungguhan ikut menyala
// tanpa ada yang memutuskannya. Keputusan Work Owner 2026-09-29: **jangan dibuka dulu**,
// dan pembukaannya harus berupa tindakan yang disengaja.
//
// # Yang dituntut rule aslinya
//
// `Connect REST/VirtualAccountClaimsPNC-ConnectREST.xml` ber-`pyUseAuthentication=true`
// dengan `pyAuthenticationProfile = LELANG`. Profil itu TIDAK ADA di export — hanya
// namanya yang dirujuk — sehingga kredensialnya harus diminta ke Tim Pega/Infra.
//
// Kredensialnya SENGAJA tidak memakai ulang HCQ: keduanya profil berbeda, dan memakai
// kredensial HCQ untuk layanan ini akan gagal dengan cara yang membingungkan.
type VirtualAccount struct {
	// Adapter bernilai `tiruan` atau `pega`. Apa pun selain `pega` diperlakukan sebagai
	// `tiruan` — supaya salah ketik menutup, bukan membuka.
	Adapter string

	// User dan Password adalah kredensial profil autentikasi LELANG.
	User     string
	Password string

	Timeout time.Duration
}

// Live menyatakan penerbit SUNGGUHAN yang dipakai.
//
// Ia menuntut ketiganya sekaligus: sakelar disetel `pega`, dan kedua kredensialnya terisi.
// Sakelar tanpa kredensial TIDAK membukanya — permintaan yang pasti ditolak layanan lebih
// buruk daripada tidak dikirim sama sekali, karena kegagalannya tampak seperti gangguan
// jaringan.
func (v VirtualAccount) Live() bool {
	return v.Adapter == VirtualAccountAdapterPega &&
		strings.TrimSpace(v.User) != "" && strings.TrimSpace(v.Password) != ""
}

// Alamat baku kedua layanan dokumen, disalin dari rule Connect REST Pega.
//
// # Kenapa ada nilai baku, padahal §3.4 melarang endpoint tertanam di kode
//
// Yang §3.4 dan `R-18` larang adalah endpoint yang **tidak dapat diganti** — sebabnya dua
// Connect REST produksi yang menunjuk host sandbox tanpa seorang pun dapat mengalihkannya.
// Larangan itu tetap dipatuhi: kedua nilai di bawah **dapat ditimpa** lewat variabel
// lingkungan, per lingkungan, tanpa rilis ulang.
//
// Yang berubah hanyalah nilai bakunya. Work Owner menetapkannya 2026-09-27: *"alamat
// layanan ada di connect-rest pega"*. Tanpa nilai baku, unggah dokumen mati di setiap
// lingkungan sampai seseorang mengingat mengisi dua variabel yang tidak pernah disebut
// layar mana pun.
//
// `D-69` tidak dilanggar: host `app13` sudah tertulis di artefak yang di-commit sejak
// awal — `docs/Steering/02-BUSINESS-UNDERSTANDING.md:238`, `docs/BRD/BRD.md:929`, dan
// `D-16` sendiri. Menuliskannya di sini tidak menambah paparan apa pun.
const (
	// DefaultDocumentStorageURL — `Connect REST/UploadDokumenPNC-ConnectREST.xml`,
	// `pyBaseURL`.
	DefaultDocumentStorageURL = "https://app13.sinarmas.co.id"

	// DefaultImageConverterURL — `Connect REST/KonversiAvif-ConnectREST.xml`, `pyBaseURL`.
	//
	// **`http://`, bukan `https://`** — begitu apa adanya di rule itu. Isi berkas, termasuk
	// dokumen nasabah, melintas tanpa enkripsi dan tanpa otentikasi. Tidak diubah di sini
	// karena memaksa TLS ke layanan yang belum melayaninya akan mematikan konversi; yang
	// dilakukan adalah membuatnya dapat dialihkan ke HTTPS lewat konfigurasi begitu Tim
	// Infra menyediakannya.
	DefaultImageConverterURL = "http://aiimage.sinarmas.co.id"
)

// DocumentStorage memuat alamat kedua layanan dokumen internal.
//
// # Kenapa alamatnya di sini, bukan di GCNM_CONNECT_REST seperti HCQ
//
// Registri `POOLDATA.GCNM_CONNECT_REST` memang tempat yang lebih baik — perpindahan
// endpoint menjadi perubahan data oleh DBA, bukan rilis ulang. Tetapi diperiksa langsung
// 2026-09-26: registri itu **tidak punya baris** untuk unggah dokumen maupun konversi
// gambar. Keduanya tertanam di dalam rule Connect REST Pega.
//
// Begitu DBA menambahkan barisnya, nilai ini dapat pindah ke sana dan kedua variabel
// lingkungannya dicabut.
type DocumentStorage struct {
	// BaseURL adalah pangkal layanan penyimpanan, TANPA jalur `/api/v1/upload`.
	BaseURL string

	// ConverterURL adalah pangkal layanan konversi gambar, TANPA jalur `/convert-avif`.
	//
	// Kosongnya TIDAK menggagalkan start dan TIDAK mematikan unggah — yang gagal hanyalah
	// berkas PNG, JPG, JPEG, dan PDF, yaitu yang memang menempuh konversi. Berkas lain
	// tetap terunggah.
	ConverterURL string

	// Timeout membatasi satu unggahan maupun satu konversi. Kosong berarti 60 detik.
	Timeout time.Duration
}

// Active menyatakan konfigurasi ini cukup untuk mengunggah dokumen.
func (d DocumentStorage) Active() bool {
	return strings.TrimSpace(d.BaseURL) != ""
}

// ConverterActive menyatakan konfigurasi ini cukup untuk mengonversi gambar.
//
// Dipisahkan dari Active karena keduanya layanan yang berbeda: penyimpanan dapat hidup
// tanpa konversi, dan matinya konversi hanya menutup empat ekstensi.
func (d DocumentStorage) ConverterActive() bool {
	return strings.TrimSpace(d.ConverterURL) != ""
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

	// TKARecipients adalah mailbox penerima pemberitahuan kelengkapan dokumen klaim
	// TKA. Ditulis sebagai daftar dipisah koma di SMTP_PENERIMA_TKA.
	//
	// # Kenapa daftarnya TERPISAH dari AlertRecipients
	//
	// Keduanya punya pembaca yang berbeda. AlertRecipients adalah Tim IT, yang
	// menerima kabar bahwa sebuah integrasi gagal dan dapat memperbaikinya.
	// TKARecipients adalah pihak bisnis, yang menerima kabar bahwa dokumen asli satu
	// klaim sudah lengkap — peristiwa yang sepenuhnya normal dan tidak menuntut
	// perbaikan apa pun. Menyatukan keduanya akan mengirimi Tim IT surel setiap kali
	// seorang petugas menyelesaikan pekerjaannya.
	//
	// # Ia menggantikan penerima yang di-hardcode di sistem lama
	//
	// `Activity/SubmitTanggalLengkapTKA-Act.xml` memilih penerimanya dengan bercabang
	// pada tiga Operator ID yang tertanam di dalam rule — salah satu cabangnya menunjuk
	// akun surel pribadi di jalur produksi. Percabangan itu dicabut (`D-15`, `D-67`),
	// sebagaimana pola yang sama sudah dicabut pada `D-52` untuk penjenjangan komite.
	//
	// Kelak ia pindah ke master Penerima Notifikasi (`F-4`), yang memungkinkan
	// penerimanya diubah tanpa deploy. Sampai master itu ada, variabel lingkungan
	// adalah tempat terdekat yang memenuhi `D-15` — nilainya tidak berada di dalam kode.
	TKARecipients []string

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

// TKAActive menyatakan pemberitahuan kelengkapan dokumen TKA dapat dikirim.
//
// Ia TERPISAH dari Active(): server surel yang sama dapat terkonfigurasi untuk peringatan
// Tim IT tanpa punya penerima TKA, dan sebaliknya. Memakai satu penanda untuk keduanya akan
// membuat modul yang penerimanya belum diisi tetap mencoba mengirim ke daftar kosong.
func (s SMTP) TKAActive() bool {
	return strings.TrimSpace(s.Host) != "" &&
		s.Port > 0 &&
		strings.TrimSpace(s.From) != "" &&
		len(s.TKARecipients) > 0
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
	// Prefix adalah awalan variabel lingkungan asal koneksi ini — portalPrefix untuk
	// basis data portal, anekaPrefix untuk koneksi keduanya.
	//
	// Ia disimpan supaya Missing() menyebut NAMA VARIABEL YANG SEBENARNYA. Tanpa itu,
	// blok ANEKA yang kurang satu baris akan dilaporkan sebagai POOLDATA yang kurang —
	// dan operator memperbaiki baris yang sudah benar.
	//
	// Kosong dibaca sebagai portalPrefix, supaya Database yang dibentuk uji lama tidak
	// berubah artinya.
	Prefix string

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
	prefix := b.Prefix
	if prefix == "" {
		prefix = portalPrefix
	}

	var missing []string
	for name, value := range map[string]string{
		prefix + b.Alias + "_HOST":     b.Host,
		prefix + b.Alias + "_SERVICE":  b.Service,
		prefix + b.Alias + "_PENGGUNA": b.User,
		prefix + b.Alias + "_SANDI":    b.Password,
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
	// 30 detik: layanan penerbit VA sendiri berbicara ke bank, sehingga jawabannya wajar
	// lebih lambat daripada pemanggilan internal biasa.
	virtualAccountTimeout, err := getDuration("VIRTUAL_ACCOUNT_BATAS_WAKTU", 30*time.Second)
	if err != nil {
		issues = append(issues, err)
	}
	// 60 detik, bukan 30 seperti pemanggilan biasa: muatannya membawa berkas.
	documentStorageTimeout, err := getDuration("PENYIMPANAN_DOKUMEN_BATAS_WAKTU", 60*time.Second)
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

	// Login mitra pada data contoh. Namanya berbahasa Indonesia karena ia variabel
	// lingkungan — pengecualian `D-80`: ia dipakai berkas `.env` dan skrip deployment.
	devReinsurerLogin := strings.TrimSpace(get("REAS_LOGIN_PENGEMBANGAN", ""))
	devReinsurerPartner := strings.TrimSpace(get("REAS_MITRA_PENGEMBANGAN", ""))

	// Syarat ini HARUS sama dengan `needsOracle` di cmd/claimpnc. Versi sebelumnya
	// memeriksa `storage == StorageOracle` saja, dan itu keliru: dengan
	// `PENYIMPANAN=memori` + `IDENTITAS_ADAPTER=hcq`, modul ini membaca Oracle sementara
	// penjagaannya menyatakan semuanya baik — sehingga `REAS_LOGIN_PENGEMBANGAN`
	// diabaikan TANPA SATU PUN keluhan. Kegagalan senyap itulah yang membuat layarnya
	// tetap menolak meski isiannya sudah benar.
	usesSampleData := storage != StorageOracle && adapter != IdentityAdapterHCQ

	switch {
	case devReinsurerPartner != "" && devReinsurerLogin == "":
		issues = append(issues, fmt.Errorf(
			"REAS_MITRA_PENGEMBANGAN diisi %q sementara REAS_LOGIN_PENGEMBANGAN kosong; "+
				"tidak ada login yang meminjamnya. Isi keduanya, atau kosongkan keduanya",
			devReinsurerPartner))

	case (devReinsurerLogin != "" || devReinsurerPartner != "") && env != Development:
		issues = append(issues, fmt.Errorf(
			"REAS_LOGIN_PENGEMBANGAN/REAS_MITRA_PENGEMBANGAN hanya berlaku pada "+
				"APP_ENV=development; sekarang %q. Kosongkan keduanya", env))

	case devReinsurerLogin != "" && !usesSampleData && devReinsurerPartner == "":
		// Inilah keadaan yang kemarin lolos tanpa suara.
		issues = append(issues, fmt.Errorf(
			"REAS_LOGIN_PENGEMBANGAN diisi %q, tetapi Inbox PLA DLA membaca Oracle "+
				"sungguhan di setelan ini (PENYIMPANAN=%s, IDENTITAS_ADAPTER=%s) sehingga "+
				"isian itu sendirian tidak berpengaruh apa pun. Pilih SATU: "+
				"(a) isi REAS_MITRA_PENGEMBANGAN dengan satu login mitra yang sudah ada "+
				"di POOLDATA.T_REINSURER.LOGIN, sehingga %[1]q meminjamnya; atau "+
				"(b) setel IDENTITAS_ADAPTER=fake agar data contoh yang dipakai; atau "+
				"(c) minta DBA menambahkan %[1]q ke POOLDATA.T_REINSURER — menu Master "+
				"Reas TIDAK dapat melakukannya, ia hanya membaca",
			devReinsurerLogin, storage, adapter))
	}

	// Penyaring pemilik Inbox Komite. Namanya berbahasa Indonesia karena ia variabel
	// lingkungan — pengecualian `D-80`, sama dengan REAS_LOGIN_PENGEMBANGAN di atas.
	komiteTanpaPenyaring := isTrue(get("KOMITE_TANPA_PENYARING_OPERATOR", ""))

	// Penjagaannya SATU baris, dan sengaja tidak punya pengecualian.
	//
	// Isian ini mematikan penyaring pemilik pada sebuah daftar pekerjaan pribadi. Di luar
	// pengembangan, akibatnya adalah setiap orang yang punya sesi melihat antrean komite
	// SELURUH perusahaan beserta nama tertanggung dan nomor polisnya. Tidak ada keadaan
	// yang membuat itu benar di staging maupun produksi.
	if komiteTanpaPenyaring && env != Development {
		issues = append(issues, fmt.Errorf(
			"KOMITE_TANPA_PENYARING_OPERATOR menyala pada APP_ENV=%q; ia hanya berlaku "+
				"pada development karena mematikan penyaring pemilik Inbox Komite — "+
				"daftarnya menjadi antrean komite seluruh perusahaan. Kosongkan isian itu",
			env))
	}

	primaryPortal := strings.ToUpper(strings.TrimSpace(get("PORTAL_UTAMA", defaultPrimaryPortal)))
	portal, portalErrs := loadPortals()
	issues = append(issues, portalErrs...)

	aneka, anekaErrs := loadAneka()
	issues = append(issues, anekaErrs...)

	k := Config{
		Environment:     env,
		Address:         get("APP_ALAMAT", ":8080"),
		IdentityAdapter: adapter,
		Storage:         storage,

		DevelopmentReinsurerLogin:   devReinsurerLogin,
		DevelopmentReinsurerPartner: devReinsurerPartner,

		KomiteTanpaPenyaringOperator: komiteTanpaPenyaring,

		Session: Session{Lifetime: masaBerlaku},
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
		// Bawaannya `tiruan`, dan itu disengaja: menyalakannya menerbitkan rekening
		// sungguhan. Lihat VirtualAccount untuk alasan lengkapnya.
		VirtualAccount: VirtualAccount{
			Adapter:  get("VIRTUAL_ACCOUNT_ADAPTER", VirtualAccountAdapterFake),
			User:     strings.TrimSpace(os.Getenv("VIRTUAL_ACCOUNT_PENGGUNA")),
			Password: os.Getenv("VIRTUAL_ACCOUNT_SANDI"),
			Timeout:  virtualAccountTimeout,
		},
		DocumentStorage: DocumentStorage{
			BaseURL:      get("PENYIMPANAN_DOKUMEN_ALAMAT", DefaultDocumentStorageURL),
			ConverterURL: get("KONVERSI_GAMBAR_ALAMAT", DefaultImageConverterURL),
			Timeout:      documentStorageTimeout,
		},
		SMTP: SMTP{
			Host:            strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:            portSMTP,
			User:            strings.TrimSpace(os.Getenv("SMTP_USER")),
			Password:        os.Getenv("SMTP_PASSWORD"),
			From:            strings.TrimSpace(os.Getenv("SMTP_DARI")),
			AlertRecipients: splitAddress(os.Getenv("SMTP_PENERIMA_PERINGATAN")),
			TKARecipients:   splitAddress(os.Getenv("SMTP_PENERIMA_TKA")),

			XOLCommitteeRecipients: splitAddress(os.Getenv("XOL_PENERIMA_KOMITE")),

			Timeout: smtpTimeout,
		},
		PrimaryPortal: primaryPortal,
		Portal:        portal,
		Aneka:         aneka,

		// Nilai bawaannya adalah kelima kode yang benar-benar ada di
		// `Activity/SetAllBusiness-Act.xml:984`, sehingga tanpa konfigurasi apa pun
		// perilakunya sudah sama dengan Pega. Mengosongkannya secara sengaja tetap
		// mungkin — cukup setel variabelnya menjadi satu spasi.
		BulkSelectExcludedBusinesses: splitAddress(
			get("BISNIS_DIKECUALIKAN_PILIH_SEMUA", "10028,10164,10114,10084,10093"),
		),
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

	// Koneksi kedua selalu MILIK sebuah portal. Alias yang muncul di blok ANEKA tetapi
	// tidak punya blok POOLDATA hampir pasti salah ketik — dan tanpa pemeriksaan ini ia
	// lolos diam-diam: blok itu terbaca, tidak pernah dipakai, dan laporan yang
	// membutuhkannya tetap mengosongkan kolomnya seolah blok itu belum diisi.
	for alias := range k.Aneka {
		if _, ada := k.Portal[alias]; !ada {
			issues = append(issues, fmt.Errorf(
				"%s%s_* terbaca, tetapi portal %q tidak punya satu pun %s%s_*; "+
					"koneksi kedua selalu milik sebuah portal. Portal yang terbaca: %s",
				anekaPrefix, alias, alias, portalPrefix, alias, aliasList(k.Portal)))
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
	return loadDatabases(portalPrefix)
}

// loadAneka menemukan koneksi kedua tiap portal dengan cara yang sama.
//
// Ia memakai pemuat yang SAMA dengan portal, bukan salinannya: keduanya membaca lima
// variabel dengan arti yang sama, dan dua pemuat berarti keduanya dapat berbeda saat
// salah satu diubah — misalnya nilai baku porta atau umur koneksi.
func loadAneka() (map[string]Database, []error) {
	return loadDatabases(anekaPrefix)
}

func loadDatabases(prefix string) (map[string]Database, []error) {
	var issues []error
	result := map[string]Database{}

	for _, alias := range aliasesFromEnvironment(prefix) {
		port, err := getInt(prefix+alias+"_PORT", 1521)
		if err != nil {
			issues = append(issues, err)
		}
		maxConnections, err := getInt(prefix+alias+"_MAKS_KONEKSI", 20)
		if err != nil {
			issues = append(issues, err)
		}
		maksIdle, err := getInt(prefix+alias+"_MAKS_IDLE", 5)
		if err != nil {
			issues = append(issues, err)
		}
		umur, err := getDuration(prefix+alias+"_UMUR_KONEKSI", 30*time.Minute)
		if err != nil {
			issues = append(issues, err)
		}

		result[alias] = Database{
			Prefix:             prefix,
			Alias:              alias,
			Host:               strings.TrimSpace(os.Getenv(prefix + alias + "_HOST")),
			Port:               port,
			Service:            strings.TrimSpace(os.Getenv(prefix + alias + "_SERVICE")),
			User:               strings.TrimSpace(os.Getenv(prefix + alias + "_PENGGUNA")),
			Password:           os.Getenv(prefix + alias + "_SANDI"),
			MaxConnections:     maxConnections,
			MaxIdle:            maksIdle,
			ConnectionLifetime: umur,
		}
	}
	return result, issues
}

func aliasesFromEnvironment(prefix string) []string {
	ditemukan := map[string]bool{}
	for _, rows := range os.Environ() {
		name, _, _ := strings.Cut(rows, "=")
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, "_HOST") {
			continue
		}
		alias := strings.TrimSuffix(strings.TrimPrefix(name, prefix), "_HOST")
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
		// Disebut di log saat start supaya keadaan "penerbit VA sungguhan menyala"
		// tidak pernah menjadi hal yang baru diketahui setelah rekening terbit.
		"virtual_account_sungguhan": k.VirtualAccount.Live(),
		"smtp_aktif":                k.SMTP.Active(),
		"smtp_tka_aktif":            k.SMTP.TKAActive(),
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

// isTrue menafsirkan sebuah isian lingkungan sebagai penanda menyala.
//
// # Kenapa hanya nilai yang disebut, bukan "apa pun selain kosong"
//
// Isian yang memakainya mematikan penjagaan. "Apa pun selain kosong" akan membuat
// `KOMITE_TANPA_PENYARING_OPERATOR=false` — bentuk yang paling wajar ditulis orang untuk
// MEMATIKANNYA — justru menyalakannya.
//
// Nilai yang tidak dikenali dianggap PADAM. Kegagalan yang aman pada penanda seperti ini
// adalah tetap menjaga, bukan terlanjur membuka.
func isTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "ya", "yes", "on":
		return true
	default:
		return false
	}
}
