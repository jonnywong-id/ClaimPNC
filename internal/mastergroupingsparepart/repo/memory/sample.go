package memory

import "claim-pnc/internal/mastergroupingsparepart"

// SampleSequence adalah nomor ID terakhir yang sudah dipakai contoh di bawah.
//
// Penambahan berikutnya karena itu menerbitkan "5" — melanjutkan deret, bukan menimpa baris
// yang sudah ada.
//
// Bentuknya angka polos, TANPA kode situs dan TANPA pengisian nol, persis seperti yang
// diterbitkan `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:11`. Itu berbeda dari ketiga master
// alat berat lain, dan perbedaannya memang harus terlihat saat aplikasi dicoba tanpa Oracle.
const SampleSequence int64 = 4

// SamplePanels adalah daftar panel contoh.
//
// Ia meniru `POOLDATA.PANEL_HE` yang SUDAH DISETUJUI — autocomplete Nama Panel menyaring
// `APPROVAL = "1"`, sehingga panel yang masih menunggu tidak diwakili di sini sama sekali.
func SamplePanels() []mastergroupingsparepart.Panel {
	return []mastergroupingsparepart.Panel{
		{ID: "PNL01", Name: "BUMPER DEPAN"},
		{ID: "PNL02", Name: "KABIN"},
		{ID: "PNL03", Name: "BUCKET"},
	}
}

// SampleSides adalah pilihan Sisi contoh per panel.
//
// Kuncinya ID_PANEL saja — bukan pasangan ID_PANEL dan NAMA seperti kueri aslinya — karena di
// repo contoh keduanya selalu sepadan. Yang ditiru adalah PERILAKUNYA: panel yang berbeda
// memberi pilihan sisi yang berbeda.
//
// Sengaja tidak merata. BUMPER DEPAN punya kiri dan kanan, KABIN punya ketiganya, dan BUCKET
// hanya "-". Panel yang seluruhnya punya pilihan yang sama tidak membuktikan apa pun tentang
// penyaringan menurut panel.
func SampleSides() map[string][]mastergroupingsparepart.Side {
	return map[string][]mastergroupingsparepart.Side{
		"PNL01": {mastergroupingsparepart.SideLeft, mastergroupingsparepart.SideRight},
		"PNL02": {
			mastergroupingsparepart.SideNone,
			mastergroupingsparepart.SideLeft,
			mastergroupingsparepart.SideRight,
		},
		"PNL03": {mastergroupingsparepart.SideNone},
	}
}

// SampleVehicleTypes adalah daftar tipe kendaraan contoh.
//
// Ia meniru `branddetail` yang `type = 'ANEKA'` dan `ACTIVESTATUS = 1`. Yang DISIMPAN adalah
// namanya, bukan ID-nya; lihat catatan pada mastergroupingsparepart.VehicleType.
func SampleVehicleTypes() []mastergroupingsparepart.VehicleType {
	return []mastergroupingsparepart.VehicleType{
		{ID: "TY01", Name: "EXCAVATOR"},
		{ID: "TY02", Name: "BULLDOZER"},
		{ID: "TY03", Name: "WHEEL LOADER"},
	}
}

// SampleParts adalah sparepart contoh yang dapat dirujuk nomornya.
//
// Ia meniru `POOLDATA.SPAREPART_HE` sejauh yang dibaca `RDB List/GetDataSparepart-SQL.xml` —
// enam kolom, dan TANPA penyaring APPROVAL. Nomor di luar daftar ini ditolak dengan pesan
// "Data Sparepart tidak ditemukan", persis seperti sistem lama.
func SampleParts() []mastergroupingsparepart.PartRef {
	return []mastergroupingsparepart.PartRef{
		{
			Number:         "SP-1001",
			Name:           "FILTER OLI",
			CategoryID:     "KAT01",
			TypeID:         "TIP01",
			Code:           "KD-1001",
			ProductionDate: "12/05/2024",
		},
		{
			Number:         "SP-1002",
			Name:           "SEAL KIT BOOM",
			CategoryID:     "KAT02",
			TypeID:         "TIP03",
			Code:           "KD-1002",
			ProductionDate: "03/01/2025",
		},
		{
			Number:         "SP-1003",
			Name:           "TRACK LINK",
			CategoryID:     "KAT03",
			TypeID:         "TIP05",
			Code:           "KD-1003",
			ProductionDate: "",
		},
	}
}

