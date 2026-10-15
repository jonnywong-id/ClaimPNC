package registrasihttp_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	registrasihttp "claim-pnc/internal/registrasi/http"
)

// Input Estimasi: simpan menghitung rupiah, tipe asing dan tanggal salah ditolak 400.
func TestEstimateSaveAndBadInput(t *testing.T) {
	e := newHTTPEnv(t)
	est := e.upToInputEstimate(t)

	w := e.do(t, http.MethodPost, "/registrasi/estimasi/simpan", oneEstimate(est.Task.ID, registrasi.Rupiah(50_000_000)))
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	claim := decode[registrasihttp.ClaimResponse](t, w).Claim
	item := claim.InsuredItem[0].Coverage[0].Item
	require.Len(t, item, 1)
	require.Equal(t, "Gudang", item[0].Name)
	require.Len(t, item[0].Estimation, 1)
	est0 := item[0].Estimation[0]
	require.Equal(t, int64(registrasi.Rupiah(50_000_000)), est0.ValueCents)
	require.Equal(t, int64(registrasi.Rupiah(50_000_000)), est0.RupiahCents)
	require.Equal(t, "2026-06-08", est0.Date)
	require.False(t, est0.FaceSheet)

	bad := oneEstimate(est.Task.ID, 1)
	bad.Object[0].Coverage[0].Item[0].Estimation[0].Type = "Z"
	for _, path := range []string{"/registrasi/estimasi/simpan", "/registrasi/estimasi"} {
		w = e.do(t, http.MethodPost, path, bad)
		requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
		require.Contains(t, decode[registrasihttp.ErrorResponse](t, w).Message, "Tipe Estimasi")
	}
	bad = oneEstimate(est.Task.ID, 1)
	bad.Object[0].Coverage[0].Item[0].Estimation[0].Date = "08-06-2026"
	w = e.do(t, http.MethodPost, "/registrasi/estimasi", bad)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
	require.Equal(t, "Tanggal Estimasi harus berformat YYYY-MM-DD", decode[registrasihttp.ErrorResponse](t, w).Message)

	// Tugas yang tidak ada: galat layanan diteruskan sebagai 404.
	w = e.do(t, http.MethodPost, "/registrasi/estimasi/simpan", oneEstimate("T-TIDAK-ADA", 1))
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/estimasi", oneEstimate("T-TIDAK-ADA", 1))
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)

	// Next tanpa Claim Face Sheet ditolak validasi.
	w = e.do(t, http.MethodPost, "/registrasi/estimasi", oneEstimate(est.Task.ID, registrasi.Rupiah(50_000_000)))
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeValidationFailed)
}

// Pilihan item dan mata uang dibaca dari master; CFS menghasilkan PDF.
func TestItemOptionsCurrenciesAndFaceSheet(t *testing.T) {
	e := newHTTPEnv(t)
	est := e.upToInputEstimate(t)

	w := e.do(t, http.MethodGet, "/registrasi/klaim/"+est.Claim.ID+"/pilihan-item?objek=OBJ-1", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	options := decode[registrasihttp.ItemOptionsResponse](t, w)
	require.NotNil(t, options.Option)

	w = e.do(t, http.MethodGet, "/registrasi/mata-uang", nil)
	require.Equal(t, http.StatusOK, w.Code)
	currencies := decode[registrasihttp.CurrenciesResponse](t, w)
	require.NotEmpty(t, currencies.Option)
	require.NotEmpty(t, currencies.Option[0].ID)

	// CFS tanpa estimasi baru ditolak.
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/cfs",
		registrasihttp.FaceSheetRequest{TaskID: est.Task.ID, Object: 1, Coverage: 1})
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeValidationFailed)

	w = e.do(t, http.MethodPost, "/registrasi/estimasi/simpan", oneEstimate(est.Task.ID, registrasi.Rupiah(50_000_000)))
	require.Equal(t, http.StatusOK, w.Code)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/cfs",
		registrasihttp.FaceSheetRequest{TaskID: est.Task.ID, Object: 1, Coverage: 1})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Contains(t, w.Header().Get("Content-Disposition"), `attachment; filename="`)
	require.True(t, strings.HasPrefix(w.Body.String(), "%PDF"))
}

