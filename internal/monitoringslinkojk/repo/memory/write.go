package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/monitoringslinkojk"
)

// Aksi tulis pada penyimpanan memori.
//
// Ia MENYIMPAN, tidak sekadar menerima dan membuang. Penyimpanan tiruan yang menelan
// tulisan membuat uji "sesudah menyusun, barisnya ada" tidak dapat ditulis sama sekali —
// dan justru uji itulah yang membuktikan `operasidata` berubah dari `'C'` menjadi `'U'`
// pada penyusunan kedua.

// CountReport menghitung baris laporan yang sudah tersimpan untuk satu klaim.
func (r *Repo) CountReport(ctx context.Context, claimID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted := strings.TrimSpace(claimID)
	count := 0
	for _, entry := range r.reports {
		if entry.ClaimID == wanted {
			count++
		}
	}
	return count, nil
}

// InsertReport menyusun satu baris laporan.
//
// MENAMBAH, tidak menimpa — sama seperti penyimpanan SQL, yang menandai baris berulang
// lewat `operasidata = 'U'` alih-alih memperbaruinya.
func (r *Repo) InsertReport(ctx context.Context, entry monitoringslinkojk.ReportEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = append(r.reports, entry.Clean())
	return nil
}

// Reports mengembalikan seluruh baris laporan yang sudah tersusun.
//
// Hanya dipakai uji. Ia salinan, bukan senarai aslinya — pemanggil yang mengubahnya tidak
// boleh menyentuh isi penyimpanan, dengan alasan yang sama seperti cloneRow.
func (r *Repo) Reports() []monitoringslinkojk.ReportEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]monitoringslinkojk.ReportEntry(nil), r.reports...)
}

// StreamSource membaca data klaim sumber contoh.
//
// Isinya diturunkan dari baris contoh segmen D01, karena itulah bentuk yang sama dengan
// yang dihasilkan kueri sumber di produksi. Empat kolom yang di produksi TIDAK tersedia
// — jumlah hari tunggakan, No KTP, NPWP perusahaan, dan client ID — dibiarkan kosong di
// sini pula, supaya uji tidak lulus atas data yang lebih lengkap daripada kenyataan.
func (r *Repo) StreamSource(
	ctx context.Context,
	filter monitoringslinkojk.Filter,
	emit func(monitoringslinkojk.ReportEntry) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, record := range r.matching(monitoringslinkojk.SegmentD01, filter) {
		if err := emit(entryFromRow(record.Row)); err != nil {
			return err
		}
	}
	return nil
}

