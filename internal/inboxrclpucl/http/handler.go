package inboxrclpuclhttp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia tidak dipakai menyaring satu pun kueri di modul ini — antreannya bersama. Yang
	// memakainya adalah jejak log, dan itulah satu-satunya kontrol yang tersisa selama
	// pemeriksaan peran belum ada (`TKT-F3-004`).
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox RCL/PUCL.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox RCL/PUCL.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-rcl-pucl/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-rcl-pucl.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		readFilter(query),
		inboxrclpucl.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// Detail menangani GET /api/inbox-rcl-pucl/klaim/{referensi}.
//
// Ia layar kerja RCL/PUCL — yang di Pega terbuka saat nomor klaim diklik, lewat Open
// Assignment. Di sini ia BACA saja; lihat `inboxrclpucl.ClaimDetail`.
//
// Kuncinya diambil dari JALUR, bukan dari parameter query, karena ia mengidentifikasi sumber
// daya — bukan menyaringnya (`10-API-STRATEGY.md` §2). Ia di-decode chi lebih dulu, sehingga
// kunci Pega yang memuat spasi (`ASM-FW-GCNMFW-WORK PNC-700001`) sampai utuh.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	detail, err := h.service.Detail(
		r.Context(),
		active.Alias,
		caller,
		chi.URLParam(r, "referensi"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toClaimDetailResponse(detail, active.Alias))
}

// Documents menangani GET /api/inbox-rcl-pucl/klaim/{referensi}/dokumen.
//
// Ia melayani tombol "Lihat Dokumen" — satu-satunya tombol layar kerja yang MEMBACA, dan
// karena itu satu-satunya yang dapat dibangun tanpa menunggu keputusan `P-1`.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	documents, err := h.service.Documents(
		r.Context(), active.Alias, caller, chi.URLParam(r, "referensi"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDocumentListResponse(documents, active.Alias))
}

// DocumentContent menangani GET /api/inbox-rcl-pucl/klaim/{referensi}/dokumen/{dokumen}.
//
// # Ia menulis BERKAS, bukan JSON
//
// Karena itulah yang diharapkan orang yang menekan "Lihat Dokumen": berkasnya terbuka.
// Membungkusnya base64 di dalam JSON memaksa layar merakit ulang berkasnya sendiri dan
// membesarkan muatan sepertiga tanpa satu pun manfaat.
func (h *Handler) DocumentContent(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	document, err := h.service.DocumentContent(
		r.Context(), active.Alias, caller,
		chi.URLParam(r, "referensi"),
		strings.TrimSpace(chi.URLParam(r, "dokumen")),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	jenis := documentContentType(document)
	w.Header().Set("Content-Type", jenis)

	// Peramban DILARANG menebak jenisnya sendiri. Tanpa ini, berkas yang kami umumkan sebagai
	// `application/octet-stream` masih dapat ditafsirkan peramban sebagai HTML — dan
	// penafsiran itulah yang menjalankan skrip, bukan pengumuman kami.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// `inline` HANYA untuk jenis yang memang aman digambar; selebihnya diunduh.
	//
	// Tombolnya bernama "Lihat Dokumen", dan yang diharapkan pengguna adalah dokumennya
	// terbuka. Tetapi berkas yang diunggah pengguna disajikan dari asal yang sama dengan
	// aplikasi, sehingga satu berkas `.html` atau `.svg` yang digambar sebagai halaman dapat
	// menjalankan skrip atas nama petugas yang sedang masuk.
	if jenisYangBolehTampil[jenis] {
		w.Header().Set("Content-Disposition", inlineHeader(document.Name))
	} else {
		w.Header().Set("Content-Disposition", attachmentHeader(document.Name))
	}

	// Dokumen milik nasabah TIDAK boleh disimpan perantara mana pun.
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document.Content)
}

