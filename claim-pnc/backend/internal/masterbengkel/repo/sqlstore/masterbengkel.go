// Package sqlstore memenuhi seam masterbengkel.Store dengan SQL.
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

	"claim-pnc/internal/masterbengkel"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// sequenceWidth adalah lebar nomor urut pada ID_BENGKEL.
//
// Meniru `Database/PEGA_M_BENGKEL_HE.prc:19` persis:
//
//	id_bengkel := id_site || lpad(to_Char(BENGKEL_HE_SEQ.nextval),10,'0');
//
// Sepuluh digit, ditambal nol di depan. Pembentukannya dilakukan di Go dan bukan di SQL
// supaya kuerinya tetap portabel — LPAD ada di kedua basis data, tetapi menyusun kunci
// di dalam kueri berarti bentuk kuncinya tersebar ke berkas .sql dan ke sini sekaligus.
const sequenceWidth = 10

// Repo membaca dan menulis POOLDATA.BENGKEL_HE, serta MEMBACA lima objek acuan.
//
// Satu struct memenuhi ketiga seam — masterbengkel.Repo, LookupRepo, dan IDSource —
// karena ketiganya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya
// ditempel — lihat berkas .sql untuk alasannya.
func (r *Repo) List(ctx context.Context, filter masterbengkel.Filter) ([]masterbengkel.Workshop, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		rows, err = r.db.QueryContext(ctx, getQuery("bengkel_list_search"),
			string(filter.Status), likePattern(keyword))
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("bengkel_list"), string(filter.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterbengkel.Workshop
	for rows.Next() {
		w, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterbengkel/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan ID_BENGKEL-nya.
func (r *Repo) Get(ctx context.Context, id string) (masterbengkel.Workshop, error) {
	row := r.db.QueryRowContext(ctx, getQuery("bengkel_get"), strings.TrimSpace(id))

	w, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}
	if err != nil {
		return masterbengkel.Workshop{}, fmt.Errorf("masterbengkel/sqlstore: membaca %q: %w", id, err)
	}
	return w, nil
}

// FindByName mencari baris menurut NAMA_BENGKEL-nya.
func (r *Repo) FindByName(ctx context.Context, name string) (masterbengkel.Workshop, error) {
	clean := strings.ToUpper(strings.TrimSpace(name))
	if clean == "" {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("bengkel_find_by_name"), clean)

	w, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}
	if err != nil {
		return masterbengkel.Workshop{}, fmt.Errorf("masterbengkel/sqlstore: mencari nama %q: %w", name, err)
	}
	return w, nil
}

// FindByLogin mencari baris menurut LOGIN_APLIKASI-nya.
func (r *Repo) FindByLogin(ctx context.Context, login string) (masterbengkel.Workshop, error) {
	clean := strings.ToUpper(strings.TrimSpace(login))
	if clean == "" {
		// Login kosong sah pada bengkel non-rekanan, dan tidak pernah dianggap bentrok.
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("bengkel_find_by_login"), clean)

	w, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}
	if err != nil {
		return masterbengkel.Workshop{}, fmt.Errorf("masterbengkel/sqlstore: mencari login %q: %w", login, err)
	}
	return w, nil
}

