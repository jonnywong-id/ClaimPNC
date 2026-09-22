// Package sqlstore memenuhi seam mastergroupingsparepart.Store dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat penyaringan
// baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring berdasarkan
// entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/mastergroupingsparepart"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// approvedLookup adalah nilai APPROVAL yang dipakai daftar panel.
//
// '1', bukan '0'. Itu yang diminta autocomplete Nama Panel lewat parameter report definition
// `APPROVAL = "1"` pada `Section/MasterGroupingSparepartHEApproval-Section.xml`.
const approvedLookup = string(mastergroupingsparepart.StatusApproved)

// vehicleTypeGroup adalah nilai kolom TYPE yang membatasi daftar tipe kendaraan.
//
// `RDB List/BrowseTypeHE_Sql-SQL.xml` menanamnya di dalam teks SQL sebagai `type = 'ANEKA'`.
// Di sini ia konstanta bernama supaya terbaca sebagai keputusan — lini bisnis Aneka — alih-
// alih tersembunyi di tengah kueri.
const vehicleTypeGroup = "ANEKA"

// Repo membaca dan menulis kedua tabel grouping, dan HANYA MEMBACA keempat sumber acuannya.
//
// Satu struct memenuhi ketiga seam — mastergroupingsparepart.Repo, LookupRepo, dan IDSource —
// karena ketiganya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel —
// lihat berkas .sql untuk alasannya.
func (r *Repo) List(
	ctx context.Context,
	filter mastergroupingsparepart.Filter,
) ([]mastergroupingsparepart.Grouping, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword != "" {
		// Pola yang sama dikirim empat kali: nomor sparepart, nama sparepart, nama panel, dan
		// nomor rangka. Lihat catatan pada grouping_list_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("grouping_list_search"),
			string(filter.Status), pattern, pattern, pattern, pattern)
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("grouping_list"), string(filter.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("mastergroupingsparepart/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastergroupingsparepart.Grouping
	for rows.Next() {
		g, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastergroupingsparepart/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan ID-nya.
func (r *Repo) Get(
	ctx context.Context,
	id string,
) (mastergroupingsparepart.Grouping, error) {
	row := r.db.QueryRowContext(ctx, getQuery("grouping_get"), strings.TrimSpace(id))

	g, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
	}
	if err != nil {
		return mastergroupingsparepart.Grouping{}, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: membaca %q: %w", id, err)
	}
	return g, nil
}

// FindByKey mencari baris menurut keempat kunci alaminya.
func (r *Repo) FindByKey(
	ctx context.Context,
	key mastergroupingsparepart.NaturalKey,
) (mastergroupingsparepart.Grouping, error) {
	partNumber, panelName, chassis, side := upperKey(key)
	if partNumber == "" && panelName == "" && chassis == "" && side == "" {
		return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("grouping_find_by_key"),
		partNumber, panelName, chassis, side)

	g, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
	}
	if err != nil {
		return mastergroupingsparepart.Grouping{}, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: mencari kunci grouping: %w", err)
	}
	return g, nil
}

// FindGroupByChassis mengembalikan nomor grup yang dipakai sebuah nomor rangka.
//
// Nilainya dikembalikan APA ADANYA, tanpa TO_NUMBER; lihat berkas .sql dan
// usecase.Service.resolveGroupNumber.
func (r *Repo) FindGroupByChassis(ctx context.Context, chassis string) (string, error) {
	clean := strings.ToUpper(strings.TrimSpace(chassis))
	if clean == "" {
		return "", mastergroupingsparepart.ErrGroupChassisNotFound
	}

	var group sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("grouping_group_by_chassis"), clean).Scan(&group)
	if errors.Is(err, sql.ErrNoRows) {
		return "", mastergroupingsparepart.ErrGroupChassisNotFound
	}
	if err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/sqlstore: mencari grup nomor rangka %q: %w", chassis, err)
	}

	found := strings.TrimSpace(group.String)
	if found == "" {
		// Barisnya ada tetapi nomor grupnya kosong. Kueri sudah menyaring NULL; yang tersisa
		// adalah nilai berisi spasi saja. Ia tidak dapat diikuti, dan mengembalikannya sebagai
		// nomor grup akan menghasilkan baris yang tidak tergabung ke mana pun.
		return "", mastergroupingsparepart.ErrGroupChassisNotFound
	}
	return found, nil
}

