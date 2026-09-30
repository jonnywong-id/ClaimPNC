// Package memory memenuhi seam inboxbandinghargasalvage.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat dan
//     tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa isinya DUA tabel, bukan satu daftar baris jadi
//
// Karena kedua grid layar ini membaca tabel yang BERBEDA — `T_CLAIM_CHEKER_SALVAGE` dan
// `PNC_SALVAGE` — dan penyaring keduanya pun berbeda bentuknya. Menyimpan "baris siap
// tampil" berarti penyaringnya tidak ikut diuji sama sekali, dan uji yang lulus di sini
// tidak akan menyatakan apa pun tentang yang berjalan di Oracle.
//
// # Kenapa penyaringnya ditiru, bukan disederhanakan
//
// Alasan yang sama. Keempat penyaringnya ditiru sedekat-dekatnya dengan predikat SQL-nya,
// termasuk tiga kenyataan yang mudah tergoda untuk "dirapikan":
//
//   - Pencarian COCOK PERSIS, bukan mengandung.
//   - Grid History tidak ber-DISTINCT, sehingga satu klaim dapat muncul beberapa kali.
//   - Penyaring giliran komite hanya berlaku pada grid Request.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// CheckerRecord adalah satu baris `POOLDATA.T_CLAIM_CHEKER_SALVAGE`.
//
// Ia memuat kolom yang DIGAMBAR maupun yang hanya MENYARING. Keduanya perlu: tanpa kolom
// penyaring, penyimpanan ini tidak dapat menirukan klausa WHERE kuerinya.
type CheckerRecord struct {
	ClaimNo       string
	SalvageID     string
	DetailObject  string // IDDETAILSALVAGE
	CommitteeName string // NAMAKOMITE
	ItemName      string // NAMABARANG
	ItemPrice     string // HARGABARANG
	RequestNote   string // ALASANREQUEST
	CheckerNote   string // NOTEAPPROVE — bukan NOTEKOMITE; lihat kepala berkas .sql

	// RequestPrice adalah `HARGAREQUEST`. Kosong berarti NULL, dan barisnya TIDAK masuk
	// antrean — kueri menyaring `HARGAREQUEST IS NOT NULL`.
	RequestPrice string

	// RequestDate adalah `TGLREQUEST`. Nil berarti NULL; umurnya lalu dihitung nol.
	RequestDate *time.Time

	// ApprovedAt adalah `TGLAPPROVE`. Nil berarti BELUM diputus — itulah yang membuat
	// barisnya masuk grid Request.
	ApprovedAt *time.Time

	// ApprovalStatus adalah `STATUSAPPROVE`. Kosong berarti NULL.
	//
	// Ia dipakai DUA penyaring yang berlawanan arah: grid History menuntutnya TERISI,
	// sedangkan penyaring giliran komite menuntutnya KOSONG.
	ApprovalStatus string

	// InDetailSalvage menyatakan `IDDETAILSALVAGE`-nya ada di `DETAIL_PNC_SALVAGE`.
	//
	// Ia menirukan `IDDETAILSALVAGE IN (SELECT … FROM POOLDATA.DETAIL_PNC_SALVAGE)`.
	// Disimpan sebagai penanda, bukan dengan menyimpan seluruh tabel detail salvage: yang
	// diuji adalah ADA-TIDAKNYA, dan modul ini tidak membaca satu pun kolom lain dari sana.
	InDetailSalvage bool
}

// SalvageRecord adalah satu baris `POOLDATA.PNC_SALVAGE` sebagaimana dibaca grid History.
type SalvageRecord struct {
	ClaimNo         string
	SalvageID       string
	SalvageType     string // JENISSALVAGE — digambar sebagai kolom "Object Name"
	SalvageLocation string // LOKASISALVAGE
	PIC             string
}

