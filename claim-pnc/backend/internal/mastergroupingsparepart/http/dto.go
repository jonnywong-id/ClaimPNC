// Package mastergroupingspareparthttp adalah lapisan transport modul Master Grouping
// Sparepart.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// mastersparepart/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastergroupingspareparthttp` supaya tidak menutupi `net/http`.
package mastergroupingspareparthttp

import "claim-pnc/internal/mastergroupingsparepart"

// GroupingDTO adalah bentuk satu baris grouping yang dikirim ke peramban.
//
// Terpisah dari mastergroupingsparepart.Grouping supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`). Yang
// dipakai adalah nama yang MENCERMINKAN ISI dan mengikuti caption layar lama — kolom
// `NAMA_PANEL` menjadi `nama_panel` karena captionnya memang "Nama Panel", bukan `panjang`
// seperti nama properti Pega yang dipinjamnya.
type GroupingDTO struct {
	// ID adalah kolom ID — kunci baris ini, diterbitkan server.
	ID string `json:"id_grouping"`

	// Kelima berikut adalah identitas sparepart. Hanya yang pertama yang diketik pengguna;
	// keempat sisanya diturunkan dari Master Sparepart. Lihat catatan pada
	// mastergroupingsparepart.Grouping.
	PartNumber string `json:"nomor_sparepart"`
	PartName   string `json:"nama_sparepart"`
	CategoryID string `json:"kategori_sparepart"`
	TypeID     string `json:"tipe_sparepart"`
	PartCode   string `json:"kode_sparepart"`

	// ProductionDate adalah kolom PROD_DATE — TEKS, bukan tanggal.
	ProductionDate string `json:"tanggal_produksi"`

	// PanelID adalah kolom ID_PANEL — isian tersembunyi di balik pilihan Nama Panel.
	PanelID string `json:"id_panel"`

	// PanelName adalah kolom NAMA_PANEL — "Nama Panel".
	PanelName string `json:"nama_panel"`

	// PanelSide adalah kolom SISI_PANEL — "Sisi", berisi sandi "-", "1", atau "2".
	PanelSide string `json:"sisi"`

	// PanelSideLabel adalah sebutan sandi itu dalam bahasa yang dibaca pengguna, dihitung
	// server supaya layar tidak menyimpan salinan ketiga sandinya.
	//
	// Sandi di luar ketiganya dikirim APA ADANYA, tidak dipaksa menjadi "KANAN" seperti yang
	// dilakukan `Activity/GetSisiPanel-Act.xml`; lihat mastergroupingsparepart.SideLabel.
	PanelSideLabel string `json:"sisi_label"`

	// ChassisNumber adalah kolom NO_RANGKA — "No Rangka".
	ChassisNumber string `json:"no_rangka"`

	// VehicleType adalah kolom TIPE pada tabel pendamping — "Tipe Kendaraan".
	VehicleType string `json:"tipe_kendaraan"`

	// GroupWithChassis adalah kolom GROUPING_DGN_RANGKA — "Grouping Dengan No Rangka".
	//
	// Isinya sebuah NOMOR RANGKA, bukan nomor grup. Kosong berarti baris ini membuka grup
	// sendiri.
	GroupWithChassis string `json:"grouping_dengan_no_rangka"`

	// GroupNumber adalah kolom NO_GROUP_RANGKA — nomor grup kendaraan.
	//
	// Baca-saja di layar: ia diterbitkan penyimpanan. Ia TIDAK digambar di layar lama sama
	// sekali, dan tetap dikirim di sini supaya baris satu grup dapat ditandai — itulah satu-
	// satunya cara pengguna melihat bahwa penggabungannya berhasil.
	GroupNumber string `json:"nomor_grup"`

	// Note adalah kolom CATATAN — "Catatan".
	Note string `json:"catatan"`

	// Status adalah kolom APPROVAL: "0" menunggu, "1" disetujui, "2" ditolak.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status dalam bahasa yang dibaca pengguna.
	StatusLabel string `json:"status_label"`
}

// ListResponse adalah jawaban GET /api/master/grouping-sparepart.
type ListResponse struct {
	Grouping []GroupingDTO `json:"grouping"`

	// Status menyebut penyaring yang benar-benar dipakai, bukan yang diminta. Keduanya sama
	// pada jalur normal; menyebutkannya membuat layar dapat memastikan tab yang ditampilkan
	// memang tab yang dijawab.
	Status string `json:"status"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang milik
	// entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan hukum, "data
	// siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan, pengambilan, dan penyimpanan.
type SingleResponse struct {
	Grouping GroupingDTO `json:"grouping"`
	Portal   string      `json:"portal"`
}

// DecisionResponse adalah jawaban keputusan borongan.
type DecisionResponse struct {
	// Changed adalah jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dikirim.
	//
	// Keduanya dapat berbeda: baris yang sudah berstatus itu, atau yang sudah tidak ada, tidak
	// ikut terhitung. Menyebutkannya membuat layar dapat mengatakan "3 dari 5" alih-alih
	// melaporkan keberhasilan atas baris yang tidak tersentuh.
	Changed int `json:"jumlah_berubah"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Portal      string `json:"portal"`
}

