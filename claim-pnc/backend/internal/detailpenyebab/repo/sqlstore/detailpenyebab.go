package sqlstore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/detailpenyebab"
)

// Repo memenuhi detailpenyebab.Store terhadap satu koneksi entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo atas satu koneksi entitas.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// document adalah bentuk dokumen di dalam POOLDATA.D_CAUSE_OF_LOSS.JSONDATA.
//
// # Kuncinya adalah nama PROPERTI PAGE, bukan nama kolom
//
// `@GCNM.GetPageJSONString()` menyerialkan page `TempDcol` apa adanya
// (`Activity/CNMInsertDetailCauseOfLoss_act-Act.xml:397`), sehingga kunci JSON-nya sama
// dengan nama properti yang dipakai layar — dan nama itu kebetulan sama dengan nama kolom
// view yang membentangkannya kembali.
//
// "Kebetulan" di sini bukan basa-basi: kesamaan itulah yang membuat pemetaan ini
// REKONSTRUKSI dan bukan bacaan. Definisi view-nya tidak ada di export (`R-08`). Cara
// membuktikannya ada di kepala berkas .sql — simpan satu baris, lalu periksa keenam kolom
// view-nya terisi.
type document struct {
	ID          jsonText           `json:"D_COL_ID"`
	LegacyID    jsonText           `json:"OLD_D_COL_ID"`
	MasterID    jsonText           `json:"M_COL_ID"`
	Description jsonText           `json:"DESCRIPTION"`
	LossCode    jsonText           `json:"LOSS_CODE"`
	Active      jsonText           `json:"STS_AKTIF"`
	Business    []documentBusiness `json:"BISNISID"`
}

// documentBusiness adalah satu baris `$.BISNISID[]`.
//
// Kunci `Note` berhuruf besar di depan saja — bukan `NOTE` dan bukan `note` — mengikuti
// `Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1784-1785`, yang mengisi
// `TempDcol.BISNISID(<LAST>).Note`. Jalur JSON bersifat case-sensitive, dan satu huruf
// saja membuat view lini bisnis berhenti mengenali dokumennya.
type documentBusiness struct {
	ID   jsonText `json:"ID"`
	Name jsonText `json:"Note"`
}

// jsonText adalah teks yang MEMAAFKAN tipe JSON lain.
//
// Dokumen di dalam JSONDATA ditulis Pega, bukan oleh aplikasi ini, dan bentuk keluaran
// `ClipboardPage.getJSON` bergantung pada tipe properti klipboardnya. Kode dan status
// dibandingkan sebagai angka di beberapa tempat, sehingga baris lama dapat memuat
// `"STS_AKTIF": 1` — angka, bukan teks.
//
// Tanpa penerima yang memaafkan, satu baris semacam itu membuat SELURUH daftar gagal
// dibaca dengan galat penguraian yang tidak menyebut baris mana penyebabnya. Dengan ini,
// angka `1` terbaca sebagai `"1"` — dan itu memang nilai yang sama.
//
// `null` menjadi teks kosong, bukan galat: tabelnya tidak punya constraint NOT NULL yang
// diketahui (`R-08`).
type jsonText string

func (t *jsonText) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*t = ""
		return nil
	}

	// Teks — bentuk yang paling lazim — diurai apa adanya supaya escape di dalamnya tetap
	// ditafsirkan.
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		*t = jsonText(text)
		return nil
	}

	// Angka, true, dan false dipakai apa adanya sebagai teks.
	switch text := string(trimmed); text {
	case "true", "false":
		*t = jsonText(text)
		return nil
	default:
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			return fmt.Errorf("detailpenyebab/sqlstore: nilai JSON %q bukan teks, angka, maupun boolean", text)
		}
		*t = jsonText(text)
		return nil
	}
}

