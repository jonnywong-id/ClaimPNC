package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// EstimationInput adalah satu baris estimasi dari layar. Kurs dan nilai rupiahnya
// dihitung layanan, bukan dikirim layar.
type EstimationInput struct {
	Type     string
	Currency string
	Date     time.Time
	Value    registrasi.Money
}

// ObjectItemInput adalah satu item objek pada sebuah coverage.
type ObjectItemInput struct {
	Name        string
	Description string
	Group       string
	Estimation  []EstimationInput
}

// EstimateCommand adalah isian tahap Input Estimasi.
//
// Item dikirim per coverage menurut urutan objek dan coverage klaim: Item[i][j] adalah
// item-item coverage ke-j pada objek ke-i. Objek dan coverage sendiri tidak diubah di
// tahap ini — keduanya milik Input Register.
type EstimateCommand struct {
	TaskID string
	Item   [][][]ObjectItemInput

	// Return menandai tombol Back: klaim kembali ke Input Register (Decision5/Decision6).
	Return bool

	// TechnicalPICNote adalah Catatan ke PIC Teknis (`.ClaimData.Remark`). nil berarti
	// catatan yang tersimpan tidak diubah.
	TechnicalPICNote *string
}

// Currencies membaca pilihan Mata Uang.
func (l *Service) Currencies(ctx context.Context) ([]registrasi.CurrencyOption, error) {
	return l.currency.Currencies(ctx)
}

// ItemOptions membaca dropdown Objek item estimasi untuk satu jaminan (objek dan kode
// coverage-nya) — lihat registrasi.ItemFromPropertyList dan ItemFromTravelPlan.
func (l *Service) ItemOptions(ctx context.Context, claimID, objectID, coverageID string) (registrasi.ItemChoices, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return registrasi.ItemChoices{}, err
	}
	choices := registrasi.ItemChoices{Default: registrasi.DefaultItemName(claim.Policy)}
	switch {
	case registrasi.ItemFromPropertyList(claim.Policy):
		choices.Option, err = l.options.ItemOptions(ctx, claim.Policy, objectID)
	case registrasi.ItemFromTravelPlan(claim.Policy):
		choices.Option, err = l.options.TravelBenefits(ctx, coverageID)
	default:
		choices.Option = []registrasi.ItemOption{{Name: registrasi.ItemOthers}}
	}
	if err != nil {
		return registrasi.ItemChoices{}, err
	}
	return choices, nil
}

// CoverageOptions adalah isi dropdown "Tambah coverage" pada tahap Input Register: coverage
// polis milik SATU objek, dibaca dari POOLDATA.T_COVERAGELIST_CARGO/ANEKA/FIRE/PERSON sesuai
// lini bisnis polis (lewat PolicyItemSource yang sama dengan pembukaan klaim).
//
// Tanggal kejadian sengaja tidak dipakai menyaring: dropdown menawarkan seluruh coverage
// objek itu, dan aturan periode tetap ditegakkan validasi saat register. Pilihan tidak
// disimpan ke tabel mana pun di sini — coverage tersimpan bersama klaim saat Save/Submit
// (keputusan Work Owner 2026-10-07: tidak ditulis ke POOLDATA.TC_PNC_COVERAGEDETAIL).
func (l *Service) CoverageOptions(ctx context.Context, claimID, objectID string) ([]registrasi.Coverage, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return nil, err
	}
	source, err := l.items.Items(ctx, claim.Policy)
	if err != nil {
		return nil, fmt.Errorf("registrasi/usecase: membaca coverage polis: %w", err)
	}
	for _, item := range registrasi.BuildInsuredItems(claim.Policy, time.Time{}, source) {
		if item.ID == objectID {
			return item.Coverage, nil
		}
	}
	return nil, nil
}

// SaveEstimate menyimpan isian Input Estimasi tanpa menutup tahap — tombol Save.
func (l *Service) SaveEstimate(ctx context.Context, p EstimateCommand, by Caller) (registrasi.Claim, error) {
	claim, _, now, err := l.prepareEstimate(ctx, p, by, registrasi.ActionInputSurveyor)
	if err != nil {
		return registrasi.Claim{}, err
	}
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "INPUT_ESTIMASI_DISIMPAN",
			Actor: by.Identity, At: now, Note: "Isian Input Estimasi disimpan tanpa menutup tahap",
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return claim, nil
}

// CompleteEstimate menutup tahap Input Estimasi — tombol Kirim PIC Teknik, atau Back bila
// Return.
//
// Kirim PIC Teknik adalah `finishAssignment` pada `InputEstimasiAdmin_SECT`: klaim menuju
// Decision5/Decision6, lalu Choose Surveyor (Non-MBU) atau Send To PIC Teknik (Travel) —
// keduanya dirutekan PNCTeknikRouter. Ia menjalankan gerbang ValidateEstimate lebih dulu,
// termasuk `isCFS`; Back tidak, seperti Back pada Input Register. Keduanya menyimpan isiannya.
func (l *Service) CompleteEstimate(ctx context.Context, p EstimateCommand, by Caller) (CompleteResult, error) {
	claim, task, now, err := l.prepareEstimate(ctx, p, by, "")
	if err != nil {
		return CompleteResult{}, err
	}
	if !p.Return {
		if err := registrasi.ValidateEstimate(claim); err != nil {
			return CompleteResult{}, err
		}
	}

	claim.RequestReturn = p.Return
	if p.Return {
		claim.ClaimStatus = registrasi.StatusReturned
	}

	newTask, trace, err := l.advance(advanceContext{
		ctx: ctx, claim: &claim, task: &task, caller: by,
		action: registrasi.ActionInputEstimate, now: now,
	})
	if err != nil {
		return CompleteResult{}, err
	}

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if newTask != nil {
			if err := l.task.Save(ctx, *newTask); err != nil {
				return err
			}
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "TAHAP_DITUTUP",
			Actor: by.Identity, At: now, Note: task.Stage + " → " + traceNote(trace),
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Claim: claim, NextTask: newTask, DecisionTrace: trace}, nil
}

