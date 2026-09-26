// Package memory memenuhi seam inboxsalvage.Repo tanpa basis data.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengembangan lokal tanpa Oracle. Kredensial basis data pengembangan tidak dibagikan
//     ke setiap orang yang menyentuh layar ini.
//   - Uji aturan bisnis TANPA infrastruktur (`14-TESTING-STRATEGY.md` §3). Aturan yang
//     diuji di sini — penyaring tiap tab, arti pencarian, pemetaan pencacah — adalah aturan
//     yang sama yang ditegakkan penyimpanan SQL.
//
// # Yang ia JANJIKAN sama dengan penyimpanan SQL, dan yang tidak
//
// Sama: baris mana yang lolos penyaring tab, baris mana yang lolos pencarian, urutan baris,
// dan bentuk halaman.
//
// TIDAK sama: ID salvage yang terbit saat menyimpan. Di Oracle ia `max(IDSALVAGE) + 1`
// dibaca dari tabel; di sini ia pencacah di dalam proses. Keduanya menghasilkan nomor yang
// menaik, dan tidak ada aturan bisnis yang bergantung pada nilainya sendiri.
package memory

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/inboxsalvage"
)

// Claim adalah satu baris `POOLDATA.T_CLAIM_PNC` sejauh yang dibaca layar ini.
//
// Hanya tujuh kolomnya yang disimpan — sisanya tidak pernah dibaca modul ini, dan menyalin
// seluruh tabel ke sini akan membuat data contoh tampak lebih berwenang daripada
// sebenarnya.
type Claim struct {
	ClaimNo      string
	PIC          string
	BusinessName string
	LossDate     string
	ObjectName   string

	// SalvageStatus adalah `STSSALVAGE`. Kosong berarti `NULL` — dan justru itulah yang
	// disaring daftar Salvage Outstanding.
	SalvageStatus string

	// WorkStatus adalah `STATUSWORK`. Hanya daftar Salvage Outstanding yang menyaringnya.
	WorkStatus string

	// HasBuybackValue menandai klaim ini punya baris adjustment ber-`NILAI_SALVAGE_A`
	// terisi — penyaring daftar Salvage Buyback.
	HasBuybackValue bool
}

// Salvage adalah satu baris `POOLDATA.PNC_SALVAGE` beserta agregat detailnya.
type Salvage struct {
	SalvageID      string
	ClaimNo        string
	InputDate      string
	PIC            string
	SalvageType    string
	Location       string
	Quantity       string
	EstimateValue  string
	Email          string
	Remark         string
	AcceptanceNo   string
	TransferStatus string

	// AcceptedValue adalah `NILAIAKSEP`. Ia tidak digambar sebagai kolom; yang digambar
	// adalah turunannya, "Status Lelang".
	AcceptedValue string

	// RequestValue dan RequestNote adalah agregat atas `DETAIL_PNC_SALVAGE`.
	//
	// Di Oracle keduanya lahir dari `max(NILAI_REQUEST)` dan `max(NOTE_REQUEST)`; di sini
	// keduanya disimpan apa adanya, karena yang diuji adalah penggambarannya, bukan cara
	// menghitungnya.
	RequestValue string
	RequestNote  string
}

// Store adalah penyimpanan di memori.
//
// Ia aman dipakai bersamaan: satu Store dipegang seluruh permintaan satu portal, dan
// permintaan datang dari goroutine yang berbeda.
type Store struct {
	mu       sync.RWMutex
	claims   []Claim
	salvages []Salvage
	nextID   int

	// items adalah daftar barang per ID salvage.
	//
	// Ia peta, bukan senarai di dalam Salvage, karena panel detail membacanya lewat ID —
	// dan Create menambahnya tanpa menyentuh baris pengajuannya.
	items map[string][]inboxsalvage.DetailBarang

	// now memasok tanggal hari ini, dapat diganti uji.
	//
	// Ia ada karena kolom "Aging" dihitung terhadap hari ini, dan uji yang memakai jam
	// sistem akan menghasilkan angka yang berubah setiap hari (`F-5`).
	now func() string
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{
		nextID: 1,
		now:    todayISO,
		items:  map[string][]inboxsalvage.DetailBarang{},
	}
}

