package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// Repo membaca antrean Inbox Manager Admin dari SATU basis data entitas.
//
// Tidak ada operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama, dan
// selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengambil SELURUH baris satu tab, belum dipaginasi.
//
// Pemotongan halaman terjadi di aplikasi (inboxmanageradmin.Slice) atas keputusan Work Owner
// 2026-09-26 — lihat catatan di kepala inboxmanageradmin.sql.
//
// # Kenapa satu kueri melayani ketiga tab
//
// Karena di Pega pun begitu: ketiga grid dilayani Report Definition yang sama, dan yang
// membedakannya hanyalah nilai parameter `OrgUnit`. Memecahnya menjadi tiga kueri yang
// hanya berbeda pada satu literal justru menciptakan tiga tempat yang dapat menyimpang.
//
// Ini berbeda dari modul Inbox Manager Receive / PUCL, yang kedua kueri tabnya sengaja
// TIDAK disatukan — di sana yang berbeda adalah ARAH pembanding (`=` lawan `<>`), dan
// penyaring yang artinya berbalik menurut nilai bind adalah tempat paling mudah menampilkan
// lini bisnis yang salah. Di sini nilainya hanya dicocokkan, tidak dibalik.
func (r *Repo) List(
	ctx context.Context,
	q inboxmanageradmin.Query,
) ([]inboxmanageradmin.WorkItem, error) {
	rows, err := r.db.QueryContext(ctx, query("list_by_org_unit"), q.Tab.OrgUnit)
	if err != nil {
		if isMissingColumn(err) {
			return nil, fmt.Errorf(
				"menjalankan kueri antrean unit organisasi %s (%w): %v",
				q.Tab.OrgUnit, inboxmanageradmin.ErrSourceColumnMissing, err)
		}
		return nil, fmt.Errorf(
			"menjalankan kueri antrean unit organisasi %s: %w", q.Tab.OrgUnit, err)
	}
	defer rows.Close()

	items := []inboxmanageradmin.WorkItem{}
	for rows.Next() {
		item, err := scanWorkItem(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"membaca baris antrean unit organisasi %s: %w", q.Tab.OrgUnit, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"menelusuri hasil antrean unit organisasi %s: %w", q.Tab.OrgUnit, err)
	}

	return items, nil
}

// CheckTable memastikan tabel inti modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
//
// # Kenapa TIGA pemeriksaan, bukan satu
//
// Karena ketiganya gagal dengan sebab yang berbeda dan layak dibedakan. Memisahkannya
// membuat pesan gagalnya menyebut satu hal saja, dan itulah yang menghemat waktu orang yang
// membacanya.
//
//  1. TABELNYA — `POOLDATA.T_CLAIMLIST_ADMIN`. Sudah dibaca modul lain, sehingga
//     kegagalannya menunjuk koneksi atau hak akses.
//
//  2. SATU KOLOM yang diminta ditambahkan saat modul ini dipindahkan ke tabel itu
//     (Work Owner 2026-09-27), menunggu `migrations/0005` tahap 1 dijalankan DBA:
//
//     PXASSIGNEDORGUNIT  penyaring ketiga tab — tanpanya ketiganya menampilkan hal sama
//
//     Dua kolom lain sempat ikut dijaga di sini dan DIKELUARKAN pada koreksi Work Owner
//     hari yang sama: `PXCREATEOPNAME` ternyata SUDAH ADA di tabel, dan `STATUSCLAIM_1`
//     tidak lagi dipakai karena Status Klaim kini diturunkan dari `PYSTATUSWORK`.
//
//  3. `M_LOGIN_PNC.LINE_BUSINESS` — tabel yang BERBEDA, milik modul Login, dan kolom yang
//     menentukan tab mana yang boleh dibuka seorang petugas. Kolomnya sudah ada, tetapi
//     namanya pernah salah ditulis tanpa garis bawah pada migrasi yang lalu dicabut
//     (`migrations/0004_DICABUT.md`), sehingga ejaannya layak dijaga.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("membaca POOLDATA.T_CLAIMLIST_ADMIN: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query("check_column")).Scan(&ignored)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf(
			"membaca kolom PXASSIGNEDORGUNIT pada POOLDATA.T_CLAIMLIST_ADMIN — kolom "+
				"ini diminta ditambahkan saat modul ini dipindahkan ke tabel tersebut "+
				"dan menunggu migrations/0005 tahap 1 dijalankan DBA. Tanpanya ketiga "+
				"tab tidak dapat dibedakan sama sekali: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query("check_line_business")).Scan(&ignored)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf(
			"membaca kolom LINE_BUSINESS pada POOLDATA.M_LOGIN_PNC — kolom ini yang "+
				"menentukan tab mana yang boleh dibuka seorang petugas, menggantikan "+
				"jabatan HCQ yang keliru dipakai sampai 2026-09-27. Perhatikan GARIS "+
				"BAWAH pada namanya: migrasi yang menulisnya `LINEBUSINESS` sudah "+
				"dicabut justru karena salah nama: %w", err)
	}

	return nil
}

