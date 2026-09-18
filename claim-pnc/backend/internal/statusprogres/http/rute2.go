package statusprogreshttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/statusprogres"
	"claim-pnc/internal/statusprogres/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// Handler2 melayani permintaan master status progres tingkat 2.
//
// Ia terpisah dari Handler tingkat 1, bukan menambah method padanya, karena keduanya
// memegang layanan yang berbeda. Menyatukannya berarti mengubah Opsi milik Handler yang
// sudah dirakit di cmd — perubahan yang membongkar, bukan menambah.
type Handler2 struct {
	layanan     *usecase.Layanan2
	logger      *slog.Logger
	tulisRespon PenulisRespon
	tulisGalat  PenulisGalat
}

// Opsi2 adalah bahan pembentuk Handler2.
type Opsi2 struct {
	Layanan *usecase.Layanan2
	Logger  *slog.Logger

	// TulisRespon dan TulisGalat disuntikkan dari cmd, bukan diimpor dari modul auth.
	TulisRespon PenulisRespon
	TulisGalat  PenulisGalat
}

// Handler2Baru membentuk handler modul master status progres tingkat 2.
func Handler2Baru(o Opsi2) (*Handler2, error) {
	if o.Layanan == nil {
		return nil, errors.New("statusprogres/http: Layanan tingkat 2 wajib diisi")
	}
	if o.TulisRespon == nil || o.TulisGalat == nil {
		return nil, errors.New("statusprogres/http: TulisRespon dan TulisGalat wajib diisi")
	}
	return &Handler2{
		layanan:     o.Layanan,
		logger:      o.Logger,
		tulisRespon: o.TulisRespon,
		tulisGalat:  o.TulisGalat,
	}, nil
}

// Daftar menangani GET /master/status-progres-2.
func (h *Handler2) Daftar(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	daftar, err := h.layanan.Daftar(r.Context(), aktif.Alias)
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	h.tulisRespon(w, r, http.StatusOK, ResponsDaftar2{
		StatusProgres2: keDaftarDTO2(daftar),
		Portal:         aktif.Alias,
	})
}

// Induk menangani GET /master/status-progres-2/induk.
//
// Rutenya DIPASANGI PortalAktif, berbeda dari daftar posisi klaim pada tingkat 1: isinya
// dibaca dari tabel tingkat 1 milik entitas yang bersangkutan, bukan daftar tetap milik
// aplikasi. Dua entitas punya Status Progres 1 yang berbeda, dan menyajikan daftar satu
// entitas kepada entitas lain adalah kebocoran yang justru dicegah R-20.
func (h *Handler2) Induk(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	daftar, err := h.layanan.DaftarInduk(r.Context(), aktif.Alias)
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	h.tulisRespon(w, r, http.StatusOK, ResponsInduk{
		Induk:  keDaftarIndukDTO(daftar),
		Portal: aktif.Alias,
	})
}

// Tambah menangani POST /master/status-progres-2.
func (h *Handler2) Tambah(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	tersimpan, err := h.layanan.Tambah(r.Context(), aktif.Alias, statusprogres.Isian2{
		Nama:    permintaan.Nama,
		IDInduk: permintaan.IDInduk,
	})
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID dan
	// NamaInduk yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.tulisRespon(w, r, http.StatusCreated, ResponsSatu2{
		StatusProgres2: keDTO2(tersimpan),
		Portal:         aktif.Alias,
	})
}

// bacaPermintaan membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler2) bacaPermintaan(w http.ResponseWriter, r *http.Request) (PermintaanSimpan2, bool) {
	var permintaan PermintaanSimpan2

	pembaca := http.MaxBytesReader(w, r.Body, batasBadanPermintaan)
	penyandi := json.NewDecoder(pembaca)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	penyandi.DisallowUnknownFields()

	if err := penyandi.Decode(&permintaan); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan2{}, false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := penyandi.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan2{}, false
	}

	return permintaan, true
}

// tulisGalatModul memakai pemetaan yang sama dengan tingkat 1.
//
// Ia disalin ke sini alih-alih dipakai bersama lewat satu method, karena keduanya
// bergantung pada penerima yang berbeda (Handler dan Handler2) sementara isinya hanya
// beberapa baris. Pemetaannya sendiri — petakanGalat — TIDAK disalin: ia satu fungsi yang
// dipakai keduanya, sehingga tidak ada dua tafsiran atas galat yang sama.
func (h *Handler2) tulisGalatModul(w http.ResponseWriter, r *http.Request, err error) {
	status, badan, dikenali := petakanGalat(err)
	if !dikenali {
		h.tulisGalat(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.Dari(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.tulisRespon(w, r, status, badan)
}

// Pasang2 mendaftarkan rute modul master status progres tingkat 2.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif — berbeda dari tingkat 1, yang menyisakan daftar
// posisi klaim di luarnya. Di sini tidak ada satu pun rute yang isinya milik aplikasi:
// daftar induk pun dibaca dari basis data entitas.
//
// # Yang TIDAK didaftarkan, dan kenapa
//
// Tidak ada PUT maupun DELETE. Sistem lama tidak memiliki satu pun pernyataan yang
// mengubah atau menghapus isi POOLDATA.GCNM_MST_PROGRESS; buktinya ada pada doc comment
// statusprogres.Repo2. Mendaftarkan rute yang tidak dapat berbuat apa-apa hanya
// memindahkan kejutannya dari layar ke API.
//
// # Kenapa jalurnya tanpa /v1
//
// Alasannya sama dengan Pasang tingkat 1: kontrak API yang ada belum memakai awalan
// versi, dan memperkenalkannya di satu modul saja akan membuat dua gaya jalur hidup
// berdampingan. Penyeragamannya dicatat sebagai utang teknis.
func Pasang2(r chi.Router, h *Handler2, bahanPortal portalhttp.BahanPortalAktif) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.PortalAktif(bahanPortal))

		// Jalur induk didaftarkan LEBIH DULU daripada rute ber-parameter apa pun pada
		// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
		// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca
		// tidak perlu mengetahui hal itu untuk yakin.
		perPortal.Get("/master/status-progres-2/induk", h.Induk)

		perPortal.Get("/master/status-progres-2", h.Daftar)
		perPortal.Post("/master/status-progres-2", h.Tambah)
	})
}
