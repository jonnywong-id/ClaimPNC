package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxpladlapredla"
)

// Repo membaca antrean PLA/DLA/Pre-DLA pada SATU basis data entitas.
//
// # Ia TIDAK MENULIS satu baris pun
//
// Ketiga tabel dokumennya masih ditulis Pega lewat tombol "Send", yang belum dibangun di
// sini. `P-1` karena itu terpenuhi dengan cara yang paling sederhana: kepemilikan
// tabelnya belum berpindah sama sekali.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// listQueryFor memilih kueri daftar menurut jenis dokumen tabnya.
//
// Ia memetakan dari AdviceKind, bukan dari Tab.Code, dan itu disengaja: kode tab adalah
// kontrak ke layar dan boleh berubah kata-katanya, sedangkan jenis dokumen menunjuk tabel
// di basis data. Memetakan dari kode tab berarti mengganti judul tab dapat memutus
// kuerinya.
func listQueryFor(kind inboxpladlapredla.AdviceKind) (string, error) {
	switch kind {
	case inboxpladlapredla.KindPLA:
		return "list_pla", nil
	case inboxpladlapredla.KindDLA:
		return "list_dla", nil
	case inboxpladlapredla.KindPreDLA:
		return "list_pre_dla", nil
	default:
		return "", fmt.Errorf(
			"inboxpladlapredla/sqlstore: jenis dokumen %q tidak dikenal", kind)
	}
}

