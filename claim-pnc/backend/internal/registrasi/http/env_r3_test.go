package registrasihttp_test

import (
	"bytes"
	"claim-pnc/internal/registrasi/acceptancenotepdf"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/dlapdf"
	"claim-pnc/internal/registrasi/facesheetpdf"
	registrasihttp "claim-pnc/internal/registrasi/http"
	"claim-pnc/internal/registrasi/lodpdf"
	"claim-pnc/internal/registrasi/plapdf"
	"claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/usecase"
)

// Berkas ini menyiapkan lingkungan uji lapisan transport registrasi: layanan nyata di atas
// repo memori, dipasang lewat Mount supaya rute yang diuji sama dengan rute produksi.

const (
	firePolicy   = "POL-FIRE-0001"
	testOperator = "ADMINPNC01"

	// Header uji: pengguna pemanggil, penanda tanpa sesi, dan penanda tanpa portal aktif.
	headerUser     = "X-Uji-Pengguna"
	headerNoPortal = "X-Uji-Tanpa-Portal"
	noSession      = "-"
)

// switchGroups adalah sumber grup yang dapat dibuat gagal untuk menguji ResolveCaller.
type switchGroups struct {
	err    error
	groups memory.Groups
}

func (g *switchGroups) GroupsOf(ctx context.Context, login string) ([]string, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.groups.GroupsOf(ctx, login)
}

type httpEnv struct {
	router     chi.Router
	service    *usecase.Service
	store      *memory.Store
	pla        *memory.PLA
	dla        *memory.DLA
	cashier    *memory.Cashier
	records    *memory.ClaimRecords
	uploader   *memory.DocumentUploader
	acceptance *memory.Acceptance
	premium    *memory.Premium
	groups     *switchGroups
}

func newHTTPEnv(t *testing.T) httpEnv {
	t.Helper()

	fixed := clock.FixedAt(time.Date(2026, time.June, 10, 3, 0, 0, 0, time.UTC))
	store := memory.NewStore()
	policyItems := memory.NewPolicyItems(memory.SamplePolicyItems())
	pla := memory.NewPLA()
	dla := memory.NewDLA()
	cashier := memory.NewCashier()
	records := memory.SampleClaimRecords()
	uploader := &memory.DocumentUploader{}
	acceptance := memory.NewAcceptance()
	premium := memory.NewPremium()
	groups := &switchGroups{groups: memory.Groups{}}

	service, err := usecase.NewService(usecase.Options{
		CauseOfLoss:            memory.NewAreaDirectory(),
		AcceptanceNoteRenderer: acceptancenotepdf.Renderer{},
		ClaimRepo:              store,
		TaskRepo:               store.TaskRepo(),
		PolicyRepo:             memory.NewPolicyStore(memory.SamplePolicies(fixed.Now())...),
		NumberIssuer:           memory.NewNumberIssuer(),
		Parameter:              memory.NewParameter(),
		ExchangeRateSource:     memory.NewExchangeRateSource(),
		Assigner: memory.NewAssigner(map[string][]string{
			registrasi.RouterPNCAdmin:     {testOperator},
			registrasi.RouterPNCTechnical: {testOperator},
			registrasi.RouterRCLDoctor:    {testOperator},
		}),
		Notifier:          store,
		AuditRecorder:     store,
		ClaimReportLink:   memory.NewClaimReportLink(),
		AreaDirectory:     memory.NewAreaDirectory(),
		PolicyItems:       policyItems,
		CurrencyDirectory: memory.CurrencyDirectory{},
		ItemOptions:       policyItems,
		ClaimRecords:      records,
		FaceSheet:         memory.NewFaceSheet(),
		FaceSheetRenderer: facesheetpdf.Renderer{},
		PLA:               pla,
		PLARenderer:       plapdf.Renderer{},
		DLA:               dla,
		DLARenderer:       dlapdf.Renderer{},
		Cashier:           cashier,
		CashierGateway:    cashier,
		LODRenderer:       lodpdf.Renderer{},
		Acceptance:        acceptance,
		Premium:           premium,
		Groups:            groups,
		Inbox:             memory.NewInboxEntries(),
		Accounts:          memory.NewAccounts(),
		CommitteeTiering:  memory.NewCommitteeTiering(),
		Committees:        memory.NewCommittees(),
		Documents:         uploader,
		Attachments:       records,
		PUCLLetters:       memory.NewPUCL(),
		PUCLOptions:       memory.NewPUCL(),
		Closures:          records,
		IDGenerator:       memory.IDGenerator{},
		UnitOfWork:        store,
		Clock:             fixed,
	})
	require.NoError(t, err)

	handler := registrasihttp.NewHandler(registrasihttp.Options{
		Service: service,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Caller: func(r *http.Request) (usecase.Caller, bool) {
			user := r.Header.Get(headerUser)
			if user == noSession {
				return usecase.Caller{}, false
			}
			if user == "" {
				user = testOperator
			}
			return usecase.Caller{
				Identity: user,
				Name:     "Petugas Uji",
				Workbasket: []string{
					registrasi.WorkbasketRCLPUCL, registrasi.WorkbasketInvestigator, registrasi.WorkbasketCompliance,
				},
			}, true
		},
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
	})

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(headerNoPortal) == "" {
				r = r.WithContext(portalhttp.WithActivePortal(r.Context(), portal.Portal{ID: "1", Alias: "ASM"}))
			}
			next.ServeHTTP(w, r)
		})
	})
	registrasihttp.Mount(router, handler)

	return httpEnv{
		router: router, service: service, store: store, pla: pla, dla: dla, cashier: cashier,
		records: records, uploader: uploader, acceptance: acceptance, premium: premium, groups: groups,
	}
}

