package inboxinvestigatorhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// InvestigationDTO adalah bentuk formulir investigasi di kontrak API.
//
// Nama medannya berbahasa Indonesia dan `snake_case`, mengikuti kontrak API aplikasi ini
// (`D-80`): ia dipakai klien, bukan nama internal.
//
// Seluruh medan pilihan dikirim sebagai TEKS — `"1"`/`"0"` dan `"true"`/`"false"` — bukan
// sebagai angka atau boolean. Itu nilai yang benar-benar tersimpan, dan berkas Export Data
// Investigation menuliskannya apa adanya (`P-5`). Menerjemahkannya di sini berarti
// menerjemahkan dua kali dan membuat kedua berkas tidak dapat dibandingkan.
type InvestigationDTO struct {
	Referensi    string `json:"referensi"`
	UrutanSurvei int    `json:"urutan_survei"`
	Urutan       int    `json:"urutan"`

	TanggalInvestigasi      *string `json:"tanggal_investigasi"`
	DapatDiinvestigasi      string  `json:"dapat_diinvestigasi"`
	TempatKejadian          string  `json:"tempat_kejadian"`
	NamaRumahSakit          string  `json:"nama_rumah_sakit"`
	NamaTempatLainnya       string  `json:"nama_tempat_lainnya"`
	AlamatRsKlinik          string  `json:"alamat_rs_klinik"`
	NomorRekamMedik         string  `json:"nomor_rekam_medik"`
	NamaPasien              string  `json:"nama_pasien"`
	TanggalLahir            *string `json:"tanggal_lahir"`
	VerifikasiTanggalLahir  string  `json:"verifikasi_tanggal_lahir"`
	KeteranganTanggalLahir  string  `json:"keterangan_tanggal_lahir"`
	PesertaTerdaftar        string  `json:"peserta_terdaftar"`
	KeteranganPendaftaran   string  `json:"keterangan_pendaftaran"`
	TanggalPerawatan        *string `json:"tanggal_perawatan"`
	TanggalSelesaiPerawatan *string `json:"tanggal_selesai_perawatan"`
	TotalPengajuan          string  `json:"total_pengajuan"`
	TagihanLunas            string  `json:"tagihan_lunas"`
	BayarPasien             string  `json:"bayar_pasien"`
	BayarPerusahaan         string  `json:"bayar_perusahaan"`
	BayarAsuransiLain       string  `json:"bayar_asuransi_lain"`
	TidakAdaPembayaran      string  `json:"tidak_ada_pembayaran"`
	NamaAsuransiLain        string  `json:"nama_asuransi_lain"`
	KonfirmasiKwitansi      string  `json:"konfirmasi_kwitansi"`
	NamaPicRs               string  `json:"nama_pic_rs"`
	NamaPenelepon           string  `json:"nama_penelepon"`
	NamaKaryawan            string  `json:"nama_karyawan"`
	KodeAreaTelepon         string  `json:"kode_area_telepon"`
	NomorTelepon            string  `json:"nomor_telepon"`
	EkstensiTelepon         string  `json:"ekstensi_telepon"`
	HasilInvestigasi        string  `json:"hasil_investigasi"`
}

// InvestigationResponse adalah jawaban GET formulir investigasi.
type InvestigationResponse struct {
	Investigasi InvestigationDTO `json:"investigasi"`

	// Tampil menyatakan isian mana yang tampil pada keadaan sekarang.
	//
	// Ia dikirim SERVER, bukan dihitung ulang layar. Keenam syaratnya adalah aturan yang
	// dibaca dari `pyCondition` section lama, dan aturan yang hidup di dua tempat akan
	// berbeda pada perubahan berikutnya.
	Tampil VisibilityDTO `json:"tampil"`

	Portal string `json:"portal"`
}

// VisibilityDTO adalah keadaan tampil keenam isian bersyarat.
type VisibilityDTO struct {
	NamaRumahSakit          bool `json:"nama_rumah_sakit"`
	NamaTempatLainnya       bool `json:"nama_tempat_lainnya"`
	AlamatRsKlinik          bool `json:"alamat_rs_klinik"`
	TanggalSelesaiPerawatan bool `json:"tanggal_selesai_perawatan"`
	NamaAsuransiLain        bool `json:"nama_asuransi_lain"`
}

