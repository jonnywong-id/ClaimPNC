// Package memory memenuhi seam inboxsurvey.Repo dan inboxsurvey.Directory dengan
// penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaring dan urutannya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Yang ditiru apa adanya:
//
//	cakupan nama surveyor       batas kewenangan, termasuk anggota bagi leader
//	predikat tab yang TERSEDIA  persis seperti count_tabs dan list_tasks
//	urutan TGLINPUT MENAIK      yang tertua lebih dulu — urutan antrean kerja
//	pencarian Claim No          tidak peka huruf besar-kecil
//
// # Empat tab TIDAK ditiru, dan itu disengaja
//
// Tab Outstanding, ALL, Invoice, dan Close bergantung pada kolom yang tidak ada di tabel mana
// pun yang dibaca modul ini (lihat inboxsurvey.UnavailableReason). Menirunya di sini akan
// menghasilkan uji yang LULUS atas perilaku yang di Oracle tidak pernah terjadi — bentuk
// pengujian yang paling menyesatkan, karena ia memberi rasa aman tanpa menjamin apa pun.
//
// Predikat keempatnya tidak hilang: ia terbaca dari konstanta inboxsurvey.AdjusterConfirmed,
// StatusInvoiceFee, dan StatusCloseCase beserta penjelasannya, siap dipakai kembali begitu
// kolomnya tiba.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxsurvey"
)

// Record adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring.
//
// Kolom penyaringnya sengaja tidak masuk `inboxsurvey.SurveyTask`: ia tidak sampai ke layar,
// dan menaruhnya di tipe domain akan membuat orang menduga ia bagian dari kontrak.
type Record struct {
	Task inboxsurvey.SurveyTask

	// Messages adalah isi `POOLDATA.M_KOMUNIKASI_PNC` untuk baris ini.
	Messages []Message
}

// Message adalah satu pesan komunikasi.
type Message struct {
	// Status adalah `KOMUNIKASISTATUS` — "0" terbuka, "1" sudah dijawab.
	Status string

	// Sender adalah `SENDER` — login pengirimnya.
	Sender string
}

// SurveyorRecord adalah satu baris `POOLDATA.MST_LOGIN_SURVEYOR`.
type SurveyorRecord struct {
	Login string
	Name  string

	// LeaderLogin adalah `LOGINLEADER` — login ATASAN, bukan namanya.
	//
	// Kosong berarti orang ini sendiri yang menjadi puncak. Itu bacaan yang sama dengan
	// sqlstore, dan kesamaannya disengaja: kalau berbeda, uji di sini tidak menyatakan apa
	// pun tentang yang berjalan di Oracle.
	LeaderLogin string
}

// Store adalah pembaca antrean survei di memori. Ia memenuhi Repo sekaligus Directory.
type Store struct {
	records   []Record
	surveyors []SurveyorRecord
	kpi       []KPIRecord
}

// KPIRecord adalah satu baris `POOLDATA.DETAIL_KPI_ADJUSTER`.
type KPIRecord struct {
	Row inboxsurvey.KPIRow

	// Adjuster adalah kolom `ADJUSTER` — nama surveyor, yang di ringkasan per adjuster juga
	// menjadi kunci kelompoknya.
	Adjuster string

	// Category adalah kolom `TIPE`.
	Category string

	// Year adalah `to_char(TANGGAL,'yyyy')`.
	Year string
}

// NewStore membentuk pembaca dari baris yang diberikan.
func NewStore(records []Record, surveyors []SurveyorRecord, kpi []KPIRecord) *Store {
	return &Store{records: records, surveyors: surveyors, kpi: kpi}
}

// ResolveSurveyor menerjemahkan login menjadi identitas surveyor beserta cakupannya.
func (s *Store) ResolveSurveyor(
	_ context.Context,
	login string,
) (inboxsurvey.SurveyorIdentity, error) {
	trimmed := strings.TrimSpace(login)
	if trimmed == "" {
		return inboxsurvey.SurveyorIdentity{}, inboxsurvey.ErrCallerUnknown
	}

	var self *SurveyorRecord
	for i := range s.surveyors {
		if strings.EqualFold(strings.TrimSpace(s.surveyors[i].Login), trimmed) {
			self = &s.surveyors[i]
			break
		}
	}
	if self == nil || strings.TrimSpace(self.Name) == "" {
		return inboxsurvey.SurveyorIdentity{}, inboxsurvey.ErrNotSurveyor
	}

	identity := inboxsurvey.SurveyorIdentity{
		Login: self.Login,
		Name:  self.Name,
		Scope: []string{self.Name},
	}

	// Pemanggil adalah leader bila ia sendiri tidak menunjuk leader lain.
	if strings.TrimSpace(self.LeaderLogin) != "" {
		return identity, nil
	}

	seen := map[string]bool{strings.ToUpper(strings.TrimSpace(self.Name)): true}
	for _, candidate := range s.surveyors {
		if !strings.EqualFold(strings.TrimSpace(candidate.LeaderLogin), self.Login) {
			continue
		}

		name := strings.TrimSpace(candidate.Name)
		key := strings.ToUpper(name)
		if name == "" || seen[key] {
			continue
		}

		seen[key] = true
		identity.Scope = append(identity.Scope, name)
		identity.IsLeader = true
	}

	return identity, nil
}

