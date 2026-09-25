package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Repo memenuhi seam inboxkomunikasicabang.Repo dengan SQL terhadap Oracle.
//
// Satu instans Repo selalu terikat pada SATU basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat kueri (`ADR-0030`). Tidak ada satu pun kueri di
// baliknya yang menyaring menurut entitas, dan memang tidak boleh ada.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan; db wajib sudah terhubung.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengembalikan satu halaman percakapan yang cocok beserta jumlah seluruhnya.
//
// Kueri dipilih dari SIFAT tab, bukan dari kodenya. Menuliskan `if tab.Code == "2"` di sini
// akan membuat penambahan tab kelak menuntut suntingan di dua tempat — dan yang satu akan
// terlewat.
func (r *Repo) List(
	ctx context.Context,
	query inboxkomunikasicabang.Query,
	page inboxkomunikasicabang.Pagination,
) (inboxkomunikasicabang.Page, error) {
	clean := page.Normalize()

	name := "list_not_answered"
	if query.Tab.Answered {
		name = "list_answered"
	}

	code := strings.TrimSpace(query.Branch.Code)
	if code == "" {
		// Tidak pernah terjadi lewat ResolveBranch, yang selalu mengisi Code. Penjagaan ini
		// ada supaya Repo yang dipanggil dengan BranchFilter kosong — misalnya dari kode
		// yang ditulis kemudian — TIDAK diam-diam menjalankan kueri tanpa batas cabang.
		return inboxkomunikasicabang.Page{}, errors.New(
			"inboxkomunikasicabang/sqlstore: batas cabang kosong; daftar tidak dijalankan")
	}

	rows, err := r.db.QueryContext(
		ctx,
		getQuery(name),

		// Urutan argumen WAJIB sama dengan urutan KEMUNCULAN penanda di dalam teks kueri,
		// bukan dengan angka pada `:n`. Lihat CATATAN PENANDA BIND di berkas .sql.
		inboxkomunikasicabang.CaseOpen, // :1 kanal
		code,                           // :2 tujuan
		code,                           // :3 asal
		clean.Offset(),                 // :4 offset
		clean.Size,                     // :5 jumlah baris
	)
	if err != nil {
		return inboxkomunikasicabang.Page{}, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menjalankan %s: %w", name, err)
	}
	defer rows.Close()

	result := inboxkomunikasicabang.Page{
		Items:      []inboxkomunikasicabang.Conversation{},
		Pagination: clean,
	}

	for rows.Next() {
		item, total, err := scanConversation(rows)
		if err != nil {
			return inboxkomunikasicabang.Page{}, fmt.Errorf(
				"inboxkomunikasicabang/sqlstore: membaca baris %s: %w", name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxkomunikasicabang.Page{}, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menutup %s: %w", name, err)
	}

	return result, nil
}

// Summarize mengembalikan kedua pencacah pada batas cabang yang sama.
//
// Kedua angkanya diambil DUA kueri, sama seperti sistem lama — `PNCCountKomunikasiCabang_Act`
// menjalankan keduanya berurutan. Menggabungkannya menjadi satu kueri ber-`CASE` akan
// menghasilkan angka yang sama, tetapi menghilangkan kesejajaran satu-lawan-satu dengan
// rule aslinya; dan kedua penyaringnya memang BUKAN saling melengkapi, sehingga
// penggabungan menuntut penulisnya menyadari itu.
func (r *Repo) Summarize(
	ctx context.Context,
	filter inboxkomunikasicabang.BranchFilter,
) (inboxkomunikasicabang.Summary, error) {
	code := strings.TrimSpace(filter.Code)
	if code == "" {
		return inboxkomunikasicabang.Summary{}, errors.New(
			"inboxkomunikasicabang/sqlstore: batas cabang kosong; pencacah tidak dijalankan")
	}

	answered, err := r.count(ctx, "count_answered", code)
	if err != nil {
		return inboxkomunikasicabang.Summary{}, err
	}

	notAnswered, err := r.count(ctx, "count_not_answered", code)
	if err != nil {
		return inboxkomunikasicabang.Summary{}, err
	}

	return inboxkomunikasicabang.Summary{
		Answered:    answered,
		NotAnswered: notAnswered,
	}, nil
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name, code string) (int, error) {
	var total int
	err := r.db.QueryRowContext(
		ctx,
		getQuery(name),
		inboxkomunikasicabang.CaseOpen, // :1 kanal
		code,                           // :2 tujuan
		code,                           // :3 asal
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menjalankan %s: %w", name, err)
	}
	return total, nil
}

// Detail mengembalikan isi layar Detail Komunikasi untuk satu percakapan.
//
// Batas cabang ikut diberlakukan, dan itu bukan kelebihan kehati-hatian: nomor percakapan
// berurutan dan mudah ditebak, sehingga tanpa batas ini layar detail menjadi pintu samping
// ke percakapan cabang mana pun. Sistem lama tidak memeriksanya — tombolnya hanya ada pada
// baris yang sudah tersaring — tetapi tombol bukan penjagaan.
func (r *Repo) Detail(
	ctx context.Context,
	id string,
	filter inboxkomunikasicabang.BranchFilter,
) (inboxkomunikasicabang.ConversationDetail, error) {
	wanted := strings.TrimSpace(id)
	code := strings.TrimSpace(filter.Code)

	if wanted == "" || code == "" {
		return inboxkomunikasicabang.ConversationDetail{},
			inboxkomunikasicabang.ErrConversationNotFound
	}

	// KEBERADAAN diperiksa LEBIH DULU, terpisah dari utasnya.
	//
	// Sejak utas dibaca dari tabel riwayat, utas yang kosong tidak lagi berarti percakapannya
	// tidak ada: percakapan yang dibuat lewat jalur lain punya baris kepala tanpa satu pun
	// baris riwayat. Menyamakan keduanya akan menjawab "tidak ditemukan" untuk percakapan
	// yang nyata — dan itu akan dilaporkan sebagai kerusakan.
	origin, found, err := r.header(ctx, wanted, code)
	if err != nil {
		return inboxkomunikasicabang.ConversationDetail{}, err
	}
	if !found {
		// Tidak ada, ATAU milik cabang lain. Keduanya dijawab sama dengan sengaja: jawaban
		// yang membedakannya akan menyatakan bahwa nomor itu ada di tempat lain.
		return inboxkomunikasicabang.ConversationDetail{},
			inboxkomunikasicabang.ErrConversationNotFound
	}

	messages, err := r.thread(ctx, wanted, code)
	if err != nil {
		return inboxkomunikasicabang.ConversationDetail{}, err
	}

	attachments, err := r.attachments(ctx, wanted, code)
	if err != nil {
		return inboxkomunikasicabang.ConversationDetail{}, err
	}

	return inboxkomunikasicabang.ConversationDetail{
		ID:          wanted,
		Messages:    messages,
		Attachments: attachments,

		// Asal percakapan datang dari KEPALA, bukan dari utasnya. Tabel riwayat tidak memuat
		// kolom asal sama sekali.
		Origin: origin,
	}, nil
}

// header memeriksa keberadaan percakapan dan membaca asalnya.
//
// Nilai kedua menyatakan percakapannya ditemukan DAN terlihat oleh batas cabang yang
// diberikan. Keduanya dijadikan satu jawaban dengan sengaja — pemanggil tidak berhak
// membedakan "tidak ada" dari "bukan milik Anda".
func (r *Repo) header(ctx context.Context, id, code string) (string, bool, error) {
	var originCode, recipientCode sql.NullString

	err := r.db.QueryRowContext(
		ctx,
		getQuery("detail_header"),
		id,   // :1 nomor percakapan
		code, // :2 tujuan
		code, // :3 asal
	).Scan(&originCode, &recipientCode)

	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menjalankan detail_header: %w", err)
	}

	// Penerjemahan dikerjakan DOMAIN, bukan SQL — supaya penyimpanan ini dan penyimpanan
	// memori tidak dapat berselisih.
	return inboxkomunikasicabang.OriginOf(text(originCode)), true, nil
}

// thread membaca utas percakapan dari TABEL RIWAYAT.
//
// Bukan dari tabel percakapan. Setiap pesan dan setiap balasan adalah satu baris di sana,
// sehingga utas yang panjang terbaca utuh — sementara tabel percakapan hanya menyimpan pesan
// dan balasan TERAKHIR. Lihat catatan pada berkas .sql.
func (r *Repo) thread(
	ctx context.Context, id, code string,
) ([]inboxkomunikasicabang.ThreadMessage, error) {
	rows, err := r.db.QueryContext(
		ctx,
		getQuery("detail_thread"),
		id,   // :1 nomor percakapan
		id,   // :2 nomor percakapan (sisi gabungan)
		code, // :3 tujuan
		code, // :4 asal
	)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menjalankan detail_thread: %w", err)
	}
	defer func() { _ = rows.Close() }()

	messages := []inboxkomunikasicabang.ThreadMessage{}
	for rows.Next() {
		var createdAt, sender, message sql.NullString

		if err := rows.Scan(&createdAt, &sender, &message); err != nil {
			return nil, fmt.Errorf(
				"inboxkomunikasicabang/sqlstore: membaca baris detail_thread: %w", err)
		}

		messages = append(messages, inboxkomunikasicabang.ThreadMessage{
			CreatedAt:      text(createdAt),
			SenderOperator: text(sender),
			Message:        text(message),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menutup detail_thread: %w", err)
	}

	return messages, nil
}

func (r *Repo) attachments(
	ctx context.Context, id, code string,
) ([]inboxkomunikasicabang.Attachment, error) {
	rows, err := r.db.QueryContext(
		ctx,
		getQuery("detail_attachments"),
		id,   // :1 nomor percakapan
		code, // :2 tujuan
		code, // :3 asal
	)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menjalankan detail_attachments: %w", err)
	}
	defer rows.Close()

	result := []inboxkomunikasicabang.Attachment{}
	for rows.Next() {
		var documentID, typeName, detailName, note, uploadedAt sql.NullString

		if err := rows.Scan(
			&documentID, &typeName, &detailName, &note, &uploadedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxkomunikasicabang/sqlstore: membaca baris detail_attachments: %w", err)
		}

		result = append(result, inboxkomunikasicabang.Attachment{
			DocumentID: text(documentID),
			TypeName:   text(typeName),
			DetailName: text(detailName),
			Note:       text(note),
			UploadedAt: text(uploadedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menutup detail_attachments: %w", err)
	}

	return result, nil
}

// Reply menyimpan balasan atas sebuah percakapan.
//
// # Kedua penulisannya dibungkus SATU transaksi, berbeda dari sistem lama
//
// `PNCReplyMessageCabang` menjalankan keduanya sebagai dua langkah activity terpisah tanpa
// transaksi apa pun (langkah 3 lalu langkah 4). Kegagalan pada langkah kedua meninggalkan
// balasan yang tersimpan tanpa riwayat, dan tidak ada apa pun yang memulihkannya.
//
// Di sini keduanya atomik. Itu mengikuti preseden `D-68`, yang membuat `B-4` dan `B-9`
// atomik dengan alasan yang sama — dan seperti di sana, ia MENGUBAH keadaan akhir saat
// gagal: sistem lama meninggalkan sebagian, sistem ini tidak meninggalkan apa pun.
//
// Transaksinya dimulai di sini, BUKAN di lapisan aplikasi, dan itu pengecualian yang
// disadari terhadap `08-TECHNICAL-STRATEGY.md` §4.5. Alasannya: keduanya satu tindakan
// bisnis yang tidak pernah dipakai terpisah, dan menaikkannya ke usecase berarti membocorkan
// `*sql.Tx` melewati seam Repo — yang justru menghapus gunanya seam itu.
func (r *Repo) Reply(
	ctx context.Context,
	command inboxkomunikasicabang.ReplyCommand,
	filter inboxkomunikasicabang.BranchFilter,
) error {
	code := strings.TrimSpace(filter.Code)
	if code == "" {
		return errors.New(
			"inboxkomunikasicabang/sqlstore: batas cabang kosong; balasan tidak disimpan")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: memulai transaksi balasan: %w", err)
	}
	// Rollback dipanggil tanpa syarat. Setelah Commit berhasil ia tidak melakukan apa pun,
	// sehingga menaruhnya di defer menutup SETIAP jalur keluar — termasuk yang ditambahkan
	// kemudian oleh orang yang lupa.
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(
		ctx,
		getQuery("reply_update"),
		command.Message,                      // :1 isi balasan
		command.Caller.Login,                 // :2 penjawab
		command.RepliedAt,                    // :3 waktu balasan
		command.Caller.Name,                  // :4 nama penjawab
		inboxkomunikasicabang.StatusAnswered, // :5 status
		command.ID,                           // :6 nomor percakapan
		inboxkomunikasicabang.CaseOpen,       // :7 kanal berjalan
		code,                                 // :8 tujuan
		code,                                 // :9 asal
	)
	if err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: menyimpan balasan: %w", err)
	}

	// Nol baris berarti percakapannya tidak ada, SUDAH DITUTUP, atau milik cabang lain.
	// Ketiganya dijawab sama, dan itu disengaja — jawaban yang membedakannya akan menyatakan
	// bahwa nomor itu ada di tempat lain.
	//
	// Memeriksanya PENTING: tanpa ini, balasan atas nomor yang tidak ada akan dilaporkan
	// berhasil, dan pengguna mengira pesannya terkirim.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: membaca hasil balasan: %w", err)
	}
	if affected == 0 {
		return inboxkomunikasicabang.ErrConversationNotFound
	}

	if _, err := tx.ExecContext(
		ctx,
		getQuery("reply_history_insert"),
		command.Caller.Login, // :1 pengirim
		command.Message,      // :2 isi balasan
		command.ID,           // :3 nomor percakapan
		// :4 penanda kanal — kolomnya bernama `kodecabang` tetapi menerima CASEID; lihat
		// catatan pada berkas .sql.
		inboxkomunikasicabang.CaseOpen,
	); err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: mencatat riwayat balasan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: menutup transaksi balasan: %w", err)
	}
	return nil
}

// Finish menutup sebuah percakapan.
//
// Ia satu pernyataan, sehingga tidak menuntut transaksi.
func (r *Repo) Finish(
	ctx context.Context,
	id string,
	filter inboxkomunikasicabang.BranchFilter,
) error {
	code := strings.TrimSpace(filter.Code)
	wanted := strings.TrimSpace(id)

	if code == "" || wanted == "" {
		return inboxkomunikasicabang.ErrConversationNotFound
	}

	result, err := r.db.ExecContext(
		ctx,
		getQuery("finish_update"),
		inboxkomunikasicabang.CaseClosed, // :1 kanal penutup
		inboxkomunikasicabang.CaseOpen,   // :2 kanal berjalan
		wanted,                           // :3 nomor percakapan
		code,                             // :4 tujuan
		code,                             // :5 asal
	)
	if err != nil {
		return fmt.Errorf("inboxkomunikasicabang/sqlstore: menutup percakapan: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: membaca hasil penutupan: %w", err)
	}

	// Nol baris punya TIGA sebab yang mungkin: percakapannya tidak ada, milik cabang lain,
	// atau SUDAH ditutup. Ketiganya dijawab ErrConversationNotFound.
	//
	// Yang ketiga patut disadari: menekan tombolnya dua kali menghasilkan "tidak ditemukan"
	// pada penekanan kedua, bukan "berhasil". Itu lebih jujur daripada melaporkan berhasil
	// atas baris yang tidak berubah — dan barisnya memang sudah hilang dari layar.
	if affected == 0 {
		return inboxkomunikasicabang.ErrConversationNotFound
	}
	return nil
}

// CheckTable memastikan tabel yang dibutuhkan dapat dibaca akun aplikasi.
//
// Kedua pemeriksaan DIPISAH karena kegagalannya berbeda artinya, dan perbaikannya menempuh
// orang yang berbeda: yang pertama menunjuk tabel percakapan — tanpanya layar tidak dapat
// dipakai sama sekali; yang kedua menunjuk tabel lampiran atau salah satu master jenis
// dokumen — tanpanya layar tetap dapat dipakai, hanya lampirannya yang kosong.
//
// Menyatukannya akan membuat gangguan kecil terbaca sama gawatnya dengan gangguan besar.
func (r *Repo) CheckTable(ctx context.Context) error {
	var total int

	// Sentinelnya TEKS di sini, dan itu benar: yang disaring `CASEID`, kolom kanal
	// percakapan yang memang berisi teks (`CABANG`).
	err := r.db.QueryRowContext(
		ctx, getQuery("check_table"), "__periksa__",
	).Scan(&total)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: POOLDATA.M_KOMUNIKASI_PNC tidak dapat dibaca: %w",
			err)
	}

	// Tabel RIWAYAT — sumber utas layar detail, dan satu-satunya tempat nama kolom ditebak.
	//
	// Ia diperiksa SEBELUM lampiran karena akibatnya lebih besar: tanpa riwayat, layar detail
	// tidak menampilkan satu pun ucapan.
	// Sentinelnya ANGKA, bukan teks: `KOMUNIKASIID` pada tabel ini bertipe NUMBER, dan
	// sentinel bertipe teks menghasilkan `ORA-01722` — galat yang terbaca seolah tabelnya
	// bermasalah padahal yang salah pemeriksanya sendiri.
	err = r.db.QueryRowContext(
		ctx, getQuery("check_history_table"), 0,
	).Scan(&total)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: POOLDATA.M_KOMUNIKASI_CABANG tidak dapat dibaca "+
				"(nama kolom tanggalnya DITEBAK): %w", err)
	}

	// Sentinelnya ANGKA, sama seperti pemeriksa riwayat: `KOMUNIKASI_ID` bertipe NUMBER.
	//
	// Sampai 2026-09-25 ia mengirim teks, sehingga pemeriksaan ini SELALU gagal dengan
	// `ORA-01722` — dan kegagalannya terbaca seolah ketiga tabel lampiran tidak dapat
	// dibaca. Cacatnya ada sejak pemeriksa ini ditulis, dan baru terlihat ketika
	// `-periksa` benar-benar dijalankan terhadap basis data sungguhan.
	err = r.db.QueryRowContext(
		ctx, getQuery("check_attachment_table"), 0,
	).Scan(&total)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: POOLDATA.D_KOMUNIKASI_PNC, V_LST_DOC_TYPE, "+
				"atau V_LST_DET_TYPE_DOC tidak dapat dibaca: %w", err)
	}

	return nil
}

