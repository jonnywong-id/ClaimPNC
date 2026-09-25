package inboxkomunikasicabang

import (
	"strconv"
	"strings"
	"time"
)

// Berkas ini memuat tindakan "Kirim Pesan" — pembuatan percakapan BARU.
//
// # Kenapa ia berkas tersendiri
//
// Karena ia satu-satunya tindakan di modul ini yang MELAHIRKAN percakapan, bukan mengubah
// yang sudah ada. Seluruh tipe lain di modul ini berangkat dari sebuah nomor percakapan yang
// sudah ada; yang di sini justru menghasilkannya.
//
// # Asalnya di sistem lama
//
// Bukan section tersendiri, melainkan BLOK TERSEMBUNYI di dalam
// `Section/InboxKomunikasi-Section.xml` — kontainer `S11` yang bersyarat
// `pyContainerVisibleWhen = IsUpdateKomunikasi`.
//
// Rantai syaratnya dua tingkat, dan tingkat keduanya yang bermakna:
//
//	IsUpdateKomunikasi  ->  evaluateWhen("IsInsertKomunikasi")
//	IsInsertKomunikasi  ->  TempInputKomunikasi.pyLabel = "Insert"
//
// Penandanya diset `"Insert"` oleh `CNMShowInsertKomunikasi_dt` saat tombol "Tambah"
// ditekan, dan dikosongkan kembali pada langkah terakhir `PNCSendMessageKomunikasiCabang`
// setelah pesannya terkirim — itulah yang menutup formnya sendiri.
//
// Mekanisme itu TIDAK dibawa: ia cara Pega menyimpan keadaan layar di server, dan di
// antarmuka baru keadaan layar tinggal di layar. Yang dibawa adalah PERILAKUNYA — form
// tertutup saat dibuka, terbuka saat "Tambah" ditekan, tertutup lagi setelah terkirim.

// Destination adalah tujuan sebuah pesan baru — isian dropdown pertama pada form.
//
// # Dua nilainya dari mana
//
// `CNMShowInsertKomunikasi_dt` mengisi daftar pilihannya secara harfiah, dua baris:
//
//	TempInputKomunikasi2.pxResults(<APPEND>).City := "PUSAT"
//	TempInputKomunikasi2.pxResults(<APPEND>).City := "CABANG"
//
// Jadi pilihannya memang hanya dua, dan bukan diambil dari master mana pun. Keduanya
// dibawa apa adanya sebagai konstanta (`P-5`) — bukan dijadikan master data, karena ia
// bukan daftar yang dapat bertambah melainkan dua arah percakapan yang mungkin.
type Destination string

const (
	// DestinationHeadOffice — pesan ditujukan ke KANTOR PUSAT.
	DestinationHeadOffice Destination = "PUSAT"

	// DestinationBranch — pesan ditujukan ke SEBUAH CABANG, yang harus dipilih.
	DestinationBranch Destination = "CABANG"
)

// Destinations mengembalikan kedua pilihan, berurutan seperti di layar lama.
func Destinations() []Destination {
	return []Destination{DestinationHeadOffice, DestinationBranch}
}

// Valid menyatakan tujuannya dikenali.
func (d Destination) Valid() bool {
	return d == DestinationHeadOffice || d == DestinationBranch
}

