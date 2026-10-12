package masterpanel

import "strings"

// Penerjemahan nilai berkas CSV — dibaca dari `Activity/PNCUploadMasterPanel_Act-Act.xml`
// dan `Activity/PNCUploadLokasiSisiPanel_Act-Act.xml`, yang **ada di export** meski kedua
// Flow Action pemanggilnya tidak.
//
// # Berkas CSV berisi KATA, bukan sandi
//
// Ini yang paling mudah salah, dan sempat salah di sini. Kolom status di berkas berisi
// "TIDAK"/"YA" dan "GANTI"/"JASA" — bukan "0"/"1"/"2". Pega menerjemahkannya saat membaca:
//
//	@IF(TempPanel.STS_PECAH="TIDAK","0","1")
//	@IF(TempPanel.STS_REPAIR="GANTI","1",(@IF(TempPanel.STS_REPAIR="JASA","2","3")))
//
// Meneruskan nilai berkas apa adanya akan menyimpan teks "TIDAK" ke kolom yang seharusnya
// berisi "0" — **tanpa satu pun galat**, dan baru terlihat saat layar menampilkan sandi
// yang tidak dikenal.
//
// # Perhatikan: "selain TIDAK" menjadi "1", bukan hanya "YA"
//
// Ekspresi Pega-nya `@IF(x="TIDAK","0","1")` — jadi sel kosong, salah ketik, maupun "YA"
// seluruhnya menjadi "1". Ditiru apa adanya (`P-5`): mengetatkannya akan menolak berkas
// yang hari ini diterima Pega.
//
// Hal yang sama pada `STS_REPAIR`: apa pun selain "GANTI" dan "JASA" menjadi "3".

// csvWordToFlag menerjemahkan kolom dua keadaan.
//
// Padanan `@IF(x="TIDAK","0","1")` — perbandingannya tidak peka huruf besar-kecil dan
// spasi, karena berkas yang disusun tangan kerap memuat " Tidak ".
func csvWordToFlag(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "TIDAK") {
		return "0"
	}
	return "1"
}

// csvWordToRepair menerjemahkan STS_REPAIR.
//
// Padanan `@IF(x="GANTI","1",(@IF(x="JASA","2","3")))`.
func csvWordToRepair(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GANTI":
		return "1"
	case "JASA":
		return "2"
	}
	return "3"
}

// csvWordToSide menerjemahkan STS_SISI pada berkas LOKASI.
//
// Padanan `@If(x=="KIRI","1",(@if(x=="KANAN","2","-")))`. Perhatikan ia berbeda dari
// `STS_SISI` pada berkas MASTER, yang merupakan kolom dua keadaan "TIDAK"/lainnya — satu
// nama kolom, dua arti, pada dua berkas yang berbeda.
func csvWordToSide(value string) Side {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "KIRI":
		return SideLeft
	case "KANAN":
		return SideRight
	}
	return SideNone
}

// ApplyCSVWords menerjemahkan satu baris berkas master menjadi nilai yang disimpan.
//
// Dipisahkan dari pembacaan berkas supaya penerjemahannya dapat diuji sendiri, dan supaya
// satu-satunya tempat yang mengetahui kosakata berkas adalah berkas ini.
//
// `STS_AKTIF` TIDAK diterjemahkan — ia tidak dibaca dari berkas sama sekali. Lihat
// ActiveStatusOnUpload.
func ApplyCSVWords(in Input) Input {
	in.Name = strings.ToUpper(strings.TrimSpace(in.Name))

	in.RepairStatus = csvWordToRepair(in.RepairStatus)
	in.EditQuantityStatus = csvWordToFlag(in.EditQuantityStatus)
	in.PremiumRepairStatus = csvWordToFlag(in.PremiumRepairStatus)
	in.ShatterStatus = csvWordToFlag(in.ShatterStatus)
	in.StickerStatus = csvWordToFlag(in.StickerStatus)
	in.SideStatus = csvWordToFlag(in.SideStatus)
	in.SevereDamageStatus = csvWordToFlag(in.SevereDamageStatus)
	in.ExclusionC = csvWordToFlag(in.ExclusionC)

	in.ActiveStatus = ActiveStatusOnUpload
	return in
}

// ActiveStatusOnUpload adalah nilai STS_AKTIF yang DIPAKSA pada setiap baris unggahan.
//
// Pega menyetelnya `"1"` tanpa membaca berkas — `TempPanel.STS_AKTIF := "1"`, tanpa satu
// pun `@IF`. Kolomnya boleh ada di berkas; isinya diabaikan.
//
// Akibatnya satu baris CSV **tidak dapat menonaktifkan** panel. Itu perilaku sistem lama,
// ditiru apa adanya, dan dinyatakan di sini supaya tidak terbaca sebagai kelalaian.
const ActiveStatusOnUpload = "1"
