// Package memory memenuhi seam inboxkomunikasicabang.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaringnya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Kelima penyaring ditiru sedekat-dekatnya dengan predikat SQL-nya: kanal
// percakapan, batas cabang, pengirim terisi, pesan terisi, dan ada-tidaknya balasan.
//
// # SATU PENYARING YANG SENGAJA DITIRU BERBEDA ANTARA GRID DAN PENCACAH
//
// Kedua kueri grid memeriksa `REPLYMESSAGE` saja; kedua kueri pencacah memeriksa
// `REPLYFROM` DAN `REPLYMESSAGE` sekaligus. Perbedaan satu kolom itu ada di sistem lama,
// dan penyimpanan ini menirunya apa adanya (lihat matchesGrid dan countsAsAnswered).
//
// Menyeragamkannya akan membuat uji lulus untuk perilaku yang TIDAK terjadi di Oracle —
// dan justru selisih inilah yang akan membuat angka pencacah berbeda dari jumlah baris
// tabel pada percakapan yang salah satu kolom balasannya terisi sendirian.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Row adalah satu baris contoh — satu baris `POOLDATA.M_KOMUNIKASI_PNC` apa adanya.
//
// Ia menyimpan kolom MENTAH, bukan nilai yang sudah diterjemahkan, justru supaya
// penerjemahnya ikut teruji. `inboxkomunikasicabang.OriginOf` dan `RecipientOf` dipanggil
// saat baris diambil — menyimpannya sudah jadi akan membuat penerjemah yang salah tetap
// lulus uji.
type Row struct {
	// ID adalah `KOMUNIKASIID`.
	ID string

	// CaseID adalah `CASEID` — kanal percakapan.
	//
	// Kedua grid menyaring `= 'CABANG'`. Baris contoh memuat satu percakapan ber-`CABANG
	// SELESAI` justru supaya penyaring itu punya sesuatu untuk menolak — tanpa itu,
	// penyaring yang hilang tidak akan ketahuan.
	CaseID string

	// CommunicateFrom adalah `COMMUNICATE_FROM` — asal pesan, MENTAH.
	CommunicateFrom string

	// CommunicateTo adalah `COMMUNICATE_TO` — tujuan pesan, MENTAH.
	CommunicateTo string

	// Sender adalah `SENDER` — Operator ID pengirim.
	//
	// Kedua kueri grid menyaring `SENDER IS NOT NULL`, sehingga baris yang pengirimnya
	// kosong TIDAK muncul. Baris contoh memuat satu di antaranya.
	Sender string

	// SenderName adalah `SENDERNAME`.
	//
	// # Ia SENGAJA TIDAK PERNAH SAMPAI KE LAYAR
	//
	// Kueri lama mengambilnya dengan alias `UserName`, lalu mengambil `COMMUNICATE_FROM`
	// dengan alias yang SAMA enam baris kemudian. Yang menang adalah yang terakhir —
	// terbukti dari `PNCGetInboxKomunikasiCabang_Act` langkah 5 yang membandingkan
	// `.UserName` dengan `"1"`, perbandingan yang hanya masuk akal untuk kode asal.
	//
	// Ia disimpan di sini, tidak dibuang, supaya uji dapat membuktikan ia memang TIDAK
	// muncul. Kolom yang tidak disimpan tidak dapat dibuktikan tidak muncul.
	SenderName string

	// Message adalah `MESSAGE` — isi pesan.
	//
	// Kedua kueri grid menyaring `MESSAGE IS NOT NULL`.
	Message string

	// ReplyMessage adalah `REPLYMESSAGE` — isi balasan. Kosong berarti NULL.
	//
	// Inilah satu-satunya penyaring yang membedakan kedua tab.
	ReplyMessage string

	// ReplyFrom adalah `REPLYFROM` — Operator ID penjawab. Kosong berarti NULL.
	//
	// Ia dipakai PENCACAH saja, tidak oleh grid. Lihat catatan di kepala paket.
	ReplyFrom string

	// ReplyFromName adalah `REPLYFROMNAME` — nama penjawab, digambar kolom "Penjawab(Dari)".
	ReplyFromName string

	// Status adalah `KOMUNIKASISTATUS`.
	Status string

	// CreatedAt adalah `CREATEDDATE` — dasar urutan tab "Belum Dijawab", MENAIK.
	//
	// Disimpan sebagai TEKS berbentuk `YYYY-MM-DD HH:MM`, bukan time.Time. Alasannya:
	// bentuk kolomnya di Oracle tidak diketahui (`R-08`), dan kolom inilah yang digambar
	// layar apa adanya. Urutan teks pada bentuk ini sama dengan urutan waktunya, sehingga
	// pengurutannya tetap benar tanpa menetapkan tipe yang belum dipastikan.
	CreatedAt string

	// RepliedAt adalah `CREATEDATEREPLY` — dasar urutan tab "Sudah Dijawab", MENURUN.
	RepliedAt string

	// Attachments adalah lampiran percakapan ini — `POOLDATA.D_KOMUNIKASI_PNC`.
	Attachments []inboxkomunikasicabang.Attachment
}