// LineBusinessFor membaca lini bisnis seorang petugas dari M_LOGIN_PNC.
//
// Ia yang menentukan tab mana yang boleh dibuka — padanan `OperatorID.pyPosition` sistem
// lama. Sampai 2026-09-27 nilainya keliru diambil dari jabatan kepegawaian HCQ; lihat
// catatan di inboxmanageradmin.Caller.LineBusiness.
//
// # Petugas tanpa baris BUKAN galat
//
// Ia mengembalikan teks kosong tanpa galat, sama seperti `inboxoutstanding` memperlakukannya.
// Alasannya bukan kelonggaran melainkan kesetaraan: di Pega `pyPosition` yang tidak cocok
// satu pun sekadar tidak membuka kontainer mana pun — ia tidak menggagalkan layarnya.
//
// Kolomnya pun baru terisi pada sebagian petugas (`migrations/0004_DICABUT.md`), sehingga
// menjadikan ketiadaannya galat akan menggagalkan layar bagi hampir semua orang — dan pesan
// galat itu tidak akan menjelaskan apa pun yang dapat mereka perbaiki.
func (r *Repo) LineBusinessFor(ctx context.Context, loginID string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return "", nil
	}

	var line sql.NullString
	err := r.db.QueryRowContext(ctx, query("line_business_for"), id).Scan(&line)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("membaca lini bisnis petugas %s: %w", id, err)
	}

	return strings.TrimSpace(line.String), nil
}

// oracleMissingColumn adalah kode galat Oracle untuk pengenal yang tidak sah — yang pada
// kueri modul ini selalu berarti kolomnya belum ada.
const oracleMissingColumn = "ORA-00904"

// isMissingColumn mengenali kegagalan yang disebabkan kolom yang belum ditambahkan.
//
// # Kenapa dicocokkan sebagai TEKS
//
// Karena kodenya ada di dalam pesan galat driver, dan menariknya sebagai nilai terstruktur
// menuntut mengimpor tipe galat khas `godror` ke sini. Itu akan mengikat lapisan ini pada
// satu driver demi satu perbandingan — pertukaran yang tidak sepadan.
//
// Kekeliruan pengenalan tidak berbahaya: yang berubah hanyalah pesan yang dibaca pengguna,
// bukan apakah permintaannya gagal. Galat yang tidak dikenali tetap dijawab sebagai galat.
func isMissingColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), oracleMissingColumn)
}

// scanner adalah bentuk minimal yang dibutuhkan scanWorkItem, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris menjadi WorkItem.
//
// Urutannya WAJIB sama dengan resultColumns dan dengan urutan kolom di
// inboxmanageradmin.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom dipindai lewat tipe yang mengizinkan NULL. Itu bukan kehati-hatian
// berlebihan: `STATUSCLAIM_1` yang tidak punya padanan di V_STS_CLAIM menghasilkan
// CLAIM_STATUS NULL, dan kolom snapshot polis memang kosong pada klaim yang polisnya belum
// terbaca.
//
// ClaimElapsed TIDAK dipindai — ia dihitung di Go dari REGISTERED_AT lewat seam Clock,
// supaya hasilnya deterministik saat diuji dan kuerinya tetap portabel.
func scanWorkItem(row scanner) (inboxmanageradmin.WorkItem, error) {
	var (
		reference, caseID, policyNumber, insuredName sql.NullString
		businessName, businessSource                 sql.NullString
		registeredAt                                 sql.NullTime
		adminName, claimStatus                       sql.NullString
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &insuredName,
		&businessName, &businessSource, &registeredAt, &adminName,
		&claimStatus,
	)
	if err != nil {
		return inboxmanageradmin.WorkItem{}, err
	}

	return inboxmanageradmin.WorkItem{
		Reference:      reference.String,
		CaseID:         caseID.String,
		PolicyNumber:   policyNumber.String,
		InsuredName:    insuredName.String,
		BusinessName:   businessName.String,
		BusinessSource: businessSource.String,
		RegisteredAt:   timeOrNil(registeredAt),
		AdminName:      adminName.String,

		// Diturunkan, bukan disalin: kolomnya berisi PYSTATUSWORK mentah
		// ('New', 'Resolved-Completed', …), sedangkan yang dibaca pengguna adalah
		// label. Lihat inboxmanageradmin.DisplayStatusFor, termasuk kenapa nilai
		// asing TIDAK dipaksa menjadi "On Progress".
		ClaimStatus: inboxmanageradmin.DisplayStatusFor(claimStatus.String),
	}, nil
}

// timeOrNil mengubah kolom tanggal yang boleh NULL menjadi pointer.
//
// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari "belum
// diisi" saat ditampilkan, dan kolom Lama Waktu Klaim yang dihitung darinya akan berbunyi
// dalam ribuan tahun.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	at := value.Time
	return &at
}
