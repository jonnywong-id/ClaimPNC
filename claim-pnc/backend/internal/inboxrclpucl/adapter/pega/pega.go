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

// claimActionRequest adalah badan permintaan ke layanan Pega.
//
// Nama isiannya mengikuti NAMA PARAMETER `PUCLPost` apa adanya — `Status`, `idObj`, `idCov`,
// `idAdj` — bukan dinamai ulang mengikuti gaya kami. Ia kontrak dengan sistem lain, dan
// menamainya ulang hanya menambah satu terjemahan yang dapat salah di antara dua pihak.
type claimActionRequest struct {
	// Aksi menyebut TINDAKAN yang diminta, dan ia yang menentukan rangkaian activity mana
	// yang dijalankan layanan.
	//
	// Ia dikirim selain `Status` karena keduanya tidak sepadan: `Status` hanya membedakan
	// ketiga tindakan `PUCLPost`, sementara "save" memakai activity yang berbeda sama
	// sekali. Menyandikan semuanya ke dalam `Status` akan menuntut layanan menebak.
	Aksi string `json:"aksi"`

	CaseNumber   string `json:"caseNumber"`
	Status       string `json:"Status"`
	IDObject     string `json:"idObj"`
	IDCoverage   string `json:"idCov"`
	IDAdjustment string `json:"idAdj"`
	Tipe         string `json:"tipe"`

	// StatusCase hanya terisi pada "Download Dokumen" (`statusCase = "1"`). Ketujuh tombol
	// lain tidak mengirimkannya sama sekali.
	StatusCase string `json:"statusCase"`

	// StatusNote adalah catatan riwayat `InsertHistoryClaimPNC`, dan isinya BERBEDA per
	// tombol. Ia dikirim dari sini — bukan ditanam di layanan — supaya keduanya tidak bisa
	// berbeda tanpa ada yang menyadarinya.
	StatusNote string `json:"statusNote"`

	Note   string `json:"note"`
	Caller string `json:"caller"`
}

// actionPlan adalah parameter satu tombol, dibaca dari rangkaian aksinya di section Pega.
type actionPlan struct {
	status     string // parameter `Status` pada PUCLPost
	tipe       string // parameter `tipe` pada InsertMitraPA
	statusCase string // parameter `statusCase` pada PUCLPost

	// `statusNote` TIDAK ada di sini: ia diambil dari `inboxrclpucl.HistoryNoteFor` supaya
	// teksnya satu sumber dengan jalur yang ditangani sendiri.
}

