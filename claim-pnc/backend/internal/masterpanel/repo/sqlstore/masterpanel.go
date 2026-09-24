// Package sqlstore memenuhi seam masterpanel.Store dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpanel"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// sequenceWidth adalah lebar nomor urut pada ID_PANEL.
//
// Meniru `Database/PEGA_M_PANEL_HE.prc:21` persis:
//
//	id_panel_he_ins := id_site || lpad(to_Char(PANEL_HE_SEQ.nextval),6,'0');
//
// ENAM digit — bukan sepuluh seperti Master Bengkel. Kedua procedure ditulis dengan pola
// yang sama dan lebarnya tetap berbeda; menyeragamkannya akan menerbitkan ID yang tidak
// sebentuk dengan ID yang sudah ada.
//
// Pembentukannya dilakukan di Go dan bukan di SQL supaya kuerinya tetap portabel — LPAD
// ada di kedua basis data, tetapi menyusun kunci di dalam kueri berarti bentuk kuncinya
// tersebar ke berkas .sql dan ke sini sekaligus.
const sequenceWidth = 6

// Repo membaca dan menulis POOLDATA.PANEL_HE beserta tabel anaknya
// POOLDATA.LOKASI_PANEL_HE.
//
// Satu struct memenuhi kedua seam — masterpanel.Repo dan IDSource — karena keduanya
// selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah interface-nya,
// bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring, LENGKAP dengan lokasinya.
//
// Dua kueri, bukan satu JOIN yang mengembalikan baris induk berulang: induk dan anak
// punya jumlah kolom yang sangat berbeda, dan JOIN akan mengirimkan kelima belas kolom
// induk sekali untuk setiap lokasi. Dua kueri juga membuat panel TANPA lokasi terbaca
// apa adanya, tanpa perlu membedakan "baris tanpa anak" dari "anak yang seluruh kolomnya
// NULL" — pembedaan yang pada OUTER JOIN selalu rapuh.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya
// ditempel — lihat berkas .sql untuk alasannya.
func (r *Repo) List(ctx context.Context, filter masterpanel.Filter) ([]masterpanel.Panel, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword != "" {
		rows, err = r.db.QueryContext(ctx, getQuery("panel_list_search"),
			string(filter.Status), likePattern(keyword))
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("panel_list"), string(filter.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpanel.Panel
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterpanel/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: menelusuri daftar: %w", err)
	}
	if len(result) == 0 {
		return result, nil
	}

	location, err := r.locationByStatus(ctx, filter.Status, keyword)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Location = location[strings.TrimSpace(result[i].ID)]
	}
	return result, nil
}