// PLA: daftar menerbitkan per anggota koasuransi, catatan tersimpan, cetak semua = ZIP.
func TestPLAListNotesAndPrint(t *testing.T) {
	e := newHTTPEnv(t)
	e.pla.Coins[firePolicy] = []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Leader: true, Share: 700_000, HasShare: true},
		{ID: "C1", Name: "ANGGOTA SATU", Share: 200_000, HasShare: true},
		{ID: "C2", Name: "ANGGOTA DUA", Share: 100_000, HasShare: true},
	}
	est := e.upToInputEstimate(t)
	w := e.do(t, http.MethodPost, "/registrasi/estimasi/simpan", oneEstimate(est.Task.ID, registrasi.Rupiah(50_000_000)))
	require.Equal(t, http.StatusOK, w.Code)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/cfs",
		registrasihttp.FaceSheetRequest{TaskID: est.Task.ID, Object: 1, Coverage: 1})
	require.Equal(t, http.StatusOK, w.Code)

	body := registrasihttp.PLARequest{TaskID: est.Task.ID, Object: 1, Coverage: 1}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/pla/daftar", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	list := decode[registrasihttp.PLAListResponse](t, w)
	require.Equal(t, 2, list.Issued)
	require.Len(t, list.PLA, 2)
	require.Equal(t, "ANGGOTA SATU", list.PLA[0].Recipient)
	number := list.PLA[0].Number

	body.Notes = map[string]string{number: "Catatan baru"}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/pla/catatan", body)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "Catatan baru", decode[registrasihttp.PLAListResponse](t, w).PLA[0].Note)

	body.Notes = map[string]string{"BUKAN-MILIK": "x"}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/pla/catatan", body)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeInvalidAction)

	all := registrasihttp.PLARequest{TaskID: est.Task.ID, Object: 1, Coverage: 1}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/pla", all)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)

	one := all
	one.Number = number
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+est.Claim.ID+"/pla", one)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "PLACOINS"+number+".pdf")

	// Tugas yang bukan milik klaim itu: 400 pada ketiga rute PLA.
	for _, p := range []string{"pla/daftar", "pla/catatan", "pla"} {
		w = e.do(t, http.MethodPost, "/registrasi/klaim/K-TIDAK-ADA/"+p, all)
		requireError(t, w, http.StatusBadRequest, registrasihttp.CodeInvalidAction)
	}
}

// Grid Adjustment: hitung tanpa simpan, tambah menyimpan baris; isian kosong ditolak 422.
func TestSettlementPreviewAndAdd(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.upToChooseSurveyor(t)

	w := e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment/hitung", finalSettlement(task.ID))
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	preview := decode[registrasihttp.SettlementPreviewResponse](t, w)
	require.Equal(t, int64(registrasi.Rupiah(1_000_000)), preview.Line.RiskValueCents)
	require.Equal(t, int64(registrasi.Rupiah(9_000_000)), preview.Line.GrossCents)
	require.Equal(t, registrasi.PaymentTypeName(registrasi.PaymentFinal), preview.Line.PaymentTypeName)
	require.Len(t, preview.Spreading, 2)

	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment", finalSettlement(task.ID))
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	claim := decode[registrasihttp.ClaimResponse](t, w).Claim
	lines := claim.InsuredItem[0].Coverage[0].Adjustment
	require.Len(t, lines, 1)
	require.Equal(t, int64(registrasi.Rupiah(9_000_000)), lines[0].ValueCents)
	require.False(t, lines[0].CashierTransferred)

	empty := registrasihttp.SettlementRequest{TaskID: task.ID, Object: 1, Coverage: 1}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment", empty)
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeValidationFailed)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment/hitung", registrasihttp.SettlementRequest{TaskID: "T-TIDAK-ADA"})
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
}