// List mengembalikan baris yang cocok dengan penyaring.
func (r *Repo) List(
	ctx context.Context,
	filter detailpenyebab.Filter,
) ([]detailpenyebab.CauseOfLossDetail, error) {
	keyword := strings.TrimSpace(filter.Keyword)
	master := strings.TrimSpace(filter.MasterID)
	business := strings.TrimSpace(filter.BusinessID)

	// Penyaring kosong dikirim sebagai NULL, bukan sebagai teks kosong: kueri memeriksanya
	// dengan `:n IS NULL`, dan teks kosong bukan NULL.
	rows, err := r.db.QueryContext(ctx, getQuery("detail_list"),
		nullable(keyword), likePattern(keyword), likePattern(keyword), likePattern(keyword),
		nullable(master), master,
		nullable(business), business,
	)
	if err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: membaca daftar: %w", err)
	}
	defer rows.Close()

	list := make([]detailpenyebab.CauseOfLossDetail, 0, 32)
	for rows.Next() {
		one, err := scanDetail(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: membaca daftar: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu baris LENGKAP dengan daftar lini bisnisnya.
func (r *Repo) Get(
	ctx context.Context,
	id string,
) (detailpenyebab.CauseOfLossDetail, error) {
	key := strings.TrimSpace(id)
	if key == "" {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}

	rows, err := r.db.QueryContext(ctx, getQuery("detail_get"), key)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: membaca satu baris: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
				"detailpenyebab/sqlstore: membaca satu baris: %w", err)
		}
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}

	found, err := scanDetail(rows)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	// rows ditutup lebih dulu: pembacaan lini bisnis memakai koneksi yang sama, dan
	// menahan kursor yang belum habis dapat menguncinya pada driver tertentu.
	rows.Close()

	found.Business, err = r.businessOf(ctx, found.ID)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	return found, nil
}

// businessOf membaca lini bisnis satu detail dari view turunannya.
func (r *Repo) businessOf(
	ctx context.Context,
	id string,
) ([]detailpenyebab.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_business_list"), strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: membaca lini bisnis: %w", err)
	}
	defer rows.Close()

	var list []detailpenyebab.Business
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("detailpenyebab/sqlstore: membaca lini bisnis: %w", err)
		}
		list = append(list, detailpenyebab.Business{
			ID:   strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: membaca lini bisnis: %w", err)
	}
	return list, nil
}

// Insert menyimpan satu baris baru beserta ID yang diterbitkan.
//
// # Kenapa seluruhnya di dalam SATU transaksi
//
// Penerbitan ID dan penyisipan barisnya adalah satu perkara. Di sistem lama keduanya juga
// berada di dalam satu procedure, tetapi tanpa transaksi yang membungkusnya — dan
// `PEGA_D_CAUSE_OF_LOSS.prc:27` bahkan memanggil `ROLLBACK` dari dalam blok exception,
// yang tidak memulihkan apa pun bila pemanggilnya sudah commit lebih dulu.
//
// Kepemilikan transaksi di sistem baru ada sepenuhnya di Go (`D-68`).
func (r *Repo) Insert(
	ctx context.Context,
	in detailpenyebab.Input,
) (detailpenyebab.CauseOfLossDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: membuka transaksi: %w", err)
	}
	// Rollback setelah Commit tidak berbahaya: ia menjawab ErrTxDone yang sengaja
	// diabaikan. Tanpa baris ini, setiap jalur gagal meninggalkan transaksi menggantung.
	defer func() { _ = tx.Rollback() }()

	site, err := siteCode(ctx, tx)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("detail_next_sequence")).Scan(&sequence); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: mengambil nomor urut: %w", err)
	}

	fresh := detailpenyebab.CauseOfLossDetail{
		ID:          detailpenyebab.FormatID(site, sequence),
		LegacyID:    in.LegacyID,
		MasterID:    in.MasterID,
		Description: in.Description,
		LossCode:    in.LossCode,
		Active:      in.Active,
		Business:    in.Business,
	}

	// Sequence yang pernah di-reset dapat menerbitkan nomor yang sudah dipakai. Procedure
	// lama tidak memeriksanya dan akan gagal dengan pelanggaran kunci yang pesannya tidak
	// menyebut sebabnya; di sini keadaannya dikenali dan diberi galat yang dapat dijelaskan.
	taken, err := idTaken(ctx, tx, fresh.ID)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	if taken {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrIDTaken
	}

	payload, err := json.Marshal(newDocument(fresh))
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: menyusun dokumen: %w", err)
	}

	if _, err := tx.ExecContext(ctx, getQuery("detail_insert"), fresh.ID, string(payload)); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: menyisipkan baris: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: menyimpan transaksi: %w", err)
	}

	fresh.MasterLabel, err = r.masterLabel(ctx, fresh.MasterID)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	return fresh, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// # Dokumen lama dibaca lebih dulu, lalu ditimpa sebagian
