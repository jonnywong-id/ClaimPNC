package inboxlaporanklaim

// Summary adalah pencacah yang digambar di atas daftar.
//
// # Asalnya
//
// `RDB List/BrowseClaimRCV_Aksep-SQL.xml` — satu kueri yang mengembalikan DELAPAN angka
// sekaligus lewat `COUNT(1)` dan tujuh `SUM(CASE WHEN … THEN 1 ELSE 0 END)`. Satu kueri
// untuk delapan angka, bukan delapan kueri, dan bentuk itu dipertahankan: menghitungnya
// terpisah berarti kedelapan angka dapat berasal dari saat yang berbeda-beda, dan
// jumlahnya tidak lagi cocok dengan totalnya.
//
// Kueri itu menyaring `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`,
// sehingga berkas yang sudah selesai atau ditolak TIDAK ikut tercacah.
//
// # Kenapa tab "Data rejected" tidak punya angka
//
// Karena kueri pencacah lama memang tidak menghitungnya — berkas ditolak justru yang
// dikecualikan penyaring di atas. Itu bukan kelalaian modul ini, dan lencananya
// dikosongkan alih-alih diisi angka yang dihitung dengan cara lain: dua angka yang
// dihitung berbeda dan ditampilkan berdampingan akan dibaca sebagai satu hal yang sama.
//
// # Nama alias yang tidak dibawa
//
// Kedelapan angka itu dialiaskan ke nama properti yang tidak ada hubungannya dengan
// isinya — `Sender`, `EmailPengirim`, `Keterangan`, `KronologisKejadian`, `PNCCaseID`,
// `PolicyNo`, `QQName`, `RCVID`. Pemetaan baliknya ada di berkas `.sql`; di sini
// masing-masing disebut menurut tab yang dihitungnya.
type Summary struct {
	// Total adalah seluruh berkas yang belum selesai dan belum ditolak.
	// Alias lama: "Sender", dari COUNT(1).
	Total int

	// NotTransferred: pnccaseid NULL dan statuslock_1 NULL. Alias lama "EmailPengirim".
	NotTransferred int

	// Unregistered: pnccaseid NULL dan statuslock_1 TIDAK NULL. Alias lama "Keterangan".
	Unregistered int

	// Outstanding: pnccaseid dan statuslock_1 keduanya TIDAK NULL.
	// Alias lama "KronologisKejadian".
	Outstanding int

	// Accepted: klaimnya sudah punya nomor akseptasi di T_CLAIM_ADJUSTMENT.
	// Alias lama "PNCCaseID".
	Accepted int

	// MessageUnanswered, MessageWaiting, MessageReplied mencacah ketiga tab komunikasi.
	// Alias lama berturut-turut "PolicyNo", "QQName", dan "RCVID".
	//
	// Lihat MessageFilter: pencacah lama dan kueri gridnya TIDAK sepakat untuk tab
	// pertama. Angka di sini mengikuti pencacah lama apa adanya supaya lencananya sama
	// dengan Pega; isi tabelnya mengikuti kueri grid. Selisih itu sudah ada sebelum
	// modul ini, dan sengaja tidak ditutupi.
	MessageUnanswered int
	MessageWaiting    int
	MessageReplied    int
}

// CountOf mengembalikan angka lencana sebuah tab.
//
// Nilai kedua false berarti tab itu memang tidak punya angka di sistem lama — lihat
// catatan tentang "Data rejected" di atas. Ia dibedakan dari angka nol dengan sengaja:
// lencana bertuliskan 0 menyatakan "tidak ada berkas", sedangkan tidak adanya lencana
// menyatakan "jumlahnya tidak dihitung". Keduanya berbeda, dan layar harus dapat
// membedakannya.
func (s Summary) CountOf(c Category) (int, bool) {
	switch c {
	case CategoryAll:
		return s.Total, true
	case CategoryNotTransferred:
		return s.NotTransferred, true
	case CategoryUnregistered:
		return s.Unregistered, true
	case CategoryOutstanding:
		return s.Outstanding, true
	case CategoryAccepted:
		return s.Accepted, true
	case CategoryMessageUnanswered:
		return s.MessageUnanswered, true
	case CategoryMessageWaiting:
		return s.MessageWaiting, true
	case CategoryMessageReplied:
		return s.MessageReplied, true
	default:
		// CategoryRejected jatuh ke sini.
		return 0, false
	}
}