// ListPanels membaca panel yang sudah disetujui.
func (r *Repo) ListPanels(ctx context.Context) ([]mastergroupingsparepart.Panel, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("grouping_panel_list"), approvedLookup)
	if err != nil {
		return nil, fmt.Errorf("mastergroupingsparepart/sqlstore: membaca panel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastergroupingsparepart.Panel, 0, 64)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris panel: %w", err)
		}
		result = append(result, mastergroupingsparepart.Panel{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
		if len(result) >= mastergroupingsparepart.MaxLookupRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastergroupingsparepart/sqlstore: menelusuri panel: %w", err)
	}
	return result, nil
}

// ListSides membaca sandi sisi yang tersedia pada sebuah panel.
func (r *Repo) ListSides(
	ctx context.Context,
	key mastergroupingsparepart.SideKey,
) ([]mastergroupingsparepart.Side, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("grouping_side_list"),
		strings.TrimSpace(key.PanelID),
		strings.ToUpper(strings.TrimSpace(key.PanelName)))
	if err != nil {
		return nil, fmt.Errorf("mastergroupingsparepart/sqlstore: membaca sisi panel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastergroupingsparepart.Side, 0, 4)
	for rows.Next() {
		var side sql.NullString
		if err := rows.Scan(&side); err != nil {
			return nil, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris sisi panel: %w", err)
		}
		// Baris yang sandinya kosong dilewati: ia bukan pilihan yang dapat disimpan, dan
		// menampilkannya sebagai pilihan kosong membuat pengguna mengira ia sudah memilih.
		if clean := strings.TrimSpace(side.String); clean != "" {
			result = append(result, mastergroupingsparepart.Side(clean))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menelusuri sisi panel: %w", err)
	}
	return result, nil
}

// ListVehicleTypes membaca tipe kendaraan yang aktif.
func (r *Repo) ListVehicleTypes(
	ctx context.Context,
) ([]mastergroupingsparepart.VehicleType, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("grouping_vehicle_type_list"), vehicleTypeGroup)
	if err != nil {
		return nil, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: membaca tipe kendaraan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastergroupingsparepart.VehicleType, 0, 64)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris tipe kendaraan: %w", err)
		}
		result = append(result, mastergroupingsparepart.VehicleType{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
		if len(result) >= mastergroupingsparepart.MaxLookupRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menelusuri tipe kendaraan: %w", err)
	}
	return result, nil
}

// FindPart mencari satu sparepart menurut nomornya.
func (r *Repo) FindPart(
	ctx context.Context,
	number string,
) (mastergroupingsparepart.PartRef, error) {
	clean := strings.ToUpper(strings.TrimSpace(number))
	if clean == "" {
		return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.ErrPartNotFound
	}

	var no, name, category, partType, code, produced sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("grouping_part_find"), clean).
		Scan(&no, &name, &category, &partType, &code, &produced)
	if errors.Is(err, sql.ErrNoRows) {
		return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.ErrPartNotFound
	}
	if err != nil {
		return mastergroupingsparepart.PartRef{}, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: mencari sparepart %q: %w", number, err)
	}

	trim := strings.TrimSpace
	return mastergroupingsparepart.PartRef{
		Number:         trim(no.String),
		Name:           trim(name.String),
		CategoryID:     trim(category.String),
		TypeID:         trim(partType.String),
		Code:           trim(code.String),
		ProductionDate: trim(produced.String),
	}, nil
}

