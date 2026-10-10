package dashboardclaimhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler.
//
// Dideklarasikan DI SINI, di paket yang memakainya, dan sesempit yang dibutuhkan. Handler
// karena itu dapat diuji dengan ganda sederhana tanpa merakit usecase maupun repo.
type Service interface {
	Counts(ctx context.Context, q usecase.Query) (usecase.CountsResult, error)
	List(ctx context.Context, q usecase.ListQuery) (usecase.ListResult, error)
	Holding(ctx context.Context, q usecase.Query) (usecase.HoldingResult, error)
	Transfer(ctx context.Context, cmd usecase.TransferCommand) (dashboardclaim.TransferRequest, error)
	TechnicalPIC(ctx context.Context, q usecase.TechnicalPICQuery) (usecase.TechnicalPICResult, error)

	// ClaimDetail membaca isi popup yang terbuka saat nomor klaim diklik.
	ClaimDetail(ctx context.Context, q usecase.ClaimDetailQuery) (dashboardclaim.ClaimDetail, error)
}

// Handler melayani rute Dashboard Claim.
type Handler struct {
	service  Service
	logger   *slog.Logger
	location *time.Location
	now      func() time.Time
	caller   callerBridge

	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service
	Logger  *slog.Logger

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	//
	// Ia disuntik, tidak dibaca dari lingkungan di sini, supaya uji dapat menetapkannya dan
	// hasilnya tidak berubah menurut mesin yang menjalankannya.
	//
	// Bawaannya WIB, bukan UTC: waktu disimpan UTC dan DITAMPILKAN WIB (`F-5`). Bawaan UTC
	// akan membuat tanggal bergeser satu hari pada kejadian menjelang tengah malam — dan
	// pergeseran itu tidak terlihat sebagai galat, hanya sebagai tanggal yang salah.
	Location *time.Location

	// WriteResponse dan WriteError disuntik dari cmd supaya bentuk respons seragam di
	// seluruh modul, bukan disusun ulang di setiap handler.
	WriteResponse JSONWriter
	WriteError    ErrorWriter

	// Caller adalah jembatan satu arah ke modul auth, dipasang di cmd supaya kedua modul
	// tetap tidak saling mengimpor.
	//
	// DUA field diambil, berbeda dari modul yang hanya membaca: jejak permintaan transfer
	// menyimpan NAMA pemohon bersama login-nya, supaya jejak itu tetap terbaca utuh tanpa
	// join ke tabel pengguna.
	Caller func(ctx context.Context) (Caller, bool)

	// Now adalah seam ke jam, dipakai menghitung kolom "Lama Waktu Klaim" dan "Aging".
	//
	// Ia disuntik supaya umur dapat diuji dengan angka pasti, dan supaya tidak ada satu pun
	// tempat di dalam modul yang memanggil time.Now() sendiri — pola yang justru melahirkan
	// `Set7Hours` di sistem lama. Kosong berarti jam sistem.
	Now func() time.Time
}

