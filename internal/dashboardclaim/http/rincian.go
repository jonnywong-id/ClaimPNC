package dashboardclaimhttp

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// claimDetailDTO adalah rincian satu klaim.
type claimDetailDTO struct {
	KlaimID      string `json:"klaim_id"`
	NomorKlaim   string `json:"nomor_klaim"`
	StatusProses string `json:"status_proses"`
	StatusKlaim  string `json:"status_klaim"`
	PICTeknik    string `json:"pic_teknik"`
	AdminPNC     string `json:"admin_pnc"`
	Didaftarkan  string `json:"didaftarkan_pada"`

	// Kedua kode lini bisnis yang menentukan VARIAN grid di layar.
	//
	// Dikirim sebagai KODE, bukan sebagai keputusan "tampilkan varian mana": pemetaan kode ke
	// varian mengikuti rule Pega dan hidup di layar bersama spesifikasi gridnya, sehingga
	// keduanya berubah bersamaan.
	GroupPanel  string `json:"group_panel"`
	JenisBisnis string `json:"jenis_bisnis"`

	// CatatanInputor dari `T_CLAIM_PNC.REMARK`.
	//
	// "Aging Amount" yang berdampingan dengannya di layar lama TIDAK dikirim: ia milik data
	// polis (`Policy.AgingAmount`), dan polis bukan milik bounded context ini (`D-04`).
	// Isiannya tetap kosong, dan itu dinyatakan — bukan diisi nilai karangan.
	CatatanInputor string `json:"catatan_inputor"`

	// Ketujuh bagian di bawah adalah ISI popup, dibaca dari TABEL RELASIONAL.
	//
	// Dulu satu field `dokumen` berisi dokumen JSON apa adanya, dan alasannya tercatat:
	// bentuk dokumen itu belum pernah diperiksa. Alasan itu tidak berlaku lagi — sumbernya
	// kini tabel yang kolomnya dibaca dari katalog satu per satu.
	//
	// Tiap bagian SELALU berupa larik, tidak pernah `null`: layar tidak perlu menjaga dua
	// keadaan yang artinya sama.
	Penerima   []penerimaRinci   `json:"penerima"`
	Objek      []objekRinci      `json:"objek"`
	Coverage   []coverageRinci   `json:"coverage"`
	Adjustment []adjustmentRinci `json:"adjustment"`
	Survey     []surveyRinci     `json:"survey"`
	Lampiran   []lampiranRinci   `json:"lampiran"`
	Progres    []progresRinci    `json:"progres"`

	// Tiga himpunan isian TUNGGAL — bukan grid, karena layar lama menggambarnya sebagai satu
	// blok nilai, bukan daftar baris.
	//
	// Dikirim sebagai objek bersarang dan bukan diratakan ke kepala, supaya terbaca dari
	// kontraknya saja isian mana milik tab mana. Diratakan, `lokasi` pada tab Registrasi dan
	// `lokasi` pada permintaan survei akan bertabrakan — dan keduanya memang nilai berbeda.
	Registrasi registrasiIsian `json:"registrasi"`
	Survei     surveiIsian     `json:"survei"`
	PUCL       puclIsian       `json:"pucl"`

	Portal string `json:"portal"`
}

// registrasiIsian memuat tiga belas isian tab "Registrasi" yang punya kolom.
//
// Tiga isian layar lama TIDAK ADA di sini, dan ketiadaannya disengaja:
//
//	"Klaim Estimasi" · "Nilai Dalam IDR"   `T_CLAIM_ESTIMASI`, per objek dan per coverage —
//	                                       bukan satu nilai per klaim. Menjumlahkannya adalah
//	                                       keputusan bisnis yang belum diambil
//	"Catatan dari Analyst"                 `M_KOMUNIKASI_PNC`, sebuah PERCAKAPAN — menampilkan
//	                                       pesan terakhirnya saja akan menyembunyikan sisanya
//
// Tanggal dikirim sebagai teks ISO, dan nilai yang tidak ada menjadi teks kosong — sama
// seperti seluruh bagian lain pada kontrak ini.
type registrasiIsian struct {
	TanggalTerimaDokumen   string `json:"tanggal_terima_dokumen"`
	TanggalKejadian        string `json:"tanggal_kejadian"`
	TanggalKeluarRawatInap string `json:"tanggal_keluar_rawat_inap"`
	TanggalLapor           string `json:"tanggal_lapor"`
	TeleponPelapor         string `json:"telepon_pelapor"`
	EmailPelapor           string `json:"email_pelapor"`
	StatusPelapor          string `json:"status_pelapor"`
	StatusPelaporLain      string `json:"status_pelapor_lain"`
	LokasiKejadian         string `json:"lokasi_kejadian"`
	Kelurahan              string `json:"kelurahan"`
	TertanggungTKI         string `json:"tertanggung_tki"`
	CatatanKeAnalyst       string `json:"catatan_ke_analyst"`
	CatatanKePICTeknis     string `json:"catatan_ke_pic_teknis"`
}