// jenisIsiBerkas memetakan akhiran nama berkas ke jenis media.
//
// # Kenapa tabel sendiri, bukan `mime.TypeByExtension` saja
//
// Karena di Windows fungsi itu membaca REGISTRY mesin. Jenis isi yang diumumkan peladen lalu
// bergantung pada mesin tempat ia berjalan — dan dokumen yang tampil di satu lingkungan
// terunduh begitu saja di lingkungan lain, tanpa satu pun galat yang menjelaskannya.
//
// Isinya tepat akhiran yang BENAR-BENAR ada pada lampiran klaim. `jfif` ikut karena ia muncul
// 116 kali dan tidak dikenali tabel bawaan mana pun.
var jenisIsiBerkas = map[string]string{
	"pdf":  "application/pdf",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"jfif": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"bmp":  "image/bmp",
	"tif":  "image/tiff",
	"tiff": "image/tiff",
	"txt":  "text/plain; charset=utf-8",
	"log":  "text/plain; charset=utf-8",
	"csv":  "text/csv; charset=utf-8",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ppt":  "application/vnd.ms-powerpoint",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"zip":  "application/zip",
}

// jenisYangBolehTampil adalah jenis isi yang boleh digambar DI DALAM halaman.
//
// Selebihnya diunduh, bukan ditampilkan. Alasannya keamanan, bukan kerapian: berkas yang
// diunggah pengguna disajikan dari asal yang sama dengan aplikasi, dan satu berkas `.html`
// atau `.svg` yang digambar sebagai halaman dapat menjalankan skrip atas nama petugas yang
// sedang masuk.
var jenisYangBolehTampil = map[string]bool{
	"application/pdf":           true,
	"image/jpeg":                true,
	"image/png":                 true,
	"image/gif":                 true,
	"image/bmp":                 true,
	"image/tiff":                true,
	"text/plain; charset=utf-8": true,
}

// documentContentType memilih jenis isi yang diumumkan.
//
// # Yang dicatat basis data BUKAN jenis media
//
// `DATA_ATTACHFILE.ATTACHMIMETYPE` berisi **akhiran telanjang** — `pdf`, `jpeg`, `PNG` —
// dan tidak satu pun dari 21 nilai berbedanya memuat tanda `/`. Mengumumkannya apa adanya
// menghasilkan `Content-Type: pdf`, yang bukan jenis media apa pun; peramban lalu tidak
// menampilkan dokumennya.
//
// Versi sebelumnya mengembalikan nilai itu apa adanya. Diperbaiki 2026-10-02.
//
// Urutannya: jenis yang tercatat bila ia memang jenis media, lalu akhiran yang tercatat,
// lalu akhiran nama berkasnya, lalu `application/octet-stream` — yang berarti "unduh saja",
// bukan "tampilkan".
func documentContentType(document inboxrclpucl.DocumentContent) string {
	tercatat := strings.TrimSpace(document.MimeType)

	// Sudah berupa jenis media, misalnya bila kelak ada jalur lain yang menulisnya penuh.
	if strings.Contains(tercatat, "/") {
		return tercatat
	}

	if jenis := jenisDariAkhiran(tercatat); jenis != "" {
		return jenis
	}
	if jenis := jenisDariAkhiran(strings.TrimPrefix(filepath.Ext(document.Name), ".")); jenis != "" {
		return jenis
	}
	return "application/octet-stream"
}

// jenisDariAkhiran mencari jenis media sebuah akhiran, tanpa titik di depan.
func jenisDariAkhiran(akhiran string) string {
	bersih := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(akhiran), "."))
	if bersih == "" {
		return ""
	}
	if jenis, ada := jenisIsiBerkas[bersih]; ada {
		return jenis
	}
	// Tabel bawaan sebagai cadangan — hasilnya dapat berbeda antar mesin, jadi ia yang
	// terakhir dicoba, bukan yang pertama.
	return mime.TypeByExtension("." + bersih)
}

// inlineHeader menyusun Content-Disposition.
//
// Nama berkas dikutip dan tanda kutip di dalamnya dibuang. Nama berkas berasal dari data —
// bukan dari kode — dan satu tanda kutip di dalamnya akan memotong header di tempat yang
// salah.
func inlineHeader(name string) string {
	return `inline; filename="` + namaBerkasAman(name) + `"`
}

// attachmentHeader menyusun Content-Disposition untuk berkas yang DIUNDUH, bukan digambar.
func attachmentHeader(name string) string {
	return `attachment; filename="` + namaBerkasAman(name) + `"`
}

