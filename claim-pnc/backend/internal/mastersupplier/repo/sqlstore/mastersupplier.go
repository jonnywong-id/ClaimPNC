// Package sqlstore memenuhi seam mastersupplier.Store dengan SQL.
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

	"claim-pnc/internal/mastersupplier"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca dan menulis M_SUPPLIER serta POOLDATA.PROTEKSI_KLAIMMBU, dan MEMBACA enam
// objek acuan.
//
// Satu struct memenuhi keempat seam — mastersupplier.Repo, LookupRepo, ApprovalRepo, dan
// IDSource — karena keempatnya selalu berasal dari koneksi entitas yang sama. Yang
// terpisah adalah interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel
// — lihat berkas .sql untuk alasannya.
func (r *Repo) List(ctx context.Context, filter mastersupplier.Filter) ([]mastersupplier.Supplier, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		rows, err = r.db.QueryContext(ctx, getQuery("supplier_list_search"), likePattern(keyword))
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("supplier_list"))
	}
	if err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersupplier.Supplier
	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mastersupplier/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan ID-nya.
func (r *Repo) Get(ctx context.Context, id string) (mastersupplier.Supplier, error) {
	row := r.db.QueryRowContext(ctx, getQuery("supplier_get"), strings.TrimSpace(id))

	one, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
	}
	if err != nil {
		return mastersupplier.Supplier{}, fmt.Errorf("mastersupplier/sqlstore: membaca %q: %w", id, err)
	}
	return one, nil
}

// FindByName mencari baris menurut kunci NAMA-nya.
func (r *Repo) FindByName(ctx context.Context, name string) (mastersupplier.Supplier, error) {
	clean := strings.ToUpper(strings.TrimSpace(name))
	if clean == "" {
		return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("supplier_find_by_name"), clean)

	one, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
	}
	if err != nil {
		return mastersupplier.Supplier{}, fmt.Errorf("mastersupplier/sqlstore: mencari nama %q: %w", name, err)
	}
	return one, nil
}

// Insert menyisipkan baris baru, menolak nama yang sudah dipakai.
//
// Kedua langkah berada di dalam SATU transaksi: penguncian yang dilepas sebelum
// penyisipan tidak menghalangi apa pun.
func (r *Repo) Insert(ctx context.Context, s mastersupplier.Supplier) error {
	document, err := buildDocument(s)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	taken, err := locked(ctx, tx, "supplier_lock_by_name", s.Name)
	if err != nil {
		return err
	}
	if taken {
		return fmt.Errorf("%w: %q", mastersupplier.ErrNameTaken, s.Name)
	}

	if _, err := tx.ExecContext(ctx, getQuery("supplier_insert"),
		strings.TrimSpace(s.ID), document); err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: menyisipkan %q: %w", s.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// # Kenapa membaca lebih dulu, padahal yang ditulis seluruh dokumen
//
// Karena yang ditulis BUKAN seluruh dokumen melainkan kunci yang dikenal saja. Dokumen
// tersimpan dibaca, kunci yang dikenal ditulis ulang di atasnya, dan kunci lain dibiarkan
// — lihat mergeDocument.
//
// Keduanya berada di dalam satu transaksi dan pembacaannya memakai FOR UPDATE. Tanpa itu,
// dua penyimpanan atas baris yang sama akan sama-sama membaca dokumen lama dan yang
// terakhir menimpa perubahan yang pertama, tanpa satu pun tanda.
//
// Pemeriksaan nama ganda TIDAK dilakukan di sini, dan itu bukan kelalaian: nama supplier
// tidak dapat diubah setelah barisnya tersimpan (mastersupplier.ErrNameLocked), sehingga
// ia tidak pernah dapat berubah menjadi nama yang sudah dipakai. Yang memeriksa larangan
// itu adalah lapisan aplikasi, yang punya baris tersimpannya sebagai pembanding.
func (r *Repo) Update(ctx context.Context, s mastersupplier.Supplier) error {
	key := strings.TrimSpace(s.ID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: memulai transaksi simpan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var stored sql.NullString
	switch err := tx.QueryRowContext(ctx, getQuery("supplier_read_document"), key).Scan(&stored); {
	case errors.Is(err, sql.ErrNoRows):
		return mastersupplier.ErrNotFound
	case err != nil:
		return fmt.Errorf("mastersupplier/sqlstore: membaca dokumen %q: %w", s.ID, err)
	}

	document, err := mergeDocument(stored.String, s)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, getQuery("supplier_update"), document, key)
	if err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: memperbarui %q: %w", s.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh
	// satu baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan
	// mengatakan "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return mastersupplier.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: menutup transaksi simpan: %w", err)
	}
	return nil
}

