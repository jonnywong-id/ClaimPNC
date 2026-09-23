package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/mastermasking"
)

// Nilai teks yang tersimpan di kolom, dibaca langsung dari basis data pada 2026-09-20.
//
// Ketiganya sengaja menjadi konstanta bernama, bukan literal yang tersebar: nilai-nilai
// inilah yang membedakan "boleh melihat nomor KTP" dari "tidak boleh", dan satu salah
// ketik pada salah satunya akan diam-diam mencabut atau memberikan kewenangan.
const (
	// yes dan no mengisi STS_KTP, STS_EMAIL, dan STS_NOTELP.
	//
	// BUKAN '1'/'0'. Dugaan itu datang dari blok yang DIKOMENTARI di
	// `Database/UPDATE_LOG_PROTEKSI.prc:126-131` dan terbukti salah saat 25 baris portal
	// ASM dibaca: yang ada hanya 'Ya' dan 'Tidak'.
	yes = "Ya"
	no  = "Tidak"

	// active dan inactive mengisi STS_AKTF.
	//
	// `Activity/DeleteMasking-Act.xml` menetapkan 'TIDAK AKTIF' sebagai nilai
	// penonaktifan, dan `RDB List/GetTipeProteksi-SQL.xml` menyaring `STS_AKTF = 'AKTIF'`.
	active   = "AKTIF"
	inactive = "TIDAK AKTIF"
)

// legacyPassword adalah nilai yang ditulis ke kolom PASSWORD.
//
// # Kenapa literal, dan kenapa justru literal INI
//
// Keputusan Work Owner 2026-09-20: kolomnya diisi persis seperti sistem lama supaya baris
// lama dan baru seragam. Nilainya bukan pilihan kita —
// `Activity/InsermaskingDataKlaimPnc_-Act.xml` menetapkan
// `InputData.ResponseCode := "saya"`, dan nilai itulah yang mengalir ke parameter
// `T_PASSWORD` lalu ke kolom PASSWORD. Pembacaan basis data membenarkannya: seluruh 25
// baris berisi nilai sepanjang tepat 4 karakter.
//
// # Yang harus diketahui siapa pun yang membacanya kelak
//
// Ia BUKAN kata sandi, dan tidak pernah menjadi kata sandi:
//
//   - Tidak ada isian PASSWORD di layar Pega — nol kemunculan di harness maupun section.
//     Pengguna tidak pernah mengetiknya.
//   - Tidak pernah diperiksa. Seluruh logika verifikasinya dikomentari di
//     `Database/UPDATE_LOG_PROTEKSI.prc:114-122`.
//   - Nilainya kata coba-coba yang tertinggal di jalur produksi.
//
// Karena itu ia ditulis sebagai konstanta bernama dengan penjelasan ini, bukan sebagai
// literal telanjang di tengah kode — supaya ia tidak pernah lagi terbaca sebagai rahasia,
// dan supaya menghapusnya kelak cukup menyentuh satu tempat.
//
// Kolom ini TIDAK pernah dibaca modul ini, TIDAK pernah dikirim ke peramban, dan TIDAK
// pernah tampil di layar.
const legacyPassword = "saya"

// Repo membaca dan menulis POOLDATA.MST_PROTEKSI_DATA_PNC, serta membaca POOLDATA.BRANCH.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master Masking adalah
// SATU-SATUNYA penulis tabel ini di sistem lama; daftar lengkap rule yang menyentuhnya ada
// di kepala berkas .sql. Memindahkan layar itu ke sini memindahkan kepemilikan tabelnya
// secara utuh, dan Pega berubah menjadi pembaca saja.
//
// POOLDATA.BRANCH tetap milik sistem lain dan hanya dibaca.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// insertAttempts membatasi percobaan ulang saat nomor ID diperebutkan.
//
// `MAX(TO_NUMBER(ID_MST))+1` dapat membaca nilai yang sama pada dua permintaan bersamaan,
// dan tabel ini tidak punya indeks unik yang menolaknya. Tiga percobaan sudah cukup untuk
// keadaan yang wajar; lebih dari itu menandakan persoalan lain, dan mengulanginya terus
// hanya menahan permintaan pengguna tanpa hasil.
const insertAttempts = 3

