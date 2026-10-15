package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// Pembacaan ketujuh bagian isi popup rincian, dari TABEL RELASIONAL.
//
// Keputusan Work Owner 2026-10-12: migrasi tidak memakai dokumen JSON lagi. Alasan dan
// angkanya ada di `rincian_bagian.sql`.
//
// # Tujuh kueri, bukan satu
//
// Menggabungkan ketujuhnya menjadi satu kueri akan MENGGANDAKAN baris: satu klaim punya
// banyak objek, tiap objek banyak coverage, tiap coverage banyak adjustment. Hasil kali
// silangnya bukan hanya boros — ia membuat "berapa objek klaim ini" tidak lagi terbaca dari
// jumlah barisnya.
//
// Tujuh perjalanan ke basis data untuk SATU popup dapat diterima: popup dibuka satu per satu
// oleh satu orang, bukan digambar per baris daftar.

// bacaBagian menjalankan satu kueri bagian dan memindai barisnya lewat `pindai`.
//
// Dipisahkan karena ketujuh pembacaannya hanya berbeda pada kuerinya dan pemindainya; tanpa
// ini, penanganan galat yang sama ditulis tujuh kali — dan ketujuhnya kelak berbeda.
func bacaBagian[T any](
	ctx context.Context,
	db penanya,
	namaKueri string,
	kunci string,
	pindai func(*sql.Rows) (T, error),
) ([]T, error) {
	rows, err := db.QueryContext(ctx, query(namaKueri), kunci)
	if err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: membaca %s: %w", namaKueri, err)
	}
	defer func() { _ = rows.Close() }()

	var keluar []T
	for rows.Next() {
		baris, err := pindai(rows)
		if err != nil {
			return nil, fmt.Errorf("dashboardclaim/sqlstore: memindai %s: %w", namaKueri, err)
		}
		keluar = append(keluar, baris)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: menutup %s: %w", namaKueri, err)
	}
	return keluar, nil
}

// penanya adalah bagian *sql.DB yang dipakai di sini — disempitkan supaya pembacaan bagian
// tidak dapat tanpa sengaja memulai transaksi atau menulis.
type penanya interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

