package memory

import (
	_ "embed"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleDetails adalah akseptasi contoh untuk pengembangan lokal dan pengujian.
//
// # Seluruh isinya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor polis, nama tertanggung, atau nama Ceding Co di bawah berasal dari
// data nyata. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan
// itu berlaku penuh pada data contoh — berkas contoh justru yang paling mudah tersalin ke
// tempat lain.
//
// # Yang TIDAK dikarang adalah bentuknya
//
// Isian dan grid di bawah dirakit DARI KATALOG (`section.go`), bukan diketik satu per satu.
// Akibatnya contoh ini tidak dapat tertinggal saat katalognya berubah: isian baru langsung
// ikut terisi, dan isian yang dihapus langsung hilang dari sini.
//
// Nilai yang tidak disebut `sampleValues` diisi penanda `—` supaya layar tetap menggambar
// seluruh isiannya. Sel kosong dan sel yang memang belum diisi tidak dapat dibedakan di layar,
// dan pada layar contoh perbedaan itu tidak penting.
//
// # Nomor klaimnya SAMA dengan contoh Inbox Claim Treaty Non Prop
//
// Itu bukan kebetulan. Satu-satunya pintu ke layar ini adalah nomor klaim di antrean, dan
// contoh yang nomornya tidak cocok membuat setiap tautan di layar antrean berakhir "klaim
// tidak ditemukan" saat `PENYIMPANAN=memori`. Keempatnya diambil dari
// `inboxclaimtreatynonprop/repo/memory/sample.go`:
//
//	CLMNP-1001  antrean Admin — akseptasi yang isinya lengkap
//	CLMNP-1002  antrean Admin — klaim yang ADA tetapi dokumennya kosong; ia membuktikan
//	            layar tetap terbuka dengan nomor klaim terbaca, bukan dijawab
//	            "tidak ditemukan"
//	CLMNP-2001  antrean Teknik — lengkap, supaya tab Teknik ikut dapat ditelusuri
//	CLMNP-2002  antrean Teknik — lengkap
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Galat di sini berarti katalog dan perakit contoh berselisih — cacat
// pemrograman yang harus terlihat saat aplikasi start, bukan saat pengguna
// membuka layar.
// Klaim yang dokumennya kosong. Ia TIDAK boleh dijawab "tidak ditemukan" — lihat
// catatan LEFT JOIN pada kuerinya.
func SampleDetails() []inputacceptation.Detail {
	return sampledata.Must[[]inputacceptation.Detail](sampleJSON, "SampleDetails")
}

// sampleValues adalah nilai contoh untuk isian yang layak terbaca sebagai kalimat.
//
// Isian yang tidak disebut di sini tetap digambar, dengan penanda `—`.
var sampleValues = sampledata.Must[map[string]string](sampleJSON, "sampleValues")

// filledValues merakit seluruh isian yang tidak terhalang dari katalog.
//
// Nomor klaimnya diterima sebagai parameter, bukan diambil dari sampleValues: isian "Claim No"
// wajib sama dengan nomor klaim yang dibuka, dan contoh yang menampilkan nomor berbeda dari
// alamatnya adalah contoh yang menyesatkan.
func filledValues(claimID string) map[string]string {
	values := map[string]string{"claim_no": claimID}
	for _, field := range inputacceptation.Fields() {
		if field.Key == "claim_no" {
			continue
		}
		if field.Blocked {
			// Isian terhalang TIDAK diisi contoh. Mengisinya akan membuat layar tampak
			// berfungsi di pengembangan lalu kosong di staging — persis kesalahpahaman yang
			// penanda terhalang ada untuk mencegahnya.
			continue
		}
		if value, listed := sampleValues[field.Key]; listed {
			values[field.Key] = value
			continue
		}
		values[field.Key] = "—"
	}
	return values
}

// sampleRows adalah baris contoh tiap grid, dikunci kode grid.
//
// Nilainya disebut per KOLOM supaya pembacanya dapat mencocokkannya dengan katalog; kolom yang
// tidak disebut diisi penanda.
var sampleRows = sampledata.Must[map[string][]map[string]string](sampleJSON, "sampleRows")

// filledGrids merakit seluruh grid dari katalog.
//
// Grid yang tidak punya baris contoh tetap ADA dengan nol baris — bukan dihilangkan. Bedanya
// bermakna: grid yang ada tetapi kosong berarti "tidak ada isinya", grid yang tidak ada sama
// sekali berarti "belum dapat dibaca".
func filledGrids() map[string][]inputacceptation.GridRow {
	return sampledata.Must[map[string][]inputacceptation.GridRow](sampleJSON, "filledGrids")
}