// List membaca baris yang cocok dengan penyaring.
func (r *Repo) List(ctx context.Context, filter mastermasking.Filter) ([]mastermasking.Masking, error) {
	filter = filter.Clean()

	// Pola LIKE dibentuk di sini, lalu DIIKAT sebagai nilai — bukan dirangkai ke teks SQL.
	// Layar lama merangkainya (`{ASIS:InputSearch.CARI1}`), dan itu membuat setiap kata
	// kunci pengguna menjadi bagian dari pernyataan SQL-nya sendiri.
	pattern := "%" + strings.ToUpper(filter.Keyword) + "%"

	// Nilai status hanya berarti saat tipe pencariannya status. Untuk tipe lain ia diisi
	// teks yang tidak mungkin cocok — bukan dibiarkan kosong — supaya cabang CASE yang
	// tidak dipakai tetap punya nilai terikat yang sah.
	status := ""
	if filter.By == mastermasking.SearchByStatus {
		if wanted, valid := filter.StatusKeyword(); valid {
			status = wanted
		}
	}

	rows, err := r.db.QueryContext(ctx, getQuery("masking_list"),
		string(filter.By), pattern, pattern, status)
	if err != nil {
		return nil, fmt.Errorf("mastermasking/sqlstore: membaca daftar masking: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastermasking.Masking
	for rows.Next() {
		masking, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mastermasking/sqlstore: membaca baris masking: %w", err)
		}
		result = append(result, masking)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastermasking/sqlstore: menelusuri daftar masking: %w", err)
	}
	return result, nil
}

// Get membaca satu baris menurut ID_MST.
func (r *Repo) Get(ctx context.Context, id string) (mastermasking.Masking, error) {
	masking, err := scanRow(r.db.QueryRowContext(ctx, getQuery("masking_get"), id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mastermasking.Masking{}, mastermasking.ErrNotFound
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: membaca masking %q: %w", id, err)
	}
	return masking, nil
}

// FindByPair mencari baris menurut pasangan cabang+login.
func (r *Repo) FindByPair(ctx context.Context, branchID, login string) (mastermasking.Masking, error) {
	masking, err := scanRow(r.db.QueryRowContext(ctx, getQuery("masking_find_pair"),
		strings.ToUpper(strings.TrimSpace(branchID)),
		strings.ToUpper(strings.TrimSpace(login))))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mastermasking.Masking{}, mastermasking.ErrNotFound
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: mencari masking cabang %q pengguna %q: %w", branchID, login, err)
	}
	return masking, nil
}

// Insert menyisipkan baris baru beserta ID yang dibentuk di dalam transaksi yang sama.
//
// Nomor dibaca dan dipakai dalam SATU transaksi supaya jeda antara membaca `MAX+1` dan
// menyisipkannya sesempit mungkin. Itu memperkecil peluang bentrok, tidak menutupnya —
// penutupnya adalah indeks unik yang belum ada, dan itu dicatat terbuka.
func (r *Repo) Insert(ctx context.Context, m mastermasking.Masking) (mastermasking.Masking, error) {
	m = m.Clean()

	var lastErr error
	for attempt := 0; attempt < insertAttempts; attempt++ {
		saved, err := r.insertOnce(ctx, m)
		if err == nil {
			return saved, nil
		}
		lastErr = err
		if !errors.Is(err, mastermasking.ErrIDTaken) {
			return mastermasking.Masking{}, err
		}
		// Nomor direbut permintaan lain. Nomor berikutnya dibaca ulang, bukan ditambah
		// sendiri: permintaan yang menang mungkin bukan satu-satunya yang berhasil.
	}
	return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: menyisipkan masking setelah %d percobaan: %w", insertAttempts, lastErr)
}

func (r *Repo) insertOnce(ctx context.Context, m mastermasking.Masking) (mastermasking.Masking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: membuka transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var next int64
	if err := tx.QueryRowContext(ctx, getQuery("masking_next_id")).Scan(&next); err != nil {
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: mengambil nomor berikutnya: %w", err)
	}
	if next <= 0 {
		return mastermasking.Masking{}, mastermasking.ErrNoSequence
	}
	id := strconv.FormatInt(next, 10)

	if _, err := tx.ExecContext(ctx, getQuery("masking_insert"),
		id, m.BranchID, m.Login, m.Module, m.SubModule,
		m.SearchQuota, m.ViewQuota, flag(m.ViewIDCard), flag(m.ViewEmail), flag(m.ViewPhone),
		status(m.Active), legacyPassword, m.InputBy, m.InputAt,
	); err != nil {
		return mastermasking.Masking{}, translateWriteError(err, "menyisipkan masking")
	}

	if err := tx.Commit(); err != nil {
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/sqlstore: menyimpan masking: %w", err)
	}

	m.ID = id
	return r.Get(ctx, id)
}

// Update mengubah baris yang sudah ada.
func (r *Repo) Update(ctx context.Context, m mastermasking.Masking) (mastermasking.Masking, error) {
	m = m.Clean()

	result, err := r.db.ExecContext(ctx, getQuery("masking_update"),
		m.BranchID, m.Login, m.Module, m.SubModule,
		m.SearchQuota, m.ViewQuota, flag(m.ViewIDCard), flag(m.ViewEmail), flag(m.ViewPhone),
		status(m.Active), legacyPassword, m.InputBy, m.InputAt, m.ID,
	)
	if err != nil {
		return mastermasking.Masking{}, translateWriteError(err, "mengubah masking")
	}
	if err := requireOneRow(result, m.ID); err != nil {
		return mastermasking.Masking{}, err
	}
	return r.Get(ctx, m.ID)
}

// SetActive mengaktifkan atau menonaktifkan satu baris.
func (r *Repo) SetActive(ctx context.Context, id string, isActive bool, by string, at time.Time) (mastermasking.Masking, error) {
	result, err := r.db.ExecContext(ctx, getQuery("masking_set_active"),
		status(isActive), by, at, id)
	if err != nil {
		return mastermasking.Masking{}, translateWriteError(err, "mengubah status masking")
	}
	if err := requireOneRow(result, id); err != nil {
		return mastermasking.Masking{}, err
	}
	return r.Get(ctx, id)
}

// ListBranches membaca pilihan cabang yang cocok dengan kata kunci.
func (r *Repo) ListBranches(ctx context.Context, keyword string, limit int) ([]mastermasking.Branch, error) {
	if limit <= 0 {
		limit = 1
	}

	// Kata kunci kosong dikirim sebagai NULL, bukan sebagai "%%".
	//
	// Bedanya nyata: `UPPER(BRANCHNAME) LIKE '%%'` memaksa Oracle memeriksa setiap baris,
	// sedangkan `:1 IS NULL` membuat syaratnya benar tanpa menyentuh kolom mana pun.
	var pattern any
	var value any
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		pattern = nil
		value = nil
	} else {
		pattern = "%" + strings.ToUpper(keyword) + "%"
		value = pattern
	}

	rows, err := r.db.QueryContext(ctx, getQuery("branch_list"), value, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("mastermasking/sqlstore: membaca daftar cabang: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastermasking.Branch
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastermasking/sqlstore: membaca baris cabang: %w", err)
		}
		result = append(result, mastermasking.Branch{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastermasking/sqlstore: menelusuri daftar cabang: %w", err)
	}
	return result, nil
}

// BranchExists menyatakan apakah kode cabang ada di POOLDATA.BRANCH.
func (r *Repo) BranchExists(ctx context.Context, branchID string) (bool, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("branch_exists"), strings.TrimSpace(branchID)).Scan(&total); err != nil {
		return false, fmt.Errorf("mastermasking/sqlstore: memeriksa cabang %q: %w", branchID, err)
	}
	return total > 0, nil
}

// scanner menyatukan *sql.Row dan *sql.Rows supaya pembacaan satu baris ditulis sekali.
type scanner interface {
	Scan(dest ...any) error
}

// scanRow membaca satu baris hasil kueri menjadi tipe domain.
//
// Seluruh kolom dibaca sebagai NullString — termasuk yang tidak pernah kosong hari ini.
// Setiap kolom tabel ini NULLABLE (diperiksa ke ALL_TAB_COLUMNS pada 2026-09-20), dan
// BRANCHNAME datang dari LEFT JOIN yang memang dapat tidak menemukan pasangannya.
// Membaca ke string biasa akan membuat satu baris berkolom NULL menggagalkan seluruh
// pemuatan daftar.
func scanRow(row scanner) (mastermasking.Masking, error) {
	var (
		id, branchID, branchName         sql.NullString
		login, module, subModule         sql.NullString
		searchQuota, viewQuota           sql.NullInt64
		viewIDCard, viewEmail, viewPhone sql.NullString
		activeStatus, inputBy            sql.NullString
		inputAt                          sql.NullTime
	)

	if err := row.Scan(&id, &branchID, &branchName, &login, &module, &subModule,
		&searchQuota, &viewQuota, &viewIDCard, &viewEmail, &viewPhone,
		&activeStatus, &inputBy, &inputAt); err != nil {
		return mastermasking.Masking{}, err
	}

	return mastermasking.Masking{
		ID:          strings.TrimSpace(id.String),
		BranchID:    strings.TrimSpace(branchID.String),
		BranchName:  strings.TrimSpace(branchName.String),
		Login:       strings.TrimSpace(login.String),
		Module:      strings.TrimSpace(module.String),
		SubModule:   strings.TrimSpace(subModule.String),
		SearchQuota: int(searchQuota.Int64),
		ViewQuota:   int(viewQuota.Int64),
		ViewIDCard:  isYes(viewIDCard.String),
		ViewEmail:   isYes(viewEmail.String),
		ViewPhone:   isYes(viewPhone.String),
		Active:      isActiveText(activeStatus.String),
		InputBy:     strings.TrimSpace(inputBy.String),
		InputAt:     inputAt.Time,
	}, nil
}

// isYes menerjemahkan STS_KTP / STS_EMAIL / STS_NOTELP menjadi keputusan.
//
// Hanya 'Ya' yang berarti boleh. Apa pun selain itu — 'Tidak', kosong, NULL, atau nilai
// yang belum pernah terlihat — berarti TIDAK boleh.
//
// Arahnya disengaja dan tidak boleh dibalik: bila kelak muncul nilai yang tak dikenal,
// akibatnya adalah data nasabah tetap tersamar. Aman salah ke arah menyamarkan, tidak
// pernah ke arah membuka.
func isYes(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), yes)
}