// Seed mengisi penyimpanan dengan data yang diberikan.
func (s *Store) Seed(claims []Claim, salvages []Salvage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.claims = append([]Claim{}, claims...)
	s.salvages = append([]Salvage{}, salvages...)

	s.nextID = 1
	for _, item := range s.salvages {
		if number, err := strconv.Atoi(item.SalvageID); err == nil && number >= s.nextID {
			s.nextID = number + 1
		}
	}
}

// SetNow mengganti pemasok tanggal hari ini. Dipakai uji.
func (s *Store) SetNow(now func() string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// List mengembalikan satu halaman baris yang cocok.
func (s *Store) List(
	ctx context.Context,
	query inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	if err := ctx.Err(); err != nil {
		return inboxsalvage.Page{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows []inboxsalvage.Row
	switch query.Tab.Family {
	case inboxsalvage.FamilySalvage:
		rows = s.salvageRows(query)
	default:
		rows = s.claimRows(query)
	}

	matched := make([]inboxsalvage.Row, 0, len(rows))
	for _, row := range rows {
		if query.Matches(row) {
			matched = append(matched, row)
		}
	}

	return inboxsalvage.Slice(matched, page), nil
}

// claimRows menyusun baris keluarga A dan B.
func (s *Store) claimRows(query inboxsalvage.Query) []inboxsalvage.Row {
	tab := query.Tab

	rows := []inboxsalvage.Row{}
	for _, claim := range s.claims {
		if !claimMatchesTab(claim, tab) {
			continue
		}

		row := inboxsalvage.Row{
			Reference:    claim.ClaimNo,
			ClaimNo:      claim.ClaimNo,
			PIC:          claim.PIC,
			BusinessName: claim.BusinessName,
		}

		// Keluarga A menggambar tanggal kejadian; keluarga B menggambar nama objek.
		// Keduanya tidak pernah digambar bersamaan, dan pembedaannya ada di kolom tab.
		if tab.Family == inboxsalvage.FamilyClaim {
			row.LossDate = claim.LossDate
		} else {
			row.ObjectName = claim.ObjectName
		}

		rows = append(rows, row)
	}

	// Urutan mengikuti nomor klaim. Kueri keluarga A dan B sistem lama TIDAK menyebut
	// `ORDER BY` sama sekali, sehingga urutannya tidak ditentukan di sana; urutan yang
	// pasti ditambahkan di sini supaya halaman kedua tidak pernah mengulang baris halaman
	// pertama. Lihat PlannedDifferences.
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ClaimNo < rows[j].ClaimNo
	})
	return rows
}

// claimMatchesTab menyatakan sebuah klaim termasuk daftar tab ini.
func claimMatchesTab(claim Claim, tab inboxsalvage.Tab) bool {
	if tab.Family == inboxsalvage.FamilyClaim {
		// Hanya daftar Salvage Outstanding yang menyaring status kerja.
		for _, closed := range inboxsalvage.ClosedWorkStatuses {
			if claim.WorkStatus == closed {
				return false
			}
		}
	}

	switch {
	case tab.BuybackFilter:
		return claim.HasBuybackValue
	case tab.SalvageStatusIsNull:
		return strings.TrimSpace(claim.SalvageStatus) == ""
	case tab.SalvageStatus != "":
		return claim.SalvageStatus == tab.SalvageStatus
	default:
		return true
	}
}

// salvageRows menyusun baris keluarga C.
func (s *Store) salvageRows(query inboxsalvage.Query) []inboxsalvage.Row {
	tab := query.Tab
	today := s.now()

	rows := []inboxsalvage.Row{}
	for _, item := range s.salvages {
		if tab.TransferStatus != "" && item.TransferStatus != tab.TransferStatus {
			continue
		}
		if tab.OwnedByCaller && !strings.EqualFold(item.PIC, query.Caller.Login) {
			continue
		}

		rows = append(rows, inboxsalvage.Row{
			Reference:       item.SalvageID,
			ClaimNo:         item.ClaimNo,
			SalvageID:       item.SalvageID,
			InputDate:       item.InputDate,
			PIC:             item.PIC,
			SalvageType:     item.SalvageType,
			SalvageLocation: item.Location,
			AuctionStatus:   inboxsalvage.AuctionStatusOf(item.AcceptedValue),
			Quantity:        item.Quantity,
			EstimateValue:   item.EstimateValue,
			Email:           item.Email,
			Remark:          item.Remark,
			AcceptanceNo:    item.AcceptanceNo,
			TransferStatus:  item.TransferStatus,
			RequestValue:    item.RequestValue,
			RequestNote:     item.RequestNote,
			SubmissionType:  inboxsalvage.SubmissionTypeOf(item.RequestNote),
			Aging:           inboxsalvage.AgingOf(item.InputDate, today),

			// Kolom "Catatan" SELALU kosong — lihat inboxsalvage.Row.Note.
			Note: "",
		})
	}

	// `order by a.tglinput desc` pada `GcnmSalvageData_CloseOs_SQL`.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].InputDate != rows[j].InputDate {
			return rows[i].InputDate > rows[j].InputDate
		}
		return rows[i].SalvageID < rows[j].SalvageID
	})
	return rows
}

