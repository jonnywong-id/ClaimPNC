package usecase

import (
	"context"
	"fmt"
	"time"

	"claim-pnc/internal/registrasi"
)

// SpreadingInput adalah satu baris pembagian risiko yang dikirim layar.
type SpreadingInput struct {
	TreatyKind   string
	Name         string
	Share        registrasi.Percent
	Removed      bool
	FacOfferItem string
}

// CoverageInput adalah satu jaminan pada sebuah objek.
type CoverageInput struct {
	ID          string
	CauseOfLoss string
	TSI         registrasi.Money
	Spreading   []SpreadingInput
}

// InsuredItemInput adalah satu objek pertanggungan yang tertimpa kejadian.
type InsuredItemInput struct {
	ID       string
	Name     string
	Location string
	Coverage []CoverageInput
}

// RegisterCommand adalah seluruh isian layar Input Register.
type RegisterCommand struct {
	TaskID string

	DateOfLoss   time.Time
	ReportDate   time.Time
	DateReceived time.Time

	Location   string
	Chronology string
	Reporter   registrasi.Reporter

	EstimateValue registrasi.Money
	Currency      string
	SLIKNumber    string
	ExGratia      bool
	TechnicalPIC  string
	RCVID         string

	InsuredItem []InsuredItemInput

	PUCLStatus         int
	ComplianceTransfer bool

	// Return menandai petugas menekan tombol Back alih-alih menyimpan maju.
	//
	// Di sistem lama tombol ini bekerja dengan menuliskan teks "Back" ke field catatan
	// (`.pyNote`), lalu rule `IsBackStage` membandingkannya. Di sini ia adalah maksud
	// yang dinyatakan langsung, bukan teks yang ditafsirkan.
	Return bool
}

// RegisterResult adalah keadaan klaim setelah tahap Input Register ditutup.
type RegisterResult struct {
	Claim    registrasi.Claim
	NextTask *registrasi.Task

	// DecisionTrace menyebut keputusan alur yang dilewati beserta cabang yang dipilih,
	// misalnya "IsPA → isPA_PNC". Ia dikirim ke layar dan direkam di jejak audit supaya
	// klaim yang berpindah ke tahap tak terduga dapat dijelaskan.
	DecisionTrace []string

	// LargeLoss menyatakan Notice of Large Losses ikut diterbitkan.
	LargeLoss bool
}

