package registrasihttp

import (
	"net/http"
	"time"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tab Survey tahap Choose Surveyor — `Section/TabSurvey_sect.xml`.

// SurveyObjectDTO adalah satu baris grid Tambah Survey.
type SurveyObjectDTO struct {
	ObjectID       string `json:"objek_id"`
	Seq            int    `json:"urutan"`
	Name           string `json:"nama_objek"`
	Location       string `json:"lokasi_objek"`
	Selected       bool   `json:"pilih"`
	SurveyLocation string `json:"lokasi_survey"`
	SurveyorType   string `json:"tipe_surveyor"`
	SurveyorName   string `json:"nama_surveyor"`
	SurveyorLogin  string `json:"login_surveyor"`
	SurveyorAddr   string `json:"alamat_surveyor"`
	SurveyorEmail  string `json:"email_surveyor"`
	BranchCode     string `json:"kode_cabang"`
	BranchName     string `json:"nama_cabang"`
	MarineName     string `json:"nama_surveyor_marine"`
	MarineLogin    string `json:"login_surveyor_marine"`
	Status         string `json:"status"`
	StatusName     string `json:"nama_status,omitempty"`
	SurveyID       string `json:"id_survey,omitempty"`
	SurveyIDMarine string `json:"id_survey_marine,omitempty"`
}

// SurveyRequestDTO adalah Permintaan Survey (`RequestSurvey`).
type SurveyRequestDTO struct {
	RequestorName string `json:"survey_atas_permintaan"`
	Location      string `json:"lokasi_survey"`
	Phone         string `json:"no_telp"`
	Date          string `json:"tanggal_request"`
	Branch        string `json:"cabang"`
	Surveyor      string `json:"surveyor"`
	Email         string `json:"email_surveyor"`
	ObjectName    string `json:"nama_objek"`
}

// SurveyTabResponse adalah isi tab Survey.
type SurveyTabResponse struct {
	Objects    []SurveyObjectDTO `json:"objek"`
	Request    *SurveyRequestDTO `json:"permintaan,omitempty"`
	AddVisible bool              `json:"tambah_tampil"`
	CoMember   bool              `json:"anggota_koasuransi"`
	MarineHull bool              `json:"marine_hull"`
	PA         bool              `json:"pa"`
}

// SurveyActionResponse adalah jawaban Transfer Survei / Transfer Komite.
type SurveyActionResponse struct {
	SurveyTabResponse
	Surveys   []string `json:"survey_terbit"`
	Committee []string `json:"komite_terbit"`
	Auto      bool     `json:"otomatis"`
}

// SurveyorOptionDTO adalah satu pilihan surveyor.
type SurveyorOptionDTO struct {
	ID         string `json:"id"`
	Name       string `json:"nama"`
	Login      string `json:"login"`
	Address    string `json:"alamat"`
	Branch     string `json:"cabang"`
	BranchName string `json:"nama_cabang"`
	Email      string `json:"email"`
	Contact    string `json:"kontak"`
}

// SurveyRequestBody adalah badan Simpan / Transfer Survei.
type SurveyRequestBody struct {
	TaskID           string            `json:"tugas_id"`
	CaseSurveyorType string            `json:"tipe_surveyor_kasus"`
	Objects          []SurveyObjectDTO `json:"objek"`
}

// SurveyCommitteeBody adalah badan modal Transfer Komite.
type SurveyCommitteeBody struct {
	SurveyRequestBody
	Date          string              `json:"tanggal"`
	Initial       string              `json:"inisial"`
	Circumstances string              `json:"circumstances"`
	Nominated     string              `json:"nominated_adjuster"`
	Remarks       string              `json:"remarks"`
	Company       string              `json:"nama_perusahaan"`
	ContactPerson string              `json:"contact_person"`
	OfficePhone   string              `json:"office_phone"`
	Email         string              `json:"email"`
	Manual        bool                `json:"penunjukan_manual"`
	NominatedList []SurveyorOptionDTO `json:"nominasi"`
}

// SurveyCancelBody adalah badan Batal Survei.
type SurveyCancelBody struct {
	TaskID   string `json:"tugas_id"`
	ObjectID string `json:"objek_id"`
}

func surveyTabDTO(v usecase.SurveyTabView) SurveyTabResponse {
	out := SurveyTabResponse{Objects: []SurveyObjectDTO{}, AddVisible: v.AddVisible, CoMember: v.CoMember,
		MarineHull: v.MarineHull, PA: v.PA}
	for _, o := range v.Objects {
		out.Objects = append(out.Objects, SurveyObjectDTO{
			ObjectID: o.ObjectID, Seq: o.Seq, Name: o.Name, Location: o.Location, Selected: o.Selected,
			SurveyLocation: o.SurveyLocation, SurveyorType: o.SurveyorType, SurveyorName: o.SurveyorName,
			SurveyorLogin: o.SurveyorLogin, SurveyorAddr: o.SurveyorAddr, SurveyorEmail: o.SurveyorEmail,
			BranchCode: o.BranchCode, BranchName: o.BranchName, MarineName: o.MarineName, MarineLogin: o.MarineLogin,
			Status: o.Status, StatusName: registrasi.SurveyStatusName[o.Status], SurveyID: o.SurveyID,
			SurveyIDMarine: o.SurveyIDMarine,
		})
	}
	if v.HasRequest {
		r := v.Request
		out.Request = &SurveyRequestDTO{RequestorName: r.RequestorName, Location: r.Location, Phone: r.Phone,
			Date: formatDate(r.Date), Branch: r.Branch, Surveyor: r.Surveyor, Email: r.Email, ObjectName: r.ObjectName}
	}
	return out
}

func (b SurveyRequestBody) command(claimID string) usecase.SurveyCommand {
	c := usecase.SurveyCommand{ClaimID: claimID, TaskID: b.TaskID, CaseSurveyorType: b.CaseSurveyorType}
	for _, o := range b.Objects {
		c.Objects = append(c.Objects, usecase.SurveyObjectInput{
			ObjectID: o.ObjectID, Selected: o.Selected, SurveyLocation: o.SurveyLocation, SurveyorType: o.SurveyorType,
			SurveyorName: o.SurveyorName, SurveyorLogin: o.SurveyorLogin, SurveyorAddr: o.SurveyorAddr,
			SurveyorEmail: o.SurveyorEmail, BranchCode: o.BranchCode, BranchName: o.BranchName,
			MarineName: o.MarineName, MarineLogin: o.MarineLogin,
		})
	}
	return c
}

// SurveyTab menangani GET …/survey/tab.
func (h *Handler) SurveyTab(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	v, err := h.service.SurveyTab(r.Context(), claimID, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, surveyTabDTO(v))
}

// SurveyorOptions menangani GET /registrasi/surveyor?tipe= — pilihan surveyor satu tipe, atau
// pilihan Nominated Loss Adjuster bila tipe "nominasi".
func (h *Handler) SurveyorOptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	kind := r.URL.Query().Get("tipe")
	var (
		list []registrasi.SurveyorOption
		err  error
	)
	if kind == "nominasi" {
		list, err = h.service.NominatedSurveyorOptions(r.Context())
	} else {
		list, err = h.service.SurveyorOptions(r.Context(), kind)
	}
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := make([]SurveyorOptionDTO, 0, len(list))
	for _, o := range list {
		out = append(out, SurveyorOptionDTO{ID: o.ID, Name: o.Name, Login: o.Login, Address: o.Address,
			Branch: o.Branch, BranchName: o.BranchName, Email: o.Email, Contact: o.Contact})
	}
	h.writeResponse(w, r, http.StatusOK, out)
}

