package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/mastersurveyors"
)

// Committee adalah anggota komite yang memutuskan.
//
// Identity dibandingkan dengan kolom KOMITE, dan keduanya berisi **Operator ID** — lihat
// catatan panjang di paket committee tentang alias `BUSINESS_CODE` yang sebenarnya
// memuat `OPERATOR_ID`.
type Committee struct {
	Identity string
	Name     string
}

// Decision adalah apa yang diputuskan komite atas seorang surveyor.
type Decision struct {
	// Status wajib StatusApproved atau StatusRejected. StatusPending ditolak — "belum
	// memutuskan" bukan keputusan.
	Status mastersurveyors.ApprovalStatus

	// Note adalah keterangan yang menyertai keputusan.
	Note string
}

// Decide mencatat keputusan komite atas seorang surveyor.
//
// # Asalnya di sistem lama — DITEMUKAN 2026-09-20
//
// Tidak ada rule Approve/Reject tersendiri, dan itulah sebabnya ia sempat disangka hilang.
// Kedua tombolnya memanggil activity YANG SAMA dengan tombol Simpan, dibedakan hanya oleh
// satu parameter. Dibaca dari `Section/BrowseDetailSuveryorsKomite-Section.xml`:
//
//	tombol Approve  →  CNMInsertDetailSurveyors_act( approval = "1" )
//	tombol Reject   →  CNMInsertDetailSurveyors_act( approval = "2" )
//	tombol Simpan   →  CNMInsertDetailSurveyors_act( approval = "0" )
//
// dan di dalam activity itu, pada langkah 4:
//
//	TempDetailSurveyors.APPROVAL := Param.approval
//
// Kedua tombol itu HANYA ada di section `-Komite`, yaitu tab "Komite Approval". Tab
// Waiting, Approve, dan Reject tidak memilikinya — ketiganya hanya menampilkan.
//
// # Yang SENGAJA dibuat berbeda di sini
//
// Sistem lama memakai SATU jalan masuk untuk menyimpan dan memutuskan. Modul ini
// memisahkannya: Submit/Update untuk isian, Decide untuk keputusan.
//
// Alasannya bukan kerapian. Satu jalan masuk berarti badan permintaan yang sama dapat
// membawa isian DAN status persetujuan sekaligus — sehingga siapa pun yang boleh menyunting
// surveyor dapat menyetujuinya sendiri dalam satu permintaan. `D-59` menetapkan tidak ada
// pemisahan tugas formal, jadi bentuk kontraknya adalah satu-satunya penjagaan yang tersisa.
//
// Keadaan akhirnya tetap sama dengan Pega: kolom APPROVAL berisi "0", "1", atau "2".
//
// # Urutan langkahnya
//
//  1. baca surveyornya
//  2. tolak bila keputusannya sudah pernah diambil
//  3. tolak bila pemutusnya bukan komite yang ditunjuk
//  4. catat keputusannya
//
// Langkah 3 adalah satu-satunya kontrol kewenangan yang benar-benar ada di modul ini.
// `D-59` menetapkan satuan izin adalah MENU dan tidak ada pemisahan tugas formal, sehingga
// siapa pun yang punya menu Master Data dapat membuka layar ini. Yang mencegah orang
// menyetujui surveyor yang bukan tanggung jawabnya hanyalah perbandingan di langkah 3 —
// dan itulah sebabnya ia tidak boleh dilonggarkan menjadi peringatan.
func (s *Service) Decide(
	ctx context.Context,
	portalAlias, id string,
	d Decision,
	by Committee,
) (mastersurveyors.Surveyor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	if d.Status != mastersurveyors.StatusApproved && d.Status != mastersurveyors.StatusRejected {
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrUnknownStatus
	}

	surveyor, err := repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, mastersurveyors.ErrNotFound) {
			return mastersurveyors.Surveyor{}, err
		}
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/usecase: membaca surveyor %q: %w", id, err)
	}

	if !surveyor.AwaitingDecision() {
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrAlreadyDecided
	}
	if err := ensureIsAssignedCommittee(surveyor, by); err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	surveyor.Status = d.Status
	surveyor.Note = strings.TrimSpace(d.Note)
	decidedAt := s.clock.Now()
	surveyor.DecidedAt = &decidedAt
	surveyor.UpdatedBy = strings.TrimSpace(by.Identity)

	// TRFKOMITE ditandai sudah diteruskan begitu keputusan diambil.
	//
	// Nilai "1" adalah DUGAAN yang dinyatakan, bukan bacaan: kolom ini hanya pernah
	// DISALIN di export (`SetDetailSurveryorsValue_act`) dan tidak pernah dibandingkan
	// dengan apa pun, sehingga nilai yang bermakna baginya tidak dapat diketahui. Nilai
	// "1" dipilih karena itu yang dipakai seluruh kolom penanda biner lain di basis data
	// yang sama. Bila Tim Pega mengirim rule-nya dan ternyata berbeda, yang berubah hanya
	// baris ini.
	surveyor.CommitteeTransferred = "1"

	if err := repo.Update(ctx, surveyor); err != nil {
		return mastersurveyors.Surveyor{}, translateRepoError(err, "menyimpan keputusan komite")
	}
	return surveyor, nil
}

// ensureIsAssignedCommittee menolak keputusan dari orang yang bukan komite yang ditunjuk.
//
// # Dua keadaan yang ditangani berbeda, dan sengaja
//
// Surveyor BERKOMITE KOSONG dapat diputuskan siapa pun yang membuka layar ini. Itu bukan
// kelonggaran yang terlewat — ia satu-satunya jalan keluar dari keadaan yang memang dapat
// terjadi: `GetKomiteApproval` mengambil `pxResults(1)` tanpa memeriksa apakah ada
// hasilnya, sehingga surveyor tanpa komite dapat lahir. Tanpa jalan keluar ini, baris
// semacam itu akan menggantung selamanya tanpa seorang pun berwenang memutuskannya.
//
// Perbandingannya mengabaikan besar-kecil huruf, dengan alasan yang sudah tercatat:
// `docs/Steering/11-SECURITY.md` §3.1 mencatat identitas di sistem lama muncul dalam dua
// kapitalisasi berbeda karena perbandingan rule lama tidak konsisten soal itu.
func ensureIsAssignedCommittee(surveyor mastersurveyors.Surveyor, by Committee) error {
	assigned := strings.TrimSpace(surveyor.Committee)
	if assigned == "" {
		return nil
	}
	if strings.EqualFold(assigned, strings.TrimSpace(by.Identity)) {
		return nil
	}
	return mastersurveyors.ErrNotAssignedCommittee
}