// entryFromRow menyusun satu baris laporan dari baris grid segmen D01.
func entryFromRow(row monitoringslinkojk.Row) monitoringslinkojk.ReportEntry {
	return monitoringslinkojk.ReportEntry{
		ClaimID:               row.Get("no_klaim"),
		ContractNo:            row.Get("contract_no"),
		FacilityAccountNo:     row.Get("nomor_rekening_fasilitas"),
		DebtorCIF:             row.Get("no_cif_debitur"),
		FacilityTypeCode:      row.Get("kode_jenis_fasilitas"),
		FundSource:            row.Get("sumber_dana"),
		PolicyStart:           row.Get("start_polis"),
		PolicyEnd:             row.Get("end_polis"),
		InterestRate:          row.Get("suku_bunga"),
		CurrencyCode:          row.Get("kode_valuta"),
		OriginalCurrencyValue: row.Get("nilai_mata_uang_asal"),
		Obligation:            row.Get("jumlah_kewajiban"),
		CollectibilityCode:    row.Get("kode_kolektibilitas"),
		DefaultDate:           row.Get("tanggal_macet"),
		DefaultReasonCode:     row.Get("kode_sebab_macet"),
		Arrears:               row.Get("tunggakan"),
		ConditionDate:         row.Get("tanggal_kondisi"),
		ConditionCode:         row.Get("kode_kondisi"),
		BranchCode:            row.Get("kode_kantor_cabang"),
		Remark:                row.Get("keterangan"),
		PolicyNo:              row.Get("no_polis"),
	}
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// NextSubmissionID mengembalikan nomor urut pengiriman berikutnya.
//
// `max + 1` dihitung dari isi penyimpanan, sama seperti `nvl(max(id),0)+1` di Oracle —
// termasuk kerapuhannya terhadap dua pengiriman bersamaan. Ditiru dengan sengaja supaya
// uji tidak lulus atas perilaku yang lebih baik daripada produksi.
func (r *Repo) NextSubmissionID(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var highest int64
	for _, submission := range r.submissions {
		if submission.ID > highest {
			highest = submission.ID
		}
	}
	return highest + 1, nil
}

// RecordSubmission mencatat satu pengiriman.
func (r *Repo) RecordSubmission(
	ctx context.Context,
	submission monitoringslinkojk.Submission,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.submissions = append(r.submissions, recordedSubmission{Submission: submission})
	return nil
}

// CompleteSubmission menyimpan jawaban sistem SLIK ke baris pengirimannya.
//
// Baris yang tidak ditemukan diabaikan tanpa galat, sama seperti `UPDATE … WHERE` yang
// tidak cocok satu baris pun di Oracle: ia berhasil, hanya tidak mengubah apa pun.
func (r *Repo) CompleteSubmission(
	ctx context.Context,
	submission monitoringslinkojk.Submission,
	result monitoringslinkojk.SubmissionResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.submissions {
		if r.submissions[i].ID == submission.ID &&
			r.submissions[i].ClaimID == submission.ClaimID {
			r.submissions[i].Result = result
			r.submissions[i].Completed = true
		}
	}
	return nil
}

// Submissions mengembalikan seluruh pengiriman yang tercatat. Hanya dipakai uji.
func (r *Repo) Submissions() []recordedSubmission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]recordedSubmission(nil), r.submissions...)
}

// recordedSubmission adalah satu baris `T_CLAIM_SLINK_INDIVIDU` di memori.
type recordedSubmission struct {
	monitoringslinkojk.Submission

	// Completed menandai jawaban SLIK sudah tersimpan — padanan `ID_TRANSACTION` terisi.
	Completed bool
	Result    monitoringslinkojk.SubmissionResult
}

// LoadDebtor menyusun data debitur contoh dari baris segmen F06.
//
// Repo dalam memori dipakai pengembangan lokal dan pengujian transport. Nilainya diambil
// dari baris yang memang ada, bukan dikarang: dengan begitu muatan yang terkirim saat
// dicoba lokal punya bentuk yang sama dengan yang nanti dikirim di produksi.
func (r *Repo) LoadDebtor(
	_ context.Context,
	claimID string,
	contractNo string,
) (monitoringslinkojk.Debtor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, record := range r.rows[monitoringslinkojk.SegmentF06] {
		if record.Row.Get("no_klaim") != claimID {
			continue
		}
		if contractNo != "" && record.Row.Get("contract_no") != contractNo {
			continue
		}

		debtor := monitoringslinkojk.Debtor{
			CustomerType: monitoringslinkojk.CustomerPerson,
			FirstName:    record.Row.Get("no_cif_debitur"),
			Gender:       record.Row.Get("jenis_kelamin"),
			DateOfBirth:  record.Row.Get("tanggal_lahir"),
		}
		if street := record.Row.Get("alamat"); street != "" {
			debtor.Addresses = []monitoringslinkojk.DebtorAddress{{
				Street:     street,
				PostalCode: record.Row.Get("kode_pos"),
				Primary:    true,
				Phones: []monitoringslinkojk.DebtorPhone{{
					Number: record.Row.Get("telepon"),
				}},
			}}
		}
		return debtor, nil
	}

	// Tidak ditemukan BUKAN galat — sama seperti pengisi SQL. Lihat di sana.
	return monitoringslinkojk.Debtor{}, nil
}