// Komite: transfer membentuk kasus, daftar menunggu per anggota, putusan berjenjang, bukan
// giliran 403, komite asing 404.
func TestCommitteeTransferListAndDecide(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.withSettlement(t)

	w := e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment/komite",
		registrasihttp.CommitteeTransferRequest{TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	transfer := decode[registrasihttp.CommitteeTransferResponse](t, w)
	require.Equal(t, "berjalan", transfer.Committee.Status)
	require.Equal(t, "KOMITE01", transfer.Committee.Current)
	require.Len(t, transfer.Committee.Members, 2)
	id := transfer.Committee.ID

	w = e.doAs(t, "KOMITE01", http.MethodGet, "/registrasi/komite", nil)
	require.Equal(t, http.StatusOK, w.Code)
	pending := decode[registrasihttp.CommitteeListResponse](t, w)
	require.Len(t, pending.Items, 1)
	require.Equal(t, id, pending.Items[0].CommitteeID)
	require.Equal(t, 1, pending.Items[0].Level)
	require.Equal(t, 2, pending.Items[0].Levels)
	require.Equal(t, firePolicy, pending.Items[0].PolicyNo)

	w = e.doAs(t, "KOMITE02", http.MethodPost, "/registrasi/komite/"+id+"/putusan",
		registrasihttp.CommitteeDecisionRequest{Decision: registrasi.DecisionApprove})
	requireError(t, w, http.StatusForbidden, registrasihttp.CodeNotCommitteeTurn)

	w = e.doAs(t, "KOMITE01", http.MethodPost, "/registrasi/komite/"+id+"/putusan",
		registrasihttp.CommitteeDecisionRequest{Decision: registrasi.DecisionApprove, Note: "ok"})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	after := decode[registrasihttp.CommitteeDTO](t, w)
	require.Equal(t, "KOMITE02", after.Current)
	require.Equal(t, "ok", after.Members[0].Note)
	require.NotEmpty(t, after.Members[0].DecidedAt)

	w = e.doAs(t, "KOMITE02", http.MethodPost, "/registrasi/komite/"+id+"/putusan",
		registrasihttp.CommitteeDecisionRequest{Decision: registrasi.DecisionReject})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "ditolak", decode[registrasihttp.CommitteeDTO](t, w).Status)

	w = e.do(t, http.MethodGet, "/registrasi/komite/"+id, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ditolak", decode[registrasihttp.CommitteeDTO](t, w).Status)

	w = e.do(t, http.MethodGet, "/registrasi/komite/KM-TIDAK-ADA", nil)
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeCommitteeNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/komite/KM-TIDAK-ADA/putusan",
		registrasihttp.CommitteeDecisionRequest{Decision: registrasi.DecisionApprove})
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeCommitteeNotFound)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/adjustment/komite",
		registrasihttp.CommitteeTransferRequest{TaskID: "T-TIDAK-ADA", Object: 1, Coverage: 1, Adjustment: 1})
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
}

// Komite yang disetujui seluruh jenjang berstatus "disetujui".
func TestCommitteeApprovedStatus(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.approved(t)
	ctx := context.Background()
	claim, err := e.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	id := claim.InsuredItem[0].Coverage[0].Settlement[0].CommitteeCaseID
	require.NotEmpty(t, id)
	w := e.do(t, http.MethodGet, "/registrasi/komite/"+id, nil)
	require.Equal(t, http.StatusOK, w.Code)
	c := decode[registrasihttp.CommitteeDTO](t, w)
	require.Equal(t, "disetujui", c.Status)
	require.Empty(t, c.Current)
}

