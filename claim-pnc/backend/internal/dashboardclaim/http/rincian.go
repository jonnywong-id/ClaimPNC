package dashboardclaimhttp

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// claimDetailDTO adalah rincian satu klaim.
type claimDetailDTO struct {
	KlaimID      string `json:"klaim_id"`
	NomorKlaim   string `json:"nomor_klaim"`
	StatusProses string `json:"status_proses"`
	StatusKlaim  string `json:"status_klaim"`
	PICTeknik    string `json:"pic_teknik"`
	AdminPNC     string `json:"admin_pnc"`
	Didaftarkan  string `json:"didaftarkan_pada"`

	// Dokumen adalah isi klaim apa adanya dari `POOLDATA.JSON_KLAIM.DATA_JSON`.
	//
	// Diteruskan UTUH, tidak dipetakan menjadi field bernama di sini. Bentuk dokumen itu
	// belum pernah diperiksa (`R-08`), dan memetakannya berarti menyatakan bentuk yang belum
	// terbukti — isian yang namanya ternyata berbeda akan hilang tanpa satu pun tanda.
	//
	// Layar yang memilih jalur mana yang digambarnya, dan jalur yang tidak ada dapat
	// dibedakan dari jalur yang ada tetapi kosong.
	Dokumen map[string]any `json:"dokumen"`

	Portal string `json:"portal"`
}

// ClaimDetail melayani `GET /dashboard-claim/klaim/{klaim_id}`.
//
// # Kenapa rute tersendiri, bukan bagian dari daftar
//
// Di Pega, klik nomor klaim menjalankan `setDataViewKlaim_Act` lalu membuka harness
// `ViewTempDetailClaim` sebagai **popup** — pengambilan datanya terpisah dari gridnya, dan
// hanya terjadi saat diklik. Menyertakan dokumen JSON pada setiap baris daftar akan membawa
// puluhan dokumen klaim pada setiap pemuatan halaman.
func (h *Handler) ClaimDetail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Kunci dibaca dari jalur dan DI-DECODE: PZINSKEY memuat spasi
	// (`ASM-FW-GCNMFW-WORK PNC-133`), dan spasi itu ter-encode di alamatnya.
	raw := chi.URLParam(r, "klaim_id")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		decoded = raw
	}

	detail, err := h.service.ClaimDetail(r.Context(), usecase.ClaimDetailQuery{
		PortalAlias: active.Alias,
		ClaimID:     strings.TrimSpace(decoded),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, claimDetailDTO{
		KlaimID:      detail.ClaimID,
		NomorKlaim:   detail.ClaimNumber,
		StatusProses: detail.ProcessStatus,
		StatusKlaim:  detail.ClaimStatus,
		PICTeknik:    detail.TechnicalPIC,
		AdminPNC:     detail.AdminPNC,
		Didaftarkan:  detail.RegisteredAt,
		Dokumen:      dokumenAman(detail.Document),
		Portal:       active.Alias,
	})
}

// dokumenAman menjamin badan jawaban membawa objek, bukan `null`.
//
// Layar membedakan "dokumen kosong" dari "jalur tidak ada"; `null` memaksanya menjaga satu
// keadaan lagi yang artinya sama dengan kosong.
func dokumenAman(d map[string]any) map[string]any {
	if d == nil {
		return map[string]any{}
	}
	return d
}