// BranchOption adalah satu baris pada pemilih cabang — isian kedua pada form.
//
// # Dari mana daftarnya
//
// Autocomplete pada section membaca `pySourceName = TempResultSurveyor.pxResults`, dan
// harness mendeklarasikan halaman itu berkelas `ASM-FW-GCNMFW-Int-V_D_SURVEYORS` — yakni
// view `POOLDATA.V_D_SURVEYORS`. Ketiga isian yang dipakainya terbaca dari section:
//
//	.BRANCHNAME   yang DITAMPILKAN di kotak pencarian
//	.BRANCH       kode cabang  -> TempInputKomunikasi.DistrictID
//	.EMAIL        alamat surel -> TempInputKomunikasi.District
//
// # Yang TIDAK terbaca, dan harus dinyatakan
//
// Kueri yang benar-benar MENGISI halaman itu tidak ada di export mana pun — tidak di
// section, tidak di harness, tidak di kedua activity yang menyebut halaman itu. Yang
// diketahui hanyalah KELASNYA.
//
// Akibatnya penyaring dan urutannya ditebak: modul ini mengambil seluruh baris yang punya
// kode dan nama, diurutkan menurut nama. Bila daftar yang tampil berbeda dari Pega, inilah
// tempat pertama yang harus diperiksa — dan itu dinyatakan lewat PlannedDifferences.
type BranchOption struct {
	// Code adalah kode cabang — yang tersimpan sebagai COMMUNICATE_TO.
	//
	// Ia yang MENENTUKAN siapa melihat percakapannya, bukan sekadar keterangan. Kode yang
	// salah menaruh percakapan di kotak masuk cabang yang keliru, dan tidak ada satu pun
	// galat yang muncul karenanya (`R-20`).
	Code string

	// Name adalah nama cabang yang dibaca pengguna saat memilih.
	Name string

	// Email adalah alamat surel cabang tujuan.
	//
	// Ia DIBAWA tetapi belum dipakai: pengiriman surelnya belum dibangun. Lihat catatan
	// pada NewMessageCommand.
	Email string
}

// NewMessageInput adalah isian mentah form "Kirim Pesan".
type NewMessageInput struct {
	// Destination adalah pilihan dropdown — "PUSAT" atau "CABANG".
	Destination string

	// BranchCode adalah kode cabang tujuan, diisi HANYA bila Destination "CABANG".
	BranchCode string

	// Message adalah isi pesan — `TempInputKomunikasi.City` pada section.
	//
	// Perhatikan namanya di Pega: sebuah properti bernama "City" menyimpan ISI PESAN. Ia
	// tidak dibawa (`D-19`).
	Message string
}

// NewMessageCommand adalah pesan baru yang sudah tervalidasi.
//
// # Yang TIDAK ada di sini, dan sengaja
//
// Pengirimnya tidak menyimpan kode cabang. Cabang asal pesan diturunkan ULANG dari login
// pengirim lewat BranchResolver saat perintah ini dijalankan — bukan dibawa dari layar.
// Membawanya dari layar berarti pengirim dapat menyatakan dirinya berasal dari cabang mana
// pun, dan kolom COMMUNICATE_FROM menentukan siapa melihat percakapannya (`R-20`).
//
// # Surel yang TIDAK dikirim
//
// Sistem lama menutup `PNCSendMessageKomunikasiCabang` dengan `Property-Set-HTML` lalu
// `Call SendSimpleEmail`, mengirim "Notifikasi Komunikasi Cabang Baru" ke alamat cabang
// tujuan. Itu TIDAK dibangun di sini, dan dua hal membuatnya bukan sekadar penundaan:
//
//   - Modul ini belum punya seam Notifier, dan penerima notifikasi WAJIB berasal dari master
//     Penerima Notifikasi berupa mailbox fungsional (`D-67`).
//   - Activity lama memuat satu alamat surel PRIBADI yang ter-hardcode di jalur produksi.
//     Ia tidak boleh dibawa apa pun yang terjadi (`D-15`, `D-67`).
//
// Yang hilang karenanya nyata dan dinyatakan: penerima tidak diberi tahu lewat surel.
// Pesannya tetap muncul di kotak masuknya, dan itulah fungsi intinya.
type NewMessageCommand struct {
	// Destination menentukan ke mana pesannya ditujukan.
	Destination Destination

	// BranchCode adalah kode cabang tujuan. Kosong bila Destination "PUSAT".
	BranchCode string

	// Message adalah isi pesannya.
	Message string

	// Sender adalah pengirimnya — login dan namanya.
	Sender Caller

	// CreatedAt adalah waktu pesannya dibuat.
	//
	// Ia TIDAK dipakai INSERT-nya. `InsertMessageCABANG_PNC` tidak menyebut kolom
	// CREATEDDATE sama sekali, sehingga tanggalnya diisi basis data — lewat default kolom
	// atau trigger yang tidak terbaca dari export (`R-08`).
	//
	// Ia dibawa tetap, untuk jejak: tanpa waktu di sisi aplikasi, pesan yang gagal
	// tersimpan tidak dapat dicocokkan dengan barisnya di log.
	CreatedAt time.Time
}

