package inboxcloseclaimhttp

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcloseclaim/usecase"
	"claim-pnc/internal/platform/logging"

	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor sebagai tipe konkret,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, q usecase.ListQuery) (usecase.ListResult, error)
	Request(ctx context.Context, cmd usecase.RequestCommand) (inboxcloseclaim.ClaimRequest, error)

	// CanRequest dibaca untuk MEMBERI TAHU layar, bukan untuk menjaga.
	//
	// Penjagaannya tetap di Request — dan itu pembagian yang mengikat: penjagaan yang hanya
	// ada di transport dapat dilewati siapa pun yang memanggil usecase dari tempat lain,
	// persis cacat `pyPrivilegeName` sistem lama yang hanya menyembunyikan menu.
	CanRequest() bool
}

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Dua field, bukan satu seperti modul yang hanya membaca: jejak permintaan menyimpan NAMA
// pemohon bersama login-nya, supaya jejak itu tetap terbaca utuh tanpa join ke tabel
// pengguna — jejak yang namanya diambil lewat join akan berubah ketika orangnya berganti
// nama.
type Caller struct {
	Login string
	Name  string
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Inbox Close Claim.
type Handler struct {
	service     Service
	getCaller   GetCaller
	logger      *slog.Logger
	writeJSON   JSONWriter
	writeErrorF ErrorWriter
	location    *time.Location
	now         func() time.Time
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   Service
	GetCaller GetCaller
	Logger    *slog.Logger

	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	//
	// Ia parameter, bukan konstanta, supaya uji dapat menetapkannya dan tidak bergantung
	// pada basis data zona waktu mesin yang menjalankan.
	Location *time.Location

	// Now dapat diisi uji supaya kolom Lama Waktu Klaim dapat diperiksa secara
	// deterministik.
	Now func() time.Time
}

// NewHandler membentuk handler modul Inbox Close Claim.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}

	return &Handler{
		service:     o.Service,
		getCaller:   o.GetCaller,
		logger:      o.Logger,
		writeJSON:   o.WriteJSON,
		writeErrorF: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		location:    location,
		now:         now,
	}
}

// jakarta mengembalikan zona WIB.
//
// Bila basis data zona waktu tidak tersedia di mesin — yang terjadi pada sebagian citra
// kontainer minimal — dipakai offset tetap +07:00. Indonesia bagian barat tidak mengenal
// daylight saving, sehingga offset tetap SETARA dan bukan penyederhanaan yang merugikan.
func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// selisihTerencana adalah perbedaan yang DISENGAJA terhadap layar Pega.
//
// Ia dikirim ke layar dan ditampilkan, bukan disembunyikan sebagai detail teknis. `D-54`
// menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan Work Owner tertulis;
// ketiganya di bawah sudah diputuskan 2026-09-23, dan menyatakannya di layar itulah yang
// membuat keputusan itu terlihat oleh orang yang memakai layarnya.
func selisihTerencana() []string {
	return []string{
		"Kolom \"Lama Waktu Klaim\" berisi umur klaim dalam hari. Di Pega kolom itu " +
			"menampilkan tanggal pendaftaran untuk kedua kalinya.",
		"Jumlah total mengikuti penyaring Status Bayar. Kueri hitung Pega tidak " +
			"menyertakannya, sehingga totalnya di sana tidak cocok dengan barisnya.",
		"ReOpen dan Copy Klaim mencatat PERMINTAAN, bukan mengubah klaim. Klaim berubah " +
			"setelah permintaannya dijalankan.",
	}
}

// List menangani GET /api/inbox-close-claim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.writeJSON, w, r, err.Error())
		return
	}

	result, err := h.service.List(r.Context(), q)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.logPendingLookupFailure(r, result)

	now := h.now()
	claims := make([]claimDTO, 0, len(result.Page.Claims))
	for _, claim := range result.Page.Claims {
		claims = append(claims, toClaimDTO(claim, result.Pending[claim.ClaimID], now, h.location))
	}

	boleh := h.service.CanRequest()

	body := listResponse{
		Klaim:             claims,
		Total:             result.Page.Total,
		PermintaanTerbaca: result.PendingLookupError == nil,
		BolehMengajukan:   boleh,
		SelisihTerencana:  selisihTerencana(),

		// Selama belum ada pelaksananya, permintaan yang tercatat tidak akan mengubah apa
		// pun. Itu dinyatakan HANYA bila pengajuannya memang terbuka — menyatakannya saat
		// tombolnya sendiri mati hanya menambah kalimat yang tidak berlaku.
		PelaksanaBelumAda: boleh,
	}
	if !boleh {
		body.AlasanTidakBoleh = inboxcloseclaim.AlasanTidakBolehMengajukan
	}

	h.writeJSON(w, r, http.StatusOK, body)
}

