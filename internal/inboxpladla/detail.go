package inboxpladla

import (
	"errors"
	"strings"
)

// Berkas ini memuat layar RINCIAN satu klaim — yang dibuka tombol **"Detail Claim"**.
//
// # Asalnya di sistem lama
//
//	Section/InboxDLAReas_sect-Section.xml   tombol "Detail Claim" pada ketiga grid daftar
//	  -> Activity/SetDataViewKlaimReas_Act  memuat objek kerja klaim
//	     -> Activity/SetViewAttachmentReas  menyusun daftar dokumen
//	  -> Harness/ViewDetailClaimReas        grid dokumen
//	     -> Section/ViewShowObjectAdjReas   grid PLA, grid DLA, dan riwayat komunikasi
//
// # Tombolnya mengirim parameter yang SALAH di Pega, dan itu tidak berakibat apa-apa
//
// `Detail Claim` memanggil `SetDataViewKlaimReas_Act(inskey=.TSI, noaksep=.BRANCH_NAME)`.
// Pada ketiga grid daftar, `.BRANCH_NAME` beralias **PIC Teknik** — bukan nomor akseptasi.
//
// Nilai itu diteruskan ke `SetViewAttachmentReas`, yang menyalinnya ke `local.noaksep`
// lalu TIDAK PERNAH MEMBACANYA LAGI. Jadi cacatnya nyata tetapi mati; ia dicatat di sini
// supaya pembaca berikutnya tidak menghabiskan waktu mencari akibatnya. Parameter itu
// tidak dibawa sama sekali ke sistem baru.
//
// # Dua cabang `SetViewAttachmentReas` yang MATI pada jalur ini
//
// Activity itu menyusun daftar dokumen dari TIGA sumber. Dua di antaranya bersyarat
// `.NoPLA == Param.pladla` dan `.NO_DLA == Param.pladla` — sementara
// `SetDataViewKlaimReas_Act` tidak pernah mengirim `pladla` sama sekali.
//
// Artinya, ketika popup pertama dibuka, hanya sumber PERTAMA yang hidup:
// `GetDokumenReas` atas `POOLDATA.T_DOC_REAS`. Kedua sumber lain baru hidup setelah
// pengguna menekan tombol **"Dokumen"** pada salah satu baris PLA atau DLA — dan itulah
// yang menjelaskan mengapa `no_pladla` menjadi bagian kunci pada kueri dokumennya.

// AdviceKind menyatakan jenis pemberitahuan pada grid rincian.
//
// Nilainya dipakai sebagai kunci pencarian dokumen (`T_DOC_REAS.TIPE_PLADLA`), bukan
// sekadar label — sehingga ia konstanta, bukan teks bebas dari layar.
type AdviceKind string

const (
	// AdviceKindPLA — Preliminary Loss Advice (`POOLDATA.T_PLALIST`).
	AdviceKindPLA AdviceKind = "PLA"

	// AdviceKindDLA — Definite Loss Advice (`POOLDATA.T_DLALIST`).
	AdviceKindDLA AdviceKind = "DLA"
)

// Valid menyatakan jenis pemberitahuannya dikenali.
func (k AdviceKind) Valid() bool {
	return k == AdviceKindPLA || k == AdviceKindDLA
}

// ParseAdviceKind membaca jenis pemberitahuan dari alamat, tanpa memedulikan huruf.
func ParseAdviceKind(raw string) (AdviceKind, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(AdviceKindPLA):
		return AdviceKindPLA, true
	case string(AdviceKindDLA):
		return AdviceKindDLA, true
	default:
		return "", false
	}
}

// ClaimHeader adalah keterangan klaim di kepala layar rincian.
//
// Isinya sengaja sama dengan baris daftarnya: layar rincian dapat dibuka lewat alamat
// langsung, dan ketika itu terjadi barisnya tidak ada di tangan layar.
type ClaimHeader struct {
	ClaimKey     string
	ClaimNo      string
	PolicyNo     string
	Insured      string
	BusinessName string
	RegisterDate string
	LossDate     string
	PICTeknik    string
	StatusCode   string
	StatusLabel  string
}

