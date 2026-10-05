package registrasihttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	registrasihttp "claim-pnc/internal/registrasi/http"
)

// Definisi alur dibaca tanpa sesi pun; tahap pertamanya View Polis.
func TestFlowListsStages(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodGet, "/registrasi/alur", nil)
	require.Equal(t, http.StatusOK, w.Code)

	body := decode[registrasihttp.FlowResponse](t, w)
	require.NotEmpty(t, body.Name)
	require.Equal(t, registrasi.StageViewPolicy, body.Start)
	require.NotEmpty(t, body.Stage)
	ids := map[string]bool{}
	for _, s := range body.Stage {
		ids[s.ID] = true
		require.NotEmpty(t, s.Name)
	}
	require.True(t, ids[registrasi.StageInputRegister])
}

// Setiap rute yang memerlukan sesi menolak permintaan tanpa sesi dengan 401.
func TestRoutesRejectMissingSession(t *testing.T) {
	e := newHTTPEnv(t)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/registrasi/inbox"},
		{http.MethodPost, "/registrasi/klaim"},
		{http.MethodGet, "/registrasi/klaim/K1"},
		{http.MethodPost, "/registrasi/register"},
		{http.MethodPost, "/registrasi/register/simpan"},
		{http.MethodPost, "/registrasi/estimasi"},
		{http.MethodPost, "/registrasi/estimasi/simpan"},
		{http.MethodGet, "/registrasi/mata-uang"},
		{http.MethodGet, "/registrasi/klaim/K1/pilihan-item"},
		{http.MethodGet, "/registrasi/klaim/K1/survey"},
		{http.MethodGet, "/registrasi/klaim/K1/dokumen"},
		{http.MethodPost, "/registrasi/klaim/K1/dokumen"},
		{http.MethodPost, "/registrasi/klaim/K1/cfs"},
		{http.MethodPost, "/registrasi/klaim/K1/pla/daftar"},
		{http.MethodPost, "/registrasi/klaim/K1/pla/catatan"},
		{http.MethodPost, "/registrasi/klaim/K1/pla"},
		{http.MethodPost, "/registrasi/klaim/K1/lod/tipe"},
		{http.MethodPost, "/registrasi/klaim/K1/lod"},
		{http.MethodPost, "/registrasi/klaim/K1/kasir/pratinjau"},
		{http.MethodPost, "/registrasi/klaim/K1/kasir"},
		{http.MethodPost, "/registrasi/klaim/K1/dla/daftar"},
		{http.MethodPost, "/registrasi/klaim/K1/dla"},
		{http.MethodPost, "/registrasi/klaim/K1/akseptasi"},
		{http.MethodPost, "/registrasi/klaim/K1/adjustment/hitung"},
		{http.MethodPost, "/registrasi/klaim/K1/adjustment"},
		{http.MethodPost, "/registrasi/klaim/K1/adjustment/komite"},
		{http.MethodGet, "/registrasi/komite"},
		{http.MethodGet, "/registrasi/komite/KM1"},
		{http.MethodPost, "/registrasi/komite/KM1/putusan"},
		{http.MethodGet, "/registrasi/rekening/123"},
		{http.MethodPost, "/registrasi/klaim/K1/penerima"},
		{http.MethodGet, "/registrasi/klaim/K1/progres"},
		{http.MethodGet, "/registrasi/wilayah/negara"},
		{http.MethodPost, "/registrasi/tugas/T1/ambil"},
		{http.MethodPost, "/registrasi/tugas/T1/selesai"},
	}
	for _, rt := range routes {
		w := e.doAs(t, noSession, rt.method, rt.path, "{}")
		requireError(t, w, http.StatusUnauthorized, "sesi_tidak_sah")
	}
}