// PanelDTO adalah satu pilihan Nama Panel.
//
// KEDUANYA dikirim: kode dipakai mencari daftar Sisi, nama disimpan sebagai NAMA_PANEL dan
// menjadi anggota kunci alami.
type PanelDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`
}

// VehicleTypeDTO adalah satu pilihan Tipe Kendaraan.
//
// `kode` dikirim untuk penelusuran tetapi TIDAK disimpan — yang tersimpan adalah namanya;
// lihat catatan pada mastergroupingsparepart.VehicleType.
type VehicleTypeDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`
}

// OptionsResponse adalah jawaban GET /api/master/grouping-sparepart/pilihan.
//
// Daftar Sisi TIDAK ada di sini: ia bergantung pada panel yang dipilih, sehingga baru dapat
// dibaca setelah pengguna memilih. Lihat SideResponse.
type OptionsResponse struct {
	Panel       []PanelDTO       `json:"panel"`
	VehicleType []VehicleTypeDTO `json:"tipe_kendaraan"`
	Portal      string           `json:"portal"`
}

// SideDTO adalah satu pilihan Sisi.
type SideDTO struct {
	// Code adalah sandi yang DISIMPAN: "-", "1", atau "2".
	Code string `json:"kode"`

	// Label adalah sebutan yang DILIHAT pengguna: "-", "KIRI", atau "KANAN".
	Label string `json:"nama"`
}

// SideResponse adalah jawaban GET /api/master/grouping-sparepart/sisi.
//
// Daftar KOSONG adalah jawaban yang sah, bukan galat: panel yang belum punya baris lokasi
// memang tidak punya sisi. Lihat usecase.Service.Sides.
type SideResponse struct {
	Side   []SideDTO `json:"sisi"`
	Portal string    `json:"portal"`
}

