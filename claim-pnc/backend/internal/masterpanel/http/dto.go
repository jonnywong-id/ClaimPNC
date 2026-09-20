// Package masterpanelhttp adalah lapisan transport modul Master Panel.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// masterbengkel/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterpanelhttp` supaya tidak menutupi `net/http`.
package masterpanelhttp

import "claim-pnc/internal/masterpanel"

// PanelLocationDTO adalah satu baris lokasi pada sebuah panel.
//
// Ia dipakai DUA arah — dikirim ke peramban pada jawaban, dan diterima dari peramban pada
// penyimpanan. Satu bentuk untuk keduanya karena isinya memang sama persis: dua kolom,
// keduanya diisi pengguna, tidak satu pun diturunkan server.
type PanelLocationDTO struct {
	// Name adalah kolom LOKASI_PANEL — KIRI, KANAN, DEPAN, BELAKANG, atau LAIN-LAIN.
	Name string `json:"lokasi_panel"`

	// Side adalah kolom SISI_PANEL: "-", "1" kiri, "2" kanan.
	Side string `json:"sisi_panel"`

	// SideLabel adalah sebutan sisi yang dibaca pengguna, dihitung server supaya layar
	// tidak menyimpan salinan ketiga sandinya.
	//
	// Ia hanya diisi pada jawaban. Pada permintaan ia diabaikan — lihat toInput.
	SideLabel string `json:"sisi_label,omitempty"`
}

// PanelDTO adalah bentuk satu baris master panel yang dikirim ke peramban.
//
// Terpisah dari masterpanel.Panel supaya perubahan internal tidak bocor ke klien dan
// sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
// Yang dipakai adalah nama yang MENCERMINKAN ISI, bukan nama kolomnya apa adanya —
// `STS_EDIT_QTY` menjadi `status_edit_quantity` karena caption layarnya memang "STATUS
// EDIT QUANTITY".
type PanelDTO struct {
	// ID adalah kolom ID_PANEL — kunci baris ini, diterbitkan server.
	ID string `json:"id_panel"`

	Name string `json:"nama_panel"`

	RepairStatus        string `json:"status_repair"`
	EditQuantityStatus  string `json:"status_edit_quantity"`
	PremiumRepairStatus string `json:"status_premium_repair"`
	ShatterStatus       string `json:"status_pecah"`
	StickerStatus       string `json:"status_sticker"`

	// SideStatus adalah kolom STS_SISI pada tabel INDUK — caption "STATUS SISI".
	//
	// Ia BUKAN sisi lokasi. Nama `STS_SISI` dipakai juga sebagai alias kolom SISI_PANEL
	// pada tabel anak (`RDB List/GetLokasiSisiPanel-SQL.xml`), dan keduanya sama sekali
	// berbeda artinya. Di kontrak ini keduanya punya nama sendiri: `status_sisi` di sini,
	// `sisi_panel` di dalam setiap baris `lokasi`.
	SideStatus string `json:"status_sisi"`

	SevereDamageStatus string `json:"status_rusak_parah"`
	ActiveStatus       string `json:"status_aktif"`
	ExclusionC         string `json:"exclusion_c"`

	// ApprovalMark adalah kolom STS_APPROVAL — dan ia BUKAN kolom APPROVAL.
	//
	// Artinya tidak disebut di mana pun dalam export: tidak punya caption, tidak dipakai
	// penyaring, tidak dibandingkan di rule mana pun. Ia dikirim apa adanya supaya petugas
	// dapat melihat isinya, dan tidak punya isian di form mana pun.
	ApprovalMark string `json:"status_approval"`

	// RejectReason adalah kolom ALASAN_TOLAK, diisi pada jalur keputusan.
	RejectReason string `json:"alasan_tolak"`

	// DocumentID adalah kolom DOKUMENID.
	//
	// Ia dikirim meski layar ini tidak menyediakan unggahan lampiran: baris lama dapat
	// memilikinya, dan menyembunyikannya berarti petugas tidak punya cara mengetahui bahwa
	// lampirannya masih ada.
	DocumentID string `json:"id_dokumen"`

	// Status adalah kolom APPROVAL: "0" menunggu, "1" disetujui, "2" ditolak.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status dalam bahasa yang dibaca pengguna, dihitung server
	// supaya layar tidak menyimpan salinan ketiga sandinya.
	StatusLabel string `json:"status_label"`

	// Location adalah baris anak pada POOLDATA.LOKASI_PANEL_HE.
	//
	// SELALU berupa slice, tidak pernah null — panel tanpa lokasi terkirim sebagai `[]`.
	// Layar yang menerima `null` harus menjaganya sendiri, dan satu layar yang lupa akan
	// gagal pada panel pertama yang belum punya lokasi.
	Location []PanelLocationDTO `json:"lokasi"`
}