// Rute JSON menolak badan yang tidak terbaca dengan 400 permintaan_cacat.
func TestRoutesRejectMalformedBody(t *testing.T) {
	e := newHTTPEnv(t)
	paths := []string{
		"/registrasi/klaim",
		"/registrasi/register",
		"/registrasi/register/simpan",
		"/registrasi/estimasi",
		"/registrasi/estimasi/simpan",
		"/registrasi/klaim/K1/cfs",
		"/registrasi/klaim/K1/pla/daftar",
		"/registrasi/klaim/K1/pla/catatan",
		"/registrasi/klaim/K1/pla",
		"/registrasi/klaim/K1/lod/tipe",
		"/registrasi/klaim/K1/lod",
		"/registrasi/klaim/K1/kasir/pratinjau",
		"/registrasi/klaim/K1/kasir",
		"/registrasi/klaim/K1/dla/daftar",
		"/registrasi/klaim/K1/dla",
		"/registrasi/klaim/K1/adjustment/hitung",
		"/registrasi/klaim/K1/adjustment",
		"/registrasi/klaim/K1/adjustment/komite",
		"/registrasi/komite/KM1/putusan",
		"/registrasi/klaim/K1/penerima",
		"/registrasi/tugas/T1/selesai",
	}
	for _, p := range paths {
		w := e.do(t, http.MethodPost, p, "{bukan json")
		requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
	}
}

// Peran pemanggil yang gagal dibaca menjadi 500 galat_internal, tanpa rincian.
func TestCallerResolutionFailureIsInternal(t *testing.T) {
	e := newHTTPEnv(t)
	e.groups.err = errGroupsDown
	w := e.do(t, http.MethodGet, "/registrasi/inbox", nil)
	requireError(t, w, http.StatusInternalServerError, registrasihttp.CodeInternalError)
	require.NotContains(t, w.Body.String(), errGroupsDown.Error())
}

// Membuka klaim: 201, klaim bernomor di View Polis, tugasnya ikut dikirim.
func TestStartClaimAndViewIt(t *testing.T) {
	e := newHTTPEnv(t)
	start := e.startClaim(t)
	require.Equal(t, registrasi.StageViewPolicy, start.Claim.CurrentStage)
	require.Equal(t, "ASM", start.Claim.Portal)
	require.Equal(t, firePolicy, start.Claim.Policy.Number)
	require.NotNil(t, start.Task)
	require.Equal(t, registrasi.StageViewPolicy, start.Task.Stage)
	require.NotEmpty(t, start.Task.StageName)
	require.NotEmpty(t, start.Task.CreatedAt)

	w := e.do(t, http.MethodGet, "/registrasi/klaim/"+start.Claim.ID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	view := decode[registrasihttp.ClaimResponse](t, w)
	require.Equal(t, start.Claim.Number, view.Claim.Number)
	require.NotNil(t, view.Task)
	require.True(t, view.Task.Workable)
	require.NotEmpty(t, view.Path)

	w = e.do(t, http.MethodGet, "/registrasi/inbox", nil)
	require.Equal(t, http.StatusOK, w.Code)
	inbox := decode[registrasihttp.InboxResponse](t, w)
	require.Len(t, inbox.Task, 1)
	require.Equal(t, start.Task.ID, inbox.Task[0].ID)
}

// Portal aktif wajib: tanpa portal, pembukaan klaim ditolak dan tidak ada klaim terbuka.
func TestStartClaimRequiresActivePortal(t *testing.T) {
	e := newHTTPEnv(t)
	r := request(t, http.MethodPost, "/registrasi/klaim", registrasihttp.StartRequest{PolicyNumber: firePolicy})
	r.Header.Set(headerNoPortal, "1")
	w := e.serve(r)
	// portal.ErrNotStated tidak punya pemetaan khusus di modul ini, sehingga menjadi 500.
	requireError(t, w, http.StatusInternalServerError, registrasihttp.CodeInternalError)

	w = e.do(t, http.MethodGet, "/registrasi/inbox", nil)
	require.Empty(t, decode[registrasihttp.InboxResponse](t, w).Task)
}

// Polis yang tidak ada: 422 polis_tidak_ditemukan.
func TestStartClaimUnknownPolicy(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodPost, "/registrasi/klaim", registrasihttp.StartRequest{PolicyNumber: "POL-TIDAK-ADA"})
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodePolicyNotFound)
}

// Klaim yang tidak ada: 404 klaim_tidak_ditemukan pada rute baca klaim.
func TestClaimReadRoutesUnknownClaim(t *testing.T) {
	e := newHTTPEnv(t)
	for _, p := range []string{
		"/registrasi/klaim/K-TIDAK-ADA",
		"/registrasi/klaim/K-TIDAK-ADA/survey",
		"/registrasi/klaim/K-TIDAK-ADA/dokumen",
		"/registrasi/klaim/K-TIDAK-ADA/progres",
		"/registrasi/klaim/K-TIDAK-ADA/pilihan-item",
	} {
		w := e.do(t, http.MethodGet, p, nil)
		requireError(t, w, http.StatusNotFound, registrasihttp.CodeClaimNotFound)
	}
}

