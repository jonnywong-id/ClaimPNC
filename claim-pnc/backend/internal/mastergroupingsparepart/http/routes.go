package mastergroupingspareparthttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat delapan isian pendek dan tidak punya daftar anak sama sekali.
// 32 KiB sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh raksasa
// ditolak sebelum memakan memori.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya. Batas di sini bukan aturan bisnis melainkan penjaga sumber
// daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan transaksi
// yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani produksi pada tabel
// yang sama (D-21).
const maxDecisionRows = 200

// Options menangani GET /master/grouping-sparepart/pilihan.
//
// Isinya DIBACA DARI BASIS DATA entitas yang sedang dibuka — panel dan tipe kendaraan ada di
// basis data setiap entitas, dan isinya berbeda. Karena itu rutenya ikut dipasangi pemeriksaan
// portal; menyajikan daftar panel satu entitas kepada entitas lain adalah kebocoran yang
// justru dicegah `R-20`.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.Service.Options(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK,
		toOptionsDTO(set.Panel, set.VehicleType, active.Alias))
}

// Sides menangani GET /master/grouping-sparepart/sisi.
//
// # Kenapa endpoint tersendiri, bukan bagian dari /pilihan
//
// Karena isinya bergantung pada panel yang dipilih, dan di Pega pun ia berjalan belakangan:
// `Activity/GetSisiPanel-Act.xml` dipanggil setelah Nama Panel dipilih, bukan saat layar
// dibuka. Memuat seluruh sisi setiap panel di muka berarti mengirim daftar yang 99 persen
// isinya tidak akan pernah dipakai.
//
// # Kenapa KEDUA parameternya diminta
//
//	?id_panel=...    ID_PANEL, diisi layar dari pilihan Nama Panel
//	?nama_panel=...  nilai yang diketik pada isian Nama Panel
//
// Keduanya dipakai bersama oleh `RDB List/GetDataSisiPanel-SQL.xml`, dan ditiru apa adanya.
// Lihat mastergroupingsparepart.LookupRepo.ListSides untuk ketidakpastian soal kolom `NAMA`
// yang keduanya saring.
func (h *Handler) Sides(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	side, err := h.Service.Sides(r.Context(), active.Alias, mastergroupingsparepart.SideKey{
		PanelID:   r.URL.Query().Get("id_panel"),
		PanelName: r.URL.Query().Get("nama_panel"),
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, toSideDTO(side, active.Alias))
}

// Part menangani GET /master/grouping-sparepart/sparepart.
//
// Ia padanan `Activity/SetDataSparepart-Act.xml`: kelima isian turunan yang muncul begitu
// Nomor Sparepart selesai diketik.
//
//	?nomor=...  NO_SPART yang diketik pengguna
//
// Ia dipanggil layar SEBELUM menyimpan, persis seperti di Pega — pengguna melihat nama
// sparepartnya muncul dan tahu nomornya sudah benar. Penyimpanan tetap mencarinya sekali lagi;
// lihat usecase.resolvePart.
func (h *Handler) Part(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	part, err := h.Service.LookupPart(r.Context(), active.Alias, r.URL.Query().Get("nomor"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, toPartDTO(part, active.Alias))
}

// Mount mendaftarkan rute modul master grouping sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Termasuk keempat rute acuan. Seluruh isinya dibaca dari basis data entitas —
// `POOLDATA.PANEL_HE`, `POOLDATA.LOKASI_PANEL_HE`, `POOLDATA.SPAREPART_HE`, dan `branddetail`
// ada di basis data setiap entitas, dan isinya berbeda. Tidak ada satu pun rutenya yang isinya
// konstanta milik aplikasi.
//
// # Urutan pendaftaran
//
// Keempat rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui hal
// itu untuk yakin bahwa `/sisi` tidak pernah terbaca sebagai sebuah ID grouping.
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
// Tidak ada DELETE terhadap grouping. Sistem lama tidak punya satu pun terhadap kedua tabel
// ini — `UpdateGroupingSparepartHE_act` tidak memuat satu pun langkah hapus — dan `D-66`
// melarang penghapusan fisik data bernilai bisnis. Grouping yang tidak lagi berlaku ditolak
// lewat keputusan, bukan dibuang.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/grouping-sparepart/pilihan", h.Options)
		perPortal.Get("/master/grouping-sparepart/sisi", h.Sides)
		perPortal.Get("/master/grouping-sparepart/sparepart", h.Part)
		perPortal.Post("/master/grouping-sparepart/keputusan", h.Decide)

		perPortal.Get("/master/grouping-sparepart", h.List)
		perPortal.Post("/master/grouping-sparepart", h.Create)
		perPortal.Get("/master/grouping-sparepart/{id}", h.Get)
		perPortal.Put("/master/grouping-sparepart/{id}", h.Save)
	})
}