// surveiIsian memuat kedelapan isian permintaan survei — `POOLDATA.T_REQ_SURVEY`.
//
// Kedelapannya berpadanan satu lawan satu dengan isian layar lama; tidak ada yang hilang dan
// tidak ada yang ditambahkan.
type surveiIsian struct {
	AtasPermintaan string `json:"atas_permintaan"`
	Lokasi         string `json:"lokasi"`
	Telepon        string `json:"telepon"`
	TanggalRequest string `json:"tanggal_request"`
	Cabang         string `json:"cabang"`
	Surveyor       string `json:"surveyor"`
	EmailSurveyor  string `json:"email_surveyor"`
	NamaObjek      string `json:"nama_objek"`
}

// puclIsian memuat dua dari empat isian tab "Catatan Untuk Analis".
//
// "Pilihan Compliance" dan "Note PilihanCompliance" tidak ada di sini karena **tidak ada
// kolomnya di mana pun** pada skema POOLDATA — dicari menyeluruh lewat `ALL_TAB_COLUMNS`,
// bukan disimpulkan dari satu tabel saja.
type puclIsian struct {
	TanggalDokumenLengkap string `json:"tanggal_dokumen_lengkap"`
	AlasanDokterTolakRCL  string `json:"alasan_dokter_tolak_rcl"`
}

// ClaimDetail melayani `GET /dashboard-claim/klaim/{klaim_id}`.
//
// # Kenapa rute tersendiri, bukan bagian dari daftar
//
// Di Pega, klik nomor klaim menjalankan `setDataViewKlaim_Act` lalu membuka harness
// `ViewTempDetailClaim` sebagai **popup** — pengambilan datanya terpisah dari gridnya, dan
// hanya terjadi saat diklik. Menyertakan dokumen JSON pada setiap baris daftar akan membawa
// puluhan dokumen klaim pada setiap pemuatan halaman.
func (h *Handler) ClaimDetail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Kunci dibaca dari jalur dan DI-DECODE: PZINSKEY memuat spasi
	// (`ASM-FW-GCNMFW-WORK PNC-133`), dan spasi itu ter-encode di alamatnya.
	raw := chi.URLParam(r, "klaim_id")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		decoded = raw
	}

	detail, err := h.service.ClaimDetail(r.Context(), usecase.ClaimDetailQuery{
		PortalAlias: active.Alias,
		ClaimID:     strings.TrimSpace(decoded),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, claimDetailDTO{
		KlaimID:        detail.ClaimID,
		NomorKlaim:     detail.ClaimNumber,
		StatusProses:   detail.ProcessStatus,
		StatusKlaim:    detail.ClaimStatus,
		PICTeknik:      detail.TechnicalPIC,
		AdminPNC:       detail.AdminPNC,
		Didaftarkan:    detail.RegisteredAt,
		GroupPanel:     detail.GroupPanel,
		JenisBisnis:    detail.BusinessType,
		CatatanInputor: detail.InputterNote,
		Penerima:       penerimaDTO(detail.Receivers),
		Objek:          objekDTO(detail.Items),
		Coverage:       coverageDTO(detail.Coverages),
		Adjustment:     adjustmentDTO(detail.Settlements),
		Survey:         barisSurveyDTO(detail.Surveys),
		Lampiran:       lampiranDTO(detail.Attachments),
		Progres:        progresDTO(detail.Progress),
		Registrasi: registrasiIsian{
			TanggalTerimaDokumen:   waktuTeks(detail.Registration.DocumentReceivedAt),
			TanggalKejadian:        waktuTeks(detail.Registration.LossAt),
			TanggalKeluarRawatInap: waktuTeks(detail.Registration.HospitalDischarge),
			TanggalLapor:           waktuTeks(detail.Registration.ReportedAt),
			TeleponPelapor:         detail.Registration.ReporterPhone,
			EmailPelapor:           detail.Registration.ReporterEmail,
			StatusPelapor:          detail.Registration.ReporterRelation,
			StatusPelaporLain:      detail.Registration.ReporterRelationEtc,
			LokasiKejadian:         detail.Registration.LossLocation,
			Kelurahan:              detail.Registration.Village,
			TertanggungTKI:         detail.Registration.InsuredIsTKI,
			CatatanKeAnalyst:       detail.Registration.NoteToAnalyst,
			CatatanKePICTeknis:     detail.Registration.NoteToTechnicalPIC,
		},
		Survei: surveiIsian{
			AtasPermintaan: detail.SurveyRequest.RequestorName,
			Lokasi:         detail.SurveyRequest.Location,
			Telepon:        detail.SurveyRequest.Phone,
			TanggalRequest: waktuTeks(detail.SurveyRequest.RequestedAt),
			Cabang:         detail.SurveyRequest.Branch,
			Surveyor:       detail.SurveyRequest.SurveyorName,
			EmailSurveyor:  detail.SurveyRequest.SurveyorEmail,
			NamaObjek:      detail.SurveyRequest.ItemName,
		},
		PUCL: puclIsian{
			TanggalDokumenLengkap: waktuTeks(detail.PUCL.DocumentCompleteAt),
			AlasanDokterTolakRCL:  detail.PUCL.DoctorRejectReason,
		},
		Portal: active.Alias,
	})
}

