package komitehttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/usecase"
)

// DateFormat adalah bentuk tanggal pada parameter kueri: `2026-09-20`.
//
// Bentuk ISO, bukan `dd/mm/yyyy` warisan. Yang terakhir tidak dapat diurutkan sebagai
// teks dan artinya berbeda antar bahasa — dua sifat yang sama-sama tidak diinginkan pada
// sebuah kontrak.
const DateFormat = "2006-01-02"

// maxDecisionBody membatasi ukuran badan permintaan keputusan.
//
// Badannya hanya dua field, dan catatannya dibatasi 1.000 karakter. Batas ini menutup
// permintaan yang sengaja dibuat besar agar menghabiskan memori — pemeriksaan panjang
// catatan baru berjalan SETELAH badan terbaca seluruhnya, sehingga ia tidak dapat
// menggantikan batas ini.
const maxDecisionBody = 64 << 10

// InboxCaller adalah identitas orang yang membuka Inbox Komite.
//
// # Kenapa Login, bukan Identity
//
// Inbox disaring terhadap `PXASSIGNEDOPERATORID` pada worklist Pega dan `OPERATOR_ID`
// pada master ambang. Keduanya berisi nama seperti `ELLENSUPRIYATI`, bukan NIK.
//
// Work Owner menetapkan kunci pencocokannya adalah **login yang DIKETIK pengguna**
// (`docs/keputusan-implementasi.md` §16.5), dan `Profile.Login` memang membawanya untuk
// kedua populasi pengguna — HCQ memantulkan kembali login yang dikirim.
//
// Konsekuensi yang harus disadari: bila `OPERATOR_ID` di data warisan ternyata BUKAN
// login yang diketik, inbox akan kosong untuk semua orang — dan kosong itu tidak muncul
// sebagai galat. Karena itu respons memantulkan kembali operator yang dipakai menyaring,
// supaya sebab kosongnya dapat dibedakan (lihat InboxListResponse.Operator).
type InboxCaller struct {
	Login string
	Name  string
}

// InboxService adalah bagian usecase yang dibutuhkan handler ini.
//
// Dinyatakan sebagai antarmuka sempit di paket yang MEMAKAInya, bukan diimpor dari
// usecase, supaya handler dapat diuji tanpa membentuk seluruh layanan beserta kedua
// penyimpanannya.
type InboxService interface {
	Inbox(ctx context.Context, f komite.InboxFilter) (usecase.InboxResult, error)
	Case(ctx context.Context, caseID string, operator string) (komite.CommitteeCase, error)
	Decide(ctx context.Context, cmd komite.DecisionCommand, actor usecase.Actor) (komite.CommitteeCase, error)
}

// InboxHandler melayani layar Inbox Komite.
type InboxHandler struct {
	service InboxService
	caller  func(context.Context) (InboxCaller, bool)
	logger  *slog.Logger

	writeResponse JSONWriter
	writeError    ErrorWriter
}