// isActiveText menerjemahkan STS_AKTF menjadi keputusan.
//
// Hanya 'AKTIF' yang berarti berlaku — sama dengan penyaring
// `RDB List/GetTipeProteksi-SQL.xml` dan `CekmaskingDataPerLoginUserKlaim`. Arahnya sama
// hati-hatinya dengan isYes: nilai tak dikenal diperlakukan sebagai tidak berlaku.
func isActiveText(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), active)
}

// flag mengubah keputusan menjadi teks yang tersimpan di kolom.
func flag(allowed bool) string {
	if allowed {
		return yes
	}
	return no
}

// status mengubah keputusan menjadi teks STS_AKTF.
func status(isActive bool) string {
	if isActive {
		return active
	}
	return inactive
}

// requireOneRow menerjemahkan "tidak ada baris tersentuh" menjadi ErrNotFound.
//
// Tanpa ini, mengubah baris yang sudah tidak ada akan tampak berhasil — dan pengguna
// mengira kewenangan sudah dicabut padahal tidak ada yang berubah.
func requireOneRow(result sql.Result, id string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mastermasking/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if affected == 0 {
		return mastermasking.ErrNotFound
	}
	return nil
}

// translateWriteError menerjemahkan galat basis data menjadi galat domain yang dapat
// dibedakan pemanggil.
//
// ORA-00001 adalah pelanggaran keunikan. Tabel ini BELUM punya indeks unik — diperiksa ke
// ALL_INDEXES pada 2026-09-20 — sehingga hari ini galat itu tidak akan muncul. Ia tetap
// diterjemahkan karena indeks uniknya diusulkan ke DBA, dan begitu dipasang, penanganannya
// sudah ada di tempatnya alih-alih menjadi "galat internal" yang membingungkan.
func translateWriteError(err error, action string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "ORA-00001") {
		// Indeks mana yang dilanggar tidak dibedakan di sini: yang mungkin hanya dua, dan
		// keduanya berarti hal yang sama bagi pengguna — baris untuk pasangan itu sudah
		// ada. Membedakannya menuntut membaca nama indeks dari teks galat, dan teks itu
		// bukan kontrak yang dijamin.
		return mastermasking.ErrIDTaken
	}
	return fmt.Errorf("mastermasking/sqlstore: %s: %w", action, err)
}