// Tujuh bentuk baris isi popup, dan tujuh pemetanya.
//
// Nama field JSON berbahasa Indonesia karena ia KONTRAK yang dibaca layar (`D-80`), sedangkan
// nama tipe dan field Go berbahasa Inggris.
//
// Tiap pemeta mengembalikan larik KOSONG, tidak pernah nil: `[]` dan `null` berarti hal yang
// sama bagi layar, dan dua bentuk untuk satu arti hanya menambah cabang yang harus dijaga.

type penerimaRinci struct {
	Nama   string `json:"nama"`
	Alamat string `json:"alamat"`

	// Dipakai sub-popup "Detail Penerima", bukan grid.
	NamaBank   string `json:"nama_bank"`
	NoRekening string `json:"no_rekening"`
}

type objekRinci struct {
	NamaObjek     string `json:"nama_objek"`
	LokasiObjek   string `json:"lokasi_objek"`
	StatusPeserta string `json:"status_peserta"`
	NIK           string `json:"nik"`
	TanggalLahir  string `json:"tanggal_lahir"`
	Pekerjaan     string `json:"pekerjaan"`
	TipeKendaraan string `json:"tipe_kendaraan"`
	NoRangka      string `json:"no_rangka"`
}

type coverageRinci struct {
	NamaCoverage     string `json:"nama_coverage"`
	KodePlan         string `json:"kode_plan"`
	TSI              string `json:"tsi"`
	MataUang         string `json:"mata_uang"`
	PenyebabKerugian string `json:"penyebab_kerugian"`
	NamaObjek        string `json:"nama_objek"`
}

type adjustmentRinci struct {
	NoAkseptasi      string `json:"no_akseptasi"`
	TanggalAkseptasi string `json:"tanggal_akseptasi"`
	StatusAkseptasi  string `json:"status_akseptasi"`
	JenisPembayaran  string `json:"jenis_pembayaran"`
	NamaPenerima     string `json:"nama_penerima"`
	MataUang         string `json:"mata_uang"`
	NilaiAkseptasi   string `json:"nilai_akseptasi"`
	NilaiUsulan      string `json:"nilai_usulan"`
	TotalKlaim       string `json:"total_klaim"`
	TanggalTransfer  string `json:"tanggal_transfer"`
	StatusPembayaran string `json:"status_pembayaran"`
}

type surveyRinci struct {
	TanggalSurvey string `json:"tanggal_survey"`
	NamaSurveyor  string `json:"nama_surveyor"`
	NamaObjek     string `json:"nama_objek"`
	LokasiObjek   string `json:"lokasi_objek"`
	Status        string `json:"status"`
	Langkah       string `json:"langkah"`

	// Dipakai sub-popup "Hasil Survey" dan "Detail Hasil Surveyor".
	NoReferensi   string `json:"no_referensi"`
	Catatan       string `json:"catatan"`
	LokasiSurvey  string `json:"lokasi_survey"`
	JenisKerugian string `json:"jenis_kerugian"`
}

type lampiranRinci struct {
	Kategori    string `json:"kategori"`
	Nama        string `json:"nama"`
	Ditambahkan string `json:"ditambahkan_pada"`
	Oleh        string `json:"oleh"`
}

type progresRinci struct {
	Tanggal    string `json:"tanggal"`
	KodeTahap  string `json:"kode_tahap"`
	NamaTahap  string `json:"nama_tahap"`
	Status     string `json:"status"`
	Keterangan string `json:"keterangan"`
	Oleh       string `json:"oleh"`

	// Dipakai sub-popup "Detail Progress".
	TindakLanjut string `json:"tindak_lanjut"`
}