// Insert menolak nama dan login yang sudah dipakai, lalu menyisipkan barisnya.
//
// Ketiganya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi:
// pemeriksaan dan penyisipan di sini bukan tiga perkara melainkan satu, dan
// memisahkannya membuka kembali lubang balapan yang justru sedang dipersempit.
//
// SEBERAPA JAUH LUBANG ITU TERTUTUP — dinyatakan supaya tidak dikira selesai. FOR UPDATE
// tidak dapat mengunci baris yang belum ada: bila dua penambahan atas nama yang sama
// sama-sama menemukan nol baris, keduanya lolos. Yang benar-benar menutupnya adalah
// constraint unik pada NAMA_BENGKEL dan LOGIN_APLIKASI, dan itu menunggu DDL (R-08)
// serta prosedur perubahan skema (D-63).
func (r *Repo) Insert(ctx context.Context, w masterbengkel.Workshop) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	taken, err := locked(ctx, tx, "bengkel_lock_by_name", w.Name)
	if err != nil {
		return err
	}
	if taken {
		return fmt.Errorf("%w: %q", masterbengkel.ErrNameTaken, w.Name)
	}

	if strings.TrimSpace(w.Login) != "" {
		taken, err := locked(ctx, tx, "bengkel_lock_by_login", w.Login)
		if err != nil {
			return err
		}
		if taken {
			return fmt.Errorf("%w: %q", masterbengkel.ErrLoginTaken, w.Login)
		}
	}

	if _, err := tx.ExecContext(ctx, getQuery("bengkel_insert"), insertArguments(w)...); err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: menyisipkan %q: %w", w.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Pemeriksaan bentrok nama dan login TIDAK dilakukan di sini, dan itu mengikuti sistem
// lama: `ValidateMasterBengkel` dipanggil dari jalur penambahan, dan
// `ValidationLoginBengkel_act` memeriksa login terhadap daftar operator — bukan terhadap
// baris bengkel lain. Menambahkannya di sini akan menolak penyimpanan yang hari ini
// diterima. Lapisan aplikasi yang memeriksanya, dan ia mengecualikan baris itu sendiri.
func (r *Repo) Update(ctx context.Context, w masterbengkel.Workshop) error {
	result, err := r.db.ExecContext(ctx, getQuery("bengkel_update"), updateArguments(w)...)
	if err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: memperbarui %q: %w", w.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh
	// satu baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan
	// mengatakan "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return masterbengkel.ErrNotFound
	}
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
// berubah-ubah — lihat `bengkel_set_status` pada berkas .sql untuk alasannya.
//
// Transaksinya melingkupi seluruh baris supaya persetujuan borongan tidak pernah
// setengah jalan: bila baris kelima gagal, keempat yang sebelumnya ikut dibatalkan.
// Sistem lama tidak menjamin itu — `SetApprovalAllMaster` menjalankan satu RDB-List per
// baris tanpa transaksi yang melingkupinya.
func (r *Repo) SetStatus(ctx context.Context, id []string, status masterbengkel.ApprovalStatus) (int, error) {
	if len(id) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("masterbengkel/sqlstore: memulai transaksi keputusan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, one := range id {
		result, err := tx.ExecContext(ctx, getQuery("bengkel_set_status"),
			string(status), strings.TrimSpace(one))
		if err != nil {
			return 0, fmt.Errorf("masterbengkel/sqlstore: menetapkan status %q: %w", one, err)
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
		return 0, fmt.Errorf("masterbengkel/sqlstore: menutup transaksi keputusan: %w", err)
	}
	return changed, nil
}

// NextID menerbitkan ID_BENGKEL berikutnya.
//
// Bentuknya meniru `Database/PEGA_M_BENGKEL_HE.prc:11,19` persis: kode situs ditambah
// nomor urut sepuluh digit bertambal nol.
//
// Kedua kueri dijalankan di dalam SATU transaksi. Bukan demi keatomikan — sequence tidak
// dapat dibatalkan — melainkan supaya keduanya pasti dilayani koneksi yang sama; kode
// situs dan sequence yang berasal dari dua koneksi berbeda pada pool yang sama tetap
// benar, tetapi jaminannya tidak berasal dari mana pun selain kebetulan.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("masterbengkel/sqlstore: memulai transaksi penomoran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var site sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery("bengkel_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Persis keadaan yang ditangkap `PEGA_M_BENGKEL_HE.prc:12-16`, yang
			// menjawabnya dengan pesan galat lalu berhenti. Di sini ia juga berhenti —
			// ID tanpa kode situs akan bertabrakan dengan ID entitas lain.
			return "", errors.New("masterbengkel/sqlstore: POOLDATA.M_SITE_DATABASE tidak punya baris CURRENT_SITE='1'")
		}
		return "", fmt.Errorf("masterbengkel/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("bengkel_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("masterbengkel/sqlstore: mengambil nomor urut: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("masterbengkel/sqlstore: menutup transaksi penomoran: %w", err)
	}
	return masterbengkel.ComposeID(strings.TrimSpace(site.String), sequence, sequenceWidth), nil
}