// Insert menolak kunci alami yang sudah dipakai, lalu menyisipkan barisnya pada KEDUA tabel.
//
// Ketiganya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi, dan
// alasannya satu: pemeriksaan dan penyisipan bukan dua perkara melainkan satu. Memisahkannya
// membuka kembali lubang balapan yang justru sedang dipersempit.
//
// Baris pendamping disisipkan di dalam transaksi yang sama. Bila ia gagal, baris induknya
// ikut dibatalkan — tanpa itu, induk tanpa pendamping akan lahir dan TIDAK AKAN PERNAH
// MUNCUL di layar mana pun, karena kedua sistem menggabungkannya dengan INNER JOIN.
func (r *Repo) Insert(ctx context.Context, g mastergroupingsparepart.Grouping) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa ini,
	// satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan kunci
	// baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if err := lockedKey(ctx, tx, g); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, getQuery("grouping_insert"),
		insertArguments(g)...); err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: menyisipkan %q: %w", g.ID, err)
	}
	if _, err := tx.ExecContext(ctx, getQuery("grouping_group_insert"),
		groupInsertArguments(g)...); err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menyisipkan pendamping %q: %w", g.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada pada KEDUA tabel.
//
// Baris pendamping di-UPDATE di tempat, dan hanya DISISIPKAN bila belum ada. Tidak ada DELETE
// di jalur ini — lihat banner berkas .sql (`D-66`).
//
// Pemeriksaan bentrok kunci TIDAK dilakukan di sini, dan itu mengikuti pembagian yang sama
// dengan master lain: lapisan aplikasi yang memeriksanya, dan ia mengecualikan baris itu
// sendiri.
func (r *Repo) Update(ctx context.Context, g mastergroupingsparepart.Grouping) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: memulai transaksi ubah: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, getQuery("grouping_update"), updateArguments(g)...)
	if err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: memperbarui %q: %w", g.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh satu
	// baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan mengatakan
	// "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan menolaknya
	// akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return mastergroupingsparepart.ErrNotFound
	}

	if err := saveCompanion(ctx, tx, g); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mastergroupingsparepart/sqlstore: menutup transaksi ubah: %w", err)
	}
	return nil
}