// Master Rekening: rekening dikenal dibaca, rekening asing 404; penerima tak lengkap 422.
func TestFindAccountAndSaveReceiver(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodGet, "/registrasi/rekening/1234567890", nil)
	require.Equal(t, http.StatusOK, w.Code)
	account := decode[registrasihttp.AccountDTO](t, w)
	require.Equal(t, "PT CONTOH PENERIMA", account.Name)
	require.Equal(t, "2026-07-03", account.CommitteeApprovedAt)
	require.Empty(t, account.CashierApprovedAt)

	w = e.do(t, http.MethodGet, "/registrasi/rekening/000", nil)
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeAccountNotFound)

	task := e.upToChooseSurveyor(t)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/penerima", registrasihttp.ReceiverRequest{
		TaskID: task.ID, ReceiverID: "1", AccountNo: "1234567890", Email: "a@contoh.internal",
	})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	claim := decode[registrasihttp.ClaimResponse](t, w).Claim
	require.NotEmpty(t, claim.Receiver)
	require.Equal(t, "1234567890", claim.Receiver[0].AccountNo)

	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/penerima", registrasihttp.ReceiverRequest{TaskID: task.ID})
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeValidationFailed)
}

func multipartAcceptance(t *testing.T, form registrasihttp.AcceptanceRequest, withFile bool) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	raw, err := json.Marshal(form)
	require.NoError(t, err)
	require.NoError(t, mw.WriteField("isian", string(raw)))
	if withFile {
		fw, err := mw.CreateFormFile("berkas", "lod.pdf")
		require.NoError(t, err)
		_, err = fw.Write([]byte("%PDF-1.4"))
		require.NoError(t, err)
		require.NoError(t, mw.WriteField("jenis_dokumen", " 14904 "))
		require.NoError(t, mw.WriteField("catatan_berkas", "LOD"))
	}
	require.NoError(t, mw.Close())
	return &buf, mw.FormDataContentType()
}

func lodValue(v int64) *int64 { return &v }

func acceptanceForm(taskID string) registrasihttp.AcceptanceRequest {
	return registrasihttp.AcceptanceRequest{
		TaskID: taskID, Object: 1, Coverage: 1, Adjustment: 1,
		LODStatus: registrasi.LODAgreed, ReceiveDate: "2026-06-09", PayableDate: "2026-06-09",
		LODValueCents: lodValue(int64(registrasi.Rupiah(5_000_000))), ReceiverID: "1",
		CommitteeName: "KOMITE02", Remark: "OK", MinutesNote: "Berita acara",
	}
}

func (e httpEnv) postAcceptance(t *testing.T, claimID string, form registrasihttp.AcceptanceRequest, withFile bool) *httptestRecorder {
	t.Helper()
	body, contentType := multipartAcceptance(t, form, withFile)
	r := request(t, http.MethodPost, "/registrasi/klaim/"+claimID+"/akseptasi", nil)
	r.Body = ioNopCloser(body)
	r.ContentLength = int64(body.Len())
	r.Header.Set("Content-Type", contentType)
	return e.serve(r)
}