// Store adalah penyimpanan antrean banding harga di memori.
//
// # Soal penguncian
//
// Store sendiri tidak mengunci apa pun, dan seluruh method-nya hanya MEMBACA. Yang menulis
// adalah Writer (decide.go), dan penguncian tinggal di sana — ia yang mengubah, ia pula yang
// bertanggung jawab menjaganya.
//
// Akibat yang perlu disadari saat menulis uji: WithSalvages dan WithNow menyerahkan salinan
// DANGKAL, sehingga salinannya berbagi baris checker yang sama dengan aslinya. Itu memang yang
// dikehendaki — keputusan yang ditulis Writer harus terlihat pada daftar yang dibaca Store,
// persis seperti di basis data — tetapi ia berarti dua Store yang tampak terpisah dapat saling
// memengaruhi.
type Store struct {
	checkers []CheckerRecord
	salvages []SalvageRecord

	// now menyerahkan waktu acuan perhitungan umur.
	//
	// Ia dapat diganti supaya uji umur dapat ditulis deterministik — di basis data ia
	// `SYSDATE`, dan menguji terhadap jam sungguhan berarti uji yang hasilnya berubah
	// setiap hari.
	now func() time.Time
}

// NewStore membentuk penyimpanan berisi baris checker yang diberikan.
//
// Daftar pengajuan salvage-nya kosong, sehingga grid History tidak punya baris. Itu
// disengaja: uji yang hanya menguji grid Request tidak perlu menyiapkan tabel kedua.
func NewStore(checkers ...CheckerRecord) *Store {
	return &Store{checkers: checkers, now: func() time.Time { return time.Now().UTC() }}
}

// WithSalvages menyerahkan salinan penyimpanan yang tabel pengajuan salvage-nya terisi.
func (s *Store) WithSalvages(salvages ...SalvageRecord) *Store {
	clone := *s
	clone.salvages = salvages
	return &clone
}

// WithNow menyerahkan salinan penyimpanan yang waktu acuannya tetap.
//
// Dipakai uji umur: dengan jam yang tetap, "12 hari" tetap 12 hari besok pagi.
func (s *Store) WithNow(at time.Time) *Store {
	clone := *s
	clone.now = func() time.Time { return at }
	return &clone
}

// NewSampleStore membentuk penyimpanan berisi baris contoh untuk pengembangan lokal.
func NewSampleStore() *Store {
	return NewStore(SampleCheckers()...).WithSalvages(SampleSalvages()...)
}

// List menyaring, mengurutkan, lalu memotong satu halaman.
//
// Urutan langkahnya sengaja sama dengan kuerinya: saring, urut, potong. Menukar urut dan
// potong akan menghasilkan halaman yang isinya benar tetapi urutannya bukan urutan yang
// dilihat pengguna.
func (s *Store) List(
	_ context.Context,
	q inboxbandinghargasalvage.Query,
	page inboxbandinghargasalvage.Pagination,
) (inboxbandinghargasalvage.Page, error) {
	matched := s.matchedRows(q)

	clean := page.Normalize()
	result := inboxbandinghargasalvage.Page{
		Items:      []inboxbandinghargasalvage.AppealRow{},
		Total:      len(matched),
		Pagination: clean,
	}

	offset := clean.Offset()
	if offset >= len(matched) {
		return result, nil
	}

	end := offset + clean.Size
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[offset:end]

	return result, nil
}

// Count menghitung baris yang cocok, tanpa paginasi.
//
// Ia memakai penyaring yang SAMA dengan List — bukan penyaring tersendiri — supaya angka
// ringkas tidak dapat berselisih dengan jumlah baris gridnya.
func (s *Store) Count(
	_ context.Context,
	q inboxbandinghargasalvage.Query,
) (int, error) {
	return len(s.matchedRows(q)), nil
}

// matchedRows menyerahkan seluruh baris yang cocok, sudah terurut.
func (s *Store) matchedRows(q inboxbandinghargasalvage.Query) []inboxbandinghargasalvage.AppealRow {
	if q.Tab.Decided {
		return s.historyRows(q)
	}
	return s.requestRows(q)
}