// SaveRegister menutup tahap Input Register.
//
// # Urutan pekerjaan, dan kenapa urutannya begitu
//
//  1. Muat klaim beserta tugas berjalannya, dan tolak bila pemanggil tidak sedang
//     berada di tahap Input Register.
//  2. Terapkan isian ke klaim.
//  3. Cari klaim ganda — satu-satunya bagian validasi yang menyentuh basis data.
//  4. Jalankan gerbang validasi.
//  5. Konversi nilai estimasi ke rupiah memakai kurs TANGGAL KEJADIAN (`ADR-0015`).
//     Kurs yang tidak ditemukan MENOLAK klaim; ia tidak diganti nilai bawaan.
//  6. Terbitkan nomor klaim — hanya bila klaim maju, dan hanya bila belum bernomor.
//  7. Pindahkan klaim ke tahap berikutnya menurut alur.
//  8. Tulis seluruhnya dalam SATU transaksi, bersama jejak audit dan pemberitahuan.
//
// Nomor klaim terbit pada langkah 6, bukan lebih awal: ia tidak dapat ditarik kembali,
// dan menerbitkannya sebelum validasi lolos berarti membakar satu nomor setiap kali
// petugas salah ketik.
func (l *Service) SaveRegister(ctx context.Context, p RegisterCommand, by Caller) (RegisterResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{
		ctx:           ctx,
		taskID:        p.TaskID,
		requiredStage: registrasi.StageInputRegister,
	})
	if err != nil {
		return RegisterResult{}, err
	}
	if task.Owned() && task.Owner != by.Identity {
		return RegisterResult{}, registrasi.ErrNotTaskOwner
	}

	now := l.clock.Now().UTC()
	applyInput(&claim, p, by, now)

	if p.Return {
		// Bukti: `InputRegister_act` langkah 6 — `Property-Set` dengan prasyarat
		// `.pyNote=="Back"` mengisi `StatusClaim` dengan `1146`.
		claim.ClaimStatus = registrasi.StatusReturned
	}

	if !p.Return || l.validateOnReturn {
		duplicate, err := l.claim.FindDuplicates(ctx, registrasi.DuplicateKeys(claim), claim.ID)
		if err != nil {
			return RegisterResult{}, fmt.Errorf("registrasi/usecase: memeriksa klaim ganda: %w", err)
		}
		if err := registrasi.Validate(claim, registrasi.Parts{Now: now, Duplicates: duplicate}); err != nil {
			return RegisterResult{}, err
		}
	}

	var (
		rupiahValue registrasi.Money
		largeLoss   bool
		recipients  []string
	)
	if !p.Return {
		rate, err := l.rate.Find(ctx, claim.Currency, claim.DateOfLoss)
		if err != nil {
			return RegisterResult{}, err
		}
		rupiahValue = claim.EstimateValue.Convert(rate)

		threshold, err := l.parameter.LargeLossThreshold(ctx)
		if err != nil {
			return RegisterResult{}, fmt.Errorf("registrasi/usecase: membaca ambang large losses: %w", err)
		}
		// Ambang dilampaui berarti LEBIH BESAR, bukan sama dengan. Sistem lama memakai
		// `> 1000000000` pada precondition langkah 4 dan langkah 10
		// `Activity/SendEmailLargeLoss_act.xml`, dan `TKT-B02-004` menegaskan nilai TEPAT
		// pada threshold tidak memicu apa pun.
		if rupiahValue > threshold {
			largeLoss = true

			// Penerima yang belum lengkap TIDAK menggagalkan pendaftaran.
			//
			// Ini mengikuti sistem lama, bukan melonggarkannya. Di sana penerima dirakit
			// dari LIMA sumber — UW menurut Group Panel, jajaran pimpinan, email PIC
			// teknis klaim, daftar akunting, dan email cabang/GL/Pincab dari sebuah
			// kueri — lalu disambung menjadi satu string (langkah 7 dan 8). Satu sumber
			// yang kosong hanya membuat sambungannya lebih pendek; ia tidak pernah
			// menghentikan registrasi.
			//
			// Memperlakukan master yang belum diisi sebagai galat akan MENOLAK setiap
			// klaim di atas Rp 1 miliar — perilaku yang tidak ada di sistem lama, dan
			// yang akibatnya jauh lebih besar daripada pemberitahuan tanpa tujuan.
			// Peristiwanya tetap terbit dan tercatat; yang kosong adalah daftar
			// penerimanya, dan itu terlihat di jejak audit maupun di mode periksa.
			recipients, err = l.parameter.LargeLossRecipients(ctx, claim.Policy.Line)
			if err != nil {
				recipients = nil
			}
		}

	}

	// Penerbitan nomor, perpindahan tahap, dan seluruh penulisan berada di dalam SATU
	// transaksi.
	//
	// Nomor klaim ikut ke dalamnya dengan sengaja. Menerbitkannya di luar berarti
	// transaksi yang gagal meninggalkan satu nomor yang sudah terpakai dan tidak pernah
	// menjadi klaim — lubang di deret nomor yang muncul di surat ke tertanggung dan di
	// PLA/DLA ke reasuransi, dan tidak dapat ditambal karena nomor tidak dapat ditarik
	// kembali (`ADR-0009`).
	var (
		newTask *registrasi.Task
		trace   []string
	)

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if !p.Return && claim.Number == "" {
			number, err := l.number.Issue(ctx, now)
			if err != nil {
				return fmt.Errorf("registrasi/usecase: menerbitkan nomor klaim: %w", err)
			}
			claim.Number = number
			claim.ClaimStatus = registrasi.StatusRegistered
		}

		var err error
		newTask, trace, err = l.advance(advanceContext{
			ctx:    ctx,
			claim:  &claim,
			task:   &task,
			caller: by,
			action: registrasi.ActionInputRegister,
			now:    now,
		})
		if err != nil {
			return err
		}

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
		if err := l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       registerEvent(p.Return),
			Actor:       by.Identity,
			At:          now,
			Note:        traceNote(trace),
		}); err != nil {
			return err
		}
		// Nomor klaim baru terbit di atas, jadi inilah saat berkas laporannya dapat
		// dipasangi NOKLAIM — yang memindahkannya ke "Outstanding".
		//
		// Hanya pada pendaftaran maju: menekan Back tidak menerbitkan nomor, dan
		// memasang nomor kosong akan membuat berkasnya lenyap dari seluruh tab.
		if !p.Return && claim.RCVID != "" && claim.Number != "" {
			if err := l.reportLink.AttachClaimNumber(ctx, claim.RCVID, claim.Number); err != nil {
				return err
			}
		}

		if !largeLoss {
			return nil
		}
		// Revisi ditentukan oleh keadaan SEBELUM pemberitahuan ini terbit, lalu
		// penandanya dinaikkan — urutan yang sama dengan langkah 7/8 lalu langkah 11.
		revision := claim.LargeLossNoticed
		if err := l.notifier.Send(ctx, registrasi.Notification{
			Kind:         registrasi.NotificationLargeLoss,
			ClaimNumber:  claim.Number,
			PolicyNumber: claim.Policy.Number,
			Recipients:   recipients,
			RupiahValue:  rupiahValue,
			Revision:     revision,
			At:           now,
		}); err != nil {
			return err
		}
		if revision {
			return nil
		}
		// Penanda disimpan lewat Save kedua, bukan dengan memindahkan Save ke belakang:
		// urutannya harus tetap simpan-klaim → tugas → audit → beritahu, dan keduanya
		// berada di dalam transaksi yang sama sehingga tidak dapat terpisah.
		claim.LargeLossNoticed = true
		return l.claim.Save(ctx, claim)
	})
	if err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		Claim:         claim,
		NextTask:      newTask,
		DecisionTrace: trace,
		LargeLoss:     largeLoss,
	}, nil
}

