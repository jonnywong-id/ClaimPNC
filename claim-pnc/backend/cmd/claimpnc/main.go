// Command claimpnc adalah satu-satunya titik masuk aplikasi Claim PNC.
//
// Aplikasi ini modular monolith (ADR-0001): satu binary yang memuat seluruh modul,
// dikompilasi tanpa dependensi runtime eksternal dan menyajikan API sekaligus berkas
// statis antarmuka.
//
// Berkas ini sengaja tipis. Tugasnya hanya tiga: membaca konfigurasi, merakit adapter
// di balik setiap seam, dan menyalakan server. **Tidak ada aturan bisnis di sini.**
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memori"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/waktu"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/statusprogres"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemori "claim-pnc/internal/portal/repo/memori"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
	statusprogreshttp "claim-pnc/internal/statusprogres/http"
	statusprogresmemori "claim-pnc/internal/statusprogres/repo/memori"
	statusprogressql "claim-pnc/internal/statusprogres/repo/sqlstore"
	statusprogresusecase "claim-pnc/internal/statusprogres/usecase"
)

// berkasEnvBaku dibaca bila ada. Nilai yang sudah ada di lingkungan proses menang atas
// isinya, sehingga satu perintah dapat menimpa satu nilai tanpa menyunting berkas.
const berkasEnvBaku = ".env"

func main() {
	if err := jalankan(); err != nil {
		// Kegagalan saat start ditulis ke stderr dan menghentikan proses. Aplikasi
		// yang setengah hidup lebih berbahaya daripada aplikasi yang tidak start.
		fmt.Fprintln(os.Stderr, "gagal menjalankan aplikasi:", err)
		os.Exit(1)
	}
}

func jalankan() error {
	// Dua flag, keduanya untuk mode periksa. Aplikasi normal tidak memakai flag sama
	// sekali — seluruh konfigurasinya dari .env atau lingkungan (ADR-0025).
	modePeriksa := flag.Bool("periksa", false,
		"periksa integrasi basis data dan HCC/HCQ lalu berhenti; tidak menulis apa pun")
	loginUji := flag.String("login", "",
		"nama pengguna yang dicoba pada mode periksa; kata sandinya dibaca dari stdin")
	flag.Parse()

	if err := config.MuatBerkasEnv(berkasEnvBaku); err != nil {
		return err
	}
	konf, err := config.Muat()
	if err != nil {
		return err
	}

	if *modePeriksa {
		return periksa(konf, *loginUji, os.Stdin, os.Stdout)
	}

	logger := logging.Baru(slog.LevelInfo)
	logger.Info("konfigurasi terbaca", slog.Any("konfigurasi", konf.Ringkas()))

	rakitan, err := rakit(konf, logger)
	if err != nil {
		return err
	}
	defer rakitan.tutup()

	berkasSPA, err := spa.Berkas()
	if err != nil {
		logger.Warn("antarmuka tidak tersedia; aplikasi hanya melayani API",
			slog.String("sebab", err.Error()))
		berkasSPA = nil
	}

	// Kedua penulis ini dipakai seluruh modul supaya klien menghadapi satu bentuk
	// respons dan satu bentuk galat saja.
	//
	// Tipenya ditulis tanpa nama dengan sengaja. Setiap modul menamai tipe penulisnya
	// sendiri — authhttp.PenulisGalat, portalhttp.PenulisGalat, dan seterusnya — dan Go
	// tidak mengizinkan nilai bertipe bernama disalin ke tipe bernama lain walau
	// tanda tangannya sama. Nilai bertipe tanpa nama dapat disalin ke semuanya, sehingga
	// modul tetap tidak perlu saling mengimpor tipe.
	var tulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any) = func(
		w http.ResponseWriter, r *http.Request, status int, badan any,
	) {
		authhttp.TulisJSON(w, r, status, badan, logger)
	}

	// Galat portal dipetakan modul portal, lalu sisanya diteruskan ke pemeta modul auth.
	// Urutan pembungkusnya menentukan: yang lebih khusus memeriksa lebih dulu.
	var tulisGalat func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.DenganGalatPortal(
		authhttp.TulisGalat(logger), tulisRespon,
	)

	handlerAuth := authhttp.HandlerBaru(rakitan.auth, logger)
	handlerPortal := portalhttp.HandlerBaru(portalhttp.Opsi{
		Repo:        rakitan.portal,
		AliasSiap:   rakitan.aliasSiap,
		AliasUtama:  konf.PortalUtama,
		Logger:      logger,
		TulisRespon: tulisRespon,
		TulisGalat:  tulisGalat,
	})

	handlerStatusProgres, err := statusprogreshttp.HandlerBaru(statusprogreshttp.Opsi{
		Layanan:     rakitan.statusProgres,
		Logger:      logger,
		TulisRespon: tulisRespon,
		TulisGalat:  tulisGalat,
	})
	if err != nil {
		return err
	}

	// Bahan penentu portal aktif dipakai setiap modul bisnis yang menyentuh basis data
	// entitas. Ia dirakit sekali di sini supaya keempat modul berikutnya memakai
	// pemeriksaan yang sama persis — bukan masing-masing menafsirkannya sendiri.
	bahanPortalAktif := portalhttp.BahanPortalAktif{
		Repo:       rakitan.portal,
		AliasSiap:  rakitan.aliasSiap,
		Logger:     logger,
		TulisGalat: tulisGalat,
	}

	router := httpserver.Router(httpserver.Bahan{
		Logger:    logger,
		BerkasSPA: berkasSPA,
		PasangAPI: func(api chi.Router) {
			// Setiap modul memasang rutenya sendiri di sini. Modul berikutnya cukup
			// menambah satu baris; server tidak perlu tahu isinya.
			authhttp.Pasang(api, handlerAuth, rakitan.auth, logger)

			// Daftar portal berada di balik sesi: pemilihnya ada di dalam aplikasi,
			// bukan di layar masuk (ADR-0030, berpindah portal tanpa login ulang).
			api.Group(func(terlindungi chi.Router) {
				terlindungi.Use(authhttp.Autentikasi(rakitan.auth, tulisGalat))
				portalhttp.Pasang(terlindungi, handlerPortal)

				// Master Status Progres 1. Rutenya memasang pemeriksaan portal sendiri
				// di dalam Pasang — hanya pada rute yang benar-benar menyentuh basis
				// data entitas.
				statusprogreshttp.Pasang(terlindungi, handlerStatusProgres, bahanPortalAktif)
			})
		},
	})

	server := &http.Server{
		Addr:              konf.Alamat,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("server menyala", slog.String("alamat", konf.Alamat))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server berhenti: %w", err)
	}
	return nil
}

