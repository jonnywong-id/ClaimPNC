// Package gateway memenuhi seam archivedokumenklaim.Gateway — jalur keluar ke sistem
// Arsip milik tim lain.
//
// Dua pengisi: klien HTTP nyata di berkas ini, dan perekam di fake.go untuk pengujian dan
// pengembangan lokal. Dua adapter, sehingga seam-nya nyata dan bukan hipotetis
// (`04-FUTURE-ARCHITECTURE.md` §3).
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/archivedokumenklaim"
)

// ArchiveServiceKind adalah nilai kolom TYPESERVICE pada POOLDATA.GCNM_CONNECT_REST yang
// menandai baris alamat layanan Arsip.
//
// # Kenapa alamatnya DATA, bukan konfigurasi aplikasi
//
// Karena begitulah alamat layanan luar sudah diperlakukan di aplikasi ini: provider
// HCC/HCQ membacanya dari tabel yang sama. Perpindahan endpoint karena itu menjadi
// perubahan data yang dilakukan DBA, bukan rilis ulang aplikasi — dan tiap portal entitas
// boleh punya alamat Arsip sendiri.
//
// Ia sekaligus menutup satu hal yang tidak boleh masuk repository: alamat layanan Arsip
// di sistem lama adalah konstanta di dalam rule, dan hostname produksi tidak pernah
// dituliskan ke berkas yang di-commit (`D-69`).
//
// BARISNYA BELUM ADA. `Database/gcnm_connect_rest.csv` hanya memuat satu baris, yaitu
// alamat login HCQ. Permintaan penambahannya tercatat di
// docs/permintaan-artefak-pega.md; sampai barisnya masuk, pengiriman ke cabang gagal
// dengan pesan yang menyebutkan tepat apa yang kurang — bukan gagal diam-diam.
const ArchiveServiceKind = "ARCHIVE-INJECT"

// ServiceCatalog adalah seam ke daftar alamat layanan luar.
//
// Ia dideklarasikan di sini, bukan diimpor dari modul auth: modul tidak saling mengimpor,
// dan yang menjembatani keduanya adalah cmd/claimpnc. Bentuknya sama persis dengan yang
// dipakai provider HCC/HCQ, sehingga satu pengisi melayani keduanya.
type ServiceCatalog interface {
	// ServiceAddress mengembalikan SERVICENAME untuk satu app dan jenis layanan.
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// DefaultTimeout adalah batas waktu pemanggilan layanan Arsip.
//
// Batas waktu WAJIB ada di setiap pemanggilan keluar (`10-API-STRATEGY.md` §8.2); tanpa
// itu, satu layanan yang menggantung akan menghabiskan seluruh koneksi kita. Tiga puluh
// detik mengikuti target NFR untuk pemanggilan sistem eksternal.
const DefaultTimeout = 30 * time.Second

// HTTP mengirim berkas arsip ke layanan Arsip lewat REST.
type HTTP struct {
	catalog ServiceCatalog
	client  *http.Client
}

// Options adalah bahan pembentuk HTTP.
type Options struct {
	Catalog ServiceCatalog

	// Client boleh dikosongkan; bila kosong, dipakai klien dengan DefaultTimeout.
	Client *http.Client
}

// NewHTTP membentuk klien layanan Arsip.
func NewHTTP(o Options) (*HTTP, error) {
	if o.Catalog == nil {
		return nil, errors.New("archivedokumenklaim/gateway: Catalog wajib diisi")
	}

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}

	return &HTTP{catalog: o.Catalog, client: client}, nil
}

// requestBody adalah badan permintaan yang diterima layanan Arsip.
//
// Nama fieldnya ditulis PERSIS seperti yang disusun
// `Activity/SendDataArchiveDOcumentByService-Act.xml` — huruf besarnya ikut menentukan,
// dan ia kontrak dengan sistem milik tim lain, bukan penamaan milik kita.
type requestBody struct {
	DocumentNumber string `json:"NoDokumen"`
	RequestedOn    string `json:"TglRequest"`
}

// responseBody adalah jawaban layanan Arsip.
//
// Kedua fieldnya diturunkan dari pemetaan jawaban di activity lama, yang menyalin
// `ResponseCode` dan `ResponseMessage` ke properti klipboard lalu menyimpannya ke kolom
// KODESERVICE dan NOTESERVICE.
type responseBody struct {
	Code    string `json:"ResponseCode"`
	Message string `json:"ResponseMessage"`
}

// Send mengirim satu berkas arsip dan mengembalikan jawabannya.
func (h *HTTP) Send(
	ctx context.Context,
	portalAlias string,
	shipment archivedokumenklaim.Shipment,
) (archivedokumenklaim.Receipt, error) {
	address, err := h.catalog.ServiceAddress(ctx, portalAlias, ArchiveServiceKind)
	if err != nil || strings.TrimSpace(address) == "" {
		// Alamat yang belum terdaftar dibedakan dari layanan yang menolak: keduanya
		// ditangani orang yang berbeda — yang pertama DBA, yang kedua tim Arsip.
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: baris TYPESERVICE %q untuk portal %q belum ada di POOLDATA.GCNM_CONNECT_REST",
			archivedokumenklaim.ErrServiceAddress, ArchiveServiceKind, portalAlias)
	}

	payload := requestBody{
		DocumentNumber: shipment.DocumentNumber(),

		// Bentuk tanggalnya `dd/mm/yyyy`, persis yang disusun activity lama dari
		// potongan teks `@substring(...)`. Ia dibentuk di Go, bukan di SQL — pemformatan
		// tanggal tidak pernah lagi dikerjakan basis data (`D-20`).
		RequestedOn: shipment.RequestedAt.Format("02/01/2006"),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: menyusun badan permintaan: %v", archivedokumenklaim.ErrServiceFailed, err)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: menyusun permintaan: %v", archivedokumenklaim.ErrServiceFailed, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := h.client.Do(request)
	if err != nil {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: %v", archivedokumenklaim.ErrServiceFailed, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	// Badan jawaban dibatasi panjangnya. Layanan yang salah konfigurasi dapat menjawab
	// dengan halaman galat sepanjang megabyte, dan membacanya seluruhnya ke memori adalah
	// cara yang mudah untuk menjatuhkan aplikasi lewat sistem orang lain.
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: membaca jawaban: %v", archivedokumenklaim.ErrServiceFailed, err)
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: layanan Arsip menjawab %d", archivedokumenklaim.ErrServiceFailed,
			response.StatusCode)
	}

	var decoded responseBody
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return archivedokumenklaim.Receipt{}, fmt.Errorf(
			"%w: jawaban layanan Arsip bukan JSON yang dikenal",
			archivedokumenklaim.ErrServiceFailed)
	}

	return archivedokumenklaim.Receipt{
		ID:      shipment.ID,
		Code:    strings.TrimSpace(decoded.Code),
		Note:    strings.TrimSpace(decoded.Message),
		Request: string(body),
		SentAt:  shipment.RequestedAt,
	}, nil
}

// HTTP wajib memenuhi seam modul.
var _ archivedokumenklaim.Gateway = (*HTTP)(nil)
