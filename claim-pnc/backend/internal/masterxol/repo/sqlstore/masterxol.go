package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"claim-pnc/internal/masterxol"
)

// Repo membaca dan menulis keempat tabel Master XOL.
//
// Kepemilikan tabelnya dijelaskan di masterxol.sql. Ringkasnya: layar ini satu-satunya
// penulis keempatnya di sistem lama, sehingga memindahkan layarnya memindahkan
// kepemilikannya secara utuh (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh induk tanpa anaknya, terurut numerik menurut ID.
func (r *Repo) List(ctx context.Context) ([]masterxol.Master, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_list"))
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca daftar master XOL: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterxol.Master
	for rows.Next() {
		master, err := scanMaster(rows)
		if err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris master XOL: %w", err)
		}
		result = append(result, master)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri daftar master XOL: %w", err)
	}

	sortMasterByID(result)
	return result, nil
}

// Get membaca satu induk lengkap dengan bisnis, lapisan, dan reas-nya.
//
// Empat perjalanan ke basis data, bukan satu JOIN besar. Itu disengaja: satu JOIN atas
// empat tingkat akan mengalikan baris induk sebanyak hasil kali seluruh anaknya — satu
// induk dengan 4 bisnis dan 3 lapisan ber-6 reas menghasilkan 72 baris untuk satu induk,
// dan kode perakitnya jauh lebih mudah salah daripada keempat kueri terpisah ini.
//
// Jumlah perjalanannya pun terbatas dan dapat diperkirakan: 3 + jumlah lapisan, dan
// lapisan terbanyak di produksi adalah empat.
func (r *Repo) Get(ctx context.Context, id string) (masterxol.Master, error) {
	id = strings.TrimSpace(id)

	master, err := scanMaster(r.db.QueryRowContext(ctx, getQuery("xol_get"), id))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterxol.Master{}, masterxol.ErrNotFound
	case err != nil:
		return masterxol.Master{}, fmt.Errorf("masterxol/sqlstore: membaca master XOL: %w", err)
	}

	if master.Business, err = r.businessOf(ctx, id); err != nil {
		return masterxol.Master{}, err
	}
	if master.Layer, err = r.layerOf(ctx, id); err != nil {
		return masterxol.Master{}, err
	}

	// Clean menghitung ulang ConvertedLimit dari Limit dan kurs induk. Nilai yang
	// tersimpan sengaja TIDAK dipakai apa adanya: satu baris produksi (lapisan 10017)
	// menyimpan 0 padahal limitnya NULL, dan membawanya apa adanya akan menampilkan
	// angka yang tidak dapat diterangkan dari isian mana pun di layar.
	return master.Clean(), nil
}

func (r *Repo) businessOf(ctx context.Context, masterID string) ([]masterxol.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_business_list"), masterID)
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca bisnis master XOL: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterxol.Business
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris bisnis: %w", err)
		}
		// Keduanya NullString: IDBUSINESS memang NULL pada dua baris produksi, dan
		// GROUPBUSINESS pun nullable.
		result = append(result, masterxol.Business{ID: id.String, Name: name.String})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri bisnis master XOL: %w", err)
	}
	return result, nil
}