//
// Alasannya ada pada komentar kueri `detail_document`: dokumen dapat memuat kunci yang
// tidak dibentangkan view mana pun, dan menyusun dokumen baru hanya dari kolom yang
// dikenal akan membuangnya diam-diam pada setiap penyimpanan.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	in detailpenyebab.Input,
) (detailpenyebab.CauseOfLossDetail, error) {
	key := strings.TrimSpace(id)
	if key == "" {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: membuka transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	raw, err := currentDocument(ctx, tx, key)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	saved := detailpenyebab.CauseOfLossDetail{
		ID:          key,
		LegacyID:    in.LegacyID,
		MasterID:    in.MasterID,
		Description: in.Description,
		LossCode:    in.LossCode,
		Active:      in.Active,
		Business:    in.Business,
	}

	payload, err := mergeDocument(raw, saved)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	outcome, err := tx.ExecContext(ctx, getQuery("detail_update"), payload, key)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: memperbarui baris: %w", err)
	}
	// Baris yang hilang di antara pemuatan layar dan penyimpanan menghasilkan nol baris
	// terpengaruh, bukan galat. Tanpa pemeriksaan ini, penyimpanan yang tidak menyentuh apa
	// pun akan dilaporkan berhasil.
	changed, err := outcome.RowsAffected()
	if err == nil && changed == 0 {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: menyimpan transaksi: %w", err)
	}

	saved.MasterLabel, err = r.masterLabel(ctx, saved.MasterID)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	return saved, nil
}

// SearchMaster mencari Master Penyebab Kerugian menurut sebutan atau kodenya.
func (r *Repo) SearchMaster(
	ctx context.Context,
	keyword string,
) ([]detailpenyebab.MasterOption, error) {
	clean := strings.TrimSpace(keyword)
	rows, err := r.db.QueryContext(ctx, getQuery("master_search"), likePattern(clean), clean)
	if err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari master penyebab: %w", err)
	}
	defer rows.Close()

	var list []detailpenyebab.MasterOption
	for rows.Next() {
		var code, label sql.NullString
		if err := rows.Scan(&code, &label); err != nil {
			return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari master penyebab: %w", err)
		}
		list = append(list, detailpenyebab.MasterOption{
			ID:    strings.TrimSpace(code.String),
			Label: strings.TrimSpace(label.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari master penyebab: %w", err)
	}
	return list, nil
}

// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
func (r *Repo) SearchBusiness(
	ctx context.Context,
	keyword string,
) ([]detailpenyebab.Business, error) {
	clean := strings.TrimSpace(keyword)
	rows, err := r.db.QueryContext(ctx, getQuery("business_search"), likePattern(clean), clean)
	if err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari lini bisnis: %w", err)
	}
	defer rows.Close()

	var list []detailpenyebab.Business
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari lini bisnis: %w", err)
		}
		list = append(list, detailpenyebab.Business{
			ID:   strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("detailpenyebab/sqlstore: mencari lini bisnis: %w", err)
	}
	return list, nil
}

// masterLabel membaca sebutan induk sebuah baris.
//
// Induk yang tidak ditemukan BUKAN galat: tidak ada foreign key yang diketahui (`R-08`),
// sehingga baris yatim mungkin ada — dan di Pega pun sub-kueri sebutannya hanya menjawab
// NULL tanpa menggagalkan apa pun.
func (r *Repo) masterLabel(ctx context.Context, masterID string) (string, error) {
	key := strings.TrimSpace(masterID)
	if key == "" {
		return "", nil
	}

	var code, label sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("master_get"), key).Scan(&code, &label)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("detailpenyebab/sqlstore: membaca sebutan master: %w", err)
	default:
		return strings.TrimSpace(label.String), nil
	}
}

