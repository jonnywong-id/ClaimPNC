// Package sqlstore memenuhi seam masterpasal.Store dengan SQL.
//
// Satu instans repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
//
// # Paket inilah satu-satunya yang mengenal JSON
//
// `POOLDATA.V_M_DATA_PASAL.JSONPASAL` menyimpan seluruh isi pasal selain No Pasal sebagai
// satu dokumen JSON. Itu BENTUK PENYIMPANAN, bukan aturan bisnis, sehingga pembongkaran
// dan penyusunannya berhenti di berkas ini — persis seperti nama kolom, yang juga tidak
// pernah bocor ke paket domain.
package sqlstore

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/masterpasal"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca dan menulis POOLDATA.V_M_DATA_PASAL, dan MEMBACA POOLDATA.BUSINESS.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// document adalah bentuk dokumen di dalam kolom JSONPASAL.
//
// Kunci-kuncinya BUKAN karangan. Dokumen itu dihasilkan `stepPage.getJSON(false)` atas
// page klipboard `TempPasalCol` (`Function/GetPageJSONString-Function.xml`), sehingga
// kuncinya adalah nama properti page itu apa adanya. Empat di antaranya terbukti langsung
// dari `RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml` yang membacanya dengan `json_value`:
// `$.DESCRIPTION`, `$.OLD_D_COL_ID`, `$.pyCountry`, dan `$.LOSS_CODE`.
//
// Dua sisanya — `M_COL_ID` dan `OLD_M_COL_ID` — memang ada di page itu
// (`Activity/CNMInsertPasalDataMaster-Act.xml` langkah 1 menyetel keduanya) tetapi TIDAK
// PERNAH dibaca kembali dari JSON: kueri lama mengambil keduanya dari kolom `IDPASAL` dan
// `IDDATA`. Keduanya tetap ditulis supaya dokumen yang dihasilkan aplikasi ini berbentuk
// sama dengan yang dihasilkan Pega, sehingga sistem lama tetap dapat membacanya selama
// masa paralel.
//
// `BISNISID` tidak terbaca dari kueri mana pun karena Pega membacanya lewat view
// `POOLDATA.View_DATA_PASAL` yang tidak ada di export. Bentuknya diturunkan dari
// `Activity/PNCGetListPasalDataCOL_Act-Act.xml` langkah 10, yang menyusun page list itu
// dengan dua properti — `.ID` dan `.Note` — dan dari repeat layout pada
// `Section/BrowsePasalDeatailMaster-Section.xml` yang mengikat `TempPasalCol.BISNISID` ke
// kelas `ASM-FW-GISFW-Int-BUSINESS`.
type document struct {
	Number        jsonText           `json:"M_COL_ID"`
	ID            jsonText           `json:"OLD_M_COL_ID"`
	Text          jsonText           `json:"DESCRIPTION"`
	Description   jsonText           `json:"OLD_D_COL_ID"`
	Category      jsonText           `json:"pyCountry"`
	CategoryLabel jsonText           `json:"LOSS_CODE"`
	Business      []documentBusiness `json:"BISNISID"`
}

// documentBusiness adalah satu butir pada `$.BISNISID[]`.
type documentBusiness struct {
	ID   jsonText `json:"ID"`
	Name jsonText `json:"Note"`
}

// jsonText adalah teks yang MEMAAFKAN tipe JSON lain.
//
// # Kenapa ia ada, dan kenapa ia tidak boleh dianggap berlebihan
//
// Dokumen di dalam JSONPASAL ditulis Pega, bukan oleh aplikasi ini, dan bentuk keluaran
// `ClipboardPage.getJSON` bergantung pada tipe properti klipboardnya. `pyCountry`
// dibandingkan sebagai angka di Pega (`JaminanPengecualianApproval==1`), sehingga baris
// lama dapat memuat `"pyCountry": 1` — angka, bukan teks.
//
// Tanpa penerima yang memaafkan, satu baris semacam itu membuat SELURUH daftar gagal
// dibaca dengan galat penguraian yang tidak menyebut baris mana penyebabnya. Dengan ini,
// angka `1` terbaca sebagai `"1"` — dan itu memang nilai yang sama.
//
// `null` menjadi teks kosong, bukan galat: tabelnya tidak punya constraint NOT NULL yang
// diketahui (R-08).
type jsonText string