// namaBerkasAman membersihkan nama berkas agar tidak merusak header.
//
// Tanda kutip dan pergantian baris dibuang: keduanya dapat menyudahi nilai header lebih awal
// dan menyisipkan header lain — dan nama berkas di sini berasal dari ketikan petugas.
func namaBerkasAman(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = "dokumen"
	}
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(clean)
}

// PerformAction menangani POST /api/inbox-rcl-pucl/klaim/{referensi}/tindakan/{aksi}.
//
// Satu-satunya rute modul ini yang MENGUBAH klaim. Ia tidak menulis satu baris pun sendiri:
// yang menulis adalah Pega, lewat layanannya — lihat `inboxrclpucl.ClaimActions`.
//
// # Kenapa SATU rute untuk beberapa tindakan
//
// Karena di Pega pun keempat tombolnya memanggil activity yang sama; yang membedakan hanya
// parameternya. Satu rute per tombol akan menyalin rangkaian yang sama berkali-kali dan
// menyembunyikan bahwa keempatnya satu jalur.
func (h *Handler) PerformAction(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	// Tindakan yang tidak dikenali dijawab 404, bukan diteruskan ke layanan Pega. Meneruskan
	// teks apa pun ke sana berarti layar dapat menyuruh Pega mengerjakan hal yang tidak
	// pernah ada tombolnya.
	kind, known := inboxrclpucl.ClaimActionOf(chi.URLParam(r, "aksi"))
	if !known {
		h.writeJSON(w, r, http.StatusNotFound, ErrorResponse{
			Code:    "tindakan_tidak_dikenal",
			Message: "Tindakan yang diminta tidak dikenali.",
		})
		return
	}

	// Badan permintaan dibaca untuk TIGA tindakan: "save" dan kedua tombol Kirim.
	//
	// Sampai 2026-10-02 hanya "save" yang membacanya, dan akibatnya catatan yang diketik
	// petugas DIBUANG begitu ia menekan Kirim. Di Pega tidak demikian: Finish Assignment
	// mem-posting form yang sama, sehingga `PUCLPost` langkah 10 menuliskan `KomentarPUCL`
	// bersama penandaan klaimnya. Lihat Service.PerformAction.
	//
	// "Tolak Klaim" dan "Download Dokumen" tetap mengirim badan kosong — keduanya tidak
	// menyentuh kedua isian itu.
	var input inboxrclpucl.ReceiptInput
	if carriesReceipt(kind) {
		var body saveReceiptRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    "badan_tidak_terbaca",
				Message: "Isian yang dikirim tidak terbaca.",
			})
			return
		}
		input = inboxrclpucl.ReceiptInput{
			Note:       body.Note,
			CompleteAt: body.CompleteAt,
		}
	}

	doc, err := h.service.PerformAction(
		r.Context(), active.Alias, caller, chi.URLParam(r, "referensi"), kind, input,
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	jawaban := map[string]any{"pesan": actionDoneMessage(kind)}

	// TIGA tindakan menerbitkan surat — "Download Dokumen" dan kedua tombol Kirim — karena
	// `PUCLPost` langkah 27 memanggil `AttachAsPDFC` berprekondisi `1==1`.
	//
	// Barisnya dikembalikan supaya layar dapat mengambilnya lewat alamat isi dokumen yang
	// sudah ada, bukan lewat alamat baru yang mengembalikan berkas dari jalur tindakan.
	// Yang MEMBUKA berkasnya hanya "Download Dokumen"; kedua tombol Kirim melampirkannya
	// saja, persis seperti Pega — lihat ClaimAction di layar.
	//
	// `nil` berarti suratnya TIDAK terbit, dan itu keadaan yang sah: tindakannya tetap
	// berhasil, klaimnya tetap berpindah. Kalimatnya menyesuaikan.
	switch {
	case doc != nil:
		jawaban["dokumen"] = documentDTO(*doc)
	case kind == inboxrclpucl.ActionPrintLetter:
		jawaban["pesan"] = "Klaim ditandai suratnya sudah dicetak dan berpindah ke tab " +
			"\"Kelengkapan Dokumen\". Berkas suratnya TIDAK berhasil diterbitkan kali ini."
	case kind == inboxrclpucl.ActionSendToAnalyst, kind == inboxrclpucl.ActionSendToPICTeknik:
		// Kalimat cadangan ini pun menyebut PERPINDAHANNYA, bukan hanya penandaan.
		// Bentuk sebelumnya berbunyi "ditandai selesai di PUCL", dan itu menyatakan lebih
		// sedikit daripada yang terjadi: klaimnya sudah menjadi tugas PIC Teknik.
		jawaban["pesan"] = "Isian disimpan dan klaim DIPINDAHKAN ke tahap Send To Analis. " +
			"Hanya suratnya yang TIDAK berhasil dilampirkan kali ini — cetak ulang dari " +
			"tab \"Cetak Surat\"."
	}

	h.writeJSON(w, r, http.StatusOK, jawaban)
}