// SaveSurvey menangani POST …/survey/simpan — tombol Simpan.
func (h *Handler) SaveSurvey(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SurveyRequestBody
	if !h.readBody(w, r, &body) {
		return
	}
	v, err := h.service.SaveSurvey(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, surveyTabDTO(v))
}

func surveyActionDTO(res usecase.SurveyResult) SurveyActionResponse {
	out := SurveyActionResponse{SurveyTabResponse: surveyTabDTO(res.View), Surveys: res.Surveys, Committee: res.Committee, Auto: res.Auto}
	if out.Surveys == nil {
		out.Surveys = []string{}
	}
	if out.Committee == nil {
		out.Committee = []string{}
	}
	return out
}

// TransferSurvey menangani POST …/survey/transfer — tombol Transfer Survei.
func (h *Handler) TransferSurvey(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SurveyRequestBody
	if !h.readBody(w, r, &body) {
		return
	}
	res, err := h.service.TransferSurvey(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, surveyActionDTO(res))
}

// TransferSurveyCommittee menangani POST …/survey/komite — modal Transfer Komite.
func (h *Handler) TransferSurveyCommittee(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SurveyCommitteeBody
	if !h.readBody(w, r, &body) {
		return
	}
	note := registrasi.SurveyCommitteeNote{
		Initial: body.Initial, Circumstances: body.Circumstances, Nominated: body.Nominated, Remarks: body.Remarks,
		Company: body.Company, ContactPerson: body.ContactPerson, OfficePhone: body.OfficePhone, Email: body.Email,
	}
	if t, err := time.Parse(time.RFC3339, body.Date); err == nil {
		note.Date = t
	}
	cmd := usecase.SurveyCommitteeCommand{SurveyCommand: body.command(claimID), Note: note, Manual: body.Manual}
	for _, n := range body.NominatedList {
		cmd.Nominated = append(cmd.Nominated, registrasi.SurveyorOption{ID: n.ID, Name: n.Name, Login: n.Login})
	}
	res, err := h.service.TransferSurveyCommittee(r.Context(), cmd, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, surveyActionDTO(res))
}

// CancelSurvey menangani POST …/survey/batal — tombol "Ya" modal Batal Survei.
func (h *Handler) CancelSurvey(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SurveyCancelBody
	if !h.readBody(w, r, &body) {
		return
	}
	v, err := h.service.CancelSurvey(r.Context(), usecase.SurveyCancelCommand{ClaimID: claimID, TaskID: body.TaskID, ObjectID: body.ObjectID}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, surveyTabDTO(v))
}