func penerimaDTO(baris []dashboardclaim.Receiver) []penerimaRinci {
	keluar := make([]penerimaRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, penerimaRinci{
			Nama:       b.Name,
			Alamat:     b.Address,
			NamaBank:   b.BankName,
			NoRekening: b.AccountNumber,
		})
	}
	return keluar
}

func objekDTO(baris []dashboardclaim.InsuredItem) []objekRinci {
	keluar := make([]objekRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, objekRinci{
			NamaObjek:     b.Name,
			LokasiObjek:   b.Location,
			StatusPeserta: b.ParticipantStatus,
			NIK:           b.IDCard,
			TanggalLahir:  waktuTeks(b.DateOfBirth),
			Pekerjaan:     b.Job,
			TipeKendaraan: b.VehicleType,
			NoRangka:      b.ChassisNumber,
		})
	}
	return keluar
}

func coverageDTO(baris []dashboardclaim.Coverage) []coverageRinci {
	keluar := make([]coverageRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, coverageRinci{
			NamaCoverage:     b.Name,
			KodePlan:         b.PlanCode,
			TSI:              pecahanTeks(b.SumInsured),
			MataUang:         b.Currency,
			PenyebabKerugian: b.CauseOfLoss,
			NamaObjek:        b.ItemName,
		})
	}
	return keluar
}

func adjustmentDTO(baris []dashboardclaim.SettlementLine) []adjustmentRinci {
	keluar := make([]adjustmentRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, adjustmentRinci{
			NoAkseptasi:      b.AcceptanceNumber,
			TanggalAkseptasi: waktuTeks(b.AcceptedAt),
			StatusAkseptasi:  b.AcceptanceStatus,
			JenisPembayaran:  b.PaymentType,
			NamaPenerima:     b.ReceiverName,
			MataUang:         b.Currency,
			NilaiAkseptasi:   pecahanTeks(b.AcceptedValue),
			NilaiUsulan:      pecahanTeks(b.ProposedValue),
			TotalKlaim:       pecahanTeks(b.TotalClaim),
			TanggalTransfer:  waktuTeks(b.TransferredAt),
			StatusPembayaran: b.PaidStatus,
		})
	}
	return keluar
}

func barisSurveyDTO(baris []dashboardclaim.SurveyStep) []surveyRinci {
	keluar := make([]surveyRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, surveyRinci{
			TanggalSurvey: waktuTeks(b.SurveyedAt),
			NamaSurveyor:  b.Surveyor,
			NamaObjek:     b.ItemName,
			LokasiObjek:   b.Location,
			Status:        b.Status,
			Langkah:       b.StepIndex,

			NoReferensi:   b.ReferenceNumber,
			Catatan:       b.Note,
			LokasiSurvey:  b.SurveyLocation,
			JenisKerugian: b.LossType,
		})
	}
	return keluar
}

func lampiranDTO(baris []dashboardclaim.Attachment) []lampiranRinci {
	keluar := make([]lampiranRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, lampiranRinci{
			Kategori:    b.Category,
			Nama:        b.Name,
			Ditambahkan: waktuTeks(b.AddedAt),
			Oleh:        b.AddedBy,
		})
	}
	return keluar
}

func progresDTO(baris []dashboardclaim.ProgressStep) []progresRinci {
	keluar := make([]progresRinci, 0, len(baris))
	for _, b := range baris {
		keluar = append(keluar, progresRinci{
			Tanggal:      waktuTeks(b.RecordedAt),
			KodeTahap:    b.StageCode,
			NamaTahap:    b.StageName,
			Status:       b.Status,
			Keterangan:   b.Note,
			Oleh:         b.RecordedBy,
			TindakLanjut: b.NextFollowUp,
		})
	}
	return keluar
}

// waktuTeks memformat stempel waktu, atau kosong bila tidak ada.
//
// Kosong, bukan "0001-01-01": tanggal nol di layar tidak dapat dibedakan dari tanggal yang
// memang belum diisi, dan yang kedua adalah keadaan yang sah.
func waktuTeks(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// pecahanTeks mengirim nilai uang sebagai TEKS, bukan angka JSON.
//
// `D-51` menetapkan nilai uang disimpan presisi penuh. Angka JSON adalah `float64` di hampir
// semua klien, dan presisi penuh hilang di sana tanpa satu pun tanda. Teks membawanya utuh,
// dan layar yang memutuskan cara menampilkannya.
func pecahanTeks(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}
