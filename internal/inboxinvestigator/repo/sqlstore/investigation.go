package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// InvestigationRepo membaca dan menulis hasil investigasi satu portal.
//
// Ia TERPISAH dari Repo, mengikuti pemisahan seam-nya di domain: Repo hanya membaca
// antrean; yang ini menulis. Satu tipe untuk keduanya akan membuat jalur tulis tersedia
// bagi kode yang hanya bermaksud menampilkan daftar.
type InvestigationRepo struct {
	db *sql.DB
}

// NewInvestigationRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang
// dituju.
func NewInvestigationRepo(db *sql.DB) *InvestigationRepo {
	return &InvestigationRepo{db: db}
}

// oracleTableMissing adalah kode galat Oracle untuk tabel atau view yang tidak ada.
//
// Ia dikenali supaya `POOLDATA.TC_PNC_INVESTIGASI` yang belum dibuat DBA dapat dijawab
// sebagai "belum disiapkan administrator" dan bukan sebagai "sistem rusak". Petugas yang
// membaca pesan kedua akan mencoba berulang kali; yang membaca pesan pertama tahu kepada
// siapa harus bertanya.
const oracleTableMissing = "ORA-00942"

// asStoreMissing mengubah galat tabel-tidak-ada menjadi galat domain yang dapat dikenali.
//
// Pencocokan TEKS, bukan kode galat terketik, karena driver `go-ora` tidak membukanya
// sebagai tipe. Itu rapuh dan disadari — yang menjaganya adalah: salah mengenali hanya
// mengubah KALIMAT yang dibaca pengguna, tidak pernah mengubah data.
func asStoreMissing(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), oracleTableMissing) {
		return fmt.Errorf("%w: %v", inboxinvestigator.ErrInvestigationStoreMissing, err)
	}
	return err
}

// Load mengembalikan hasil investigasi yang sudah tersimpan untuk satu pekerjaan.
//
// Pekerjaan yang belum pernah diinvestigasi mengembalikan found=false tanpa galat; lihat
// catatan pada seam-nya.
func (r *InvestigationRepo) Load(
	ctx context.Context,
	claimRef string,
) (inboxinvestigator.Investigation, bool, error) {
	row := r.db.QueryRowContext(ctx, getQuery("investigasi_ambil"), strings.TrimSpace(claimRef))

	one, err := scanInvestigation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxinvestigator.Investigation{}, false, nil
	}
	if err != nil {
		return inboxinvestigator.Investigation{}, false, asStoreMissing(err)
	}
	return one, true, nil
}

// Save menyimpan formulir DAN memindahkan klaimnya, dalam SATU transaksi.
//
// # Kenapa satu transaksi, dan bukan dua panggilan berurutan
//
// Keduanya satu peristiwa. Klaim yang berpindah ke Analyst tanpa hasil investigasinya
// tersimpan adalah klaim yang tidak dapat ditindaklanjuti siapa pun — dan sebaliknya, hasil
// yang tersimpan tanpa klaimnya berpindah membuat pekerjaan itu tetap menggantung di
// antrean Investigator meski sudah selesai.
//
// Sistem lama tidak punya jaminan itu: `SetStatusInvestigator_Act` menyetel properti lalu
// memanggil `Obj-Save`, dan procedure yang dipanggilnya melakukan `COMMIT` sendiri — pola
// yang `D-68` cabut. Kepemilikan transaksi di sini ada di Go.
//
// # Urutannya: perbarui dulu, sisip bila tidak tersentuh
//
// Bukan MERGE. `MERGE ... FROM DUAL` adalah bentuk Oracle, dan `D-20` menuntut satu set SQL
// yang berjalan sama di PostgreSQL. Pola ini sudah dipakai modul Registrasi.
func (r *InvestigationRepo) Save(
	ctx context.Context,
	one inboxinvestigator.Investigation,
	move inboxinvestigator.Transition,
	by string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("inboxinvestigator/sqlstore: membuka transaksi: %w", err)
	}
	// Rollback yang dipanggil setelah Commit tidak berakibat apa pun; yang berbahaya
	// justru kebalikannya — jalur galat yang lupa menutup transaksinya.
	defer func() { _ = tx.Rollback() }()

	values := investigationValues(one, by, move.At)

	result, err := tx.ExecContext(ctx, getQuery("investigasi_perbarui"),
		append(values, one.ClaimRef, one.SurveyIndex, one.Index)...)
	if err != nil {
		return asStoreMissing(fmt.Errorf(
			"inboxinvestigator/sqlstore: memperbarui hasil investigasi: %w", err))
	}

	touched, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inboxinvestigator/sqlstore: membaca jumlah baris tersentuh: %w", err)
	}
	if touched == 0 {
		_, err = tx.ExecContext(ctx, getQuery("investigasi_sisip"),
			append(values, one.ClaimRef, one.SurveyIndex, one.Index)...)
		if err != nil {
			return asStoreMissing(fmt.Errorf(
				"inboxinvestigator/sqlstore: menyisipkan hasil investigasi: %w", err))
		}
	}

	if _, err := tx.ExecContext(ctx, getQuery("investigasi_pindahkan_klaim"),
		move.ClaimStatus, move.At, one.ClaimRef); err != nil {
		return fmt.Errorf("inboxinvestigator/sqlstore: memindahkan klaim ke Analyst: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inboxinvestigator/sqlstore: menutup transaksi: %w", err)
	}
	return nil
}