// AdviceRow adalah satu baris grid **"PLA"** atau **"DLA"** pada layar rincian.
//
// # Kolomnya diambil dari `Section/ViewShowObjectAdjReas-Section.xml`
//
//	grid PLAList   No PLA · Tipe PLA · Nilai PLA ·                  [Dokumen]
//	grid DLAList   No DLA · Tipe DLA · Nilai DLA · No Akseptasi ·   [Dokumen]
//
// Keduanya dibawa satu tipe karena bentuknya sama kecuali satu kolom. Kolom yang tidak ada
// pada PLA — nomor akseptasi — dibiarkan KOSONG, bukan dihilangkan, supaya satu pemindai
// melayani kedua kueri.
type AdviceRow struct {
	// Kind menyatakan baris ini PLA atau DLA.
	Kind AdviceKind

	// No — kolom **"No PLA"** / **"No DLA"** <- `NOPLA` / `NODLA`.
	//
	// Ia sekaligus KUNCI pencarian dokumen: `T_DOC_REAS.NO_PLADLA`.
	No string

	// Type — kolom **"Tipe PLA"** / **"Tipe DLA"** <- `TIPEPLA` / `TIPEDLA`.
	Type string

	// Amount — kolom **"Nilai PLA"** / **"Nilai DLA"** <- `NILAIPLA` / `NILAIDLA`.
	//
	// Ia dibawa sebagai TEKS, bukan angka. Nilainya hanya digambar — tidak satu pun
	// dihitung di layar ini — dan membawanya sebagai angka pecahan akan memperkenalkan
	// pembulatan pada nilai uang yang `D-51` larang justru untuk keadaan seperti ini.
	Amount string

	// AcceptanceNo — kolom **"No Akseptasi"** <- `T_DLALIST.NOAKSEP`. Kosong pada PLA.
	AcceptanceNo string

	// AdviceDate adalah `TGLPLA` / `TGLDLA`. Tidak digambar sebagai kolom di Pega;
	// ia yang MENGURUTKAN barisnya.
	AdviceDate string

	// SentDate adalah `TGLKIRIM` — tanggal dokumennya dikirimkan kepada pemanggil.
	//
	// DITAMBAHKAN. Grid Pega tidak menggambarnya, dan ketiadaannya nyata akibatnya:
	// reasuradur melihat daftar pemberitahuan tanpa tahu kapan masing-masing sampai
	// kepadanya. Seluruh baris di sini SUDAH terkirim — lihat Repo.Advices.
	SentDate string
}

// DocumentRow adalah satu baris grid dokumen pada layar rincian.
//
// # Kolomnya diambil dari `Harness/ViewDetailClaimReas-Harness.xml`
//
//	Jenis Dokumen  <- `.ATTACHNOTE`, diisi `.GCNMCategory`
//	Kategori Dokumen <- `.DATAID`,   diisi `.GCNMType`
//	Nama           <- `.ATTACHNAME`
//
// Perhatikan kolom bernama `.DATAID` yang justru TIDAK berisi id dokumen — ia berisi jenis
// dokumen. Id dokumen yang sebenarnya dipakai mengunduh isinya, dan ia dibawa terpisah.
type DocumentRow struct {
	// ID adalah `POOLDATA.DATA_ATTACHFILE.DATAID` — dipakai mengunduh isinya.
	ID string

	// Category — kolom **"Jenis Dokumen"**.
	//
	// Namanya dicari ke `POOLDATA.V_LST_DOC_TYPE`; bila tidak ada, KODENYA yang digambar.
	Category string

	// SubCategory — kolom **"Kategori Dokumen"**.
	//
	// Namanya dicari ke `POOLDATA.V_LST_DET_TYPE_DOC`; bila tidak ada, KODENYA.
	SubCategory string

	// Name — kolom **"Nama"** <- `ATTACHNAME`.
	Name string

	// MimeType adalah `ATTACHMIMETYPE`. Tidak digambar; ia menentukan bagaimana berkasnya
	// diserahkan saat diunduh.
	MimeType string
}

// DocumentContent adalah isi satu dokumen beserta keterangan penyerahannya.
type DocumentContent struct {
	Name     string
	MimeType string

	// Content adalah isi berkasnya.
	//
	// `GetAttachmentFromDB_Sql` membungkusnya `pooldata.base64encode(attachfile)`, dan
	// pembungkusan itu TIDAK dibawa: ia memanggil procedure basis data, yang `D-02`
	// larang, dan ia membesarkan muatan sepertiga tanpa satu pun manfaat di sini. Isinya
	// dibaca sebagai bita apa adanya dari kolom BLOB.
	Content []byte
}

// Conversation adalah satu baris grid riwayat komunikasi pada layar rincian.
//
// # Kolomnya diambil dari `Section/ViewShowObjectAdjReas-Section.xml`
//
//	Date     <- `.CloseClaimDate`   (!) alias Pega; isinya tanggal pesan
//	Sender   <- `.UserName`
//	Message  <- `.Email`            (!) alias Pega; isinya ISI PESAN
//	Reply    <- `.CloseClaimNote`   (!) alias Pega; isinya balasan
//
// Ketiga alias bertanda `(!)` tidak menyatakan isinya sama sekali — `.Email` yang berisi
// isi pesan adalah yang paling jauh menyesatkan. Nama di sini menyebut isinya (`D-19`).
type Conversation struct {
	// ID adalah `M_KOMUNIKASI_PNC.KOMUNIKASIID` — kunci yang dipakai membalas.
	ID string

	// CreatedAt — kolom **"Date"** <- `CREATEDDATE`.
	CreatedAt string

	// SenderName — kolom **"Sender"** <- `SENDERNAME`.
	SenderName string

	// Message — kolom **"Message"** <- `MESSAGE`.
	Message string

	// Reply — kolom **"Reply"** <- `REPLYMESSAGE`. Kosong berarti belum dijawab.
	Reply string

	// ReplierName adalah `REPLYFROMNAME`. Tidak digambar sebagai kolom di Pega;
	// ia digambar sebagai keterangan di bawah balasannya.
	ReplierName string

	// RepliedAt adalah `CREATEDATEREPLY`.
	RepliedAt string

	// Answered menyatakan `KOMUNIKASISTATUS = '1'`.
	//
	// Ia dibawa terpisah dari Reply yang terisi, dan itu bukan kelebihan: keduanya DAPAT
	// berbeda pada data lama, dan yang menentukan sebuah percakapan muncul di daftar
	// "Sudah Dijawab" adalah penandanya — bukan isinya.
	Answered bool

	// CanReply menyatakan pemanggil boleh membalas percakapan ini.
	//
	// Ia dihitung di PELADEN, bukan disimpulkan layar dari Answered. Layar yang
	// menyimpulkannya sendiri akan menggambar tombol balas pada percakapan yang
	// permintaannya akan ditolak — dan tombol yang selalu ditolak lebih buruk daripada
	// tombol yang tidak ada.
	CanReply bool
}