// plans memetakan tindakan menjadi parameter keempat activity Pega.
//
// # Urutan langkahnya BERBEDA per tombol, dan urutan itu bagian dari kontrak
//
// Dibaca apa adanya dari `pyBehaviors` tiap tombol di `SectionPenerimaanDokumenPUCL` dan
// `SectionLampiranSuratPUCL`. Nomornya adalah URUTAN KLIK yang sebenarnya:
//
//	Download Dokumen     1 InsertMitraPA(tipe="cetak")
//	                     2 PUCLPost(Status="", statusCase="1", idObj, idCov, idAdj)
//	                     3 InsertHistoryClaimPNC(statusNote="Wait for Complete PUCL Document ")
//
//	Tolak Klaim          1 InsertMitraPA(tipe="dokumen")
//	                     2 PUCLPost(Status="0", idObj, idCov, idAdj)
//	                     3 refresh currentharness  — BUKAN Finish Assignment
//
//	Kirim Ke Analyst     1 InsertMitraPA(tipe="dokumen")        pyAction=runActivity
//	                     2 PUCLPost(Status="1", idObj, idCov, idAdj)
//	                                                            pyAction=refresh ← SECTION
//	                     3 Finish Assignment → flow action SendtoRCLPUCL
//	                                                            pyAction=finishAssignment
//
//	Kirim ke PIC Teknik  1 PUCLPost(Status="1", idObj, idCov, idAdj)      ← PUCLPost DULU
//	                     2 InsertHistoryClaimPNC(statusNote="send by PUCL to PIC Teknis")
//	                     3 Finish Assignment → flow action SendtoRCLPUCL
//
//	Save                 1 SaveInputRegisterDetail2  — tanpa PUCLPost sama sekali
//
// Tiga hal yang mudah terbaca terbalik, dan ketiganya sudah pernah salah di catatan kami:
//
//   - "Kirim ke PIC Teknik" menjalankan `PUCLPost` LEBIH DULU, baru `InsertHistoryClaimPNC`.
//     Pada "Download Dokumen" urutannya kebalikannya.
//   - "Tolak Klaim" TIDAK menyelesaikan penugasan. Ia hanya menyegarkan harness, sehingga
//     klaimnya tetap di tangan petugas yang sama.
//   - "Download Dokumen" punya TIGA langkah, bukan dua — langkah ketiganya menulis riwayat.
//
// Spasi di ujung "Wait for Complete PUCL Document " ADA di Pega dan dipertahankan (`P-5`).
// Ia terbawa ke kolom riwayat apa adanya, dan membuangnya mengubah data yang tersimpan.
//
// # Ketiga langkah di atas BUKAN seluruh rangkaiannya
//
// Keduanya memanggil activity lain di dalamnya, dan rantai itu baru terbaca utuh 2026-10-02
// setelah Work Owner menyebutkan dua belas activity yang benar-benar berjalan. Ditelusuri
// dari elemen `pyStepsActivityName` tiap berkas:
//
//	InsertMitraPA  → PNCInsertMitraLog_Act   (langkah 2, hanya bila IsDokLengkap sudah "1")
//	PUCLPost       → GetLinkAppClaim         (langkah 3)
//	               → currencyAct             (langkah 8)
//	               → SetTicket               (langkah 17, hanya Status==1)
//	               → SetSignaturePA          (langkah 18, hanya IsPA) → BrowseSignature ×3
//	               → AttachAsPDFC            (langkah 27, prekondisi `1==1` — SELALU)
//	                   → HTMLToPDF · AttachToWork → CallVirusCheck · View
//	               → SetUploadDocument       (langkah 29)
//	               → InsertHistoryClaimPNC   (langkah 35)
//	Finish Assignment → Register_Flow → PNCTeknikRouter
//
// # URUTANNYA wajib, dan terbalik berarti klaim DITUTUP
//
// Shape `RCL/PUCL` (`Assignment6`) di `Register_Flow` punya **tepat satu** konektor keluar,
// tanpa syarat, dan tujuannya **`End1`** — `Data-MO-Event-End`, akhir flow:
//
//	Assignment6  --SendtoRCLPUCL-->  End1
//
// Yang menyelamatkan klaim dari berakhir adalah `SetTicket(SendtoAnalysator)` pada langkah 17,
// yang berjalan LEBIH DULU dan melompatkan flow ke `Assignment5` (Send To Analis). Menyerahkan
// flow action tanpa `PUCLPost` berjalan lebih dulu **menutup klaim**.
//
// Flow action yang sama dipakai dua konektor dengan akibat berlawanan — dari `Assignment5` ia
// MEMASUKKAN klaim ke RCL/PUCL, dari `Assignment6` ia MENGAKHIRI flow. Namanya menyesatkan di
// tempat kedua.
//
// Satu akibat untuk siapa pun yang hendak memakai API assignment bawaan Pega sebagai jalan
// pintas: `Flow Action/SendtoRCLPUCL-FA.xml` **tidak memuat `pyPostProcessingActivity` sama
// sekali**. `PUCLPost` melekat pada TOMBOL, bukan pada flow action — sehingga memanggil flow
// action lewat API melewati `PUCLPost` seluruhnya, dan klaimnya berakhir di `End1`.
//
// # Di mana `PUCLPost` dipasang — terbaca dari `pyBehaviors` tombolnya
//
// Ketiga aksinya berurutan pada event `click`, dan JENIS aksinya berbeda-beda:
//
//	1  pyAction = runActivity        pyActivity = InsertMitraPA
//	2  pyAction = refresh            pyActivity = PUCLPost   pyTarget = thisSection
//	3  pyAction = finishAssignment   (tanpa activity)
//
// Langkah 2 adalah **Refresh Section yang menjalankan activity lebih dulu** — bukan
// `runActivity`. Perbedaannya penting untuk satu hal: ia berarti `PUCLPost` MEMANG punya
// permukaan HTTP, yaitu alamat yang ditembak peramban saat tombolnya ditekan.
//
// Permukaan itu tetap tidak dapat kami pakai, tetapi sebabnya BUKAN "tidak ada jalan masuk":
//
//   - ia menuntut sesi Pega terautentikasi sebagai operator yang berwenang;
//   - ia menuntut clipboard yang SUDAH memuat klaimnya pada `pyWorkPage` — di Pega itu
//     terjadi karena pengguna membuka klaimnya lebih dulu;
//   - dan ia alamat internal peramban, bukan kontrak — ia boleh berubah antarversi Pega
//     tanpa pemberitahuan.
//
// Jadi yang kurang bukan artefak melainkan **satu pintu yang sah**: rule Pega yang
// menjalankan `PUCLPost` dan dapat dipanggil dari luar.
//
// # Dua hal yang dikoreksi dari catatan sebelumnya, keduanya terbaca dari XML
//
//   - Teks riwayat "Kirim Ke Analyst" adalah **"Send by PUCL to Analyst"** — nilai
//     `<statusNote>` pada langkah 35. Yang sebelumnya tercatat, "PUCL send to ANALYST",
//     adalah `pyStepsDescription` langkah itu, bukan parameternya.
//   - `InsertJsonClaimNonMBU_act` **TIDAK berjalan** pada "Kirim Ke Analyst". Prekondisinya
//     `param.Status==0 && .ClaimData.PUCLStatus.RCL_PUCL==1` — jalur Tolak Klaim pada RCL.
//
// `PNCTeknikRouter` tidak ada di export (`R-04`), sehingga aturan penugasannya tidak terbaca.
// `statusNote` TIDAK ditulis di sini melainkan diambil dari `inboxrclpucl.HistoryNoteFor`.
// Teks yang sama dipakai jalur yang ditangani sendiri, dan dua salinan akan dapat berselisih
// tanpa ada yang menyadarinya — selisihnya baru terlihat sebagai dua baris riwayat berbeda
// untuk tombol yang sama.
var plans = map[inboxrclpucl.ClaimActionKind]actionPlan{
	inboxrclpucl.ActionPrintLetter:   {status: "", tipe: "cetak", statusCase: "1"},
	inboxrclpucl.ActionRejectClaim:   {status: "0", tipe: "dokumen"},
	inboxrclpucl.ActionSendToAnalyst: {status: "1", tipe: "dokumen"},
	inboxrclpucl.ActionSendToPICTeknik: {
		// Tanpa `tipe`: jalur ini memakai `InsertHistoryClaimPNC`, bukan `InsertMitraPA`.
		status: "1",
	},
	// ActionSave sengaja TIDAK ada di peta ini — ia tidak memanggil `PUCLPost` maupun
	// `InsertMitraPA`, sehingga keempat parameternya kosong.
}