// CheckTables memastikan seluruh objek yang dipakai modul ini dapat dibaca akun aplikasi.
//
// Dipakai mode periksa pada cmd. Ia menyebut objeknya satu per satu — bukan satu
// pemeriksaan gabungan — supaya keluarannya menyebut objek MANA yang bermasalah.
func (r *Repo) CheckTables(ctx context.Context) map[string]error {
	checks := map[string]string{
		"POOLDATA.V_D_CAUSE_OF_LOSS":          "detail_check_table",
		"POOLDATA.D_CAUSE_OF_LOSS":            "detail_check_writable",
		"POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS": "detail_check_business_view",
		"POOLDATA.V_M_CAUSE_OF_LOSS":          "master_check_table",
		"POOLDATA.BUSINESS":                   "business_check_table",
	}

	result := map[string]error{}
	for object, name := range checks {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			result[object] = err
			continue
		}
		result[object] = rows.Err()
		rows.Close()
	}
	return result
}

// scanRow adalah apa yang dibutuhkan scanDetail — *sql.Rows memenuhinya.
type scanRow interface {
	Scan(target ...any) error
}

// scanDetail membaca satu baris hasil detail_list maupun detail_get.
//
// Keduanya memilih kolom yang sama dengan urutan yang sama; menyatukannya di sini membuat
// perubahan daftar kolom menyentuh satu tempat, bukan dua.
func scanDetail(rows scanRow) (detailpenyebab.CauseOfLossDetail, error) {
	var id, legacy, master, description, lossCode, active, masterLabel sql.NullString
	if err := rows.Scan(
		&id, &legacy, &master, &description, &lossCode, &active, &masterLabel,
	); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, fmt.Errorf(
			"detailpenyebab/sqlstore: membaca baris: %w", err)
	}

	return detailpenyebab.CauseOfLossDetail{
		ID:          strings.TrimSpace(id.String),
		LegacyID:    strings.TrimSpace(legacy.String),
		MasterID:    strings.TrimSpace(master.String),
		MasterLabel: strings.TrimSpace(masterLabel.String),
		Description: strings.TrimSpace(description.String),
		LossCode:    strings.TrimSpace(lossCode.String),
		Active:      strings.TrimSpace(active.String),
	}, nil
}

// siteCode membaca kode situs, bagian pertama setiap D_COL_ID.
//
// Ketiadaan barisnya adalah galat, bukan nilai kosong: ID tanpa kode situs akan
// bertabrakan dengan ID entitas lain begitu basis datanya digabung, dan kesalahan itu baru
// terlihat setelah barisnya tersimpan.
func siteCode(ctx context.Context, tx *sql.Tx) (string, error) {
	var site sql.NullString
	err := tx.QueryRowContext(ctx, getQuery("detail_site")).Scan(&site)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", errors.New(
			"detailpenyebab/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris ber-CURRENT_SITE '1'")
	case err != nil:
		return "", fmt.Errorf("detailpenyebab/sqlstore: membaca kode situs: %w", err)
	}

	clean := strings.TrimSpace(site.String)
	if clean == "" {
		return "", errors.New(
			"detailpenyebab/sqlstore: kode situs pada POOLDATA.M_SITE_DATABASE kosong")
	}
	return clean, nil
}

