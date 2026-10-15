package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini memenuhi bagian tab **KPI Admin** dari seam reportkpi.Repo.
//
// # Yang berubah pada 2026-10-09
//
// Kueri tab ini dulu MENGHITUNG di basis data: pencacahan, tangga nilai, bobot, dan rasio
// pencapaian seluruhnya ada di dalam teks SQL. Semuanya bergantung pada satu fungsi lintas
// DB Link — `datamining.get_working_hours@asmd.sinarmas.co.id` — yang ternyata **tidak
// dapat dijangkau** dari basis data kita, sehingga tab ini gagal seluruhnya.
//
// Sekarang kuerinya MENGEMBALIKAN BARIS, dan tiga hal pindah ke Go:
//
//	selisih hari kerja   reportkpi.WorkingDaysBetween, memakai kalender libur HRD_LBR
//	pencacahan           di berkas ini
//	tangga nilai & bobot reportkpi.BuildAdminTotals* — aturan bisnis, tempatnya di domain
//
// Rinciannya di kepala `reportkpi_admin.sql`, termasuk selisih yang diketahui: umur kini
// bilangan BULAT, sedangkan Pega menampilkannya pecahan.

// slaThresholdDays adalah ambang pelanggaran SLA kelompok NON-MBU.
//
// `tat_regis > 1` pada kueri lama — lebih dari satu hari kerja.
const slaThresholdDays = 1

// paSLAThresholdDays adalah ambang kelompok PA.
//
// `tat_regis > 0` dan `tat_bayar > 0` pada kueri lama — BERBEDA dari NON-MBU, dan bukan
// salah ketik: untuk PA, satu hari kerja saja sudah terhitung melanggar.
const paSLAThresholdDays = 0

// AdminTotals mengambil angka mentah kartu skor.
func (r *Repo) AdminTotals(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	if q.Group == reportkpi.AdminGroupPA {
		return r.adminTotalsPA(ctx, q)
	}
	return r.adminTotalsNonMBU(ctx, q)
}

// rentangKerja adalah sepasang tanggal yang selisih hari kerjanya dihitung.
type rentangKerja struct {
	kunci string
	awal  time.Time
	akhir time.Time
	tanda string // kolom penanda tambahan; dipakai NON-MBU untuk `reinsurer`
}

