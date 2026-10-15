package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// FindClaimDetail membaca rincian satu klaim: kepalanya, lalu ketujuh bagian isinya.
//
// Menggantikan `setDataViewKlaim_Act`. Sejak 2026-10-12 isinya dibaca dari TABEL
// RELASIONAL, bukan dari dokumen JSON — lihat `rincian_bagian.sql`.
func (r *Repo) FindClaimDetail(
	ctx context.Context,
	claimID string,
) (dashboardclaim.ClaimDetail, error) {
	key := strings.TrimSpace(claimID)
	if key == "" {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}

	var (
		nomor, status, statusKlaim, pic, admin sql.NullString
		panel, jenisBisnis, catatan            sql.NullString
		kunci                                  sql.NullString
		didaftarkan                            sql.NullTime

		// Isian tab "Registrasi" — tiga belas kolom `T_CLAIM_PNC`.
		terimaDokumen, kejadian, keluarRawat, lapor sql.NullTime
		telepon, surel, hubungan, hubunganLain      sql.NullString
		lokasi, kelurahan, tki                      sql.NullString
		keAnalyst, kePICTeknis                      sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query("klaim_rincian"), key, key).Scan(
		&nomor, &kunci, &status, &statusKlaim, &pic, &admin, &didaftarkan, &panel, &jenisBisnis, &catatan,
		&terimaDokumen, &kejadian, &keluarRawat, &lapor, &telepon, &hubungan, &hubunganLain,
		&lokasi, &kelurahan, &tki, &keAnalyst, &kePICTeknis, &surel)

	if errors.Is(err, sql.ErrNoRows) {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}
	if err != nil {
		if needsDBA(err) {
			return dashboardclaim.ClaimDetail{}, fmt.Errorf(
				"%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return dashboardclaim.ClaimDetail{}, fmt.Errorf(
			"dashboardclaim/sqlstore: membaca rincian klaim: %w", err)
	}

	rincian := dashboardclaim.ClaimDetail{
		ClaimID:       text(kunci),
		ClaimNumber:   text(nomor),
		ProcessStatus: text(status),
		ClaimStatus:   text(statusKlaim),
		TechnicalPIC:  text(pic),
		AdminPNC:      text(admin),
		RegisteredAt:  waktuISO(didaftarkan),
		GroupPanel:    text(panel),
		BusinessType:  text(jenisBisnis),
		InputterNote:  text(catatan),
		Registration: dashboardclaim.RegistrationFields{
			DocumentReceivedAt:  waktuOpsional(terimaDokumen),
			LossAt:              waktuOpsional(kejadian),
			HospitalDischarge:   waktuOpsional(keluarRawat),
			ReportedAt:          waktuOpsional(lapor),
			ReporterPhone:       text(telepon),
			ReporterEmail:       text(surel),
			ReporterRelation:    text(hubungan),
			ReporterRelationEtc: text(hubunganLain),
			LossLocation:        text(lokasi),
			Village:             text(kelurahan),
			InsuredIsTKI:        text(tki),
			NoteToAnalyst:       text(keAnalyst),
			NoteToTechnicalPIC:  text(kePICTeknis),
		},
	}

	// Isi popup dibaca SETELAH kepalanya ada, dan galatnya DIKEMBALIKAN.
	//
	// Menelannya menjadi bagian kosong akan memberitahu pengguna "klaim ini belum diisi" —
	// pernyataan yang salah dan tidak dapat dibantah dari layar. Itu alasan yang sama yang
	// dulu membuat dokumen rusak dibedakan dari dokumen kosong.
	// Kuncinya diambil dari HASIL pencarian, bukan dari yang dikirim pemanggil.
	//
	// Pemanggil boleh mengirim `PZINSKEY` atau NOMOR klaim; ketujuh bagian di bawah hanya
	// mengenal `PZINSKEY`. Memakai nilai yang dikirim akan membuat seluruh bagian kosong saat
	// pemanggilnya mengirim nomor — kepalanya terbaca, isinya tidak, dan layar menyatakan
	// "klaim ini belum punya isi rincian" yang tidak benar.
	if err := r.bacaSeluruhBagian(ctx, text(kunci), text(nomor), &rincian); err != nil {
		return dashboardclaim.ClaimDetail{}, err
	}
	return rincian, nil
}

// waktuISO memformat stempel waktu, atau kosong bila tidak ada.
//
// Lapisan HTTP yang memformatnya menurut zona tampilan; di sini bentuknya baku supaya adapter
// memori dan adapter SQL menghasilkan teks yang sama.
func waktuISO(t sql.NullTime) string {
	if !t.Valid || t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// CheckClaimDetailReadable membuktikan kedua tabel kepala rincian terbaca.
//
// Dipanggil `claimpnc -periksa`. Ia menyebut kolomnya satu per satu, termasuk ketiga belas
// kolom isian tab "Registrasi" — Oracle memvalidasi SELURUH rujukan kolom saat parse, bahkan
// di balik `WHERE 1 = 0`, sehingga probe yang menyebut kolomnya akan gagal di DEV bila ada satu
// nama yang salah. Probe yang hanya membuktikan tabelnya ada akan lulus terhadapnya.
//
// `POOLDATA.JSON_KLAIM` tidak lagi disebut: sejak 2026-10-12 kueri rincian tidak membacanya
// (`§233`). Pesan galat yang menyebut tabel yang tidak dibaca akan mengarahkan penelusuran ke
// tempat yang salah.
func (r *Repo) CheckClaimDetailReadable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("klaim_rincian_check")); err != nil {
		return fmt.Errorf(
			"rincian klaim tidak terbaca akun aplikasi — periksa POOLDATA.T_CLAIMLIST_ADMIN "+
				"dan POOLDATA.T_CLAIM_PNC: %w", err)
	}
	return nil
}