// locationByStatus membaca seluruh baris lokasi milik panel pada satu status sekaligus,
// dikelompokkan menurut ID_PANEL induknya.
//
// Penyaringnya WAJIB sama persis dengan penyaring daftar induknya. Bila berbeda, sebagian
// panel akan tampil tanpa lokasinya tanpa satu pun tanda bahwa ada yang salah.
func (r *Repo) locationByStatus(
	ctx context.Context,
	status masterpanel.ApprovalStatus,
	keyword string,
) (map[string][]masterpanel.PanelLocation, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if keyword != "" {
		rows, err = r.db.QueryContext(ctx, getQuery("panel_location_by_status_search"),
			string(status), likePattern(keyword))
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("panel_location_by_status"), string(status))
	}
	if err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: membaca lokasi panel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := map[string][]masterpanel.PanelLocation{}
	for rows.Next() {
		var owner, name, side sql.NullString
		if err := rows.Scan(&owner, &name, &side); err != nil {
			return nil, fmt.Errorf("masterpanel/sqlstore: membaca baris lokasi: %w", err)
		}
		key := strings.TrimSpace(owner.String)
		result[key] = append(result[key], masterpanel.PanelLocation{
			Name: strings.TrimSpace(name.String),
			Side: masterpanel.Side(strings.TrimSpace(side.String)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: menelusuri lokasi panel: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan ID_PANEL-nya, beserta lokasinya.
func (r *Repo) Get(ctx context.Context, id string) (masterpanel.Panel, error) {
	key := strings.TrimSpace(id)
	row := r.db.QueryRowContext(ctx, getQuery("panel_get"), key)

	p, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterpanel.Panel{}, masterpanel.ErrNotFound
	}
	if err != nil {
		return masterpanel.Panel{}, fmt.Errorf("masterpanel/sqlstore: membaca %q: %w", id, err)
	}

	p.Location, err = r.locationOf(ctx, key)
	if err != nil {
		return masterpanel.Panel{}, err
	}
	return p, nil
}

// locationOf membaca baris lokasi satu panel.
func (r *Repo) locationOf(ctx context.Context, id string) ([]masterpanel.PanelLocation, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("panel_location_get"), strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: membaca lokasi %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpanel.PanelLocation
	for rows.Next() {
		var name, side sql.NullString
		if err := rows.Scan(&name, &side); err != nil {
			return nil, fmt.Errorf("masterpanel/sqlstore: membaca baris lokasi %q: %w", id, err)
		}
		result = append(result, masterpanel.PanelLocation{
			Name: strings.TrimSpace(name.String),
			Side: masterpanel.Side(strings.TrimSpace(side.String)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpanel/sqlstore: menelusuri lokasi %q: %w", id, err)
	}
	return result, nil
}

// FindByName mencari baris menurut NAME-nya.
//
// Lokasinya TIDAK ikut dibaca: pemanggilnya hanya perlu tahu barisnya ada dan apa
// kuncinya. Membacanya berarti satu perjalanan basis data tambahan pada setiap
// penyimpanan, untuk data yang tidak dipakai siapa pun.
func (r *Repo) FindByName(ctx context.Context, name string) (masterpanel.Panel, error) {
	clean := strings.ToUpper(strings.TrimSpace(name))
	if clean == "" {
		return masterpanel.Panel{}, masterpanel.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("panel_find_by_name"), clean)

	p, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterpanel.Panel{}, masterpanel.ErrNotFound
	}
	if err != nil {
		return masterpanel.Panel{}, fmt.Errorf("masterpanel/sqlstore: mencari nama %q: %w", name, err)
	}
	return p, nil
}

// Insert menolak nama yang sudah dipakai, lalu menyisipkan barisnya beserta lokasinya.
//
// Ketiganya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi,
// dan di modul ini alasannya dua:
//
//  1. Pemeriksaan dan penyisipan induk bukan dua perkara melainkan satu; memisahkannya
//     membuka kembali lubang balapan yang justru sedang dipersempit.
//  2. Induk dan anak WAJIB atomik. Panel yang tersimpan tanpa lokasinya adalah keadaan
//     yang tidak dapat dibedakan dari panel yang memang tidak punya lokasi — dan tidak
//     ada satu pun cara mengetahui mana yang sebenarnya terjadi.
func (r *Repo) Insert(ctx context.Context, p masterpanel.Panel) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterpanel/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	taken, err := locked(ctx, tx, p.Name)
	if err != nil {
		return err
	}
	if taken {
		return fmt.Errorf("%w: %q", masterpanel.ErrNameTaken, p.Name)
	}

	if _, err := tx.ExecContext(ctx, getQuery("panel_insert"), insertArguments(p)...); err != nil {
		return fmt.Errorf("masterpanel/sqlstore: menyisipkan %q: %w", p.ID, err)
	}
	if err := insertLocation(ctx, tx, p); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("masterpanel/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada beserta seluruh lokasinya.
//
// Lokasinya DIGANTI, bukan digabung: dibuang seluruhnya lalu disisipkan ulang dari daftar
// yang dikirim. Lihat banner berkas .sql untuk alasannya, dan untuk pertentangannya
// dengan D-66 yang dinyatakan terbuka alih-alih disembunyikan.
//
// Pemeriksaan bentrok nama TIDAK dilakukan di sini, dan itu mengikuti sistem lama:
// `ValidateMasterPanel` dipanggil dari layar, bukan dari jalur penyimpanan. Lapisan
// aplikasi yang memeriksanya, dan ia mengecualikan baris itu sendiri.
func (r *Repo) Update(ctx context.Context, p masterpanel.Panel) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterpanel/sqlstore: memulai transaksi ubah: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, getQuery("panel_update"), updateArguments(p)...)
	if err != nil {
		return fmt.Errorf("masterpanel/sqlstore: memperbarui %q: %w", p.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh
	// satu baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan
	// mengatakan "tersimpan" atas baris yang sudah tidak ada — lebih buruk lagi, lokasi
	// yang dibuang di langkah berikutnya TIDAK akan kembali.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return masterpanel.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, getQuery("panel_location_clear"), strings.TrimSpace(p.ID)); err != nil {
		return fmt.Errorf("masterpanel/sqlstore: membuang lokasi %q: %w", p.ID, err)
	}
	if err := insertLocation(ctx, tx, p); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("masterpanel/sqlstore: menutup transaksi ubah: %w", err)
	}
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
// berubah-ubah — lihat `panel_set_status` pada berkas .sql untuk alasannya.
//
// Transaksinya melingkupi seluruh baris supaya persetujuan borongan tidak pernah setengah
// jalan: bila baris kelima gagal, keempat yang sebelumnya ikut dibatalkan. Sistem lama
// tidak menjamin itu — `SetApprovalAllMaster` menjalankan satu RDB-List per baris tanpa
// transaksi yang melingkupinya.
//
// Lokasi TIDAK disentuh: keputusan persetujuan mengenai induknya saja.
func (r *Repo) SetStatus(
	ctx context.Context,
	id []string,
	status masterpanel.ApprovalStatus,
	reason string,
) (int, error) {
	if len(id) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("masterpanel/sqlstore: memulai transaksi keputusan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, one := range id {
		result, err := tx.ExecContext(ctx, getQuery("panel_set_status"),
			string(status), reason, strings.TrimSpace(one))
		if err != nil {
			return 0, fmt.Errorf("masterpanel/sqlstore: menetapkan status %q: %w", one, err)
		}
		// Driver yang tidak mendukung RowsAffected membuat pencacahnya tidak dapat
		// diandalkan. Barisnya tetap dianggap berubah: pernyataannya sudah berhasil, dan
		// melaporkan nol akan menampilkan kegagalan palsu.
		affected, err := result.RowsAffected()
		if err != nil || affected > 0 {
			changed++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("masterpanel/sqlstore: menutup transaksi keputusan: %w", err)
	}
	return changed, nil
}

// NextID menerbitkan ID_PANEL berikutnya.
//
// Bentuknya meniru `Database/PEGA_M_PANEL_HE.prc:12,21` persis: kode situs ditambah nomor
// urut ENAM digit bertambal nol.
//
// Kedua kueri dijalankan di dalam SATU transaksi. Bukan demi keatomikan — sequence tidak
// dapat dibatalkan — melainkan supaya keduanya pasti dilayani koneksi yang sama; kode
// situs dan sequence yang berasal dari dua koneksi berbeda pada pool yang sama tetap
// benar, tetapi jaminannya tidak berasal dari mana pun selain kebetulan.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: memulai transaksi penomoran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var site sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery("panel_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Persis keadaan yang ditangkap `PEGA_M_PANEL_HE.prc:13-18`, yang menjawabnya
			// dengan pesan galat lalu berhenti. Di sini ia juga berhenti — ID tanpa kode
			// situs akan bertabrakan dengan ID entitas lain.
			return "", errors.New("masterpanel/sqlstore: POOLDATA.M_SITE_DATABASE tidak punya baris CURRENT_SITE='1'")
		}
		return "", fmt.Errorf("masterpanel/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("panel_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: mengambil nomor urut: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("masterpanel/sqlstore: menutup transaksi penomoran: %w", err)
	}
	return masterpanel.ComposeID(strings.TrimSpace(site.String), sequence, sequenceWidth), nil
}

// CheckTable memastikan POOLDATA.PANEL_HE ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	return r.checkReadable(ctx, "panel_check_table", "POOLDATA.PANEL_HE")
}

// CheckLocationTable memastikan POOLDATA.LOKASI_PANEL_HE ada beserta keempat kolomnya.
func (r *Repo) CheckLocationTable(ctx context.Context) error {
	return r.checkReadable(ctx, "panel_check_location_table", "POOLDATA.LOKASI_PANEL_HE")
}

// CheckJSONMirror memastikan POOLDATA.M_PANEL_HE — tabel JSON milik Pega — dapat dibaca.
//
// Dipakai mode periksa saja. Lihat banner berkas .sql: perbandingan jumlah barisnya dengan
// PANEL_HE adalah cara termurah mengetahui apakah keduanya satu sumber.
func (r *Repo) CheckJSONMirror(ctx context.Context) error {
	return r.checkReadable(ctx, "panel_check_json_mirror", "POOLDATA.M_PANEL_HE")
}

func (r *Repo) checkReadable(ctx context.Context, name, table string) error {
	rows, err := r.db.QueryContext(ctx, getQuery(name))
	if err != nil {
		return fmt.Errorf("masterpanel/sqlstore: %s tidak dapat dibaca: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountByStatus menghitung baris pada satu status persetujuan.
func (r *Repo) CountByStatus(ctx context.Context, status masterpanel.ApprovalStatus) (int, error) {
	return r.count(ctx, "panel_count_pending", string(status))
}

// CountAll menghitung seluruh baris PANEL_HE.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_all")
}

// CountJSONMirror menghitung baris POOLDATA.M_PANEL_HE.
func (r *Repo) CountJSONMirror(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_json_mirror")
}

// CountLocation menghitung seluruh baris POOLDATA.LOKASI_PANEL_HE.
func (r *Repo) CountLocation(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_location")
}

// CountLocationOrphan menghitung baris lokasi yang induknya tidak ada.
func (r *Repo) CountLocationOrphan(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_location_orphan")
}

// CountLocationNameMismatch menghitung baris yang NAMA-nya berbeda dari LOKASI_PANEL.
//
// Inilah pemeriksaan yang menentukan apakah asumsi pada banner berkas .sql benar. Lihat
// checkPanel pada cmd/claimpnc untuk apa yang dilakukan atas hasilnya.
func (r *Repo) CountLocationNameMismatch(ctx context.Context) (int, error) {
	return r.count(ctx, "panel_count_location_name_mismatch")
}

func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterpanel/sqlstore: menjalankan %s: %w", name, err)
	}
	return total, nil
}

// insertLocation menyisipkan seluruh baris lokasi sebuah panel.
//
// Kolom NAMA diisi nilai yang SAMA dengan LOKASI_PANEL; lihat banner berkas .sql untuk
// asumsi di baliknya dan cara memeriksanya.
func insertLocation(ctx context.Context, tx *sql.Tx, p masterpanel.Panel) error {
	owner := strings.TrimSpace(p.ID)
	for _, one := range p.Location {
		name := strings.TrimSpace(one.Name)
		if _, err := tx.ExecContext(ctx, getQuery("panel_location_insert"),
			owner, name, string(one.Side), name); err != nil {
			return fmt.Errorf("masterpanel/sqlstore: menyisipkan lokasi %q pada %q: %w",
				one.Name, p.ID, err)
		}
	}
	return nil
}

// likePattern menyusun pola LIKE dari sebuah kata kunci.
//
// Tanda persen, garis bawah, dan backslash pada kata kunci DILOLOSKAN lebih dulu. Tanpa
// itu, pengguna yang mengetik "%" menarik seluruh tabel dan yang mengetik "_" mencocoki
// karakter apa pun — bukan celah keamanan karena nilainya tetap terikat sebagai parameter,
// tetapi hasil yang tidak dapat dijelaskan kepada yang mengetiknya.
//
// `ESCAPE '\'` disebut eksplisit di kuerinya karena Oracle tidak punya karakter pelolos
// bawaan pada LIKE.
func likePattern(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

type scanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris induk master panel.
//
// Seluruh kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui
// (R-08).
//
// Urutan kolomnya mengikuti berkas .sql, dan kelima belas kolom dibaca pada urutan yang
// sama oleh panel_list, panel_list_search, panel_get, dan panel_find_by_name. Itu yang
// membuat satu fungsi cukup untuk keempatnya — dan yang membuat uji urutan kolom pada
// query_test.go layak ada.
//
// Location TIDAK diisi di sini: ia dibaca kueri tersendiri. Pemanggil yang mengisinya.
func scanRow(p scanner) (masterpanel.Panel, error) {
	var (
		id, name, repair, editQuantity, premiumRepair sql.NullString
		shatter, sticker, side, severe, active        sql.NullString
		exclusionC, approvalMark, reason              sql.NullString
		documentID, status                            sql.NullString
	)
	if err := p.Scan(
		&id, &name, &repair, &editQuantity, &premiumRepair,
		&shatter, &sticker, &side, &severe, &active,
		&exclusionC, &approvalMark, &reason, &documentID, &status,
	); err != nil {
		return masterpanel.Panel{}, err
	}

	trim := strings.TrimSpace
	return masterpanel.Panel{
		ID:                  trim(id.String),
		Name:                trim(name.String),
		RepairStatus:        trim(repair.String),
		EditQuantityStatus:  trim(editQuantity.String),
		PremiumRepairStatus: trim(premiumRepair.String),
		ShatterStatus:       trim(shatter.String),
		StickerStatus:       trim(sticker.String),
		SideStatus:          trim(side.String),
		SevereDamageStatus:  trim(severe.String),
		ActiveStatus:        trim(active.String),
		ExclusionC:          trim(exclusionC.String),
		ApprovalMark:        trim(approvalMark.String),
		RejectReason:        trim(reason.String),
		DocumentID:          trim(documentID.String),
		Status:              masterpanel.ApprovalStatus(trim(status.String)),
	}, nil
}

// insertArguments menyusun kelima belas nilai panel_insert pada urutan kolomnya.
//
// Urutannya WAJIB sama dengan daftar kolom pada kueri. Ia dipisahkan menjadi fungsi
// tersendiri supaya urutan itu dapat diuji terhadap berkas .sql, bukan hanya dipercaya.
func insertArguments(p masterpanel.Panel) []any {
	return []any{
		p.ID, p.Name, p.RepairStatus, p.EditQuantityStatus, p.PremiumRepairStatus,
		p.ShatterStatus, p.StickerStatus, p.SideStatus, p.SevereDamageStatus, p.ActiveStatus,
		p.ExclusionC, p.ApprovalMark, p.RejectReason, p.DocumentID, string(p.Status),
	}
}

// updateArguments menyusun nilai panel_update; ID_PANEL berada di posisi TERAKHIR karena
// ia penyaring WHERE, bukan kolom yang ditulis.
func updateArguments(p masterpanel.Panel) []any {
	return []any{
		p.Name, p.RepairStatus, p.EditQuantityStatus, p.PremiumRepairStatus, p.ShatterStatus,
		p.StickerStatus, p.SideStatus, p.SevereDamageStatus, p.ActiveStatus, p.ExclusionC,
		p.ApprovalMark, p.RejectReason, p.DocumentID, string(p.Status),
		strings.TrimSpace(p.ID),
	}
}

// locked menjalankan panel_lock_by_name dan menyatakan barisnya ada.
func locked(ctx context.Context, tx *sql.Tx, value string) (bool, error) {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return false, nil
	}

	rows, err := tx.QueryContext(ctx, getQuery("panel_lock_by_name"), clean)
	if err != nil {
		return false, fmt.Errorf("masterpanel/sqlstore: memeriksa nama %q: %w", value, err)
	}
	defer func() { _ = rows.Close() }()

	found := rows.Next()
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("masterpanel/sqlstore: menelusuri pemeriksaan %q: %w", value, err)
	}
	return found, nil
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("masterpanel/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterpanel/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterpanel/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterpanel/sqlstore: nama kueri ganda: " + name)
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
		var statement []string
		for _, rows := range body {
			if strings.HasPrefix(strings.TrimSpace(rows), "--") {
				continue
			}
			statement = append(statement, rows)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, rows := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(rows); strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}

var _ masterpanel.Store = (*Repo)(nil)
