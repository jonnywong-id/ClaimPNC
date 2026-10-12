package inboxpladla

import (
	"strconv"
	"strings"
	"time"
)

// Berkas ini memuat tindakan **"Balas Pesan"** — satu-satunya operasi MENULIS di modul ini.
//
// # Kenapa ia layak disebut tersendiri
//
// Karena yang menulis adalah PIHAK LUAR. Di seluruh modul yang sudah dibangun, inilah
// penulisan pertama yang pelakunya bukan pegawai Asuransi Sinar Mas melainkan mitra
// reasuransi, dan tulisannya masuk ke tabel yang dibaca petugas internal lewat modul
// `inboxkomunikasicabang`.
//
// # Asalnya di sistem lama
//
//	Section/ViewShowObjectAdjReas-Section.xml
//	  grid tempHistoryKomunikasi, kolom terakhir -> SUB_SECTION SectionBalasKomunikasi_Reas
//	  tombol "Balas Pesan" -> Activity PNCReplyMessage
//
// **Keduanya HILANG dari export** (`R-16`), diperiksa dengan `find` atas seluruh berkas.
// Yang ada hanyalah pernyataan yang mereka jalankan:
//
//	RDB List/ReplyKomunikasi-SQL.xml
//	  update POOLDATA.m_komunikasi_pnc
//	     set replymessage={tempReply.ADDRESS},
//	         replyfrom={OperatorID.pyUserIdentifier},
//	         CREATEDATEREPLY={temp.AnalystTransferDate DateTime},
//	         replyfromname={OperatorID.pyUserName},
//	         komunikasistatus='1'
//	   where komunikasiid={tempReply.M_SURVEY_ID}
//
// Bentuk tulisannya karena itu BUKAN tebakan — ia terbaca kolom demi kolom. Yang tidak
// terbaca adalah tata letak formnya, dan itu tidak menentukan apa yang tersimpan.
//
// Preseden lengkapnya sudah ada pula: modul `inboxkomunikasicabang` menulis balasan ke
// tabel yang SAMA lewat `PNCReplyMessageCabang`, dan kedua penyimpangan yang diambilnya —
// pemagaran kunci dan waktu yang lahir di satu tempat — diambil ulang di sini dengan
// alasan yang sama.

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi pada form balasan.
const (
	FieldConversation = "percakapan"
	FieldReply        = "balasan"
)

// maxReplyLength membatasi panjang balasan.
//
// Ia BUKAN tebakan atas lebar kolom `REPLYMESSAGE` — DDL-nya belum ada (`R-08`) —
// melainkan penjaga terhadap kiriman yang jelas tidak masuk akal. Angkanya disamakan
// dengan modul `inboxkomunikasicabang`, dan itu disengaja: keduanya mengisi KOLOM YANG
// SAMA, sehingga dua batas yang berbeda akan menolak kalimat yang sama pada satu layar dan
// menerimanya pada layar lain.
const maxReplyLength = 4000

// ReplyInput adalah isian mentah form balasan.
type ReplyInput struct {
	// ConversationID adalah `M_KOMUNIKASI_PNC.KOMUNIKASIID`.
	ConversationID string

	// Message adalah isi balasannya.
	Message string
}

// ReplyCommand adalah balasan yang sudah tervalidasi.
//
// Ia hanya lahir lewat NewReplyCommand.
type ReplyCommand struct {
	// ClaimKey adalah klaim tempat percakapannya berada (`CASEID`).
	//
	// Ia DIBAWA meski `ReplyKomunikasi` tidak memakainya, dan itu bukan kelebihan: ia
	// bagian dari pemagaran. Tanpa kunci klaim, sebuah nomor percakapan milik klaim lain
	// dapat dibalas lewat alamat klaim yang memang berhak dibuka pemanggil.
	ClaimKey string

	// ConversationID adalah nomor percakapan yang dibalas.
	ConversationID string

	// Message adalah isi balasannya.
	Message string

	// Replier adalah pembalasnya — login dan namanya.
	//
	// Keduanya disimpan: `REPLYFROM` menerima login, `REPLYFROMNAME` menerima nama. Di
	// Pega keduanya pun diambil dari dua properti berbeda (`pyUserIdentifier` dan
	// `pyUserName`), dan menyamakannya akan membuat kolom nama berisi kode.
	Replier Caller

	// RepliedAt adalah waktu balasannya.
	//
	// Ia lahir di SATU tempat — lapisan aplikasi — lalu diikat sebagai parameter, bukan
	// diisi `sysdate` di dalam pernyataan. `09-DATABASE-STRATEGY.md` §4 menuntutnya, dan
	// alasannya di sini nyata: waktu yang lahir di basis data tidak dapat diuji secara
	// deterministik, dan balasan pihak luar adalah hal yang paling mungkin dipersoalkan
	// kelak.
	RepliedAt time.Time
}

