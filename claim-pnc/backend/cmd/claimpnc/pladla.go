// Perakitan kedua modul pemberitahuan reasuransi: `MENU_ID 44` dan `MENU_ID 45`.
//
// # Kenapa berkas tersendiri
//
// `main.go` sudah 6.700 baris, sedangkan `docs/Steering/07` §2 aturan 3 menyebut berkas
// melebihi ~400 baris sebagai tanda perlu dipecah. Presedennya sudah ada dua —
// `registration.go` dan `modules.go` — dan yang tinggal di `main.go` hanyalah dua field
// beserta empat titik panggil ke berkas ini.
//
// # Kenapa KEDUANYA di satu berkas, padahal modulnya terpisah
//
// Karena keduanya bersebelahan di menu, judulnya hampir sama, dan yang paling mudah
// keliru justru perakitannya: menukar handler-nya tidak menghasilkan satu pun galat —
// hanya layar yang menampilkan antrean yang salah kepada orang yang salah.
//
// Merakit keduanya di satu tempat membuat perbedaannya terbaca berdampingan:
//
//	MENU_ID 44  inboxpladlapredla  petugas INTERNAL  dokumen yang BELUM dikirim
//	MENU_ID 45  inboxpladla        REASURADUR        dokumen yang SUDAH dikirim kepadanya
package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladlapredla"
	inboxpladlapredlanotif "claim-pnc/internal/inboxpladlapredla/notification"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"

	authhttp "claim-pnc/internal/auth/http"
	portalhttp "claim-pnc/internal/portal/http"

	inboxpladlahttp "claim-pnc/internal/inboxpladla/http"
	inboxpladlamemory "claim-pnc/internal/inboxpladla/repo/memory"
	inboxpladlasql "claim-pnc/internal/inboxpladla/repo/sqlstore"
	inboxpladlausecase "claim-pnc/internal/inboxpladla/usecase"

	inboxpladlapredlahttp "claim-pnc/internal/inboxpladlapredla/http"
	inboxpladlapredlamemory "claim-pnc/internal/inboxpladlapredla/repo/memory"
	inboxpladlapredlasql "claim-pnc/internal/inboxpladlapredla/repo/sqlstore"
	inboxpladlapredlausecase "claim-pnc/internal/inboxpladlapredla/usecase"
)

// pladlaSelectors memegang pemilih penyimpanan kedua modul.
type pladlaSelectors struct {
	// queue memilih penyimpanan antrean petugas internal (`MENU_ID 44`).
	queue inboxpladlapredla.RepoSelector

	// reinsurer memilih penyimpanan layar reasuradur (`MENU_ID 45`).
	reinsurer inboxpladla.RepoSelector
}

// pladlaServices memegang layanan kedua modul.
type pladlaServices struct {
	queue     *inboxpladlapredlausecase.Service
	reinsurer *inboxpladlausecase.Service
}

// setPLADLAOracleSelectors memasang pemilih penyimpanan di atas Oracle.
//
// Keduanya per portal, dan itu bukan pilihan: ketiga tabel pemberitahuan
// (`T_PLALIST`, `T_DLALIST`, `T_PREDLALIST`) ada di basis data SETIAP entitas
// (`ADR-0030`). Satu repo bersama akan menampilkan pemberitahuan satu badan hukum kepada
// petugas badan hukum lain — dan pada layar reasuradur, kepada MITRA badan hukum lain
// (`R-20`).
// reinsurerBorrow menyatakan SATU login pengembangan yang meminjam identitas SATU mitra
// nyata pada layar Inbox PLA DLA.
//
// # Kenapa peminjaman, bukan pelonggaran gerbang
//
// Layar ini menyaring dengan `POOLDATA.T_REINSURER.LOGIN` secara langsung — bukan dengan
// daftar kode yang dibawa terpisah. Memberi login pengembangan sekadar "izin masuk" akan
// membuka layar yang SELURUH tabnya kosong, dan kosongnya tidak dapat dibedakan dari
// penyaring yang rusak. Meminjam login mitra membuat layar berjalan pada jalur yang sama
// persis dengan yang dipakai mitra sungguhan.
//
// Nilai kosong berarti tidak ada peminjaman, dan `apply` mengembalikan loginnya apa adanya.
type reinsurerBorrow struct {
	from string // login pengembangan
	as   string // login mitra yang dipinjam
}