// scanConversation membaca satu baris daftar beserta jumlah seluruh barisnya.
//
// Urutan pemindaian WAJIB sama dengan urutan kolom di kedua kueri daftar, dan dengan
// listColumns. Ketiganya diuji kesesuaiannya di query_test.go — sebuah kolom yang tertukar
// di sini tidak menghasilkan satu pun galat bila tipenya sama, dan seluruh kolom di sini
// bertipe teks.
func scanConversation(rows *sql.Rows) (inboxkomunikasicabang.Conversation, int, error) {
	var id, createdAt, originCode, sender, message sql.NullString
	var reply, replierName, recipientCode, status, repliedAt sql.NullString
	var total int

	if err := rows.Scan(
		&id, &createdAt, &originCode, &sender, &message,
		&reply, &replierName, &recipientCode, &status, &repliedAt,
		&total,
	); err != nil {
		return inboxkomunikasicabang.Conversation{}, 0, err
	}

	return inboxkomunikasicabang.Conversation{
		ID:        text(id),
		CreatedAt: text(createdAt),

		// Kedua penerjemah asal dipanggil DI SINI, di lapisan yang sama dengan penyimpanan
		// memori memanggilnya. Menuliskannya sebagai `CASE` di dalam SQL akan membuat kedua
		// pengisi seam punya penerjemah masing-masing, dan keduanya dapat menyimpang.
		SenderOrigin:    inboxkomunikasicabang.OriginOf(text(originCode)),
		SenderOperator:  text(sender),
		Message:         text(message),
		Reply:           text(reply),
		ReplierName:     text(replierName),
		RecipientOrigin: inboxkomunikasicabang.RecipientOf(text(recipientCode)),
		Status:          text(status),
		RepliedAt:       text(repliedAt),
	}, total, nil
}

// text memangkas spasi nilai kolom yang boleh NULL.
//
// Pemangkasan BUKAN kerapian: kolom `CHAR` berlebar tetap memadatkan nilainya dengan spasi
// tanpa memberi tanda apa pun, dan kode cabang yang tidak dipangkas tidak akan pernah cocok
// dengan penyaringnya sendiri. Di sini ia juga menentukan hasil penerjemahan asal — `"1 "`
// dan `"1"` harus menghasilkan teks yang sama.
func text(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

var _ inboxkomunikasicabang.Repo = (*Repo)(nil)