func registerEvent(back bool) string {
	if back {
		return "REGISTER_DIKEMBALIKAN"
	}
	return "KLAIM_TERDAFTAR"
}

func traceNote(trace []string) string {
	if len(trace) == 0 {
		return "tanpa percabangan"
	}
	result := trace[0]
	for _, j := range trace[1:] {
		result += "; " + j
	}
	return result
}

// applyInput menyalin isian layar ke klaim.
//
// Ia tidak memvalidasi apa pun: pemisahan itu disengaja, supaya validasi berjalan atas
// klaim yang utuh — persis seperti yang akan tersimpan — bukan atas potongan isian.
func applyInput(k *registrasi.Claim, p RegisterCommand, by Caller, now time.Time) {
	k.DateOfLoss = p.DateOfLoss.UTC()
	k.ReportDate = p.ReportDate.UTC()
	k.DateReceived = p.DateReceived.UTC()

	k.Location = p.Location
	k.Chronology = p.Chronology
	k.Reporter = p.Reporter

	k.EstimateValue = p.EstimateValue
	if p.Currency != "" {
		k.Currency = p.Currency
	}
	k.SLIKNumber = p.SLIKNumber
	k.ExGratia = p.ExGratia
	k.TechnicalPIC = p.TechnicalPIC

	// RCVID hanya DITAMBAHKAN, tidak pernah dikosongkan oleh form ini.
	//
	// Tautan ke berkas Receive Document dibuat saat klaim dibuka, dan form Input Register
	// tidak memilikinya. Menimpanya apa adanya membuat layar yang tidak mengirim medan ini
	// MEMUTUS tautannya — dan akibatnya baru terlihat jauh kemudian, sebagai berkas yang
	// tidak pernah berpindah dari "Not Registered" ke "Outstanding".
	if p.RCVID != "" {
		k.RCVID = p.RCVID
	}

	k.PUCLStatus = p.PUCLStatus
	k.ComplianceTransfer = p.ComplianceTransfer
	k.RequestReturn = p.Return

	k.InsuredItem = make([]registrasi.InsuredItem, 0, len(p.InsuredItem))
	for _, o := range p.InsuredItem {
		insuredItem := registrasi.InsuredItem{
			ID:       o.ID,
			Name:     o.Name,
			Location: o.Location,
			Coverage: make([]registrasi.Coverage, 0, len(o.Coverage)),
		}
		for _, c := range o.Coverage {
			coverage := registrasi.Coverage{
				ID:          c.ID,
				CauseOfLoss: c.CauseOfLoss,
				TSI:         c.TSI,
				Spreading:   make([]registrasi.Spreading, 0, len(c.Spreading)),
			}
			for _, s := range c.Spreading {
				spr := registrasi.Spreading{
					TreatyKind:   s.TreatyKind,
					Name:         s.Name,
					Share:        s.Share,
					Removed:      s.Removed,
					FacOfferItem: s.FacOfferItem,
				}
				// Klaim ex gratia mengubah jenis treaty menjadi ORS (langkah
				// 37.3.5.10 pada InputRegister_act).
				if p.ExGratia {
					spr.TreatyKind = registrasi.TreatyExGratia
				}
				coverage.Spreading = append(coverage.Spreading, spr)
			}
			insuredItem.Coverage = append(insuredItem.Coverage, coverage)
		}
		k.InsuredItem = append(k.InsuredItem, insuredItem)
	}

	k.UpdatedBy = by.Identity
	k.UpdatedAt = now
}
