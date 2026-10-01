// Package pega memenuhi seam inboxrclpucl.ClaimActions dengan memanggil layanan REST Pega.
//
// # Kenapa memanggil Pega, bukan menulis sendiri
//
// Alasannya ada di `inboxrclpucl.ClaimActions`, dan ia bukan soal kepatuhan `P-1` melainkan
// soal klaim yang berhenti bergerak: inbox Analyst membaca `PC_ASM_FW_GCNMFW_WORK` INNER JOIN
// `PC_ASSIGN_WORKLIST`, sehingga klaim sampai ke Analyst HANYA lewat baris penugasan yang
// dibuat mesin alur kerja Pega. Menulis kolom persetujuan dari sini membuat klaim hilang dari
// antrean PUCL tanpa sampai ke siapa pun.
//
// Polanya bukan hal baru di aplikasi ini: `Service REST/` sudah memuat empat layanan masuk
// pada kelas `ASM-FW-GCNMFW-Work-PNC`, salah satunya menulis data klaim atas permintaan sistem
// luar.
//
// # Saat Pega dimatikan
//
// Paket ini diganti pengisi yang menulis sendiri. Yang memakainya — usecase dan transport —
// tidak berubah sedikit pun, karena keduanya hanya mengenal seam-nya.
package pega

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/inboxrclpucl"
)

// Client memanggil layanan REST Pega untuk tindakan klaim.
type Client struct {
	// baseURL adalah alamat layanan. KOSONG berarti layanannya belum disediakan.
	baseURL string

	// path adalah `pyResourcePath` layanannya.
	path string

	// user dan password mengisi Basic Auth. Keduanya kosong berarti permintaan dikirim
	// TANPA otentikasi — sah hanya bila Pega memang mengeksposnya begitu.
	user     string
	password string

	http *http.Client
}

// Config adalah tatanan pemanggil.
type Config struct {
	// BaseURL berasal dari konfigurasi per portal. Kosong berarti belum tersedia — dan itu
	// keadaan yang SAH hari ini, bukan kekeliruan tatanan.
	BaseURL string

	// Path adalah `pyResourcePath` layanannya. Kosong memakai resourcePUCLPost.
	//
	// # Kenapa ia KONFIGURASI, bukan konstanta
	//
	// Karena namanya ditentukan Tim Pega, bukan kami. Menjadikannya konstanta berarti nama
	// yang berbeda dari usulan kami menuntut perubahan KODE, bangun ulang, dan rilis — untuk
	// satu kata. Sebagai konfigurasi, ia satu baris di berkas lingkungan.
	Path string

	// User dan Password mengisi Basic Auth.
	//
	// Keduanya kosong mengirim permintaan TANPA otentikasi. Itu dibiarkan mungkin karena
	// keempat layanan Pega yang sudah ada memang ber-`pyUseAuthentication=false` — tetapi
	// untuk layanan INI kami meminta sebaliknya (`permintaan-artefak-pega.md` §12.8e).
	User     string
	Password string

	// Timeout wajib. Tanpa batas waktu, satu layanan yang menggantung akan menahan permintaan
	// pengguna tanpa batas (`10-API-STRATEGY.md` §8.2).
	Timeout time.Duration
}

// NewClient membentuk pemanggil layanan Pega.
//
// BaseURL kosong TIDAK menghasilkan galat di sini. Ia menghasilkan pemanggil yang setiap
// tindakannya mengembalikan ErrPegaServiceUnavailable — supaya aplikasi tetap menyala dan
// layarnya tetap dapat dibaca, dan supaya yang gagal hanyalah tindakan yang memang belum dapat
// dijalankan.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	path := strings.Trim(strings.TrimSpace(cfg.Path), "/")
	if path == "" {
		path = resourceActionClaimPUCL
	}
	return &Client{
		baseURL:  strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		path:     path,
		user:     strings.TrimSpace(cfg.User),
		password: cfg.Password,
		http:     &http.Client{Timeout: timeout},
	}
}

// sendToAnalystRequest adalah badan permintaan ke layanan Pega.
//
// Nama isiannya mengikuti NAMA PARAMETER `PUCLPost` apa adanya — `Status`, `idObj`, `idCov`,
// `idAdj` — bukan dinamai ulang mengikuti gaya kami. Ia kontrak dengan sistem lain, dan
// menamainya ulang hanya menambah satu terjemahan yang dapat salah di antara dua pihak.
type sendToAnalystRequest struct {
	CaseNumber   string `json:"caseNumber"`
	Status       string `json:"Status"`
	IDObject     string `json:"idObj"`
	IDCoverage   string `json:"idCov"`
	IDAdjustment string `json:"idAdj"`
	Tipe         string `json:"tipe"`
	Note         string `json:"note"`
	Caller       string `json:"caller"`
}