// List mengambil satu halaman satu tab.
func (s *Store) List(
	_ context.Context,
	identity inboxsurvey.SurveyorIdentity,
	f inboxsurvey.Filter,
) (inboxsurvey.Page, error) {
	clean := f.Normalize()

	// Tab yang belum dapat dihitung dicegat di sini, sama seperti sqlstore.Repo.List.
	if !clean.Tab.Available() {
		return inboxsurvey.Page{Tasks: []inboxsurvey.SurveyTask{}}, nil
	}

	matched := []Record{}
	for _, record := range s.records {
		if !inScope(record, identity.Scope) {
			continue
		}
		if !inTab(record, clean.Tab, identity.Login) {
			continue
		}
		if !matchesSearch(record, clean.Search) {
			continue
		}
		matched = append(matched, record)
	}

	// Urutan MENAIK, sama dengan `ORDER BY s.TGLINPUT` pada kueri — yang tertua lebih dulu.
	// Pemutus serinya CASEID lalu INDEX_SURVEY, tanpanya dua janji survei yang diinput pada
	// waktu yang sama berpindah-pindah urutan antar halaman.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].Task.CreatedAt, matched[j].Task.CreatedAt
		if !left.Equal(right) {
			return left.Before(right)
		}
		if matched[i].Task.SurveyID != matched[j].Task.SurveyID {
			return matched[i].Task.SurveyID < matched[j].Task.SurveyID
		}
		return matched[i].Task.SurveyIndex < matched[j].Task.SurveyIndex
	})

	page := inboxsurvey.Page{
		Tasks: []inboxsurvey.SurveyTask{},
		Total: len(matched),
	}

	for i := clean.Offset; i < len(matched) && len(page.Tasks) < clean.Limit; i++ {
		page.Tasks = append(page.Tasks, matched[i].Task)
	}

	return page, nil
}

// Counts menghitung isi tab yang DAPAT dihitung, sekaligus.
//
// Tab yang belum tersedia TIDAK dikembalikan sama sekali — sama seperti sqlstore.Repo.Counts.
// Mengembalikan nol untuknya akan menyatakan "tab ini kosong", padahal yang benar adalah "tab
// ini belum dapat dihitung".
func (s *Store) Counts(
	_ context.Context,
	identity inboxsurvey.SurveyorIdentity,
) ([]inboxsurvey.TabCount, error) {
	tabs := inboxsurvey.Tabs()
	result := make([]inboxsurvey.TabCount, 0, len(tabs))

	for _, tab := range tabs {
		if !tab.Available() {
			continue
		}

		total := 0
		for _, record := range s.records {
			if inScope(record, identity.Scope) && inTab(record, tab, identity.Login) {
				total++
			}
		}
		result = append(result, inboxsurvey.TabCount{Tab: tab, Total: total})
	}

	return result, nil
}

// KPI mengambil ringkasan KPI adjuster.
//
// Ia MENGELOMPOKKAN, sama seperti kuerinya — bukan mengembalikan baris mentah. Tanpa
// pengelompokan, uji yang lulus di sini tidak menyatakan apa pun tentang `GROUP BY` di
// Oracle.
func (s *Store) KPI(
	_ context.Context,
	identity inboxsurvey.SurveyorIdentity,
	f inboxsurvey.KPIFilter,
) ([]inboxsurvey.KPIRow, error) {
	clean := f.Normalize()

	category := clean.Category
	if clean.Kind != inboxsurvey.KPIOutstanding {
		category = inboxsurvey.KPITypeFinal
	}

	sums := map[string]*inboxsurvey.KPIRow{}
	counts := map[string]int{}
	order := []string{}

	for _, entry := range s.kpi {
		if !nameInScope(entry.Adjuster, identity.Scope) {
			continue
		}
		if category != "" && !strings.EqualFold(strings.TrimSpace(entry.Category), category) {
			continue
		}
		if clean.Year != "" && entry.Year != clean.Year {
			continue
		}

		key := entry.Adjuster
		if clean.Kind == inboxsurvey.KPIQuarterly {
			key = entry.Year
		}

		if _, exists := sums[key]; !exists {
			sums[key] = &inboxsurvey.KPIRow{Group: key}
			order = append(order, key)
		}

		accumulate(sums[key], entry.Row)
		counts[key]++
	}

	sort.Strings(order)
	if clean.Kind == inboxsurvey.KPIQuarterly {
		// Ringkasan per tahun diurutkan MENURUN, sama dengan kuerinya — tahun terbaru lebih
		// dulu.
		sort.Sort(sort.Reverse(sort.StringSlice(order)))
	}

	result := make([]inboxsurvey.KPIRow, 0, len(order))
	for _, key := range order {
		result = append(result, average(*sums[key], counts[key]))
	}
	return result, nil
}