// rakitan memegang seluruh modul yang sudah terpasang beserta cara menutupnya.
type rakitan struct {
	auth          *usecase.Layanan
	portal        portal.Repo
	statusProgres *statusprogresusecase.Layanan
	aliasSiap     func() []string
	tutup         func()
}

// penyimpanan memegang seluruh repo yang sudah terpasang di atas sumbernya.
type penyimpanan struct {
	pengguna auth.PenggunaRepo
	sesi     auth.SesiRepo
	portal   portal.Repo

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	warisan *sqlstore.Warisan

	// pemilihStatusProgres memilih penyimpanan master status progres milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena tabelnya ada di basis data SETIAP entitas
	// (ADR-0030). Satu repo bersama akan menulis data seluruh entitas ke satu tempat —
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	pemilihStatusProgres statusprogres.PemilihRepo

	aliasSiap func() []string
	tutup     func()
}

// rakit menyusun seluruh modul di balik seam-nya masing-masing.
func rakit(konf config.Konfigurasi, logger *slog.Logger) (rakitan, error) {
	produksi := konf.Lingkungan == config.Produksi

	simpan, err := rakitPenyimpanan(konf, produksi, logger)
	if err != nil {
		return rakitan{}, err
	}

	sistemIdentitas, err := rakitIdentitas(konf, produksi, simpan.warisan)
	if err != nil {
		simpan.tutup()
		return rakitan{}, err
	}

	layanan, err := usecase.LayananBaru(usecase.Opsi{
		Identitas:       sistemIdentitas,
		PenggunaRepo:    simpan.pengguna,
		SesiRepo:        simpan.sesi,
		Jam:             waktu.JamSistem{},
		MasaBerlakuSesi: konf.Sesi.MasaBerlaku,
	})
	if err != nil {
		simpan.tutup()
		return rakitan{}, err
	}

	layananStatusProgres, err := statusprogresusecase.LayananBaru(statusprogresusecase.Opsi{
		PemilihRepo: simpan.pemilihStatusProgres,
	})
	if err != nil {
		simpan.tutup()
		return rakitan{}, err
	}

	return rakitan{
		auth:          layanan,
		portal:        simpan.portal,
		statusProgres: layananStatusProgres,
		aliasSiap:     simpan.aliasSiap,
		tutup:         simpan.tutup,
	}, nil
}