// Input Register: tanggal dibaca sebagai kalender WIB, tahap berpindah ke Input Estimasi.
func TestSaveRegisterMovesToEstimate(t *testing.T) {
	e := newHTTPEnv(t)
	out := e.upToInputEstimate(t)
	require.Equal(t, "2026-06-05", out.Claim.DateOfLoss)
	require.Equal(t, "2026-06-06", out.Claim.ReportDate)
	require.Equal(t, "2026-06-07", out.Claim.DateReceived)
	require.Equal(t, "Gudang A", out.Claim.Location)
	require.Equal(t, "Pelapor Uji", out.Claim.Reporter.Name)
	require.Equal(t, registrasi.CountryIndonesia, out.Claim.Area.Country)
	require.NotEmpty(t, out.DecisionTrace)
	require.Len(t, out.Claim.InsuredItem, 1)
	require.Len(t, out.Claim.InsuredItem[0].Coverage, 1)
	cov := out.Claim.InsuredItem[0].Coverage[0]
	require.Equal(t, int64(registrasi.Rupiah(500_000_000)), cov.TSICents)
	require.Len(t, cov.Spreading, 2)
	require.Equal(t, registrasihttp.Percent(600_000), cov.Spreading[0].Share)
	require.Equal(t, "Fire", out.Claim.Policy.LineName)
}

// Tanggal yang kosong atau salah format, dan tugas kosong, menjadi 400 sebelum layanan dipanggil.
func TestSaveRegisterRejectsBadDates(t *testing.T) {
	e := newHTTPEnv(t)
	cases := map[string]func(*registrasihttp.RegisterRequest){
		"Tanggal Kejadian wajib diisi":                      func(b *registrasihttp.RegisterRequest) { b.DateOfLoss = "" },
		"Tanggal Lapor harus berformat YYYY-MM-DD":          func(b *registrasihttp.RegisterRequest) { b.ReportDate = "06/06/2026" },
		"Tanggal Terima Dokumen harus berformat YYYY-MM-DD": func(b *registrasihttp.RegisterRequest) { b.DateReceived = "x" },
		"tugas_id wajib diisi":                              func(b *registrasihttp.RegisterRequest) { b.TaskID = "" },
	}
	for message, mutate := range cases {
		for _, path := range []string{"/registrasi/register", "/registrasi/register/simpan"} {
			body := validRegister("T1")
			mutate(&body)
			w := e.do(t, http.MethodPost, path, body)
			requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
			require.Equal(t, message, decode[registrasihttp.ErrorResponse](t, w).Message)
		}
	}
}

// Tanggal Terima Dokumen kosong bukan galat bentuk: isiannya tidak tampil pada lini selain
// Travel dan PA, sehingga klaim Fire tetap dapat diregistrasi.
func TestSaveRegisterAllowsEmptyDateReceived(t *testing.T) {
	e := newHTTPEnv(t)
	reg := e.upToInputRegister(t)
	body := validRegister(reg.Task.ID)
	body.DateReceived = ""
	w := e.do(t, http.MethodPost, "/registrasi/register", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Empty(t, decode[registrasihttp.ClaimResponse](t, w).Claim.DateReceived)
}

// Tugas yang tidak ada: 404 pada register, simpan, ambil, dan selesai.
func TestTaskRoutesUnknownTask(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodPost, "/registrasi/register", validRegister("T-TIDAK-ADA"))
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/register/simpan", validRegister("T-TIDAK-ADA"))
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/tugas/T-TIDAK-ADA/ambil", nil)
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/tugas/T-TIDAK-ADA/selesai", registrasihttp.CompleteRequest{Action: "x"})
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
}

// Validasi isian Register yang gagal menjadi 422 berisi daftar pelanggaran.
func TestSaveRegisterValidationFailure(t *testing.T) {
	e := newHTTPEnv(t)
	reg := e.upToInputRegister(t)
	body := validRegister(reg.Task.ID)
	body.DateOfLoss = "2026-06-09"
	body.ReportDate = "2026-06-06"
	w := e.do(t, http.MethodPost, "/registrasi/register", body)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, "badan = %s", w.Body.String())
	resp := decode[registrasihttp.ErrorResponse](t, w)
	require.Equal(t, registrasihttp.CodeValidationFailed, resp.Code)
	require.NotEmpty(t, resp.Violation)
	require.Equal(t, resp.Violation[0].Message, resp.Message)
}

