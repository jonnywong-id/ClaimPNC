package masterrekeninghttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/usecase"
)

// Pemanggil adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah tiga field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Opsi.Pemanggil, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
type Pemanggil struct {
	Identitas string
	Nama      string
	Email     string
}

// Layanan adalah bagian usecase yang dibutuhkan handler ini.
//
// Dinyatakan sebagai antarmuka di paket yang MEMAKAInya, bukan di paket yang
// mengisinya (docs/Steering/08-TECHNICAL-STRATEGY.md §2) — itulah yang membuat handler
// dapat diuji tanpa membentuk seluruh layanan beserta repo dan seam Kasirnya.
type Layanan interface {
	Daftar(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Rekening, int, error)
	Ambil(ctx context.Context, k masterrekening.Kunci) (masterrekening.Rekening, error)
	DaftarBank(ctx context.Context) ([]masterrekening.Bank, error)
	Ajukan(ctx context.Context, p usecase.Pengajuan, oleh usecase.Pengaju) (masterrekening.Rekening, error)
	Ubah(ctx context.Context, k masterrekening.Kunci, p usecase.Pengajuan, oleh usecase.Pengaju) (masterrekening.Rekening, error)
	Putuskan(ctx context.Context, k masterrekening.Kunci, putusan usecase.Keputusan, oleh usecase.Komite, logger *slog.Logger) (masterrekening.Rekening, error)
}

// Handler melayani permintaan master rekening.
type Handler struct {
	layanan   Layanan
	pemanggil func(context.Context) (Pemanggil, bool)
	logger    *slog.Logger

	tulisGalat func(w http.ResponseWriter, r *http.Request, err error)
}

// Opsi adalah bahan pembentuk Handler.
type Opsi struct {
	Layanan Layanan

	// Pemanggil membaca identitas pemanggil dari context. Diisi saat perakitan
	// dengan pembaca konteks milik modul auth.
	Pemanggil func(context.Context) (Pemanggil, bool)

	Logger     *slog.Logger
	TulisGalat func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerBaru membentuk handler modul master rekening.
func HandlerBaru(o Opsi) *Handler {
	tulis := o.TulisGalat
	if tulis == nil {
		tulis = TulisGalat(o.Logger)
	}
	return &Handler{
		layanan:    o.Layanan,
		pemanggil:  o.Pemanggil,
		logger:     o.Logger,
		tulisGalat: tulis,
	}
}

// batasTertinggiKlien menahan permintaan halaman yang tidak masuk akal dari peramban.
const batasTertinggiKlien = 200

// Daftar menangani GET /master-rekening.
//
// Lima tab layar lama — Cari Data, Komite Approval, Waiting Approval, Approve, dan
// Reject — seluruhnya dilayani endpoint ini dengan saringan yang berbeda. Membuat lima
// endpoint untuk lima tab berarti lima tempat yang harus diubah setiap kali kolomnya
// bertambah.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	status := masterrekening.StatusApproval(strings.TrimSpace(q.Get("status")))
	if status != "" && !status.Dikenal() {
		h.tulisGalat(w, r, masterrekening.ErrStatusTidakDikenal)
		return
	}

	f := masterrekening.Filter{
		Status:        status,
		NomorRekening: strings.TrimSpace(q.Get("nomor_rekening")),
		NamaPemilik:   strings.TrimSpace(q.Get("nama_pemilik")),
		NamaBank:      strings.TrimSpace(q.Get("nama_bank")),
		Batas:         angka(q.Get("batas"), 50, batasTertinggiKlien),
		Lewati:        angka(q.Get("lewati"), 0, 0),
	}

	// Tab "Komite Approval" hanya menampilkan yang menunggu keputusan komite yang
	// sedang masuk. Identitas komitenya diambil dari sesi, TIDAK dari query string —
	// kalau dari query string, siapa pun dapat melihat antrean komite mana pun.
	if q.Get("komite_saya") == "1" {
		pemanggil, ada := h.identitas(r)
		if !ada {
			h.tulisGalat(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
			return
		}
		f.HanyaKomiteSaya = true
		f.IdentitasKomite = pemanggil.Identitas
	}

	baris, jumlah, err := h.layanan.Daftar(r.Context(), f)
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}

	isi := make([]RekeningDTO, 0, len(baris))
	for _, b := range baris {
		isi = append(isi, DariRekening(b))
	}
	TulisJSON(w, r, http.StatusOK, ResponsDaftar{
		Rekening: isi,
		Jumlah:   jumlah,
		Batas:    f.Batas,
		Lewati:   f.Lewati,
	}, h.logger)
}

// Ambil menangani GET /master-rekening/{kodeBank}/{nomorRekening}.
func (h *Handler) Ambil(w http.ResponseWriter, r *http.Request) {
	rek, err := h.layanan.Ambil(r.Context(), kunciDariJalur(r))
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusOK, DariRekening(rek), h.logger)
}