// InboxHandlerOptions adalah bahan pembentuk InboxHandler.
type InboxHandlerOptions struct {
	Service InboxService

	// Caller adalah jembatan satu arah dari modul auth. Ia dipasang saat perakitan di
	// cmd/claimpnc, sehingga modul ini tidak pernah tahu bagaimana sesi bekerja dan
	// kedua modul tetap tidak saling mengimpor.
	Caller func(context.Context) (InboxCaller, bool)

	Logger              *slog.Logger
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewInboxHandler membentuk handler Inbox Komite.
func NewInboxHandler(o InboxHandlerOptions) *InboxHandler {
	return &InboxHandler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/komite/inbox.
//
// Ia menggantikan TIGA kueri sistem lama sekaligus — `GetKomitePAOutstanding`,
// `GetKomitePAditerima`, dan `ShowKomiteTerimaTolakNonMBU` — yang di sana menjadi tiga
// rule terpisah karena penyaringnya dirangkai ke dalam teks SQL lewat `{ASIS:...}`.
// Perangkaian itu sekaligus celah injeksi (`K-29`); di sini seluruh nilai lewat parameter
// binding tanpa perkecualian.
func (h *InboxHandler) List(w http.ResponseWriter, r *http.Request) {
	caller, signedIn := h.callerFrom(r)
	if !signedIn {
		h.writeError(w, r, komite.ErrNotAssigned)
		return
	}

	filter, err := inboxFilterFrom(r, caller.Login)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.service.Inbox(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	normalized := filter.Normalize()
	cases := make([]CommitteeCaseDTO, 0, len(result.Cases))
	for _, c := range result.Cases {
		cases = append(cases, toCommitteeCaseDTO(c, caller.Login, result.Now))
	}

	h.writeResponse(w, r, http.StatusOK, InboxListResponse{
		Cases:  cases,
		Total:  result.Total,
		Offset: normalized.Offset,
		Limit:  normalized.Limit,
		Summary: InboxSummaryDTO{
			Outstanding: result.Summary.Outstanding,
			Accepted:    result.Summary.Accepted,
			Rejected:    result.Summary.Rejected,
		},
		Kind:     string(normalized.Kind),
		Operator: normalized.Operator,
		Now:      result.Now.UTC().Format(time.RFC3339),
	})
}

// Detail menangani GET /api/komite/inbox/{nomor}.
func (h *InboxHandler) Detail(w http.ResponseWriter, r *http.Request) {
	caller, signedIn := h.callerFrom(r)
	if !signedIn {
		h.writeError(w, r, komite.ErrNotAssigned)
		return
	}

	found, err := h.service.Case(r.Context(), chi.URLParam(r, "nomor"), caller.Login)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	now := time.Now().UTC()
	h.writeResponse(w, r, http.StatusOK, CommitteeCaseResponse{
		Case: toCommitteeCaseDTO(found, caller.Login, now),
		Now:  now.Format(time.RFC3339),
	})
}

// Decide menangani POST /api/komite/inbox/{nomor}/keputusan.
//
// # Kenapa POST, berbeda dari seluruh rute modul ini yang lain
//
// Ia MENGUBAH keadaan dan memicu pencatatan permanen. Ketiga rute lain modul Komite
// adalah GET karena tidak satu pun menyentuh baris mana pun — perbedaan yang `D-73`
// tetapkan: aksi bisnis dimodelkan sebagai peristiwa lewat POST justru karena ia punya
// invarian dan meninggalkan jejak.
func (h *InboxHandler) Decide(w http.ResponseWriter, r *http.Request) {
	caller, signedIn := h.callerFrom(r)
	if !signedIn {
		h.writeError(w, r, komite.ErrNotAssigned)
		return
	}

	var body DecisionRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxDecisionBody))

	// Field yang tidak dikenali DITOLAK, bukan diabaikan. Klien yang mengirim `jenjang`
	// atau `pada` harus tahu bahwa keduanya tidak dipakai — mengabaikannya diam-diam
	// akan membuatnya mengira ia menentukan sesuatu yang sebenarnya ditetapkan server.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	updated, err := h.service.Decide(r.Context(), komite.DecisionCommand{
		CaseID: chi.URLParam(r, "nomor"),
		Kind:   komite.DecisionKind(body.Decision),
		Note:   body.Note,
	}, usecase.Actor{Login: caller.Login, Name: caller.Name})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	now := time.Now().UTC()
	h.writeResponse(w, r, http.StatusOK, CommitteeCaseResponse{
		Case: toCommitteeCaseDTO(updated, caller.Login, now),
		Now:  now.Format(time.RFC3339),
	})
}

func (h *InboxHandler) callerFrom(r *http.Request) (InboxCaller, bool) {
	if h.caller == nil {
		return InboxCaller{}, false
	}
	caller, found := h.caller(r.Context())
	if !found || strings.TrimSpace(caller.Login) == "" {
		return InboxCaller{}, false
	}
	return caller, true
}

// inboxFilterFrom membaca penyaring dari parameter kueri.
//
// Tanggal yang tidak dapat diurai DITOLAK sebagai validasi, bukan diabaikan. Penyaring
// yang gagal terurai lalu diam-diam dianggap kosong akan menampilkan SELURUH riwayat
// kepada seseorang yang mengira ia sedang melihat satu minggu.
func inboxFilterFrom(r *http.Request, operator string) (komite.InboxFilter, error) {
	params := r.URL.Query()

	filter := komite.InboxFilter{
		Operator: operator,
		Kind:     komite.InboxKind(strings.TrimSpace(params.Get("kotak"))),
		Search:   params.Get("cari"),
		Offset:   atoiOrZero(params.Get("lewati")),
		Limit:    atoiOrZero(params.Get("batas")),
	}

	var violations []komite.Violation

	if raw := strings.TrimSpace(params.Get("dari")); raw != "" {
		parsed, err := time.ParseInLocation(DateFormat, raw, time.UTC)
		if err != nil {
			violations = append(violations, komite.Violation{
				Field:   "dari",
				Message: "Tanggal harus berbentuk YYYY-MM-DD. Contoh: 2026-09-20.",
			})
		} else {
			filter.DateFrom = parsed
		}
	}
	if raw := strings.TrimSpace(params.Get("sampai")); raw != "" {
		parsed, err := time.ParseInLocation(DateFormat, raw, time.UTC)
		if err != nil {
			violations = append(violations, komite.Violation{
				Field:   komite.FieldDateTo,
				Message: "Tanggal harus berbentuk YYYY-MM-DD. Contoh: 2026-09-20.",
			})
		} else {
			filter.DateTo = parsed
		}
	}

	if err := komite.NewValidationError(violations); err != nil {
		return komite.InboxFilter{}, err
	}
	return filter, nil
}