// documentQueryFor memilih kueri rincian menurut jenis dokumen tabnya.
//
// Pre-DLA sengaja TIDAK punya pasangan: tabnya tidak menggambar grid rincian di Pega, dan
// usecase sudah menolaknya lebih dulu. Cabang default di sini adalah jaring pengaman
// terakhir, bukan jalur yang diharapkan pernah terpakai.
func documentQueryFor(kind inboxpladlapredla.AdviceKind) (string, error) {
	switch kind {
	case inboxpladlapredla.KindPLA:
		return "documents_pla", nil
	case inboxpladlapredla.KindDLA:
		return "documents_dla", nil
	default:
		return "", fmt.Errorf(
			"inboxpladlapredla/sqlstore: %q tidak punya grid rincian", kind)
	}
}

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (r *Repo) List(
	ctx context.Context,
	q inboxpladlapredla.Query,
	page inboxpladlapredla.Pagination,
) (inboxpladlapredla.Page, error) {
	name, err := listQueryFor(q.Tab.Kind)
	if err != nil {
		return inboxpladlapredla.Page{}, err
	}

	clean := page.Normalize()
	args := listArgs(q, clean)

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxpladlapredla.Page{}, fmt.Errorf(
			"inboxpladlapredla/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	result := inboxpladlapredla.Page{
		Pagination: clean,
		Items:      []inboxpladlapredla.Row{},
	}

	for rows.Next() {
		var (
			claimKey, claimNo, policyNo, insured sql.NullString
			picTeknik                            sql.NullString
			registerDate, lossDate, adviceDate   sql.NullTime
			total                                int
		)

		if err := rows.Scan(
			&claimKey, &claimNo, &policyNo, &insured,
			&registerDate, &lossDate, &picTeknik, &adviceDate,
			&total,
		); err != nil {
			return inboxpladlapredla.Page{}, fmt.Errorf(
				"inboxpladlapredla/sqlstore: %s: memindai baris: %w", name, err)
		}

		result.Items = append(result.Items, inboxpladlapredla.Row{
			ClaimKey:     strings.TrimSpace(claimKey.String),
			ClaimNo:      strings.TrimSpace(claimNo.String),
			PolicyNo:     strings.TrimSpace(policyNo.String),
			Insured:      strings.TrimSpace(insured.String),
			RegisterDate: dateText(registerDate),
			LossDate:     dateText(lossDate),
			PICTeknik:    strings.TrimSpace(picTeknik.String),
			AdviceDate:   dateText(adviceDate),
		})
		result.Total = total
	}

	if err := rows.Err(); err != nil {
		return inboxpladlapredla.Page{}, fmt.Errorf(
			"inboxpladlapredla/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return result, nil
}

// listArgs menyusun kedelapan bind kueri daftar.
//
// # Kenapa setiap nilai diikat DUA KALI
//
// Karena penanda kehadiran dan nilainya dipisah — `(:1 IS NULL OR … LIKE :2)` — dan
// keduanya memakai nomor bind yang BERBEDA meski nilainya sama. Penanda yang dipakai ulang
// dengan satu nomor berperilaku berbeda antar driver; dua nomor menghilangkan
// ketergantungan itu. Pola yang sama dipakai modul Archive Dokumen Klaim.
//
// # Kenapa penandanya bertipe TEKS, bahkan untuk rentang tanggal
//
// Supaya uji `IS NULL` tidak pernah jatuh pada bind bertipe tanggal. Bind yang HANYA
// muncul di dalam `IS NULL` tidak punya konteks tipe di dalam kueri, sehingga tipenya
// ditentukan driver — dan itu tempat galat yang hanya muncul di Oracle. Di sini bind
// tanggal selalu muncul di dalam perbandingan, tempat tipenya tidak mungkin ambigu.
func listArgs(
	q inboxpladlapredla.Query,
	page inboxpladlapredla.Pagination,
) []any {
	var (
		searchFlag sql.NullString
		pattern    sql.NullString
	)
	if q.Search != "" {
		searchFlag = sql.NullString{String: "1", Valid: true}
		pattern = sql.NullString{
			String: "%" + strings.ToUpper(escapeLike(q.Search)) + "%",
			Valid:  true,
		}
	}

	fromFlag, from := optionalDate(q.From)
	toFlag, to := optionalDate(q.To)

	return []any{
		searchFlag, pattern,
		fromFlag, from,
		toFlag, to,
		page.Offset(), page.Size,
	}
}

// optionalDate memecah satu batas rentang menjadi penanda kehadiran dan nilainya.
func optionalDate(value *time.Time) (sql.NullString, sql.NullTime) {
	if value == nil {
		return sql.NullString{}, sql.NullTime{}
	}
	return sql.NullString{String: "1", Valid: true},
		sql.NullTime{Time: *value, Valid: true}
}

// Documents mengembalikan isi grid rincian satu klaim.
//
// Ia menembak DUA kueri, dan urutannya penting: keberadaan klaimnya diperiksa LEBIH DULU.
// Tanpa itu, kunci yang salah dan klaim yang belum punya dokumen menghasilkan jawaban yang
// sama persis — daftar kosong — dan hanya yang pertama yang merupakan kekeliruan. Lihat
// catatan pada kueri `claim_exists`.
func (r *Repo) Documents(
	ctx context.Context,
	tab inboxpladlapredla.Tab,
	claimKey string,
) ([]inboxpladlapredla.Document, error) {
	key := strings.TrimSpace(claimKey)
	if key == "" {
		return nil, inboxpladlapredla.ErrRowNotFound
	}

	name, err := documentQueryFor(tab.Kind)
	if err != nil {
		return nil, err
	}

	var exists int
	err = r.db.QueryRowContext(ctx, query("claim_exists"), key).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, inboxpladlapredla.ErrRowNotFound
	case err != nil:
		return nil, fmt.Errorf(
			"inboxpladlapredla/sqlstore: claim_exists: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query(name), key)
	if err != nil {
		return nil, fmt.Errorf("inboxpladlapredla/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	items := []inboxpladlapredla.Document{}
	for rows.Next() {
		var (
			adviceNo, reinsurer, adviceType  sql.NullString
			revision, sent, notes, email     sql.NullString
			acceptanceNo                     sql.NullString
			adviceDate, sentDate, receivedAt sql.NullTime
		)

		if err := rows.Scan(
			&adviceNo, &reinsurer, &adviceType, &revision, &adviceDate,
			&sent, &sentDate, &receivedAt, &notes, &email, &acceptanceNo,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxpladlapredla/sqlstore: %s: memindai baris: %w", name, err)
		}

		items = append(items, inboxpladlapredla.Document{
			AdviceNo:     strings.TrimSpace(adviceNo.String),
			Reinsurer:    strings.TrimSpace(reinsurer.String),
			AdviceType:   strings.TrimSpace(adviceType.String),
			Revision:     strings.TrimSpace(revision.String),
			AdviceDate:   dateText(adviceDate),
			Sent:         strings.TrimSpace(sent.String),
			SentDate:     dateText(sentDate),
			ReceivedDate: dateText(receivedAt),
			Notes:        strings.TrimSpace(notes.String),
			Email:        strings.TrimSpace(email.String),
			AcceptanceNo: strings.TrimSpace(acceptanceNo.String),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladlapredla/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return items, nil
}

// sentDateOfPreDLAPanel menuliskan "Tgl Kirim" pada panel "Print Pre DLA".
//
// # Ia BERBEDA dari kolom tanggal lain di modul ini, dan itu bukan kekeliruan
//
// `Activity/PNCGetListPreDla-Act.xml` menambahkan **7 jam** ke kolom itu sebelum
// menggambarnya:
//
//	.TglDLA := @addToDate(.TglDLA, "0", "7", "0", "0")
//
// Tujuh jam adalah selisih GMT ke WIB. Artinya `TGLKIRIM` disimpan dalam GMT, dan panel
// itu menampilkannya dalam waktu Jakarta.
//
// # Yang membuatnya patut dicatat: Pega TIDAK konsisten
//
// Kolom `TGLKIRIM` yang SAMA digambar **tanpa** penambahan itu di grid rincian —
// `Activity/GetDetailPLAList-Act.xml` dan `GetDetailDLAList-Act.xml` keduanya memuat nol
// `addToDate`. Jadi satu kolom yang sama ditampilkan berbeda di dua layar yang
// bersebelahan.
//
// Perilakunya ditiru apa adanya (`P-5`), termasuk ketidakkonsistenannya. Menyeragamkannya
// akan menggeser tanggal pada salah satu dari dua layar, dan itu selisih yang belum
// diminta siapa pun.
//
// # Kenapa penambahannya TIDAK ditulis sebagai "+7 jam"
//
// `08-TECHNICAL-STRATEGY.md` §4.4 melarang penambahan 7 jam manual tanpa perkecualian —
// itulah utang teknis yang `F-5` hapus. Yang dilakukan di sini adalah **konversi zona
// waktu**: nilainya dibaca sebagai GMT lalu dinyatakan dalam `Asia/Jakarta`. Hasilnya
// sama dengan Pega hari ini, dan tetap benar bila aturan zona waktunya kelak berubah.
//
// # Yang belum diverifikasi
//
// Apakah driver mengembalikan nilai kolom ini sudah ber-zona atau polos belum diuji
// terhadap Oracle sungguhan — `R-08`, DDL-nya tidak ada di export. Karena itu nilainya
// diperlakukan sebagai GMT secara eksplisit, bukan dibiarkan mengikuti zona sesi basis
// data: zona sesi dapat berubah antar lingkungan tanpa satu pun tanda.
func sentDateOfPreDLAPanel(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}

	gmt := time.Date(
		value.Time.Year(), value.Time.Month(), value.Time.Day(),
		value.Time.Hour(), value.Time.Minute(), value.Time.Second(),
		value.Time.Nanosecond(), time.UTC,
	)

	return gmt.In(jakarta).Format(inboxpladlapredla.DateLayout)
}

// jakarta adalah zona waktu WIB, disebut NAMANYA — bukan offset tetap.
//
// Bila basis datanya tidak mengenal nama zona itu, ia jatuh ke UTC+7 tetap. Jatuhnya
// disengaja dan aman: WIB memang UTC+7 sepanjang tahun, tanpa waktu musim panas.
var jakarta = loadJakarta()

func loadJakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// dateText menuliskan tanggal yang dibaca basis data sebagai teks `YYYY-MM-DD`.
//
// Kolom yang kosong menghasilkan teks KOSONG, bukan `0001-01-01`. Perbedaannya nyata di
// layar ini: kolom tanggal advice memang kadang kosong — lihat catatan kejanggalan
// `ISKIRIM` pada inboxpladlapredla.sql — dan `0001-01-01` akan terbaca sebagai tanggal
// yang benar-benar tercatat.
//
// Bagian JAM dibuang. Tidak satu pun kolom tanggal di layar ini digambar beserta jamnya,
// dan membawanya hanya membuat dua nilai yang sama tampak berbeda saat dibandingkan
// sebagai teks.
func dateText(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(inboxpladlapredla.DateLayout)
}

// PrintPreDLA mengembalikan isi panel "Print Pre DLA" satu klaim.
//
// Sama dengan Documents, keberadaan klaimnya diperiksa LEBIH DULU. Di sini pemeriksaan itu
// justru lebih perlu, bukan kurang: panel ini punya DUA sebab berbeda untuk kosong yang
// keduanya sah — klaimnya belum punya Pre-DLA sama sekali, atau Pre-DLAnya ada tetapi
// belum satu pun lampirannya cocok. Tanpa pemeriksaan pendahuluan, kunci klaim yang keliru
// menjadi sebab KETIGA yang tidak terbedakan dari keduanya.
func (r *Repo) PrintPreDLA(
	ctx context.Context,
	claimKey string,
) ([]inboxpladlapredla.PreDLADocument, error) {
	key := strings.TrimSpace(claimKey)
	if key == "" {
		return nil, inboxpladlapredla.ErrRowNotFound
	}

	var exists int
	err := r.db.QueryRowContext(ctx, query("claim_exists"), key).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, inboxpladlapredla.ErrRowNotFound
	case err != nil:
		return nil, fmt.Errorf("inboxpladlapredla/sqlstore: claim_exists: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("print_pre_dla"), key)
	if err != nil {
		return nil, fmt.Errorf("inboxpladlapredla/sqlstore: print_pre_dla: %w", err)
	}
	defer rows.Close()

	items := []inboxpladlapredla.PreDLADocument{}
	for rows.Next() {
		var (
			adviceNo, reinsurer, adviceType sql.NullString
			sent, attachmentKey             sql.NullString
			sentDate                        sql.NullTime
		)

		if err := rows.Scan(
			&adviceNo, &reinsurer, &adviceType, &sentDate, &sent, &attachmentKey,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxpladlapredla/sqlstore: print_pre_dla: memindai baris: %w", err)
		}

		items = append(items, inboxpladlapredla.PreDLADocument{
			AdviceNo:      strings.TrimSpace(adviceNo.String),
			Reinsurer:     strings.TrimSpace(reinsurer.String),
			AdviceType:    strings.TrimSpace(adviceType.String),
			SentDate:      sentDateOfPreDLAPanel(sentDate),
			Sent:          strings.TrimSpace(sent.String),
			AttachmentKey: strings.TrimSpace(attachmentKey.String),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladlapredla/sqlstore: print_pre_dla: membaca hasil: %w", err)
	}

	return items, nil
}

// PrintPreDLADiagnosis adalah hitungan baris yang lolos tiap tahap penyaring panel.
type PrintPreDLADiagnosis struct {
	PreDLARows    int
	AdviceNoIs11  int
	HasAttachment int
	CategoryIsDLA int
}

// DiagnosePrintPreDLA menghitung di tahap mana baris panel "Print Pre DLA" gugur.
//
// # Kenapa ia BUKAN bagian dari seam inboxpladlapredla.Repo
//
// Karena ia bukan kebutuhan domain. Layar tidak pernah memanggilnya, dan pengisi memori
// tidak perlu memilikinya — menaruhnya di seam berarti memaksa setiap pengisi berikutnya
// menjawab pertanyaan yang hanya berarti bagi Oracle.
//
// Ia dipakai perintah pemeriksaan kesiapan, dan hanya di sana.
func (r *Repo) DiagnosePrintPreDLA(
	ctx context.Context,
	claimKey string,
) (PrintPreDLADiagnosis, error) {
	var out PrintPreDLADiagnosis

	err := r.db.QueryRowContext(
		ctx, query("print_pre_dla_diagnosa"), strings.TrimSpace(claimKey),
	).Scan(&out.PreDLARows, &out.AdviceNoIs11, &out.HasAttachment, &out.CategoryIsDLA)
	if err != nil {
		return PrintPreDLADiagnosis{}, fmt.Errorf(
			"inboxpladlapredla/sqlstore: print_pre_dla_diagnosa: %w", err)
	}

	return out, nil
}

// MarkPreDLASent menandai satu Pre-DLA sebagai terkirim.
//
// Mengembalikan `false` bila tidak ada baris yang berubah — Pre-DLA itu sudah terkirim,
// atau nomornya tidak ada pada klaim itu. Keduanya tidak dibedakan di sini, dan itu
// disengaja: pemanggil memeriksa keberadaan klaimnya lebih dulu, dan nomor Pre-DLA yang
// tidak ada pada klaim yang ada hanya dapat berasal dari layar yang sudah basi — yang
// jawabannya sama dengan "sudah terkirim": muat ulang panelnya.
func (r *Repo) MarkPreDLASent(
	ctx context.Context,
	claimKey, adviceNo string,
) (bool, error) {
	key := strings.TrimSpace(claimKey)
	no := strings.TrimSpace(adviceNo)
	if key == "" || no == "" {
		return false, inboxpladlapredla.ErrRowNotFound
	}

	hasil, err := r.db.ExecContext(ctx, query("mark_pre_dla_sent"), key, no)
	if err != nil {
		return false, fmt.Errorf(
			"inboxpladlapredla/sqlstore: mark_pre_dla_sent: %w", err)
	}

	// Jumlah baris terpengaruh adalah SATU-SATUNYA cara membedakan "baru saja ditandai"
	// dari "sudah ditandai sebelumnya" — penyaring `ISKIRIM <> '1'` pada kuerinya yang
	// membuat perbedaan itu ada. Driver yang tidak mendukungnya akan menghasilkan galat,
	// bukan diam-diam mengembalikan nol.
	baris, err := hasil.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"inboxpladlapredla/sqlstore: mark_pre_dla_sent: membaca jumlah baris: %w",
			err)
	}

	return baris > 0, nil
}

// sendableColumns adalah alias yang dikembalikan kedua kueri pembaca dokumen kirim.
var sendableColumns = []string{
	"ADVICE_NO", "ADVICE_TYPE", "REINSURER", "REINSURER_ID", "EMAIL", "SENT",
	"REINSURER_LOGIN", "COUNTRY", "CLAIM_NO", "POLICY_NO", "INSURED", "BUSINESS",
	"LOSS_DATE",
}

// AdviceForSending membaca satu PLA/DLA beserta keterangan reasuradur dan klaimnya.
func (r *Repo) AdviceForSending(
	ctx context.Context,
	tab inboxpladlapredla.Tab,
	claimKey, adviceNo string,
) (inboxpladlapredla.SendableAdvice, inboxpladlapredla.ClaimSummary, error) {
	var (
		advice inboxpladlapredla.SendableAdvice
		claim  inboxpladlapredla.ClaimSummary
	)

	key := strings.TrimSpace(claimKey)
	no := strings.TrimSpace(adviceNo)
	if key == "" || no == "" {
		return advice, claim, inboxpladlapredla.ErrRowNotFound
	}

	name, err := sendQueryFor(tab.Kind)
	if err != nil {
		return advice, claim, err
	}

	var (
		adviceNoCol, adviceType, reinsurer, reinsurerID sql.NullString
		email, sent, login, country                     sql.NullString
		claimNo, policyNo, insured, business            sql.NullString
		lossDate                                        sql.NullTime
	)

	err = r.db.QueryRowContext(ctx, query(name), key, no).Scan(
		&adviceNoCol, &adviceType, &reinsurer, &reinsurerID, &email, &sent,
		&login, &country, &claimNo, &policyNo, &insured, &business, &lossDate,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return advice, claim, inboxpladlapredla.ErrRowNotFound
	case err != nil:
		return advice, claim, fmt.Errorf(
			"inboxpladlapredla/sqlstore: %s: %w", name, err)
	}

	advice = inboxpladlapredla.SendableAdvice{
		AdviceNo:    strings.TrimSpace(adviceNoCol.String),
		AdviceType:  strings.TrimSpace(adviceType.String),
		Reinsurer:   strings.TrimSpace(reinsurer.String),
		ReinsurerID: strings.TrimSpace(reinsurerID.String),
		Email:       strings.TrimSpace(email.String),
		Sent:        strings.TrimSpace(sent.String),
		Login:       strings.TrimSpace(login.String),
		Country:     strings.TrimSpace(country.String),
	}

	claim = inboxpladlapredla.ClaimSummary{
		ClaimNo:  strings.TrimSpace(claimNo.String),
		PolicyNo: strings.TrimSpace(policyNo.String),
		Insured:  strings.TrimSpace(insured.String),
		Business: strings.TrimSpace(business.String),
		LossDate: dateText(lossDate),
	}

	return advice, claim, nil
}

// sendQueryFor memilih kueri pembaca menurut jenis dokumennya.
func sendQueryFor(kind inboxpladlapredla.AdviceKind) (string, error) {
	switch kind {
	case inboxpladlapredla.KindPLA:
		return "advice_for_sending_pla", nil
	case inboxpladlapredla.KindDLA:
		return "advice_for_sending_dla", nil
	default:
		return "", inboxpladlapredla.ErrDocumentsNotOnTab
	}
}

// AttachmentsForClaim membaca berkas lampiran satu klaim pada satu kategori.
func (r *Repo) AttachmentsForClaim(
	ctx context.Context,
	claimKey, category string,
) ([]inboxpladlapredla.Attachment, error) {
	key := strings.TrimSpace(claimKey)
	if key == "" {
		return nil, inboxpladlapredla.ErrRowNotFound
	}

	rows, err := r.db.QueryContext(
		ctx, query("attachments_for_claim"), key, strings.TrimSpace(category))
	if err != nil {
		return nil, fmt.Errorf(
			"inboxpladlapredla/sqlstore: attachments_for_claim: %w", err)
	}
	defer rows.Close()

	items := []inboxpladlapredla.Attachment{}
	for rows.Next() {
		var (
			name, mime sql.NullString
			content    []byte
		)
		if err := rows.Scan(&name, &mime, &content); err != nil {
			return nil, fmt.Errorf(
				"inboxpladlapredla/sqlstore: attachments_for_claim: memindai baris: %w",
				err)
		}

		// Lampiran TANPA isi dilewati, bukan dikirim sebagai berkas kosong.
		//
		// Berkas kosong yang sampai ke reasuradur tampak seperti dokumen yang rusak, dan
		// yang menerimanya tidak dapat membedakannya dari kegagalan pengiriman.
		if len(content) == 0 {
			continue
		}

		items = append(items, inboxpladlapredla.Attachment{
			Name:     strings.TrimSpace(name.String),
			MIMEType: strings.TrimSpace(mime.String),
			Content:  content,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladlapredla/sqlstore: attachments_for_claim: membaca hasil: %w", err)
	}

	return items, nil
}

// MarkAdviceSent menandai satu PLA/DLA terkirim dan memperbarui master reasuransi.
//
// # Keduanya dalam SATU transaksi
//
// Mereka menggambarkan satu peristiwa yang sama. Sistem lama menempuhnya dengan dua
// pernyataan terpisah, dan `Database/UPDATEREAS.prc` menambahkan empat `COMMIT` di
// dalamnya — sehingga kegagalan di tengah meninggalkan keadaan separuh jalan yang tidak
// dapat dibedakan dari keberhasilan. `D-68` melepaskan modul ini dari pola itu.
func (r *Repo) MarkAdviceSent(
	ctx context.Context,
	tab inboxpladlapredla.Tab,
	claimKey, adviceNo string,
	advice inboxpladlapredla.SendableAdvice,
) (bool, error) {
	key := strings.TrimSpace(claimKey)
	no := strings.TrimSpace(adviceNo)
	if key == "" || no == "" {
		return false, inboxpladlapredla.ErrRowNotFound
	}

	name, err := markQueryFor(tab.Kind)
	if err != nil {
		return false, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf(
			"inboxpladlapredla/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	hasil, err := tx.ExecContext(ctx, query(name), key, no, advice.Email)
	if err != nil {
		return false, fmt.Errorf("inboxpladlapredla/sqlstore: %s: %w", name, err)
	}

	baris, err := hasil.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"inboxpladlapredla/sqlstore: %s: membaca jumlah baris: %w", name, err)
	}
	if baris == 0 {
		// Dokumennya sudah terkirim. Transaksinya dibatalkan, dan master reasuransi
		// TIDAK ikut diperbarui — memperbaruinya berarti mengubah master atas peristiwa
		// yang tidak terjadi.
		return false, nil
	}

	// Master reasuransi diperbarui hanya bila kuncinya lengkap.
	//
	// Baris dokumen yang belum menunjuk kode reasuradur tetap dapat dikirim — alamatnya
	// ada di baris itu sendiri — dan mencoba memperbarui master dengan kunci kosong akan
	// menyentuh baris yang tidak dimaksud siapa pun.
	if advice.ReinsurerID != "" && advice.Reinsurer != "" && advice.AdviceNo != "" {
		tipe := advice.AdviceNo[:1]

		if _, err := tx.ExecContext(ctx, query("update_reinsurer_email"),
			advice.Email, advice.ReinsurerID, advice.Reinsurer, tipe); err != nil {
			return false, fmt.Errorf(
				"inboxpladlapredla/sqlstore: update_reinsurer_email: %w", err)
		}
		// Nol baris terpengaruh BUKAN galat — reasuradurnya belum terdaftar di master.
		// Lihat catatan pada kuerinya.
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf(
			"inboxpladlapredla/sqlstore: menyimpan transaksi: %w", err)
	}

	return true, nil
}

// markQueryFor memilih kueri penanda menurut jenis dokumennya.
func markQueryFor(kind inboxpladlapredla.AdviceKind) (string, error) {
	switch kind {
	case inboxpladlapredla.KindPLA:
		return "mark_advice_sent_pla", nil
	case inboxpladlapredla.KindDLA:
		return "mark_advice_sent_dla", nil
	default:
		return "", inboxpladlapredla.ErrDocumentsNotOnTab
	}
}
