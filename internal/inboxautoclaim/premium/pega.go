// Package premium memenuhi seam inboxautoclaim.PremiumChecker.
//
// Dua pengisi: Pega di berkas ini — jalur nyata — dan Fake di fake.go, untuk pengujian
// serta mode memori tanpa jaringan.
//
// # Apa yang dibaca dari sistem lama
//
// `InboxAutoClaim/CekPremiAutoKlaim-Act.xml` memanggil Connect REST
// `getPremiumPaidOn_before` (`Connect REST/getPremiumPaidOn_before-ConnectREST.xml`):
//
//   - metode POST, badan permintaan KOSONG (tidak dipetakan, :120-124);
//   - empat parameter kueri: noPolis, prodKe, tglRequest (waktu saat ini), caseId;
//   - jawaban JSON dipetakan utuh ke `TempPNC2.PaymentData`, dan yang dibaca pemanggilnya
//     hanya `AgingAmount` (InsertKlaimToTable_Kredit :8382).
//
// Alamatnya TIDAK ditulis di kode. Pega mencarinya di POOLDATA.GCNM_CONNECT_REST lewat
// APPLICATIONIP; Work Owner menetapkan (2026-09-29) pencariannya lewat `APP = <alias portal>`
// dan `TYPESERVICE = 'PREMI'` — sama dengan katalog yang sudah dipakai modul lain.
//
// Tab Cek Premi memakai layanan KEDUA — `GetPremiumPaid_SPK`, TYPESERVICE `PREMI-API` —
// lihat PremiumPaidBySource.
//
// # Yang BELUM terverifikasi
//
// Bentuk jawaban kedua layanan disimpulkan dari pemetaan Pega, bukan dari jawaban
// sungguhan: keduanya belum pernah dipanggil dari aplikasi baru. `AgingAmount` dan
// `TotalPremiumPaid` karena itu dicari sebagai kunci tingkat atas maupun di dalam
// `PaymentData`, tanpa peduli besar-kecil hurufnya.
package premium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"claim-pnc/internal/inboxautoclaim"
)

// DefaultServiceKind adalah nilai TYPESERVICE yang dicari di POOLDATA.GCNM_CONNECT_REST
// (`ConnectRestPNC_act` dipanggil dengan `typeservice="PREMI"`, CekPremiAutoKlaim :421-442).
const DefaultServiceKind = "PREMI"

// DefaultTotalServiceKind adalah TYPESERVICE layanan total premi tab Cek Premi
// (`CekPremi-Act` memanggil `ConnectRestPNC_act` dengan `typeservice="PREMI-API"`).
const DefaultTotalServiceKind = "PREMI-API"

// pegaDateTime adalah bentuk DateTime Pega yang dikirim `@CurrentDateTime()` (:647).
const pegaDateTime = "20060102T150405.000 GMT"

// maxResponseBytes membatasi badan jawaban yang dibaca.
const maxResponseBytes = 1 << 20

