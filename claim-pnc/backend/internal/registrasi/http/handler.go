package registrasihttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
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
	// Peran pemanggil dari POOLDATA.M_LOGIN_GROUP_PNC, dibaca sekali per permintaan.
	resolved, err := h.service.ResolveCaller(r.Context(), p)
	if err != nil {
		h.failure(w, r, err)
		return usecase.Caller{}, false
	}
	return resolved, true
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

	// Portal diambil dari portal AKTIF, bukan dari badan permintaan.
	//
	// Badan permintaan dikendalikan pemanggil, dan portal menentukan MILIK SIAPA data
	// yang ditulis (`D-75`). Menerima nilainya dari sana berarti satu permintaan yang
	// disusun tangan dapat menuliskan klaim atas nama badan hukum lain — kegagalan
	// `R-20`, yang tidak terlihat sebagai galat karena layarnya tampak normal.
	//
	// Middleware ActivePortal sudah memeriksa haknya; yang dilakukan di sini hanya
	// memakai hasilnya.
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.failure(w, r, portal.ErrNotStated)
		return
	}

	result, err := h.service.Start(r.Context(), usecase.StartCommand{
		PolicyNumber: body.PolicyNumber,
		RCVID:        body.RCVID,
		Portal:       active.Alias,
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
		t.Workable = h.service.CanWork(*summary.Task, caller)
		t.Analyst = registrasi.IsAnalyst(caller.Roles)
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

// SaveDraft menangani POST /api/registrasi/register/simpan — tombol Save.
//
// Isiannya sama dengan SaveRegister, tetapi tahapnya tidak ditutup dan validasinya tidak
// dijalankan. Lihat usecase.SaveDraft.
func (h *Handler) SaveDraft(w http.ResponseWriter, r *http.Request) {
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

	claim, err := h.service.SaveDraft(r.Context(), command, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}

// AreaOptions menangani GET /api/registrasi/wilayah/{tingkat}?induk=….
//
// induk adalah nilai tingkat di atasnya: nama negara untuk provinsi, kode provinsi untuk
// kota, dan seterusnya. Negara tidak memakai induk.
func (h *Handler) AreaOptions(w http.ResponseWriter, r *http.Request, level string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}

	option, err := h.service.AreaOptions(r.Context(), registrasi.AreaLevel(level), r.URL.Query().Get("induk"))
	if err != nil {
		h.failure(w, r, err)
		return
	}

	body := make([]AreaOptionDTO, 0, len(option))
	for _, o := range option {
		body = append(body, AreaOptionDTO{ID: o.ID, Name: o.Name, PostalCode: o.PostalCode})
	}
	h.writeResponse(w, r, http.StatusOK, AreaOptionsResponse{Option: body})
}

// CauseOfLossOptions menangani GET /api/registrasi/penyebab-kerugian?bisnis=….
//
// bisnis adalah kode bisnis polis (Quotation.BusinessCode) — parameter id autocomplete Pega.
func (h *Handler) CauseOfLossOptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	option, err := h.service.CauseOfLossOptions(r.Context(), r.URL.Query().Get("bisnis"))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := make([]CauseOfLossOptionDTO, 0, len(option))
	for _, o := range option {
		body = append(body, CauseOfLossOptionDTO{ID: o.ID, Name: o.Name})
	}
	h.writeResponse(w, r, http.StatusOK, CauseOfLossOptionsResponse{Option: body})
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

// SendToInputor menangani POST /api/registrasi/tugas/{taskID}/kirim-inputor.
func (h *Handler) SendToInputor(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body SendToInputorRequest
	if !h.readBody(w, r, &body) {
		return
	}

	result, err := h.service.SendToInputor(r.Context(), usecase.SendToInputorCommand{
		TaskID: taskID,
		Note:   body.Note,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{Claim: claimDTO(result.Claim)}
	if result.NextTask != nil {
		t := taskDTO(*result.NextTask, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// SendToRCLPUCL menangani POST /api/registrasi/tugas/{taskID}/kirim-rclpucl — tombol
// Kirim pada modal "Kirim ke RCL/PUCL".
func (h *Handler) SendToRCLPUCL(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body SendToRCLPUCLRequest
	if !h.readBody(w, r, &body) {
		return
	}

	result, err := h.service.SendToRCLPUCL(r.Context(), usecase.SendToRCLPUCLCommand{
		TaskID:      taskID,
		Track:       body.Track,
		AnalystNote: body.Note,
		Subject:     body.Subject,
		OpeningNote: body.OpeningNote,
		BodyNote:    body.BodyNote,
		ClosingNote: body.ClosingNote,
		DoctorName:  body.DoctorName,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{Claim: claimDTO(result.Claim)}
	if result.NextTask != nil {
		t := taskDTO(*result.NextTask, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// PUCLSubjects menangani GET /api/registrasi/rclpucl/perihal?jalur=N.
func (h *Handler) PUCLSubjects(w http.ResponseWriter, r *http.Request) {
	track, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("jalur")))

	options, err := h.service.PUCLSubjectOptions(r.Context(), track)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := PUCLSubjectResponse{Pilihan: make([]PUCLSubjectDTO, 0, len(options))}
	for _, o := range options {
		response.Pilihan = append(response.Pilihan, PUCLSubjectDTO{ID: o.ID, Nama: o.Name})
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// PUCLReasons menangani GET /api/registrasi/rclpucl/alasan?cari=…&batas=N.
func (h *Handler) PUCLReasons(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("batas")))

	reasons, err := h.service.PUCLRejectReasons(r.Context(), r.URL.Query().Get("cari"), limit)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := PUCLReasonResponse{Pilihan: make([]PUCLReasonDTO, 0, len(reasons))}
	for _, o := range reasons {
		response.Pilihan = append(response.Pilihan, PUCLReasonDTO{ID: o.ID, Nama: o.Name, Deskripsi: o.Description})
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// RCLDoctors menangani GET /api/registrasi/rclpucl/dokter.
//
// Tidak dapat gagal: isinya daftar tetap pada property `NamaDokterRCL`, bukan kueri.
func (h *Handler) RCLDoctors(w http.ResponseWriter, r *http.Request) {
	doctors := h.service.RCLDoctorOptions()

	response := RCLDoctorResponse{Pilihan: make([]RCLDoctorDTO, 0, len(doctors))}
	for _, o := range doctors {
		response.Pilihan = append(response.Pilihan, RCLDoctorDTO{ID: o.ID, Nama: o.Label})
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// CloseClaim menangani POST /api/registrasi/tugas/{taskID}/tutup-klaim — tombol Ya dialog
// "Prevent Close Claim".
func (h *Handler) CloseClaim(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CloseClaimRequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.CloseClaim(r.Context(), usecase.CloseClaimCommand{
		TaskID: taskID,
		Closure: registrasi.Closure{
			Note: body.Note, Proposal: body.Proposal, Effort: body.Effort, Obstacle: body.Obstacle,
			Temporary: body.Temporary,
		},
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(result.Claim)})
}

// TransferToAnalyst menangani POST /api/registrasi/tugas/{taskID}/transfer-analis.
func (h *Handler) TransferToAnalyst(w http.ResponseWriter, r *http.Request, taskID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}

	var body TransferToAnalystRequest
	if !h.readBody(w, r, &body) {
		return
	}

	result, err := h.service.TransferToAnalyst(r.Context(), usecase.TransferToAnalystCommand{
		TaskID:     taskID,
		ObjectID:   body.ObjectID,
		CoverageID: body.CoverageID,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	response := ClaimResponse{Claim: claimDTO(result.Claim)}
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

// birthDate menormalkan ASMDATEOFBIRTH T_PERSONLIST — yyyymmdd atau dd/mm/yyyy — menjadi
// YYYY-MM-DD. Bentuk lain dikosongkan, bukan ditebak.
func birthDate(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"20060102", "02/01/2006"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format(dateLayout)
		}
	}
	return ""
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
	// Tanggal Terima Dokumen boleh kosong: isiannya hanya tampil pada Travel dan PA
	// (`IsTravelPA`), dan kewajibannya ditegakkan domain untuk kedua lini itu saja.
	var receivedDate time.Time
	if b.DateReceived != "" {
		receivedDate, err = parseDate(b.DateReceived, "Tanggal Terima Dokumen")
		if err != nil {
			return usecase.RegisterCommand{}, err
		}
	}
	// Tanggal Keluar Rawat Inap boleh kosong: isiannya hanya tampil pada PA yang rawat inap.
	var dischargeDate time.Time
	if b.DischargeDate != "" {
		dischargeDate, err = parseDate(b.DischargeDate, "Tanggal Keluar Rawat Inap")
		if err != nil {
			return usecase.RegisterCommand{}, err
		}
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
				Name:        c.Name,
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
		TaskID:        b.TaskID,
		DateOfLoss:    lossDate,
		ReportDate:    reportDate,
		DateReceived:  receivedDate,
		DischargeDate: dischargeDate,
		Location:      b.Location,
		Chronology:    b.Chronology,
		Area: registrasi.Area{
			Country: b.Area.Country, CountryID: b.Area.CountryID,
			Province: b.Area.Province, ProvinceID: b.Area.ProvinceID,
			City: b.Area.City, CityID: b.Area.CityID,
			District: b.Area.District, DistrictID: b.Area.DistrictID,
			RW: b.Area.RW, RWID: b.Area.RWID,
			PostalCode: b.Area.PostalCode,
		},
		CustomerPrinciple: b.CustomerPrinciple,
		SuspiciousComment: b.SuspiciousComment,

		EmailLOD:             b.EmailLOD,
		RemarkRecommendation: b.RemarkRecommendation,
		SubjectEmail:         b.SubjectEmail,
		SalvageStatus:        b.SalvageStatus,
		InsuredUpdate: registrasi.InsuredUpdate{
			IDCard: b.UpdateIDCard, Phone: b.UpdatePhone, Email: b.UpdateEmail,
		},
		ReportType: b.ReportType,
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
			ID:          o.ID,
			Name:        o.Name,
			Location:    o.Location,
			Coverage:    make([]CoverageDTO, 0, len(o.Coverage)),
			Job:         o.Job,
			DateOfBirth: birthDate(o.DateOfBirth),

			IDCard:            o.IDCard,
			ParticipantStatus: o.ParticipantStatus,
			VehicleModel:      o.VehicleModel,
			VehicleBrand:      o.VehicleBrand,
			VehicleType:       o.VehicleType,
			ChassisNumber:     o.ChassisNumber,
		}
		for _, c := range o.Coverage {
			cov := CoverageDTO{
				ID:          c.ID,
				Name:        c.Name,
				CauseOfLoss: c.CauseOfLoss,
				TSICents:    int64(c.TSI),
				Spreading:   make([]SpreadingDTO, 0, len(c.Spreading)),
				Item:        itemDTO(c.Item),
				Adjustment:  settlementDTO(c.Settlement),

				AnalystTransferred: c.AnalystTransferred,
			}
			committee := committeeNoteDTO(c.Committee)
			cov.Committee = &committee
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
			BusinessCode:    k.Policy.BusinessCode,
			CoverageStart:   formatDate(k.Policy.CoverageStart),
			CoverageEnd:     formatDate(k.Policy.CoverageEnd),
			Currency:        k.Policy.Currency,
			InsuredName:     k.Policy.InsuredName,
			Declaration:     k.Policy.Declaration,
			CreditGuarantee: k.Policy.CreditGuarantee,
			CoinsType:       k.Policy.TypeOfCoins,
			CoinsRole:       k.Policy.Coinsurance.Role,

			SourceOfBusinessName: k.Policy.SourceOfBusinessName,
			BusinessName:         k.Policy.BusinessName,
			BranchCode:           k.Policy.BranchCode,
		},
		Receiver:      receiverDTO(k.Receiver),
		DateOfLoss:    formatDate(k.DateOfLoss),
		ReportDate:    formatDate(k.ReportDate),
		DateReceived:  formatDate(k.DateReceived),
		DischargeDate: formatDate(k.DischargeDate),
		Location:      k.Location,
		Chronology:    k.Chronology,
		Area: AreaDTO{
			Country: k.Area.Country, CountryID: k.Area.CountryID,
			Province: k.Area.Province, ProvinceID: k.Area.ProvinceID,
			City: k.Area.City, CityID: k.Area.CityID,
			District: k.Area.District, DistrictID: k.Area.DistrictID,
			RW: k.Area.RW, RWID: k.Area.RWID,
			PostalCode: k.Area.PostalCode,
		},
		CustomerPrinciple: k.CustomerPrinciple,
		SuspiciousComment: k.SuspiciousComment,

		EmailLOD:             k.EmailLOD,
		RemarkRecommendation: k.RemarkRecommendation,
		SubjectEmail:         k.SubjectEmail,
		SalvageStatus:        k.SalvageStatus,
		TechnicalPICNote:     k.TechnicalPICNote,
		UpdateIDCard:         k.InsuredUpdate.IDCard,
		UpdatePhone:          k.InsuredUpdate.Phone,
		UpdateEmail:          k.InsuredUpdate.Email,
		ReportType:           k.ReportType,
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
		ClaimStatusName:        k.ClaimStatusName,
		AnalystTransferred:     !k.AnalystTransferredAt.IsZero(),
		PendingClose:           k.PendingClose,
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

func receiverDTO(receivers []registrasi.Receiver) []ReceiverDTO {
	out := make([]ReceiverDTO, 0, len(receivers))
	for _, r := range receivers {
		out = append(out, ReceiverDTO{ID: r.ID, Name: r.Name, Address: r.Address, BankName: r.BankName, AccountNo: r.AccountNo})
	}
	return out
}