// saveCompanion menyimpan baris pendamping: UPDATE lebih dulu, INSERT bila belum ada.
//
// Urutan itu dipilih supaya jalur yang LAZIM — baris pendamping sudah ada — hanya menempuh
// satu pernyataan. Jalur sebaliknya hanya terjadi pada baris warisan yang pendampingnya
// hilang, dan baris seperti itu memang tidak pernah tampil di layar sampai pendampingnya ada.
func saveCompanion(
	ctx context.Context,
	tx *sql.Tx,
	g mastergroupingsparepart.Grouping,
) error {
	result, err := tx.ExecContext(ctx, getQuery("grouping_group_update"),
		g.ChassisNumber, g.VehicleType, strings.TrimSpace(g.ID))
	if err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: memperbarui pendamping %q: %w", g.ID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// Driver tidak dapat melaporkannya. Pernyataannya sendiri sudah berhasil; menyisipkan
		// "untuk berjaga-jaga" justru berisiko menghasilkan baris pendamping ganda.
		return nil
	}
	if affected > 0 {
		return nil
	}

	if _, err := tx.ExecContext(ctx, getQuery("grouping_group_insert"),
		groupInsertArguments(g)...); err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menyisipkan pendamping %q: %w", g.ID, err)
	}
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
// berubah-ubah — lihat `grouping_set_status` pada berkas .sql untuk alasannya.
//
// Transaksinya melingkupi seluruh baris supaya persetujuan borongan tidak pernah setengah
// jalan: bila baris kelima gagal, keempat yang sebelumnya ikut dibatalkan.
//
// Tabel pendamping TIDAK disentuh: keputusan persetujuan mengenai induknya saja, dan
// APPROVAL memang hanya ada di sana.
func (r *Repo) SetStatus(
	ctx context.Context,
	id []string,
	status mastergroupingsparepart.ApprovalStatus,
) (int, error) {
	if len(id) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: memulai transaksi keputusan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, one := range id {
		result, err := tx.ExecContext(ctx, getQuery("grouping_set_status"),
			string(status), strings.TrimSpace(one))
		if err != nil {
			return 0, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: menetapkan status %q: %w", one, err)
		}
		// Driver yang tidak mendukung RowsAffected membuat pencacahnya tidak dapat diandalkan.
		// Barisnya tetap dianggap berubah: pernyataannya sudah berhasil, dan melaporkan nol
		// akan menampilkan kegagalan palsu.
		affected, err := result.RowsAffected()
		if err != nil || affected > 0 {
			changed++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menutup transaksi keputusan: %w", err)
	}
	return changed, nil
}

// NextID menerbitkan ID berikutnya.
//
// Bentuknya meniru `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:11` — `MAX(ID)+1`, tanpa kode
// situs dan tanpa pengisian nol.
//
// # KEDUA penyimpanan dibaca, dan yang terbesar yang menang
//
// Tabel yang ditulis aplikasi ini DAN penyimpanan JSON yang masih ditulis Pega selama masa
// paralel (D-21). Membaca salah satu saja berarti kedua sistem dapat menerbitkan ID yang sama
// pada hari yang sama.
//
// Keduanya dibaca di dalam SATU transaksi supaya keduanya pasti dilayani koneksi yang sama —
// dua pembacaan dari dua koneksi pada pool yang sama tetap benar, tetapi jaminannya tidak
// berasal dari mana pun selain kebetulan.
//
// # Keterbatasan yang disadari
//
// Angka yang dibaca di sini dapat sudah basi begitu transaksinya ditutup. Yang mempersempitnya
// adalah penguncian kunci alami di dalam transaksi penyisipan; yang benar-benar menutupnya
// adalah sequence atau constraint unik, dan keduanya menunggu DDL (R-08) serta prosedur
// perubahan skema (D-63). Sistem lama tidak mempersempitnya sama sekali.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/sqlstore: memulai transaksi penomoran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	live, err := maxNumber(ctx, tx, "grouping_max_id")
	if err != nil {
		return "", err
	}

	mirror, err := maxNumber(ctx, tx, "grouping_max_id_mirror")
	if err != nil {
		// Penyimpanan JSON boleh tidak dapat dibaca — akun aplikasi mungkin belum diberi
		// haknya, atau tabelnya memang tidak ada di entitas itu. Yang dipakai kemudian hanya
		// tabel yang benar-benar ditulis; ketidaktersediaannya dilaporkan `claimpnc -periksa`,
		// bukan menggagalkan penambahan.
		mirror = 0
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menutup transaksi penomoran: %w", err)
	}

	next := live
	if mirror > next {
		next = mirror
	}
	return strconv.FormatInt(next+1, 10), nil
}

// NextGroupNumber menerbitkan nomor grup berikutnya, sudah berbentuk akhir.
//
// Nomor terbesarnya dihitung SECARA ANGKA, bukan lewat `MAX()` atas kolom teks — lihat
// `grouping_group_numbers` pada berkas .sql untuk cacat yang dihindarinya.
//
// Nilai yang tidak dapat diurai sebagai angka DILEWATI, bukan menggagalkan penambahan: satu
// baris warisan yang isinya rusak tidak boleh menghentikan seluruh layar. Jumlahnya tidak
// dilaporkan dari sini — ia bukan tempat yang dibaca siapa pun saat terjadi — melainkan lewat
// pemeriksaan tersendiri; lihat CountUnreadableGroupNumber.
func (r *Repo) NextGroupNumber(ctx context.Context) (string, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("grouping_group_numbers"))
	if err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/sqlstore: membaca nomor grup: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var highest int64
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			return "", fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris nomor grup: %w", err)
		}
		number, err := mastergroupingsparepart.ParseGroupNumber(value.String)
		if err != nil {
			continue
		}
		if number > highest {
			highest = number
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menelusuri nomor grup: %w", err)
	}

	return mastergroupingsparepart.ComposeGroupNumber(highest + 1), nil
}