// AnsweredStatus adalah nilai `KOMUNIKASISTATUS` yang ditulis sebuah balasan.
//
// Ia sama dengan CommunicationAnswered, dan ditulis terpisah di sini supaya terbaca bahwa
// penandanya DIPAKAI MENULIS — bukan hanya menyaring.
const AnsweredStatus = CommunicationAnswered

// NewReplyCommand membentuk balasan yang sah, atau menyatakan apa yang salah.
//
// Seluruh pelanggaran dikumpulkan, bukan yang pertama saja (`P-5`).
func NewReplyCommand(
	claimKey string,
	input ReplyInput,
	replier Caller,
	now time.Time,
) (ReplyCommand, error) {
	cleanReplier := replier.Clean()
	if cleanReplier.Login == "" {
		return ReplyCommand{}, ErrCallerUnknown
	}

	key := strings.TrimSpace(claimKey)
	if key == "" {
		return ReplyCommand{}, ErrRowNotFound
	}

	conversation := strings.TrimSpace(input.ConversationID)
	message := strings.TrimSpace(input.Message)

	violations := []Violation{}

	if conversation == "" {
		violations = append(violations, Violation{
			Field:   FieldConversation,
			Message: "Percakapan yang dibalas tidak disebutkan.",
		})
	}

	if message == "" {
		violations = append(violations, Violation{
			Field:   FieldReply,
			Message: "Balasan tidak boleh kosong.",
		})
	}

	// Dihitung dalam RUNE, bukan bita. Pembacanya mengetik dalam bahasa Indonesia dan
	// Inggris, tetapi kolomnya dapat menerima huruf apa pun; menghitung bita akan menolak
	// kalimat yang lebih pendek daripada batasnya hanya karena hurufnya bukan ASCII.
	if length := len([]rune(message)); length > maxReplyLength {
		violations = append(violations, Violation{
			Field: FieldReply,
			Message: "Balasan terlalu panjang — " + strconv.Itoa(length) +
				" karakter, sementara batasnya " + strconv.Itoa(maxReplyLength) + ".",
		})
	}

	if len(violations) > 0 {
		return ReplyCommand{}, NewValidationError(violations)
	}

	// Nama pembalas boleh kosong, dan itu keputusan sadar.
	//
	// `REPLYFROMNAME` hanya keterangan yang digambar; `REPLYFROM` yang mengikat
	// identitasnya. Menolak balasan karena profil pemanggil tidak memuat nama akan
	// menghalangi pekerjaan demi sebuah kolom tampilan — dan login-nya sudah tersimpan.
	return ReplyCommand{
		ClaimKey:       key,
		ConversationID: conversation,
		Message:        message,
		Replier:        cleanReplier,
		RepliedAt:      now.UTC(),
	}, nil
}

// ReplierName mengembalikan nama yang ditulis ke `REPLYFROMNAME`.
//
// Bila profil pemanggil tidak memuat nama, LOGIN-nya yang dipakai — bukan teks kosong.
// Kolom nama yang kosong membuat petugas internal membaca balasan tanpa tahu dari siapa,
// sementara login selalu ada dan selalu dapat ditelusuri.
func (c ReplyCommand) ReplierName() string {
	if name := strings.TrimSpace(c.Replier.Name); name != "" {
		return name
	}
	return c.Replier.Login
}
