package masterreashttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// MaxKeywordLength membatasi panjang penyaring `cari`.
//
// Ia BUKAN aturan bisnis — sistem lama tidak punya kotak pencarian sama sekali di layar ini.
// Batasnya dipasang karena kata kunci masuk ke dalam pola LIKE, dan kata kunci sepanjang
// megabita hanya menghasilkan pemindaian penuh yang pasti tidak menemukan apa pun.
//
// Seratus dipilih agar sama dengan batas nama pada modul master lain: tidak ada nama
// perusahaan reasuransi, login, maupun surel yang lebih panjang dari itu, sehingga batas ini
// tidak pernah menolak pencarian yang sungguh-sungguh.
const MaxKeywordLength = 100

// List menangani GET /master/reas.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada kode reas, nama reas, login, dan email
//
// TANPA penyaring status: tabelnya tidak punya kolom persetujuan, dan layar lamanya tidak
// bertab — harness-nya memuat satu grid dan satu tombol Refresh.
//
// TANPA penyaring TYPE, meski kolomnya ada. Sistem lama menyaringnya hanya di alur PLA/DLA,
// dengan nilai yang diambil dari nomor dokumen yang sedang dikirim — bukan dari pilihan
// pengguna. Menyediakannya di sini berarti menawarkan penyaring yang nilainya sahnya sendiri
// tidak diketahui (`R-16`).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, keyword, ok := portalhttp.ActiveKeyword(w, r, MaxKeywordLength, h.writeModuleError, h.keywordTooLong)
	if !ok {
		return
	}

	list, err := h.Service.List(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, ListResponse{
		Member: toListDTO(list),
		Portal: active.Alias,
	})
}

// Mount mendaftarkan rute modul master reas.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// `POOLDATA.T_REINSURER` ada di basis data SETIAP entitas dan isinya berbeda — ia menentukan
// mitra reasuransi mana yang menerima pemberitahuan klaim badan hukum itu, beserta surel
// tujuannya. Tidak ada satu pun rute di modul ini yang isinya milik aplikasi, sehingga tidak
// ada yang dikecualikan.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan, dan itu keputusan berdasar bukti
//
// **Tidak ada POST, PUT, maupun DELETE.** Layar lamanya tidak punya jalur tulis: harness
// `DataMemberReas` memuat satu grid dan satu tombol Refresh, dan satu-satunya penulis
// `T_REINSURER` di sistem lama adalah alur PLA/DLA lewat `Database/UPDATEREAS.prc` —
// dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`, bukan layar ini.
//
// **Tidak ada GET satu baris.** Tidak ada layar detail di sistem lama, seluruh kolomnya muat
// di dalam grid, dan kunci alaminya tiga kolom sehingga harus dipaksakan ke jalur URL.
//
// Bila kelak terbukti layar lamanya punya tombol simpan — section gridnya memang tidak ada
// di export (`R-16`) — yang perlu ditambahkan adalah Repo.Insert/Update beserta rutenya.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	portalhttp.MountGets(r, portalDeps,
		portalhttp.Route{Path: "/master/reas", Handler: h.List},
	)
}

// keywordTooLong menjawab kata pencarian yang melampaui MaxKeywordLength.
func (h *Handler) keywordTooLong(w http.ResponseWriter, r *http.Request) {
	h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: "Kata pencarian terlalu panjang."})
}
