package cashierlink

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"claim-pnc/internal/registrasi"
)

type catalog map[string]string

func (c catalog) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	return c[app+"/"+kind], nil
}

// Muatan terkirim sebagai TAllPaymentData dengan Basic Auth; jawaban dengan CaseIDCashier
// diterima.
func TestTransferSendsPayloadAndReadsCaseID(t *testing.T) {
	var got map[string]any
	var user, pass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ = r.BasicAuth()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		_, _ = w.Write([]byte(`{"ResponseMessage":"SUCCESS","CaseIDCashier":"ECR-1"}`))
	}))
	defer srv.Close()

	h := New(catalog{"ASM/KASIRPAID": srv.URL}, "u", "p", nil)
	reply, err := h.Transfer(context.Background(), "ASM", "KASIRPAID", registrasi.CashierPayload{
		TAllPaymentData: []registrasi.CashierPayment{{NoTrans: "A26", Nett: "100.00"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reply.Accepted() || reply.CaseID() != "ECR-1" || !reply.Logged() {
		t.Fatalf("jawaban salah: %+v", reply)
	}
	if user != "u" || pass != "p" {
		t.Fatalf("basic auth salah: %s/%s", user, pass)
	}
	rows, _ := got["TAllPaymentData"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["NoTrans"] != "A26" {
		t.Fatalf("muatan salah: %v", got)
	}
}

// Tanpa alamat terdaftar, tidak ada yang dikirim.
func TestTransferWithoutAddressFails(t *testing.T) {
	_, err := New(catalog{}, "", "", nil).Transfer(context.Background(), "ASM", "KASIRPAID", registrasi.CashierPayload{})
	if err == nil {
		t.Fatal("harus gagal tanpa alamat")
	}
}

// Jawaban tanpa CaseIDCashier/NoTransClaim tidak diterima; NoTransClaim menjadi CaseID.
func TestParseReply(t *testing.T) {
	if r := Parse([]byte(`{"ResponseMsg":"Rekening tidak valid"}`)); r.Accepted() || r.ResponseMessage != "Rekening tidak valid" {
		t.Fatalf("penolakan salah: %+v", r)
	}
	if r := Parse([]byte(`{"NoTransClaim":"NT-9"}`)); !r.Accepted() || r.CaseID() != "NT-9" {
		t.Fatalf("NoTransClaim salah: %+v", r)
	}
	if r := Parse([]byte(`bukan json`)); r.Accepted() || r.Raw != "bukan json" {
		t.Fatalf("bukan JSON salah: %+v", r)
	}
}

// Bentuk badan mengikuti log Pega yang diterima Kasir: Nett, Deductible, KaliDeduct berupa
// angka JSON; DOL dan Panel tidak ikut.
func TestTransferSendsNumbersWithoutDOLAndPanel(t *testing.T) {
	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"ResponseMessage":"Success","CaseIDCashier":"ECR-2"}`))
	}))
	defer srv.Close()
	h := New(catalog{"ASM/KASIRPAID": srv.URL}, "", "", nil)
	_, err := h.Transfer(context.Background(), "ASM", "KASIRPAID", registrasi.CashierPayload{
		TAllPaymentData: []registrasi.CashierPayment{{Nett: "2500000.00", Deductible: "0.00", KaliDeduct: "0", DOL: "01-09-2026", Panel: "X"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		TAllPaymentData []map[string]any `json:"TAllPaymentData"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	row := got.TAllPaymentData[0]
	for _, k := range []string{"Nett", "Deductible", "KaliDeduct"} {
		if _, ok := row[k].(float64); !ok {
			t.Errorf("%s bukan angka: %#v", k, row[k])
		}
	}
	for _, k := range []string{"DOL", "Panel"} {
		if _, ok := row[k]; ok {
			t.Errorf("%s tidak boleh terkirim", k)
		}
	}
}

// HTTP 400 berpesan diteruskan sebagai jawaban (penolakan), bukan "tidak terjangkau".
func TestTransferPassesRejectionMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ResponseCode":"0","ResponseMessage":"Rekening tidak terdaftar"}`))
	}))
	defer srv.Close()
	reply, err := New(catalog{"ASM/KASIRPAID": srv.URL}, "", "", nil).
		Transfer(context.Background(), "ASM", "KASIRPAID", registrasi.CashierPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Accepted() || reply.ResponseMessage != "Rekening tidak terdaftar" {
		t.Fatalf("jawaban = %+v", reply)
	}
}