// Metadata menangani GET /api/inbox-close-claim/penyaring.
//
// Isinya bentuk layar, bukan data: kelima pilihan lini bisnis beserta kedua pilihan status
// adalah hasil pembacaan activity Pega, dan tempat pembacaan itu tercatat adalah backend.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	lines := make([]pilihanDTO, 0, len(inboxcloseclaim.BusinessLines()))
	for _, line := range inboxcloseclaim.BusinessLines() {
		lines = append(lines, pilihanDTO{Nilai: string(line), Label: line.Label()})
	}

	h.writeJSON(w, r, http.StatusOK, metadataResponse{
		LiniBisnis: lines,
		StatusTransfer: []pilihanDTO{
			{Nilai: string(inboxcloseclaim.TransferAny), Label: "Semua Status Transfer"},
			{Nilai: string(inboxcloseclaim.TransferDone), Label: "Sudah Transfer"},
			{Nilai: string(inboxcloseclaim.TransferNone), Label: "Belum Transfer"},
		},
		StatusBayar: []pilihanDTO{
			{Nilai: string(inboxcloseclaim.PaymentAny), Label: "Semua Status Bayar"},
			{Nilai: string(inboxcloseclaim.PaymentPaid), Label: "Lunas"},
			{Nilai: string(inboxcloseclaim.PaymentUnpaid), Label: "Belum Lunas"},
		},
	})
}

// maxBodyBytes membatasi ukuran badan permintaan.
//
// Isian terpanjangnya adalah alasan, 1.500 karakter. Batas di bawah jauh lebih longgar dari
// itu dan tetap jauh lebih ketat daripada tanpa batas — badan permintaan tanpa batas adalah
// cara termurah membuat satu proses menghabiskan memori.
const maxBodyBytes = 64 * 1024