func (r *Repo) layerOf(ctx context.Context, masterID string) ([]masterxol.Layer, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_layer_list"), masterID)
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca layer master XOL: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterxol.Layer
	for rows.Next() {
		var (
			id                            string
			name                          sql.NullString
			limit, excess, convertedLimit sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &limit, &excess, &convertedLimit); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris layer: %w", err)
		}
		result = append(result, masterxol.Layer{
			ID:             strings.TrimSpace(id),
			Name:           name.String,
			Limit:          masterxol.Amount(limit.Int64),
			Excess:         masterxol.Amount(excess.Int64),
			ConvertedLimit: masterxol.Amount(convertedLimit.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri layer master XOL: %w", err)
	}

	// Diurutkan numerik di sini, dengan alasan yang sama seperti ID induk.
	sort.SliceStable(result, func(i, j int) bool { return lessByNumber(result[i].ID, result[j].ID) })

	for index := range result {
		reinsurer, err := r.reinsurerOf(ctx, result[index].ID)
		if err != nil {
			return nil, err
		}
		result[index].Reinsurer = reinsurer
	}
	return result, nil
}

func (r *Repo) reinsurerOf(ctx context.Context, layerID string) ([]masterxol.Reinsurer, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_reas_list"), layerID)
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca reas layer: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterxol.Reinsurer
	for rows.Next() {
		var (
			id, name sql.NullString
			share    sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &share); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris reas: %w", err)
		}
		result = append(result, masterxol.Reinsurer{
			ID:    id.String,
			Name:  name.String,
			Share: masterxol.Share(share.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri reas layer: %w", err)
	}
	return result, nil
}

// Save menyimpan satu induk beserta seluruh anaknya dalam SATU transaksi.
//
// # Satu transaksi, bukan sebelas commit
//
// Sistem lama memanggil procedure berkali-kali dalam satu penyimpanan — sekali untuk
// induk, sekali per grup bisnis, sekali per lapisan, dan sekali per baris reas — dan
// procedure itu COMMIT sendiri di dalam dua cabangnya. Sebuah induk dengan 4 bisnis, 3
// lapisan, dan 10 reas menempuh 18 pemanggilan yang masing-masing berdiri sendiri.
//
// Akibatnya: kegagalan di tengah meninggalkan induk yang tersimpan sebagian — sebagian
// lapisan ada, sisanya tidak, dan tidak ada pesan yang menyebut sampai mana ia berhasil.
// Di sini keseluruhannya satu transaksi, sehingga yang tersisa saat gagal adalah keadaan
// sebelum penyimpanan (`D-68`).
func (r *Repo) Save(ctx context.Context, master masterxol.Master) (masterxol.Master, error) {
	master = master.Clean()

	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterxol.Master{}, fmt.Errorf("masterxol/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = transaction.Rollback() }()

	if master.ID == "" {
		master.ID, err = insertMaster(ctx, transaction, master)
	} else {
		err = updateMaster(ctx, transaction, master)
	}
	if err != nil {
		return masterxol.Master{}, err
	}

	if err := saveBusiness(ctx, transaction, master); err != nil {
		return masterxol.Master{}, err
	}
	if master.Layer, err = saveLayer(ctx, transaction, master); err != nil {
		return masterxol.Master{}, err
	}

	if err := transaction.Commit(); err != nil {
		return masterxol.Master{}, fmt.Errorf("masterxol/sqlstore: menyimpan master XOL: %w", err)
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya — termasuk
	// kolom komite, yang Simpan tidak sentuh tetapi layar tetap perlu tampilkan.
	return r.Get(ctx, master.ID)
}

func insertMaster(ctx context.Context, tx *sql.Tx, master masterxol.Master) (string, error) {
	existing, err := lockIDs(ctx, tx, "xol_lock_master_ids")
	if err != nil {
		return "", err
	}
	id := masterxol.NextID(existing)

	// Pemeriksaan terakhir sebelum menyisipkan. Ia dapat diandalkan karena seluruh ID
	// sudah dibaca DAN dikunci di dalam transaksi yang sama.
	for _, taken := range existing {
		if taken == id {
			return "", masterxol.ErrIDTaken
		}
	}

	_, err = tx.ExecContext(ctx, getQuery("xol_insert_master"),
		id, nullIfEmpty(master.Name), nullIfEmpty(master.Year),
		int64(master.ExchangeRate), nullIfEmpty(string(master.Type)))
	if err != nil {
		return "", fmt.Errorf("masterxol/sqlstore: menyisipkan master XOL: %w", err)
	}
	return id, nil
}

func updateMaster(ctx context.Context, tx *sql.Tx, master masterxol.Master) error {
	result, err := tx.ExecContext(ctx, getQuery("xol_update_master"),
		nullIfEmpty(master.Name), nullIfEmpty(master.Year),
		int64(master.ExchangeRate), nullIfEmpty(string(master.Type)), master.ID)
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: mengubah master XOL: %w", err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi. Procedure lama melakukan persis itu — ia
	// mengembalikan `'Data Sudah Diupdate dengan ID : ' || IDMST` tanpa memeriksanya.
	touched, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if touched == 0 {
		return masterxol.ErrNotFound
	}
	return nil
}

// saveBusiness menyisipkan grup bisnis yang belum ada.
//
// Hanya menyisipkan, tidak pernah mengubah — meniru `INSERT_UPDATE_MST_XOL.prc:52-66`,
// yang memeriksa `count(1)` lalu diam saja bila barisnya sudah ada. Memang tidak ada yang
// dapat diubah: kedua kolomnya adalah kuncinya sendiri.
func saveBusiness(ctx context.Context, tx *sql.Tx, master masterxol.Master) error {
	for _, b := range master.Business {
		var total int
		err := tx.QueryRowContext(ctx, getQuery("xol_business_count"), master.ID, nullIfEmpty(b.ID)).Scan(&total)
		if err != nil {
			return fmt.Errorf("masterxol/sqlstore: memeriksa bisnis master XOL: %w", err)
		}
		if total > 0 {
			continue
		}
		_, err = tx.ExecContext(ctx, getQuery("xol_insert_business"),
			master.ID, nullIfEmpty(b.Name), nullIfEmpty(b.ID))
		if err != nil {
			return fmt.Errorf("masterxol/sqlstore: menyisipkan bisnis master XOL: %w", err)
		}
	}
	return nil
}

// saveLayer menyisipkan atau mengubah lapisan beserta reas-nya, dan mengembalikan
// daftarnya lengkap dengan nomor yang baru diterbitkan.
func saveLayer(ctx context.Context, tx *sql.Tx, master masterxol.Master) ([]masterxol.Layer, error) {
	var used []string
	needID := false
	for _, l := range master.Layer {
		if l.ID == "" {
			needID = true
			break
		}
	}
	if needID {
		var err error
		if used, err = lockIDs(ctx, tx, "xol_lock_layer_ids"); err != nil {
			return nil, err
		}
	}

	result := make([]masterxol.Layer, 0, len(master.Layer))
	for _, l := range master.Layer {
		converted := masterxol.ConvertedLimit(l.Limit, master.ExchangeRate)

		if l.ID == "" {
			l.ID = masterxol.NextID(used)
			used = append(used, l.ID)
			_, err := tx.ExecContext(ctx, getQuery("xol_insert_layer"),
				master.ID, l.ID, nullIfEmpty(l.Name),
				int64(l.Limit), int64(l.Excess), int64(converted))
			if err != nil {
				return nil, fmt.Errorf("masterxol/sqlstore: menyisipkan layer: %w", err)
			}
		} else {
			_, err := tx.ExecContext(ctx, getQuery("xol_update_layer"),
				nullIfEmpty(l.Name), int64(l.Limit), int64(l.Excess), int64(converted), l.ID)
			if err != nil {
				return nil, fmt.Errorf("masterxol/sqlstore: mengubah layer: %w", err)
			}
		}
		l.ConvertedLimit = converted

		if err := saveReinsurer(ctx, tx, l); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

// saveReinsurer menyisipkan atau mengubah baris reas sebuah lapisan.
func saveReinsurer(ctx context.Context, tx *sql.Tx, layer masterxol.Layer) error {
	for _, reas := range layer.Reinsurer {
		var total int
		err := tx.QueryRowContext(ctx, getQuery("xol_reas_count"), layer.ID, nullIfEmpty(reas.ID)).Scan(&total)
		if err != nil {
			return fmt.Errorf("masterxol/sqlstore: memeriksa reas layer: %w", err)
		}

		if total > 0 {
			_, err = tx.ExecContext(ctx, getQuery("xol_update_reas"),
				nullIfEmpty(reas.Name), int64(reas.Share), layer.ID, nullIfEmpty(reas.ID))
			if err != nil {
				return fmt.Errorf("masterxol/sqlstore: mengubah reas layer: %w", err)
			}
			continue
		}
		_, err = tx.ExecContext(ctx, getQuery("xol_insert_reas"),
			layer.ID, nullIfEmpty(reas.Name), nullIfEmpty(reas.ID), int64(reas.Share))
		if err != nil {
			return fmt.Errorf("masterxol/sqlstore: menyisipkan reas layer: %w", err)
		}
	}
	return nil
}

// DeleteMaster menghapus satu induk beserta seluruh anaknya, dalam satu transaksi.
//
// Urutannya mengikat: reas lebih dulu, lalu lapisan, lalu bisnis, baru induknya. Membalik
// dua yang pertama akan menghilangkan satu-satunya cara mengetahui IDLAYER mana yang
// milik induk ini.
func (r *Repo) DeleteMaster(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)

	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: memulai transaksi hapus: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	for _, name := range []string{
		"xol_delete_reas_of_master",
		"xol_delete_layer_of_master",
		"xol_delete_business_of_master",
	} {
		if _, err := transaction.ExecContext(ctx, getQuery(name), id); err != nil {
			return fmt.Errorf("masterxol/sqlstore: menghapus anak master XOL (%s): %w", name, err)
		}
	}

	result, err := transaction.ExecContext(ctx, getQuery("xol_delete_master"), id)
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: menghapus master XOL: %w", err)
	}
	touched, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: membaca jumlah baris terhapus: %w", err)
	}
	if touched == 0 {
		return masterxol.ErrNotFound
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("masterxol/sqlstore: menyelesaikan hapus master XOL: %w", err)
	}
	return nil
}

// DeleteBusiness menghapus satu baris grup bisnis.
//
// # Baris ber-IDBUSINESS kosong TIDAK dapat dihapus, dan itu diwarisi apa adanya
//
// Kueri hapus lama mencocokkan `IDBUSINESS = {TempMst.CaseID}`, dan pencocokan `= NULL`
// tidak pernah bernilai benar. Dua baris produksi milik induk 10009 karena itu **sudah
// tidak dapat dihapus dari layar Pega hari ini** — keduanya ber-IDBUSINESS NULL, ditambahkan
// `ShowDetailGroupBisnisXol_Act` sebagai baris "TREATY INWARD" tanpa ID.
//
// Batasan yang sama dipertahankan (`P-5`), dan layar menyatakannya: tombol hapus pada
// baris seperti itu dimatikan beserta keterangannya, bukan dibiarkan menekan tanpa akibat.
// Membersihkannya menempuh jalur DBA (`D-63`), bukan layar ini.
func (r *Repo) DeleteBusiness(ctx context.Context, masterID, businessID string) error {
	masterID = strings.TrimSpace(masterID)
	businessID = strings.TrimSpace(businessID)

	if businessID == "" {
		return masterxol.ErrNotFound
	}

	_, err := r.db.ExecContext(ctx, getQuery("xol_delete_business"), masterID, businessID)
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: menghapus bisnis master XOL: %w", err)
	}
	// Jumlah baris tidak diperiksa: MST_XOL_BUSINESS tidak punya kunci unik, sehingga
	// "tidak ada yang terhapus" dan "barisnya memang sudah tidak ada" tidak dapat
	// dibedakan dengan berarti. Hasil akhirnya sama — baris itu tidak ada lagi.
	return nil
}

// DeleteLayer menghapus satu lapisan beserta seluruh reas-nya.
func (r *Repo) DeleteLayer(ctx context.Context, masterID, layerID string) error {
	masterID = strings.TrimSpace(masterID)
	layerID = strings.TrimSpace(layerID)

	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: memulai transaksi hapus layer: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	if err := ownedLayer(ctx, transaction, masterID, layerID); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, getQuery("xol_delete_reas_of_layer"), layerID); err != nil {
		return fmt.Errorf("masterxol/sqlstore: menghapus reas layer: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, getQuery("xol_delete_layer"), masterID, layerID); err != nil {
		return fmt.Errorf("masterxol/sqlstore: menghapus layer: %w", err)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("masterxol/sqlstore: menyelesaikan hapus layer: %w", err)
	}
	return nil
}

// DeleteReinsurer menghapus satu baris reas dari sebuah lapisan.
func (r *Repo) DeleteReinsurer(ctx context.Context, layerID, reinsurerID string) error {
	_, err := r.db.ExecContext(ctx, getQuery("xol_delete_reas"),
		strings.TrimSpace(layerID), strings.TrimSpace(reinsurerID))
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: menghapus reas: %w", err)
	}
	return nil
}

// ownedLayer memastikan sebuah lapisan benar-benar milik induk yang disebut.
//
// Tanpa pemeriksaan ini, permintaan hapus dapat menyebut induk A beserta lapisan milik
// induk B — dan penghapusannya tetap berjalan, karena IDLAYER sudah cukup mengenali
// barisnya sendirian.
func ownedLayer(ctx context.Context, tx *sql.Tx, masterID, layerID string) error {
	var found string
	err := tx.QueryRowContext(ctx, getQuery("xol_layer_owner"), masterID, layerID).Scan(&found)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterxol.ErrLayerNotFound
	case err != nil:
		return fmt.Errorf("masterxol/sqlstore: memeriksa pemilik layer: %w", err)
	}
	return nil
}

// ListYear membaca pilihan tahun, terurut menurun — tahun terbaru lebih dulu.
func (r *Repo) ListYear(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_year_list"))
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca daftar tahun: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var year sql.NullString
		if err := rows.Scan(&year); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris tahun: %w", err)
		}
		if clean := strings.TrimSpace(year.String); clean != "" {
			result = append(result, clean)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri daftar tahun: %w", err)
	}

	// Menurun: tahun yang sedang berjalan hampir selalu yang dicari, dan menaruhnya di
	// pucuk menghemat gulir pada daftar 36 baris.
	sort.SliceStable(result, func(i, j int) bool { return lessByNumber(result[j], result[i]) })
	return result, nil
}

// ListBusinessGroup membaca pilihan grup bisnis untuk sebuah Type XOL.
func (r *Repo) ListBusinessGroup(ctx context.Context, t masterxol.Type) ([]masterxol.Business, error) {
	pattern := masterxol.BusinessGroupPattern(t)

	rows, err := r.db.QueryContext(ctx, getQuery("xol_business_group_list"),
		pattern[0], pattern[1], pattern[2])
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: membaca pilihan grup bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterxol.Business
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca baris grup bisnis: %w", err)
		}
		result = append(result, masterxol.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri pilihan grup bisnis: %w", err)
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })

	// Baris TREATY INWARD ditambahkan tanpa ID, meniru
	// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml` yang menambahkannya pada setiap
	// Type. Ia memang tidak ada di POOLDATA.BUSINESSGROUP — diperiksa pada 2026-09-20.
	result = append(result, masterxol.Business{Name: masterxol.TreatyInwardName})
	return result, nil
}