// RequestApproval menyisipkan satu baris permintaan persetujuan.
//
// Dokumen supplier disalin ke kolom JSONDATA-nya dengan perangkai yang SAMA dengan yang
// dipakai tabel master, sehingga keduanya tidak pernah dapat berbeda bentuk.
//
// # Ia BELUM berada di transaksi yang sama dengan penyimpanan master, dan itu disadari
//
// Keduanya dua pernyataan terpisah pada koneksi yang sama, sehingga kegagalan di antaranya
// meninggalkan supplier yang tersimpan tanpa baris permintaannya. Sistem lama pun begitu —
// bahkan lebih longgar, karena `PEGA_M_SUPPLIER.prc:25,37` melakukan COMMIT sendiri
// sebelum `InsertProteksiKlaimMBU_SQL` dijalankan sama sekali.
//
// Menyatukannya menuntut transaksi yang dipegang lapisan aplikasi, dan itu perubahan
// bentuk seam yang menyentuh seluruh modul master — lingkup tersendiri, bukan keputusan
// modul ini. Yang dikerjakan sekarang: kegagalannya dibedakan dari kegagalan penyimpanan
// di dalam pesan galatnya, sehingga petugas tahu supplier-nya ada tetapi antreannya tidak.
func (r *Repo) RequestApproval(ctx context.Context, request mastersupplier.ApprovalRequest) error {
	document, err := buildDocument(request.Snapshot)
	if err != nil {
		return err
	}

	if _, err := r.db.ExecContext(ctx, getQuery("approval_insert"),
		strings.TrimSpace(request.ID),
		request.RequestedAt.UTC(),
		request.Decision,
		request.RequestedBy,
		request.Note,
		request.Reason,
		request.Position,
		document,
		strings.TrimSpace(request.SupplierID),
	); err != nil {
		return fmt.Errorf("mastersupplier/sqlstore: menyisipkan permintaan persetujuan %q: %w",
			request.ID, err)
	}
	return nil
}

// NextID menerbitkan ID supplier berikutnya.
//
// Bentuknya meniru `Database/PEGA_M_SUPPLIER.prc:12,21` persis: kode situs ditambah nomor
// urut SEBELAS digit bertambal nol — bukan sepuluh seperti Master Bengkel.
//
// Kedua kueri dijalankan di dalam SATU transaksi. Bukan demi keatomikan — sequence tidak
// dapat dibatalkan — melainkan supaya keduanya pasti dilayani koneksi yang sama; kode situs
// dan sequence yang berasal dari dua koneksi berbeda pada pool yang sama tetap benar,
// tetapi jaminannya tidak berasal dari mana pun selain kebetulan.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("mastersupplier/sqlstore: memulai transaksi penomoran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var site sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery("supplier_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Persis keadaan yang ditangkap `PEGA_M_SUPPLIER.prc:13-18`, yang menjawabnya
			// dengan pesan galat lalu berhenti. Di sini ia juga berhenti — ID tanpa kode
			// situs akan bertabrakan dengan ID entitas lain.
			return "", errors.New("mastersupplier/sqlstore: POOLDATA.M_SITE_DATABASE tidak punya baris CURRENT_SITE='1'")
		}
		return "", fmt.Errorf("mastersupplier/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("supplier_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("mastersupplier/sqlstore: mengambil nomor urut: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("mastersupplier/sqlstore: menutup transaksi penomoran: %w", err)
	}
	return mastersupplier.ComposeID(
		strings.TrimSpace(site.String), sequence, mastersupplier.SequenceWidth), nil
}

// ListBranches membaca daftar cabang dari M_BRANCH.
func (r *Repo) ListBranches(ctx context.Context) ([]mastersupplier.Branch, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("supplier_branch_list"))
	if err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: membaca daftar cabang: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersupplier.Branch
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastersupplier/sqlstore: membaca baris cabang: %w", err)
		}
		result = append(result, mastersupplier.Branch{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar cabang: %w", err)
	}
	return result, nil
}

