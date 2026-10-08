// Package memory memenuhi seam inboxrcl.Repo dengan penyimpanan di memori.
//
// Dipakai pengujian aturan modul tanpa basis data dan pengembangan lokal saat `PENYIMPANAN`
// tidak menunjuk basis data mana pun.
//
// Penyaring dan urutannya ditiru persis seperti kueri TC_PNC_PUCL:
//
//	operator = LOGIN_ID aktif di M_LOGIN_PNC
//	ASSIGNED_OPERATOR_ID  = operator              penyaring A
//	STATUS_WORK          != Resolved-Completed    penyaring B
//	TGL_KIRIM_PUCL IS NOT NULL                    penyaring C
//	RCL_PUCL IN ('1','3')                         penyaring D
//	urutan TGL_CREATE_PUCL DESC, CLAIMID DESC
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxrcl"
)

// LoginRow adalah satu baris `POOLDATA.M_LOGIN_PNC` — cukup kolom yang dibaca modul ini.
type LoginRow struct {
	LoginID string
	Active  bool
}

// Record adalah satu baris `POOLDATA.TC_PNC_PUCL` sebagaimana dibaca modul ini.
type Record struct {
	Detail inboxrcl.RCLDetail

	// RegisteredAt adalah TGL_CREATE_PUCL — kunci urutan, tidak ada di RCLDetail.
	RegisteredAt time.Time

	// State adalah kolom yang dibaca keputusan (STATUS_CASE, PUCL_APPROVE, dst.). Mode dan
	// alasan dokter diambil dari Detail.
	State inboxrcl.PUCLState

	// TechnicalPIC adalah T_CLAIM_PNC.PICTEKNIK — pemilik tugas Send To Analis.
	TechnicalPIC string
}

// Decision adalah satu keputusan yang tercatat — padanan tulisan ke TC_PNC_PUCL, CPNC_TUGAS,
// dan LIST_HISTORY_CLAIM_PNC.
type Decision struct {
	ClaimNumber string
	Operator    string
	Outcome     inboxrcl.Outcome
	Assignee    string
}

// Store adalah pembaca antrean RCL Dokter di memori.
type Store struct {
	mu        sync.Mutex
	logins    []LoginRow
	records   []Record
	decisions []Decision
}

// NewStore membentuk pembaca dari baris yang diberikan.
func NewStore(logins []LoginRow, records []Record) *Store {
	// Disalin supaya keputusan tidak mengubah baris milik pemanggil (mis. SampleRecords).
	copied := make([]Record, len(records))
	copy(copied, records)
	return &Store{logins: logins, records: copied}
}

// Decisions menyerahkan salinan keputusan yang tercatat.
func (s *Store) Decisions() []Decision {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Decision, len(s.decisions))
	copy(result, s.decisions)
	return result
}

// Decide meniru transaksi `Decide` milik sqlstore: penyaring antrean, Plan, pemilik baru.
func (s *Store) Decide(_ context.Context, cmd inboxrcl.DecisionCommand) (inboxrcl.Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	number := normalize(cmd.ClaimNumber)
	for i := range s.records {
		record := &s.records[i]
		if normalize(record.Detail.ClaimNumber) != number || !inQueue(record.Detail, cmd.Operator) {
			continue
		}

		state := record.State
		state.Mode = record.Detail.Mode
		state.DoctorReason = record.Detail.DoctorReason

		out, err := inboxrcl.Plan(state, cmd.Decision, cmd.DoctorReason, cmd.At)
		if err != nil {
			return inboxrcl.Outcome{}, err
		}

		assignee := inboxrcl.WorkbasketRCLPUCL
		if out.NextQueue == inboxrcl.QueueWorklist {
			assignee = strings.TrimSpace(record.TechnicalPIC)
			if assignee == "" {
				return inboxrcl.Outcome{}, inboxrcl.ErrTechnicalPICUnknown
			}
		}

		record.Detail.StatusClaim = out.StatusClaim
		record.Detail.DoctorReason = out.DoctorReason
		record.Detail.AssignedOperator = assignee
		record.Detail.SentToRCLAt = out.SentToPUCLAt
		record.State.StatusCase = out.StatusCase
		record.State.PUCLApprove = out.PUCLApprove
		record.State.StatusKlaim = out.StatusKlaim
		record.State.LetterPrintedAt = out.LetterPrintedAt

		s.decisions = append(s.decisions, Decision{
			ClaimNumber: record.Detail.ClaimNumber,
			Operator:    normalize(cmd.Operator),
			Outcome:     out,
			Assignee:    assignee,
		})
		return out, nil
	}
	return inboxrcl.Outcome{}, inboxrcl.ErrClaimNotFound
}

