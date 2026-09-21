package registrasihttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Handler melayani permintaan modul registrasi.
type Handler struct {
	service *usecase.Service
	logger  *slog.Logger

	// caller menerjemahkan permintaan HTTP menjadi identitas pemanggil.
	//
	// Ia disuntikkan, bukan dibaca langsung dari modul auth, supaya lapisan transport
	// modul ini tidak bergantung pada lapisan transport modul lain — dan supaya
	// handler dapat diuji tanpa membentuk seluruh layanan autentikasi.
	caller func(r *http.Request) (usecase.Caller, bool)

	writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service       *usecase.Service
	Logger        *slog.Logger
	Caller        func(r *http.Request) (usecase.Caller, bool)
	WriteResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
}

// NewHandler membentuk handler modul registrasi.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		caller:        o.Caller,
		writeResponse: o.WriteResponse,
	}
}

func (h *Handler) failure(w http.ResponseWriter, r *http.Request, err error) {
	status, body := mapError(err)
	if status >= http.StatusInternalServerError {
		// Rincian galat internal hanya masuk log; peramban menerima pesan umum.
		logging.From(r.Context(), h.logger).Error("permintaan registrasi gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeResponse(w, r, status, body)
}

func (h *Handler) callerOf(w http.ResponseWriter, r *http.Request) (usecase.Caller, bool) {
	p, ok := h.caller(r)
	if !ok {
		h.writeResponse(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    "sesi_tidak_sah",
			Message: "Sesi tidak sah. Silakan masuk kembali.",
		})
		return usecase.Caller{}, false
	}
	return p, true
}

func (h *Handler) readBody(w http.ResponseWriter, r *http.Request, target any) bool {
	// Badan permintaan dibatasi supaya satu permintaan cacat tidak dapat menghabiskan
	// memori proses. Satu klaim dengan ratusan objek dan spreading tetap jauh di bawah
	// limit ini.
	const limit = 4 << 20
	r.Body = http.MaxBytesReader(w, r.Body, limit)

	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}
	return true
}

// Flow menangani GET /api/registrasi/alur.
func (h *Handler) Flow(w http.ResponseWriter, r *http.Request) {
	definition := h.service.Flow()

	stage := definition.Stages()
	body := make([]StageDTO, 0, len(stage))
	for _, t := range stage {
		body = append(body, StageDTO{
			ID:         t.ID,
			Name:       t.Name,
			Queue:      string(t.Queue),
			Workbasket: t.Workbasket,
			Router:     t.Router,
			ExitAction: t.ExitAction,
			PegaID:     t.PegaID,
		})
	}

	h.writeResponse(w, r, http.StatusOK, FlowResponse{
		Name:  definition.Name,
		Start: definition.Start,
		Stage: body,
	})
}