// request menyiapkan permintaan JSON; body nil berarti tanpa badan, body string dikirim
// apa adanya (untuk menguji badan cacat).
func request(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	r := httptest.NewRequest(method, path, reader)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func (e httpEnv) serve(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, r)
	return w
}

// do mengirim permintaan JSON sebagai petugas uji.
func (e httpEnv) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return e.serve(request(t, method, path, body))
}

// doAs mengirim permintaan JSON sebagai pengguna tertentu.
func (e httpEnv) doAs(t *testing.T, user, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	r := request(t, method, path, body)
	r.Header.Set(headerUser, user)
	return e.serve(r)
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), "badan = %s", w.Body.String())
	return out
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return decode[registrasihttp.ErrorResponse](t, w).Code
}

// requireError memastikan status dan kode galat jawaban.
func requireError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, code, errorCode(t, w))
}

var errGroupsDown = errors.New("grup tidak terbaca")

// ── Langkah alur yang dipakai berulang ─────────────────────────────────────────

// startClaim membuka klaim Fire lewat HTTP dan mengembalikan jawabannya.
func (e httpEnv) startClaim(t *testing.T) registrasihttp.ClaimResponse {
	t.Helper()
	w := e.do(t, http.MethodPost, "/registrasi/klaim", registrasihttp.StartRequest{PolicyNumber: firePolicy})
	require.Equal(t, http.StatusCreated, w.Code, "badan = %s", w.Body.String())
	return decode[registrasihttp.ClaimResponse](t, w)
}

// upToInputRegister membuka klaim lalu menutup View Polis lewat HTTP.
func (e httpEnv) upToInputRegister(t *testing.T) registrasihttp.ClaimResponse {
	t.Helper()
	start := e.startClaim(t)
	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/selesai",
		registrasihttp.CompleteRequest{Action: registrasi.ActionViewPolicy})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	out := decode[registrasihttp.ClaimResponse](t, w)
	require.Equal(t, registrasi.StageInputRegister, out.Claim.CurrentStage)
	return out
}

func validRegister(taskID string) registrasihttp.RegisterRequest {
	return registrasihttp.RegisterRequest{
		TaskID:       taskID,
		DateOfLoss:   "2026-06-05",
		ReportDate:   "2026-06-06",
		DateReceived: "2026-06-07",
		Location:     "Gudang A",
		Chronology:   "Kebakaran pada gudang penyimpanan.",
		Reporter: registrasihttp.ReporterDTO{
			Name: "Pelapor Uji", Phone: "0800000000", Relation: 1,
		},
		Area:               registrasihttp.AreaDTO{Country: registrasi.CountryIndonesia, CountryID: "100009"},
		EstimateValueCents: int64(registrasi.Rupiah(10_000_000)),
		Currency:           "IDR",
		TechnicalPIC:       testOperator,
		InsuredItem: []registrasihttp.InsuredItemDTO{{
			ID: "OBJ-1", Name: "Gudang", Location: "Gudang A",
			Coverage: []registrasihttp.CoverageDTO{{
				ID: "CVG-1", CauseOfLoss: "11817", TSICents: int64(registrasi.Rupiah(500_000_000)),
				Spreading: []registrasihttp.SpreadingDTO{
					{TreatyKind: "10007", Name: "OR", Share: 600_000},
					{TreatyKind: "10008", Name: "Treaty", Share: 400_000},
				},
			}},
		}},
	}
}

