// Package memory memenuhi seam inboxautoclaim.Repo di dalam memori proses.
//
// Ia ada untuk dua keperluan yang keduanya nyata:
//
//   - PENGUJIAN. Aturan modul dapat diuji tanpa basis data dan tanpa jaringan, sehingga
//     uji berjalan di setiap mesin dan di CI tanpa prasyarat apa pun.
//   - PENGEMBANGAN. Layarnya dapat dicoba utuh sebelum DBA menyediakan akses ke
//     POOLDATA.TMP_BATCH_AUTO_CLAIM.
//
// Ia MENIRU perilaku sqlstore, bukan menyederhanakannya: paginasi, penyaringan,
// pengelompokan per tanggal proses, dan penurunan nomor batch bekerja dengan aturan yang
// sama. Fake yang lebih longgar dari aslinya adalah fake yang meloloskan cacat.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxautoclaim"
)

// Repo menyimpan baris Inbox Auto Claim di memori.
type Repo struct {
	lock sync.RWMutex

	// line adalah seluruh baris, tersimpan datar persis seperti satu tabel.
	//
	// Bentuk datar dipilih dengan sengaja walau pengelompokan per batch akan lebih cepat:
	// yang ditiru adalah TABEL, dan agregasinya harus dihitung dari baris seperti yang
	// dilakukan SQL. Menyimpan hitungan yang sudah jadi akan membuat fake ini menyetujui
	// hasil yang belum tentu disetujui basis data.
	// Kuncinya TAB. Ketiga tab membaca TABEL yang berbeda di Oracle
	// (TMP_BATCH_AUTO_CLAIM, TMP_BATCH_CLAIM_KREDIT, TMP_BATCH_AUTO_TRAVEL), jadi fake
	// yang menyimpannya dalam satu senarai akan meloloskan kebocoran antartab yang
	// mustahil terjadi di basis data.
	line map[inboxautoclaim.Source][]inboxautoclaim.Line

	// master adalah isi POOLDATA.M_AUTO_CLAIM_PNC: kode perusahaan -> namanya.
	master map[string]string

	// policy meniru pencarian polis: nomor polis -> kode perusahaan dan prodke-nya.
	//
	// Di basis data keduanya berasal dari dua kueri berbeda (T_GENERAL dan JSON_POLIS);
	// di sini keduanya disatukan karena yang ditiru adalah HASILNYA, bukan jalannya.
	policy map[string]PolicyRow

	// now memberi tanggal proses pada baris yang baru disisipkan. Ia dapat diganti uji
	// supaya tanggalnya tetap dan hasilnya dapat diperiksa.
	now func() string
}

// PolicyRow adalah hasil pencarian polis yang ditiru penyimpanan memori.
type PolicyRow struct {
	// CompanyCode kosong berarti polisnya ada tetapi tidak menunjuk perusahaan rekanan
	// mana pun yang aktif — itu MessageReceiverNotFound.
	CompanyCode string

	// ProductSeq kosong berarti polisnya tidak ditemukan di JSON_POLIS.
	ProductSeq string
}

// NewRepo membentuk penyimpanan memori beserta isi awalnya.
func NewRepo(master map[string]string, policy map[string]PolicyRow, line ...inboxautoclaim.Line) *Repo {
	masterCopy := make(map[string]string, len(master))
	for code, name := range master {
		masterCopy[strings.TrimSpace(code)] = name
	}
	policyCopy := make(map[string]PolicyRow, len(policy))
	for number, row := range policy {
		policyCopy[strings.ToUpper(strings.TrimSpace(number))] = row
	}

	// Baris yang diberikan masuk ke tab ANEKA, DISEBUT NAMANYA — bukan ke DefaultSource.
	//
	// Semula ia memakai DefaultSource, dan ketika tab bawaan dipindahkan ke Asuransi
	// Kredit seluruh data contoh ANEKA ikut berpindah ke sana: tab ANEKA menjadi kosong
	// tanpa satu baris kode pun berubah di modulnya. Tab bawaan adalah keputusan
	// TAMPILAN; ia tidak boleh menentukan isi tabel mana yang dibaca.
	//
	// Pemanggil yang memaksudkan tab lain memakai Seed.
	return &Repo{
		line:   map[inboxautoclaim.Source][]inboxautoclaim.Line{inboxautoclaim.SourceAneka: append([]inboxautoclaim.Line(nil), line...)},
		master: masterCopy,
		policy: policyCopy,
		now:    func() string { return "19/09/2026" },
	}
}

// Seed mengisi baris contoh untuk satu tab.
//
// Dipakai penyusun data contoh dan uji yang perlu isi berbeda per tab; NewRepo sendiri
// hanya mengisi tab bawaan.
func (r *Repo) Seed(source inboxautoclaim.Source, line ...inboxautoclaim.Line) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.line[source] = append(r.line[source], line...)
}