// atoiOrZero mengurai bilangan dan mengembalikan nol bila gagal.
//
// Paginasi yang cacat TIDAK ditolak, berbeda dari tanggal, dan perbedaannya disengaja:
// `lewati=abc` tidak mengubah APA yang ditampilkan, hanya dari mana halamannya dimulai,
// sehingga jatuh ke halaman pertama adalah pemulihan yang benar. Tanggal yang cacat
// mengubah apa yang ditampilkan, dan karena itu ditolak.
func atoiOrZero(text string) int {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// errMalformedBody dipetakan ke 400 — bentuk permintaannya yang salah, bukan isinya.
var errMalformedBody = errors.New("komite: badan permintaan tidak dapat dibaca")

func toCommitteeCaseDTO(c komite.CommitteeCase, viewer string, now time.Time) CommitteeCaseDTO {
	return CommitteeCaseDTO{
		CaseID:      c.CaseID,
		ClaimNumber: c.ClaimNumber,

		PolicyNumber:     c.PolicyNumber,
		InsuredName:      c.InsuredName,
		BusinessName:     c.BusinessName,
		SourceOfBusiness: c.SourceOfBusiness,
		BranchName:       c.BranchName,
		GroupPanel:       c.GroupPanel,
		ClaimPIC:         c.ClaimPIC,

		CommitteeDate: formatTime(c.CommitteeDate),
		CreatedAt:     formatTime(c.CreatedAt),
		AgingDays:     c.AgingDays(now),

		WorkStatus:    c.WorkStatus,
		CommitteeKind: c.CommitteeKind,

		ClaimValue:    c.ClaimValue.String(),
		ASMShareValue: c.ASMShareValue.String(),
		ORValue:       c.ORValue.String(),

		CommitteeNote: c.CommitteeNote,

		HasAIAssessment: c.HasAIAssessment,
		AIResult:        c.AIResult,
		AINoteAccepted:  c.AINoteAccepted,
		AINoteRejected:  c.AINoteRejected,
		AIAssessedAt:    formatTime(c.AIAssessedAt),

		LegacyOutcome: legacyOutcomeText(c.LegacyOutcome),
		LegacyTier:    c.LegacyTier,

		Progress: toProgressDTO(c.Progress, viewer),
	}
}

// legacyOutcomeText menyembunyikan "menunggu" dari kontrak.
//
// Pada keputusan warisan, "menunggu" berarti Pega BELUM memutuskan apa pun — dan itu
// sama artinya dengan tidak ada keputusan warisan sama sekali. Mengirimnya sebagai teks
// akan membuat layar menampilkan "Pega: menunggu" pada setiap kasus baru, keterangan yang
// tidak menambah apa pun.
func legacyOutcomeText(outcome komite.Outcome) string {
	if outcome == "" || outcome == komite.OutcomePending {
		return ""
	}
	return string(outcome)
}

func toProgressDTO(p komite.Progress, viewer string) ProgressDTO {
	decisions := make([]DecisionDTO, 0, len(p.Decisions))
	for _, d := range p.Decisions {
		decisions = append(decisions, DecisionDTO{
			ID:         d.ID,
			Tier:       d.Tier,
			Kind:       string(d.Kind),
			Note:       d.Note,
			ActorLogin: d.ActorLogin,
			ActorName:  d.ActorName,
			DecidedAt:  formatTime(d.DecidedAt),
		})
	}

	return ProgressDTO{
		Outcome:          string(p.Outcome),
		TierCount:        p.TierCount,
		CurrentTier:      p.CurrentTier,
		ApprovedTiers:    p.ApprovedTiers,
		TierCountUnknown: p.TierCountUnknown(),
		Closed:           p.Closed(),
		DecidedByMe:      p.DecidedBy(viewer),
		Decisions:        decisions,
	}
}

// formatTime mengembalikan RFC 3339 dalam UTC, atau teks kosong untuk waktu yang tidak
// terisi.
//
// Waktu yang tidak terisi dikirim KOSONG, bukan sebagai `0001-01-01T00:00:00Z`. Yang
// terakhir adalah nilai nol Go yang akan tergambar di layar sebagai tanggal tahun 1 —
// terbaca seperti data rusak, padahal artinya "belum ada".
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