// SubmitToCommittee mencatat pengajuan sebuah induk ke komite.
func (r *Repo) SubmitToCommittee(ctx context.Context, id, pic, remark string) error {
	result, err := r.db.ExecContext(ctx, getQuery("xol_submit_committee"),
		nullIfEmpty(strings.TrimSpace(pic)),
		string(masterxol.CommitteePending),
		nullIfEmpty(strings.TrimSpace(remark)),
		strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: mencatat pengajuan komite: %w", err)
	}

	touched, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("masterxol/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if touched == 0 {
		return masterxol.ErrNotFound
	}
	return nil
}

// lockIDs membaca seluruh nomor sambil menguncinya di dalam transaksi berjalan.
func lockIDs(ctx context.Context, tx *sql.Tx, queryName string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQuery(queryName))
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: mengunci daftar nomor (%s): %w", queryName, err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca nomor terkunci: %w", err)
		}
		// Dirapikan di sini juga: bila kolomnya ternyata CHAR, nilainya kembali membawa
		// padding dan "10001  " tidak akan terbaca sebagai angka.
		result = append(result, strings.TrimSpace(id.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri nomor terkunci: %w", err)
	}
	return result, nil
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanMaster(rows rowScanner) (masterxol.Master, error) {
	var (
		id                              string
		name, year, typeXOL             sql.NullString
		pic, committeeStatus, committee sql.NullString
		remarkPIC, remarkCommittee      sql.NullString
		exchangeRate                    sql.NullInt64
	)
	err := rows.Scan(&id, &name, &year, &exchangeRate, &typeXOL,
		&pic, &committeeStatus, &committee, &remarkPIC, &remarkCommittee)
	if err != nil {
		return masterxol.Master{}, err
	}

	// Seluruh kolom kecuali ID dibaca sebagai nullable, dan itu bukan kehati-hatian
	// berlebih: ALL_TAB_COLUMNS pada 2026-09-20 menyatakan hanya ID yang NOT NULL, dan
	// produksi memang memuat TYPEXOL, STSKOMITE, PIC, serta KOMITE yang kosong.
	return masterxol.Master{
		ID:              id,
		Name:            name.String,
		Year:            year.String,
		ExchangeRate:    masterxol.Amount(exchangeRate.Int64),
		Type:            masterxol.Type(typeXOL.String),
		PIC:             pic.String,
		CommitteeStatus: masterxol.CommitteeStatus(committeeStatus.String),
		Committee:       committee.String,
		RemarkPIC:       remarkPIC.String,
		RemarkCommittee: remarkCommittee.String,
	}.Clean(), nil
}

// nullIfEmpty mengirim NULL, bukan string kosong, untuk isian yang tidak diisi.
//
// Perbedaannya nyata di Oracle: string kosong dan NULL diperlakukan sama pada VARCHAR2,
// tetapi tidak di PostgreSQL — dan `D-20` menuntut satu set SQL yang berjalan di
// keduanya. Mengirim NULL secara eksplisit membuat keduanya berperilaku sama, dan
// menyamai apa yang tersimpan hari ini.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// sortMasterByID mengurutkan numerik, bukan leksikografis.
func sortMasterByID(list []masterxol.Master) {
	sort.SliceStable(list, func(i, j int) bool { return lessByNumber(list[i].ID, list[j].ID) })
}

// lessByNumber membandingkan dua nomor sebagai angka, dan jatuh ke perbandingan teks
// bila salah satunya bukan angka.
//
// Perlu karena nomor di modul ini tidak bernol di depan: sebagai teks, "10010" mendahului
// "1009".
func lessByNumber(left, right string) bool {
	leftValue, leftErr := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
	rightValue, rightErr := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
	if leftErr != nil || rightErr != nil {
		return left < right
	}
	return leftValue < rightValue
}

var _ masterxol.Repo = (*Repo)(nil)

// OrphanCount menghitung baris anak yang induknya sudah tidak ada.
//
// Dipakai MODE PERIKSA saja, bukan jalur layar. Ia mengukur akibat cacat sistem lama:
// hapus di sana tidak berkaskade, sehingga induk yang dibuang meninggalkan anaknya —
// induk 10003 sudah terhapus di produksi dan lapisan serta baris bisnisnya masih ada.
//
// Aplikasi ini berkaskade sehingga tidak menambah yatim baru; yang sudah ada tetap perlu
// dilaporkan supaya pembersihannya dapat diajukan ke DBA (`D-63`).
func (r *Repo) OrphanCount(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_orphan_count"))
	if err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menghitung baris yatim: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := map[string]int{}
	for rows.Next() {
		var kind string
		var total int
		if err := rows.Scan(&kind, &total); err != nil {
			return nil, fmt.Errorf("masterxol/sqlstore: membaca jumlah baris yatim: %w", err)
		}
		result[kind] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterxol/sqlstore: menelusuri jumlah baris yatim: %w", err)
	}
	return result, nil
}

// CheckTable menguji apakah keempat tabel ada dan kolomnya dapat dibaca akun aplikasi,
// tanpa mengambil satu baris pun.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("xol_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}
