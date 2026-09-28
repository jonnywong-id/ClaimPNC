package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxpladla"
)

// Repo membaca daftar PLA/DLA milik reasuradur pada SATU basis data entitas.
//
// Ia TIDAK MENULIS satu baris pun; layarnya pun baca-saja di Pega.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ReinsurerCodes mengembalikan kode reasuradur milik satu login, terurut MENURUN.
//
// Senarai kosong berarti loginnya bukan reasuradur. Ia dikembalikan sebagai senarai
// kosong tanpa galat — yang menerjemahkannya menjadi pesan bagi pengguna adalah usecase,
// karena keputusan "ini bukan galat teknis melainkan keadaan yang perlu dijelaskan" adalah
// keputusan bisnis.
func (r *Repo) ReinsurerCodes(ctx context.Context, login string) ([]string, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, query("reinsurer_codes"), clean)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: reinsurer_codes: %w", err)
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var code sql.NullString
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: reinsurer_codes: memindai baris: %w", err)
		}
		if trimmed := strings.TrimSpace(code.String); trimmed != "" {
			codes = append(codes, trimmed)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: reinsurer_codes: membaca hasil: %w", err)
	}

	return codes, nil
}

// listQueryFor memilih kueri daftar dan kueri ringkas menurut kode tabnya.
//
// Keduanya dipilih BERSAMAAN, dan itu disengaja: penyaring daftar dan penyaring tabel
// ringkas harus selalu sepasang. Memisahkan pemilihannya memungkinkan salah satu berpindah
// tanpa yang lain — dan angka ringkas yang menghitung populasi berbeda dari daftarnya
// adalah hal pertama yang dilaporkan pengguna sebagai kerusakan.
func queriesFor(tab inboxpladla.Tab) (list string, count string, err error) {
	switch tab.Code {
	case inboxpladla.TabPLA:
		return "list_pla", "count_pla", nil
	case inboxpladla.TabPLADLA:
		return "list_dla", "count_dla", nil
	case inboxpladla.TabClose:
		return "list_close", "count_close", nil

	case inboxpladla.TabInbound:
		return "list_komunikasi_recipient", "count_komunikasi_recipient", nil

	// Kedua tab di bawah memakai kueri yang SAMA, dan yang membedakannya adalah NILAI
	// bind status — `0` belum dijawab, `1` sudah. Lihat komunikasi.sql.
	case inboxpladla.TabOutbound, inboxpladla.TabAnswered:
		return "list_komunikasi_sender", "count_komunikasi_sender", nil

	default:
		return "", "", fmt.Errorf(
			"inboxpladla/sqlstore: daftar %q tidak dikenal", tab.Code)
	}
}

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (r *Repo) List(
	ctx context.Context,
	q inboxpladla.Query,
	page inboxpladla.Pagination,
) (inboxpladla.Page, error) {
	name, _, err := queriesFor(q.Tab)
	if err != nil {
		return inboxpladla.Page{}, err
	}

	clean := page.Normalize()
	args := append(filterArgs(q), clean.Offset(), clean.Size)

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxpladla.Page{}, fmt.Errorf(
			"inboxpladla/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	result := inboxpladla.Page{Pagination: clean, Items: []inboxpladla.Row{}}

	for rows.Next() {
		var (
			claimKey, claimNo, policyNo, insured sql.NullString
			businessName, picTeknik              sql.NullString
			statusCode, statusLabel              sql.NullString
			adviceNo, closeNote                  sql.NullString
			registerDate, lossDate               sql.NullTime
			total                                int
		)

		if err := rows.Scan(
			&claimKey, &claimNo, &policyNo, &insured, &businessName,
			&registerDate, &lossDate, &picTeknik,
			&statusCode, &statusLabel, &adviceNo, &closeNote,
			&total,
		); err != nil {
			return inboxpladla.Page{}, fmt.Errorf(
				"inboxpladla/sqlstore: %s: memindai baris: %w", name, err)
		}

		result.Items = append(result.Items, inboxpladla.Row{
			ClaimKey:     strings.TrimSpace(claimKey.String),
			ClaimNo:      strings.TrimSpace(claimNo.String),
			PolicyNo:     strings.TrimSpace(policyNo.String),
			Insured:      strings.TrimSpace(insured.String),
			BusinessName: strings.TrimSpace(businessName.String),
			RegisterDate: dateText(registerDate),
			LossDate:     dateText(lossDate),
			PICTeknik:    strings.TrimSpace(picTeknik.String),
			StatusCode:   strings.TrimSpace(statusCode.String),
			StatusLabel:  strings.TrimSpace(statusLabel.String),
			AdviceNo:     strings.TrimSpace(adviceNo.String),
			CloseNote:    strings.TrimSpace(closeNote.String),
		})
		result.Total = total
	}

	if err := rows.Err(); err != nil {
		return inboxpladla.Page{}, fmt.Errorf(
			"inboxpladla/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return result, nil
}

// Counts mengembalikan tabel ringkas "Status / Jumlah".
func (r *Repo) Counts(
	ctx context.Context,
	q inboxpladla.Query,
) ([]inboxpladla.StatusCount, error) {
	_, name, err := queriesFor(q.Tab)
	if err != nil {
		return nil, err
	}

	// Kueri ringkas memakai penyaring yang SAMA, tetapi tanpa paginasi — dan karena
	// bind login-nya lebih sedikit satu pada tab PLA (kolom "No PLA" tidak diambil),
	// susunannya dipangkas di countArgs, bukan disalin ulang.
	rows, err := r.db.QueryContext(ctx, query(name), countArgs(q)...)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	counts := []inboxpladla.StatusCount{}
	for rows.Next() {
		var (
			code, label sql.NullString
			total       int
		)
		if err := rows.Scan(&code, &label, &total); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: %s: memindai baris: %w", name, err)
		}

		counts = append(counts, inboxpladla.StatusCount{
			Code:  strings.TrimSpace(code.String),
			Label: strings.TrimSpace(label.String),
			Total: total,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return counts, nil
}

// XOL mengembalikan isi grid "DATA PLA DLA XOL KLAIM".
func (r *Repo) XOL(
	ctx context.Context,
	login string,
) ([]inboxpladla.XOLRow, error) {
	clean := strings.TrimSpace(login)

	// Login yang sama diikat DUA KALI: satu untuk bagian DLA, satu untuk bagian PLA.
	// Penanda yang dipakai ulang dengan satu nomor berperilaku berbeda antar driver.
	rows, err := r.db.QueryContext(ctx, query("xol_summary"), clean, clean)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: xol_summary: %w", err)
	}
	defer rows.Close()

	items := []inboxpladla.XOLRow{}
	for rows.Next() {
		var (
			year, cause, kind sql.NullString
			lastInsert        sql.NullTime
		)
		if err := rows.Scan(&year, &cause, &kind, &lastInsert); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: xol_summary: memindai baris: %w", err)
		}

		items = append(items, inboxpladla.XOLRow{
			Year:           strings.TrimSpace(year.String),
			CauseOfLoss:    strings.TrimSpace(cause.String),
			Kind:           strings.TrimSpace(kind.String),
			LastInsertDate: dateText(lastInsert),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: xol_summary: membaca hasil: %w", err)
	}

	return items, nil
}

// searchArgs menyusun kedua bind pencarian.
//
// Keduanya NULL ketika tidak ada kata kunci, dan penandanya BERTIPE TEKS — tidak pernah
// bertipe tanggal maupun angka. Bind yang hanya muncul di dalam `IS NULL` tidak punya
// konteks tipe di dalam kueri, sehingga tipenya ditentukan driver; itu kelas galat yang
// hanya muncul di Oracle.
func searchArgs(q inboxpladla.Query) (sql.NullString, sql.NullString) {
	if q.Search == "" {
		return sql.NullString{}, sql.NullString{}
	}
	return sql.NullString{String: "1", Valid: true},
		sql.NullString{
			String: "%" + strings.ToUpper(escapeLike(q.Search)) + "%",
			Valid:  true,
		}
}

// filterArgs menyusun bind penyaring kueri DAFTAR, tanpa paginasi.
//
// # URUTANNYA MENGIKUTI KEMUNCULAN PENANDA DI DALAM TEKS SQL, BUKAN NOMORNYA
//
// Oracle mengikat argumen menurut urutan KEMUNCULAN penanda, bukan menurut angka pada
// `:n`. Kolom "No PLA" berada di klausa SELECT — sebelum WHERE — sehingga login untuknya
// adalah argumen PERTAMA, bukan ketiga.
//
// Melupakannya tidak menghasilkan galat: setiap bind tetap terisi sesuatu. Yang terjadi
// adalah kolom "No PLA" selalu kosong, dan pencarian tidak pernah menemukan apa pun. Cacat
// yang sama pernah lolos di modul lain dan baru ketahuan setelah Oracle hidup
// (`catatan-pengembangan.md` §19.14).
//
// Susunan per jenis daftar:
//
//	pemberitahuan  login · penanda · pola · login × (n-1)
//	komunikasi     login · penanda · pola · status · login
func filterArgs(q inboxpladla.Query) []any {
	searchFlag, pattern := searchArgs(q)
	login := q.Caller.Login

	if q.Tab.Source == inboxpladla.SourceCommunication {
		return []any{login, searchFlag, pattern, q.Tab.CommunicationStatus, login}
	}

	args := []any{login, searchFlag, pattern}
	for i := 0; i < loginBindCount(q.Tab)-1; i++ {
		args = append(args, login)
	}
	return args
}

// countArgs menyusun bind kueri RINGKAS.
//
// Ia sama dengan filterArgs kecuali satu hal: kueri ringkas tidak mengambil kolom
// "No PLA", sehingga rantai reasuradur untuk kolom itu tidak ada di sana — dan karena itu
// penanda pencarianlah yang muncul pertama.
func countArgs(q inboxpladla.Query) []any {
	searchFlag, pattern := searchArgs(q)
	login := q.Caller.Login

	if q.Tab.Source == inboxpladla.SourceCommunication {
		return []any{searchFlag, pattern, q.Tab.CommunicationStatus, login}
	}

	args := []any{searchFlag, pattern}
	for i := 0; i < loginBindCount(q.Tab)-1; i++ {
		args = append(args, login)
	}
	return args
}

// loginBindCount menyatakan berapa kali login diikat pada kueri DAFTAR pemberitahuan.
//
// Jumlahnya BERBEDA per tab, dan perbedaannya bukan kelalaian — ia mengikuti berapa kali
// rantai reasuradur muncul di kueri masing-masing:
//
//	list_pla    3  kolom No PLA · syarat PLA terkirim · syarat DLA belum terkirim
//	list_dla    2  kolom No PLA · syarat DLA terkirim
//	list_close  2  kolom No PLA · syarat PLA terkirim
//
// Angkanya diperiksa terhadap teks SQL di query_test.go. Tanpa uji itu, satu rantai
// reasuradur yang ditambahkan ke SQL tanpa menambah bind di sini akan menghasilkan galat
// bind yang menyebut NOMOR, bukan menyebut tab mana yang rusak.
func loginBindCount(tab inboxpladla.Tab) int {
	if tab.ExcludeWhenDLASent {
		return 3
	}
	return 2
}

// dateText menuliskan tanggal yang dibaca basis data sebagai teks `YYYY-MM-DD`.
//
// Kolom yang kosong menghasilkan teks KOSONG, bukan `0001-01-01`.
//
// Bagian JAM dibuang. Tidak satu pun kolom tanggal di layar ini digambar beserta jamnya,
// dan kueri lamanya pun sudah membuangnya — dengan cara yang salah, lewat
// `to_char(...,'dd/mm/yyyy')` yang mengubah tanggal menjadi teks dan membuat
// pengurutannya menjadi pengurutan teks.
func dateText(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(inboxpladla.DateLayout)
}