// CountUnreadableGroupNumber menghitung nomor grup yang tidak dapat dibaca sebagai angka.
//
// Dipakai mode periksa saja. Nilai seperti itu dilewati saat menerbitkan nomor baru, sehingga
// keberadaannya tidak menghentikan apa pun — tetapi ia menandakan baris yang tidak akan pernah
// dapat diikuti sebagai grup, dan itu layak terlihat.
func (r *Repo) CountUnreadableGroupNumber(ctx context.Context) (int, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("grouping_group_numbers"))
	if err != nil {
		return 0, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: membaca nomor grup: %w", err)
	}
	defer func() { _ = rows.Close() }()

	broken := 0
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			return 0, fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca baris nomor grup: %w", err)
		}
		if _, err := mastergroupingsparepart.ParseGroupNumber(value.String); err != nil {
			broken++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menelusuri nomor grup: %w", err)
	}
	return broken, nil
}

// maxNumber membaca MAX sebuah kolom kunci dan menguraikannya sebagai angka.
//
// Kolomnya dibaca sebagai TEKS lebih dulu, bukan langsung sebagai angka: tipenya belum
// diketahui (R-08), dan driver yang memindai kolom teks ke dalam int64 menggagalkan seluruh
// pembacaan dengan galat yang tidak menyebut kolom mana yang salah.
func maxNumber(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	var value sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery(name)).Scan(&value); err != nil {
		return 0, fmt.Errorf("mastergroupingsparepart/sqlstore: menjalankan %s: %w", name, err)
	}

	clean := strings.TrimSpace(value.String)
	if clean == "" {
		// Tabelnya kosong. `NVL(MAX(ID),0)` pada procedure lama menjawabnya dengan nol, dan
		// penambahannya menghasilkan ID pertama bernomor 1.
		return 0, nil
	}

	number, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"mastergroupingsparepart/sqlstore: %s menghasilkan %q yang bukan angka; "+
				"kolom ID diasumsikan numerik mengikuti PEGA_M_GROUPING_SPAREPART_HE.prc",
			name, clean)
	}
	return number, nil
}

// CheckTable memastikan tabel induk ada dan kelima belas kolomnya dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	return r.checkReadable(ctx, "grouping_check_table", "POOLDATA.SPAREPART_HE_VIN_KEY")
}

// CheckGroupTable memastikan tabel pendamping dapat dibaca.
func (r *Repo) CheckGroupTable(ctx context.Context) error {
	return r.checkReadable(ctx, "grouping_check_group_table",
		"POOLDATA.SPAREPART_HE_VIN_GROUP")
}

// CheckVehicleTypeTable memastikan tabel tipe kendaraan dapat dibaca.
func (r *Repo) CheckVehicleTypeTable(ctx context.Context) error {
	return r.checkReadable(ctx, "grouping_check_vehicle_type_table", "branddetail")
}

// CheckChildNameColumn memastikan kolom NAMA pada tabel anak Master Panel dapat dibaca.
//
// Ia satu-satunya kolom di luar modul ini yang keberadaannya menentukan apakah daftar Sisi
// dapat terisi sama sekali; lihat LookupRepo.ListSides.
func (r *Repo) CheckChildNameColumn(ctx context.Context) error {
	return r.checkReadable(ctx, "grouping_check_child_name_column",
		"POOLDATA.LOKASI_PANEL_HE")
}

// CheckJSONMirror memastikan penyimpanan JSON milik Pega dapat dibaca.
func (r *Repo) CheckJSONMirror(ctx context.Context) error {
	return r.checkReadable(ctx, "grouping_check_json_mirror",
		"POOLDATA.M_SPAREPART_HE_VIN_KEY")
}

func (r *Repo) checkReadable(ctx context.Context, name, table string) error {
	rows, err := r.db.QueryContext(ctx, getQuery(name))
	if err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: %s tidak dapat dibaca: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountByStatus menghitung baris pada satu status persetujuan.
func (r *Repo) CountByStatus(
	ctx context.Context,
	status mastergroupingsparepart.ApprovalStatus,
) (int, error) {
	return r.count(ctx, "grouping_count_by_status", string(status))
}

// CountAll menghitung seluruh baris induk.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_all")
}

// CountWithoutGroup menghitung baris induk yang tidak punya pendamping.
func (r *Repo) CountWithoutGroup(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_without_group")
}

// CountChassisMismatch menghitung baris yang nomor rangka kedua tabelnya berbeda.
func (r *Repo) CountChassisMismatch(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_chassis_mismatch")
}