// newReinsurerBorrow menyusun peminjaman, dan MENCATATNYA.
//
// Catatannya bukan kelengkapan: tanpa baris ini, seorang pengembang yang melihat daftar
// terisi tidak punya cara mengetahui bahwa yang ia lihat adalah klaim milik mitra lain.
func newReinsurerBorrow(devLogin, partnerLogin string, logger *slog.Logger) reinsurerBorrow {
	from := strings.TrimSpace(devLogin)
	as := strings.TrimSpace(partnerLogin)
	if from == "" || as == "" {
		return reinsurerBorrow{}
	}

	if logger != nil {
		logger.Warn("identitas mitra DIPINJAM pada layar Inbox PLA DLA",
			slog.String("login_pengembangan", from),
			slog.String("berjalan_sebagai", as),
			slog.String("akibat",
				"seluruh layar — termasuk balasan komunikasi — berjalan atas nama mitra "+
					"itu; balasan tercatat sebagai miliknya, bukan milik pemakainya"),
			slog.String("berlaku", "hanya APP_ENV=development"))
	}

	return reinsurerBorrow{from: from, as: as}
}

// apply mengganti login pemanggil bila ia login pengembangan yang meminjam.
//
// Perbandingannya MENGABAIKAN besar-kecil huruf, dengan alasan yang sama seperti `F-3`
// menormalkan nama peran: `T_ACCESS_GROUP_PNC` dan rule Pega terbukti tidak konsisten
// soal itu, dan setelan yang gagal karena satu huruf kapital akan terbaca seperti setelan
// yang diabaikan — persis kesalahan yang penjagaan di `config` baru saja tutup.
func (b reinsurerBorrow) apply(login string) string {
	if b.from == "" || !strings.EqualFold(strings.TrimSpace(login), b.from) {
		return login
	}
	return b.as
}

func setPLADLAOracleSelectors(pool *db.Pool, store *storage) {
	store.pladla.queue = func(alias string) (inboxpladlapredla.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxpladlapredlasql.NewRepo(conn), nil
	}

	store.pladla.reinsurer = func(alias string) (inboxpladla.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxpladlasql.NewRepo(conn), nil
	}
}

// setPLADLAMemorySelectors memasang pemilih di memori untuk menjalankan tanpa basis data.
//
// Hanya portal utama yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan.
// Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan entitas, karena
// justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
func setPLADLAMemorySelectors(
	primaryAlias string,
	devReinsurerLogin string,
	logger *slog.Logger,
	store *storage,
) {
	queueStore := inboxpladlapredlamemory.NewSampleStore()
	store.pladla.queue = func(alias string) (inboxpladlapredla.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return queueStore, nil
	}

	// Data contoh layar reasuradur memakai login khusus — lihat
	// `inboxpladlamemory.SampleReinsurerLogin`. Login pengembangan biasa TIDAK terdaftar
	// sebagai reasuradur di sana, dan layarnya karena itu menjawab penolakan yang
	// menjelaskan sebabnya.
	//
	// Itu bukan kekurangan data contoh melainkan keadaan yang memang harus dapat dilihat:
	// di produksi pun petugas internal yang membuka menu ini akan menerima jawaban yang
	// sama, dan jawaban itulah yang paling perlu diuji dengan mata sendiri.
	//
	// `REAS_LOGIN_PENGEMBANGAN` menggantikan login mitra pada data contoh, sehingga layar
	// itu dapat dilihat tanpa melemahkan penyaringnya: aturan penyaringnya tidak
	// disentuh, dan login lain tetap ditolak dengan pesan yang sama.
	//
	// Ia TIDAK dapat menyentuh data sungguhan — data contoh hanya ada pada penyimpanan
	// memori, dan penyimpanan memori sudah menolak berjalan bila `APP_ENV=production`.
	reinsurerStore := inboxpladlamemory.NewSampleStoreFor(devReinsurerLogin)
	store.pladla.reinsurer = func(alias string) (inboxpladla.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return reinsurerStore, nil
	}

	if logger == nil {
		return
	}

	// Penggantiannya DICATAT, bukan diam-diam.
	//
	// Tanpa baris ini, seorang pengembang yang melihat daftar terisi tidak punya cara
	// mengetahui bahwa yang ia lihat adalah data contoh yang dialihkan ke namanya —
	// bukan bukti bahwa penyaring reasuradurnya bekerja.
	clean := strings.TrimSpace(devReinsurerLogin)
	if clean == "" {
		logger.Info("layar Inbox PLA DLA memakai login mitra bawaan",
			slog.String("login_mitra", inboxpladlamemory.SampleReinsurerLogin),
			slog.String("catatan",
				"login lain ditolak; isi REAS_LOGIN_PENGEMBANGAN untuk melihat layarnya"))
		return
	}

	logger.Warn("login mitra pada DATA CONTOH Inbox PLA DLA diganti",
		slog.String("login_mitra", clean),
		slog.String("menggantikan", inboxpladlamemory.SampleReinsurerLogin),
		slog.String("catatan",
			"hanya berlaku pada PENYIMPANAN=memori; tidak menyentuh POOLDATA.T_REINSURER"))
}