// Akseptasi, LOD, DLA, dan Transfer Kasir berurutan seperti di layar.
func TestAcceptanceLODDLAAndCashier(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.approved(t)

	// Tipe PDF dialog Print LOD, lalu unduhan LOD.
	lod := registrasihttp.LODRequest{TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1, Type: "4"}
	w := e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/lod/tipe", lod)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	types := decode[registrasihttp.LODTypesResponse](t, w)
	require.NotEmpty(t, types.Types)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/lod", lod)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.Len(t, e.acceptance.LODPrint, 1)

	// Isian tanggal yang salah ditolak sebelum layanan dipanggil.
	bad := acceptanceForm(task.ID)
	bad.PayableDate = "kemarin"
	w = e.postAcceptance(t, task.ClaimID, bad, false)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
	require.Equal(t, "tanggal_boleh_bayar harus berformat YYYY-MM-DD", decode[registrasihttp.ErrorResponse](t, w).Message)
	bad = acceptanceForm(task.ID)
	bad.ReceiveDate = "x"
	w = e.postAcceptance(t, task.ClaimID, bad, false)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)
	bad = acceptanceForm(task.ID)
	bad.AnalystReceiveDate = "x"
	w = e.postAcceptance(t, task.ClaimID, bad, false)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)

	// Simpan akseptasi beserta berkas LOD.
	w = e.postAcceptance(t, task.ClaimID, acceptanceForm(task.ID), true)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	claim := decode[registrasihttp.ClaimResponse](t, w).Claim
	line := claim.InsuredItem[0].Coverage[0].Adjustment[0]
	require.NotEmpty(t, line.AcceptedNo)
	require.Equal(t, "1", line.AcceptanceLODStatus)

	// DLA: daftar menerbitkan, cetak satu nomor menghasilkan PDF.
	e.dla.Policies[firePolicy] = registrasi.DLAPolicy{Coins: []registrasi.PLACoinsMember{
		{ID: "1", Name: "PT ASURANSI SINAR MAS", Leader: true, Share: registrasi.Percent(70 * 10_000), HasShare: true},
		{ID: "77", Name: "KOASURADUR UJI", Share: registrasi.Percent(30 * 10_000), HasShare: true},
	}}
	dla := registrasihttp.DLARequest{TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/dla/daftar", dla)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	list := decode[registrasihttp.DLAListResponse](t, w)
	require.NotEmpty(t, list.DLA)
	require.NotNil(t, list.Warning)
	number := list.DLA[0].Number

	dla.Number = number
	dla.Remarks = map[string]string{number: "Catatan uji"}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/dla", dla)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "DLACOINS"+number+".pdf")

	// Transfer Kasir: pratinjau lalu kirim.
	e.cashier.Banks["BANK CONTOH"] = "001"
	e.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}
	cashier := registrasihttp.CashierRequest{TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/kasir/pratinjau", cashier)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	preview := decode[registrasihttp.CashierPreviewResponse](t, w)
	require.Empty(t, preview.Problem)
	require.Equal(t, "PT CONTOH PENERIMA", preview.Receiver)
	require.Equal(t, line.AcceptedNo, preview.AcceptedNo)

	// Kasir menolak: 422 dengan pesan Kasir; tidak terhubung: 502.
	e.cashier.Reply = registrasi.CashierReply{ResponseMessage: "Rekening diblokir"}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/kasir", cashier)
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeCashierRejected)
	require.Equal(t, "Rekening diblokir", decode[registrasihttp.ErrorResponse](t, w).Message)
	e.cashier.Err = errors.New("timeout")
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/kasir", cashier)
	requireError(t, w, http.StatusBadGateway, registrasihttp.CodeCashierUnavailable)
	// Kasir menerima tetapi tidak menjawab dalam batas waktu: 504, pembayaran MUNGKIN sudah
	// diproses — pesannya meminta petugas memeriksa Kasir, bukan menyatakan tidak terkirim.
	e.cashier.Err = fmt.Errorf("cashierlink: memanggil kasir: %w", context.DeadlineExceeded)
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/kasir", cashier)
	requireError(t, w, http.StatusGatewayTimeout, registrasihttp.CodeCashierNoReply)
	require.Contains(t, decode[registrasihttp.ErrorResponse](t, w).Message, "may already have been received")

	e.cashier.Err = nil
	e.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}
	w = e.do(t, http.MethodPost, "/registrasi/klaim/"+task.ClaimID+"/kasir", cashier)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	after := decode[registrasihttp.ClaimResponse](t, w).Claim
	require.Equal(t, string(registrasi.StatusClaimTransferCashier), after.ClaimStatus)
	// Penandaan baris dilakukan penyimpanan Kasir (memori mencatatnya di Marked).
	require.Len(t, e.cashier.Marked, 1)
	require.Equal(t, "ECR-1", e.cashier.Marked[0].CaseID)

	// Riwayat baris: Status Penerimaan Komite dan Histori Transfer Kasir (InputAdjustment_sect).
	w = e.do(t, http.MethodGet, "/registrasi/klaim/"+task.ClaimID+"/adjustment/riwayat?objek=1&coverage=1&adjustment=1", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	history := decode[registrasihttp.SettlementHistoryResponse](t, w)
	require.NotEmpty(t, history.Committee)
	for _, m := range history.Committee {
		require.Equal(t, registrasi.DecisionApprove, m.Decision)
		require.NotEmpty(t, m.Name)
	}
	require.Len(t, history.Cashier, 1)
	require.Equal(t, registrasi.CashierLogReasonTransfer, history.Cashier[0].Note)
	require.Equal(t, registrasi.CashierLogStatusTransfer, history.Cashier[0].Status)

	w = e.do(t, http.MethodGet, "/registrasi/klaim/"+task.ClaimID+"/adjustment/riwayat?objek=1&coverage=1&adjustment=9", nil)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeInvalidAction)
	w = e.do(t, http.MethodGet, "/registrasi/klaim/K-TIDAK-ADA/adjustment/riwayat?objek=1&coverage=1&adjustment=1", nil)
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeClaimNotFound)

	// Tugas yang bukan milik klaim itu pada rute akhir: 400.
	for _, p := range []string{"lod/tipe", "lod", "dla/daftar", "dla", "kasir/pratinjau", "kasir"} {
		w = e.do(t, http.MethodPost, "/registrasi/klaim/K-TIDAK-ADA/"+p, cashier)
		requireError(t, w, http.StatusBadRequest, registrasihttp.CodeInvalidAction)
	}
}