// Store adalah penyimpanan percakapan di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: modul ini hanya
// membaca, dan tidak ada satu pun operasi yang menulis.
type Store struct {
	rows []Row
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	return &Store{rows: rows}
}

// NewSampleStore membentuk penyimpanan berisi baris contoh.
func NewSampleStore() *Store {
	return NewStore(SampleRows()...)
}

// List mengembalikan satu halaman percakapan yang cocok.
func (s *Store) List(
	_ context.Context,
	query inboxkomunikasicabang.Query,
	page inboxkomunikasicabang.Pagination,
) (inboxkomunikasicabang.Page, error) {
	matched := []Row{}
	for _, row := range s.rows {
		if !matchesGrid(row, query.Branch) {
			continue
		}
		if row.answered() != query.Tab.Answered {
			continue
		}
		matched = append(matched, row)
	}

	sortForTab(matched, query.Tab.Answered)

	items := make([]inboxkomunikasicabang.Conversation, 0, len(matched))
	for _, row := range matched {
		items = append(items, row.toConversation())
	}

	return inboxkomunikasicabang.Slice(items, page), nil
}

// Summarize mengembalikan kedua pencacah.
//
// Ia memakai countsAsAnswered, BUKAN answered — kedua penyaring itu berbeda satu kolom di
// sistem lama, dan perbedaan itulah yang sedang ditiru.
func (s *Store) Summarize(
	_ context.Context,
	filter inboxkomunikasicabang.BranchFilter,
) (inboxkomunikasicabang.Summary, error) {
	summary := inboxkomunikasicabang.Summary{}

	for _, row := range s.rows {
		// Pencacah TIDAK menyaring pengirim dan pesan.
		//
		// Kedua kueri pencacah hanya memeriksa kanal, batas cabang, dan kolom balasan —
		// tidak ada `SENDER IS NOT NULL` maupun `MESSAGE IS NOT NULL` di sana, padahal
		// kedua kueri grid memilikinya. Selisih itu dibawa apa adanya.
		if !matchesChannel(row) || !matchesBranch(row, filter) {
			continue
		}

		if row.countsAsAnswered() {
			summary.Answered++
			continue
		}
		if row.countsAsNotAnswered() {
			summary.NotAnswered++
		}
	}

	return summary, nil
}