// DaftarBank menangani GET /master-rekening/bank.
func (h *Handler) DaftarBank(w http.ResponseWriter, r *http.Request) {
	bank, err := h.layanan.DaftarBank(r.Context())
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	isi := make([]BankDTO, 0, len(bank))
	for _, b := range bank {
		isi = append(isi, BankDTO{Kode: b.Kode, Nama: b.Nama})
	}
	TulisJSON(w, r, http.StatusOK, ResponsDaftarBank{Bank: isi}, h.logger)
}

// Ajukan menangani POST /master-rekening.
func (h *Handler) Ajukan(w http.ResponseWriter, r *http.Request) {
	var badan PermintaanSimpan
	if !h.bacaBadan(w, r, &badan) {
		return
	}
	pemanggil, ada := h.identitas(r)
	if !ada {
		h.tulisGalat(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	rek, err := h.layanan.Ajukan(r.Context(), pengajuanDari(badan), usecase.Pengaju{
		Identitas: pemanggil.Identitas,
		Nama:      pemanggil.Nama,
		Email:     pemanggil.Email,
	})
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusCreated, DariRekening(rek), h.logger)
}

// Ubah menangani PUT /master-rekening/{kodeBank}/{nomorRekening}.
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	var badan PermintaanSimpan
	if !h.bacaBadan(w, r, &badan) {
		return
	}
	pemanggil, ada := h.identitas(r)
	if !ada {
		h.tulisGalat(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	rek, err := h.layanan.Ubah(r.Context(), kunciDariJalur(r), pengajuanDari(badan), usecase.Pengaju{
		Identitas: pemanggil.Identitas,
		Nama:      pemanggil.Nama,
		Email:     pemanggil.Email,
	})
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusOK, DariRekening(rek), h.logger)
}

// Putuskan menangani POST /master-rekening/{kodeBank}/{nomorRekening}/keputusan.
func (h *Handler) Putuskan(w http.ResponseWriter, r *http.Request) {
	var badan PermintaanPutuskan
	if !h.bacaBadan(w, r, &badan) {
		return
	}
	pemanggil, ada := h.identitas(r)
	if !ada {
		h.tulisGalat(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	rek, err := h.layanan.Putuskan(r.Context(), kunciDariJalur(r), usecase.Keputusan{
		Status:    masterrekening.StatusApproval(strings.TrimSpace(badan.Status)),
		Catatan:   badan.Catatan,
		IDDokumen: badan.IDDokumen,
	}, usecase.Komite{
		Identitas: pemanggil.Identitas,
		Nama:      pemanggil.Nama,
		Email:     pemanggil.Email,
	}, h.logger)
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusOK, DariRekening(rek), h.logger)
}

func (h *Handler) identitas(r *http.Request) (Pemanggil, bool) {
	if h.pemanggil == nil {
		return Pemanggil{}, false
	}
	return h.pemanggil(r.Context())
}

func (h *Handler) bacaBadan(w http.ResponseWriter, r *http.Request, tujuan any) bool {
	pembaca := json.NewDecoder(r.Body)
	// Field yang tidak dikenal ditolak, bukan diabaikan diam-diam: klien yang salah
	// mengeja nama field harus tahu isiannya tidak sampai, bukan menemukan kolomnya
	// kosong berminggu-minggu kemudian.
	pembaca.DisallowUnknownFields()

	if err := pembaca.Decode(tujuan); err != nil {
		TulisJSON(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		}, h.logger)
		return false
	}
	return true
}

func pengajuanDari(b PermintaanSimpan) usecase.Pengajuan {
	return usecase.Pengajuan{
		NomorRekening:     b.NomorRekening,
		NamaPemilik:       b.NamaPemilik,
		NamaBank:          b.NamaBank,
		CabangBank:        b.CabangBank,
		AlamatBank:        b.AlamatBank,
		KodeBank:          b.KodeBank,
		TipeRekening:      b.TipeRekening,
		Email:             b.Email,
		Telepon:           b.Telepon,
		NIK:               b.NIK,
		IDDokumen:         b.IDDokumen,
		Catatan:           b.Catatan,
		Aktif:             b.Aktif,
		KodeBankLama:      b.KodeBankLama,
		NomorRekeningLama: b.NomorRekeningLama,
		NamaPemilikLama:   b.NamaPemilikLama,
	}
}

func kunciDariJalur(r *http.Request) masterrekening.Kunci {
	return masterrekening.Kunci{
		KodeBank:      strings.TrimSpace(chi.URLParam(r, "kodeBank")),
		NomorRekening: strings.TrimSpace(chi.URLParam(r, "nomorRekening")),
	}
}

// angka membaca bilangan dari query string dengan nilai baku dan batas atas.
//
// Masukan yang tidak dapat dibaca jatuh ke nilai baku alih-alih menjadi galat: sebuah
// nomor halaman yang salah ketik tidak sebanding dengan menolak seluruh permintaan.
func angka(teks string, baku, tertinggi int) int {
	nilai, err := strconv.Atoi(strings.TrimSpace(teks))
	if err != nil || nilai < 0 {
		return baku
	}
	if tertinggi > 0 && nilai > tertinggi {
		return tertinggi
	}
	return nilai
}