// ListResponse adalah jawaban GET /api/master/panel.
type ListResponse struct {
	Panel []PanelDTO `json:"panel"`

	// Status menyebut penyaring yang benar-benar dipakai, bukan yang diminta. Keduanya
	// sama pada jalur normal; menyebutkannya membuat layar dapat memastikan tab yang
	// ditampilkan memang tab yang dijawab.
	Status string `json:"status"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan hukum,
	// "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan, pengambilan, dan penyimpanan.
type SingleResponse struct {
	Panel  PanelDTO `json:"panel"`
	Portal string   `json:"portal"`
}

// DecisionResponse adalah jawaban keputusan borongan.
type DecisionResponse struct {
	// Changed adalah jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dikirim.
	//
	// Keduanya dapat berbeda: baris yang sudah berstatus itu, atau yang sudah tidak ada,
	// tidak ikut terhitung. Menyebutkannya membuat layar dapat mengatakan "3 dari 5"
	// alih-alih melaporkan keberhasilan atas baris yang tidak tersentuh.
	Changed int `json:"jumlah_berubah"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Portal      string `json:"portal"`
}

// OptionsResponse adalah jawaban GET /api/master/panel/pilihan.
//
// Isinya KONSTANTA, bukan bacaan basis data: kelima nama lokasi dan ketiga sandi sisi
// ditanam di `Activity/SetLokasiSisiPanel-Act.xml`, bukan disimpan di tabel acuan mana
// pun. Ia tetap disajikan lewat endpoint supaya layar tidak memuat salinan ketiganya —
// satu-satunya daftar pilihan modul ini yang artinya benar-benar diketahui tidak boleh
// hidup di dua tempat.
type OptionsResponse struct {
	// Location adalah kelima pilihan Lokasi Panel, pada urutan yang sama dengan Pega.
	Location []string `json:"lokasi_panel"`

	// Side adalah ketiga pilihan Sisi Panel beserta sebutannya.
	Side []SideOptionDTO `json:"sisi_panel"`
}

// SideOptionDTO adalah satu pilihan Sisi Panel.
type SideOptionDTO struct {
	Value string `json:"nilai"`
	Label string `json:"label"`
}