// Counts menyusun tabel ringkas "Status Salvage / Jumlah".
func (s *Store) Counts(
	ctx context.Context,
	caller inboxsalvage.Caller,
) ([]inboxsalvage.StatusCount, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make([]inboxsalvage.StatusCount, 0, len(inboxsalvage.CountRows()))
	for _, row := range inboxsalvage.CountRows() {
		counts = append(counts, inboxsalvage.StatusCount{
			Label: row.Label,
			Tab:   row.Tab,
			Total: s.countFor(row, caller),
		})
	}
	return counts, nil
}

// countFor menghitung satu baris pencacah.
//
// Ia sengaja TIDAK memakai penyaring tab yang bersangkutan: dua baris pencacah memang
// menghitung populasi yang berbeda dari daftar yang dibukanya, dan itu direplikasi
// (`P-5`). Lihat inboxsalvage.CountRow.
func (s *Store) countFor(row inboxsalvage.CountRow, caller inboxsalvage.Caller) int {
	total := 0

	switch row.Source {
	case inboxsalvage.CountFromSalvage:
		for _, item := range s.salvages {
			if !containsValue(row.TransferStatuses, item.TransferStatus) {
				continue
			}
			if row.OwnedByCaller && !strings.EqualFold(item.PIC, caller.Login) {
				continue
			}
			total++
		}

	case inboxsalvage.CountFromClaimBuyback:
		for _, claim := range s.claims {
			if claim.HasBuybackValue {
				total++
			}
		}

	default:
		for _, claim := range s.claims {
			if row.SalvageStatusIsNull {
				if strings.TrimSpace(claim.SalvageStatus) == "" {
					total++
				}
				continue
			}
			if containsValue(row.SalvageStatuses, claim.SalvageStatus) {
				total++
			}
		}
	}

	return total
}