// RecipientCode mengembalikan nilai kolom COMMUNICATE_TO.
//
// Kantor pusat ditandai HeadOfficeCode (`"1"`), bukan HeadOfficeBranch (`"100081"`) — sama
// persis seperti pada penyaring daftar. Menukar keduanya menaruh percakapan di tempat yang
// tidak dibaca siapa pun, tanpa satu pun galat.
func (c NewMessageCommand) RecipientCode() string {
	if c.Destination == DestinationBranch {
		return c.BranchCode
	}
	return HeadOfficeCode
}

// maxMessageLength membatasi panjang pesan baru.
//
// Angkanya SAMA dengan batas balasan, dan itu disengaja: keduanya mengisi kolom yang sama
// (`MESSAGE` dan `REPLYMESSAGE` pada tabel yang sama), sehingga dua batas yang berbeda akan
// menolak kalimat yang sama pada satu layar dan menerimanya pada layar lain.
//
// Sama seperti batas balasan, ia BUKAN tebakan atas lebar kolomnya — DDL-nya belum ada
// (`R-08`) — melainkan penjaga terhadap kiriman yang jelas tidak masuk akal.
const maxMessageLength = maxReplyLength

// NewMessageCommandOf membentuk pesan baru yang sah, atau menyatakan apa yang salah.
//
// # Satu pesan galat dibawa APA ADANYA dari sistem lama
//
// `PNCSendMessageKomunikasiCabang` menyetel `Local.msgErr = "Silakan pilih cabang terlebih
// dahulu"` lalu `Page-Set-Messages`. Kalimat itu dipakai persis, karena pengguna layar ini
// sudah mengenalnya (`D-13`) — menggantinya dengan kalimat yang "lebih baik" hanya menambah
// satu hal yang harus dipelajari ulang tanpa menambah kejelasan.
func NewMessageCommandOf(
	input NewMessageInput, sender Caller, now time.Time,
) (NewMessageCommand, error) {
	cleanSender := sender.Clean()
	if cleanSender.Login == "" || cleanSender.Name == "" {
		return NewMessageCommand{}, ErrCallerUnknown
	}

	destination := Destination(strings.TrimSpace(input.Destination))
	branch := strings.TrimSpace(input.BranchCode)
	message := strings.TrimSpace(input.Message)

	violations := []Violation{}

	if !destination.Valid() {
		violations = append(violations, Violation{
			Field:   FieldDestination,
			Message: "Tujuan harus dipilih — PUSAT atau CABANG.",
		})
	}

	// Cabang WAJIB bila tujuannya cabang. Isian ini bertanda `pyRequired` pada section, dan
	// activity memeriksanya sekali lagi sebelum menyimpan — dua pemeriksaan untuk satu
	// aturan, dan yang menentukan adalah yang di peladen.
	if destination == DestinationBranch && branch == "" {
		violations = append(violations, Violation{
			Field:   FieldBranch,
			Message: "Silakan pilih cabang terlebih dahulu",
		})
	}

	// Cabang yang diisi pada tujuan PUSAT DIABAIKAN, bukan ditolak.
	//
	// Layar mengosongkan pemilih cabang saat tujuannya berpindah ke PUSAT; menolak
	// permintaannya akan menghukum pengguna atas isian yang sudah tidak terlihat olehnya.
	if destination == DestinationHeadOffice {
		branch = ""
	}

	if message == "" {
		violations = append(violations, Violation{
			Field:   FieldMessageBody,
			Message: "Pesan tidak boleh kosong.",
		})
	}

	// Dihitung dalam RUNE, bukan bita — alasannya sama dengan pada balasan.
	if length := len([]rune(message)); length > maxMessageLength {
		violations = append(violations, Violation{
			Field: FieldMessageBody,
			Message: "Pesan terlalu panjang — " + strconv.Itoa(length) +
				" karakter, sementara batasnya " + strconv.Itoa(maxMessageLength) + ".",
		})
	}

	if len(violations) > 0 {
		return NewMessageCommand{}, NewValidationError(violations)
	}

	return NewMessageCommand{
		Destination: destination,
		BranchCode:  branch,
		Message:     message,
		Sender:      cleanSender,
		CreatedAt:   now.UTC(),
	}, nil
}