// bacaRentang menjalankan satu kueri yang mengembalikan pasangan tanggal.
//
// Ketiga kolomnya SELALU dalam urutan yang sama — kunci, tanggal awal, tanggal akhir —
// sehingga satu pembaca melayani seluruh kueri baris tab ini. Kolom keempat opsional
// dibaca sebagai `tanda`.
func (r *Repo) bacaRentang(
	ctx context.Context,
	nama string,
	denganTanda bool,
	args ...any,
) ([]rentangKerja, error) {
	rows, err := r.db.QueryContext(ctx, query(nama), args...)
	if err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca %s: %w", nama, err)
	}
	defer func() { _ = rows.Close() }()

	var hasil []rentangKerja
	for rows.Next() {
		var (
			kunci       sql.NullString
			awal, akhir sql.NullTime
			tanda       sql.NullString
		)

		var scanErr error
		if denganTanda {
			scanErr = rows.Scan(&tanda, &awal, &akhir)
		} else {
			scanErr = rows.Scan(&kunci, &awal, &akhir)
		}
		if scanErr != nil {
			return nil, fmt.Errorf("reportkpi/sqlstore: memindai %s: %w", nama, scanErr)
		}

		hasil = append(hasil, rentangKerja{
			kunci: kunci.String,
			awal:  awal.Time,
			akhir: akhir.Time,
			tanda: tanda.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca %s: %w", nama, err)
	}
	return hasil, nil
}

// liburUntuk mengambil kalender libur yang mencakup SELURUH rentang yang akan dihitung.
//
// Jendelanya diturunkan dari datanya sendiri, bukan dari periode yang dipilih pengguna:
// tanggal akhir sebuah rentang kerap jatuh DI LUAR periode itu — klaim yang didaftarkan
// akhir bulan dapat ditransfer bulan berikutnya. Mengambil libur hanya sepanjang periode
// akan melewatkan hari libur di ekornya, dan umurnya menjadi lebih panjang dari semestinya.
func (r *Repo) liburUntuk(
	ctx context.Context,
	kumpulan ...[]rentangKerja,
) ([]time.Time, error) {
	var paling, teratas time.Time
	for _, kumpul := range kumpulan {
		for _, item := range kumpul {
			for _, saat := range []time.Time{item.awal, item.akhir} {
				if saat.IsZero() {
					continue
				}
				if paling.IsZero() || saat.Before(paling) {
					paling = saat
				}
				if teratas.IsZero() || saat.After(teratas) {
					teratas = saat
				}
			}
		}
	}
	if paling.IsZero() {
		return nil, nil
	}
	return r.Holidays(ctx, paling, teratas)
}

// adminTotalsNonMBU mengambil kartu skor NON-MBU.
func (r *Repo) adminTotalsNonMBU(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	baris, err := r.bacaRentang(ctx, "admin_rows_nonmbu", true, q.Range.From, q.Range.To)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	libur, err := r.liburUntuk(ctx, baris)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	var cacah reportkpi.AdminCountsNonMBU
	for _, item := range baris {
		lewat := reportkpi.WorkingDaysBetween(item.awal, item.akhir, libur) > slaThresholdDays

		// `reinsurer = '1'` berarti LEADER; selain itu member. Perbandingannya persis
		// seperti kueri lama — `<> '1'`, bukan "bukan kosong".
		if item.tanda == "1" {
			cacah.LeaderTotal++
			if lewat {
				cacah.LeaderOverSLA++
			}
			continue
		}
		cacah.MemberTotal++
		if lewat {
			cacah.MemberOverSLA++
		}
	}

	return reportkpi.BuildAdminTotalsNonMBU(cacah), nil
}

// adminTotalsPA mengambil kartu skor PA.
//
// Tiga kueri, bukan satu: tahap registrasi, tahap pembayaran pada periode yang dipilih,
// dan tahap pembayaran pada rentang 2023 yang tertanam. Yang ketiga sengaja tidak menerima
// periode — lihat `admin_rows_pa_payment_total`.
func (r *Repo) adminTotalsPA(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	registrasi, err := r.bacaRentang(ctx, "admin_rows_pa_register", false,
		q.Range.From, q.Range.To)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	pembayaran, err := r.bacaRentang(ctx, "admin_rows_pa_payment", false,
		q.Range.From, q.Range.To)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	pembayaran2023, err := r.bacaRentang(ctx, "admin_rows_pa_payment_total", false)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	libur, err := r.liburUntuk(ctx, registrasi, pembayaran, pembayaran2023)
	if err != nil {
		return reportkpi.AdminTotals{}, err
	}

	cacah := reportkpi.AdminCountsPA{ClaimTotal: len(registrasi)}
	for _, item := range registrasi {
		if reportkpi.WorkingDaysBetween(item.awal, item.akhir, libur) > paSLAThresholdDays {
			cacah.RegisterOverSLA++
		}
	}
	cacah.PaymentOverSLA = cacahKlaimMelewatiSLA(pembayaran, libur)
	cacah.PaymentTotal = cacahKlaimMelewatiSLA(pembayaran2023, libur)

	return reportkpi.BuildAdminTotalsPA(cacah), nil
}

// cacahKlaimMelewatiSLA menghitung JUMLAH KLAIM — bukan jumlah baris — yang melewati SLA.
//
// Satu klaim dapat punya beberapa baris akseptasi, dan kueri lama menghitungnya sekali
// lewat `COUNT(DISTINCT CLAIMNO)`. Menghitung barisnya akan melebihkan cacahnya pada
// klaim yang dibayar bertahap.
//
// Tanggal awalnya jatuh ke tanggal akseptasi bila tanggal terima LOD kosong — meniru
// `CASE WHEN RECEIVEDATELOD IS NOT NULL THEN … ELSE TGLAKSEPTASI END`. Akibatnya umur
// baris seperti itu NOL, bukan kosong, sehingga ia tidak pernah terhitung melanggar.
func cacahKlaimMelewatiSLA(baris []rentangKerja, libur []time.Time) int {
	unik := map[string]bool{}
	for _, item := range baris {
		awal := item.awal
		if awal.IsZero() {
			awal = item.akhir
		}
		if reportkpi.WorkingDaysBetween(awal, item.akhir, libur) > paSLAThresholdDays {
			unik[item.kunci] = true
		}
	}
	return len(unik)
}

// AdminDetail mengambil satu halaman grid rincian tab KPI Admin.
func (r *Repo) AdminDetail(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	if q.Group == reportkpi.AdminGroupPA {
		return r.adminDetailPA(ctx, q, page)
	}
	return r.adminDetailNonMBU(ctx, q, page)
}

// adminDetailNonMBU mengambil rincian NON-MBU.
func (r *Repo) adminDetailNonMBU(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	clean := page.Normalize()

	rows, err := r.db.QueryContext(ctx, query("admin_detail_nonmbu"),
		q.Range.From, q.Range.To, clean.Offset(), clean.Size,
	)
	if err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian NON-MBU: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type mentah struct {
		baris       reportkpi.AdminDetailRow
		awal, akhir time.Time
	}

	var kumpulan []mentah
	result := reportkpi.AdminDetailPage{Rows: []reportkpi.AdminDetailRow{}}

	for rows.Next() {
		var (
			claimNumber, policyNumber, businessName, teamFlag sql.NullString
			registerDate, transferDate                        sql.NullTime
			total                                             sql.NullInt64
		)
		if err := rows.Scan(
			&claimNumber, &policyNumber, &businessName,
			&registerDate, &transferDate, &teamFlag, &total,
		); err != nil {
			return reportkpi.AdminDetailPage{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai rincian NON-MBU: %w", err)
		}

		kumpulan = append(kumpulan, mentah{
			baris: reportkpi.AdminDetailRow{
				ClaimNumber:  claimNumber.String,
				PolicyNumber: policyNumber.String,
				BusinessName: businessName.String,
				RegisterDate: isoDate(registerDate),
				TransferDate: isoDate(transferDate),
				TeamFlag:     teamFlag.String,
			},
			awal:  registerDate.Time,
			akhir: transferDate.Time,
		})
		result.Total = int(total.Int64)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian NON-MBU: %w", err)
	}

	rentang := make([]rentangKerja, 0, len(kumpulan))
	for _, item := range kumpulan {
		rentang = append(rentang, rentangKerja{awal: item.awal, akhir: item.akhir})
	}
	libur, err := r.liburUntuk(ctx, rentang)
	if err != nil {
		return reportkpi.AdminDetailPage{}, err
	}

	for _, item := range kumpulan {
		baris := item.baris
		baris.RegisterAging = reportkpi.NewScore(
			float64(reportkpi.WorkingDaysBetween(item.awal, item.akhir, libur)))
		result.Rows = append(result.Rows, baris)
	}
	return result, nil
}

// adminDetailPA mengambil rincian PA — dua umur dan dua penanda SLA.
func (r *Repo) adminDetailPA(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	clean := page.Normalize()

	rows, err := r.db.QueryContext(ctx, query("admin_detail_pa"),
		q.Range.From, q.Range.To, clean.Offset(), clean.Size,
	)
	if err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian PA: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type mentah struct {
		baris                      reportkpi.AdminDetailRow
		terima, daftar             time.Time
		terimaLOD, tanggalAkseptasi time.Time
	}

	var kumpulan []mentah
	result := reportkpi.AdminDetailPage{Rows: []reportkpi.AdminDetailRow{}}

	for rows.Next() {
		var (
			claimNumber, policyNumber, adminName sql.NullString
			claimStatus                          sql.NullString
			policyStart, policyEnd               sql.NullTime
			receiveDate, registerDate            sql.NullTime
			lodReceiveDate, acceptanceDate       sql.NullTime
			total                                sql.NullInt64
		)
		if err := rows.Scan(
			&claimNumber, &policyNumber, &policyStart, &policyEnd, &adminName,
			&receiveDate, &registerDate, &lodReceiveDate, &acceptanceDate,
			&claimStatus, &total,
		); err != nil {
			return reportkpi.AdminDetailPage{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai rincian PA: %w", err)
		}

		// Tanggal terima LOD yang kosong jatuh ke tanggal akseptasi — termasuk pada
		// kolom yang DITAMPILKAN, persis seperti kueri lama yang menampilkan hasil
		// `CASE`-nya, bukan kolom aslinya.
		tampilLOD := lodReceiveDate
		if !tampilLOD.Valid {
			tampilLOD = acceptanceDate
		}

		kumpulan = append(kumpulan, mentah{
			// `policyStart` dan `policyEnd` DIPINDAI lalu dibuang — sama seperti kode
			// sebelumnya. Keduanya diambil kueri lama tetapi tidak punya medan di
			// AdminDetailRow dan tidak digambar grid mana pun.
			baris: reportkpi.AdminDetailRow{
				ClaimNumber:    claimNumber.String,
				PolicyNumber:   policyNumber.String,
				AdminName:      adminName.String,
				ReceiveDate:    isoDate(receiveDate),
				RegisterDate:   isoDate(registerDate),
				LODReceiveDate: isoDate(tampilLOD),
				AcceptanceDate: isoDate(acceptanceDate),
				ClaimStatus:    claimStatus.String,
			},
			terima:           receiveDate.Time,
			daftar:           registerDate.Time,
			terimaLOD:        tampilLOD.Time,
			tanggalAkseptasi: acceptanceDate.Time,
		})
		result.Total = int(total.Int64)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian PA: %w", err)
	}

	rentang := make([]rentangKerja, 0, len(kumpulan)*2)
	for _, item := range kumpulan {
		rentang = append(rentang,
			rentangKerja{awal: item.terima, akhir: item.daftar},
			rentangKerja{awal: item.terimaLOD, akhir: item.tanggalAkseptasi})
	}
	libur, err := r.liburUntuk(ctx, rentang)
	if err != nil {
		return reportkpi.AdminDetailPage{}, err
	}

	for _, item := range kumpulan {
		umurDaftar := reportkpi.WorkingDaysBetween(item.terima, item.daftar, libur)
		umurBayar := reportkpi.WorkingDaysBetween(item.terimaLOD, item.tanggalAkseptasi, libur)

		baris := item.baris
		baris.RegisterAging = reportkpi.NewScore(float64(umurDaftar))
		baris.PaymentAging = reportkpi.NewScore(float64(umurBayar))
		baris.RegisterSLA = penandaSLA(umurDaftar)
		baris.PaymentSLA = penandaSLA(umurBayar)
		result.Rows = append(result.Rows, baris)
	}
	return result, nil
}

// penandaSLA menerjemahkan umur menjadi teks seperti layar lama.
//
// Ambangnya `> 1`, BUKAN `> 0` seperti pencacahan kartu skor PA. Perbedaan itu ada di
// kueri lama dan dipertahankan: grid menandai pelanggaran pada hari kedua, sedangkan
// kartu skor menghitungnya sejak hari pertama.
func penandaSLA(hariKerja int) string {
	if hariKerja > slaThresholdDays {
		return "TIDAK SLA"
	}
	return "SLA"
}
