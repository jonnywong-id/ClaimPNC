// Package absenlink membaca absensi PIC Teknik — Connect REST `ServiceGetDataAbsenPIC`
// yang dipanggil `getRandomTeam_act` step 15.2.
//
// Bentuk permintaan, dari `Connect REST/ServiceGetDataAbsenPIC-ConnectREST.xml`:
//
//	GET {ABSEN_PIC_URL}/prweb/PRRestService/HCC/Absen/attendance/{PIC}/{yyyyMMdd}?caseId={nomor klaim}
//
// {yyyyMMdd} adalah `@getCurrentDateStamp()` (tanggal WIB hari ini). Jawabannya disimpan
// Pega sebagai teks, dibuang semua tanda `[` dan `]`-nya (step 15.5), lalu diurai sebagai
// satu objek JSON (step 15.6). Field yang dibaca: Day, Holiday, RuleTimeIn, TimeIn.
package absenlink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// Path adalah jalur sumber daya rule Pega, tanpa dua segmen terakhir.
const Path = "/prweb/PRRestService/HCC/Absen/attendance/"

// HTTP memenuhi registrasi.AttendanceSource.
type HTTP struct {
	base           string
	user, password string
	client         *http.Client
}

// New membentuk klien absensi. Client boleh nil.
func New(baseURL, user, password string, client *http.Client) *HTTP {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTP{base: strings.TrimRight(strings.TrimSpace(baseURL), "/"), user: user, password: password, client: client}
}

var _ registrasi.AttendanceSource = (*HTTP)(nil)

// Attendance memenuhi registrasi.AttendanceSource.
func (h *HTTP) Attendance(ctx context.Context, operator string, date time.Time, claimNumber string) (registrasi.Attendance, error) {
	address := h.base + Path + url.PathEscape(strings.TrimSpace(operator)) + "/" + date.Format("20060102") +
		"?caseId=" + url.QueryEscape(strings.TrimSpace(claimNumber))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return registrasi.Attendance{}, fmt.Errorf("absenlink: alamat tidak sah: %w", err)
	}
	if h.user != "" || h.password != "" {
		req.SetBasicAuth(h.user, h.password)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return registrasi.Attendance{}, fmt.Errorf("absenlink: memanggil layanan absensi: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return registrasi.Attendance{}, fmt.Errorf("absenlink: membaca jawaban: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return registrasi.Attendance{}, fmt.Errorf("absenlink: layanan absensi menjawab HTTP %d", resp.StatusCode)
	}
	return Parse(raw)
}

// Parse meniru step 15.5–15.6: buang `[` dan `]`, lalu urai satu objek JSON.
func Parse(raw []byte) (registrasi.Attendance, error) {
	text := strings.NewReplacer("[", "", "]", "").Replace(string(raw))
	if strings.TrimSpace(text) == "" {
		return registrasi.Attendance{}, nil
	}
	var a struct {
		Day        any `json:"Day"`
		Holiday    any `json:"Holiday"`
		RuleTimeIn any `json:"RuleTimeIn"`
		TimeIn     any `json:"TimeIn"`
	}
	if err := json.Unmarshal([]byte(text), &a); err != nil {
		return registrasi.Attendance{}, fmt.Errorf("absenlink: jawaban bukan JSON yang dikenali: %w", err)
	}
	return registrasi.Attendance{
		Day: str(a.Day), Holiday: str(a.Holiday), RuleTimeIn: str(a.RuleTimeIn), TimeIn: str(a.TimeIn),
	}, nil
}

func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", t))
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