// ServiceCatalog adalah seam ke daftar alamat layanan luar — dipenuhi pembaca
// POOLDATA.GCNM_CONNECT_REST yang sama dengan modul lain, dirakit di cmd/claimpnc.
type ServiceCatalog interface {
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// Pega memeriksa status premi lewat layanan REST milik Pega.
type Pega struct {
	catalog     ServiceCatalog
	client      *http.Client
	serviceKind string
	totalKind   string
	now         func() time.Time
}

// Options adalah bahan pembentuk Pega.
type Options struct {
	// Catalog membaca alamat layanan per portal. Wajib.
	Catalog ServiceCatalog

	// ServiceKind menimpa DefaultServiceKind.
	ServiceKind string

	// Timeout per panggilan. Bawaannya 15 detik — satu unggahan memanggil layanan ini
	// sekali per polis, dan satu polis yang menggantung tidak boleh menahan seluruhnya.
	Timeout time.Duration
	Client  *http.Client

	// Now memberi waktu untuk parameter tglRequest; bawaannya jam sistem.
	Now func() time.Time
}

// NewPega membentuk adapter cek premi dan menolak bahan yang tidak lengkap.
func NewPega(o Options) (*Pega, error) {
	if o.Catalog == nil {
		return nil, errors.New("inboxautoclaim/premium: katalog layanan wajib diisi")
	}
	client := o.Client
	if client == nil {
		limit := o.Timeout
		if limit <= 0 {
			limit = 15 * time.Second
		}
		client = &http.Client{Timeout: limit}
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	kind := strings.TrimSpace(o.ServiceKind)
	if kind == "" {
		kind = DefaultServiceKind
	}
	return &Pega{catalog: o.Catalog, client: client, serviceKind: kind, totalKind: DefaultTotalServiceKind, now: now}, nil
}

// CheckPremium memanggil layanan premi untuk satu polis.
//
// Galat apa pun dikembalikan tanpa alamat layanan di dalam pesannya — pesan galat dapat
// sampai ke log dan ke layar, dan alamat internal tidak boleh ikut (D-69).
func (p *Pega) CheckPremium(
	ctx context.Context,
	portalAlias string,
	query inboxautoclaim.PremiumQuery,
) (inboxautoclaim.PremiumAnswer, error) {
	parameter := url.Values{}
	parameter.Set("noPolis", strings.TrimSpace(query.PolicyNo))
	parameter.Set("prodKe", strings.TrimSpace(query.ProductSeq))
	parameter.Set("tglRequest", p.now().UTC().Format(pegaDateTime))
	// caseId diisi IDPEGA di Pega; saat unggah klaimnya belum ada, jadi dikirim kosong.
	parameter.Set("caseId", "")

	content, err := p.call(ctx, portalAlias, p.serviceKind, parameter)
	if err != nil {
		return inboxautoclaim.PremiumAnswer{}, err
	}
	aging, err := amountFor(content, "AgingAmount")
	if err != nil {
		return inboxautoclaim.PremiumAnswer{}, err
	}
	return inboxautoclaim.PremiumAnswer{AgingAmount: aging}, nil
}

// PremiumPaidBySource membaca total premi terbayar untuk tab Cek Premi.
//
// Padanan `InboxAutoClaim/CekPremi-Act.xml` langkah 4: Connect REST
// `GetPremiumPaid_SPK` (`Connect REST/GetPremiumPaid_SPK-ConnectREST.xml`) — POST,
// badan kosong, parameter kueri SourceOfBizCode dan BizCode (:243-256), jawaban dipetakan
// ke `PaymentData` dan yang dibaca `TotalPremiumPaid`.
//
// # Alamatnya dari katalog, tidak dari rule
//
// Rule Pega menanam alamat http-nya sendiri (pyBaseURL), padahal activity-nya lebih dulu
// membaca katalog TYPESERVICE `PREMI-API` lalu tidak memakai hasilnya. Aplikasi baru
// membaca katalog itu (D-15: tidak ada hostname di kode); pemeriksaan `-periksa`
// memastikan baris katalognya berakhir di `/getPaymentDataSumbis`.
func (p *Pega) PremiumPaidBySource(
	ctx context.Context,
	portalAlias string,
	query inboxautoclaim.PremiumCheckQuery,
) (string, error) {
	parameter := url.Values{}
	parameter.Set("SourceOfBizCode", strings.TrimSpace(query.SourceOfBusiness))
	parameter.Set("BizCode", strings.TrimSpace(query.BusinessCode))

	content, err := p.call(ctx, portalAlias, p.totalKind, parameter)
	if err != nil {
		return "", err
	}
	return amountFor(content, "TotalPremiumPaid")
}

// call mencari alamat layanan di katalog, mengirim POST berbadan kosong dengan parameter
// kueri, dan mengembalikan badan jawaban.
func (p *Pega) call(ctx context.Context, portalAlias, serviceKind string, parameter url.Values) ([]byte, error) {
	address, err := p.catalog.ServiceAddress(ctx, strings.ToUpper(strings.TrimSpace(portalAlias)), serviceKind)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxautoclaim/premium: alamat layanan APP=%q TYPESERVICE=%q tidak terbaca: %w",
			portalAlias, serviceKind, err)
	}

	target, err := url.Parse(address)
	if err != nil {
		return nil, errors.New("inboxautoclaim/premium: alamat layanan tidak sah")
	}
	merged := target.Query()
	for key, value := range parameter {
		merged[key] = value
	}
	target.RawQuery = merged.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), http.NoBody)
	if err != nil {
		return nil, errors.New("inboxautoclaim/premium: permintaan tidak dapat disusun")
	}
	request.Header.Set("Accept", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		// Hanya JENIS galatnya. *url.Error memuat alamat lengkap dan *net.OpError memuat
		// IP:port tujuan — keduanya tidak boleh sampai ke log maupun layar (D-69).
		kind := "tidak terhubung"
		var timeout interface{ Timeout() bool }
		if errors.As(err, &timeout) && timeout.Timeout() {
			kind = "batas waktu habis"
		}
		return nil, fmt.Errorf("inboxautoclaim/premium: layanan tidak dapat dihubungi (%s)", kind)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("inboxautoclaim/premium: layanan menjawab status HTTP %d", response.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("inboxautoclaim/premium: jawaban tidak terbaca: %v", err)
	}
	return content, nil
}

// amountFor mengambil satu angka dari jawaban JSON — sebagai kunci tingkat atas maupun di
// dalam `PaymentData` (tempat Pega memetakan jawabannya), tanpa peduli besar-kecil huruf.
//
// Kunci yang tidak ada berarti teks kosong. Untuk AgingAmount kosong berarti BELUM LUNAS,
// persis kondisi Pega (`AgingAmount==""`). Yang menjadi GALAT hanya jawaban yang bukan
// objek JSON.
func amountFor(content []byte, key string) (string, error) {
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.UseNumber()

	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		return "", errors.New("inboxautoclaim/premium: jawaban bukan objek JSON")
	}
	if value, ok := findKey(body, key); ok {
		return value, nil
	}
	if nested, ok := findObject(body, "PaymentData"); ok {
		if value, ok := findKey(nested, key); ok {
			return value, nil
		}
	}
	return "", nil
}

func findKey(body map[string]any, key string) (string, bool) {
	for name, value := range body {
		if !strings.EqualFold(name, key) {
			continue
		}
		switch v := value.(type) {
		case nil:
			return "", true
		case string:
			return strings.TrimSpace(v), true
		case json.Number:
			return v.String(), true
		default:
			return fmt.Sprint(v), true
		}
	}
	return "", false
}

func findObject(body map[string]any, key string) (map[string]any, bool) {
	for name, value := range body {
		if strings.EqualFold(name, key) {
			nested, ok := value.(map[string]any)
			return nested, ok
		}
	}
	return nil, false
}

var _ inboxautoclaim.PremiumChecker = (*Pega)(nil)
