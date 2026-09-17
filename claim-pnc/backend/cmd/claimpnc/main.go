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
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memori"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/waktu"
	"claim-pnc/internal/portal"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	rekeninghttp "claim-pnc/internal/masterrekening/http"
	rekeningkasir "claim-pnc/internal/masterrekening/kasir"
	rekeningnotif "claim-pnc/internal/masterrekening/notifikasi"
	rekeningmemori "claim-pnc/internal/masterrekening/repo/memori"
	rekeningsql "claim-pnc/internal/masterrekening/repo/sqlstore"
	rekeningusecase "claim-pnc/internal/masterrekening/usecase"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemori "claim-pnc/internal/portal/repo/memori"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
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

	handlerAuth := authhttp.HandlerBaru(rakitan.auth, logger)
	handlerPortal := portalhttp.HandlerBaru(portalhttp.Opsi{
		Repo:       rakitan.portal,
		AliasSiap:  rakitan.aliasSiap,
		AliasUtama: konf.PortalUtama,
		Logger:     logger,
		TulisRespon: func(w http.ResponseWriter, r *http.Request, status int, badan any) {
			authhttp.TulisJSON(w, r, status, badan, logger)
		},
		TulisGalat: authhttp.TulisGalat(logger),
	})

	handlerRekening := rekeninghttp.HandlerBaru(rekeninghttp.Opsi{
		Layanan: rakitan.masterRekening,
		// Jembatan satu arah dari modul auth ke modul master rekening. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		Pemanggil: func(ctx context.Context) (rekeninghttp.Pemanggil, bool) {
			konteks, ada := authhttp.KonteksPengguna(ctx)
			if !ada {
				return rekeninghttp.Pemanggil{}, false
			}
			return rekeninghttp.Pemanggil{
				Identitas: konteks.Pengguna.Identitas,
				Nama:      konteks.Pengguna.Nama,
				Email:     konteks.Pengguna.Email,
			}, true
		},
		Logger: logger,
	})

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
				terlindungi.Use(authhttp.Autentikasi(rakitan.auth, authhttp.TulisGalat(logger)))
				portalhttp.Pasang(terlindungi, handlerPortal)

				// Master rekening memuat nama, NIK, nomor rekening, dan surel pihak
				// ketiga; tidak satu pun boleh terbaca tanpa sesi.
				rekeninghttp.Pasang(terlindungi, handlerRekening)
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
	auth           *usecase.Layanan
	portal         portal.Repo
	masterRekening *rekeningusecase.Layanan
	aliasSiap      func() []string
	tutup          func()
}

// penyimpanan memegang seluruh repo yang sudah terpasang di atas sumbernya.
type penyimpanan struct {
	pengguna auth.PenggunaRepo
	sesi     auth.SesiRepo
	portal   portal.Repo

	rekening     masterrekening.Repo
	bankRekening masterrekening.BankRepo

	// rekeningDiOracle menyatakan master rekening dipasang di atas POOLDATA.LST_ACCOUNT
	// yang sungguhan, bukan di memori. Adapter tiruan yang menulis jejak karangan
	// dilarang di atasnya — lihat rakitMasterRekening.
	rekeningDiOracle bool

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	warisan *sqlstore.Warisan

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

	return rakitan{
		auth:           layanan,
		portal:         simpan.portal,
		masterRekening: rakitMasterRekening(konf, simpan, logger),
		aliasSiap:      simpan.aliasSiap,
		tutup:          simpan.tutup,
	}, nil
}

