// Package cashierlink mengirim data pembayaran klaim ke sistem Kasir — Connect REST
// `SendDataPaidASMtoCashier_2` yang dipanggil `TransferCashierDataASM_act`.
//
// # Alamat adalah DATA, kredensial dari lingkungan
//
// Alamatnya dibaca dari POOLDATA.GCNM_CONNECT_REST (SERVICENAME) per APP portal dan
// TYPESERVICE (KASIRPAID, KASIRPAIDSYARIAH, KASIRIVEST) — keputusan Work Owner 2026-09-30,
// sama seperti layanan status premi. Rule Pega memuat header Basic Auth tertanam; nilainya
// TIDAK disalin (ADR-0025) — user dan password dibaca dari KASIR_USER / KASIR_PASSWORD.
//
// # Bentuk jawaban BELUM terverifikasi terhadap Kasir sungguhan
//
// Export hanya mengurai jawaban utuh ke `ReturnPaymentChas`; field yang dibaca rule-nya
// ResponseMessage, ResponseMsg, CaseIDCashier, dan NoTransClaim.
package cashierlink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// DefaultTimeout adalah batas waktu pemanggilan Kasir.
//
// Dua menit, bukan 30 detik: Kasir memproses pembayaran sebelum menjawab, dan 30 detik
// terbukti habis sebelum jawabannya datang (2026-10-07).
const DefaultTimeout = 2 * time.Minute

// ServiceCatalog adalah seam ke daftar alamat layanan (POOLDATA.GCNM_CONNECT_REST).
type ServiceCatalog interface {
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// ErrServiceAddress menandai layanan Kasir yang belum terdaftar untuk portal itu.
var ErrServiceAddress = errors.New("cashierlink: alamat layanan kasir belum terdaftar")

// HTTP memenuhi registrasi.CashierGateway.
type HTTP struct {
	catalog        ServiceCatalog
	user, password string
	client         *http.Client
}

// New membentuk klien Kasir. Client boleh nil.
func New(catalog ServiceCatalog, user, password string, client *http.Client) *HTTP {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	return &HTTP{catalog: catalog, user: user, password: password, client: client}
}

var _ registrasi.CashierGateway = (*HTTP)(nil)

// Transfer mengirim satu muatan TAllPaymentData.
func (h *HTTP) Transfer(ctx context.Context, app, service string, payload registrasi.CashierPayload) (registrasi.CashierReply, error) {
	address, err := h.catalog.ServiceAddress(ctx, app, service)
	if err != nil || strings.TrimSpace(address) == "" {
		return registrasi.CashierReply{}, fmt.Errorf("%w: TYPESERVICE %q APP %q di POOLDATA.GCNM_CONNECT_REST", ErrServiceAddress, service, app)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return registrasi.CashierReply{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(address), bytes.NewReader(body))
	if err != nil {
		return registrasi.CashierReply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.user != "" || h.password != "" {
		req.SetBasicAuth(h.user, h.password)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return registrasi.CashierReply{}, fmt.Errorf("cashierlink: memanggil kasir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return registrasi.CashierReply{}, fmt.Errorf("cashierlink: membaca jawaban kasir: %w", err)
	}
	reply := Parse(raw)
	if resp.StatusCode/100 != 2 && !reply.Accepted() {
		// Penolakan yang BERPESAN diteruskan sebagai jawaban, supaya alasannya sampai ke layar
		// (CashierRejectedError) — bukan dilaporkan sebagai Kasir yang tidak terjangkau.
		if reply.ResponseMessage != "" {
			return reply, nil
		}
		return registrasi.CashierReply{}, fmt.Errorf("cashierlink: kasir menjawab HTTP %d: %s", resp.StatusCode, snippet(reply.Raw))
	}
	return reply, nil
}

// Parse membaca jawaban Kasir; jawaban yang bukan JSON dikembalikan sebagai Raw saja.
func Parse(raw []byte) registrasi.CashierReply {
	var a struct {
		ResponseMessage any `json:"ResponseMessage"`
		ResponseMsg     any `json:"ResponseMsg"`
		CaseIDCashier   any `json:"CaseIDCashier"`
		NoTransClaim    any `json:"NoTransClaim"`
		// ResponseDataList memuat pesan per baris; dipakai bila pesan utamanya kosong.
		ResponseDataList []struct {
			ResponseMessageData any `json:"ResponseMessageData"`
		} `json:"ResponseDataList"`
		// Jawaban galat ASP.NET Web API memakai Message.
		Message any `json:"Message"`
	}
	out := registrasi.CashierReply{Raw: strings.TrimSpace(string(raw)), Body: string(raw)}
	if len(out.Raw) > 500 {
		out.Raw = out.Raw[:500]
	}
	if json.Unmarshal(raw, &a) != nil {
		return out
	}
	out.ResponseMessage = text(a.ResponseMessage)
	if out.ResponseMessage == "" {
		out.ResponseMessage = text(a.ResponseMsg)
	}
	for _, d := range a.ResponseDataList {
		if out.ResponseMessage == "" {
			out.ResponseMessage = text(d.ResponseMessageData)
		}
	}
	if out.ResponseMessage == "" {
		out.ResponseMessage = text(a.Message)
	}
	out.CaseIDCashier = text(a.CaseIDCashier)
	out.NoTransClaim = text(a.NoTransClaim)
	return out
}

// snippet memotong jawaban mentah untuk log galat.
func snippet(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

func text(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
