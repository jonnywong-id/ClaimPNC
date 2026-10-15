package inboxpladlapredla

import (
	"strings"
	"time"
)

// QueryInput adalah isian mentah dari panel pencarian, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode daftar yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Search adalah isi kotak "No Klaim". Kosong berarti tidak menyaring.
	Search string

	// From dan To adalah isi kotak "Dari" dan "Sampai".
	//
	// Keduanya nil berarti tidak menyaring tanggal. Salah satunya nil pun DITERIMA — lihat
	// catatan pada Query.From.
	From *time.Time
	To   *time.Time
}

// Query adalah permintaan isi satu daftar yang sudah tervalidasi.
type Query struct {
	// Tab adalah daftar yang diminta, lengkap dengan kolom dan penyaringnya.
	Tab Tab

	// Search adalah kata kunci pencarian yang sudah dipangkas. Kosong berarti tidak
	// menyaring.
	//
	// Yang dicocokkan adalah `T_CLAIM_PNC.CLAIMID`, BUKAN `CLAIMNO`. Itu perilaku Pega —
	// `GetSearchPNCList_DLA` menyusun `AND B.CLAIMID LIKE '%…%'` — dan ia tetap menemukan
	// nomor klaim yang diketik pengguna karena `CLAIMID` berbentuk
	// `ASM-FW-GCNMFW-WORK <nomor klaim>`, sehingga nomor klaimnya ada di ujungnya.
	//
	// Akibat sampingannya dibawa apa adanya: mengetik `WORK` menemukan SELURUH klaim,
	// karena kata itu ada di setiap kunci. Pengguna tidak akan melakukannya, dan
	// memperbaikinya berarti mengubah kolom yang dicari — selisih yang tidak diminta
	// siapa pun.
	Search string

	// From adalah batas bawah rentang tanggal dokumen, INKLUSIF.
	//
	// Nil berarti tidak ada batas bawah. Pega tidak mengenal keadaan ini: isian kosong di
	// sana tetap dirangkai menjadi `to_date('','dd/mm/yyyy')`, yang ditolak Oracle. Lihat
	// PlannedDifferences.
	From *time.Time

	// To adalah batas atas rentang tanggal dokumen, EKSKLUSIF — ia sudah berisi tengah
	// malam HARI BERIKUTNYA setelah tanggal yang diketik pengguna.
	//
	// # Kenapa eksklusif, padahal pengguna mengetik batas yang inklusif
	//
	// Supaya kuerinya tidak perlu `TRUNC(TGLPLA)`. Pega memakai
	// `trunc(a.tglpla) <= to_date(<sampai>)`, dan `TRUNC` pada kolom membuat setiap index
	// atas kolom itu tidak terpakai — pada tabel dokumen yang tumbuh terus, itu pemindaian
	// penuh setiap kali panel pencarian dipakai.
	//
	// `TGLPLA >= <dari> AND TGLPLA < <sampai+1hari>` memilih baris yang SAMA PERSIS dan
	// tetap dapat memakai index. Pola yang sama sudah dipakai modul Archive Dokumen Klaim.
	To *time.Time

	// Caller adalah identitas pemanggil. Ia dibawa untuk jejak, bukan untuk menyaring —
	// lihat Caller.
	Caller Caller
}

