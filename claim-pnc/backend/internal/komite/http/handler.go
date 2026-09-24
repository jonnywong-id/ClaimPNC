package komitehttp

import (
	"context"
	"log/slog"
	"net/http"
	"sort"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh service beserta penyimpanannya.
type Service interface {
	ListThresholds(ctx context.Context) ([]komite.Threshold, error)
	TieringWith(ctx context.Context, value money.Money, line komite.BusinessLine, applicant string) (komite.Tiering, error)
	Integrity(ctx context.Context) ([]komite.Finding, error)
	Policy() komite.Policy
}

// Handler melayani permintaan modul Komite.
type Handler struct {
	service       Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service
	Logger  *slog.Logger

	// WriteResponse dan FallbackErrorWriter dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat sesi dengan cara yang sama.
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Komite.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// ListThresholds menangani GET /api/master/ambang-komite.
//
// Ia menggantikan pembacaan POOLDATA.EMAILKOMITE yang di sistem lama tersebar di 17
// kueri berbeda — `EmailKomiteBerjenjang_sql` beserta varian PA, Travel, Bonding,
// Simasnet, Adjuster, dan Salvage — masing-masing dengan kombinasi penyaring sendiri.
// Di sini tangganya dibaca sekali dan ditampilkan utuh.
func (h *Handler) ListThresholds(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListThresholds(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	items := make([]ThresholdDTO, 0, len(list))
	tierCount := 0
	for _, t := range list {
		items = append(items, toThresholdDTO(t))
		if t.IsApprovalTier() {
			tierCount++
		}
	}

	lines := komite.ListBusinessLines(list)
	lineNames := make([]string, 0, len(lines))
	for _, l := range lines {
		lineNames = append(lineNames, string(l))
	}

	policy := h.service.Policy()
	h.writeResponse(w, r, http.StatusOK, ThresholdListResponse{
		Thresholds:    items,
		Total:         len(items),
		TotalTiers:    tierCount,
		BusinessLines: lineNames,
		BandPolicies:  toBandPolicyDTO(policy),
		Mode:          string(policy.EffectiveMode()),
	})
}

// Integrity menangani GET /api/master/ambang-komite/integritas.
//
// Tidak ada padanannya di sistem lama: `LIMIT_TOP` di sana tersimpan tetapi tidak pernah
// dipakai satu kueri pun. `D-47` memberinya peran — memeriksa apakah tangganya tersusun
// rapi — dan endpoint inilah yang menjalankannya.
func (h *Handler) Integrity(w http.ResponseWriter, r *http.Request) {
	findings, err := h.service.Integrity(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	items := make([]FindingDTO, 0, len(findings))
	defects, warnings := 0, 0
	for _, f := range findings {
		items = append(items, FindingDTO{
			Severity:     string(f.Severity),
			Kind:         f.Kind,
			BusinessLine: string(f.BusinessLine),
			Band:         f.Band,
			Message:      f.Message,
			ThresholdIDs: f.ThresholdIDs,
		})
		switch f.Severity {
		case komite.SeverityDefect:
			defects++
		case komite.SeverityWarning:
			warnings++
		}
	}

	// 200 walau ada cacat. Cacat pada master adalah TEMUAN yang dilaporkan endpoint ini,
	// bukan kegagalan permintaan — menjawabnya dengan galat akan membuat layar
	// menampilkan halaman gagal justru pada saat ia paling perlu menampilkan isinya.
	h.writeResponse(w, r, http.StatusOK, IntegrityResponse{
		Findings:     items,
		DefectCount:  defects,
		WarningCount: warnings,
	})
}

// Tiering menangani GET /api/komite/penjenjangan?nilai=&lini=.
//
// # Kenapa GET, bukan POST
//
// Ia tidak mengubah apa pun. Yang dikerjakannya adalah membaca master lalu menghitung —
// tidak ada klaim yang tersentuh, tidak ada baris yang tertulis. `D-73` menetapkan aksi
// bisnis dimodelkan sebagai peristiwa lewat POST justru karena aksi seperti itu punya
// invarian dan memicu jejak audit; di sini tidak ada keduanya.
//
// Akibat praktisnya: hasilnya dapat ditautkan. Seseorang yang menemukan angka yang
// meragukan dapat mengirimkan tautannya apa adanya kepada Work Owner.
func (h *Handler) Tiering(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	value, err := money.Parse(params.Get("nilai"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Penginput dikecualikan dari calon penyetuju pada KEDUA mode (Work Owner,
	// 2026-09-18) supaya tidak menyetujui pengajuannya sendiri.
	result, err := h.service.TieringWith(
		r.Context(),
		value,
		komite.BusinessLine(params.Get("lini")),
		params.Get("penginput"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, TieringResponse{
		Value:             result.Value.String(),
		BusinessLine:      string(result.BusinessLine),
		Mode:              string(result.Mode),
		UsesBand:          result.UsesBand,
		Band:              result.Band,
		Approvers:         toApproverDTO(result.Approvers),
		Candidates:        toApproverDTO(result.Candidates),
		ExcludedApplicant: result.ExcludedApplicant,
		Excluded:          toApproverDTO(result.Excluded),
		TierCount:         result.TierCount(),
		NoApprovers:       result.NoApprovers(),
		AmbiguousOrder:    result.AmbiguousOrder,
	})
}

func toApproverDTO(approvers []komite.Approver) []ApproverDTO {
	if len(approvers) == 0 {
		return nil
	}
	result := make([]ApproverDTO, 0, len(approvers))
	for _, a := range approvers {
		result = append(result, ApproverDTO{
			Order:       a.Order,
			Tier:        a.Tier,
			Name:        a.Name,
			OperatorID:  a.OperatorID,
			LowerBound:  a.LowerBound.String(),
			Absent:      a.Absent,
			ThresholdID: a.ThresholdID,
		})
	}
	return result
}

func toThresholdDTO(t komite.Threshold) ThresholdDTO {
	return ThresholdDTO{
		ID:              t.ID,
		Name:            t.Name,
		OperatorID:      t.OperatorID,
		BusinessLine:    string(t.BusinessLine),
		CommitteeType:   t.CommitteeType,
		LowerBound:      t.LowerBound.String(),
		UpperBound:      t.UpperBound.String(),
		Tier:            t.Tier,
		Active:          t.Active,
		ForAdjustment:   t.ForAdjustment,
		ForRegistration: t.ForRegistration,
		ForRejection:    t.ForRejection,
		Absent:          t.Absent,
		IsApprovalTier:  t.IsApprovalTier(),
	}
}

// toBandPolicyDTO mengurutkan hasilnya menurut lini.
//
// Peta di Go tidak punya urutan, dan mengirimkannya apa adanya akan membuat respons
// berubah susunan antar permintaan tanpa ada yang berubah isinya — cukup untuk membuat
// perbandingan keluaran gagal tanpa sebab.
func toBandPolicyDTO(p komite.Policy) []BandPolicyDTO {
	result := make([]BandPolicyDTO, 0, len(p.Bands))
	for line, rule := range p.Bands {
		result = append(result, BandPolicyDTO{
			BusinessLine: string(line),
			Boundary:     rule.Boundary.String(),
			Lower:        rule.Lower,
			Upper:        rule.Upper,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].BusinessLine < result[j].BusinessLine })
	return result
}