// SetProcessedDate mengganti tanggal proses yang diberikan pada baris baru.
//
// Dipakai uji yang memeriksa pengelompokan per tanggal: tanpa ini, seluruh baris yang
// disisipkan satu uji akan selalu bertanggal sama dan pemisahan dua barisnya tidak pernah
// teruji.
func (r *Repo) SetProcessedDate(date string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.now = func() string { return date }
}

// ListBatch mengelompokkan baris menjadi batch lalu memotongnya sesuai halaman.
func (r *Repo) ListBatch(_ context.Context, filter inboxautoclaim.BatchFilter) (inboxautoclaim.BatchPage, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	company := strings.TrimSpace(filter.CompanyCode)
	all := r.groupBatch(filter.Source, company)

	page := filter.Page.Clean()
	offset := filter.Page.Offset()
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + page.Size
	if end > len(all) {
		end = len(all)
	}

	// Potongan disalin, tidak dikembalikan apa adanya: senarai yang dikembalikan tidak
	// boleh menjadi jendela ke penyimpanan.
	item := append([]inboxautoclaim.Batch(nil), all[offset:end]...)
	return inboxautoclaim.BatchPage{Item: item, Total: len(all)}, nil
}

// ListCompany mengembalikan SELURUH perusahaan di master, terurut menurut namanya.
//
// Sumbernya master, bukan tabel batch — mengikuti BrowseCompanyClaimCredit.
// SummarizeCompany menghitung jumlah batch per perusahaan.
//
// Ia memakai groupBatch yang sama dengan ListBatch, tanpa penyaring. Itu bukan sekadar
// hemat kode: dengan sumber yang sama, angka ringkasan dan jumlah baris grid TIDAK DAPAT
// berselisih di sini — dan bila kelak keduanya berselisih pada Oracle, penyebabnya pasti
// di kueri SQL, bukan di pemahaman kita tentang apa yang dihitung.
func (r *Repo) SummarizeCompany(_ context.Context, source inboxautoclaim.Source) (inboxautoclaim.Summary, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	count := map[string]int{}
	name := map[string]string{}

	// HANYA perusahaan yang punya batch DI TAB INI, sama dengan kueri SQL-nya.
	//
	// Versi sebelumnya menyemai seluruh perusahaan master berjumlah 0 lebih dulu.
	// Akibatnya ketiga tab menampilkan daftar perusahaan yang sama persis — masternya
	// memang satu — dan panel yang seharusnya menjadi penyaring justru penuh baris yang
	// bila diklik menghasilkan grid kosong. Itu yang dilaporkan Work Owner sebagai
	// "penyaringnya tidak berfungsi" pada 2026-09-20.
	//
	// Perusahaan yang kodenya tidak ada di master TETAP ikut, dengan nama kosong —
	// baris seperti itu justru yang tidak akan pernah berhasil diproses.
	for _, batch := range r.groupBatch(source, "") {
		count[batch.CompanyCode]++
		name[batch.CompanyCode] = batch.CompanyName
	}

	summary := inboxautoclaim.Summary{
		Company: make([]inboxautoclaim.CompanySummary, 0, len(count)),
	}
	for code, jumlah := range count {
		summary.Company = append(summary.Company, inboxautoclaim.CompanySummary{
			Code:       code,
			Name:       name[code],
			BatchCount: jumlah,
		})
		summary.Total += jumlah
	}

	// Menurun berdasarkan jumlah, lalu kode — sama dengan ORDER BY kueri SQL-nya.
	// Urutan yang berbeda antara kedua penyimpanan membuat uji layar lulus pada susunan
	// yang tidak pernah terjadi di produksi.
	sort.Slice(summary.Company, func(a, b int) bool {
		if summary.Company[a].BatchCount != summary.Company[b].BatchCount {
			return summary.Company[a].BatchCount > summary.Company[b].BatchCount
		}
		return summary.Company[a].Code < summary.Company[b].Code
	})
	return summary, nil
}

func (r *Repo) ListCompany(_ context.Context) ([]inboxautoclaim.Company, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	result := make([]inboxautoclaim.Company, 0, len(r.master))
	for code, name := range r.master {
		result = append(result, inboxautoclaim.Company{Code: code, Name: name})
	}
	sort.Slice(result, func(a, b int) bool {
		if result[a].Name != result[b].Name {
			return result[a].Name < result[b].Name
		}
		return result[a].Code < result[b].Code
	})
	return result, nil
}