// inScope menyatakan baris ini termasuk cakupan kewenangan pemanggil.
func inScope(record Record, scope []string) bool {
	return nameInScope(record.Task.AdjusterPIC, scope)
}

// nameInScope mencocokkan satu nama ke cakupan, tidak peka huruf besar-kecil.
//
// Pencocokannya PENUH, bukan sebagian — sama dengan pembatas `|` pada sisi SQL. Pencocokan
// sebagian akan membuat "BUDI" melihat pekerjaan "BUDIONO".
func nameInScope(name string, scope []string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	for _, allowed := range scope {
		if strings.EqualFold(strings.TrimSpace(allowed), trimmed) {
			return true
		}
	}
	return false
}

// inTab menyatakan baris ini termasuk keranjang tab tertentu.
//
// Hanya tab TERSEDIA yang punya cabang, ditiru dari `count_tabs` dan `list_tasks` keranjang
// demi keranjang. Tab lain jatuh ke `default` dan mengembalikan false — bukan karena terlupa,
// melainkan karena kolom penggeraknya tidak ada; lihat kepala paket.
func inTab(record Record, tab inboxsurvey.Tab, login string) bool {
	switch tab {
	case inboxsurvey.TabNotAnswered:
		return hasMessage(record, inboxsurvey.CommunicationOpen, login, false)

	case inboxsurvey.TabNotReplied:
		return hasMessage(record, inboxsurvey.CommunicationOpen, login, true)

	case inboxsurvey.TabReplied:
		return hasMessage(record, inboxsurvey.CommunicationAnswered, login, true)

	default:
		return false
	}
}

// hasMessage mencari satu pesan berstatus tertentu.
//
// `fromCaller` membalik arah perbandingan pengirim: true berarti pesan dari pemanggil
// sendiri, false berarti dari pihak lain. Itu persis pembedaan `= :3` versus `<> :3` pada
// kueri, dan membalikkannya menukar tab "belum dijawab" dengan "belum dibalas ASM" — dua
// layar yang sama-sama masuk akal, sehingga tertukarnya tidak akan disadari.
func hasMessage(record Record, status, login string, fromCaller bool) bool {
	for _, message := range record.Messages {
		if strings.TrimSpace(message.Status) != status {
			continue
		}
		sameSender := strings.EqualFold(
			strings.TrimSpace(message.Sender),
			strings.TrimSpace(login),
		)
		if sameSender == fromCaller {
			return true
		}
	}
	return false
}

// matchesSearch mencocokkan kata kunci ke Claim No.
//
// HANYA Claim No, sama dengan kuerinya. Di Pega ia mencari pada dua kolom — yang kedua
// `REFNO_1`, dan kolom itu belum tersedia. Menirunya di sini akan menghasilkan uji yang lulus
// atas pencarian yang di Oracle tidak pernah terjadi.
func matchesSearch(record Record, search string) bool {
	if search == "" {
		return true
	}
	needle := strings.ToUpper(search)

	return strings.Contains(strings.ToUpper(record.Task.ClaimNumber), needle)
}

// accumulate menjumlahkan kesembilan angka KPI.
func accumulate(target *inboxsurvey.KPIRow, row inboxsurvey.KPIRow) {
	target.SurveyScheduling += row.SurveyScheduling
	target.ImmediateAdvice += row.ImmediateAdvice
	target.PreliminaryAdvice += row.PreliminaryAdvice
	target.InterimReport += row.InterimReport
	target.ProgressUpdate += row.ProgressUpdate
	target.CommunicationResponse += row.CommunicationResponse
	target.ProposeAdjustment += row.ProposeAdjustment
	target.FinalReport += row.FinalReport
	target.Value += row.Value
}

// average membagi kesembilan jumlah dengan banyaknya baris, lalu membulatkan dua desimal —
// sama dengan `round(avg(...),2)` pada kuerinya.
func average(row inboxsurvey.KPIRow, count int) inboxsurvey.KPIRow {
	if count <= 0 {
		return row
	}

	divide := func(value float64) float64 {
		return round2(value / float64(count))
	}

	row.SurveyScheduling = divide(row.SurveyScheduling)
	row.ImmediateAdvice = divide(row.ImmediateAdvice)
	row.PreliminaryAdvice = divide(row.PreliminaryAdvice)
	row.InterimReport = divide(row.InterimReport)
	row.ProgressUpdate = divide(row.ProgressUpdate)
	row.CommunicationResponse = divide(row.CommunicationResponse)
	row.ProposeAdjustment = divide(row.ProposeAdjustment)
	row.FinalReport = divide(row.FinalReport)
	row.Value = divide(row.Value)
	return row
}

// round2 membulatkan ke dua desimal.
//
// Ditulis tanpa math.Round atas nilai negatif karena skor KPI tidak pernah negatif; bila
// kelak bisa, pembulatannya harus ditinjau ulang bersama kuerinya.
func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