func containsValue(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

// Detail mengembalikan isi panel "Detail Salvage" untuk satu pengajuan.
func (s *Store) Detail(
	ctx context.Context,
	salvageID string,
) (inboxsalvage.Detail, error) {
	if err := ctx.Err(); err != nil {
		return inboxsalvage.Detail{}, err
	}

	clean := strings.TrimSpace(salvageID)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.salvages {
		if item.SalvageID != clean {
			continue
		}

		detail := inboxsalvage.Detail{
			HasSubmission:  true,
			SalvageID:      item.SalvageID,
			ClaimNo:        item.ClaimNo,
			BusinessName:   s.businessNameOf(item.ClaimNo),
			InputDate:      item.InputDate,
			SalvageType:    item.SalvageType,
			Quantity:       item.Quantity,
			EstimateValue:  item.EstimateValue,
			Location:       item.Location,
			TransferStatus: item.TransferStatus,
			Position:       inboxsalvage.PositionLabelOf(item.TransferStatus),
			AcceptanceNo:   item.AcceptanceNo,
			Remark:         item.Remark,
			AcceptedValue:  item.AcceptedValue,
			Email:          item.Email,
			Items:          []inboxsalvage.DetailBarang{},
		}

		for _, barang := range s.items[item.SalvageID] {
			detail.Items = append(detail.Items, barang)
		}

		// historyOfLocked, BUKAN historyOf: kunci baca sudah dipegang di atas, dan
		// `sync.RWMutex` melarang penguncian baca bertingkat — ia dapat mengunci mati
		// bila ada penulis yang menunggu di antara keduanya.
		detail.History = s.historyOfLocked(item.ClaimNo)
		return detail, nil
	}

	return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
}

// DetailByClaim membaca rincian lewat nomor klaim.
//
// Menirukan urutan repo SQL: klaim dibaca lebih dulu, lalu pengajuan TERAKHIR miliknya
// dicari. Klaim tanpa pengajuan menghasilkan panel ber-HasSubmission salah — bukan galat.
func (s *Store) DetailByClaim(
	ctx context.Context,
	claimNo string,
) (inboxsalvage.Detail, error) {
	if err := ctx.Err(); err != nil {
		return inboxsalvage.Detail{}, err
	}

	clean := strings.TrimSpace(claimNo)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	s.mu.RLock()
	base := inboxsalvage.Detail{Items: []inboxsalvage.DetailBarang{}}
	found := false
	for _, claim := range s.claims {
		if claim.ClaimNo != clean {
			continue
		}
		found = true
		base.ClaimNo = claim.ClaimNo
		base.PIC = claim.PIC
		base.BusinessName = claim.BusinessName
		base.LossDate = claim.LossDate
		break
	}

	base.History = s.historyOfLocked(clean)

	// Pengajuan terakhir = ID tertinggi, sama seperti `MAX(IDSALVAGE)` di Oracle.
	latest := ""
	for _, item := range s.salvages {
		if item.ClaimNo != clean {
			continue
		}
		if latest == "" || lessID(latest, item.SalvageID) {
			latest = item.SalvageID
		}
	}
	s.mu.RUnlock()

	if !found {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}
	if latest == "" {
		return base, nil
	}

	detail, err := s.Detail(ctx, latest)
	if err != nil {
		if errors.Is(err, inboxsalvage.ErrRowNotFound) {
			return base, nil
		}
		return inboxsalvage.Detail{}, err
	}

	detail.PIC = base.PIC
	detail.LossDate = base.LossDate
	if detail.BusinessName == "" {
		detail.BusinessName = base.BusinessName
	}
	detail.History = base.History

	return detail, nil
}

// historyOfLocked menyusun grid riwayat satu klaim; pemanggil sudah memegang kunci.
//
// Urutannya TERBARU LEBIH DULU, sama seperti `ORDER BY TGLINPUT DESC, IDSALVAGE DESC` di
// Oracle — bukan urutan penyimpanan. Grid ini dibaca dari atas, dan yang dicari orang
// adalah pengajuan yang paling baru.
func (s *Store) historyOfLocked(claimNo string) []inboxsalvage.HistoryRow {
	history := []inboxsalvage.HistoryRow{}
	for _, item := range s.salvages {
		if item.ClaimNo != claimNo {
			continue
		}
		history = append(history, inboxsalvage.HistoryRow{
			SalvageID:    item.SalvageID,
			InputDate:    item.InputDate,
			ClaimNo:      item.ClaimNo,
			PIC:          item.PIC,
			MinimumValue: item.EstimateValue,
			Position: inboxsalvage.HistoryPositionOf(
				item.TransferStatus, strings.TrimSpace(item.AcceptanceNo) != ""),
		})
	}

	sort.SliceStable(history, func(a, b int) bool {
		if history[a].InputDate != history[b].InputDate {
			return history[a].InputDate > history[b].InputDate
		}
		return lessID(history[b].SalvageID, history[a].SalvageID)
	})

	return history
}

// lessID membandingkan dua ID pengajuan sebagai ANGKA bila keduanya angka.
//
// ID salvage diterbitkan sebagai bilangan yang bertambah, sehingga perbandingan teks akan
// menempatkan "9" di atas "10" — persis kekeliruan yang dijaga `MAX(IDSALVAGE)` di Oracle
// karena kolomnya numerik di sana.
func lessID(a, b string) bool {
	na, errA := strconv.Atoi(strings.TrimSpace(a))
	nb, errB := strconv.Atoi(strings.TrimSpace(b))
	if errA == nil && errB == nil {
		return na < nb
	}
	return a < b
}

// businessNameOf mencari lini bisnis klaim sebuah pengajuan.
//
// Di Oracle ia sub-kueri ke `T_CLAIM_PNC`; di sini pencarian biasa. Klaim yang tidak ada
// menghasilkan teks kosong — sama seperti sub-kueri yang tidak menemukan baris.
func (s *Store) businessNameOf(claimNo string) string {
	for _, claim := range s.claims {
		if claim.ClaimNo == claimNo {
			return claim.BusinessName
		}
	}
	return ""
}

// Create menyimpan satu pengajuan dan mengembalikan ID salvage yang terbit.
func (s *Store) Create(ctx context.Context, form inboxsalvage.Form) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if form.Mode == inboxsalvage.FormModeUpdate {
		for index, item := range s.salvages {
			if item.SalvageID != form.SalvageID {
				continue
			}
			s.salvages[index] = applyForm(item, form)

			// ID salvage TIDAK ditimpa kosong. Procedure lama menuliskan `IDSALVAGE`
			// dengan variabel yang tidak pernah diisi pada cabang ini, sehingga kunci
			// barisnya sendiri menjadi kosong — butir 12 daftar perbaikan `P-5`
			// (`D-49` #9).
			s.salvages[index].SalvageID = form.SalvageID
			s.items[form.SalvageID] = append(
				s.items[form.SalvageID], barangOf(form)...)
			return form.SalvageID, nil
		}
		return "", inboxsalvage.ErrRowNotFound
	}

	id := strconv.Itoa(s.nextID)
	s.nextID++

	record := applyForm(Salvage{SalvageID: id}, form)
	s.salvages = append(s.salvages, record)
	s.items[id] = append(s.items[id], barangOf(form)...)
	return id, nil
}