// statusSendToAnalyst adalah nilai `Status` yang membuat `PUCLPost` meneruskan klaim.
//
// Ketiga nilainya terbaca dari rangkaian aksi tombol di `SectionPenerimaanDokumenPUCL`:
// kosong untuk cetak, `0` untuk tolak, `1` untuk kirim.
const statusSendToAnalyst = "1"

// tipeDokumen adalah nilai `tipe` yang dikirim tombol Kirim — berbeda dari `"cetak"` yang
// dikirim tombol Download Dokumen.
const tipeDokumen = "dokumen"

// resourceActionClaimPUCL adalah `pyResourcePath` layanan yang menjalankan rangkaian
// `InsertMitraPA` → `PUCLPost` → `Finish Assignment`.
//
// # Kenapa BUKAN dinamai "PUCLPost"
//
// Karena `PUCLPost` adalah ACTIVITY, dan menamai layanannya sama akan menghasilkan DUA rule
// bernama sama di Pega — `Rule-Obj-Activity PUCLPost` dan `Rule-Service-REST …!PUCLPOST` —
// yang berbeda hanya pada jenis rule-nya. Work Owner menolaknya 2026-10-01, dan itu benar.
//
// Namanya mengikuti kebiasaan keempat layanan yang sudah ada, yang dinamai menurut
// TINDAKANNYA: `KomiteAcceptAdjustment`, `RequestCreateClaimCredit2`,
// `RecivedDataandAttachmentLelangASMSimasbid`.
//
// Ia menyebut TINDAKAN, bukan satu tombol, karena satu layanan ini melayani empat tombol —
// yang membedakan hanya parameter `Status` (kosong · `0` · `1`).
//
// Bila Tim Pega memilih nama lain, yang berubah hanya `PEGA_LAYANAN_KLAIM_PATH` — bukan kode.
const resourceActionClaimPUCL = "ActionClaimPUCL"

// SendToAnalyst menjalankan tindakan "Kirim Ke Analyst" lewat layanan Pega.
func (c *Client) SendToAnalyst(
	ctx context.Context,
	cmd inboxrclpucl.SendToAnalystCommand,
) error {
	if c.baseURL == "" {
		return inboxrclpucl.ErrPegaServiceUnavailable
	}

	body, err := json.Marshal(sendToAnalystRequest{
		CaseNumber:   cmd.CaseNumber,
		Status:       statusSendToAnalyst,
		IDObject:     cmd.IDObject,
		IDCoverage:   cmd.IDCoverage,
		IDAdjustment: cmd.IDAdjustment,
		Tipe:         tipeDokumen,
		Note:         cmd.Note,
		Caller:       cmd.Caller,
	})
	if err != nil {
		return fmt.Errorf("menyusun permintaan kirim ke analyst: %w", err)
	}

	// Jalurnya dari konfigurasi, dengan bawaan resourceActionClaimPUCL.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/"+c.path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("menyusun permintaan HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.user != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	res, err := c.http.Do(req)
	if err != nil {
		// Gagal menghubungi DISATUKAN dengan "belum tersedia", dan itu disengaja: bagi
		// petugas yang menekan tombol, keduanya berarti hal yang sama — tindakannya belum
		// dapat dijalankan dari sini. Yang membedakan keduanya ada di log, bukan di layar.
		return fmt.Errorf("%w: %v", inboxrclpucl.ErrPegaServiceUnavailable, err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 500 {
		return fmt.Errorf("%w: layanan menjawab %d",
			inboxrclpucl.ErrPegaServiceUnavailable, res.StatusCode)
	}
	if res.StatusCode >= 400 {
		// 4xx BUKAN ketidaktersediaan: permintaannya sampai dan DITOLAK. Membungkusnya
		// sebagai "belum tersedia" akan menyuruh orang menunggu Tim Pega, padahal yang salah
		// ada di permintaan kita.
		return fmt.Errorf("layanan Pega menolak permintaan kirim ke analyst: %d",
			res.StatusCode)
	}
	return nil
}