// prepareEstimate memuat tugas Input Estimasi, memeriksa pemiliknya, lalu memasang isian
// beserta kurs dan nilai rupiahnya.
//
// alsoAction menerima tahap kedua selain Input Estimasi — InputSurveyor untuk Save, karena
// section InputEstimasiDetail juga ditanam di sana. Kirim PIC Teknik tidak memakainya.
func (l *Service) prepareEstimate(ctx context.Context, p EstimateCommand, by Caller, alsoAction string) (registrasi.Claim, registrasi.Task, time.Time, error) {
	claim, task, err := l.loadOpenTask(loadContext{
		ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputEstimate, alsoAction: alsoAction,
	})
	if err != nil {
		return registrasi.Claim{}, registrasi.Task{}, time.Time{}, err
	}
	if !l.canWork(task, by) {
		return registrasi.Claim{}, registrasi.Task{}, time.Time{}, registrasi.ErrNotTaskOwner
	}

	now := l.clock.Now().UTC()
	if err := l.applyEstimate(ctx, &claim, p, now); err != nil {
		return registrasi.Claim{}, registrasi.Task{}, time.Time{}, err
	}
	if p.TechnicalPICNote != nil {
		note := strings.TrimSpace(*p.TechnicalPICNote)
		if err := registrasi.ValidateTechnicalPICNote(note); err != nil {
			return registrasi.Claim{}, registrasi.Task{}, time.Time{}, err
		}
		claim.TechnicalPICNote = note
	}
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now
	return claim, task, now, nil
}

// applyEstimate memasang item dan estimasi ke coverage klaim.
//
// # Kurs
//
// Kurs diambil pada TANGGAL KEJADIAN (`ADR-0015`, `D-48`). Activity lama
// `SetConvertValueKurs_Estimation` memakai `getcurrencystandard(currency, sysdate)` — kurs
// hari penginputan. Perbedaannya adalah perbaikan eksplisit `D-49` butir 4, bukan selisih
// tak terduga. Mata uang kosong berarti mata uang polis, seperti langkah 2 activity itu.
//
// # Estimasi yang sudah tersimpan tidak dapat dihapus
//
// T_CLAIM_ESTIMASI tidak punya penanda hapus, dan penghapusan fisik dilarang (`D-66`).
// Permintaan yang mengurangi jumlah estimasi sebuah item karena itu ditolak.
func (l *Service) applyEstimate(ctx context.Context, claim *registrasi.Claim, p EstimateCommand, now time.Time) error {
	if len(p.Item) != len(claim.InsuredItem) {
		return fmt.Errorf("%w: jumlah objek estimasi %d, klaim %d", registrasi.ErrInvalidAction, len(p.Item), len(claim.InsuredItem))
	}
	for i := range claim.InsuredItem {
		object := &claim.InsuredItem[i]
		if len(p.Item[i]) != len(object.Coverage) {
			return fmt.Errorf("%w: jumlah coverage estimasi objek %d", registrasi.ErrInvalidAction, i+1)
		}
		for j := range object.Coverage {
			coverage := &object.Coverage[j]
			items := make([]registrasi.ObjectItem, 0, len(p.Item[i][j]))
			for n := len(p.Item[i][j]); n < len(coverage.Item); n++ {
				if len(coverage.Item[n].Estimation) > 0 {
					return fmt.Errorf("%w: item yang sudah punya estimasi tidak dapat dihapus", registrasi.ErrInvalidAction)
				}
			}
			for n, in := range p.Item[i][j] {
				if n < len(coverage.Item) && len(in.Estimation) < len(coverage.Item[n].Estimation) {
					return fmt.Errorf("%w: estimasi yang sudah disimpan tidak dapat dihapus", registrasi.ErrInvalidAction)
				}
				item := registrasi.ObjectItem{
					Name:        strings.TrimSpace(in.Name),
					Description: strings.TrimSpace(in.Description),
					Group:       strings.TrimSpace(in.Group),
				}
				var stored []registrasi.Estimation
				if n < len(coverage.Item) {
					stored = coverage.Item[n].Estimation
				}
				for m, e := range in.Estimation {
					// Estimasi yang sudah dibuatkan Claim Face Sheet tidak dapat diubah
					// (section Estimasi: `pyReadOnlyCondition .PrintFaceClaim = '1'`).
					// Isian layar untuknya diabaikan; nilai tersimpan dipakai apa adanya.
					if m < len(stored) && stored[m].FaceSheet {
						item.Estimation = append(item.Estimation, stored[m])
						continue
					}
					currency := strings.TrimSpace(e.Currency)
					if currency == "" {
						currency = claim.Policy.Currency
					}
					kind := strings.TrimSpace(e.Type)
					if kind == "" {
						kind = registrasi.EstimateClaim
					}
					date := e.Date
					if date.IsZero() {
						date = now
					}
					rate, err := l.rate.Find(ctx, currency, claim.DateOfLoss)
					if err != nil {
						return err
					}
					item.Estimation = append(item.Estimation, registrasi.Estimation{
						Type: kind, Currency: currency, Date: date.UTC(), Value: e.Value,
						Rate: rate, Converted: e.Value.Convert(rate),
					})
				}
				if err := registrasi.CheckNewEstimates(item, len(stored)); err != nil {
					return err
				}
				items = append(items, item)
			}
			coverage.Item = items
		}
	}
	return nil
}