// buildPLADLAServices merakit layanan kedua modul.
func buildPLADLAServices(
	cfg config.Config,
	store storage,
	logger *slog.Logger,
) (pladlaServices, error) {
	// Logger WAJIB pada keduanya, dan alasannya berbeda di masing-masing.
	//
	// Pada `MENU_ID 44`: ketiga daftarnya bersama, tidak ada satu pun penyaring berbasis
	// pengguna, dan barisnya memuat nama tertanggung beserta nomor polis. Selama
	// `TKT-F3-004` belum selesai, jejak inilah satu-satunya kontrol pengimbang (`D-59`).
	//
	// Pada `MENU_ID 45`: yang membacanya PIHAK LUAR. Inilah satu-satunya layar yang sudah
	// dibangun yang datanya keluar dari dinding perusahaan.
	// Pengirim surat dipasang HANYA bila SMTP dikonfigurasi. Bila tidak, ia NIL.
	//
	// Modul lain memakai pengirim TIRUAN ketika SMTP belum siap — di sini itu akan
	// menjadi bencana. Tombol "SEND" menandai dokumen terkirim setelah pengirimnya
	// menjawab berhasil; pengirim tiruan yang menjawab berhasil akan membuat dokumen
	// ditandai terkirim padahal tidak satu pun surat sampai ke reasuradur, dan barisnya
	// hilang dari antrean tanpa jejak.
	//
	// Nil karena itu bukan kelalaian melainkan pagar: tombolnya menjawab alasan yang
	// menyebut isian konfigurasi mana yang kurang.
	// Syarat aktifnya diambil dari konfigurasi PENGIRIMNYA SENDIRI, bukan dari
	// `cfg.SMTP.Active()`.
	//
	// `Active()` mensyaratkan `SMTP_PENERIMA_PERINGATAN` terisi, dan syarat itu benar
	// untuk modul yang mengirim ke daftar penerima tetap. Modul ini tidak: penerimanya
	// adalah reasuradur pada BARIS DOKUMEN yang sedang dikirim, dan daftar peringatan
	// Tim IT tidak ada hubungannya.
	//
	// Memakai `Active()` di sini akan membuat tombol "SEND" tetap menolak meski host,
	// port, dan alamat pengirim sudah lengkap — penolakan yang sebabnya tidak dapat
	// ditemukan siapa pun dari membaca pesannya.
	//
	// Preseden yang sama sudah ada: `TKAActive()` terpisah dari `Active()` karena
	// alasan yang persis sama.
	konfigurasiSurat := inboxpladlapredlanotif.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		Timeout:  cfg.SMTP.Timeout,
	}

	var pengirimSurat inboxpladlapredla.Notifier
	if konfigurasiSurat.Complete() {
		pengirimSurat = inboxpladlapredlanotif.NewSender(konfigurasiSurat)

		// Pengaktifannya DICATAT, bukan diam-diam.
		//
		// Mulai saat ini satu penekanan tombol mengirim surat ke luar perusahaan dan
		// tidak dapat ditarik kembali. Baris log ini yang menjawab "sejak kapan" bila
		// kelak ada surat yang dipersoalkan.
		logger.Info("pengiriman surat PLA/DLA AKTIF",
			slog.String("host", cfg.SMTP.Host),
			slog.Int("port", cfg.SMTP.Port),
			slog.String("pengirim", cfg.SMTP.From),
			slog.String("catatan",
				"tombol SEND kini benar-benar mengirim surat ke reasuradur"))
	} else {
		logger.Warn("pengiriman surat PLA/DLA belum aktif",
			slog.String("kurang", strings.Join(konfigurasiSurat.Missing(), ", ")),
			slog.String("akibat",
				"tombol SEND menolak dengan alasan, dan TIDAK ada dokumen yang "+
					"ditandai terkirim"),
			slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, dan SMTP_DARI"))
	}

	queue, err := inboxpladlapredlausecase.NewService(
		inboxpladlapredlausecase.Options{
			RepoSelector: store.pladla.queue,
			Logger:       logger,
			Notifier:     pengirimSurat,
			ComposeBody:  inboxpladlapredlanotif.HTMLBody,
		})
	if err != nil {
		return pladlaServices{}, err
	}

	reinsurer, err := inboxpladlausecase.NewService(
		inboxpladlausecase.Options{
			RepoSelector: store.pladla.reinsurer,
			Logger:       logger,
		})
	if err != nil {
		return pladlaServices{}, err
	}

	return pladlaServices{queue: queue, reinsurer: reinsurer}, nil
}

