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