// Detail mengembalikan isi layar Detail Komunikasi untuk satu percakapan.
//
// Batas cabang tetap diberlakukan di sini, dan itu bukan kelebihan kehati-hatian: nomor
// percakapan berurutan dan mudah ditebak, sehingga tanpa batas ini layar detail menjadi
// pintu samping ke percakapan cabang mana pun.
//
// Sistem lama tidak memeriksanya — tombolnya hanya ada pada baris yang sudah tersaring —
// tetapi tombol bukan penjagaan. Ini perluasan yang diambil sendiri dan dinyatakan di sini
// serta di uji, supaya dapat dikoreksi.
func (s *Store) Detail(
	_ context.Context,
	id string,
	filter inboxkomunikasicabang.BranchFilter,
) (inboxkomunikasicabang.ConversationDetail, error) {
	wanted := strings.TrimSpace(id)

	thread := []Row{}
	for _, row := range s.rows {
		if strings.TrimSpace(row.ID) != wanted {
			continue
		}
		if !matchesBranch(row, filter) {
			continue
		}
		thread = append(thread, row)
	}

	if len(thread) == 0 {
		return inboxkomunikasicabang.ConversationDetail{},
			inboxkomunikasicabang.ErrConversationNotFound
	}

	// Utas SELALU menaik menurut tanggal pesan, pada kedua tab.
	//
	// Layar detail adalah percakapan, dan percakapan dibaca dari awal. Urutan menurun tab
	// "Sudah Dijawab" berlaku untuk DAFTARNYA, bukan untuk isi satu percakapan.
	sort.SliceStable(thread, func(i, j int) bool {
		return thread[i].CreatedAt < thread[j].CreatedAt
	})

	detail := inboxkomunikasicabang.ConversationDetail{
		ID:          wanted,
		Messages:    make([]inboxkomunikasicabang.ThreadMessage, 0, len(thread)),
		Attachments: []inboxkomunikasicabang.Attachment{},
		Origin:      inboxkomunikasicabang.OriginOf(thread[0].CommunicateFrom),
	}

	for _, row := range thread {
		detail.Messages = append(detail.Messages, inboxkomunikasicabang.ThreadMessage{
			CreatedAt:      row.CreatedAt,
			SenderOrigin:   inboxkomunikasicabang.OriginOf(row.CommunicateFrom),
			SenderOperator: row.Sender,
			Message:        row.Message,
			Reply:          row.ReplyMessage,
			ReplierName:    row.ReplyFromName,
			RepliedAt:      row.RepliedAt,
		})
		detail.Attachments = append(detail.Attachments, row.Attachments...)
	}

	return detail, nil
}

// answered menyatakan penyaring GRID: balasannya terisi.
//
// Ia memeriksa SATU kolom saja, persis seperti kedua kueri grid.
func (r Row) answered() bool {
	return strings.TrimSpace(r.ReplyMessage) != ""
}

// countsAsAnswered menyatakan penyaring PENCACAH "Answered".
//
// Dari `GetCountKomunikasiCabangAnswered`:
//
//	AND REPLYFROM IS NOT NULL
//	AND REPLYMESSAGE IS NOT NULL
//
// DUA kolom, bukan satu. Percakapan yang dibalas tanpa penjawab tercatat muncul di tabel
// tetapi TIDAK terhitung di pencacah — dan itu perilaku sistem lama apa adanya.
func (r Row) countsAsAnswered() bool {
	return strings.TrimSpace(r.ReplyFrom) != "" && strings.TrimSpace(r.ReplyMessage) != ""
}

// countsAsNotAnswered menyatakan penyaring PENCACAH "Not Answered".
//
// Dari `GetCountKomunikasiCabangNotAnswered`:
//
//	AND REPLYFROM IS NULL
//	AND REPLYMESSAGE IS NULL
//
// Perhatikan keduanya BUKAN saling melengkapi: percakapan yang salah satu kolomnya terisi
// sendirian tidak terhitung di mana pun, sehingga jumlah kedua pencacah dapat KURANG dari
// jumlah seluruh percakapan. Itu dibawa apa adanya dan dinyatakan lewat PlannedDifferences.
func (r Row) countsAsNotAnswered() bool {
	return strings.TrimSpace(r.ReplyFrom) == "" && strings.TrimSpace(r.ReplyMessage) == ""
}