// SearchCities mencari kota menurut namanya, atau menurut ID persisnya.
func (r *Repo) SearchCities(ctx context.Context, keyword string) ([]mastersupplier.City, error) {
	clean := strings.TrimSpace(keyword)
	if len(clean) < mastersupplier.MinLookupKeyword {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, getQuery("supplier_city_search"),
		likePattern(clean), strings.ToUpper(clean))
	if err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: mencari kota: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersupplier.City
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastersupplier/sqlstore: membaca baris kota: %w", err)
		}
		result = append(result, mastersupplier.City{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar kota: %w", err)
	}
	return result, nil
}

// ListCountries membaca daftar negara.
func (r *Repo) ListCountries(ctx context.Context) ([]mastersupplier.Country, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("supplier_country_list"))
	if err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: membaca daftar negara: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersupplier.Country
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastersupplier/sqlstore: membaca baris negara: %w", err)
		}
		result = append(result, mastersupplier.Country{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar negara: %w", err)
	}
	return result, nil
}

// ListBanks membaca daftar bank.
func (r *Repo) ListBanks(ctx context.Context) ([]mastersupplier.Bank, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("supplier_bank_list"))
	if err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersupplier.Bank
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("mastersupplier/sqlstore: membaca baris bank: %w", err)
		}
		result = append(result, mastersupplier.Bank{
			Code: strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar bank: %w", err)
	}
	return result, nil
}

// ListCodes membaca sandi yang benar-benar dipakai baris yang ada.
//
// Hasilnya BELUM digabung dengan sandi yang artinya terbukti dari export — itu urusan
// lapisan aplikasi, yang memutuskan bagaimana daftarnya dilengkapi. Adapter hanya menjawab
// "apa yang ada di data".
//
// Labelnya diisi sandinya sendiri, bukan tebakan artinya: sandi yang hanya ditemukan di
// data memang tidak diketahui artinya, dan menampilkan tebakan yang tampak meyakinkan
// lebih buruk daripada menampilkan sandinya apa adanya.
func (r *Repo) ListCodes(ctx context.Context) (mastersupplier.CodeSet, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("supplier_code_distinct"))
	if err != nil {
		return mastersupplier.CodeSet{}, fmt.Errorf("mastersupplier/sqlstore: membaca daftar sandi: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result mastersupplier.CodeSet
	for rows.Next() {
		var group, value sql.NullString
		if err := rows.Scan(&group, &value); err != nil {
			return mastersupplier.CodeSet{}, fmt.Errorf("mastersupplier/sqlstore: membaca baris sandi: %w", err)
		}

		code := mastersupplier.CodeOption{
			Value: strings.TrimSpace(value.String),
			Label: strings.TrimSpace(value.String),
		}
		if code.Value == "" {
			continue
		}

		switch strings.TrimSpace(group.String) {
		case keyPartnerStatus:
			result.PartnerStatus = append(result.PartnerStatus, code)
		case keySupplyType:
			result.SupplyType = append(result.SupplyType, code)
		case keySupplierType:
			result.SupplierType = append(result.SupplierType, code)
		case keyActiveRequested:
			result.Active = append(result.Active, code)
		case keyAutoPayment:
			result.AutoPayment = append(result.AutoPayment, code)
		}
	}
	if err := rows.Err(); err != nil {
		return mastersupplier.CodeSet{}, fmt.Errorf("mastersupplier/sqlstore: menelusuri daftar sandi: %w", err)
	}
	return result, nil
}

// locked menjalankan sebuah kueri penguncian dan menyatakan barisnya sudah ada.
func locked(ctx context.Context, tx *sql.Tx, name, value string) (bool, error) {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return false, nil
	}

	rows, err := tx.QueryContext(ctx, getQuery(name), clean)
	if err != nil {
		return false, fmt.Errorf("mastersupplier/sqlstore: mengunci %q: %w", value, err)
	}
	defer func() { _ = rows.Close() }()

	taken := rows.Next()
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("mastersupplier/sqlstore: menelusuri penguncian %q: %w", value, err)
	}
	return taken, nil
}

// likePattern menyiapkan kata kunci menjadi pola LIKE yang aman.
//
// Karakter khas LIKE diloloskan lebih dulu supaya tanda persen yang diketik pengguna
// dicari apa adanya, bukan berlaku sebagai wildcard. Pelolosnya `\`, dan ia disebut
// eksplisit di setiap kueri lewat `ESCAPE '\'`.
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