// kalimatKirim dibaca petugas setelah kedua tombol Kirim berhasil.
//
// # Kenapa SATU kalimat untuk dua tombol
//
// Karena akibatnya memang sama persis: keduanya menempuh `PUCLPost` dengan `Status = 1`,
// dan keduanya meninggalkan baris antrean Pega yang sama. Menuliskannya dua kali membuat
// keduanya dapat berselisih tanpa ada yang menyadarinya.
//
// # Kalimatnya berubah karena PERILAKUNYA berubah
//
// Dua bentuk sebelumnya menyatakan klaimnya tidak bergerak — *"perpindahannya dikerjakan
// Pega"*, lalu *"MASIH di antrean RCL/PUCL dan BELUM bergerak"*. Keduanya benar pada
// saatnya, dan **keduanya tidak lagi benar**: sejak tombolnya memindahkan tugas sendiri,
// klaim berpindah ke tahap Send To Analis di aplikasi ini.
//
// # Yang tetap disebut, dan kenapa
//
// Baris antrean RCL/PUCL milik Pega **tidak** dihapus — menulis tabel penugasan Pega sudah
// terbukti merusak. Jadi klaim yang sama masih tergambar di antrean Pega, dan petugas yang
// membuka Pega akan melihatnya. Tidak menyebutkannya akan membuat ia mengira perpindahannya
// gagal, lalu menekan tombolnya lagi.
const kalimatKirim = "Isian disimpan, surat dilampirkan ke klaim, dan klaim DIPINDAHKAN ke " +
	"tahap Send To Analis — kini menjadi tugas PIC Teknik klaim ini, meniru ticket " +
	"SendtoAnalysator. Di Pega baris antrean RCL/PUCL lamanya masih tergambar; itu tidak " +
	"menghalangi apa pun dan tidak perlu dikerjakan ulang."

// carriesReceipt menyatakan tindakan ini membawa kedua isian Penerimaan Dokumen.
//
// Dipisah menjadi fungsi, bukan ditulis sebagai rangkaian `||` di tempat pemakaiannya, karena
// daftarnya adalah ATURAN — dan aturan yang sama dipakai layar saat memutuskan tombol mana
// yang mengirim badan permintaan. Menyebarkannya membuat keduanya dapat berselisih.
func carriesReceipt(kind inboxrclpucl.ClaimActionKind) bool {
	switch kind {
	case inboxrclpucl.ActionSave,
		inboxrclpucl.ActionSendToAnalyst,
		inboxrclpucl.ActionSendToPICTeknik:
		return true
	default:
		return false
	}
}