// mountPLADLA memasang rute kedua modul di dalam kelompok yang sudah dijaga sesi.
//
// Pemeriksaan portal dipasang modulnya sendiri di dalam `Mount`.
func mountPLADLA(
	protected chi.Router,
	service pladlaServices,
	portalDeps portalhttp.ActivePortalDeps,
	writeJSON func(http.ResponseWriter, *http.Request, int, any),
	writeError func(http.ResponseWriter, *http.Request, error),
	logger *slog.Logger,
	borrow reinsurerBorrow,
) {
	// `MENU_ID 44` — antrean petugas internal.
	//
	// Jembatan pemanggilnya membawa LOGIN, bukan NIK. Ia tidak menyaring apa pun di sini;
	// yang membutuhkannya adalah jejak.
	queueHandler := inboxpladlapredlahttp.NewHandler(
		inboxpladlapredlahttp.Options{
			Service: service.queue,
			GetCaller: func(
				ctx context.Context,
			) (inboxpladlapredlahttp.Caller, bool) {
				base, ok := authhttp.CallerFromContext(ctx)
				if !ok {
					return inboxpladlapredlahttp.Caller{}, false
				}
				return inboxpladlapredlahttp.Caller{Login: base.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: inboxpladlapredlahttp.JSONWriter(writeJSON),
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxpladlapredlahttp.ErrorWriter(writeError),
		})
	inboxpladlapredlahttp.Mount(protected, queueHandler, portalDeps)

	// `MENU_ID 45` — layar reasuradur.
	//
	// Jembatan pemanggilnya WAJIB membawa LOGIN, dan di sini itu bukan soal jejak
	// melainkan soal penyaring: loginnya dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`, dan
	// itulah yang menentukan klaim mana yang terlihat.
	//
	// Memakai NIK di sini akan membuat layar menolak SETIAP reasuradur dengan pesan
	// "bukan mitra terdaftar" — jawaban yang terdengar masuk akal dan sepenuhnya salah.
	reinsurerHandler := inboxpladlahttp.NewHandler(
		inboxpladlahttp.Options{
			Service: service.reinsurer,
			GetCaller: func(ctx context.Context) (inboxpladlahttp.Caller, bool) {
				base, ok := authhttp.CallerFromContext(ctx)
				if !ok {
					return inboxpladlahttp.Caller{}, false
				}
				// NAMA ikut dibawa sejak balasan komunikasi dibangun.
				//
				// Ia TIDAK menyaring apa pun — yang menyaring tetap Login. Satu-satunya
				// pemakainya adalah `M_KOMUNIKASI_PNC.REPLYFROMNAME`, kolom yang dibaca
				// petugas internal untuk tahu balasan itu dari siapa.
				//
				// Nama yang kosong tidak menghalangi balasan; login-nya yang dipakai
				// sebagai gantinya. Lihat inboxpladla.ReplyCommand.ReplierName.
				// Peminjaman identitas mitra — hanya hidup di pengembangan.
				//
				// Ia dipasang DI SINI dan tidak di tempat lain, karena seluruh
				// penyaring modul ini — daftar, ringkasan, rincian, dokumen,
				// komunikasi — turun dari satu nilai yang sama: Login. Mengganti
				// satu nilai di satu tempat membuat layar berjalan persis sebagai
				// mitra itu; menggantinya di lapisan repo menuntut menyentuh
				// sebelas method, dan satu yang terlewat menghasilkan layar yang
				// separuh dipinjam — kelas cacat yang tidak menimbulkan galat.
				return inboxpladlahttp.Caller{
					Login: borrow.apply(base.User.Login),
					Name:  base.User.Name,
				}, true
			},
			Logger:              logger,
			WriteJSON:           inboxpladlahttp.JSONWriter(writeJSON),
			FallbackErrorWriter: inboxpladlahttp.ErrorWriter(writeError),
		})
	inboxpladlahttp.Mount(protected, reinsurerHandler, portalDeps)
}