// Tombol Save menyimpan isian tanpa menutup tahap.
func TestSaveDraftKeepsStage(t *testing.T) {
	e := newHTTPEnv(t)
	reg := e.upToInputRegister(t)
	body := validRegister(reg.Task.ID)
	body.Location = "Gudang Draf"
	w := e.do(t, http.MethodPost, "/registrasi/register/simpan", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	out := decode[registrasihttp.ClaimResponse](t, w)
	require.Equal(t, registrasi.StageInputRegister, out.Claim.CurrentStage)
	require.Equal(t, "Gudang Draf", out.Claim.Location)
	require.Nil(t, out.Task)
}

// Menyelesaikan tugas yang sudah selesai: 409 tugas_sudah_selesai; tindakan salah: 400.
func TestCompleteTaskConflicts(t *testing.T) {
	e := newHTTPEnv(t)
	start := e.startClaim(t)
	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/selesai",
		registrasihttp.CompleteRequest{Action: "BUKAN-TINDAKAN"})
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeInvalidAction)

	w = e.do(t, http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/selesai",
		registrasihttp.CompleteRequest{Action: registrasi.ActionViewPolicy})
	require.Equal(t, http.StatusOK, w.Code)
	next := decode[registrasihttp.ClaimResponse](t, w)
	require.NotNil(t, next.Task)

	w = e.do(t, http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/selesai",
		registrasihttp.CompleteRequest{Action: registrasi.ActionViewPolicy})
	requireError(t, w, http.StatusConflict, registrasihttp.CodeTaskAlreadyDone)
}

// Mengambil tugas yang sudah dimiliki sendiri dijawab dengan tugas itu.
func TestClaimTaskReturnsTask(t *testing.T) {
	e := newHTTPEnv(t)
	start := e.startClaim(t)
	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/ambil", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	task := decode[registrasihttp.TaskDTO](t, w)
	require.Equal(t, start.Task.ID, task.ID)
	require.Equal(t, testOperator, task.Owner)
	require.False(t, task.Claimable)

	// Pengguna lain tidak dapat mengambil tugas yang sudah bertuan: 409.
	w = e.doAs(t, "PETUGASLAIN", http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/ambil", nil)
	requireError(t, w, http.StatusConflict, registrasihttp.CodeTaskAlreadyClaimed)

	// Pemilik lain yang menyelesaikan tugas ini: 403 bukan_pemilik_tugas.
	w = e.doAs(t, "PETUGASLAIN", http.MethodPost, "/registrasi/tugas/"+start.Task.ID+"/selesai",
		registrasihttp.CompleteRequest{Action: registrasi.ActionViewPolicy})
	requireError(t, w, http.StatusForbidden, registrasihttp.CodeNotTaskOwner)
}

// Wilayah: tingkat dikenal mengembalikan pilihan; tingkat asing 400.
func TestAreaOptions(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodGet, "/registrasi/wilayah/"+string(registrasi.AreaCountry), nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	countries := decode[registrasihttp.AreaOptionsResponse](t, w)
	require.Len(t, countries.Option, 2)

	w = e.do(t, http.MethodGet, "/registrasi/wilayah/"+string(registrasi.AreaVillage)+"?induk=10000925", nil)
	require.Equal(t, http.StatusOK, w.Code)
	villages := decode[registrasihttp.AreaOptionsResponse](t, w)
	require.Len(t, villages.Option, 1)
	require.Equal(t, "55281", villages.Option[0].PostalCode)

	w = e.do(t, http.MethodGet, "/registrasi/wilayah/planet", nil)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
}

// LineName menerjemahkan kode Group Panel; kode asing dikembalikan apa adanya.
func TestLineName(t *testing.T) {
	require.Equal(t, "Personal Accident", registrasihttp.LineName(registrasi.LinePersonalAccident))
	require.Equal(t, "Aneka", registrasihttp.LineName(registrasi.LineMiscellaneous))
	require.Equal(t, "Marine Cargo", registrasihttp.LineName(registrasi.LineMarineCargo))
	require.Equal(t, "Travel", registrasihttp.LineName(registrasi.LineTravel))
	require.Equal(t, "Fire", registrasihttp.LineName(registrasi.LineFire))
	require.Equal(t, "999", registrasihttp.LineName(registrasi.LineOfBusiness("999")))
}