// toConversation menyusun baris layar dari baris penyimpanan.
//
// Kedua penerjemah asal dipanggil DI SINI, bukan disimpan sudah jadi, supaya keduanya ikut
// teruji setiap kali baris diambil.
func (r Row) toConversation() inboxkomunikasicabang.Conversation {
	return inboxkomunikasicabang.Conversation{
		ID:              r.ID,
		CreatedAt:       r.CreatedAt,
		SenderOrigin:    inboxkomunikasicabang.OriginOf(r.CommunicateFrom),
		SenderOperator:  r.Sender,
		Message:         r.Message,
		Reply:           r.ReplyMessage,
		ReplierName:     r.ReplyFromName,
		RecipientOrigin: inboxkomunikasicabang.RecipientOf(r.CommunicateTo),
		Status:          r.Status,
		RepliedAt:       r.RepliedAt,
	}
}

// matchesGrid memberlakukan SELURUH penyaring kedua kueri grid kecuali kolom balasan.
func matchesGrid(row Row, filter inboxkomunikasicabang.BranchFilter) bool {
	if !matchesChannel(row) {
		return false
	}
	if !matchesBranch(row, filter) {
		return false
	}
	// `SENDER IS NOT NULL AND MESSAGE IS NOT NULL` pada kedua kueri grid.
	if strings.TrimSpace(row.Sender) == "" || strings.TrimSpace(row.Message) == "" {
		return false
	}
	return true
}

// matchesChannel memberlakukan `CASEID = 'CABANG'`.
//
// Perbandingannya PERSIS, bukan awalan: `CABANG SELESAI` berawalan `CABANG` dan TIDAK boleh
// lolos — justru nilai itulah yang menandai percakapan yang sudah ditutup.
func matchesChannel(row Row) bool {
	return strings.TrimSpace(row.CaseID) == inboxkomunikasicabang.CaseOpen
}

// matchesBranch memberlakukan batas cabang.
//
// Bentuknya meniru penyaring yang dirangkai `PNCCountKomunikasiCabang_Act`:
//
//	(COMMUNICATE_TO = :kode OR COMMUNICATE_FROM = :kode)
//
// Perhatikan ia `OR`, bukan `AND`: sebuah percakapan terlihat oleh cabang yang MENGIRIM
// maupun yang MENERIMA. Menukarnya dengan `AND` mengosongkan seluruh layar, karena tidak
// ada percakapan yang asal dan tujuannya sama.
func matchesBranch(row Row, filter inboxkomunikasicabang.BranchFilter) bool {
	code := strings.TrimSpace(filter.Code)
	if code == "" {
		// Tidak pernah terjadi lewat ResolveBranch, yang selalu mengisi Code. Penjagaan ini
		// ada supaya Repo yang dipanggil dengan BranchFilter kosong — misalnya dari uji
		// yang dibuat kemudian — TIDAK diam-diam menampilkan seluruh percakapan.
		return false
	}
	return strings.TrimSpace(row.CommunicateTo) == code ||
		strings.TrimSpace(row.CommunicateFrom) == code
}

// sortForTab mengurutkan baris sesuai tab.
//
// Kedua arah BERLAWANAN, dan itu perilaku sistem lama apa adanya:
//
//	Belum Dijawab   ORDER BY CREATEDDATE ASC       yang paling lama menunggu di atas
//	Sudah Dijawab   ORDER BY CREATEDATEREPLY DESC  yang paling baru dibalas di atas
//
// Nomor percakapan dipakai sebagai pemutus seri. Kueri lama TIDAK punya pemutus seri, dan
// tanpa itu dua baris bertanggal sama dapat berpindah urutan di antara dua permintaan —
// yang membuat paginasi melewatkan atau menggandakan baris. Menambahkannya TIDAK mengubah
// baris mana yang tampil, hanya memastikannya tetap.
func sortForTab(rows []Row, answered bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]

		if answered {
			if left.RepliedAt != right.RepliedAt {
				return left.RepliedAt > right.RepliedAt
			}
			return left.ID > right.ID
		}

		if left.CreatedAt != right.CreatedAt {
			return left.CreatedAt < right.CreatedAt
		}
		return left.ID < right.ID
	})
}

var _ inboxkomunikasicabang.Repo = (*Store)(nil)
