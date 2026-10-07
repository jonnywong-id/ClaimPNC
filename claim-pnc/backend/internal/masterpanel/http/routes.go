package masterpanelhttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat sepuluh isian pendek ditambah daftar lokasi yang dibatasi
// masterpanel.MaxLocationRows baris — dan setiap barisnya hanya dua nilai pendek. 64 KiB
// sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh raksasa ditolak
// sebelum memakan memori.
const maxRequestBody = 64 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya: `SetApprovalAllMaster` menelusuri seluruh baris
// bercentang berapa pun jumlahnya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani
// produksi pada tabel yang sama (D-21).
const maxDecisionRows = 200

// Decide menangani POST /master/panel/keputusan.
//
// Ia padanan `Activity/SetApprovalAllMaster` beserta layarnya
// `Section/ApprovalMasterPanelHE-Section.xml`: daftar baris bercentang, satu status, dan
// satu catatan, ditetapkan sekaligus.
//
// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
//
// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai
// peristiwa, bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan
// membuat keputusan yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.Caller(r.Context())
	if !known {
		h.WriteError(w, r, errors.New("masterpanel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, masterpanel.OneViolation("id_panel",
			"Terlalu banyak panel dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := masterpanel.ApprovalStatus(request.Status)
	changed, err := h.Service.Decide(r.Context(), active.Alias, request.ID, status, request.Reason,
		usecase.Actor{Login: by.Login}, h.Logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, DecisionResponse{
		Changed:     changed,
		Status:      string(status),
		StatusLabel: status.Label(),
		Portal:      active.Alias,
	})
}

// Options menangani GET /master/panel/pilihan.
//
// Isinya KONSTANTA — kelima nama lokasi dan ketiga sandi sisi ditanam di
// `Activity/SetLokasiSisiPanel-Act.xml`, bukan dibaca dari tabel acuan mana pun.
//
// Ia tetap disajikan lewat endpoint supaya layar tidak memuat salinan ketiganya:
// satu-satunya daftar pilihan modul ini yang artinya benar-benar diketahui tidak boleh
// hidup di dua tempat. Karena isinya tidak menyentuh basis data, ia TIDAK menuntut portal
// — lihat Mount.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	h.WriteResponse(w, r, http.StatusOK, optionsDTO())
}

// readRequest mengurai badan JSON dengan aturan yang sama seperti mesin bersama.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	return h.ReadRequest(w, r, target)
}

// Mount mendaftarkan rute modul master panel.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Yang dipasangi pemeriksaan portal, dan yang tidak
//
// SELURUH rute data dipasangi PortalAktif: POOLDATA.PANEL_HE dan LOKASI_PANEL_HE ada di
// basis data setiap entitas, dan menyajikan daftar satu entitas kepada entitas lain adalah
// kebocoran yang justru dicegah R-20.
//
// `/pilihan` TIDAK, dan itu bukan kelalaian: isinya konstanta yang ditanam di dalam
// activity Pega, bukan bacaan basis data mana pun. Memasangi pemeriksaan portal padanya
// akan membuat form tidak dapat menggambar dropdown-nya sebelum portal dipilih —
// penolakan yang tidak melindungi apa pun. Perlakuannya sama dengan daftar Kategori pada
// Master Pasal Kerugian.
//
// # Urutan pendaftaran
//
// Kedua rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}`
// pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu
// mengetahui hal itu untuk yakin bahwa `/keputusan` tidak pernah terbaca sebagai sebuah ID
// panel.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan
//
// Tidak ada DELETE terhadap panel. Sistem lama tidak punya satu pun terhadap tabel induk,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis — panel yang tidak lagi
// dipakai ditolak atau ditandai lewat STS_AKTIF, bukan dibuang.
//
// Baris LOKASI memang dibuang saat penyimpanan, dan itu pertentangan dengan `D-66` yang
// dinyatakan terbuka di banner masterpanel.sql — bukan disembunyikan di balik ketiadaan
// endpoint DELETE.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Get("/master/panel/pilihan", h.Options)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Post("/master/panel/keputusan", h.Decide)

		perPortal.Get("/master/panel", h.List)
		perPortal.Post("/master/panel", h.Create)
		perPortal.Get("/master/panel/{id}", h.Get)
		perPortal.Put("/master/panel/{id}", h.Save)
	})
}