// actionDoneMessage adalah kalimat yang dibaca petugas setelah tindakannya berhasil.
//
// Ia menyebut AKIBATNYA, bukan "berhasil": petugas perlu tahu klaimnya berpindah ke mana,
// karena itu yang menentukan apakah ia masih harus mengerjakan sesuatu.
func actionDoneMessage(kind inboxrclpucl.ClaimActionKind) string {
	switch kind {
	case inboxrclpucl.ActionPrintLetter:
		// Kalimatnya menyebut KEDUA akibatnya, dan sejak 2026-10-02 yang kedua memang
		// terjadi: templat `SuratPUCL` diterima Work Owner, sehingga suratnya terbit dan
		// melampir ke klaim. Bentuk sebelumnya berbunyi "belum dapat diterbitkan", dan
		// itu tidak lagi benar.
		return "Surat diterbitkan dan dilampirkan ke klaim — lihat \"Lihat Dokumen\". " +
			"Klaim berpindah ke tab \"Kelengkapan Dokumen\"."
	case inboxrclpucl.ActionRejectClaim:
		return "Klaim ditolak."
	case inboxrclpucl.ActionSendToAnalyst:
		// Kalimatnya TIDAK menjanjikan perpindahan di Pega, dan itu bukan kehati-hatian
		// berlebihan: bentuk sebelumnya berbunyi "Klaim diteruskan ke Analyst", dan Work
		// Owner menemukan bahwa klaimnya **masih berada di antrean RCL/PUCL milik Pega** —
		// hanya tidak tergambar lagi di layar ini.
		//
		// Yang memindahkannya di Pega adalah ticket `SendtoAnalysator`, yang dilepas
		// `PUCLPost` langkah 17 dan menempel pada shape `SendToAnalis`. Ticket itu hanya
		// dapat dilepas Pega sendiri; percobaan memindahkan penugasannya dari sini membuat
		// satu klaim tidak dapat dibuka lagi (§163).
		return kalimatKirim
	case inboxrclpucl.ActionSendToPICTeknik:
		return kalimatKirim
	case inboxrclpucl.ActionSave:
		return "Isian disimpan. Klaimnya tetap di antrean ini."
	default:
		return "Tindakan dijalankan."
	}
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya dua tindakan yang menulis — mencetak surat
// PUCL/RCL, yang mengisi `TANGGALCETAKDOKUMENPUCL_1` sehingga klaimnya BERPINDAH dari tab
// "Cetak Surat" ke tab "Kelengkapan Dokumen", dan mengirim Reminder PUCL. Tindakan yang
// dijawab "halaman tidak ditemukan" terbaca sebagai kerusakan, sementara yang dibutuhkan
// pengguna adalah tahu ke mana ia harus pergi.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang menyebut
// portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar mengira ia
// sudah berada di portal yang benar.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda seberapa
	// sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi dasar
	// memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	if h.logger != nil {
		h.logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", strings.TrimSpace(r.URL.Query().Get("tindakan"))),
		)
	}

	h.writeError(w, r, inboxrclpucl.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List, Export, dan Report menuntut keduanya dengan urutan yang sama,
// dan urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal
// dijawab sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxrclpucl.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxrclpucl.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxrclpucl.ErrCallerUnknown)
		return portal.Portal{}, inboxrclpucl.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring grid dari parameter query.
//
// Hanya satu: tab mana yang diminta. Kedua isian tanggal TIDAK dibaca di sini, dan itu
// bukan kelalaian — keduanya tidak menyaring grid sama sekali. Yang memakainya adalah
// laporan harian, lewat readReport di bawah.
//
// Ia tetap dikumpulkan sebagai fungsi tersendiri supaya daftar dan ekspor membaca parameter
// yang SAMA PERSIS. Ekspor yang membaca tab dengan cara berbeda akan menghasilkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func readFilter(query url.Values) inboxrclpucl.QueryInput {
	return inboxrclpucl.QueryInput{Tab: query.Get("tab")}
}