// ListBranches membaca daftar cabang.
func (r *Repo) ListBranches(ctx context.Context) ([]masterbengkel.Branch, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("bengkel_branch_list"))
	if err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: membaca daftar cabang: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterbengkel.Branch
	for rows.Next() {
		var name, id sql.NullString
		if err := rows.Scan(&name, &id); err != nil {
			return nil, fmt.Errorf("masterbengkel/sqlstore: membaca baris cabang: %w", err)
		}
		branch := masterbengkel.Branch{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}
		// Cabang tanpa kode tidak dapat disimpan ke CABANG_ID, dan tanpa nama tidak dapat
		// dikenali pengguna. Keduanya dilewati di sini, bukan dibiarkan muncul di layar.
		if branch.ID == "" || branch.Name == "" {
			continue
		}
		result = append(result, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: menelusuri daftar cabang: %w", err)
	}
	return result, nil
}

// SearchCities mencari kota pada tabel CITY.
func (r *Repo) SearchCities(ctx context.Context, keyword string) ([]masterbengkel.City, error) {
	clean := strings.ToUpper(strings.TrimSpace(keyword))

	rows, err := r.db.QueryContext(ctx, getQuery("bengkel_city_search"), likePattern(clean), clean)
	if err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: mencari kota: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterbengkel.City
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("masterbengkel/sqlstore: membaca baris kota: %w", err)
		}
		city := masterbengkel.City{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}
		if city.ID == "" || city.Name == "" {
			continue
		}
		result = append(result, city)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: menelusuri daftar kota: %w", err)
	}
	return result, nil
}

// ListBanks membaca seluruh bank pada GENERAL.LST_BANK_GROUP.
func (r *Repo) ListBanks(ctx context.Context) ([]masterbengkel.Bank, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("bengkel_bank_list"))
	if err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterbengkel.Bank
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("masterbengkel/sqlstore: membaca baris bank: %w", err)
		}
		bank := masterbengkel.Bank{
			Code: strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		}
		if bank.Code == "" || bank.Name == "" {
			continue
		}
		result = append(result, bank)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterbengkel/sqlstore: menelusuri daftar bank: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.BENGKEL_HE ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("bengkel_check_table"))
	if err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: POOLDATA.BENGKEL_HE tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CheckJSONMirror memastikan POOLDATA.M_BENGKEL_HE — tabel JSON milik Pega — dapat
// dibaca.
//
// Dipakai mode periksa saja. Lihat banner berkas .sql: perbandingan jumlah barisnya
// dengan BENGKEL_HE adalah cara termurah mengetahui apakah keduanya satu sumber.
func (r *Repo) CheckJSONMirror(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("bengkel_check_json_mirror"))
	if err != nil {
		return fmt.Errorf("masterbengkel/sqlstore: POOLDATA.M_BENGKEL_HE tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountByStatus menghitung baris pada satu status persetujuan.
func (r *Repo) CountByStatus(ctx context.Context, status masterbengkel.ApprovalStatus) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("bengkel_count_pending"), string(status)).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterbengkel/sqlstore: menghitung baris status %q: %w", status, err)
	}
	return total, nil
}

// CountAll menghitung seluruh baris BENGKEL_HE.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("bengkel_count_all")).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterbengkel/sqlstore: menghitung baris: %w", err)
	}
	return total, nil
}

// CountJSONMirror menghitung baris POOLDATA.M_BENGKEL_HE.
func (r *Repo) CountJSONMirror(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("bengkel_count_json_mirror")).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterbengkel/sqlstore: menghitung baris M_BENGKEL_HE: %w", err)
	}
	return total, nil
}

