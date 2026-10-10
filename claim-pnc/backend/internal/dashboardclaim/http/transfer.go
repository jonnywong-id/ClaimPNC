package dashboardclaimhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxTransferBodyBytes membatasi badan permintaan Transfer.
//
// Formnya lima isian pendek; batas ini jauh di atasnya dan masih mencegah satu permintaan
// menahan memori. Badan yang lebih besar ditolak sebelum diurai, bukan sesudahnya.
const maxTransferBodyBytes = 4 << 10

// Caller adalah identitas pemanggil yang dibutuhkan modul ini.
//
// Dideklarasikan DI SINI, sesempit yang dibutuhkan, supaya modul ini tidak mengimpor modul
// auth. Jembatannya dipasang di cmd — keduanya karena itu tetap tidak saling mengenal.
type Caller struct {
	Login string
	Name  string
}

// transferRequestDTO adalah badan permintaan POST /dashboard-claim/transfer.
type transferRequestDTO struct {
	Lingkup      string `json:"lingkup"`
	KlaimID      string `json:"klaim_id"`
	NomorKlaim   string `json:"nomor_klaim"`
	UserIDLama   string `json:"user_id_lama"`
	UserIDBaru   string `json:"user_id_baru"`
	TipePengguna string `json:"tipe_pengguna"`
	Alasan       string `json:"alasan"`
}

// transferResponseDTO adalah jawabannya.
type transferResponseDTO struct {
	Permintaan transferDTO `json:"permintaan"`
	Portal     string      `json:"portal"`
}

// transferDTO adalah satu permintaan transfer.
type transferDTO struct {
	ID           string `json:"id"`
	Lingkup      string `json:"lingkup"`
	KlaimID      string `json:"klaim_id,omitempty"`
	NomorKlaim   string `json:"nomor_klaim,omitempty"`
	UserIDLama   string `json:"user_id_lama,omitempty"`
	UserIDBaru   string `json:"user_id_baru"`
	TipePengguna string `json:"tipe_pengguna,omitempty"`
	Alasan       string `json:"alasan,omitempty"`
	Status       string `json:"status"`

	// JumlahPindah adalah BERAPA klaim yang berpindah.
	//
	// Tidak `omitempty`: nol adalah jawaban yang bermakna di sini — "tidak ada klaim milik
	// petugas itu" — dan menghilangkannya dari badan jawaban membuat layar tidak dapat
	// membedakannya dari jawaban yang memang tidak membawa angka.
	JumlahPindah int `json:"jumlah_pindah"`

	Pemohon     string `json:"pemohon"`
	PemohonNama string `json:"pemohon_nama,omitempty"`
	Pada        string `json:"pada"`
}

// Transfer menjawab POST /dashboard-claim/transfer.
//
// Menggantikan tombol "Transfer" pada setiap baris dan "Transfer All Case By UserID" pada
// layar lama — keduanya di satu rute, dibedakan field `lingkup`.
//
// # Ia MEMINDAHKAN, bukan mencatat permintaan
//
// Sebelumnya rute ini menulis baris permintaan ke tabel milik aplikasi dan menyerahkan
// pelaksanaannya kepada Pega. Itu dicabut (Work Owner, 2026-10-06): **tidak ada satu pun job
// Pega yang membaca tabel itu**, sehingga permintaannya tidak pernah akan dijalankan.
//
// Yang dilakukan sekarang sama dengan yang dilakukan Pega: mengubah PIC Teknik pada klaimnya
// — satu kolom, `USERTEKNIS_1`. Rinciannya di `pindahpic.sql`.
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.caller(r.Context())
	if !known || caller.Login == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, mencatat permintaan tanpa
		// pemohon menghapus satu-satunya kontrol pengimbang yang tersisa (`D-59`).
		h.writeResponse(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Identitas pemohon tidak terbaca dari sesi.",
		})
		return
	}

	var body transferRequestDTO
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxTransferBodyBytes))
	if err := decoder.Decode(&body); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Badan permintaan tidak dapat dibaca.",
		})
		return
	}

	scope, valid := dashboardclaim.ParseTransferScope(body.Lingkup)
	if !valid {
		// Dijawab sebagai pelanggaran validasi, bukan 400: bentuk permintaannya benar, yang
		// salah isinya — dan frontend menempelkan pesannya ke isian yang bersangkutan.
		h.writeError(w, r, dashboardclaim.NewValidationError([]dashboardclaim.Violation{{
			Field:   dashboardclaim.FieldScope,
			Message: "Lingkup transfer tidak dikenal.",
		}}))
		return
	}

	saring, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	request, err := h.service.Transfer(r.Context(), usecase.TransferCommand{
		PortalAlias: active.Alias,
		Caller:      usecase.TransferCaller{Login: caller.Login, Name: caller.Name},
		Request: dashboardclaim.TransferCommand{
			Scope: scope,

			// Penyaring dibaca dari PARAMETER KUERI permintaan ini, dengan pembaca yang sama
			// dengan daftarnya (`readFilter`). Itu yang membuat "Select All" memindahkan tepat
			// klaim yang terlihat: dua pembaca berbeda akan menyimpang diam-diam begitu salah
			// satunya berubah.
			//
			// Pada lingkup selain `saring` isinya diabaikan, jadi membacanya selalu tidak
			// berbahaya — dan menyusunnya bersyarat justru menyembunyikan dari mana ia datang.
			Filter: saring,

			ClaimID:      body.KlaimID,
			ClaimNumber:  body.NomorKlaim,
			FromOperator: body.UserIDLama,
			ToOperator:   body.UserIDBaru,
			UserType:     dashboardclaim.UserType(strings.TrimSpace(body.TipePengguna)),
			Reason:       body.Alasan,
		},
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, transferResponseDTO{
		Permintaan: adaptTransfer(request, h.location),
		Portal:     active.Alias,
	})
}