// NewHandler membentuk handler dan menolak Options yang tidak lengkap.
//
// Penolakan terjadi saat aplikasi START, bukan saat pengguna membuka layar.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("dashboardclaim/http: Service wajib diisi")
	}
	if o.WriteResponse == nil {
		return nil, errors.New("dashboardclaim/http: WriteResponse wajib diisi")
	}
	if o.WriteError == nil {
		return nil, errors.New("dashboardclaim/http: WriteError wajib diisi")
	}
	location := o.Location
	if location == nil {
		location = jakarta()
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	caller := o.Caller
	if caller == nil {
		// Kosong berarti tombol Transfer selalu menolak — bukan panik. Modul ini tetap dapat
		// dirakit untuk layar baca-saja tanpa menyentuh modul auth.
		caller = func(context.Context) (Caller, bool) { return Caller{}, false }
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		location:      location,
		now:           now,
		caller:        caller,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// Summary menjawab GET /dashboard-claim/ringkasan.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	filter, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.service.Counts(r.Context(), usecase.Query{
		PortalAlias: active.Alias,
		Filter:      filter,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	counts := result.Counts
	h.writeResponse(w, r, http.StatusOK, summaryResponse{
		Kartu: []cardDTO{
			card(dashboardclaim.TileOutstanding, counts.Outstanding),
			card(dashboardclaim.TileCloseClaim, counts.CloseClaim),
			card(dashboardclaim.TileLossAdjuster, counts.LossAdjuster),
			card(dashboardclaim.TileInternalSurveyor, counts.InternalSurveyor),
		},
		LiniBisnis:       string(result.Filter.Business),
		Portal:           active.Alias,
		CatatanWarisan:   inheritedNotes(),
	})
}

// List menjawab GET /dashboard-claim/{tile}.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	tile, known := dashboardclaim.ParseTile(chi.URLParam(r, "tile"))
	if !known {
		h.writeError(w, r, dashboardclaim.ErrTileNotFound)
		return
	}

	filter, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.service.List(r.Context(), usecase.ListQuery{
		PortalAlias: active.Alias,
		Tile:        tile,
		Filter:      filter,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Kedua larik selalu dibentuk, tidak pernah nil, supaya frontend dapat memetakannya
	// tanpa memeriksa null lebih dulu — dan supaya JSON-nya `[]`, bukan `null`.
	claims := make([]claimDTO, 0, len(result.Claims.Rows))
	for _, row := range result.Claims.Rows {
		claims = append(claims, adaptClaim(row, h.now(), h.location))
	}
	surveys := make([]surveyDTO, 0, len(result.Surveys.Rows))
	for _, row := range result.Surveys.Rows {
		surveys = append(surveys, adaptSurvey(row, h.now(), h.location))
	}

	total := result.Claims.Total
	if result.Shape == dashboardclaim.ShapeSurvey {
		total = result.Surveys.Total
	}

	h.writeResponse(w, r, http.StatusOK, listResponse{
		Tile:       string(result.Tile),
		Judul:      result.Tile.Label(),
		Bentuk:     string(result.Shape),
		Klaim:      claims,
		Survei:     surveys,
		Halaman:    pagination(total, result.Filter.Limit, result.Filter.Offset),
		LiniBisnis: string(result.Filter.Business),
		Portal:     active.Alias,
	})
}

// Metadata menjawab GET /dashboard-claim/penyaring.
//
// Ia TIDAK menyentuh basis data sama sekali — isinya bentuk layar, bukan data. Karena itu ia
// tetap menjawab meski koneksi entitas sedang bermasalah, dan layar dapat menggambar
// kerangkanya sebelum angka apa pun selesai dihitung.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	lines := dashboardclaim.BusinessLines()
	choices := make([]choiceDTO, 0, len(lines))
	for _, line := range lines {
		choices = append(choices, choiceDTO{Nilai: string(line), Label: line.Label()})
	}

	// Kedua dropdown panel penyaring. Daftarnya datang dari SERVER, bukan ditulis ulang di
	// layar: nilai yang dikirim layar dicocokkan dengan konstanta di dalam SQL, sehingga
	// salah ketik satu huruf di frontend akan menghasilkan penyaring yang tidak menyaring
	// apa pun — tanpa galat.
	cashiers := dashboardclaim.CashierStatuses()
	cashierChoices := make([]choiceDTO, 0, len(cashiers))
	for _, status := range cashiers {
		cashierChoices = append(cashierChoices,
			choiceDTO{Nilai: string(status), Label: status.Label()})
	}

	payments := dashboardclaim.PaymentStatuses()
	paymentChoices := make([]choiceDTO, 0, len(payments))
	for _, status := range payments {
		paymentChoices = append(paymentChoices,
			choiceDTO{Nilai: string(status), Label: status.Label()})
	}

	// Isi dropdown "Pilih Type User" pada Transfer All Case. Nilainya dibandingkan sebagai
	// TEKS oleh pelaksananya, sehingga daftarnya datang dari server — salah ketik satu huruf
	// di layar menghasilkan permintaan yang diam saat dijalankan.
	types := dashboardclaim.UserTypes()
	typeChoices := make([]choiceDTO, 0, len(types))
	for _, t := range types {
		typeChoices = append(typeChoices, choiceDTO{Nilai: string(t), Label: t.Label()})
	}

	all := dashboardclaim.Tiles()
	tileList := make([]tileDTO, 0, len(all))
	for _, tile := range all {
		tileList = append(tileList, tileDTO{
			Tile:   string(tile),
			Judul:  tile.Label(),
			Bentuk: string(tile.Shape()),
		})
	}

	h.writeResponse(w, r, http.StatusOK, metadataResponse{
		LiniBisnis:     choices,
		StatusTransfer: cashierChoices,
		StatusBayar:    paymentChoices,
		TipePengguna:   typeChoices,
		Tile:           tileList,
		Portal:         active.Alias,
	})
}

// jakarta mengembalikan zona waktu tampilan.
//
// Jatuh ke offset tetap bila basis data zona waktu tidak tersedia di mesin penjalan —
// lazim pada citra kontainer minimal. Gagal keras di sini akan membuat aplikasi menolak
// start hanya karena berkas zona waktu tidak ikut disalin, dan WIB memang tidak mengenal
// waktu musim panas sehingga offset tetapnya benar sepanjang tahun.
func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// card membentuk satu kartu penghitung.
func card(tile dashboardclaim.Tile, total int) cardDTO {
	return cardDTO{
		Tile:   string(tile),
		Judul:  tile.Label(),
		Jumlah: total,
		Bentuk: string(tile.Shape()),
	}
}

// plannedDifferences menyebutkan selisih yang DISENGAJA terhadap sistem lama.
//
// Daftarnya dikirim ke layar, bukan hanya ditulis di komentar kode, supaya penguji gerbang 1
// membacanya saat membandingkan angka. Selisih yang ditemukan tanpa dinyatakan lebih dulu
// akan dilaporkan sebagai cacat, dan menjelaskannya belakangan jauh lebih mahal (`D-54`).
//
// # Kenapa daftarnya pendek
//
// Work Owner memutuskan 2026-09-26 bahwa ketiga butir terbuka mengikuti Pega APA ADANYA.
// Dua selisih yang sempat direncanakan pada angka kartu survei karena itu DICABUT, dan
// kuerinya dikembalikan ke bentuk aslinya.
//
// Yang tersisa satu, dan ia tidak menyentuh angka mana pun.
func plannedDifferences() []string {
	return []string{
		"Kolom No Klaim pada telusur survei menampilkan NOMOR klaim induk. Sistem lama " +
			"menampilkan kunci internal Pega apa adanya pada tile Internal Surveyor — nilai " +
			"yang memuat nama kelas dan tidak terbaca pengguna (`D-22`).",
	}
}

// inheritedNotes menyebutkan perilaku sistem lama yang DIPERTAHANKAN tetapi perlu dijelaskan.
//
// Ia terpisah dari plannedDifferences dengan sengaja: yang di sana adalah hal yang BERBEDA
// dari Pega, yang di sini adalah hal yang SAMA dengan Pega tetapi mudah dibaca sebagai cacat.
// Menggabungkan keduanya akan membuat penguji gerbang 1 mencari selisih yang tidak ada.
//
// Butir di bawah satu-satunya yang benar-benar mengganggu pembacaan: pengguna melihat satu
// angka pada kartu lalu menemukan jumlah baris yang lain saat menelusurinya. Di sistem lama
// hal itu terjadi tanpa penjelasan apa pun; di sini ia dinyatakan. Yang ditambahkan hanyalah
// penjelasannya — angkanya tidak disentuh.
func inheritedNotes() []string {
	return []string{
		"Angka pada kartu Loss Adjuster menghitung JUMLAH KLAIM yang punya survei, sedangkan " +
			"telusurnya menampilkan BARIS SURVEI. Keduanya karena itu tidak selalu sama.",
		"Angka pada kartu Internal Surveyor menghitung jumlah survei pada klaim yang masih " +
			"berjalan, sedangkan telusurnya menyaring lini bisnis pada baris surveinya sendiri. " +
			"Keduanya karena itu dapat berbeda saat penyaring lini bisnis dipakai.",
		"Klaim yang memegang lebih dari satu penugasan ikut terhitung berkali-kali pada kedua " +
			"kartu survei — perilaku kueri lama yang dipertahankan apa adanya.",
	}
}

// readFilter membaca penyaring dari query string.
//
// Seluruh pelanggaran dikumpulkan, tidak berhenti pada yang pertama: layar lama menampilkan
// seluruh pesan sekaligus, dan mengembalikan satu pesan per percobaan akan menyiksa pengguna
// (`12-CROSSCUTTING` §1.2 butir 1).
//
// Nama parameternya sama persis dengan konstanta Field* di domain, sehingga frontend dapat
// menempelkan pesannya ke isian yang benar tanpa tabel pemetaan.
func readFilter(values url.Values) (dashboardclaim.Filter, error) {
	var violations []dashboardclaim.Violation

	business, known := dashboardclaim.ParseBusinessLine(values.Get("lini_bisnis"))
	if !known {
		violations = append(violations, dashboardclaim.Violation{
			Field:   dashboardclaim.FieldBusinessLine,
			Message: "Lini bisnis tidak dikenal.",
		})
	}

	cashier, known := dashboardclaim.ParseCashierStatus(values.Get("status_transfer"))
	if !known {
		violations = append(violations, dashboardclaim.Violation{
			Field:   dashboardclaim.FieldCashierStatus,
			Message: "Status transfer tidak dikenal.",
		})
	}

	payment, known := dashboardclaim.ParsePaymentStatus(values.Get("status_bayar"))
	if !known {
		violations = append(violations, dashboardclaim.Violation{
			Field:   dashboardclaim.FieldPaymentStatus,
			Message: "Status pembayaran tidak dikenal.",
		})
	}

	page, err := readPositiveInt(values.Get("halaman"))
	if err != nil {
		violations = append(violations, dashboardclaim.Violation{
			Field:   dashboardclaim.FieldPage,
			Message: "Halaman harus berupa angka bulat positif.",
		})
	}

	size, err := readPositiveInt(values.Get("ukuran"))
	if err != nil {
		violations = append(violations, dashboardclaim.Violation{
			Field:   dashboardclaim.FieldSize,
			Message: "Ukuran halaman harus berupa angka bulat positif.",
		})
	}

	if failure := dashboardclaim.NewValidationError(violations); failure != nil {
		return dashboardclaim.Filter{}, failure
	}

	// Halaman dan ukuran adalah KONTRAK; limit dan offset adalah bentuk yang dipakai SQL.
	// Konversinya terjadi di sini, satu tempat — bukan di setiap kueri.
	//
	// Halaman dihitung mulai 1, sehingga halaman 1 tidak melewati satu baris pun. Nol dan
	// kosong keduanya diperlakukan sebagai halaman pertama: layar yang baru dibuka belum
	// menyebut halaman, dan itu bukan kesalahan.
	if page < 1 {
		page = 1
	}
	filter := dashboardclaim.Filter{
		Business: business,
		Search:   values.Get("cari"),

		// Kelima penyaring panel `FilterDashboardClaim_sec`. Namanya di sini mengikuti
		// ISINYA, bukan properti Pega yang menyimpannya — yang terakhir menyesatkan
		// (`TempInputFilter.CaseID` berisi nomor polis, `.City` berisi status pembayaran,
		// `.CityID` berisi status transfer).
		PolicyNumber: values.Get("nomor_polis"),
		ClaimNumber:  values.Get("nomor_klaim"),
		TechnicalPIC: values.Get("pic"),
		Cashier:      cashier,
		Payment:      payment,

		Limit: size,
	}.Normalize()
	filter.Offset = (page - 1) * filter.Limit

	return filter, nil
}

// readPositiveInt membaca angka yang boleh kosong.
//
// Kosong menghasilkan nol, yang kemudian diganti nilai bawaan. Isian yang BUKAN angka
// menghasilkan galat, bukan diam-diam menjadi nol — permintaan `ukuran=banyak` yang dijawab
// dengan halaman bawaan menyembunyikan kesalahan klien.
func readPositiveInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, errors.New("bukan angka bulat tidak negatif")
	}
	return value, nil
}

// Holding menjawab GET /dashboard-claim/tampungan.
//
// Tab kedua layar ini — klaim yang sudah terdaftar tetapi belum punya PIC Teknik.
//
// Ia TIDAK menerima penyaring lini bisnis, dan itu bukan kelalaian: kueri lamanya tidak
// punya penandanya, dan layar lamanya tidak menggambar dropdown Bisnis pada tab ini.
func (h *Handler) Holding(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	filter, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.service.Holding(r.Context(), usecase.Query{
		PortalAlias: active.Alias,
		Filter:      filter,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	rows := make([]holdingDTO, 0, len(result.Page.Rows))
	for _, row := range result.Page.Rows {
		rows = append(rows, adaptHolding(row, h.location))
	}

	h.writeResponse(w, r, http.StatusOK, holdingResponse{
		Klaim:   rows,
		Halaman: pagination(result.Page.Total, result.Filter.Limit, result.Filter.Offset),
		Portal:  active.Alias,
	})
}