// Request menangani POST /api/inbox-close-claim/permintaan.
//
// Melayani KEDUA aksi — ReOpen dan Copy Klaim — yang dibedakan field `jenis`. Keduanya
// menempuh pemeriksaan, penyimpanan, dan penolakan yang sama persis; memisahkannya menjadi
// dua handler akan menggandakan aturan yang sama dan membuka celah keduanya menyimpang.
func (h *Handler) Request(w http.ResponseWriter, r *http.Request) {
	caller, found := h.getCaller(r.Context())
	if !found || strings.TrimSpace(caller.Login) == "" {
		writeBadRequest(h.writeJSON, w, r, "identitas pemanggil tidak dikenali")
		return
	}

	activePortal, portalFound := portalhttp.ActivePortalFrom(r.Context())
	if !portalFound {
		writeBadRequest(h.writeJSON, w, r, "portal aktif tidak dikenali")
		return
	}

	var body requestBody
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeBadRequest(h.writeJSON, w, r, "badan permintaan tidak dapat dibaca")
		return
	}

	kind, known := inboxcloseclaim.ParseRequestKind(body.Jenis)
	if !known {
		writeBadRequest(h.writeJSON, w, r,
			"jenis permintaan harus \"reopen\" atau \"salin\"")
		return
	}

	request, err := h.service.Request(r.Context(), usecase.RequestCommand{
		PortalAlias: activePortal.Alias,
		Kind:        kind,
		ClaimID:     strings.TrimSpace(body.KlaimID),
		Reason:      strings.TrimSpace(body.Alasan),
		ActorLogin:  caller.Login,
		ActorName:   caller.Name,
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	// 201: sebuah PERMINTAAN dibuat. Bukan 200 — yang akan menyiratkan bahwa tindakannya
	// sendiri sudah dijalankan, dan itu justru yang belum terjadi.
	h.writeJSON(w, r, http.StatusCreated, requestResponse{
		Permintaan: toRequestDTO(request, h.location),
		Pesan:      pesanPermintaan(kind),
	})
}

// pesanPermintaan menjelaskan apa yang sudah dan belum terjadi.
//
// Kalimatnya sengaja menyebut keduanya. Tombol yang berhasil ditekan tanpa perubahan apa
// pun di layar adalah keadaan yang paling mudah disalahpahami sebagai kegagalan — atau,
// lebih buruk, sebagai keberhasilan yang sudah tuntas.
func pesanPermintaan(kind inboxcloseclaim.RequestKind) string {
	switch kind {
	case inboxcloseclaim.RequestReopen:
		return "Permintaan buka kembali tercatat. Klaim belum berubah — ia terbuka kembali " +
			"setelah permintaan ini dijalankan."
	case inboxcloseclaim.RequestCopy:
		return "Permintaan salin klaim tercatat. Klaim baru terbit setelah permintaan ini " +
			"dijalankan, dengan nomornya sendiri."
	default:
		return "Permintaan tercatat."
	}
}

// exportBatchSize adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Export ditulis SAMBIL dibaca, sekumpulan baris pada satu waktu, sehingga memori tetap
// datar berapa pun jumlah barisnya (`15-NFR` §3.2 butir 6).
const exportBatchSize = 500

// exportMaxRows membatasi banyaknya baris yang diekspor.
//
// Sistem lama membatasi 500 baris lewat `pyMaxRecords` pada 54 dari 56 laporannya, sehingga
// kebutuhan export bervolume besar BELUM PERNAH benar-benar dilayani dan tidak ada data
// historis yang sahih untuk menentukan angkanya (`15-NFR` §3.2). Angka di bawah karena itu
// batas pengaman yang dipilih sadar, bukan peniruan. Pertanyaan terbukanya `ADR-0011`.
const exportMaxRows = 10000

// Export menangani GET /api/inbox-close-claim/unduh.
//
// # Kenapa CSV, bukan XLSX
//
// Tombol di layar lama berbunyi "Export to Excel", tetapi yang dipanggilnya adalah
// `Activity/ExportDataCloseClaim-Act.xml`, yang isinya `Page-Copy` diikuti
// `Call pxConvertResultsToCSV` — keluarannya CSV. Memakai `encoding/csv` dari pustaka
// standar karena itu SETARA dengan sistem lama, bukan penyederhanaan.
//
// # Yang diekspor: seluruh hasil, bukan halaman yang sedang dilihat
//
// Ini berbeda dari Pega, yang menyalin HALAMAN hasil yang sudah dimuat. Perbedaannya
// mengikuti keputusan yang sama seperti pada modul Inbox Outstanding, dan arahnya
// menambah — bukan menyembunyikan baris.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.writeJSON, w, r, err.Error())
		return
	}

	q.Filter.Offset = 0
	q.Filter.Limit = exportBatchSize

	first, err := h.service.List(r.Context(), q)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.logPendingLookupFailure(r, first)

	// Header ditulis SEBELUM baris pertama dikirim. Setelah badan respons mulai mengalir,
	// status HTTP tidak dapat diubah lagi — sehingga galat yang terjadi di tengah tidak
	// dapat dijawab dengan 500. Itu diterima, dan alasannya ada di bawah.
	filename := "inbox-close-claim-" + h.now().In(h.location).Format("20060102-150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader()); err != nil {
		h.logExportInterrupted(r, err)
		return
	}

	now := h.now()
	written := 0
	page := first

	for {
		for _, claim := range page.Page.Claims {
			if written >= exportMaxRows {
				return
			}
			row := exportRow(toClaimDTO(claim, page.Pending[claim.ClaimID], now, h.location))
			if err := writer.Write(row); err != nil {
				// Sambungan putus di tengah unduhan adalah kejadian biasa — pengguna
				// menutup tab. Ia dicatat sebagai peringatan, bukan galat, dan tidak dapat
				// diberitahukan ke klien karena badan respons sudah mengalir.
				h.logExportInterrupted(r, err)
				return
			}
			written++
		}

		// Baris yang diterima lebih sedikit dari yang diminta berarti sudah habis.
		if len(page.Page.Claims) < q.Filter.Limit ||
			written >= page.Page.Total ||
			written >= exportMaxRows {
			return
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportInterrupted(r, err)
			return
		}

		q.Filter.Offset = written
		page, err = h.service.List(r.Context(), q)
		if err != nil {
			h.logExportInterrupted(r, err)
			return
		}
	}
}

// exportHeader adalah judul kolom CSV.
//
// Judulnya SAMA PERSIS dengan kolom layar, dan urutannya pun sama — berkas yang diunduh
// harus terbaca sebagai salinan apa yang dilihat pengguna. Teksnya berbahasa Indonesia
// karena begitulah layar lama menulisnya (`D-13`).
//
// Kolom "Pilih" tidak ikut: ia kotak centang, bukan data.
func exportHeader() []string {
	return []string{
		"No Klaim", "No Polis", "Nama Tertanggung", "Nama Bisnis", "Sumber Bisnis",
		"Nama Cabang", "Tanggal Pendaftaran", "Lama Waktu Klaim", "PIC Teknik", "Admin PNC",
		"Status", "Status Klaim", "Sudah Transfer", "Permintaan Tertunda",
	}
}

func exportRow(c claimDTO) []string {
	pending := ""
	for i, request := range c.PermintaanTertunda {
		if i > 0 {
			pending += "; "
		}
		pending += request.Jenis
	}

	return []string{
		c.NomorKlaim, c.NomorPolis, c.Tertanggung, c.NamaBisnis, c.SumberBisnis,
		c.NamaCabang, c.TanggalPendaftaran, strconv.Itoa(c.LamaHari), c.PICTeknik, c.AdminPNC,
		c.StatusTampil, c.StatusKlaimLabel, boolText(c.SudahTransfer), pending,
	}
}