// SampleList adalah daftar grouping contoh untuk pengembangan tanpa Oracle.
//
// Isinya dipilih supaya seluruh alur layar dapat dicoba, termasuk keadaan yang paling mudah
// terlupa diuji:
//
//	"1" dan "2"  DUA baris satu grup — nomor rangka yang sama, panel yang berbeda.
//	             Inilah bentuk grouping yang sebenarnya, dan tanpa dua baris seperti ini
//	             layar tidak pernah memperlihatkan apa gunanya modul ini.
//	"2"          dibuat dengan MENGIKUTI grup baris "1" — GroupWithChassis terisi, dan nomor
//	             grupnya sama persis dengan milik "1".
//	"3"          grup tersendiri, dan berstatus MENUNGGU — supaya tab Waiting Approval dan
//	             tombol keputusan tidak pernah kosong saat dicoba.
//	"4"          berstatus DITOLAK, dan catatannya kosong — dua keadaan sah yang paling mudah
//	             terlupa.
func SampleList() []mastergroupingsparepart.Grouping {
	return []mastergroupingsparepart.Grouping{
		{
			ID:               "1",
			PartNumber:       "SP-1001",
			PartName:         "FILTER OLI",
			PartCode:         "KD-1001",
			CategoryID:       "KAT01",
			TypeID:           "TIP01",
			ProductionDate:   "12/05/2024",
			PanelID:          "PNL02",
			PanelName:        "KABIN",
			PanelSide:        string(mastergroupingsparepart.SideLeft),
			ChassisNumber:    "MHFXW1234K5678901",
			VehicleType:      "EXCAVATOR",
			GroupWithChassis: "",
			GroupNumber:      mastergroupingsparepart.ComposeGroupNumber(1),
			Note:             "Pemasangan awal unit.",
			Status:           mastergroupingsparepart.StatusApproved,
		},
		{
			ID:               "2",
			PartNumber:       "SP-1002",
			PartName:         "SEAL KIT BOOM",
			PartCode:         "KD-1002",
			CategoryID:       "KAT02",
			TypeID:           "TIP03",
			ProductionDate:   "03/01/2025",
			PanelID:          "PNL01",
			PanelName:        "BUMPER DEPAN",
			PanelSide:        string(mastergroupingsparepart.SideRight),
			ChassisNumber:    "MHFXW1234K5678901",
			VehicleType:      "EXCAVATOR",
			GroupWithChassis: "MHFXW1234K5678901",
			GroupNumber:      mastergroupingsparepart.ComposeGroupNumber(1),
			Note:             "",
			Status:           mastergroupingsparepart.StatusApproved,
		},
		{
			ID:               "3",
			PartNumber:       "SP-1003",
			PartName:         "TRACK LINK",
			PartCode:         "KD-1003",
			CategoryID:       "KAT03",
			TypeID:           "TIP05",
			ProductionDate:   "",
			PanelID:          "PNL03",
			PanelName:        "BUCKET",
			PanelSide:        string(mastergroupingsparepart.SideNone),
			ChassisNumber:    "MHFZZ9876K1234567",
			VehicleType:      "BULLDOZER",
			GroupWithChassis: "",
			GroupNumber:      mastergroupingsparepart.ComposeGroupNumber(2),
			Note:             "Menunggu konfirmasi teknik.",
			Status:           mastergroupingsparepart.StatusPending,
		},
		{
			ID:               "4",
			PartNumber:       "SP-1001",
			PartName:         "FILTER OLI",
			PartCode:         "KD-1001",
			CategoryID:       "KAT01",
			TypeID:           "TIP01",
			ProductionDate:   "12/05/2024",
			PanelID:          "PNL01",
			PanelName:        "BUMPER DEPAN",
			PanelSide:        string(mastergroupingsparepart.SideLeft),
			ChassisNumber:    "MHFQQ5555K7654321",
			VehicleType:      "WHEEL LOADER",
			GroupWithChassis: "",
			GroupNumber:      mastergroupingsparepart.ComposeGroupNumber(3),
			Note:             "",
			Status:           mastergroupingsparepart.StatusRejected,
		},
	}
}