// CheckInvestigationTable membuktikan tabel hasil investigasi ada dan dapat dibaca.
func (r *InvestigationRepo) CheckInvestigationTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("investigasi_check_table"))
	if err != nil {
		return asStoreMissing(fmt.Errorf(
			"inboxinvestigator/sqlstore: memeriksa tabel hasil investigasi: %w", err))
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// investigationValues menyusun ketiga puluh nilai isian beserta jejaknya, pada urutan yang
// dipakai investigasi_perbarui MAUPUN investigasi_sisip.
//
// SATU fungsi untuk kedua kueri, dan itu bukan kerapian: kedua kueri menerima nilai yang
// sama pada urutan yang sama, dan dua penyusun terpisah dapat berbeda urutannya tanpa satu
// pun galat kompilasi. Yang terjadi adalah Nama PIC tersimpan di kolom Nama Penelepon.
func investigationValues(
	one inboxinvestigator.Investigation,
	by string,
	at time.Time,
) []any {
	return []any{
		nullTime(one.InvestigatedAt),
		nullText(one.Investigated),
		nullText(one.HospitalKindCode),
		nullText(one.HospitalName),
		nullText(one.OtherPlaceName),
		nullText(one.HospitalAddress),
		nullText(one.MedicalRecordNumber),
		nullText(one.PatientName),
		nullTime(one.DateOfBirth),
		nullText(one.BirthDateVerified),
		nullText(one.BirthDateNote),
		nullText(one.PatientRegistered),
		nullText(one.RegistrationNote),
		nullTime(one.TreatmentStart),
		nullTime(one.TreatmentEnd),
		nullText(one.BillTotal),
		nullText(one.BillSettled),
		nullText(one.PaidByPatient),
		nullText(one.PaidByCompany),
		nullText(one.PaidByOtherInsurer),
		nullText(one.NoPayment),
		nullText(one.OtherInsurer),
		nullText(one.ReceiptConfirmation),
		nullText(one.HospitalPIC),
		nullText(one.CallerName),
		nullText(one.StaffName),
		nullText(one.PhoneArea),
		nullText(one.Phone),
		nullText(one.PhoneExt),
		nullText(one.Remarks),
		nullText(by),
		at,
	}
}

// nullText mengirim teks kosong sebagai NULL, bukan sebagai teks kosong.
//
// Keduanya berbeda di basis data, dan perbedaannya terbaca pengguna: `JSON_VALUE` pada
// kueri ekspor mengembalikan NULL untuk jalur yang tidak ada, sehingga isian yang tidak
// pernah diisi dan isian yang dikosongkan menjadi sama — seperti seharusnya.
func nullText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// nullTime mengirim waktu kosong sebagai NULL.
func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

// scanInvestigation membaca satu baris menjadi Investigation.
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada investigasi_ambil.
//
// Seluruh kolom teks dibaca lewat sql.NullString: ketiga puluh isian formulir OPSIONAL —
// layar lama tidak memasang satu pun `pyRequired` pada section investigasinya.
func scanInvestigation(row rowScanner) (inboxinvestigator.Investigation, error) {
	var (
		claimRef                   sql.NullString
		surveyIndex, index         sql.NullInt64
		investigatedAt             sql.NullTime
		investigated, hospitalKind sql.NullString
		hospitalName, otherPlace   sql.NullString
		hospitalAddress            sql.NullString
		medicalRecord, patientName sql.NullString
		dateOfBirth                sql.NullTime
		birthVerified, birthNote   sql.NullString
		registered, registerNote   sql.NullString
		treatmentStart             sql.NullTime
		treatmentEnd               sql.NullTime
		billTotal, billSettled     sql.NullString
		paidPatient, paidCompany   sql.NullString
		paidInsurer, noPayment     sql.NullString
		otherInsurer, receipt      sql.NullString
		hospitalPIC, callerName    sql.NullString
		staffName, phoneArea       sql.NullString
		phone, phoneExt            sql.NullString
		remarks                    sql.NullString
	)

	if err := row.Scan(
		&claimRef, &surveyIndex, &index,
		&investigatedAt, &investigated, &hospitalKind,
		&hospitalName, &otherPlace, &hospitalAddress,
		&medicalRecord, &patientName, &dateOfBirth,
		&birthVerified, &birthNote, &registered, &registerNote,
		&treatmentStart, &treatmentEnd,
		&billTotal, &billSettled,
		&paidPatient, &paidCompany, &paidInsurer, &noPayment,
		&otherInsurer, &receipt,
		&hospitalPIC, &callerName, &staffName,
		&phoneArea, &phone, &phoneExt,
		&remarks,
	); err != nil {
		return inboxinvestigator.Investigation{}, err
	}

	return inboxinvestigator.Investigation{
		ClaimRef:            strings.TrimSpace(claimRef.String),
		SurveyIndex:         int(surveyIndex.Int64),
		Index:               int(index.Int64),
		InvestigatedAt:      nullableTime(investigatedAt),
		Investigated:        strings.TrimSpace(investigated.String),
		HospitalKindCode:    strings.TrimSpace(hospitalKind.String),
		HospitalName:        strings.TrimSpace(hospitalName.String),
		OtherPlaceName:      strings.TrimSpace(otherPlace.String),
		HospitalAddress:     strings.TrimSpace(hospitalAddress.String),
		MedicalRecordNumber: strings.TrimSpace(medicalRecord.String),
		PatientName:         strings.TrimSpace(patientName.String),
		DateOfBirth:         nullableTime(dateOfBirth),
		BirthDateVerified:   strings.TrimSpace(birthVerified.String),
		BirthDateNote:       strings.TrimSpace(birthNote.String),
		PatientRegistered:   strings.TrimSpace(registered.String),
		RegistrationNote:    strings.TrimSpace(registerNote.String),
		TreatmentStart:      nullableTime(treatmentStart),
		TreatmentEnd:        nullableTime(treatmentEnd),
		BillTotal:           strings.TrimSpace(billTotal.String),
		BillSettled:         strings.TrimSpace(billSettled.String),
		PaidByPatient:       strings.TrimSpace(paidPatient.String),
		PaidByCompany:       strings.TrimSpace(paidCompany.String),
		PaidByOtherInsurer:  strings.TrimSpace(paidInsurer.String),
		NoPayment:           strings.TrimSpace(noPayment.String),
		OtherInsurer:        strings.TrimSpace(otherInsurer.String),
		ReceiptConfirmation: strings.TrimSpace(receipt.String),
		HospitalPIC:         strings.TrimSpace(hospitalPIC.String),
		CallerName:          strings.TrimSpace(callerName.String),
		StaffName:           strings.TrimSpace(staffName.String),
		PhoneArea:           strings.TrimSpace(phoneArea.String),
		Phone:               strings.TrimSpace(phone.String),
		PhoneExt:            strings.TrimSpace(phoneExt.String),
		Remarks:             strings.TrimSpace(remarks.String),
	}, nil
}

var _ inboxinvestigator.InvestigationRepo = (*InvestigationRepo)(nil)