func (t *jsonText) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*t = ""
		return nil
	}

	// Teks — bentuk yang paling lazim — diurai apa adanya supaya escape di dalamnya
	// tetap ditafsirkan.
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		*t = jsonText(text)
		return nil
	}

	// Angka, true, dan false dipakai apa adanya sebagai teks. `strconv.Unquote` tidak
	// dipakai di sini karena nilainya memang belum berkutip.
	switch text := string(trimmed); text {
	case "true", "false":
		*t = jsonText(text)
		return nil
	default:
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			return fmt.Errorf("masterpasal/sqlstore: nilai JSON %q bukan teks, angka, maupun boolean", text)
		}
		*t = jsonText(text)
		return nil
	}
}

func (t jsonText) trimmed() string { return strings.TrimSpace(string(t)) }

// List membaca seluruh pasal kerugian.
//
// Daftar lini bisnis TIDAK diisi di sini meski dokumennya sudah memuatnya: nama yang
// ditampilkan harus berasal dari master lini bisnis yang berlaku sekarang, dan
// mengambilnya untuk setiap baris berarti satu pembacaan master untuk setiap pasal — demi
// kolom yang tidak ada di grid. Lihat Get, yang mengisinya untuk satu baris saja.
func (r *Repo) List(ctx context.Context) ([]masterpasal.Clause, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("clause_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpasal.Clause
	for rows.Next() {
		clause, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		// Daftar lini bisnis dibuang dari hasil daftar, supaya tidak ada pemanggil yang
		// diam-diam memakai nama SNAPSHOT dari dokumen sebagai kalau-kalau itu nama yang
		// berlaku sekarang.
		clause.Business = nil
		result = append(result, clause)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu pasal beserta daftar lini bisnisnya.
func (r *Repo) Get(ctx context.Context, id string) (masterpasal.Clause, error) {
	clause, err := scanRow(r.db.QueryRowContext(ctx, getQuery("clause_get"), strings.TrimSpace(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return masterpasal.Clause{}, masterpasal.ErrNotFound
	}
	if err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: membaca %q: %w", id, err)
	}

	business, err := r.resolveBusiness(ctx, clause.Business)
	if err != nil {
		return masterpasal.Clause{}, err
	}
	clause.Business = business
	return clause, nil
}

// resolveBusiness menyegarkan nama setiap lini bisnis dari master yang berlaku sekarang.
//
// # Kenapa disegarkan, padahal namanya sudah ada di dokumen
//
// Karena itulah yang dilakukan Pega. `Activity/PNCGetListPasalDataCOL_Act-Act.xml`
// langkah 7 membaca ulang lewat sub-kueri `(select NOTE from BUSINESS c where c.ID =
// A.D_COL_ID)`, lalu langkah 10 menyalinnya ke `.Note`. Nama yang tersimpan di dokumen
// adalah SALINAN pada saat pasal itu disimpan, dan salinan itu menjadi usang begitu master
// lini bisnisnya berubah nama.
//
// # Satu hal yang TIDAK ditiru, dan itu disengaja
//
// Pada Pega, butir yang kodenya tidak cocok dengan satu pun baris master menghasilkan
// `.Note` KOSONG — sehingga nama lini bisnis yang diketik bebas (autocomplete-nya
// ber-`pyAllowFreeFormInput=true`, dan butir semacam itu memang tidak punya kode) HILANG
// dari layar begitu pasalnya dibuka kembali. Yang diketik pengguna lenyap tanpa satu pun
// pesan.
//
// Di sini butir yang kodenya tidak ketemu — termasuk butir tanpa kode sama sekali —
// MEMPERTAHANKAN nama yang tersimpan. Yang direplikasi adalah hasil yang teramati pada
// jalur normal, bukan jalur yang merusak; alasan yang sama dipakai modul Master Auto
// Claim saat menutup penghapusan data client secara diam-diam.
func (r *Repo) resolveBusiness(
	ctx context.Context,
	stored []masterpasal.Business,
) ([]masterpasal.Business, error) {
	if len(stored) == 0 {
		return nil, nil
	}

	// Kode yang sama dapat muncul lebih dari sekali pada satu pasal — tidak ada yang
	// mencegahnya di layar lama. Cache ini membuat kode kembar dibaca sekali saja.
	known := make(map[string]string, len(stored))

	result := make([]masterpasal.Business, 0, len(stored))
	for _, business := range stored {
		code := strings.TrimSpace(business.ID)
		if code == "" {
			result = append(result, business)
			continue
		}

		name, cached := known[code]
		if !cached {
			var err error
			name, err = r.businessName(ctx, code)
			if err != nil {
				return nil, err
			}
			known[code] = name
		}

		if name != "" {
			business.Name = name
		}
		result = append(result, business)
	}
	return result, nil
}

// businessName membaca nama satu lini bisnis; teks kosong bila kodenya tidak ada.
//
// Kode yang tidak ketemu BUKAN galat: master lini bisnis dimiliki sistem lain, barisnya
// dapat hilang tanpa sepengetahuan modul ini, dan sebuah pasal yang merujuk kode yang
// sudah lenyap tetap harus dapat dibuka untuk diperbaiki.
func (r *Repo) businessName(ctx context.Context, code string) (string, error) {
	var id, note sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("business_get"), code).Scan(&id, &note)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("masterpasal/sqlstore: membaca lini bisnis %q: %w", code, err)
	}
	return strings.TrimSpace(note.String), nil
}

// Insert menyimpan satu pasal baru.
//
// Seluruhnya berjalan di dalam SATU transaksi — pengecualian yang disadari terhadap
// §4.5 `08-TECHNICAL-STRATEGY.md` yang menempatkan batas transaksi di lapisan aplikasi.
// Alasannya: nomor IDDATA diturunkan dari isi tabel itu sendiri, dan memisahkan
// penurunannya dari penyisipannya membuka lubang balapan yang justru sedang ditutup.
func (r *Repo) Insert(ctx context.Context, in masterpasal.Input) (masterpasal.Clause, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	used, err := lockedIDs(ctx, tx)
	if err != nil {
		return masterpasal.Clause{}, err
	}

	fresh := clauseFrom(masterpasal.NextSequence(used), in)

	payload, err := encode(fresh)
	if err != nil {
		return masterpasal.Clause{}, err
	}

	if _, err := tx.ExecContext(ctx, getQuery("clause_insert"), fresh.ID, fresh.Number, payload); err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: menyisipkan %q: %w", fresh.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: menutup transaksi sisip: %w", err)
	}
	return fresh, nil
}

