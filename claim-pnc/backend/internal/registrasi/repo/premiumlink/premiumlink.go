// Package premiumlink memanggil layanan status premi (`Connect REST/getPremiumPaidOn_before`)
// untuk pemeriksaan premi saat akseptasi.
//
// # Alamat layanan adalah DATA
//
// Sama seperti provider HCC/HCQ dan layanan Arsip, alamatnya dibaca dari
// POOLDATA.GCNM_CONNECT_REST (SERVICENAME) per APP dan TYPESERVICE 'PREMI'. Pega memilih
// barisnya dengan membandingkan APPLICATIONIP terhadap nama server yang ditulis di rule
// (`ConnectRestPNC_act`); di sini pembedanya kolom APP (ADR-0030).
//
// # Bentuk jawaban BELUM terverifikasi terhadap layanan sungguhan
//
// Export hanya memetakan jawaban JSON utuh ke `.PaymentData` tanpa format. Nama field
// (AgingAmount, Payment.ListInstallment[].DueDate/PaymentDate/PaymentAmount) diambil dari
// rule yang membacanya; tanggal diterima dalam bentuk `yyyyMMdd` (seperti ListInstallment
// dokumen polis) atau DateTime Pega, dan angka sebagai angka maupun teks.
package premiumlink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// ServiceKind adalah TYPESERVICE baris alamat layanan premi.
const ServiceKind = "PREMI"

// DefaultTimeout adalah batas waktu pemanggilan (10-API-STRATEGY.md §8.2). Rule Pega tidak
// memberi batas sama sekali.
const DefaultTimeout = 30 * time.Second

// ServiceCatalog adalah seam ke daftar alamat layanan luar — bentuknya sama dengan yang
// dipakai provider HCC/HCQ, sehingga satu pengisi melayani keduanya.
type ServiceCatalog interface {
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// ErrServiceAddress menandai alamat layanan premi yang belum terdaftar untuk entitas itu.
var ErrServiceAddress = errors.New("premiumlink: alamat layanan premi belum terdaftar")

// HTTP memenuhi registrasi.PremiumService.
type HTTP struct {
	catalog ServiceCatalog
	client  *http.Client
}

// New membentuk klien layanan premi. Client boleh nil.
func New(catalog ServiceCatalog, client *http.Client) *HTTP {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	return &HTTP{catalog: catalog, client: client}
}

// pegaDateTime adalah bentuk DateTime Pega yang dikirim rule tanpa format mask.
const pegaDateTime = "20060102T150405.000 GMT"

// Statement memanggil `getPremiumPaidOn_before`: POST tanpa badan, empat parameter query.
func (h *HTTP) Statement(ctx context.Context, app string, q registrasi.PremiumQuery) (registrasi.PremiumStatement, error) {
	address, err := h.catalog.ServiceAddress(ctx, app, ServiceKind)
	if err != nil || strings.TrimSpace(address) == "" {
		return registrasi.PremiumStatement{}, fmt.Errorf("%w: TYPESERVICE %q APP %q di POOLDATA.GCNM_CONNECT_REST",
			ErrServiceAddress, ServiceKind, app)
	}
	target, err := url.Parse(strings.TrimSpace(address))
	if err != nil {
		return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: alamat layanan tidak sah: %w", err)
	}
	values := target.Query()
	values.Set("noPolis", q.PolicyNumber)
	values.Set("prodKe", q.ProdKe)
	if !q.RequestedAt.IsZero() {
		values.Set("tglRequest", q.RequestedAt.UTC().Format(pegaDateTime))
	} else {
		values.Set("tglRequest", "")
	}
	values.Set("caseId", q.CaseID)
	target.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), nil)
	if err != nil {
		return registrasi.PremiumStatement{}, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: memanggil layanan premi: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: membaca jawaban layanan premi: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: layanan premi menjawab HTTP %d", resp.StatusCode)
	}
	return Parse(body)
}

type answer struct {
	AgingAmount json.RawMessage `json:"AgingAmount"`
	Payment     struct {
		ListInstallment []struct {
			DueDate       json.RawMessage `json:"DueDate"`
			PaymentDate   json.RawMessage `json:"PaymentDate"`
			PaymentAmount json.RawMessage `json:"PaymentAmount"`
		} `json:"ListInstallment"`
	} `json:"Payment"`
}

// Parse menguraikan jawaban layanan premi.
func Parse(body []byte) (registrasi.PremiumStatement, error) {
	var a answer
	if err := json.Unmarshal(body, &a); err != nil {
		return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: jawaban layanan premi bukan JSON yang dikenali: %w", err)
	}
	out := registrasi.PremiumStatement{}
	if v := text(a.AgingAmount); v != "" {
		r, ok := new(big.Rat).SetString(v)
		if !ok {
			return registrasi.PremiumStatement{}, fmt.Errorf("premiumlink: AgingAmount %q bukan angka", v)
		}
		out.AgingAmount = r
	}
	for _, i := range a.Payment.ListInstallment {
		due, err := date(text(i.DueDate))
		if err != nil {
			return registrasi.PremiumStatement{}, err
		}
		out.Installments = append(out.Installments, registrasi.PremiumInstallment{
			DueDate: due, PaymentDate: text(i.PaymentDate), PaymentAmount: text(i.PaymentAmount),
		})
	}
	return out, nil
}

// text membaca nilai JSON teks atau angka sebagai teks; null dan kosong menjadi "".
func text(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	var v string
	if json.Unmarshal(raw, &v) == nil {
		return strings.TrimSpace(v)
	}
	return s
}

// date membaca DueDate. Kosong menjadi waktu nol — cicilan tanpa jatuh tempo dianggap sudah
// jatuh tempo, seperti @CompareDates Pega atas nilai kosong.
func date(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{"20060102", pegaDateTime, "20060102T150405 GMT", time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("premiumlink: DueDate %q tidak dikenali", v)
}

var _ registrasi.PremiumService = (*HTTP)(nil)