// barangOf menerjemahkan Detail Item Salvage pada form menjadi baris panel detail.
//
// Nilai `STATUSTERJUAL` yang diterima baris baru adalah `NewDetailStatus` di penyimpanan
// SQL; di sini yang disimpan adalah LABELNYA, karena penyimpanan memori tidak menyimpan
// kodenya sama sekali.
func barangOf(form inboxsalvage.Form) []inboxsalvage.DetailBarang {
	barang := make([]inboxsalvage.DetailBarang, 0, len(form.Items))
	for _, item := range form.Items {
		barang = append(barang, inboxsalvage.DetailBarang{
			Name:       item.Name,
			Count:      1,
			Unit:       item.Unit,
			TotalValue: item.Quantity,
			SoldStatus: inboxsalvage.SoldStatusOf("3"),
			Remark:     item.Remarks,
		})
	}
	return barang
}

// applyForm menyalin isian form ke satu baris salvage.
func applyForm(item Salvage, form inboxsalvage.Form) Salvage {
	item.ClaimNo = form.ClaimNo
	item.InputDate = form.InputDate
	item.PIC = form.Caller.Login
	item.SalvageType = form.SalvageType
	item.Location = form.Location
	item.Quantity = form.Quantity
	item.EstimateValue = form.MinimumValue
	item.Email = form.Email
	item.Remark = form.Remark
	item.TransferStatus = form.TransferStatus
	item.AcceptedValue = form.InsuredShare
	return item
}

// todayISO mengembalikan tanggal hari ini dalam bentuk `YYYY-MM-DD`.
//
// Ia sengaja memakai zona waktu yang sama dengan penyimpanan SQL — lihat
// inboxsalvage.AgingOf.
func todayISO() string {
	return inboxsalvage.TodayWIB()
}