// OperatorFor meniru `operator_for` — LOGIN_ID yang aktif, huruf besar.
func (s *Store) OperatorFor(_ context.Context, loginID string) (string, error) {
	id := normalize(loginID)
	if id == "" {
		return "", nil
	}
	for _, row := range s.logins {
		if row.Active && normalize(row.LoginID) == id {
			return id, nil
		}
	}
	return "", nil
}

// List mengambil satu halaman antrean milik seorang operator.
func (s *Store) List(
	_ context.Context,
	operator string,
	f inboxrcl.Filter,
) (inboxrcl.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := f.Normalize()

	matched := []Record{}
	for _, record := range s.records {
		if inQueue(record.Detail, operator) && matchesSearch(record.Detail, clean.Search) {
			matched = append(matched, record)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i], matched[j]
		if left.RegisteredAt.Equal(right.RegisteredAt) {
			return left.Detail.ClaimNumber > right.Detail.ClaimNumber
		}
		return left.RegisteredAt.After(right.RegisteredAt)
	})

	result := inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}, Total: len(matched)}
	if clean.Offset >= len(matched) {
		return result, nil
	}
	end := clean.Offset + clean.Limit
	if end > len(matched) {
		end = len(matched)
	}
	for _, record := range matched[clean.Offset:end] {
		result.Tasks = append(result.Tasks, toTask(record))
	}
	return result, nil
}

// Detail meniru `claim_detail` — penyaring yang sama dengan antrean.
func (s *Store) Detail(
	_ context.Context,
	operator, claimNumber string,
) (inboxrcl.RCLDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	number := normalize(claimNumber)
	for _, record := range s.records {
		if normalize(record.Detail.ClaimNumber) == number && inQueue(record.Detail, operator) {
			return record.Detail, nil
		}
	}
	return inboxrcl.RCLDetail{}, inboxrcl.ErrClaimNotFound
}

// inQueue meniru keempat penyaring Report Definition.
func inQueue(d inboxrcl.RCLDetail, operator string) bool {
	op := normalize(operator)
	return op != "" &&
		normalize(d.AssignedOperator) == op &&
		strings.TrimSpace(d.ProcessStatus) != inboxrcl.StatusKerjaSelesai &&
		!d.SentToRCLAt.IsZero() &&
		inboxrcl.IsDoctorMode(d.Mode)
}

func matchesSearch(d inboxrcl.RCLDetail, search string) bool {
	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(d.ClaimNumber), needle) ||
		strings.Contains(strings.ToUpper(d.PolicyNumber), needle)
}

func toTask(r Record) inboxrcl.RCLTask {
	return inboxrcl.RCLTask{
		ClaimNumber:      r.Detail.ClaimNumber,
		PolicyNumber:     r.Detail.PolicyNumber,
		InsuredName:      r.Detail.InsuredName,
		SentToRCLAt:      r.Detail.SentToRCLAt,
		AnalystNote:      r.Detail.AnalystNote,
		Mode:             r.Detail.Mode,
		RegisteredAt:     r.RegisteredAt,
		ProcessStatus:    r.Detail.ProcessStatus,
		AssignedOperator: r.Detail.AssignedOperator,
	}
}

func normalize(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