// ClaimDetail adalah seluruh isi layar rincian satu klaim.
type ClaimDetail struct {
	Header ClaimHeader

	// PLA dan DLA adalah kedua grid pemberitahuan.
	PLA []AdviceRow
	DLA []AdviceRow

	// Conversations adalah riwayat komunikasi klaim ini yang menyangkut pemanggil.
	Conversations []Conversation
}

// AdviceColumns adalah kolom grid PLA atau DLA pada layar rincian.
//
// Judulnya mengikuti section Pega apa adanya (`D-13`), termasuk campuran bahasanya:
// "No PLA" dan "Nilai PLA" berbahasa Indonesia sementara grid daftar di layar induknya
// berbahasa Inggris. Itu memang begitu di Pega, dan menyeragamkannya berarti mengubah teks
// yang sudah dikenal pembacanya.
func AdviceColumns(kind AdviceKind) []Column {
	suffix := string(kind)

	columns := []Column{
		{Key: "nomor", Title: "No " + suffix},
		{Key: "tipe", Title: "Tipe " + suffix},
		{Key: "nilai", Title: "Nilai " + suffix},
	}

	if kind == AdviceKindDLA {
		columns = append(columns, Column{Key: "no_akseptasi", Title: "No Akseptasi"})
	}

	// DITAMBAHKAN — lihat AdviceRow.SentDate.
	return append(columns, Column{Key: "tanggal_kirim", Title: "Tanggal Kirim", Date: true})
}

// DocumentColumns adalah kolom grid dokumen pada layar rincian.
func DocumentColumns() []Column {
	return []Column{
		{Key: "jenis_dokumen", Title: "Jenis Dokumen"},
		{Key: "kategori_dokumen", Title: "Kategori Dokumen"},
		{Key: "nama", Title: "Nama"},
	}
}

// ConversationColumns adalah kolom grid riwayat komunikasi pada layar rincian.
func ConversationColumns() []Column {
	return []Column{
		{Key: "tanggal", Title: "Date", Date: true},
		{Key: "pengirim", Title: "Sender"},
		{Key: "pesan", Title: "Message"},
		{Key: "balasan", Title: "Reply"},
	}
}

// ErrAdviceKindUnknown berarti jenis pemberitahuan pada alamat tidak dikenali.
var ErrAdviceKindUnknown = errors.New(
	"inboxpladla: jenis pemberitahuan harus PLA atau DLA")

// ErrDocumentNotFound berarti dokumennya tidak ada, atau bukan milik pemanggil.
//
// # Kenapa KEDUANYA satu galat
//
// Karena membedakannya membocorkan keberadaan dokumen milik mitra lain. Jawaban "dokumen
// itu ada tetapi bukan milik Anda" memberi tahu penanya bahwa dokumennya ada — dan pada
// layar yang dibaca pihak luar, itu keterangan yang tidak berhak ia terima.
var ErrDocumentNotFound = errors.New("inboxpladla: dokumen tidak ditemukan")

// ErrConversationNotFound berarti percakapannya tidak ada, atau bukan milik pemanggil.
//
// Alasan penggabungannya sama dengan ErrDocumentNotFound.
var ErrConversationNotFound = errors.New("inboxpladla: percakapan tidak ditemukan")

// ErrConversationAlreadyAnswered berarti percakapannya sudah pernah dijawab.
//
// # Kenapa ia penolakan, bukan penimpaan
//
// Karena `M_KOMUNIKASI_PNC` menyimpan SATU balasan per percakapan — `REPLYMESSAGE` adalah
// kolom tunggal, bukan tabel anak. Balasan kedua akan MENIMPA yang pertama, dan yang
// pertama tidak dapat dipulihkan dari mana pun.
//
// `ReplyKomunikasi-SQL.xml` tidak memagarinya: ia menulis `where komunikasiid = …` saja.
// Di Pega itu tidak terasa karena tombol balas hanya dapat dicapai dari daftar yang sudah
// menyaring percakapan belum terjawab. Di sini alamatnya dapat dipanggil langsung,
// sehingga penjagaannya harus ada pada pernyataannya sendiri.
var ErrConversationAlreadyAnswered = errors.New(
	"inboxpladla: percakapan ini sudah dijawab")