// likePattern menyusun pola LIKE dari sebuah kata kunci.
//
// Tanda persen, garis bawah, dan backslash pada kata kunci DILOLOSKAN lebih dulu. Tanpa
// itu, pengguna yang mengetik "%" menarik seluruh tabel dan yang mengetik "_" mencocoki
// karakter apa pun — bukan celah keamanan karena nilainya tetap terikat sebagai
// parameter, tetapi hasil yang tidak dapat dijelaskan kepada yang mengetiknya.
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

// scanRow membaca satu baris master bengkel.
//
// Seluruh kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang
// diketahui (R-08).
//
// Urutan kolomnya mengikuti berkas .sql, dan keempat puluh satu kolom dibaca pada urutan
// yang sama oleh bengkel_list, bengkel_list_search, bengkel_get, bengkel_find_by_name,
// dan bengkel_find_by_login. Itu yang membuat satu fungsi cukup untuk kelimanya — dan
// yang membuat uji urutan kolom pada query_test.go layak ada.
func scanRow(p scanner) (masterbengkel.Workshop, error) {
	var (
		id, name, address, phone, mobile                  sql.NullString
		email, workOrderEmail, branchID, branchName       sql.NullString
		cityID, cityName, partnerStatus, workshopStatus   sql.NullString
		statusReason, statusDate, login, bankID, bankName sql.NullString
		accountNumber, accountName, accountID             sql.NullString
		taxName, taxNumber, taxAddress, incomeTaxType     sql.NullString
		vat, serviceDiscount, partDiscount                sql.NullString
		materialPercent, priceListGap, sla                sql.NullString
		suppliedByASM, supplier, eClaim, autoAccept       sql.NullString
		payment, autoPayment, tekno, order_               sql.NullString
		documentID, status                                sql.NullString
	)
	if err := p.Scan(
		&id, &name, &address, &phone, &mobile,
		&email, &workOrderEmail, &branchID, &branchName, &cityID,
		&cityName, &partnerStatus, &workshopStatus, &statusReason, &statusDate,
		&login, &bankID, &bankName, &accountNumber, &accountName,
		&accountID, &taxName, &taxNumber, &taxAddress, &incomeTaxType,
		&vat, &serviceDiscount, &partDiscount, &materialPercent, &priceListGap,
		&sla, &suppliedByASM, &supplier, &eClaim, &autoAccept,
		&payment, &autoPayment, &tekno, &order_, &documentID,
		&status,
	); err != nil {
		return masterbengkel.Workshop{}, err
	}

	trim := strings.TrimSpace
	return masterbengkel.Workshop{
		ID:                  trim(id.String),
		Name:                trim(name.String),
		Address:             trim(address.String),
		Phone:               trim(phone.String),
		Mobile:              trim(mobile.String),
		Email:               trim(email.String),
		WorkOrderEmail:      trim(workOrderEmail.String),
		BranchID:            trim(branchID.String),
		BranchName:          trim(branchName.String),
		CityID:              trim(cityID.String),
		CityName:            trim(cityName.String),
		PartnerStatus:       trim(partnerStatus.String),
		WorkshopStatus:      trim(workshopStatus.String),
		StatusReason:        trim(statusReason.String),
		StatusDate:          trim(statusDate.String),
		Login:               trim(login.String),
		BankID:              trim(bankID.String),
		BankName:            trim(bankName.String),
		AccountNumber:       trim(accountNumber.String),
		AccountName:         trim(accountName.String),
		AccountID:           trim(accountID.String),
		TaxName:             trim(taxName.String),
		TaxNumber:           trim(taxNumber.String),
		TaxAddress:          trim(taxAddress.String),
		IncomeTaxType:       trim(incomeTaxType.String),
		ValueAddedTax:       trim(vat.String),
		ServiceDiscount:     trim(serviceDiscount.String),
		PartDiscount:        trim(partDiscount.String),
		MaterialPercent:     trim(materialPercent.String),
		PriceListGapPercent: trim(priceListGap.String),
		SLA:                 trim(sla.String),
		SuppliedByASM:       trim(suppliedByASM.String),
		Supplier:            trim(supplier.String),
		EClaimStatus:        trim(eClaim.String),
		AutoAcceptStatus:    trim(autoAccept.String),
		PaymentStatus:       trim(payment.String),
		AutoPaymentStatus:   trim(autoPayment.String),
		TeknoStatus:         trim(tekno.String),
		OrderStatus:         trim(order_.String),
		DocumentID:          trim(documentID.String),
		Status:              masterbengkel.ApprovalStatus(trim(status.String)),
	}, nil
}