// Akseptasi: badan bukan multipart atau isian bukan JSON ditolak 400; tanpa portal 500;
// galat layanan diteruskan.
func TestAcceptanceRejectsUnreadableForm(t *testing.T) {
	e := newHTTPEnv(t)
	w := e.do(t, http.MethodPost, "/registrasi/klaim/K1/akseptasi", "{}")
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("isian", "{bukan json"))
	require.NoError(t, mw.Close())
	r := request(t, http.MethodPost, "/registrasi/klaim/K1/akseptasi", nil)
	r.Body = ioNopCloser(&buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w = e.serve(r)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeMalformedRequest)

	r = request(t, http.MethodPost, "/registrasi/klaim/K1/akseptasi", nil)
	r.Header.Set(headerNoPortal, "1")
	w = e.serve(r)
	requireError(t, w, http.StatusInternalServerError, registrasihttp.CodeInternalError)

	w = e.postAcceptance(t, "K-TIDAK-ADA", acceptanceForm("T-TIDAK-ADA"), false)
	requireError(t, w, http.StatusNotFound, registrasihttp.CodeTaskNotFound)
}

// Unggah Dokumen: berkas masuk checklist dan jawabannya checklist terbarui.
func TestUploadDocument(t *testing.T) {
	e := newHTTPEnv(t)
	start := e.startClaim(t)
	ctx := context.Background()
	claim, err := e.store.Get(ctx, start.Claim.ID)
	require.NoError(t, err)
	claim.Policy.BusinessCode = "10027"
	require.NoError(t, e.store.Save(ctx, claim))

	upload := func(field, fileName, kind string, content []byte, noPortal bool) *httptestRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		if field != "" {
			fw, err := mw.CreateFormFile(field, fileName)
			require.NoError(t, err)
			_, err = fw.Write(content)
			require.NoError(t, err)
		}
		require.NoError(t, mw.WriteField("jenis_dokumen", kind))
		require.NoError(t, mw.WriteField("catatan", "dari tertanggung"))
		require.NoError(t, mw.Close())
		r := request(t, http.MethodPost, "/registrasi/klaim/"+start.Claim.ID+"/dokumen", nil)
		r.Body = ioNopCloser(&buf)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		if noPortal {
			r.Header.Set(headerNoPortal, "1")
		}
		return e.serve(r)
	}

	w := upload("berkas", "Pelaporan.PDF", " 14901 ", []byte("%PDF-1.4"), false)
	require.Equal(t, http.StatusCreated, w.Code, "badan = %s", w.Body.String())
	docs := decode[registrasihttp.DocumentsResponse](t, w)
	require.NotEmpty(t, docs.Category)
	require.Len(t, docs.Attachment, 1)
	require.True(t, docs.Attachment[0].Stored)
	require.Equal(t, "pdf", docs.Attachment[0].MimeType)
	require.NotEmpty(t, docs.Attachment[0].UploadedAt)
	uploaded := map[string]int{}
	for _, c := range docs.Category {
		for _, row := range c.Row {
			uploaded[row.ID] = row.Uploaded
		}
	}
	require.Equal(t, 1, uploaded["14901"])

	w = e.do(t, http.MethodGet, "/registrasi/klaim/"+start.Claim.ID+"/dokumen", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, decode[registrasihttp.DocumentsResponse](t, w).Attachment, 1)

	// Tanpa berkas: 400 berkas kosong; jenis di luar checklist: 422.
	w = upload("", "", "14901", nil, false)
	requireError(t, w, http.StatusBadRequest, registrasihttp.CodeDocumentUpload)
	w = upload("berkas", "x.pdf", "99999", []byte("%PDF"), false)
	requireError(t, w, http.StatusUnprocessableEntity, registrasihttp.CodeDocumentUpload)
	w = upload("berkas", "x.pdf", "14901", []byte("%PDF"), true)
	requireError(t, w, http.StatusInternalServerError, registrasihttp.CodeInternalError)
}