// PartResponse adalah jawaban GET /api/master/grouping-sparepart/sparepart.
//
// Ia padanan `Activity/SetDataSparepart-Act.xml`: kelima isian turunan yang muncul begitu
// Nomor Sparepart selesai diketik.
type PartResponse struct {
	Number         string `json:"nomor_sparepart"`
	Name           string `json:"nama_sparepart"`
	CategoryID     string `json:"kategori_sparepart"`
	TypeID         string `json:"tipe_sparepart"`
	Code           string `json:"kode_sparepart"`
	ProductionDate string `json:"tanggal_produksi"`
	Portal         string `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan DAN penyimpanan.
//
// # Kenapa satu bentuk untuk dua jalur
//
// Karena isiannya memang sama: `Section/MasterGroupingSparepartHEApproval-Section.xml`
// memakai form yang sama untuk menambah dan mengubah, dan
// `Activity/UpdateGroupingSparepartHE_act` melayani keduanya — yang membedakannya hanya
// `@IF(TempSparepart.ID!="", TempSparepart.ID, "UnknownID")`.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	id_grouping       kunci baris. Pada penambahan ia diterbitkan server; pada penyimpanan ia
//	                  diambil dari jalur URL.
//	nomor_grup        diterbitkan atau diwarisi penyimpanan; lihat resolveGroupNumber.
//	status            bukan isian melainkan akibat. Penambahan dan penyimpanan SELALU
//	                  menghasilkan status menunggu; keputusan menempuh endpoint sendiri.
//	nama_sparepart    \
//	kategori_sparepart |
//	tipe_sparepart     > kelimanya DIBACA SERVER dari Master Sparepart menurut nomornya.
//	kode_sparepart     |  Menerimanya dari klien berarti mempercayai klien soal isi master
//	tanggal_produksi  /   lain — dan klien dapat mengirim nama yang tidak berhubungan sama
//	                      sekali dengan nomornya.
type SaveRequest struct {
	PartNumber string `json:"nomor_sparepart"`

	PanelID   string `json:"id_panel"`
	PanelName string `json:"nama_panel"`
	PanelSide string `json:"sisi"`

	ChassisNumber string `json:"no_rangka"`
	VehicleType   string `json:"tipe_kendaraan"`

	GroupWithChassis string `json:"grouping_dengan_no_rangka"`

	Note string `json:"catatan"`
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// Bentuknya — DAFTAR kunci ditambah satu status — mengikuti Master Bengkel, Master Panel, dan
// Master Sparepart, sehingga keempat master alat berat diputuskan dengan cara yang sama.
//
// TANPA catatan: kedua tabel modul ini tidak punya kolom penampung alasan penolakan, dan layar
// persetujuan Pega pun tidak punya isian catatan.
type DecisionRequest struct {
	// ID adalah kunci baris yang dicentang pengguna.
	ID []string `json:"id_grouping"`

	// Status adalah keputusan yang dikehendaki: "1" approve, "2" reject.
	//
	// "0" juga diterima — ia yang mengembalikan baris ke antrean, dan `Param.Approval` pada
	// activity lama pun menerima nilai apa pun.
	Status string `json:"status"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — ditambah `detail` untuk pelanggaran per
// isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan mencocokkan teks
// `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterbengkel, masterpanel, mastersparepart,
// masterstatusprogres, dan masterautoclaim. Penyeragamannya adalah TKT-F1-004 yang masih
// terhalang; frontend sudah menampung keduanya lewat `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(g mastergroupingsparepart.Grouping) GroupingDTO {
	return GroupingDTO{
		ID:               g.ID,
		PartNumber:       g.PartNumber,
		PartName:         g.PartName,
		CategoryID:       g.CategoryID,
		TypeID:           g.TypeID,
		PartCode:         g.PartCode,
		ProductionDate:   g.ProductionDate,
		PanelID:          g.PanelID,
		PanelName:        g.PanelName,
		PanelSide:        g.PanelSide,
		PanelSideLabel:   mastergroupingsparepart.SideLabel(g.PanelSide),
		ChassisNumber:    g.ChassisNumber,
		VehicleType:      g.VehicleType,
		GroupWithChassis: g.GroupWithChassis,
		GroupNumber:      g.GroupNumber,
		Note:             g.Note,
		Status:           string(g.Status),
		StatusLabel:      g.Status.Label(),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim sebagai
// `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan satu layar
// yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []mastergroupingsparepart.Grouping) []GroupingDTO {
	result := make([]GroupingDTO, 0, len(list))
	for _, g := range list {
		result = append(result, toDTO(g))
	}
	return result
}

// toInput mengubah badan permintaan menjadi isian domain.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya keduanya tidak pernah berbeda soal isian
// mana yang diterima.
func (r SaveRequest) toInput() mastergroupingsparepart.Input {
	return mastergroupingsparepart.Input{
		PartNumber:       r.PartNumber,
		PanelID:          r.PanelID,
		PanelName:        r.PanelName,
		PanelSide:        r.PanelSide,
		ChassisNumber:    r.ChassisNumber,
		VehicleType:      r.VehicleType,
		GroupWithChassis: r.GroupWithChassis,
		Note:             r.Note,
	}
}

// toOptionsDTO menyusun kedua daftar acuan.
//
// Kedua slice selalu dibuat, tidak pernah dibiarkan nil, dengan alasan yang sama seperti
// toListDTO.
func toOptionsDTO(
	panel []mastergroupingsparepart.Panel,
	vehicle []mastergroupingsparepart.VehicleType,
	portalAlias string,
) OptionsResponse {
	panelDTO := make([]PanelDTO, 0, len(panel))
	for _, one := range panel {
		panelDTO = append(panelDTO, PanelDTO{ID: one.ID, Name: one.Name})
	}

	vehicleDTO := make([]VehicleTypeDTO, 0, len(vehicle))
	for _, one := range vehicle {
		vehicleDTO = append(vehicleDTO, VehicleTypeDTO{ID: one.ID, Name: one.Name})
	}

	return OptionsResponse{Panel: panelDTO, VehicleType: vehicleDTO, Portal: portalAlias}
}

// toSideDTO menyusun daftar pilihan Sisi beserta sebutannya.
func toSideDTO(side []mastergroupingsparepart.Side, portalAlias string) SideResponse {
	result := make([]SideDTO, 0, len(side))
	for _, one := range side {
		result = append(result, SideDTO{
			Code:  string(one),
			Label: mastergroupingsparepart.SideLabel(string(one)),
		})
	}
	return SideResponse{Side: result, Portal: portalAlias}
}

// toPartDTO menyusun kelima isian turunan.
func toPartDTO(part mastergroupingsparepart.PartRef, portalAlias string) PartResponse {
	return PartResponse{
		Number:         part.Number,
		Name:           part.Name,
		CategoryID:     part.CategoryID,
		TypeID:         part.TypeID,
		Code:           part.Code,
		ProductionDate: part.ProductionDate,
		Portal:         portalAlias,
	}
}