// boolText menuliskan penanda biner sebagai kata, bukan "true"/"false".
//
// Berkas ini dibuka pengguna bisnis di pengolah angka; "true" bukan kata yang dipakai di
// layar mana pun pada aplikasi ini.
func boolText(value bool) string {
	if value {
		return "Ya"
	}
	return "Tidak"
}

// queryFrom membaca penyaring dan paginasi dari query string.
//
// Identitas pemanggil diambil dari KONTEKS, tidak pernah dari badan permintaan maupun query
// string.
func (h *Handler) queryFrom(r *http.Request) (usecase.ListQuery, error) {
	caller, found := h.getCaller(r.Context())
	if !found || strings.TrimSpace(caller.Login) == "" {
		return usecase.ListQuery{}, fmt.Errorf("identitas pemanggil tidak dikenali")
	}

	// Portal aktif sudah diperiksa middleware; ketiadaannya di sini berarti rute dipasang di
	// luar middleware itu — cacat perakitan, bukan kesalahan pengguna. Ia ditolak, tidak
	// pernah dilayani portal utama sebagai cadangan (`R-20`).
	activePortal, portalFound := portalhttp.ActivePortalFrom(r.Context())
	if !portalFound {
		return usecase.ListQuery{}, fmt.Errorf("portal aktif tidak dikenali")
	}

	q := r.URL.Query()

	business, known := inboxcloseclaim.ParseBusinessLine(q.Get("lini"))
	if !known {
		return usecase.ListQuery{}, fmt.Errorf("pilihan lini bisnis tidak dikenal")
	}
	transfer, known := inboxcloseclaim.ParseTransferStatus(q.Get("status_transfer"))
	if !known {
		return usecase.ListQuery{}, fmt.Errorf("pilihan status transfer tidak dikenal")
	}
	payment, known := inboxcloseclaim.ParsePaymentStatus(q.Get("status_bayar"))
	if !known {
		return usecase.ListQuery{}, fmt.Errorf("pilihan status bayar tidak dikenal")
	}

	limit, err := positiveInt(q.Get("batas"), inboxcloseclaim.DefaultLimit)
	if err != nil {
		return usecase.ListQuery{}, fmt.Errorf("parameter batas tidak sah")
	}
	if limit > inboxcloseclaim.MaxLimit {
		// Ditolak, bukan dipangkas diam-diam (`10-API-STRATEGY.md` §4): klien yang meminta
		// seribu baris lalu menerima seratus tanpa diberi tahu akan menampilkan daftar yang
		// ia kira lengkap.
		return usecase.ListQuery{}, fmt.Errorf(
			"parameter batas melebihi maksimum %d", inboxcloseclaim.MaxLimit)
	}

	offset, err := positiveInt(q.Get("lewati"), 0)
	if err != nil {
		return usecase.ListQuery{}, fmt.Errorf("parameter lewati tidak sah")
	}

	return usecase.ListQuery{
		PortalAlias: activePortal.Alias,
		Filter: inboxcloseclaim.Filter{
			Search:       strings.TrimSpace(q.Get("cari")),
			PolicyNumber: strings.TrimSpace(q.Get("no_polis")),
			ClaimNumber:  strings.TrimSpace(q.Get("no_klaim")),
			TechnicalPIC: strings.TrimSpace(q.Get("pic")),
			Business:     business,
			Transfer:     transfer,
			Payment:      payment,
			Limit:        limit,
			Offset:       offset,
		},
	}, nil
}

// positiveInt membaca bilangan bulat tak negatif; kosong berarti nilai baku.
func positiveInt(raw string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("bukan bilangan bulat tak negatif")
	}
	return value, nil
}

// logPendingLookupFailure mencatat kegagalan membaca permintaan tertunda.
//
// Ia WAJIB dipanggil pada setiap jalur yang memakai ListResult. Kegagalan itu tidak
// menghentikan layar — daftarnya tetap tampil — sehingga tanpa catatan ini, tabel yang belum
// dibuat DBA menjadi tidak terlihat oleh siapa pun sampai seseorang melaporkan bahwa
// penanda permintaan tidak pernah muncul.
func (h *Handler) logPendingLookupFailure(r *http.Request, result usecase.ListResult) {
	if result.PendingLookupError == nil {
		return
	}
	logging.From(r.Context(), h.logger).Warn(
		"permintaan tertunda tidak dapat dibaca; penanda di layar tidak akan muncul",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", result.PendingLookupError.Error()),
	)
}

// logExportInterrupted mencatat export yang berhenti di tengah.
func (h *Handler) logExportInterrupted(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Warn("export berhenti sebelum selesai",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