// SaveRequest adalah badan permintaan penambahan DAN penyimpanan.
//
// # Kenapa satu bentuk untuk dua jalur
//
// Karena isiannya memang sama: `Section/BrowsePanelHEApproval-Section.xml` memakai form
// yang sama untuk menambah dan mengubah, dan `Activity/CNMUpdatePanelHE_act` melayani
// keduanya — yang membedakannya hanya `InputData.ID_PANEL == "UnknownID"`.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	id_panel         kunci baris. Pada penambahan ia diterbitkan server dari kode situs
//	                 dan sequence; pada penyimpanan ia diambil dari jalur URL. Menerimanya
//	                 dari badan permintaan berarti dua sumber untuk satu nilai.
//	status           bukan isian melainkan akibat. Penambahan dan penyimpanan SELALU
//	                 menghasilkan status menunggu; keputusan menempuh endpoint tersendiri.
//	status_approval  tidak punya caption di layar mana pun. Nilainya dipertahankan dari
//	                 baris yang tersimpan.
//	alasan_tolak     diisi pada jalur keputusan, bukan pada jalur simpan.
//	id_dokumen       hasil unggah lampiran, jalur yang tidak dibawa modul ini.
type SaveRequest struct {
	Name string `json:"nama_panel"`

	RepairStatus        string `json:"status_repair"`
	EditQuantityStatus  string `json:"status_edit_quantity"`
	PremiumRepairStatus string `json:"status_premium_repair"`
	ShatterStatus       string `json:"status_pecah"`
	StickerStatus       string `json:"status_sticker"`
	SideStatus          string `json:"status_sisi"`
	SevereDamageStatus  string `json:"status_rusak_parah"`
	ActiveStatus        string `json:"status_aktif"`
	ExclusionC          string `json:"exclusion_c"`

	// Location adalah SELURUH baris lokasi yang dikehendaki, bukan hanya yang berubah.
	//
	// Bentuknya sengaja begitu: layar lama mengirim seluruh halaman klipboard sebagai satu
	// dokumen JSON (`@GCNM.GetPageJSONString()`), sehingga daftar yang dikirim memang
	// selalu daftar yang utuh. Baris yang dibuang pengguna cukup tidak ikut dikirim.
	//
	// Nilai nil dan `[]` keduanya berarti panel tanpa lokasi, dan keduanya SAH.
	Location []PanelLocationDTO `json:"lokasi"`
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// Bentuknya — DAFTAR kunci ditambah satu status dan satu catatan — mengikuti sistem lama:
// `Activity/SetApprovalAllMaster` menelusuri baris yang dicentang lalu menetapkan status
// yang sama pada seluruhnya, dan layarnya menyediakan satu isian "Catatan"
// (`TempStsClaim.pyNote`) untuk seluruh pilihan. Satu permintaan per baris akan mengubah
// operasi yang di Pega berupa satu tindakan menjadi sederet tindakan yang dapat gagal
// separuh jalan.
type DecisionRequest struct {
	// ID adalah kunci baris yang dicentang pengguna.
	ID []string `json:"id_panel"`

	// Status adalah keputusan yang dikehendaki: "1" approve, "2" reject.
	//
	// "0" juga diterima — ia yang mengembalikan baris ke antrean, dan
	// `SetApprovalAllMaster` pun menerima nilai apa pun lewat `Param.approval`.
	Status string `json:"status"`

	// Reason adalah isi kolom ALASAN_TOLAK, dan ia hanya tersimpan pada keputusan TOLAK.
	//
	// Menuliskannya pada persetujuan akan mengisi kolom bernama "alasan tolak" pada baris
	// yang justru disetujui; lihat usecase.Service.Decide.
	Reason string `json:"catatan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — ditambah `detail` untuk pelanggaran
// per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan mencocokkan
// teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterbengkel, masterstatusprogres, dan
// masterautoclaim. Ketiga modul master belum sepakat menamainya — masterstatus memakai
// `field` — dan penyeragamannya adalah TKT-F1-004 yang masih terhalang. Frontend sudah
// menampung keduanya lewat `APIError.violations()`.
//
// Pelanggaran pada baris lokasi memakai bentuk `lokasi.<indeks>.<isian>`, sehingga layar
// dapat menyorot baris yang tepat pada daftar yang panjangnya berubah-ubah.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(p masterpanel.Panel) PanelDTO {
	return PanelDTO{
		ID:                  p.ID,
		Name:                p.Name,
		RepairStatus:        p.RepairStatus,
		EditQuantityStatus:  p.EditQuantityStatus,
		PremiumRepairStatus: p.PremiumRepairStatus,
		ShatterStatus:       p.ShatterStatus,
		StickerStatus:       p.StickerStatus,
		SideStatus:          p.SideStatus,
		SevereDamageStatus:  p.SevereDamageStatus,
		ActiveStatus:        p.ActiveStatus,
		ExclusionC:          p.ExclusionC,
		ApprovalMark:        p.ApprovalMark,
		RejectReason:        p.RejectReason,
		DocumentID:          p.DocumentID,
		Status:              string(p.Status),
		StatusLabel:         p.Status.Label(),
		Location:            toLocationListDTO(p.Location),
	}
}

// toLocationListDTO mengubah daftar lokasi domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya panel tanpa lokasi terkirim
// sebagai `[]` dan bukan `null`.
func toLocationListDTO(list []masterpanel.PanelLocation) []PanelLocationDTO {
	result := make([]PanelLocationDTO, 0, len(list))
	for _, one := range list {
		result = append(result, PanelLocationDTO{
			Name:      one.Name,
			Side:      string(one.Side),
			SideLabel: one.Side.Label(),
		})
	}
	return result
}

// toInput mengubah badan permintaan menjadi isian domain.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya keduanya tidak pernah berbeda soal
// isian mana yang diterima.
//
// `sisi_label` pada setiap baris lokasi DIABAIKAN meski ikut terkirim: ia dihitung server
// dari `sisi_panel`, dan menerimanya kembali berarti klien dapat mengirim label yang tidak
// cocok dengan sandinya.
func (r SaveRequest) toInput() masterpanel.Input {
	location := make([]masterpanel.PanelLocation, 0, len(r.Location))
	for _, one := range r.Location {
		location = append(location, masterpanel.PanelLocation{
			Name: one.Name,
			Side: masterpanel.Side(one.Side),
		})
	}

	return masterpanel.Input{
		Name:                r.Name,
		RepairStatus:        r.RepairStatus,
		EditQuantityStatus:  r.EditQuantityStatus,
		PremiumRepairStatus: r.PremiumRepairStatus,
		ShatterStatus:       r.ShatterStatus,
		StickerStatus:       r.StickerStatus,
		SideStatus:          r.SideStatus,
		SevereDamageStatus:  r.SevereDamageStatus,
		ActiveStatus:        r.ActiveStatus,
		ExclusionC:          r.ExclusionC,
		Location:            location,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan
// satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []masterpanel.Panel) []PanelDTO {
	result := make([]PanelDTO, 0, len(list))
	for _, p := range list {
		result = append(result, toDTO(p))
	}
	return result
}

// optionsDTO menyusun daftar pilihan Lokasi dan Sisi.
//
// Keduanya konstanta domain, bukan bacaan basis data; lihat OptionsResponse.
func optionsDTO() OptionsResponse {
	location := make([]string, len(masterpanel.LocationOptions))
	copy(location, masterpanel.LocationOptions)

	side := make([]SideOptionDTO, 0, 3)
	for _, one := range []masterpanel.Side{
		masterpanel.SideNone,
		masterpanel.SideLeft,
		masterpanel.SideRight,
	} {
		side = append(side, SideOptionDTO{Value: string(one), Label: one.Label()})
	}

	return OptionsResponse{Location: location, Side: side}
}