// ListLine memotong rincian satu batch sesuai halaman.
func (r *Repo) ListLine(_ context.Context, q inboxautoclaim.LineQuery) (inboxautoclaim.LinePage, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	match := r.matchLine(q)

	page := q.Page.Clean()
	offset := q.Page.Offset()
	if offset > len(match) {
		offset = len(match)
	}
	end := offset + page.Size
	if end > len(match) {
		end = len(match)
	}
	return inboxautoclaim.LinePage{
		Item:  append([]inboxautoclaim.Line(nil), match[offset:end]...),
		Total: len(match),
	}, nil
}

// ExportLine mengembalikan seluruh baris yang cocok.
func (r *Repo) ExportLine(_ context.Context, q inboxautoclaim.LineQuery) ([]inboxautoclaim.Line, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.matchLine(q), nil
}

// BatchExists menyatakan pasangan (perusahaan, batch) ada.
func (r *Repo) BatchExists(_ context.Context, source inboxautoclaim.Source, companyCode, batchNumber string) (bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	code := strings.TrimSpace(companyCode)
	number := strings.TrimSpace(batchNumber)
	for _, line := range r.line[source] {
		if strings.EqualFold(strings.TrimSpace(line.CompanyCode), code) &&
			strings.TrimSpace(line.BatchNumber) == number {
			return true, nil
		}
	}
	return false, nil
}

// ResolveReceiver menurunkan perusahaan rekanan dari nomor polis.
func (r *Repo) ResolveReceiver(_ context.Context, policyNo string) (inboxautoclaim.Company, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	row, exists := r.policy[strings.ToUpper(strings.TrimSpace(policyNo))]
	if !exists || row.CompanyCode == "" {
		return inboxautoclaim.Company{}, false, nil
	}
	code := strings.TrimSpace(row.CompanyCode)
	return inboxautoclaim.Company{Code: code, Name: r.master[code]}, true, nil
}

// FindPolicyProductSeq mencari PRODKE termutakhir sebuah polis.
func (r *Repo) FindPolicyProductSeq(_ context.Context, policyNo string) (string, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	row, exists := r.policy[strings.ToUpper(strings.TrimSpace(policyNo))]
	if !exists || row.ProductSeq == "" {
		return "", false, nil
	}
	return row.ProductSeq, true, nil
}