// Tab Survey dan Progress membaca catatan warisan; tanggal dikirim dalam WIB.
func TestSurveysAndProgress(t *testing.T) {
	e := newHTTPEnv(t)
	start := e.startClaim(t)
	number := start.Claim.Number
	moment := time.Date(2026, time.June, 9, 17, 30, 0, 0, time.UTC)
	e.records.Survey = map[string][]registrasi.Survey{number: {{
		CaseID: "SV-1", Type: "INTERNAL", SurveyorName: "Surveyor", Date: moment, InputDate: moment, Status: "DONE",
	}}}
	e.records.ProgressEntry = map[string][]registrasi.ProgressEntry{number: {{
		Seq: 1, InputAt: moment, Status1: "014", Note: "progres", InputBy: testOperator,
	}}}
	e.records.Communication = map[string][]registrasi.Communication{number: {{
		CaseID: "KM-1", ID: "1", SentAt: moment, Message: "halo", Reply: "ya",
	}}}

	w := e.do(t, http.MethodGet, "/registrasi/klaim/"+start.Claim.ID+"/survey", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	surveys := decode[registrasihttp.SurveysResponse](t, w)
	require.Len(t, surveys.Survey, 1)
	require.Equal(t, "SV-1", surveys.Survey[0].CaseID)
	require.Equal(t, "2026-06-10", surveys.Survey[0].Date)

	w = e.do(t, http.MethodGet, "/registrasi/klaim/"+start.Claim.ID+"/progres", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	progress := decode[registrasihttp.ProgressResponse](t, w)
	require.Len(t, progress.Progress, 1)
	require.Equal(t, "2026-06-10T00:30:00+07:00", progress.Progress[0].InputAt)
	require.Empty(t, progress.Progress[0].NextFollowUp)
	require.Len(t, progress.Communication, 1)
	require.Equal(t, "ya", progress.Communication[0].Reply)
	require.Empty(t, progress.Communication[0].RepliedAt)
}