// SubmitResponse adalah jawaban POST formulir investigasi.
type SubmitResponse struct {
	// Pindah menyatakan apa yang terjadi pada klaimnya.
	//
	// Dikirim supaya layar dapat MENYEBUTKANNYA. Pekerjaan hilang dari antrean setelah
	// disimpan, dan tanpa keterangan itu ia terbaca seperti data yang lenyap.
	Pindah TransitionDTO `json:"pindah"`
}

// TransitionDTO adalah perpindahan klaim ke Analyst.
type TransitionDTO struct {
	Referensi    string `json:"referensi"`
	StatusSurvei string `json:"status_survei"`
	StatusPNC    string `json:"status_pnc"`
	StatusKlaim  string `json:"status_klaim"`
	Pada         string `json:"pada"`
}

// Investigation menangani GET /inbox/investigator/{referensi}/investigasi.
//
// # Ia membuka FORMULIR, bukan menampilkan klaim
//
// Padanannya di sistem lama adalah pra-proses Flow Action `InputInvestigator`, yaitu
// `PresetInvestigation` — yang mengisi Tanggal Investigasi dengan waktu kini.
//
// Ia GET meski mengisi sesuatu, karena yang diisi TIDAK disimpan: nilainya hanya
// dikembalikan ke layar sebagai isian awal, dan baru tersimpan bila pengguna menekan
// Simpan. Membuka formulir berkali-kali karena itu tidak mengubah apa pun.
func (h *Handler) Investigation(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	reference := strings.TrimSpace(chi.URLParam(r, "referensi"))
	if reference == "" {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Pekerjaan yang diinvestigasi tidak disebut.",
		})
		return
	}

	one, err := h.service.OpenInvestigation(r.Context(), active.Alias, reference)
	if err != nil {
		h.writeInvestigationError(w, r, err)
		return
	}

	// Rawat inap datang dari klaim (`ClaimData.SelectRawatInap`), dan kolomnya belum
	// dipastikan ada setelah T_CLAIM_PNC dipangkas — lihat
	// `docs/ddl/tc_pnc_investigasi.sql` §3. Sampai itu jelas, kedua isian rawat inap
	// diperlakukan TAMPIL: menyembunyikan isian yang mungkin seharusnya terisi lebih
	// merugikan daripada menampilkan isian yang mungkin tidak perlu.
	const inpatientUnknown = true

	h.writeResponse(w, r, http.StatusOK, InvestigationResponse{
		Investigasi: toInvestigationDTO(one),
		Tampil:      toVisibilityDTO(inboxinvestigator.VisibleFor(one, inpatientUnknown)),
		Portal:      active.Alias,
	})
}

// SubmitInvestigation menangani POST /inbox/investigator/{referensi}/investigasi.
//
// # Ia POST, dan itu bukan pilihan gaya
//
// Ia MENGUBAH keadaan: menyimpan hasil investigasi dan memindahkan klaim dari Investigator
// ke Analyst. Pekerjaannya HILANG dari antrean setelah ini.
//
// Padanannya `SetStatusInvestigator_Act`, local action Flow Action `InputInvestigator`.
func (h *Handler) SubmitInvestigation(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	reference := strings.TrimSpace(chi.URLParam(r, "referensi"))
	if reference == "" {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Pekerjaan yang diinvestigasi tidak disebut.",
		})
		return
	}

	var body InvestigationDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Isian formulir tidak dapat dibaca.",
		})
		return
	}

	// Referensi diambil dari JALUR, bukan dari badan permintaan. Keduanya dapat berbeda,
	// dan yang menang harus yang sudah melewati pemeriksaan portal — kalau tidak, satu
	// permintaan dapat menyimpan hasil investigasi atas pekerjaan milik entitas lain
	// (`R-20`).
	body.Referensi = reference

	form, problem := fromInvestigationDTO(body)
	if problem != "" {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: problem,
		})
		return
	}

	const inpatientUnknown = true

	move, err := h.service.SubmitInvestigation(
		r.Context(), active.Alias, form, inpatientUnknown, callerOf(r))
	if err != nil {
		h.writeInvestigationError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SubmitResponse{
		Pindah: TransitionDTO{
			Referensi:    move.ClaimRef,
			StatusSurvei: move.SurveyStatus,
			StatusPNC:    move.PNCStatus,
			StatusKlaim:  move.ClaimStatus,
			Pada:         move.At.UTC().Format(dateTimeLayout),
		},
	})
}