// idTaken menyatakan apakah sebuah D_COL_ID sudah dipakai.
func idTaken(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	rows, err := tx.QueryContext(ctx, getQuery("detail_exists"), strings.TrimSpace(id))
	if err != nil {
		return false, fmt.Errorf("detailpenyebab/sqlstore: memeriksa ID: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return true, nil
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("detailpenyebab/sqlstore: memeriksa ID: %w", err)
	}
	return false, nil
}

// currentDocument membaca dokumen JSON yang tersimpan sekarang.
func currentDocument(ctx context.Context, tx *sql.Tx, id string) (string, error) {
	var raw sql.NullString
	err := tx.QueryRowContext(ctx, getQuery("detail_document"), id).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", detailpenyebab.ErrNotFound
	case err != nil:
		return "", fmt.Errorf("detailpenyebab/sqlstore: membaca dokumen: %w", err)
	default:
		return raw.String, nil
	}
}

// newDocument menyusun dokumen JSON baris baru.
func newDocument(one detailpenyebab.CauseOfLossDetail) document {
	doc := document{
		ID:          jsonText(one.ID),
		LegacyID:    jsonText(one.LegacyID),
		MasterID:    jsonText(one.MasterID),
		Description: jsonText(one.Description),
		LossCode:    jsonText(one.LossCode),
		Active:      jsonText(one.Active),
	}
	// Daftar kosong ditulis sebagai array kosong, bukan null: view yang membentangkannya
	// menjadi baris memperlakukan keduanya sama, tetapi array kosong menyatakan "tidak ada
	// lini bisnis" secara tegas sedangkan null dapat terbaca sebagai "belum pernah diisi".
	doc.Business = make([]documentBusiness, 0, len(one.Business))
	for _, business := range one.Business {
		doc.Business = append(doc.Business, documentBusiness{
			ID:   jsonText(business.ID),
			Name: jsonText(business.Name),
		})
	}
	return doc
}

// mergeDocument menimpa kunci yang disunting layar ini ke dalam dokumen yang sudah ada,
// dan MEMBIARKAN kunci lain apa adanya.
//
// Dokumen lama yang kosong atau tidak dapat diurai diperlakukan sebagai dokumen baru —
// bukan galat. Baris yang JSONDATA-nya rusak tetap harus dapat diperbaiki lewat layar,
// dan menolaknya justru mengunci satu-satunya jalan memperbaikinya.
func mergeDocument(raw string, one detailpenyebab.CauseOfLossDetail) (string, error) {
	fields := map[string]any{}
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
			// Dokumen yang tidak dapat diurai dibuang dan disusun ulang dari nol.
			fields = map[string]any{}
		}
	}

	fields["D_COL_ID"] = one.ID
	fields["OLD_D_COL_ID"] = one.LegacyID
	fields["M_COL_ID"] = one.MasterID
	fields["DESCRIPTION"] = one.Description
	fields["LOSS_CODE"] = one.LossCode
	fields["STS_AKTIF"] = one.Active

	business := make([]map[string]string, 0, len(one.Business))
	for _, line := range one.Business {
		business = append(business, map[string]string{"ID": line.ID, "Note": line.Name})
	}
	fields["BISNISID"] = business

	payload, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("detailpenyebab/sqlstore: menyusun dokumen: %w", err)
	}
	return string(payload), nil
}

// likePattern menyusun pola LIKE yang huruf besar-kecilnya sudah diseragamkan, dengan
// karakter khas LIKE dilolosi.
//
// `%`, `_`, dan `\` di dalam kata kunci dilolosi supaya keduanya dicari sebagai huruf
// biasa. Tanpa itu, seorang petugas yang mengetik `_` akan menerima hasil yang cocok
// dengan karakter apa pun — dan tidak ada apa pun di layar yang menjelaskan kenapa.
func likePattern(keyword string) string {
	clean := strings.TrimSpace(keyword)
	if clean == "" {
		return "%"
	}
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToUpper(replacer.Replace(clean)) + "%"
}

// nullable mengubah teks kosong menjadi NULL basis data.
//
// Kueri memeriksa penyaring dengan `:n IS NULL`, dan teks kosong BUKAN NULL — mengirimnya
// apa adanya akan membuat penyaring kosong menyaring sungguhan dan mengosongkan daftar.
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