// maxSearchLength membatasi panjang kata kunci pencarian.
//
// Kunci terpanjang yang dapat diketik masuk akal adalah `ASM-FW-GCNMFW-WORK PNCN.YY.xxxx`,
// sekitar 32 karakter — jauh di bawah batas ini. Batasnya ada bukan untuk menolak nomor
// klaim yang sah, melainkan supaya kata kunci sepanjang satu megabita tidak pernah sampai
// ke basis data.
const maxSearchLength = 100

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama
// (`11-CROSSCUTTING.md` §1.2).
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		// Daftar yang tidak dikenal ditolak SENDIRIAN, sebelum isian lain diperiksa.
		//
		// Tanpa tab, tidak ada yang dapat dikatakan tentang isian lainnya: judul rentang
		// tanggal dan kolom yang dicari keduanya milik tab. Mengumpulkannya bersama
		// pelanggaran lain akan menghasilkan pesan yang menyebut isian pada daftar yang
		// tidak ada.
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Daftar tidak dikenal. Pilih PLA, DLA, atau Pre DLA.",
		}})
	}

	var violations []Violation

	search := strings.TrimSpace(input.Search)
	if len(search) > maxSearchLength {
		violations = append(violations, Violation{
			Field:   FieldSearch,
			Message: "Kata kunci pencarian terlalu panjang.",
		})
	}

	from := calendarDate(input.From)
	to := calendarDate(input.To)

	// Rentang terbalik DITOLAK, bukan ditukar diam-diam.
	//
	// Menukarnya akan menampilkan hasil yang benar untuk pertanyaan yang TIDAK diajukan,
	// dan pengguna tidak akan pernah tahu ia salah mengisi.
	if from != nil && to != nil && to.Before(*from) {
		violations = append(violations, Violation{
			Field:   FieldTo,
			Message: "\"Sampai\" tidak boleh lebih awal daripada \"Dari\".",
		})
	}

	if err := NewValidationError(violations); err != nil {
		return Query{}, err
	}

	// Batas atas digeser satu hari SETELAH validasi, bukan sebelumnya.
	//
	// Bila digeser lebih dulu, pesan "tidak boleh lebih awal" akan membandingkan tanggal
	// yang sudah bergeser — dan rentang satu hari (Dari = Sampai) akan terbaca sebagai
	// rentang yang sah dua hari, atau sebaliknya, tergantung arah gesernya.
	if to != nil {
		exclusive := to.AddDate(0, 0, 1)
		to = &exclusive
	}

	return Query{
		Tab:    tab,
		Search: search,
		From:   from,
		To:     to,
		Caller: cleanCaller,
	}, nil
}

// calendarDate memangkas jam dari sebuah tanggal, menyisakan tengah malam.
//
// # Kenapa UTC dan bukan WIB
//
// Karena kolom tanggal di basis data lama disimpan TANPA zona waktu, dan yang dibandingkan
// dengannya harus berupa tanggal kalender polos. Mengonversinya ke WIB lebih dulu akan
// menggeser batas rentang tujuh jam, dan dokumen yang terbit sebelum pukul 07.00 akan
// masuk ke hari yang salah.
//
// Ini konvensi yang sama dengan modul Archive Dokumen Klaim, dan keterkaitannya dengan
// `R-12` — pergeseran zona waktu pada data lama — berlaku pula di sini.
func calendarDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	truncated := time.Date(
		value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &truncated
}

// Matches menyatakan apakah sebuah baris lolos penyaring query ini.
//
// # Kenapa ia di DOMAIN, bukan di penyimpanan memori
//
// Karena arti pencarian adalah aturan bisnis, bukan detail penyimpanan: ia mencocokkan
// `CLAIMID`, bukan nomor klaim, dan itu keputusan yang diambil dari membaca Pega.
// Menaruhnya di penyimpanan memori berarti penyimpanan SQL dapat memahaminya berbeda tanpa
// ada satu pun uji yang gagal.
//
// Penyimpanan SQL tidak memanggil fungsi ini — ia menyusun klausa WHERE yang setara — dan
// kesetaraan keduanya dijaga uji di kedua sisi.
//
// Rentang tanggal TIDAK diperiksa di sini: ia menyaring tabel DOKUMEN, bukan baris klaim
// yang sudah jadi, sehingga penyimpanan memori menerapkannya saat menyusun barisnya.
func (q Query) Matches(row Row) bool {
	if q.Search == "" {
		return true
	}
	return strings.Contains(
		strings.ToUpper(row.ClaimKey), strings.ToUpper(q.Search))
}

// WithinRange menyatakan apakah sebuah tanggal dokumen masuk rentang query ini.
//
// Dipakai penyimpanan memori saat menyusun baris. Nil pada salah satu batas berarti batas
// itu tidak berlaku.
//
// Batas atas dibandingkan dengan `Before`, bukan `!After`, karena Query.To sudah EKSKLUSIF
// — lihat catatan di sana.
func (q Query) WithinRange(moment time.Time) bool {
	if q.From != nil && moment.Before(*q.From) {
		return false
	}
	if q.To != nil && !moment.Before(*q.To) {
		return false
	}
	return true
}