// adaptTransfer memetakan satu permintaan menjadi DTO.
func adaptTransfer(request dashboardclaim.TransferRequest, loc *time.Location) transferDTO {
	return transferDTO{
		ID:           request.ID,
		Lingkup:      string(request.Scope),
		KlaimID:      request.ClaimID,
		NomorKlaim:   request.ClaimNumber,
		UserIDLama:   request.FromOperator,
		UserIDBaru:   request.ToOperator,
		TipePengguna: string(request.UserType),
		Alasan:       request.Reason,
		Status:       string(request.Status),
		JumlahPindah: request.MovedCount,
		Pemohon:      request.RequestedBy,
		PemohonNama:  request.RequestedByName,
		Pada:         formatDateTime(request.RequestedAt, loc),
	}
}

// formatDateTime memformat stempel waktu jejak.
//
// Berbeda dari formatDate: jejak permintaan menyebutkan JAM, karena dua permintaan pada hari
// yang sama harus dapat diurutkan oleh orang yang membacanya.
func formatDateTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}

// callerBridge adalah bentuk jembatan identitas yang disuntik cmd.
type callerBridge func(ctx context.Context) (Caller, bool)

// picDTO adalah satu baris daftar PIC Teknik.
//
// Nama field mengikuti grid layar lama: kolomnya berjudul **"Nama"**, dan tombolnya
// mengirim OPERATOR_ID.
type picDTO struct {
	OperatorID string `json:"operator_id"`
	Nama       string `json:"nama"`
	Email      string `json:"email"`
	Tim        string `json:"tim"`

	// Beban adalah COUNTER_QUOTA — pencacah yang dipakai pemilihan otomatis (`R-04`).
	Beban int `json:"beban"`
}

// picResponse membungkus daftarnya.
type picResponse struct {
	PIC     []picDTO      `json:"pic"`
	Halaman paginationDTO `json:"halaman"`
	Portal  string        `json:"portal"`
}

// TechnicalPIC melayani `GET /pic-teknik`.
//
// Lini bisnis **tidak** menyaring apa pun — lihat `picteknik.sql`. Parameternya masih
// diterima karena layar masih mengirimnya, dan membuang isiannya adalah perubahan tersendiri.
func (h *Handler) TechnicalPIC(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Lini bisnis datang dari LAYAR, dan itu keterbatasan yang dinyatakan — bukan replikasi.
	//
	// Di Pega penyaringnya `OperatorID.pyPosition`
	// (`PNCTransferManagement_sec:3494` → `GCNMTransferAssignmentManager_act:967`), dan
	// properti itu di sana rupanya diisi "NONMBU", "TRAVEL", "BONDING", atau "PA" — bukan
	// jabatan dalam arti biasa.
	//
	// Nilai itu TIDAK tersedia di sistem baru: HCC/HCQ mengembalikan jabatan sebenarnya
	// (`Placement.PositionName`), dan `auth.User.Position` memuat itu. Memakainya sebagai
	// penyaring akan mencocokkan "Staf" dengan `TYPE_BUSINESS` dan mengembalikan nol baris —
	// daftar kosong yang tidak menjelaskan dirinya.
	//
	// Modul `inboxprogressclaim` sudah menempuh persoalan yang sama dan menyelesaikannya
	// dengan cara yang sama: pengguna memilih lini bisnis, dan keterbatasannya dinyatakan di
	// layar. Sampai pemetaan pengguna ke lini bisnis menjadi master data (`F-4`), itu
	// jawaban yang paling jujur.
	//
	// Kosong TIDAK ditolak — Pega pun tidak menolaknya; ia hanya tidak mengembalikan baris.
	lini := strings.TrimSpace(r.URL.Query().Get("lini_bisnis"))

	filter, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	hasil, err := h.service.TechnicalPIC(r.Context(), usecase.TechnicalPICQuery{
		PortalAlias:  active.Alias,
		BusinessType: lini,
		Search:       filter.Search,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	daftar := make([]picDTO, 0, len(hasil.Page.Rows))
	for _, row := range hasil.Page.Rows {
		daftar = append(daftar, picDTO{
			OperatorID: row.OperatorID,
			Nama:       row.Name,
			Email:      row.Email,
			Tim:        row.TeamGroup,
			Beban:      row.Workload,
		})
	}

	h.writeResponse(w, r, http.StatusOK, picResponse{
		PIC:     daftar,
		Halaman: pagination(hasil.Page.Total, filter.Limit, filter.Offset),
		Portal:  active.Alias,
	})
}