// butuhOracle menyatakan apakah koneksi basis data harus dibuka.
//
// Dua sebab yang BERBEDA, dan memisahkannya penting:
//
//   - PENYIMPANAN=oracle  → tabel CPNC_PENGGUNA dan CPNC_SESI_AKTIF hidup di sana.
//   - IDENTITAS_ADAPTER=hcq → alamat layanan HCQ (GCNM_CONNECT_REST) dan daftar login
//     non-karyawan (M_LOGIN_PNC) dibaca dari sana, tanpa menyentuh tabel CPNC_ sama
//     sekali.
//
// Menyatukan keduanya — seperti yang saya lakukan mula-mula — memaksa migrasi 0001
// selesai sebelum integrasi HCC/HCQ dapat dicoba lewat layar, padahal keduanya tidak
// saling bergantung.
func butuhOracle(konf config.Konfigurasi) bool {
	return konf.Penyimpanan == config.PenyimpananOracle ||
		konf.AdapterIdentitas == config.AdapterIdentitasNyata
}

// rakitPenyimpanan membuka koneksi portal bila diperlukan dan memasang repo di atasnya.
func rakitPenyimpanan(konf config.Konfigurasi, produksi bool, logger *slog.Logger) (penyimpanan, error) {
	if konf.Penyimpanan == config.PenyimpananMemori && produksi {
		// Sesi di memori satu instans melanggar tuntutan stateless (D-27): instans
		// kedua di belakang load balancer tidak akan mengenali sesi yang diterbitkan
		// instans pertama. Penolakannya ada di kode, bukan di nilai konfigurasi.
		return penyimpanan{}, errors.New(
			"penyimpanan memori menolak berjalan di lingkungan produksi: sesi wajib dikenali seluruh instans (D-27)")
	}

	simpan := penyimpanan{
		tutup:     func() {},
		aliasSiap: func() []string { return nil },
	}

	if butuhOracle(konf) {
		ctx, batal := context.WithTimeout(context.Background(), 30*time.Second)
		defer batal()

		kumpulan, err := db.KumpulanBaru(ctx, konf.PortalUtama, parameterPortal(konf), func(alias string, err error) {
			// Portal yang gagal dibuka dicatat tetapi tidak menghentikan aplikasi:
			// pengisian kredensial tiap entitas berjalan bertahap, dan satu entitas
			// yang belum siap tidak boleh menghalangi entitas yang sudah siap.
			logger.Warn("portal tidak tersedia",
				slog.String("portal", alias),
				slog.String("sebab", err.Error()))
		})
		if err != nil {
			return penyimpanan{}, err
		}
		logger.Info("koneksi portal terbuka", slog.Any("portal", kumpulan.Tersedia()))

		utama := kumpulan.Utama()
		simpan.warisan = sqlstore.WarisanBaru(utama)
		simpan.portal = portalsql.RepoBaru(utama)
		simpan.aliasSiap = kumpulan.Tersedia
		simpan.tutup = kumpulan.Tutup

		// Setiap permintaan memilih koneksi entitasnya sendiri. Portal yang tidak
		// dikenal atau koneksinya belum hidup menghasilkan galat dari Untuk() — TIDAK
		// pernah dialihkan ke koneksi utama sebagai cadangan.
		simpan.pemilihStatusProgres = func(alias string) (statusprogres.Repo, error) {
			koneksi, err := kumpulan.Untuk(alias)
			if err != nil {
				return nil, err
			}
			return statusprogressql.RepoBaru(koneksi), nil
		}
	} else {
		simpan.portal = portalmemori.RepoBaru(portalmemori.DaftarContoh()...)
		simpan.aliasSiap = func() []string { return []string{konf.PortalUtama} }
		simpan.pemilihStatusProgres = pemilihStatusProgresMemori(konf.PortalUtama)
	}

	switch konf.Penyimpanan {
	case config.PenyimpananOracle:
		utama := simpan.warisan.DB()
		simpan.pengguna = sqlstore.PenggunaRepoBaru(utama)
		simpan.sesi = sqlstore.SesiRepoBaru(utama)

	case config.PenyimpananMemori:
		// Catatan dan sesi di memori. Dipakai bersama IDENTITAS_ADAPTER=hcq, ini
		// memungkinkan masuk dengan kredensial SUNGGUHAN sebelum migrasi 0001
		// dijalankan DBA — yang hilang hanya ketahanan sesi terhadap restart dan
		// pengenalan sesi lintas instans.
		if konf.AdapterIdentitas == config.AdapterIdentitasNyata {
			logger.Warn("identitas nyata dengan penyimpanan memori",
				slog.String("akibat", "sesi hilang saat restart dan tidak dikenali instans lain; hanya untuk pengujian"))
		}
		simpan.pengguna = memori.PenggunaRepoBaru()
		simpan.sesi = memori.SesiRepoBaru()

	default:
		simpan.tutup()
		return penyimpanan{}, fmt.Errorf("penyimpanan %q tidak dikenal", konf.Penyimpanan)
	}

	return simpan, nil
}