// scanRow membaca satu baris master supplier.
//
// Seluruh nilai dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kunci yang tidak ada
// di dalam dokumen membuat JSON_VALUE menjawab NULL, dan kolom ID bertipe CHAR berlebar
// tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun (R-08).
//
// Urutan kolomnya mengikuti berkas .sql, dan kedua puluh sembilan kolom dibaca pada urutan
// yang sama oleh supplier_list, supplier_list_search, supplier_get, dan
// supplier_find_by_name. Itu yang membuat satu fungsi cukup untuk keempatnya — dan yang
// membuat uji urutan kolom pada query_test.go layak ada.
//
// # JENIS_STATUS diturunkan dari SUPPLIER_HE, bukan dibaca apa adanya
//
// Itu bukan tambahan melainkan tiruan: `Activity/GetDataSupplier_pre` step 6.3 melakukan
// hal yang sama persis setiap kali sebuah supplier dimuat ke form.
//
// Alasannya nyata, bukan kerapian. Cacat pada `CreateNewMasterSupplier_post` step 7 — yang
// hanya punya cabang "bila" tanpa cabang "selain itu" — meninggalkan baris yang kedua
// nilainya tidak sejalan, dan yang menentukan perilaku sistem hilir adalah SUPPLIER_HE.
// Membacanya dari sanalah yang membuat layar menampilkan keadaan yang sebenarnya berlaku.
func scanRow(p scanner) (mastersupplier.Supplier, error) {
	var (
		id, oldID, name, address, city              sql.NullString
		branchName, postalCode, country, phone, fax sql.NullString
		email, taxNumber, contactPerson             sql.NullString
		partnerStatus, supplyType, heavyEquipment   sql.NullString
		termOfPayment, termOfDelivery, note         sql.NullString
		bank, accountNumber, accountName            sql.NullString
		bankBranch, supplierType                    sql.NullString
		activeRequested, active, autoPayment        sql.NullString
		updatedBy, updatedAt                        sql.NullString
	)
	if err := p.Scan(
		&id, &oldID, &name, &address, &city,
		&branchName, &postalCode, &country, &phone, &fax,
		&email, &taxNumber, &contactPerson, &partnerStatus, &supplyType,
		&heavyEquipment, &termOfPayment, &termOfDelivery, &note, &bank,
		&accountNumber, &accountName, &bankBranch, &supplierType, &activeRequested,
		&active, &autoPayment, &updatedBy, &updatedAt,
	); err != nil {
		return mastersupplier.Supplier{}, err
	}

	trim := strings.TrimSpace
	return mastersupplier.Supplier{
		ID:    trim(id.String),
		OldID: trim(oldID.String),
		Name:  trim(name.String),

		Address:    trim(address.String),
		City:       trim(city.String),
		BranchName: trim(branchName.String),
		PostalCode: trim(postalCode.String),
		Country:    trim(country.String),

		Phone: trim(phone.String),
		Fax:   trim(fax.String),
		Email: trim(email.String),

		TaxNumber:     trim(taxNumber.String),
		ContactPerson: trim(contactPerson.String),

		PartnerStatus:  trim(partnerStatus.String),
		SupplyType:     mastersupplier.DeriveSupplyType(trim(heavyEquipment.String)),
		HeavyEquipment: trim(heavyEquipment.String),

		TermOfPayment:  trim(termOfPayment.String),
		TermOfDelivery: trim(termOfDelivery.String),
		Note:           trim(note.String),

		Bank:          trim(bank.String),
		AccountNumber: trim(accountNumber.String),
		AccountName:   trim(accountName.String),
		BankBranch:    trim(bankBranch.String),

		SupplierType:    trim(supplierType.String),
		ActiveRequested: trim(activeRequested.String),
		Active:          trim(active.String),
		AutoPayment:     trim(autoPayment.String),

		UpdatedBy: trim(updatedBy.String),
		UpdatedAt: trim(updatedAt.String),
	}, nil
}

func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("mastersupplier/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("mastersupplier/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("mastersupplier/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("mastersupplier/sqlstore: nama kueri ganda: " + name)
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
		if strings.HasPrefix(strings.TrimSpace(rows), marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rows), marker))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}