// InsertUpload menambahkan baris unggahan beserta nomor batch barunya.
func (r *Repo) InsertUpload(
	_ context.Context,
	source inboxautoclaim.Source,
	line []inboxautoclaim.UploadLine,
	uploadedBy string,
) (inboxautoclaim.UploadResult, error) {
	if len(line) == 0 {
		return inboxautoclaim.UploadResult{}, inboxautoclaim.ErrEmptyUpload
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	order, group := inboxautoclaim.GroupUploadLine(line)
	processedDate := r.now()

	result := inboxautoclaim.UploadResult{}
	for _, companyCode := range order {
		batchNumber := inboxautoclaim.NextBatchNumber(r.usedBatchNumber(source, companyCode))

		reference := inboxautoclaim.BatchRef{
			CompanyCode: companyCode,
			CompanyName: r.master[companyCode],
			BatchNumber: batchNumber,
		}

		for _, l := range group[companyCode] {
			// Ketiga kolom penanda menerima nilai yang SAMA — pesan galat bila gagal,
			// kosong bila lolos. Yang kosong itulah yang membuat barisnya terambil
			// pemrosesan.
			r.line[source] = append(r.line[source], inboxautoclaim.Line{
				CompanyCode:   companyCode,
				BatchNumber:   batchNumber,
				PolicyNo:      l.Row.PolicyNo,
				ProductSeq:    l.ProductSeq,
				ClaimID:       l.Message,
				AcceptanceNo:  l.Message,
				Message:       l.Message,
				ClaimAmount:   l.Row.ClaimAmount,
				CauseOfLoss:   l.Row.CauseOfLoss,
				DateOfLoss:    l.Row.DateOfLoss,
				ReportDate:    l.Row.ReportDate,
				ProcessedDate: processedDate,
				Note:          l.Row.Reason,
				Keyword:       l.Row.Keyword,
				ObjectName:    l.Row.ObjectName,
				FlagNoPayout:  l.Row.FlagNoPayout,
				UploadedBy:    uploadedBy,
				// Currency sengaja kosong: Pega mengambilnya dari snapshot polis, dan
				// modul ini belum dapat membacanya (menunggu B-1).
			})

			reference.Rows++
			if l.Accepted() {
				reference.Succeeded++
			} else {
				reference.Failed++
			}
		}

		result.Batch = append(result.Batch, reference)
		result.Rows += reference.Rows
	}
	return result, nil
}

// usedBatchNumber mengumpulkan nomor batch yang sudah dipakai satu perusahaan.
func (r *Repo) usedBatchNumber(source inboxautoclaim.Source, companyCode string) []string {
	var used []string
	for _, line := range r.line[source] {
		if strings.EqualFold(strings.TrimSpace(line.CompanyCode), companyCode) {
			used = append(used, strings.TrimSpace(line.BatchNumber))
		}
	}
	return used
}

// matchLine menyaring baris satu batch sesuai penyaring hasil, lalu mengurutkannya.
func (r *Repo) matchLine(q inboxautoclaim.LineQuery) []inboxautoclaim.Line {
	source := q.Source
	code := strings.TrimSpace(q.CompanyCode)
	number := strings.TrimSpace(q.BatchNumber)

	var match []inboxautoclaim.Line
	for _, line := range r.line[source] {
		if !strings.EqualFold(strings.TrimSpace(line.CompanyCode), code) {
			continue
		}
		if strings.TrimSpace(line.BatchNumber) != number {
			continue
		}
		switch q.Result {
		case inboxautoclaim.ResultSucceeded:
			if !line.Succeeded() {
				continue
			}
		case inboxautoclaim.ResultFailed:
			if !line.Processed() || line.Succeeded() {
				continue
			}
		}
		match = append(match, line)
	}

	// Urutannya sama dengan kueri SQL: NOPOLIS, TGLPROSES, TGLKEJADIAN — kunci primer
	// tabelnya, sehingga urutannya pasti.
	sort.SliceStable(match, func(a, b int) bool {
		if match[a].PolicyNo != match[b].PolicyNo {
			return match[a].PolicyNo < match[b].PolicyNo
		}
		if match[a].ProcessedDate != match[b].ProcessedDate {
			return match[a].ProcessedDate < match[b].ProcessedDate
		}
		return match[a].DateOfLoss < match[b].DateOfLoss
	})
	return match
}

// groupBatch menghitung agregat per (perusahaan, batch, tanggal proses).
//
// Tanggal proses ikut menjadi kunci, mengikuti GROUP BY kueri aslinya. Hitungannya sendiri
// TIDAK disaring tanggal — subkueri pada kueri asli hanya menyaring (BATCH, INISIALID) —
// sehingga dua baris tanggal dari satu batch membawa keempat angka yang sama.
func (r *Repo) groupBatch(source inboxautoclaim.Source, companyCode string) []inboxautoclaim.Batch {
	type total struct{ uploaded, processed, succeeded, failed int }

	// Tahap satu: hitung agregat per (perusahaan, batch), tanpa tanggal.
	count := map[string]total{}
	for _, line := range r.line[source] {
		key := strings.TrimSpace(line.CompanyCode) + "\x00" + strings.TrimSpace(line.BatchNumber)
		t := count[key]
		t.uploaded++
		if line.Processed() {
			t.processed++
			if line.Succeeded() {
				t.succeeded++
			} else {
				t.failed++
			}
		}
		count[key] = t
	}

	// Tahap dua: susun baris grid per (perusahaan, batch, pengunggah, tanggal proses).
	index := map[string]int{}
	var result []inboxautoclaim.Batch
	for _, line := range r.line[source] {
		code := strings.TrimSpace(line.CompanyCode)
		if companyCode != "" && code != companyCode {
			continue
		}
		number := strings.TrimSpace(line.BatchNumber)
		rowKey := code + "\x00" + number + "\x00" + line.UploadedBy + "\x00" + line.ProcessedDate
		if _, exists := index[rowKey]; exists {
			continue
		}

		t := count[code+"\x00"+number]
		result = append(result, inboxautoclaim.Batch{
			CompanyCode:   code,
			CompanyName:   r.master[code],
			BatchNumber:   number,
			ProcessedDate: line.ProcessedDate,
			Uploaded:      t.uploaded,
			Processed:     t.processed,
			Succeeded:     t.succeeded,
			Failed:        t.failed,
			UploadedBy:    line.UploadedBy,
		})
		index[rowKey] = len(result) - 1
	}

	// ORDER BY BATCH DESC — batch terbaru di atas, mengikuti kueri aslinya. Nomornya
	// dibandingkan sebagai ANGKA karena kolomnya memang NUMBER (DDL 2026-09-19).
	sort.SliceStable(result, func(a, b int) bool {
		left, leftOK := asNumber(result[a].BatchNumber)
		right, rightOK := asNumber(result[b].BatchNumber)
		if leftOK && rightOK && left != right {
			return left > right
		}
		if result[a].BatchNumber != result[b].BatchNumber {
			return result[a].BatchNumber > result[b].BatchNumber
		}
		return result[a].CompanyCode < result[b].CompanyCode
	})
	return result
}

func asNumber(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	result := 0
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, false
		}
		result = result*10 + int(character-'0')
	}
	return result, true
}

var _ inboxautoclaim.Repo = (*Repo)(nil)