// Update menyimpan perubahan pada pasal yang sudah ada.
//
// Baris yang tidak tersentuh dijawab ErrNotFound, bukan dianggap berhasil. Procedure lama
// memutuskannya lebih dulu dengan `select count(*)` lalu bercabang
// (`PEGA_D_PASAL_MASTER.prc:6-8`) — dan pada cabang "tidak ada" ia justru MENYISIPKAN
// baris baru. Perilaku itu tidak dibawa: `PUT` atas baris yang sudah dihapus petugas lain
// akan diam-diam menerbitkan baris kedua, dan pengguna tidak punya cara mengetahuinya.
func (r *Repo) Update(ctx context.Context, id string, in masterpasal.Input) (masterpasal.Clause, error) {
	key := strings.TrimSpace(id)
	saved := clauseFrom(key, in)

	payload, err := encode(saved)
	if err != nil {
		return masterpasal.Clause{}, err
	}

	outcome, err := r.db.ExecContext(ctx, getQuery("clause_update"), saved.Number, payload, key)
	if err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: memperbarui %q: %w", key, err)
	}
	affected, err := outcome.RowsAffected()
	if err != nil {
		return masterpasal.Clause{}, fmt.Errorf("masterpasal/sqlstore: membaca hasil pembaruan %q: %w", key, err)
	}
	if affected == 0 {
		return masterpasal.Clause{}, masterpasal.ErrNotFound
	}
	return saved, nil
}

// Delete membuang satu baris secara FISIK.
//
// Baca peringatan pada doc comment masterpasal.Repo sebelum memakainya: ia menyupersede
// D-66 untuk tabel ini, barisnya tidak dapat dipulihkan, dan tabelnya tidak punya kolom
// pencatat siapa dan kapan.
func (r *Repo) Delete(ctx context.Context, id string) error {
	key := strings.TrimSpace(id)

	outcome, err := r.db.ExecContext(ctx, getQuery("clause_delete"), key)
	if err != nil {
		return fmt.Errorf("masterpasal/sqlstore: menghapus %q: %w", key, err)
	}
	affected, err := outcome.RowsAffected()
	if err != nil {
		return fmt.Errorf("masterpasal/sqlstore: membaca hasil penghapusan %q: %w", key, err)
	}
	if affected == 0 {
		// Barisnya sudah tidak ada. Dijawab ErrNotFound alih-alih dianggap berhasil,
		// supaya layar dapat mengatakan bahwa ia sudah dihapus petugas lain — bukan
		// menampilkan keberhasilan atas sesuatu yang tidak terjadi.
		return masterpasal.ErrNotFound
	}
	return nil
}

// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
func (r *Repo) SearchBusiness(ctx context.Context, keyword string) ([]masterpasal.Business, error) {
	pattern, exact := lookupArguments(keyword)

	rows, err := r.db.QueryContext(ctx, getQuery("business_search"), pattern, exact)
	if err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: mencari lini bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpasal.Business
	for rows.Next() {
		var id, note sql.NullString
		if err := rows.Scan(&id, &note); err != nil {
			return nil, fmt.Errorf("masterpasal/sqlstore: membaca lini bisnis: %w", err)
		}
		business := masterpasal.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(note.String),
		}
		// Baris tanpa kode tidak dapat disimpan sebagai rujukan dan hanya akan menambah
		// butir yang tidak menunjuk apa pun. Ia dilewati di sini, bukan dibiarkan muncul
		// di daftar pilihan.
		if business.ID == "" {
			continue
		}
		result = append(result, business)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: menelusuri lini bisnis: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.V_M_DATA_PASAL ada dan dapat dibaca akun aplikasi.
//
// Dipakai mode `-periksa`, bukan oleh jalur permintaan. Ia membedakan "tabelnya belum ada"
// dari "akun aplikasi tidak punya hak bacanya" — dua keadaan yang dari layar terlihat
// sama persis.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("clause_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}

// CheckBusinessTable memastikan POOLDATA.BUSINESS ada dan dapat dibaca akun aplikasi.
//
// Ia terpisah dari CheckTable karena tabelnya dimiliki sistem lain: hak bacanya diberikan
// pihak yang berbeda, dan kegagalannya menuntut perbaikan yang berbeda pula.
func (r *Repo) CheckBusinessTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("business_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}

// clauseFrom menyusun baris yang akan disimpan dari isian yang sudah bersih.
//
// CategoryLabel DITURUNKAN di sini, tidak pernah diterima dari layar — persis seperti
// `Activity/CNMInsertPasalDataMaster-Act.xml` langkah 1 menurunkannya dari kode yang
// dipilih pengguna. Menerimanya dari layar berarti sebutan yang tersimpan dapat berbeda
// dari kodenya, dan tidak ada satu pun yang akan menyadarinya.
func clauseFrom(id string, in masterpasal.Input) masterpasal.Clause {
	return masterpasal.Clause{
		ID:            id,
		Number:        in.Number,
		Text:          in.Text,
		Description:   in.Description,
		Category:      in.Category,
		CategoryLabel: masterpasal.CategoryLabel(in.Category),
		Business:      in.Business,
	}
}

// encode menyusun dokumen JSONPASAL dari sebuah baris.
//
// `M_COL_ID` dan `OLD_M_COL_ID` ikut ditulis supaya bentuk dokumennya sama dengan yang
// dihasilkan Pega. Bedanya satu: Pega menulis `OLD_M_COL_ID` KOSONG pada penambahan,
// karena page form-nya baru saja dibersihkan dan IDDATA-nya belum ada saat dokumen
// disusun — sedangkan di sini nomornya sudah diturunkan lebih dulu, sehingga yang ditulis
// adalah IDDATA yang sebenarnya.
//
// Perbedaan itu tidak terlihat di mana pun: kedua kunci itu tidak pernah dibaca kembali
// dari JSON oleh kueri mana pun, dan menuliskannya dengan benar hanya dapat membuat
// dokumennya lebih konsisten, tidak pernah kurang.
func encode(clause masterpasal.Clause) (string, error) {
	doc := document{
		Number:        jsonText(clause.Number),
		ID:            jsonText(clause.ID),
		Text:          jsonText(clause.Text),
		Description:   jsonText(clause.Description),
		Category:      jsonText(clause.Category),
		CategoryLabel: jsonText(clause.CategoryLabel),
	}
	for _, business := range clause.Business {
		doc.Business = append(doc.Business, documentBusiness{
			ID:   jsonText(business.ID),
			Name: jsonText(business.Name),
		})
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("masterpasal/sqlstore: menyusun dokumen pasal %q: %w", clause.ID, err)
	}
	return string(payload), nil
}

type scanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris beserta dokumennya.
//
// Ketiga kolom dibaca lewat sql.NullString lalu dipangkas: kolom bertipe CHAR berlebar
// tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris lama dapat
// memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui (R-08).
//
// # Dua keadaan dokumen yang diperlakukan BERBEDA, dan itu disengaja
//
//	JSONPASAL kosong atau NULL  -> pasal tanpa isi; dibaca sebagai baris bermuatan kosong
//	JSONPASAL tidak dapat diurai -> galat yang MENYEBUT IDDATA-nya
//
// Yang pertama keadaan yang sah: tidak ada yang mencegah sebuah baris disisipkan dengan
// CLOB kosong, dan membuat seluruh daftar gagal karenanya berarti satu baris setengah jadi
// mematikan layar untuk seluruh entitas.
//
// Yang kedua cacat data yang nyata. Ia TIDAK dimaafkan menjadi baris berisian kosong:
// pasal yang isinya tampak hilang akan disimpan ulang oleh petugas yang mengira isiannya
// memang kosong, dan saat itu isi aslinya benar-benar hilang. Galatnya menyebut IDDATA
// supaya barisnya dapat langsung dicari.
func scanRow(p scanner) (masterpasal.Clause, error) {
	var id, number, payload sql.NullString
	if err := p.Scan(&id, &number, &payload); err != nil {
		return masterpasal.Clause{}, err
	}

	clause := masterpasal.Clause{
		ID:     strings.TrimSpace(id.String),
		Number: strings.TrimSpace(number.String),
	}

	raw := strings.TrimSpace(payload.String)
	if raw == "" {
		clause.CategoryLabel = masterpasal.CategoryLabel(clause.Category)
		return clause, nil
	}

	var doc document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return masterpasal.Clause{}, fmt.Errorf(
			"masterpasal/sqlstore: dokumen JSONPASAL baris %q tidak dapat diurai: %w", clause.ID, err)
	}

	clause.Text = doc.Text.trimmed()
	clause.Description = doc.Description.trimmed()
	clause.Category = doc.Category.trimmed()

	// Sebutan kategori DITURUNKAN ulang dari kodenya, tidak dipakai apa adanya dari
	// dokumen. Keduanya tersimpan berdampingan, dan tidak ada apa pun yang menjaga
	// keduanya tetap sejalan — sebuah baris yang kodenya pernah diubah lewat jalur lain
	// akan memuat sebutan yang tidak lagi sesuai. Yang menjadi sumber kebenaran adalah
	// kodenya, karena itulah yang dipakai layar untuk memilih ulang.
	clause.CategoryLabel = masterpasal.CategoryLabel(clause.Category)

	for _, business := range doc.Business {
		clause.Business = append(clause.Business, masterpasal.Business{
			ID:   business.ID.trimmed(),
			Name: business.Name.trimmed(),
		})
	}
	return clause, nil
}

// lockedIDs mengunci seluruh baris lalu mengembalikan IDDATA-nya.
func lockedIDs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQuery("clause_list_id_locked"))
	if err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: mengunci daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var used []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterpasal/sqlstore: membaca ID terpakai: %w", err)
		}
		used = append(used, id.String)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpasal/sqlstore: menelusuri ID terpakai: %w", err)
	}
	return used, nil
}

// lookupArguments menyusun kedua argumen kueri pencarian lini bisnis.
//
// Kata kunci di-uppercase supaya cocok dengan `UPPER(NOTE)` pada kuerinya, lalu ketiga
// karakter berarti pada LIKE — `\`, `%`, dan `_` — diloloskan. Tanpa itu, tanda persen
// yang diketik pengguna berlaku sebagai wildcard dan hasil pencariannya tidak dapat
// dijelaskan kepada yang mengetiknya.
func lookupArguments(keyword string) (pattern, exact string) {
	clean := strings.ToUpper(strings.TrimSpace(keyword))

	escaped := clean
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%", clean
}

func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("masterpasal/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterpasal/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterpasal/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterpasal/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		result[name] = strings.TrimSpace(strings.Join(body, "\n"))
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		body = append(body, line)
	}
	save()
	return result
}