// writeInvestigationError memetakan galat formulir menjadi jawaban yang terbaca.
//
// Tiga kelompok, dan masing-masing perlu kalimatnya sendiri:
//
//	isian tidak sah          400 — yang salah isiannya; sebutkan isian mana
//	tabel belum tersedia     503 — yang salah BUKAN pengguna; sebutkan kepada siapa bertanya
//	selebihnya               diserahkan ke pemetaan bersama
//
// Kelompok kedua yang paling penting dibedakan. Selama
// `POOLDATA.TC_PNC_INVESTIGASI` belum dibuat DBA, setiap simpan akan gagal — dan petugas
// yang membaca "terjadi kesalahan pada sistem" akan mencoba berulang kali.
func (h *Handler) writeInvestigationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, inboxinvestigator.ErrInvestigationInvalid):
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: investigationMessage(err),
		})
	case errors.Is(err, inboxinvestigator.ErrInvestigationStoreMissing):
		h.writeResponse(w, r, http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeStoreMissing,
			Message: "Penyimpanan hasil investigasi belum disiapkan di basis data entitas " +
				"ini. Hubungi administrator Claim PNC — mengulang tidak akan menolong.",
		})
	default:
		h.writeModuleError(w, r, err)
	}
}

// investigationMessage membuang awalan teknis galat domain.
func investigationMessage(err error) string {
	message := err.Error()
	const prefix = "isian investigasi tidak sah: "
	if strings.HasPrefix(message, prefix) {
		return "Formulir belum dapat disimpan: " + message[len(prefix):] + "."
	}
	return "Formulir belum dapat disimpan: isiannya belum lengkap."
}

