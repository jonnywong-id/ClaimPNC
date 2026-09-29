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
	Case(ctx context.Context, caseID string, operator string, allOperators bool) (komite.CommitteeCase, error)
	Detail(ctx context.Context, caseID string, operator string, allOperators bool) (usecase.CaseDetail, error)
	Decide(ctx context.Context, cmd komite.DecisionCommand, actor usecase.Actor) (komite.CommitteeCase, error)
}

// InboxHandler melayani layar Inbox Komite.
type InboxHandler struct {
	service      InboxService
	caller       func(context.Context) (InboxCaller, bool)
	logger       *slog.Logger
	allOperators bool

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

	// AllOperators mematikan penyaring pemilik pada SELURUH permintaan daftar.
	//
	// Ia dipasang saat perakitan dari `KOMITE_TANPA_PENYARING_OPERATOR`, dan konfigurasi
	// MENOLAK menyalakannya di luar `APP_ENV=development`. Handler tidak membaca
	// lingkungan sendiri: yang menentukan lingkungan adalah satu tempat, dan modul ini
	// bukan tempat itu.
	//
	// Ia TIDAK dapat dinyalakan lewat parameter kueri. Penanda yang dapat dikirim klien
	// berarti siapa pun yang punya sesi dapat meminta antrean komite seluruh perusahaan —
	// tepat yang penyaring ini ada untuk mencegahnya.
	AllOperators bool

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
		allOperators:  o.AllOperators,
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

	filter, err := inboxFilterFrom(r, caller.Login, h.allOperators)
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
		Kind:               string(normalized.Kind),
		Operator:           normalized.Operator,
		Now:                result.Now.UTC().Format(time.RFC3339),
		DecisionsAvailable: result.DecisionsAvailable,
		OwnerFilterActive:  !normalized.AllOperators,
	})
}

// Detail menangani GET /api/komite/inbox/{nomor}.
func (h *InboxHandler) Detail(w http.ResponseWriter, r *http.Request) {
	caller, signedIn := h.callerFrom(r)
	if !signedIn {
		h.writeError(w, r, komite.ErrNotAssigned)
		return
	}

	detail, err := h.service.Detail(r.Context(), chi.URLParam(r, "nomor"), caller.Login, h.allOperators)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	transfer := toTransferDTO(detail.Transfer)
	h.writeResponse(w, r, http.StatusOK, CommitteeCaseResponse{
		Case:     toCommitteeCaseDTO(detail.Case, caller.Login, detail.Now),
		Now:      detail.Now.UTC().Format(time.RFC3339),
		Transfer: &transfer,
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
func inboxFilterFrom(
	r *http.Request,
	operator string,
	allOperators bool,
) (komite.InboxFilter, error) {
	params := r.URL.Query()

	filter := komite.InboxFilter{
		Operator: operator,

		// Datang dari perakitan, BUKAN dari parameter kueri. Lihat
		// InboxHandlerOptions.AllOperators.
		AllOperators: allOperators,

		Kind:   komite.InboxKind(strings.TrimSpace(params.Get("kotak"))),
		Search: params.Get("cari"),
		Offset: atoiOrZero(params.Get("lewati")),
		Limit:  atoiOrZero(params.Get("batas")),
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

		CommitteeDate: formatTime(c.CommitteeDate),
		CreatedAt:     formatTime(c.CreatedAt),
		AgingDays:     c.AgingDays(now),

		WorkStatus:    c.WorkStatus,
		LegacyOutcome: legacyOutcomeText(c.LegacyOutcome),

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

// toTransferDTO memetakan rincian transfer ke bentuk kontraknya.
//
// Senarai barisnya dibentuk dengan panjang nol, bukan nil: `nil` menjadi `null` di JSON,
// dan layar yang melakukan `baris.map(...)` atasnya akan gagal — bukan menampilkan daftar
// kosong seperti yang dimaksud.
func toTransferDTO(d komite.TransferDetail) TransferDetailDTO {
	lines := make([]AdjustmentLineDTO, 0, len(d.Lines))
	for _, l := range d.Lines {
		lines = append(lines, AdjustmentLineDTO{
			ClaimNumber:  l.ClaimNumber,
			ObjectID:     l.ObjectID,
			CoverageID:   l.CoverageID,
			AcceptanceNo: l.AcceptanceNo,
			AcceptedAt:   formatTime(l.AcceptedAt),

			Currency:    l.Currency,
			PaymentType: l.PaymentType,

			GrossValue:     l.GrossValue.String(),
			ProposeValue:   l.ProposeValue.String(),
			AcceptedValue:  l.AcceptedValue.String(),
			SalvageValue:   l.SalvageValue.String(),
			ASMShareValue:  l.ASMShareValue.String(),
			IndividualRisk: l.IndividualRisk.String(),

			ASMSharePercent: l.ASMSharePercent,
			ExGratia:        l.ExGratia,
			Notes:           l.Notes,
			CauseOfLoss:     l.CauseOfLoss,
		})
	}

	coverages := make([]CoverageAnalysisDTO, 0, len(d.Coverages))
	for _, c := range d.Coverages {
		coverages = append(coverages, CoverageAnalysisDTO{
			ObjectID:   c.ObjectID,
			CoverageID: c.CoverageID,

			ObjectName:   c.ObjectName,
			CoverageName: c.CoverageName,
			CauseOfLoss:  c.CauseOfLoss,

			SumInsured: c.SumInsured.String(),
			Currency:   c.Currency,

			Circumstances:  c.Circumstances,
			ExtentOfLoss:   c.ExtentOfLoss,
			LegalLiability: c.LegalLiability,
			Remarks:        c.Remarks,
			Diagnose:       c.Diagnose,
			InitialName:    c.InitialName,

			CommitteeDate:  formatTime(c.CommitteeDate),
			AnalysisFilled: c.Filled(),
		})
	}

	dto := TransferDetailDTO{
		Judul:          d.Judul(),
		HEDapatDinilai: strings.TrimSpace(d.BusinessType) != "",
		Lines:          lines,
		Coverages:      coverages,
		Empty:          d.Empty(),
		MoneyEmpty:     d.MoneyEmpty(),
	}
	if d.HasClaim {
		c := d.Claim
		dto.Claim = &ClaimSummaryDTO{
			DateOfLoss:   formatTime(c.DateOfLoss),
			RegisterDate: formatTime(c.RegisterDate),

			Location:    c.Location,
			Chronology:  c.Chronology,
			ClaimStatus: c.ClaimStatus,

			Recommendation: c.Recommendation,

			ASMShare:  c.ASMShare,
			CoinsName: c.CoinsName,
			Currency:  c.Currency,
			ExGratia:  c.ExGratia,
		}
	}
	if d.HasCommitteeRecord {
		c := d.Committee
		dto.Committee = &CommitteeRecordDTO{
			MemberName: c.MemberName,
			Tier:       c.Tier,
			Kind:       c.Kind,
			Note:       c.Note,
			ClaimValue: c.ClaimValue.String(),
			ASMShare:   c.ASMShare,
			DecidedAt:  formatTime(c.DecidedAt),
			Outcome:    legacyOutcomeText(c.Outcome),
		}
	}
	return dto
}