// requestRows menirukan list_request.
func (s *Store) requestRows(
	q inboxbandinghargasalvage.Query,
) []inboxbandinghargasalvage.AppealRow {
	komite := normal(q.Reviewer.Name)
	waitFor := normal(q.Reviewer.WaitFor)
	today := truncateDay(s.now())

	type scored struct {
		row    inboxbandinghargasalvage.AppealRow
		detail string
	}

	matched := []scored{}
	for _, record := range s.checkers {
		// `IDDETAILSALVAGE IN (SELECT … FROM DETAIL_PNC_SALVAGE)`
		if !record.InDetailSalvage {
			continue
		}
		// `HARGAREQUEST IS NOT NULL`
		if strings.TrimSpace(record.RequestPrice) == "" {
			continue
		}
		// `TGLAPPROVE IS NULL`
		if record.ApprovedAt != nil {
			continue
		}
		// `UPPER(TRIM(NAMAKOMITE)) = :1`
		if normal(record.CommitteeName) != komite {
			continue
		}
		// `:2 IS NULL OR UPPER(TRIM(NOKLAIM)) = :3` — cocok persis, bukan mengandung.
		if q.Searching() && normal(record.ClaimNo) != q.Keyword {
			continue
		}
		// `:4 IS NULL OR NOT EXISTS (…)` — giliran komite sebelumnya.
		if waitFor != "" && s.pendingExists(record.DetailObject, waitFor) {
			continue
		}

		matched = append(matched, scored{
			row: inboxbandinghargasalvage.AppealRow{
				ClaimNo:       record.ClaimNo,
				RequestDate:   record.RequestDate,
				DetailObject:  record.DetailObject,
				ItemName:      record.ItemName,
				ItemPrice:     record.ItemPrice,
				RequestPrice:  record.RequestPrice,
				RequestNote:   record.RequestNote,
				AgingDays:     agingDays(today, record.RequestDate),
				CheckerNote:   record.CheckerNote,
				SalvageID:     record.SalvageID,
				CommitteeName: record.CommitteeName,
			},
			detail: record.DetailObject,
		})
	}

	// `ORDER BY AGING_DAYS DESC, IDDETAILSALVAGE` — yang paling lama menunggu di atas.
	//
	// Perbandingan umurnya ANGKA, bukan teks. Itulah selisih terencana nomor 1: di Pega ia
	// teks, sehingga '9 days' mendahului '30 days'.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].row.AgingDays != matched[j].row.AgingDays {
			return matched[i].row.AgingDays > matched[j].row.AgingDays
		}
		return matched[i].detail < matched[j].detail
	})

	rows := make([]inboxbandinghargasalvage.AppealRow, 0, len(matched))
	for _, item := range matched {
		rows = append(rows, item.row)
	}
	return rows
}

// historyRows menirukan list_history.
func (s *Store) historyRows(
	q inboxbandinghargasalvage.Query,
) []inboxbandinghargasalvage.AppealRow {
	komite := normal(q.Reviewer.Name)

	// Sub-kueri `NOKLAIM IN (SELECT … WHERE STATUSAPPROVE IS NOT NULL AND TGLAPPROVE IS NOT
	// NULL)`, disusun lebih dulu supaya penelusurannya tidak bersarang.
	decided := map[string]bool{}
	for _, record := range s.checkers {
		if normal(record.CommitteeName) != komite {
			continue
		}
		if strings.TrimSpace(record.ApprovalStatus) == "" || record.ApprovedAt == nil {
			continue
		}
		decided[normal(record.ClaimNo)] = true
	}

	matched := []SalvageRecord{}
	for _, record := range s.salvages {
		if !decided[normal(record.ClaimNo)] {
			continue
		}
		if q.Searching() && normal(record.ClaimNo) != q.Keyword {
			continue
		}
		matched = append(matched, record)
	}

	// `ORDER BY NOKLAIM, IDSALVAGE`. Tanpa DISTINCT — satu klaim dapat muncul beberapa kali
	// bila ia punya lebih dari satu pengajuan salvage, persis seperti kuerinya.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].ClaimNo != matched[j].ClaimNo {
			return matched[i].ClaimNo < matched[j].ClaimNo
		}
		return matched[i].SalvageID < matched[j].SalvageID
	})

	rows := make([]inboxbandinghargasalvage.AppealRow, 0, len(matched))
	for _, record := range matched {
		rows = append(rows, inboxbandinghargasalvage.AppealRow{
			ClaimNo:         record.ClaimNo,
			SalvageType:     record.SalvageType,
			SalvageLocation: record.SalvageLocation,
			PIC:             record.PIC,
		})
	}
	return rows
}

