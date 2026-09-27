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
			SentDate:      dateText(sentDate),
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