// bacaSeluruhBagian mengisi ketujuh bagian ClaimDetail.
//
// `kunci` adalah PZINSKEY; `nomor` adalah nomor klaim. KEDUANYA dibutuhkan, dan itu bukan
// kelonggaran: `GCNM_PROGRESS_CLAIM` menaut lewat NOMOR klaim, enam tabel lain lewat kunci
// Pega. Diukur 2026-10-12 — lewat kunci Pega, progres hanya cocok pada 98 dari 2.758 klaim.
func (r *Repo) bacaSeluruhBagian(
	ctx context.Context,
	kunci string,
	nomor string,
	rincian *dashboardclaim.ClaimDetail,
) error {
	var err error

	if rincian.Receivers, err = bacaBagian(ctx, r.db, "rincian_penerima", kunci,
		func(rows *sql.Rows) (dashboardclaim.Receiver, error) {
			var nama, alamat, bank, rekening sql.NullString
			if err := rows.Scan(&nama, &alamat, &bank, &rekening); err != nil {
				return dashboardclaim.Receiver{}, err
			}
			return dashboardclaim.Receiver{
				Name:          text(nama),
				Address:       text(alamat),
				BankName:      text(bank),
				AccountNumber: text(rekening),
			}, nil
		}); err != nil {
		return err
	}

	if rincian.Items, err = bacaBagian(ctx, r.db, "rincian_objek", kunci,
		func(rows *sql.Rows) (dashboardclaim.InsuredItem, error) {
			var nama, lokasi, status, nik, kerja, tipe, rangka sql.NullString
			var lahir sql.NullTime
			if err := rows.Scan(&nama, &lokasi, &status, &nik, &lahir, &kerja, &tipe, &rangka); err != nil {
				return dashboardclaim.InsuredItem{}, err
			}
			return dashboardclaim.InsuredItem{
				Name:              text(nama),
				Location:          text(lokasi),
				ParticipantStatus: text(status),
				IDCard:            text(nik),
				DateOfBirth:       waktuOpsional(lahir),
				Job:               text(kerja),
				VehicleType:       text(tipe),
				ChassisNumber:     text(rangka),
			}, nil
		}); err != nil {
		return err
	}

	if rincian.Coverages, err = bacaBagian(ctx, r.db, "rincian_coverage", kunci,
		func(rows *sql.Rows) (dashboardclaim.Coverage, error) {
			var nama, kode, mata, sebab, objek sql.NullString
			var tsi sql.NullFloat64
			if err := rows.Scan(&nama, &kode, &tsi, &mata, &sebab, &objek); err != nil {
				return dashboardclaim.Coverage{}, err
			}
			return dashboardclaim.Coverage{
				Name:        text(nama),
				PlanCode:    text(kode),
				SumInsured:  pecahanOpsional(tsi),
				Currency:    text(mata),
				CauseOfLoss: text(sebab),
				ItemName:    text(objek),
			}, nil
		}); err != nil {
		return err
	}

	if rincian.Settlements, err = bacaBagian(ctx, r.db, "rincian_adjustment", kunci,
		func(rows *sql.Rows) (dashboardclaim.SettlementLine, error) {
			var no, statusAksep, jenis, penerima, mata, statusBayar sql.NullString
			var tglAksep, tglTransfer sql.NullTime
			var nilai, usul, total sql.NullFloat64
			if err := rows.Scan(&no, &tglAksep, &statusAksep, &jenis, &penerima, &mata,
				&nilai, &usul, &total, &tglTransfer, &statusBayar); err != nil {
				return dashboardclaim.SettlementLine{}, err
			}
			return dashboardclaim.SettlementLine{
				AcceptanceNumber: text(no),
				AcceptedAt:       waktuOpsional(tglAksep),
				AcceptanceStatus: text(statusAksep),
				PaymentType:      text(jenis),
				ReceiverName:     text(penerima),
				Currency:         text(mata),
				AcceptedValue:    pecahanOpsional(nilai),
				ProposedValue:    pecahanOpsional(usul),
				TotalClaim:       pecahanOpsional(total),
				TransferredAt:    waktuOpsional(tglTransfer),
				PaidStatus:       text(statusBayar),
			}, nil
		}); err != nil {
		return err
	}

	if rincian.Surveys, err = bacaBagian(ctx, r.db, "rincian_survey", kunci,
		func(rows *sql.Rows) (dashboardclaim.SurveyStep, error) {
			var surveyor, objek, lokasi, status, langkah sql.NullString
			var refno, catatan, lokasiSurvei, jenisRugi sql.NullString
			var tanggal sql.NullTime
			if err := rows.Scan(&tanggal, &surveyor, &objek, &lokasi, &status, &langkah,
				&refno, &catatan, &lokasiSurvei, &jenisRugi); err != nil {
				return dashboardclaim.SurveyStep{}, err
			}
			return dashboardclaim.SurveyStep{
				SurveyedAt: waktuOpsional(tanggal),
				Surveyor:   text(surveyor),
				ItemName:   text(objek),
				Location:   text(lokasi),
				Status:     text(status),
				StepIndex:  text(langkah),

				ReferenceNumber: text(refno),
				Note:            text(catatan),
				SurveyLocation:  text(lokasiSurvei),
				LossType:        text(jenisRugi),
			}, nil
		}); err != nil {
		return err
	}

	if rincian.Attachments, err = bacaBagian(ctx, r.db, "rincian_lampiran", kunci,
		func(rows *sql.Rows) (dashboardclaim.Attachment, error) {
			var kategori, nama, oleh sql.NullString
			var pada sql.NullTime
			if err := rows.Scan(&kategori, &nama, &pada, &oleh); err != nil {
				return dashboardclaim.Attachment{}, err
			}
			return dashboardclaim.Attachment{
				Category: text(kategori),
				Name:     text(nama),
				AddedAt:  waktuOpsional(pada),
				AddedBy:  text(oleh),
			}, nil
		}); err != nil {
		return err
	}

	// Satu-satunya bagian yang ditaut NOMOR klaim, bukan kunci Pega.
	if rincian.Progress, err = bacaBagian(ctx, r.db, "rincian_progress", nomor,
		func(rows *sql.Rows) (dashboardclaim.ProgressStep, error) {
			var kode, status, catatan, oleh, namaTahap, tindakLanjut sql.NullString
			var pada sql.NullTime
			if err := rows.Scan(&pada, &kode, &status, &catatan, &oleh, &namaTahap,
				&tindakLanjut); err != nil {
				return dashboardclaim.ProgressStep{}, err
			}
			return dashboardclaim.ProgressStep{
				RecordedAt: waktuOpsional(pada),
				StageCode:  text(kode),
				Status:     text(status),
				Note:       text(catatan),
				RecordedBy: text(oleh),
				StageName:  text(namaTahap),

				NextFollowUp: text(tindakLanjut),
			}, nil
		}); err != nil {
		return err
	}

	// Dua himpunan isian TUNGGAL, bukan daftar.
	//
	// Keduanya memakai `bacaBagian` yang sama lalu mengambil baris pertamanya, alih-alih
	// `QueryRow`: kueri keduanya sudah dibatasi `FETCH FIRST 1 ROWS ONLY`, dan `QueryRow`
	// akan mengembalikan `sql.ErrNoRows` untuk klaim yang memang belum punya permintaan
	// survei — keadaan yang sah, bukan galat.
	survei, err := bacaBagian(ctx, r.db, "rincian_permintaan_survey", kunci,
		func(rows *sql.Rows) (dashboardclaim.SurveyRequestFields, error) {
			var nama, lokasi, telp, cabang, surveyor, surel, objek sql.NullString
			var pada sql.NullTime
			if err := rows.Scan(&nama, &lokasi, &telp, &pada,
				&cabang, &surveyor, &surel, &objek); err != nil {
				return dashboardclaim.SurveyRequestFields{}, err
			}
			return dashboardclaim.SurveyRequestFields{
				RequestorName: text(nama),
				Location:      text(lokasi),
				Phone:         text(telp),
				RequestedAt:   waktuOpsional(pada),
				Branch:        text(cabang),
				SurveyorName:  text(surveyor),
				SurveyorEmail: text(surel),
				ItemName:      text(objek),
			}, nil
		})
	if err != nil {
		return err
	}
	if len(survei) > 0 {
		rincian.SurveyRequest = survei[0]
	}

	pucl, err := bacaBagian(ctx, r.db, "rincian_pucl", kunci,
		func(rows *sql.Rows) (dashboardclaim.PUCLFields, error) {
			var alasan sql.NullString
			var pada sql.NullTime
			if err := rows.Scan(&pada, &alasan); err != nil {
				return dashboardclaim.PUCLFields{}, err
			}
			return dashboardclaim.PUCLFields{
				DocumentCompleteAt: waktuOpsional(pada),
				DoctorRejectReason: text(alasan),
			}, nil
		})
	if err != nil {
		return err
	}
	if len(pucl) > 0 {
		rincian.PUCL = pucl[0]
	}

	return nil
}