// rakitMasterRekening menyusun modul Master Rekening di balik seam-nya.
//
// Seam Kasir diisi klien HTTP nyata bila alamatnya sudah dikonfigurasi, dan tiruan bila
// belum. Perbedaannya diumumkan di log: layar yang tampak bekerja padahal pendaftaran
// ke Kasir tidak pernah terjadi adalah kegagalan yang tidak terlihat siapa pun sampai
// pembayaran pertama tertahan.
func rakitMasterRekening(konf config.Konfigurasi, simpan penyimpanan, logger *slog.Logger) *rekeningusecase.Layanan {
	var (
		sistemKasir masterrekening.Kasir
		pemberitahu masterrekening.Notifier
	)

	if konf.SMTP.Aktif() {
		pemberitahu = rekeningnotif.PengirimBaru(rekeningnotif.Konfigurasi{
			Host:      konf.SMTP.Host,
			Port:      konf.SMTP.Port,
			Pengguna:  konf.SMTP.Pengguna,
			KataSandi: konf.SMTP.KataSandi,
			Dari:      konf.SMTP.Dari,
			Kepada:    konf.SMTP.PenerimaPeringatan,
			Tenggang:  konf.SMTP.Batas,
		})
	} else {
		pemberitahu = &rekeningnotif.Tiruan{}
		logger.Warn("pengirim surel tiruan dipakai",
			slog.String("akibat", "Tim IT TIDAK diberi tahu lewat surel bila pendaftaran ke Kasir gagal"),
			slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan SMTP_PENERIMA_PERINGATAN"))
	}

	switch {
	case konf.Kasir.Aktif():
		sistemKasir = rekeningkasir.KlienBaru(rekeningkasir.Konfigurasi{
			URLDaftar:   konf.Kasir.URLDaftar,
			URLPerbarui: konf.Kasir.URLPerbarui,
			Pengguna:    konf.Kasir.Pengguna,
			Sandi:       konf.Kasir.KataSandi,
			Tenggang:    konf.Kasir.Batas,
		})

	case simpan.rekeningDiOracle:
		// KASIR TIRUAN DILARANG DI ATAS BASIS DATA SUNGGUHAN.
		//
		// Tiruan menjawab "berhasil" beserta nomor rekening Kasir karangan. Bila
		// jawaban itu ditulis ke POOLDATA.LST_ACCOUNT yang asli, kolom STS_SERVICE
		// dan ID_REKASIR akan memuat jejak pendaftaran yang tidak pernah terjadi —
		// dan tidak ada apa pun sesudahnya yang dapat membedakannya dari pendaftaran
		// yang sungguhan. Data palsu di master rekening lebih berbahaya daripada
		// langkah yang hilang.
		//
		// Seam dibiarkan nil. Usecase sudah menanganinya: rekening tetap dapat
		// disetujui komite, dan langkah pendaftaran ke Kasir dilewati tanpa
		// meninggalkan jejak apa pun.
		sistemKasir = nil
		logger.Warn("pendaftaran ke Kasir DILEWATI",
			slog.String("sebab", "alamat sistem Kasir belum dikonfigurasi, sedangkan master rekening membaca basis data sungguhan"),
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Kasir, dan tidak ada jejak Kasir yang ditulis"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))

	default:
		// Penyimpanan di memori: tiruan aman dipakai dan memang berguna, karena ia
		// membuat seluruh alur dapat dicoba tanpa basis data dan tanpa jaringan.
		sistemKasir = rekeningkasir.TiruanBaru()
		logger.Warn("sistem Kasir tiruan dipakai",
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Kasir yang sesungguhnya"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))
	}

	return rekeningusecase.LayananBaru(rekeningusecase.Opsi{
		Repo:        simpan.rekening,
		Bank:        simpan.bankRekening,
		Kasir:       sistemKasir,
		Notifier:    pemberitahu,
		Jam:         waktu.JamSistem{},
		PortalAlias: konf.PortalUtama,
	})
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
		simpan.rekening = rekeningsql.RepoBaru(utama)
		simpan.bankRekening = rekeningsql.BankRepoBaru(utama)
		simpan.rekeningDiOracle = true
		simpan.aliasSiap = kumpulan.Tersedia
		simpan.tutup = kumpulan.Tutup
	} else {
		simpan.portal = portalmemori.RepoBaru(portalmemori.DaftarContoh()...)
		simpan.rekening = rekeningmemori.RepoBaru()
		simpan.bankRekening = rekeningmemori.BankRepoBaru(rekeningmemori.DaftarBankContoh()...)
		simpan.aliasSiap = func() []string { return []string{konf.PortalUtama} }
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