// callerOf mengambil identitas pemanggil untuk kolom jejak.
//
// Modul ini tidak punya CallerReader — antrean bersama tidak memerlukannya. Jalur TULIS
// memerlukannya, dan sampai pembacanya dirakit cmd, nilainya diambil dari konteks sesi
// yang sudah dipasang middleware auth.
//
// Kosong BUKAN galat: kolom jejak yang kosong lebih baik daripada menolak menyimpan hasil
// investigasi yang sudah diketik pengguna. Yang mencatat siapa mengubah apa secara mengikat
// adalah modul `S-5` Jejak Audit, yang belum dibangun.
func callerOf(r *http.Request) string {
	if value, ok := r.Context().Value(callerKey).(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

// callerKey adalah kunci konteks identitas pemanggil.
type callerContextKey struct{}

var callerKey = callerContextKey{}

// toInvestigationDTO memetakan domain menjadi bentuk kontrak.
func toInvestigationDTO(one inboxinvestigator.Investigation) InvestigationDTO {
	return InvestigationDTO{
		Referensi:               one.ClaimRef,
		UrutanSurvei:            one.SurveyIndex,
		Urutan:                  one.Index,
		TanggalInvestigasi:      momentText(one.InvestigatedAt),
		DapatDiinvestigasi:      one.Investigated,
		TempatKejadian:          one.HospitalKindCode,
		NamaRumahSakit:          one.HospitalName,
		NamaTempatLainnya:       one.OtherPlaceName,
		AlamatRsKlinik:          one.HospitalAddress,
		NomorRekamMedik:         one.MedicalRecordNumber,
		NamaPasien:              one.PatientName,
		TanggalLahir:            momentText(one.DateOfBirth),
		VerifikasiTanggalLahir:  one.BirthDateVerified,
		KeteranganTanggalLahir:  one.BirthDateNote,
		PesertaTerdaftar:        one.PatientRegistered,
		KeteranganPendaftaran:   one.RegistrationNote,
		TanggalPerawatan:        momentText(one.TreatmentStart),
		TanggalSelesaiPerawatan: momentText(one.TreatmentEnd),
		TotalPengajuan:          one.BillTotal,
		TagihanLunas:            one.BillSettled,
		BayarPasien:             one.PaidByPatient,
		BayarPerusahaan:         one.PaidByCompany,
		BayarAsuransiLain:       one.PaidByOtherInsurer,
		TidakAdaPembayaran:      one.NoPayment,
		NamaAsuransiLain:        one.OtherInsurer,
		KonfirmasiKwitansi:      one.ReceiptConfirmation,
		NamaPicRs:               one.HospitalPIC,
		NamaPenelepon:           one.CallerName,
		NamaKaryawan:            one.StaffName,
		KodeAreaTelepon:         one.PhoneArea,
		NomorTelepon:            one.Phone,
		EkstensiTelepon:         one.PhoneExt,
		HasilInvestigasi:        one.Remarks,
	}
}

// fromInvestigationDTO memetakan bentuk kontrak menjadi domain.
//
// Tanggal yang tidak dapat dibaca menghasilkan penolakan BERKALIMAT, bukan tanggal nol:
// tanggal nol tahun 1 akan tersimpan tanpa satu pun tanda bahwa yang diketik pengguna tidak
// terbaca.
func fromInvestigationDTO(body InvestigationDTO) (inboxinvestigator.Investigation, string) {
	investigatedAt, problem := momentOf(body.TanggalInvestigasi, "Tanggal Investigasi")
	if problem != "" {
		return inboxinvestigator.Investigation{}, problem
	}
	birth, problem := momentOf(body.TanggalLahir, "Tanggal Lahir")
	if problem != "" {
		return inboxinvestigator.Investigation{}, problem
	}
	treatmentStart, problem := momentOf(body.TanggalPerawatan, "Tanggal Perawatan/Kejadian")
	if problem != "" {
		return inboxinvestigator.Investigation{}, problem
	}
	treatmentEnd, problem := momentOf(body.TanggalSelesaiPerawatan, "Tanggal Selesai Perawatan")
	if problem != "" {
		return inboxinvestigator.Investigation{}, problem
	}

	return inboxinvestigator.Investigation{
		ClaimRef:            body.Referensi,
		SurveyIndex:         body.UrutanSurvei,
		Index:               body.Urutan,
		InvestigatedAt:      investigatedAt,
		Investigated:        body.DapatDiinvestigasi,
		HospitalKindCode:    body.TempatKejadian,
		HospitalName:        body.NamaRumahSakit,
		OtherPlaceName:      body.NamaTempatLainnya,
		HospitalAddress:     body.AlamatRsKlinik,
		MedicalRecordNumber: body.NomorRekamMedik,
		PatientName:         body.NamaPasien,
		DateOfBirth:         birth,
		BirthDateVerified:   body.VerifikasiTanggalLahir,
		BirthDateNote:       body.KeteranganTanggalLahir,
		PatientRegistered:   body.PesertaTerdaftar,
		RegistrationNote:    body.KeteranganPendaftaran,
		TreatmentStart:      treatmentStart,
		TreatmentEnd:        treatmentEnd,
		BillTotal:           body.TotalPengajuan,
		BillSettled:         body.TagihanLunas,
		PaidByPatient:       body.BayarPasien,
		PaidByCompany:       body.BayarPerusahaan,
		PaidByOtherInsurer:  body.BayarAsuransiLain,
		NoPayment:           body.TidakAdaPembayaran,
		OtherInsurer:        body.NamaAsuransiLain,
		ReceiptConfirmation: body.KonfirmasiKwitansi,
		HospitalPIC:         body.NamaPicRs,
		CallerName:          body.NamaPenelepon,
		StaffName:           body.NamaKaryawan,
		PhoneArea:           body.KodeAreaTelepon,
		Phone:               body.NomorTelepon,
		PhoneExt:            body.EkstensiTelepon,
		Remarks:             body.HasilInvestigasi,
	}, ""
}

// toVisibilityDTO memetakan keadaan tampil menjadi bentuk kontrak.
func toVisibilityDTO(shown inboxinvestigator.Visible) VisibilityDTO {
	return VisibilityDTO{
		NamaRumahSakit:          shown.HospitalName,
		NamaTempatLainnya:       shown.OtherPlaceName,
		AlamatRsKlinik:          shown.HospitalAddress,
		TanggalSelesaiPerawatan: shown.TreatmentEnd,
		NamaAsuransiLain:        shown.OtherInsurer,
	}
}

// momentText menuliskan waktu dalam bentuk kontrak, atau nil bila kosong.
func momentText(moment *time.Time) *string {
	if moment == nil {
		return nil
	}
	text := moment.UTC().Format(dateTimeLayout)
	return &text
}

// momentOf membaca waktu dari bentuk kontrak.
func momentOf(value *string, label string) (*time.Time, string) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, ""
	}
	moment, err := time.Parse(dateTimeLayout, strings.TrimSpace(*value))
	if err != nil {
		return nil, label + " tidak dapat dibaca."
	}
	return &moment, ""
}