// resourceActionClaimPUCL adalah `pyResourcePath` layanan yang menjalankan rangkaian
// `InsertMitraPA` → `PUCLPost` → penyerahan flow action `SendtoRCLPUCL`.
//
// Langkah ketiga dinamai flow action-nya, bukan "Finish Assignment", karena yang terakhir itu
// nama TOMBOL — dan satu tombol Finish Assignment dapat menyerahkan flow action mana pun yang
// sedang berlaku. Yang dijalankan tombol ini satu dan tertentu:
//
//	RULE-OBJ-FLOWACTION  ASM-FW-GCNMFW-WORK-PNC  SENDTORCLPUCL   (dipakai Register_Flow)
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

// Perform menjalankan satu tindakan klaim lewat layanan Pega.
func (c *Client) Perform(
	ctx context.Context,
	cmd inboxrclpucl.ClaimActionCommand,
) error {
	if c.baseURL == "" {
		return inboxrclpucl.ErrPegaServiceUnavailable
	}

	// Tindakan yang tidak ada di peta menghasilkan actionPlan kosong, dan itu BENAR untuk
	// ActionSave — bukan keadaan galat.
	plan := plans[cmd.Kind]
	body, err := json.Marshal(claimActionRequest{
		Aksi:         string(cmd.Kind),
		CaseNumber:   cmd.CaseNumber,
		Status:       plan.status,
		IDObject:     cmd.IDObject,
		IDCoverage:   cmd.IDCoverage,
		IDAdjustment: cmd.IDAdjustment,
		Tipe:         plan.tipe,
		StatusCase:   plan.statusCase,

		// Dari domain, bukan dari `plans` — satu sumber untuk kedua jalur.
		StatusNote: inboxrclpucl.HistoryNoteFor(cmd.Kind),
		Note:       cmd.Note,
		Caller:     cmd.Caller,
	})
	if err != nil {
		return fmt.Errorf("menyusun permintaan tindakan %s: %w", cmd.Kind, err)
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
		return fmt.Errorf("layanan Pega menolak tindakan %s: %d", cmd.Kind, res.StatusCode)
	}
	return nil
}