// insertArguments menyusun keempat puluh satu nilai bengkel_insert pada urutan kolomnya.
//
// Urutannya WAJIB sama dengan daftar kolom pada kueri. Ia dipisahkan menjadi fungsi
// tersendiri supaya urutan itu dapat diuji terhadap berkas .sql, bukan hanya dipercaya.
func insertArguments(w masterbengkel.Workshop) []any {
	return []any{
		w.ID, w.Name, w.Address, w.Phone, w.Mobile,
		w.Email, w.WorkOrderEmail, w.BranchID, w.BranchName, w.CityID,
		w.CityName, w.PartnerStatus, w.WorkshopStatus, w.StatusReason, w.StatusDate,
		w.Login, w.BankID, w.BankName, w.AccountNumber, w.AccountName,
		w.AccountID, w.TaxName, w.TaxNumber, w.TaxAddress, w.IncomeTaxType,
		w.ValueAddedTax, w.ServiceDiscount, w.PartDiscount, w.MaterialPercent, w.PriceListGapPercent,
		w.SLA, w.SuppliedByASM, w.Supplier, w.EClaimStatus, w.AutoAcceptStatus,
		w.PaymentStatus, w.AutoPaymentStatus, w.TeknoStatus, w.OrderStatus, w.DocumentID,
		string(w.Status),
	}
}

// updateArguments menyusun nilai bengkel_update; ID_BENGKEL berada di posisi TERAKHIR
// karena ia penyaring WHERE, bukan kolom yang ditulis.
func updateArguments(w masterbengkel.Workshop) []any {
	return []any{
		w.Name, w.Address, w.Phone, w.Mobile, w.Email,
		w.WorkOrderEmail, w.BranchID, w.BranchName, w.CityID, w.CityName,
		w.PartnerStatus, w.WorkshopStatus, w.StatusReason, w.StatusDate, w.Login,
		w.BankID, w.BankName, w.AccountNumber, w.AccountName, w.AccountID,
		w.TaxName, w.TaxNumber, w.TaxAddress, w.IncomeTaxType, w.ValueAddedTax,
		w.ServiceDiscount, w.PartDiscount, w.MaterialPercent, w.PriceListGapPercent, w.SLA,
		w.SuppliedByASM, w.Supplier, w.EClaimStatus, w.AutoAcceptStatus, w.PaymentStatus,
		w.AutoPaymentStatus, w.TeknoStatus, w.OrderStatus, w.DocumentID, string(w.Status),
		strings.TrimSpace(w.ID),
	}
}

// locked menjalankan salah satu kueri FOR UPDATE dan menyatakan barisnya ada.
func locked(ctx context.Context, tx *sql.Tx, name, value string) (bool, error) {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return false, nil
	}

	rows, err := tx.QueryContext(ctx, getQuery(name), clean)
	if err != nil {
		return false, fmt.Errorf("masterbengkel/sqlstore: memeriksa %q: %w", value, err)
	}
	defer func() { _ = rows.Close() }()

	found := rows.Next()
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("masterbengkel/sqlstore: menelusuri pemeriksaan %q: %w", value, err)
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
		panic(fmt.Sprintf("masterbengkel/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterbengkel/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterbengkel/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterbengkel/sqlstore: nama kueri ganda: " + name)
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

var _ masterbengkel.Store = (*Repo)(nil)
