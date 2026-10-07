package memory

import (
	_ "embed"

	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

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
	return sampledata.Must[[]mastergroupingsparepart.Panel](sampleJSON, "SamplePanels")
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
	return sampledata.Must[map[string][]mastergroupingsparepart.Side](sampleJSON, "SampleSides")
}

// SampleVehicleTypes adalah daftar tipe kendaraan contoh.
//
// Ia meniru `branddetail` yang `type = 'ANEKA'` dan `ACTIVESTATUS = 1`. Yang DISIMPAN adalah
// namanya, bukan ID-nya; lihat catatan pada mastergroupingsparepart.VehicleType.
func SampleVehicleTypes() []mastergroupingsparepart.VehicleType {
	return sampledata.Must[[]mastergroupingsparepart.VehicleType](sampleJSON, "SampleVehicleTypes")
}

// SampleParts adalah sparepart contoh yang dapat dirujuk nomornya.
//
// Ia meniru `POOLDATA.SPAREPART_HE` sejauh yang dibaca `RDB List/GetDataSparepart-SQL.xml` —
// enam kolom, dan TANPA penyaring APPROVAL. Nomor di luar daftar ini ditolak dengan pesan
// "Data Sparepart tidak ditemukan", persis seperti sistem lama.
func SampleParts() []mastergroupingsparepart.PartRef {
	return sampledata.Must[[]mastergroupingsparepart.PartRef](sampleJSON, "SampleParts")
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
	return sampledata.Must[[]mastergroupingsparepart.Grouping](sampleJSON, "SampleList")
}