// Inbox menangani GET /api/registrasi/inbox.
func (h *Handler) Inbox(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	task, err := h.service.Inbox(r.Context(), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	definition := h.service.Flow()
	body := make([]TaskDTO, 0, len(task))
	for _, t := range task {
		body = append(body, taskDTO(t, definition))
	}
	h.writeResponse(w, r, http.StatusOK, InboxResponse{Task: body})
}

// Start menangani POST /api/registrasi/klaim.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body StartRequest
	if !h.readBody(w, r, &body) {
		return
	}

	result, err := h.service.Start(r.Context(), usecase.StartCommand{
		PolicyNumber: body.PolicyNumber,
		Portal:       body.Portal,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	definition := h.service.Flow()
	task := taskDTO(result.Task, definition)
	h.writeResponse(w, r, http.StatusCreated, ClaimResponse{
		Claim: claimDTO(result.Claim),
		Task:  &task,
	})
}

// ViewClaim menangani GET /api/registrasi/klaim/{claimID}.
func (h *Handler) ViewClaim(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	summary, err := h.service.ViewClaim(r.Context(), claimID, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{Claim: claimDTO(summary.Claim), Path: summary.Path}
	if summary.Task != nil {
		t := taskDTO(*summary.Task, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// SaveRegister menangani POST /api/registrasi/register.
func (h *Handler) SaveRegister(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body RegisterRequest
	if !h.readBody(w, r, &body) {
		return
	}

	command, err := registerCommand(body)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: err.Error(),
		})
		return
	}

	result, err := h.service.SaveRegister(r.Context(), command, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{
		Claim:         claimDTO(result.Claim),
		DecisionTrace: result.DecisionTrace,
		LargeLoss:     result.LargeLoss,
	}
	if result.NextTask != nil {
		t := taskDTO(*result.NextTask, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// ClaimTask menangani POST /api/registrasi/tugas/{taskID}/ambil.
func (h *Handler) ClaimTask(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	task, err := h.service.ClaimTask(r.Context(), taskID, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, taskDTO(task, h.service.Flow()))
}

// CompleteTask menangani POST /api/registrasi/tugas/{taskID}/selesai.
func (h *Handler) CompleteTask(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body CompleteRequest
	if !h.readBody(w, r, &body) {
		return
	}

	result, err := h.service.CompleteStage(r.Context(), usecase.CompleteCommand{
		TaskID: taskID,
		Action: body.Action,
		Return: body.Return,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{
		Claim:         claimDTO(result.Claim),
		DecisionTrace: result.DecisionTrace,
	}
	if result.NextTask != nil {
		t := taskDTO(*result.NextTask, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// ── Terjemahan antara bentuk wire dan tipe modul ─────────────────────────────────

const dateLayout = "2006-01-02"

// parseDate membaca tanggal `YYYY-MM-DD` sebagai tanggal kalender WIB.
//
// Tidak ada jam yang ikut dikirim, sehingga tidak ada jam yang perlu ditafsirkan. Ini
// menutup seluruh kelas kesalahan yang melahirkan 101 penyesuaian tujuh jam di sistem
// lama: tanggal yang bergeser sehari karena dikirim sebagai tengah malam UTC.
func parseDate(s, name string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("%s wajib diisi", name)
	}
	t, err := time.ParseInLocation(dateLayout, s, clock.ZoneWIB)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s harus berformat YYYY-MM-DD", name)
	}
	return t, nil
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return clock.DateWIB(t).Format(dateLayout)
}

func registerCommand(b RegisterRequest) (usecase.RegisterCommand, error) {
	lossDate, err := parseDate(b.DateOfLoss, "Tanggal Kejadian")
	if err != nil {
		return usecase.RegisterCommand{}, err
	}
	reportDate, err := parseDate(b.ReportDate, "Tanggal Lapor")
	if err != nil {
		return usecase.RegisterCommand{}, err
	}
	receivedDate, err := parseDate(b.DateReceived, "Tanggal Terima Dokumen")
	if err != nil {
		return usecase.RegisterCommand{}, err
	}
	if b.TaskID == "" {
		return usecase.RegisterCommand{}, errors.New("tugas_id wajib diisi")
	}

	insuredItem := make([]usecase.InsuredItemInput, 0, len(b.InsuredItem))
	for _, o := range b.InsuredItem {
		input := usecase.InsuredItemInput{
			ID:       o.ID,
			Name:     o.Name,
			Location: o.Location,
			Coverage: make([]usecase.CoverageInput, 0, len(o.Coverage)),
		}
		for _, c := range o.Coverage {
			cov := usecase.CoverageInput{
				ID:          c.ID,
				CauseOfLoss: c.CauseOfLoss,
				TSI:         registrasi.Money(c.TSICents),
				Spreading:   make([]usecase.SpreadingInput, 0, len(c.Spreading)),
			}
			for _, s := range c.Spreading {
				cov.Spreading = append(cov.Spreading, usecase.SpreadingInput{
					TreatyKind:   s.TreatyKind,
					Name:         s.Name,
					Share:        registrasi.Percent(s.Share),
					Removed:      s.Removed,
					FacOfferItem: s.FacOfferItem,
				})
			}
			input.Coverage = append(input.Coverage, cov)
		}
		insuredItem = append(insuredItem, input)
	}

	return usecase.RegisterCommand{
		TaskID:       b.TaskID,
		DateOfLoss:   lossDate,
		ReportDate:   reportDate,
		DateReceived: receivedDate,
		Location:     b.Location,
		Chronology:   b.Chronology,
		Reporter: registrasi.Reporter{
			Name:          b.Reporter.Name,
			Phone:         b.Reporter.Phone,
			Email:         b.Reporter.Email,
			Address:       b.Reporter.Address,
			Relation:      b.Reporter.Relation,
			OtherRelation: b.Reporter.OtherRelation,
		},
		EstimateValue:      registrasi.Money(b.EstimateValueCents),
		Currency:           b.Currency,
		SLIKNumber:         b.SLIKNumber,
		ExGratia:           b.ExGratia,
		TechnicalPIC:       b.TechnicalPIC,
		RCVID:              b.RCVID,
		InsuredItem:        insuredItem,
		PUCLStatus:         b.PUCLStatus,
		ComplianceTransfer: b.ComplianceTransfer,
		Return:             b.Return,
	}, nil
}

func taskDTO(t registrasi.Task, definition registrasi.Definition) TaskDTO {
	dto := TaskDTO{
		ID:          t.ID,
		ClaimID:     t.ClaimID,
		ClaimNumber: t.ClaimNumber,
		Stage:       t.Stage,
		Queue:       string(t.Queue),
		Workbasket:  t.Workbasket,
		Owner:       t.Owner,
		Claimable:   t.Open() && !t.Owned(),
		CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
	}
	if stage, ok := definition.Stage(t.Stage); ok {
		dto.StageName = stage.Name
		dto.ExitAction = stage.ExitAction
	}
	return dto
}

func claimDTO(k registrasi.Claim) ClaimDTO {
	insuredItem := make([]InsuredItemDTO, 0, len(k.InsuredItem))
	for _, o := range k.InsuredItem {
		item := InsuredItemDTO{
			ID:       o.ID,
			Name:     o.Name,
			Location: o.Location,
			Coverage: make([]CoverageDTO, 0, len(o.Coverage)),
		}
		for _, c := range o.Coverage {
			cov := CoverageDTO{
				ID:          c.ID,
				CauseOfLoss: c.CauseOfLoss,
				TSICents:    int64(c.TSI),
				Spreading:   make([]SpreadingDTO, 0, len(c.Spreading)),
			}
			for _, s := range c.Spreading {
				cov.Spreading = append(cov.Spreading, SpreadingDTO{
					TreatyKind:   s.TreatyKind,
					Name:         s.Name,
					Share:        Percent(s.Share),
					Removed:      s.Removed,
					FacOfferItem: s.FacOfferItem,
				})
			}
			item.Coverage = append(item.Coverage, cov)
		}
		insuredItem = append(insuredItem, item)
	}

	return ClaimDTO{
		ID:     k.ID,
		Number: k.Number,
		Portal: k.Portal,
		Policy: PolicyDTO{
			Number:          k.Policy.Number,
			Line:            string(k.Policy.Line),
			LineName:        LineName(k.Policy.Line),
			BusinessType:    k.Policy.BusinessType,
			CoverageStart:   formatDate(k.Policy.CoverageStart),
			CoverageEnd:     formatDate(k.Policy.CoverageEnd),
			Currency:        k.Policy.Currency,
			InsuredName:     k.Policy.InsuredName,
			Declaration:     k.Policy.Declaration,
			CreditGuarantee: k.Policy.CreditGuarantee,
		},
		DateOfLoss:   formatDate(k.DateOfLoss),
		ReportDate:   formatDate(k.ReportDate),
		DateReceived: formatDate(k.DateReceived),
		Location:     k.Location,
		Chronology:   k.Chronology,
		Reporter: ReporterDTO{
			Name:          k.Reporter.Name,
			Phone:         k.Reporter.Phone,
			Email:         k.Reporter.Email,
			Address:       k.Reporter.Address,
			Relation:      k.Reporter.Relation,
			OtherRelation: k.Reporter.OtherRelation,
		},
		EstimateValueCents:     int64(k.EstimateValue),
		Currency:               k.Currency,
		SLIKNumber:             k.SLIKNumber,
		ExGratia:               k.ExGratia,
		TechnicalPIC:           k.TechnicalPIC,
		RCVID:                  k.RCVID,
		InsuredItem:            insuredItem,
		PUCLStatus:             k.PUCLStatus,
		ComplianceTransfer:     k.ComplianceTransfer,
		ProcessStatus:          string(k.ProcessStatus),
		ClaimStatus:            string(k.ClaimStatus),
		ClaimFlag:              string(k.ClaimFlag),
		ProgressPositionStatus: string(k.ProgressPositionStatus),
		CurrentStage:           k.CurrentStage,
	}
}

// LineName menerjemahkan kode Group Panel menjadi nama yang dikenali petugas.
//
// Daftarnya turun dari rule When yang dipakai alur; kode di luar daftar dikembalikan apa
// adanya, bukan diganti "Lainnya" — menyembunyikan kode yang tidak dikenali akan
// menyembunyikan pula bahwa daftar ini belum lengkap.
func LineName(l registrasi.LineOfBusiness) string {
	switch l {
	case registrasi.LinePersonalAccident:
		return "Personal Accident"
	case registrasi.LineMiscellaneous:
		return "Aneka"
	case registrasi.LineMarineCargo:
		return "Marine Cargo"
	case registrasi.LineTravel:
		return "Travel"
	case registrasi.LineFire:
		return "Fire"
	default:
		return string(l)
	}
}