// pendingExists menirukan sub-kueri `NOT EXISTS` penyaring giliran komite.
//
// Ia bernilai benar bila komite yang ditunggu MASIH punya baris belum diputus atas barang
// yang sama — dan barisnya karena itu belum boleh tampil.
func (s *Store) pendingExists(detailObject, waitFor string) bool {
	for _, record := range s.checkers {
		if record.DetailObject != detailObject {
			continue
		}
		if normal(record.CommitteeName) != waitFor {
			continue
		}
		if strings.TrimSpace(record.ApprovalStatus) == "" {
			return true
		}
	}
	return false
}

// agingDays menirukan `CAST(CURRENT_TIMESTAMP AS DATE) - CAST(TGLREQUEST AS DATE)`.
//
// Kedua sisi dipangkas ke tanggalnya lebih dulu — tanpa itu, selisih dihitung dari JAM dan
// banding yang masuk kemarin sore terhitung nol hari sampai lewat 24 jam.
//
// Tanggal yang kosong menghasilkan nol, sama dengan pemindai sqlstore.
func agingDays(today time.Time, requestDate *time.Time) int {
	if requestDate == nil {
		return 0
	}
	return int(today.Sub(truncateDay(*requestDate)).Hours() / 24)
}

// truncateDay memotong waktu ke tanggalnya, menirukan `TRUNC` Oracle.
func truncateDay(at time.Time) time.Time {
	utc := at.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

// normal menyeragamkan teks sebelum dibandingkan, menirukan `UPPER(TRIM(...))` pada SQL.
func normal(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// ListDecisions menirukan list_decisions.
//
// Penyaringnya ditiru sedekat-dekatnya, termasuk `NAMAKOMITE` — tanpa itu, uji yang lulus di
// sini tidak menyatakan apa pun tentang kebocoran antarkomite yang justru sedang dijaga.
func (s *Store) ListDecisions(
	_ context.Context,
	q inboxbandinghargasalvage.DecisionQuery,
) ([]inboxbandinghargasalvage.Decision, error) {
	klaim := normal(q.ClaimNo)
	komite := normal(q.Reviewer.Name)

	matched := []inboxbandinghargasalvage.Decision{}
	for _, record := range s.checkers {
		if normal(record.ClaimNo) != klaim {
			continue
		}
		if normal(record.CommitteeName) != komite {
			continue
		}
		// `STATUSAPPROVE IS NOT NULL` — keputusan yang belum diambil tidak masuk panel.
		if strings.TrimSpace(record.ApprovalStatus) == "" {
			continue
		}

		matched = append(matched, inboxbandinghargasalvage.Decision{
			ApprovedAt:    record.ApprovedAt,
			DetailObject:  record.DetailObject,
			ItemName:      record.ItemName,
			ItemPrice:     record.ItemPrice,
			RequestPrice:  record.RequestPrice,
			Status:        record.ApprovalStatus,
			CommitteeName: record.CommitteeName,
		})
	}

	// `ORDER BY TGLAPPROVE DESC, IDDETAILSALVAGE` — yang paling baru diputus di atas.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].ApprovedAt, matched[j].ApprovedAt
		switch {
		case left == nil && right == nil:
		case left == nil:
			// Tanggal kosong ditaruh di BELAKANG, sama seperti `DESC` pada Oracle yang
			// bawaannya menaruh NULL lebih dulu — dibalik di sini supaya keputusan yang
			// tanggalnya terbaca selalu tampil lebih dahulu.
			return false
		case right == nil:
			return true
		case !left.Equal(*right):
			return left.After(*right)
		}
		return matched[i].DetailObject < matched[j].DetailObject
	})

	return matched, nil
}