// upToInputEstimate membawa klaim sampai tahap Input Estimasi lewat HTTP.
func (e httpEnv) upToInputEstimate(t *testing.T) registrasihttp.ClaimResponse {
	t.Helper()
	reg := e.upToInputRegister(t)
	w := e.do(t, http.MethodPost, "/registrasi/register", validRegister(reg.Task.ID))
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	out := decode[registrasihttp.ClaimResponse](t, w)
	require.NotNil(t, out.Task)
	require.Equal(t, registrasi.StageEstimateAdmin, out.Claim.CurrentStage)
	return out
}

func oneEstimate(taskID string, value registrasi.Money) registrasihttp.EstimateRequest {
	return registrasihttp.EstimateRequest{TaskID: taskID, Object: []registrasihttp.EstimateObjectDTO{{
		Coverage: []registrasihttp.EstimateCoverageDTO{{Item: []registrasihttp.ObjectItemDTO{{
			Name: "Gudang", Description: "Atap rusak",
			Estimation: []registrasihttp.EstimationDTO{{
				Type: registrasi.EstimateClaim, Currency: "IDR", Date: "2026-06-08", ValueCents: int64(value),
			}},
		}}}},
	}}}
}

// upToChooseSurveyor menyimpan estimasi, mengunduh CFS, lalu menutup Input Estimasi.
func (e httpEnv) upToChooseSurveyor(t *testing.T) registrasihttp.TaskDTO {
	t.Helper()
	est := e.upToInputEstimate(t)
	body := oneEstimate(est.Task.ID, registrasi.Rupiah(50_000_000))
	w := e.do(t, http.MethodPost, "/registrasi/estimasi/simpan", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/cfs",
		registrasihttp.FaceSheetRequest{TaskID: est.Task.ID, Object: 1, Coverage: 1})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	w = e.do(t, http.MethodPost, "/registrasi/estimasi", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	out := decode[registrasihttp.ClaimResponse](t, w)
	require.NotNil(t, out.Task)
	require.Equal(t, registrasi.StageChooseSurveyor, out.Task.Stage)
	return *out.Task
}

// withSettlement menambah satu adjustment Final pada klaim di Choose Surveyor.
func (e httpEnv) withSettlement(t *testing.T) registrasihttp.TaskDTO {
	t.Helper()
	task := e.upToChooseSurveyor(t)
	w := e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment", finalSettlement(task.ID))
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	return task
}

func finalSettlement(taskID string) registrasihttp.SettlementRequest {
	return registrasihttp.SettlementRequest{
		TaskID: taskID, Object: 1, Coverage: 1, PaymentType: registrasi.PaymentFinal,
		ProposeCents: int64(registrasi.Rupiah(10_000_000)), SubmittedCents: int64(registrasi.Rupiah(12_000_000)),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}
}

// approved membawa adjustment 1 sampai disetujui kedua jenjang komite, dengan penerima
// berekening lengkap dan kode bisnis polis yang punya checklist.
func (e httpEnv) approved(t *testing.T) registrasihttp.TaskDTO {
	t.Helper()
	task := e.withSettlement(t)
	w := e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment/komite",
		registrasihttp.CommitteeTransferRequest{TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	committee := decode[registrasihttp.CommitteeTransferResponse](t, w).Committee
	for _, member := range []string{"KOMITE01", "KOMITE02"} {
		w = e.doAs(t, member, http.MethodPost, "/registrasi/komite/"+committee.ID+"/putusan",
			registrasihttp.CommitteeDecisionRequest{Decision: registrasi.DecisionApprove})
		require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/penerima", registrasihttp.ReceiverRequest{
		TaskID: task.ID, ReceiverID: "1", AccountNo: "1234567890", Email: "a@contoh.internal",
	})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())

	ctx := context.Background()
	claim, err := e.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.Policy.BusinessCode = "10140"
	require.NoError(t, e.store.Save(ctx, claim))
	return task
}

// httptestRecorder dan ioNopCloser menyingkat penulisan pada uji multipart.
type httptestRecorder = httptest.ResponseRecorder

func ioNopCloser(r io.Reader) io.ReadCloser { return io.NopCloser(r) }