// CountChildNameAsPanel menghitung baris lokasi yang NAMA-nya sama dengan nama panel induknya.
func (r *Repo) CountChildNameAsPanel(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_child_name_as_panel")
}

// CountChildNameAsLocation menghitung baris lokasi yang NAMA-nya sama dengan LOKASI_PANEL-nya.
func (r *Repo) CountChildNameAsLocation(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_child_name_as_location")
}

// CountChildRows menghitung seluruh baris lokasi panel.
func (r *Repo) CountChildRows(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_child_rows")
}

// CountOrphanPart menghitung baris yang nomor sparepartnya tidak ada di Master Sparepart.
func (r *Repo) CountOrphanPart(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_orphan_part")
}

// CountOrphanPanel menghitung baris yang nama panelnya tidak ada di Master Panel.
func (r *Repo) CountOrphanPanel(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_orphan_panel")
}

// CountJSONMirror menghitung baris penyimpanan JSON milik Pega.
func (r *Repo) CountJSONMirror(ctx context.Context) (int, error) {
	return r.count(ctx, "grouping_count_json_mirror")
}

func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastergroupingsparepart/sqlstore: menjalankan %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyusun pola LIKE dari sebuah kata kunci.
//
// Tanda persen, garis bawah, dan backslash pada kata kunci DILOLOSKAN lebih dulu. Tanpa itu,
// pengguna yang mengetik "%" menarik seluruh tabel dan yang mengetik "_" mencocoki karakter
// apa pun — bukan celah keamanan karena nilainya tetap terikat sebagai parameter, tetapi hasil
// yang tidak dapat dijelaskan kepada yang mengetiknya.
//
// `ESCAPE '\'` disebut eksplisit di kuerinya karena Oracle tidak punya karakter pelolos bawaan
// pada LIKE.
func likePattern(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

// upperKey menyiapkan keempat nilai kunci alami untuk dibandingkan.
//
// Keempatnya di-uppercase DAN dipangkas, sepasang dengan `UPPER(TRIM(...))` pada kuerinya.
// Bila keduanya berbeda, pencarian kunci akan meleset tanpa satu pun galat.
func upperKey(key mastergroupingsparepart.NaturalKey) (string, string, string, string) {
	upper := func(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }
	return upper(key.PartNumber), upper(key.PanelName),
		upper(key.ChassisNumber), upper(key.PanelSide)
}

type scanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris grouping.
//
// Seluruh kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris lama
// dapat memuat NULL karena kedua tabel ini tidak punya constraint NOT NULL yang diketahui
// (R-08).
//
// TIDAK ada satu pun kolom waktu, sehingga tidak ada sql.NullTime di sini — berbeda dari
// Master Sparepart. Kedua tabel modul ini memang tidak punya kolom waktu sama sekali.
//
// Urutan kolomnya mengikuti berkas .sql, dan keenam belas kolom dibaca pada urutan yang sama
// oleh grouping_list, grouping_list_search, grouping_get, dan grouping_find_by_key. Itu yang
// membuat satu fungsi cukup untuk keempatnya — dan yang membuat uji urutan kolom pada
// query_test.go layak ada.
func scanRow(p scanner) (mastergroupingsparepart.Grouping, error) {
	var (
		id, partNumber, partName, partCode     sql.NullString
		categoryID, typeID, productionDate     sql.NullString
		panelID, panelName, panelSide          sql.NullString
		chassis, vehicleType, groupWithChassis sql.NullString
		groupNumber, note, status              sql.NullString
	)
	if err := p.Scan(
		&id, &partNumber, &partName, &partCode,
		&categoryID, &typeID, &productionDate,
		&panelID, &panelName, &panelSide,
		&chassis, &vehicleType, &groupWithChassis,
		&groupNumber, &note, &status,
	); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	trim := strings.TrimSpace
	return mastergroupingsparepart.Grouping{
		ID:               trim(id.String),
		PartNumber:       trim(partNumber.String),
		PartName:         trim(partName.String),
		PartCode:         trim(partCode.String),
		CategoryID:       trim(categoryID.String),
		TypeID:           trim(typeID.String),
		ProductionDate:   trim(productionDate.String),
		PanelID:          trim(panelID.String),
		PanelName:        trim(panelName.String),
		PanelSide:        trim(panelSide.String),
		ChassisNumber:    trim(chassis.String),
		VehicleType:      trim(vehicleType.String),
		GroupWithChassis: trim(groupWithChassis.String),
		GroupNumber:      trim(groupNumber.String),
		Note:             trim(note.String),
		Status:           mastergroupingsparepart.ApprovalStatus(trim(status.String)),
	}, nil
}

// insertArguments menyusun kelima belas nilai grouping_insert pada urutan kolomnya.
//
// Urutannya WAJIB sama dengan daftar kolom pada kueri. Ia dipisahkan menjadi fungsi tersendiri
// supaya urutan itu dapat diuji terhadap berkas .sql, bukan hanya dipercaya.
func insertArguments(g mastergroupingsparepart.Grouping) []any {
	return []any{
		g.ID, g.PartNumber, g.PartName, g.PartCode, g.CategoryID,
		g.TypeID, g.ProductionDate, g.PanelID, g.PanelName, g.PanelSide,
		g.ChassisNumber, g.GroupWithChassis, g.GroupNumber, g.Note, string(g.Status),
	}
}

// updateArguments menyusun nilai grouping_update; ID berada di posisi TERAKHIR karena ia
// penyaring WHERE, bukan kolom yang ditulis.
func updateArguments(g mastergroupingsparepart.Grouping) []any {
	return []any{
		g.PartNumber, g.PartName, g.PartCode, g.CategoryID, g.TypeID,
		g.ProductionDate, g.PanelID, g.PanelName, g.PanelSide, g.ChassisNumber,
		g.GroupWithChassis, g.GroupNumber, g.Note, string(g.Status),
		strings.TrimSpace(g.ID),
	}
}

// groupInsertArguments menyusun ketiga nilai grouping_group_insert pada urutan kolomnya.
//
// NO_RANGKA-nya adalah nilai yang SAMA dengan yang ditulis ke tabel induk; lihat banner berkas
// .sql untuk alasannya.
func groupInsertArguments(g mastergroupingsparepart.Grouping) []any {
	return []any{strings.TrimSpace(g.ID), g.ChassisNumber, g.VehicleType}
}

// lockedKey menjalankan grouping_lock_by_key dan menolak kunci alami yang sudah dipakai.
//
// Galat sentinel ErrDuplicate yang dikembalikan, bukan ValidationError seperti Master
// Sparepart: kuncinya SATU — keempat kolom bersama-sama — sehingga tidak ada pilihan isian
// mana yang harus disorot, dan transport yang memetakannya ke 409.
func lockedKey(ctx context.Context, tx *sql.Tx, g mastergroupingsparepart.Grouping) error {
	partNumber, panelName, chassis, side := upperKey(mastergroupingsparepart.KeyOf(g))
	if partNumber == "" && panelName == "" && chassis == "" && side == "" {
		return nil
	}

	rows, err := tx.QueryContext(ctx, getQuery("grouping_lock_by_key"),
		partNumber, panelName, chassis, side)
	if err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: memeriksa kunci %q: %w", g.ID, err)
	}
	defer func() { _ = rows.Close() }()

	taken := false
	for rows.Next() {
		var other sql.NullString
		if err := rows.Scan(&other); err != nil {
			return fmt.Errorf(
				"mastergroupingsparepart/sqlstore: membaca hasil pemeriksaan: %w", err)
		}
		taken = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf(
			"mastergroupingsparepart/sqlstore: menelusuri pemeriksaan kunci: %w", err)
	}

	if taken {
		return mastergroupingsparepart.ErrDuplicate
	}
	return nil
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan masukan
// pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf(
			"mastergroupingsparepart/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("mastergroupingsparepart/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("mastergroupingsparepart/sqlstore: tidak dapat membaca " +
				file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("mastergroupingsparepart/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris komentar
// dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
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

var _ mastergroupingsparepart.Store = (*Repo)(nil)