// pemilihStatusProgresMemori menyusun penyimpanan master status progres di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani di sini, sejalan dengan aliasSiap pada cabang tanpa
// Oracle yang juga menyebut portal utama saja. Memilih portal lain tanpa basis data
// karena itu ditolak dengan galat yang sama seperti di produksi — perilaku penolakannya
// ikut teruji saat pengembangan, bukan hanya nanti.
func pemilihStatusProgresMemori(aliasUtama string) statusprogres.PemilihRepo {
	var kunci sync.Mutex
	simpanan := map[string]statusprogres.Repo{}

	return func(alias string) (statusprogres.Repo, error) {
		bersih := strings.ToUpper(strings.TrimSpace(alias))
		if bersih != strings.ToUpper(strings.TrimSpace(aliasUtama)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrBelumSiap, alias)
		}

		kunci.Lock()
		defer kunci.Unlock()
		if ada, sudah := simpanan[bersih]; sudah {
			return ada, nil
		}
		baru := statusprogresmemori.RepoBaru(statusprogresmemori.DaftarContoh()...)
		simpanan[bersih] = baru
		return baru, nil
	}
}

// rakitIdentitas menyusun rantai sumber identitas.
//
// Urutannya adalah aturan bisnis yang ditetapkan Work Owner 2026-09-16: HCC/HCQ lebih
// dulu untuk karyawan, lalu POOLDATA.M_LOGIN_PNC untuk non-karyawan.
func rakitIdentitas(konf config.Konfigurasi, produksi bool, warisan *sqlstore.Warisan) (auth.Identitas, error) {
	switch konf.AdapterIdentitas {
	case config.AdapterIdentitasTiruan:
		// Penolakan terhadap produksi ada di dalam provider, bukan hanya di sini —
		// satu nilai konfigurasi tidak boleh cukup untuk menyalakannya di produksi.
		return provider.TiruanBaru(produksi, nil)

	case config.AdapterIdentitasNyata:
		if warisan == nil {
			return nil, errors.New("IDENTITAS_ADAPTER=hcq menuntut koneksi basis data: alamat layanan HCQ dan daftar login non-karyawan keduanya dibaca dari sana")
		}
		hcq, err := provider.HCQBaru(provider.OpsiHCQ{
			Katalog:     warisan,
			PortalAlias: konf.PortalUtama,
			Pengguna:    konf.HCQ.Pengguna,
			KataSandi:   konf.HCQ.KataSandi,
			BatasWaktu:  konf.HCQ.Batas,
		})
		if err != nil {
			return nil, err
		}
		lokal, err := provider.LokalBaru(warisan)
		if err != nil {
			return nil, err
		}
		return provider.BerantaiBaru(
			provider.MataRantai{Nama: "hcq", Sumber: hcq},
			provider.MataRantai{Nama: "lokal", Sumber: lokal},
		)

	default:
		return nil, fmt.Errorf("adapter identitas %q tidak dikenal", konf.AdapterIdentitas)
	}
}

// parameterPortal mengubah konfigurasi portal menjadi parameter koneksi, melewati
// portal yang variabel wajibnya belum terisi.
func parameterPortal(konf config.Konfigurasi) []db.Parameter {
	alias := make([]string, 0, len(konf.Portal))
	for a := range konf.Portal {
		alias = append(alias, a)
	}
	sort.Strings(alias)

	parameter := make([]db.Parameter, 0, len(alias))
	for _, a := range alias {
		b := konf.Portal[a]
		if !b.Lengkap() {
			continue
		}
		parameter = append(parameter, db.Parameter{
			Alias:       b.Alias,
			Host:        b.Host,
			Port:        b.Port,
			Service:     b.Service,
			Pengguna:    b.Pengguna,
			KataSandi:   b.KataSandi,
			MaksKoneksi: b.MaksKoneksi,
			MaksIdle:    b.MaksIdle,
			UmurKoneksi: b.UmurKoneksi,
		})
	}
	return parameter
}