// readReport membaca permintaan laporan harian dari parameter query.
//
// Nama parameternya mengikuti nama isian di layar lama — "dari" dan "sampai" adalah padanan
// Indonesia dari "FROM RCL/PUCL" dan "TO RCL/PUCL" (`D-13`). Keduanya kontrak API, sehingga
// tetap berbahasa Indonesia (`D-80`).
func readReport(query url.Values) inboxrclpucl.ReportInput {
	return inboxrclpucl.ReportInput{
		Tab:  query.Get("tab"),
		From: query.Get("dari"),
		To:   query.Get("sampai"),
	}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxrclpucl.Caller, bool) {
	if h.caller == nil {
		return inboxrclpucl.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxrclpucl.Caller{}, false
	}
	return inboxrclpucl.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// DocumentCategories melayani pilihan kolom "Category" pada dialog unggah.
//
// GET, dan tanpa nomor klaim di jalurnya: isinya master jenis dokumen, sama bagi setiap klaim
// dan setiap petugas. Menyarangkannya di bawah sebuah klaim akan menyiratkan ia berbeda-beda
// per klaim, dan layar lalu menariknya ulang setiap kali klaim dibuka.
//
// Tetap di balik pemeriksaan portal, karena masternya hidup di basis data entitas — satu
// badan hukum boleh memakai daftar jenis dokumen yang berbeda dari badan hukum lain.
func (h *Handler) DocumentCategories(w http.ResponseWriter, r *http.Request) {
	active, _, ready := h.prepare(w, r)
	if !ready {
		return
	}

	categories, err := h.service.DocumentCategories(r.Context(), active.Alias)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	items := make([]documentCategoryDTO, 0, len(categories))
	for _, category := range categories {
		items = append(items, documentCategoryDTO{
			Nilai: category.Value,
			Nama:  category.Label,
		})
	}

	h.writeJSON(w, r, http.StatusOK, map[string]any{"kategori": items})
}

// UploadDocument menerima satu berkas dan melampirkannya ke klaim.
//
// # Kenapa multipart, bukan JSON ber-base64
//
// Karena base64 membesarkan muatan sepertiga TANPA manfaat, dan seluruhnya tetap melewati
// memori. Alasan yang sama membuat `document_content` tidak membungkus hasilnya
// `pooldata.base64encode` seperti kueri lama.
//
// # Batas besar berkas ditegakkan DUA KALI
//
// `MaxBytesReader` memotongnya di lapisan HTTP, dan UploadedDocument.Validate memeriksanya
// lagi di domain. Yang pertama melindungi memori peladen dari badan permintaan yang mengaku
// kecil lalu mengirim besar; yang kedua menjaga aturannya tetap berlaku bagi pemanggil lain.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, inboxrclpucl.MaxDocumentSize+1<<20)

	berkas, header, err := r.FormFile("berkas")
	if err != nil {
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    "berkas_tidak_terbaca",
			Message: "Berkas unggahan tidak terbaca. Pilih satu berkas lalu coba lagi.",
		})
		return
	}
	defer berkas.Close()

	isi, err := io.ReadAll(berkas)
	if err != nil {
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    "berkas_tidak_terbaca",
			Message: "Berkas unggahan tidak dapat dibaca sampai selesai.",
		})
		return
	}

	// Nama berkasnya DAPAT DIUBAH petugas — kolom "Name" pada dialog `SetUploadDocPUCL`
	// adalah `pxTextInput`, terpisah dari kolom "File" yang hanya menampilkan nama aslinya.
	// Yang tersimpan sebagai `ATTACHNAME` adalah kolom Name itu.
	//
	// Kosong berarti petugas tidak mengubahnya; nama berkas aslinya yang dipakai.
	nama := strings.TrimSpace(r.FormValue("nama"))
	if nama == "" {
		nama = header.Filename
	}

	doc, err := h.service.UploadDocument(
		r.Context(), active.Alias, caller, chi.URLParam(r, "referensi"),
		inboxrclpucl.UploadedDocument{
			Name: nama,
			// Kategori lampiran, disimpan apa adanya sebagai nama.
			Category: strings.TrimSpace(r.FormValue("kategori")),
			// Jenis isi diambil dari NAMA BERKASNYA, bukan dari header yang dikirim
			// peramban: `DATA_ATTACHFILE.ATTACHMIMETYPE` pada baris Pega berisi ekstensi
			// saja (`pdf`), dan menyimpan bentuk yang berbeda pada kolom yang sama akan
			// membuat baris kami tidak sebangun dengan baris Pega (`P-5`).
			// Jenis isi tetap dari nama berkas ASLI — petugas boleh menamai ulang
			// lampirannya tanpa ekstensi, dan menebaknya dari nama ketikan akan
			// menyimpan jenis yang kosong pada berkas yang jelas PDF.
			MimeType: extensionOf(header.Filename),
			Note:     strings.TrimSpace(r.FormValue("keterangan")),
			Content:  isi,
		})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusCreated, map[string]any{
		"pesan":   "Dokumen diunggah.",
		"dokumen": documentDTO(doc),
	})
}

// extensionOf mengambil ekstensi berkas tanpa titik, huruf kecil.
//
// Kosong bila namanya tidak berekstensi — lebih jujur daripada menebak "bin", yang akan
// tersimpan sebagai jenis isi yang tidak pernah benar.
func extensionOf(name string) string {
	ext := strings.TrimPrefix(filepath.Ext(strings.TrimSpace(name)), ".")
	return strings.ToLower(ext)
}