// waktuOpsional membedakan "tidak ada tanggal" dari "tanggal nol".
//
// Mengembalikan penunjuk, bukan time.Time: nilai nol `time.Time` akan tampil sebagai tahun 1
// di layar, dan itu tidak dapat dibedakan dari tanggal yang memang belum diisi.
func waktuOpsional(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	nilai := t.Time
	return &nilai
}

// pecahanOpsional membedakan "tidak ada nilai" dari "nilai nol".
//
// Pembedaan ini menentukan pada nilai uang: akseptasi bernilai nol adalah keputusan, sedangkan
// akseptasi yang belum diisi adalah keadaan — dan keduanya menuntut tindakan berbeda.
func pecahanOpsional(f sql.NullFloat64) *float64 {
	if !f.Valid {
		return nil
	}
	nilai := f.Float64
	return &nilai
}

// CheckDetailSectionsReadable membuktikan ketujuh tabel bagian terbaca akun aplikasi.
//
// Dipanggil `claimpnc -periksa`. Satu pernyataan untuk tujuh tabel — lihat alasannya pada
// `rincian_bagian_check` di `rincian_bagian.sql`.
func (r *Repo) CheckDetailSectionsReadable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("rincian_bagian_check")); err != nil {
		return fmt.Errorf(
			"isi rincian klaim tidak terbaca akun aplikasi — periksa POOLDATA.T_CLAIM_RECEIVER, "+
				"T_CLAIM_OBJECTLIST, T_CLAIM_OBJECTCOVERAGE, T_CLAIM_ADJUSTMENT, T_SURVEYORLIST, "+
				"DATA_ATTACHFILE, GCNM_PROGRESS_CLAIM, GCNM_MST_PROGRESS_KLAIM: %w", err)
	}
	return nil
}
