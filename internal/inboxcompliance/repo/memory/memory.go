// Package memory memenuhi seam inboxcompliance.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaringnya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Penyaring workbasket dan urutan barisnya ditiru sedekat-dekatnya dengan
// predikat SQL-nya — termasuk urutan `PXCREATEDATETIME DESC, PYID DESC` yang menentukan
// baris mana masuk halaman pertama.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring atau
// mengurutkan.
//
// Kedua kolom tambahan itu tidak masuk inboxcompliance.WorkItem dengan sengaja: keduanya
// tidak pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga
// ia bagian dari kontrak.
type Row struct {
	Item inboxcompliance.WorkItem

	// Tab adalah kode tab tempat baris ini muncul.
	//
	// Ia ada karena kedua tab membaca TABEL YANG BERBEDA, bukan satu tabel dengan
	// penyaring berbeda — dan penyimpanan di memori harus meniru pemisahan itu. Tanpanya,
	// baris Post Audit akan bocor ke tab Compliance saat pengembangan lokal, lalu
	// perbedaannya baru ketahuan di Oracle.
	//
	// Kosong berarti tab Compliance, supaya baris contoh yang sudah ada tidak perlu
	// diubah.
	Tab string

	// Workbasket adalah antrean tempat baris ini menunggu —
	// `PC_ASSIGN_WORKBASKET.PXASSIGNEDOPERATORID`. Hanya berlaku pada tab Compliance;
	// tab Post Audit tidak membaca workbasket sama sekali.
	Workbasket string

	// CreatedAt — `PXCREATEDATETIME`, kunci pengurutan utama.
	CreatedAt time.Time

	// Resolved menandai klaimnya sudah selesai (`PYSTATUSWORK = 'Resolved-Completed'`),
	// sehingga barisnya HILANG dari antrean.
	//
	// Baris seperti ini sengaja ada di data contoh: tanpanya, penyaring yang lupa
	// dipasang tidak akan pernah ketahuan saat pengembangan lokal.
	Resolved bool
}

// tab mengembalikan kode tab baris ini, dengan tab Compliance sebagai bawaan.
func (r Row) tab() string {
	if r.Tab == "" {
		return inboxcompliance.TabCompliance
	}
	return r.Tab
}

// sortKey mengembalikan waktu yang dipakai mengurutkan baris ini pada sebuah tab.
//
// Tab Compliance memakai waktu buat barisnya; tab Post Audit memakai Tanggal Kirim Post
// Audit, karena tabel datarnya memang tidak punya kolom waktu buat.
func (r Row) sortKey(tabCode string) time.Time {
	if tabCode == inboxcompliance.TabPostAudit {
		if r.Item.PostAuditSentDate == nil {
			// Tanggal kosong diurutkan paling belakang, meniru `NULLS LAST` pada
			// kuerinya.
			return time.Time{}
		}
		return *r.Item.PostAuditSentDate
	}
	return r.CreatedAt
}

// Store adalah penyimpanan antrean di memori.
//
// Ia dilindungi mutex karena CreatePostAudit dan SaveDecision MENGUBAH keadaan. Operasi
// bacanya sendiri tidak mengubah apa pun, tetapi seluruhnya menyentuh senarai yang sama.
type Store struct {
	mu   sync.Mutex
	rows []Row

	// sequence meniru POOLDATA.CLAIM_COMPLIENCE_SEQ, termasuk titik mulainya.
	sequence int64

	// decisions meniru POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE, berkunci `PZINSKEY` klaimnya.
	//
	// Map, bukan senarai, karena satu klaim hanya punya satu keputusan BERLAKU —
	// menyimpan ulang menimpa yang sebelumnya, sama seperti `MERGE` pada kuerinya.
	decisions map[string]inboxcompliance.Decision

	// claimEffects meniru kolom `T_CLAIM_PNC` yang disentuh keputusan Compliance:
	// `STATUSCLAIM`, `CPLVALID_DATE`, `POSTAUDIT_TF_ANALYSTDATE`.
	//
	// Terpisah dari `rows` karena ketiganya BUKAN kolom grid — lihat ApplyDecisionToClaim.
	claimEffects map[string]inboxcompliance.ClaimEffect

	// history meniru `POOLDATA.LIST_HISTORY_CLAIM_PNC` — senarai, bukan map, karena ia
	// APPEND-ONLY: satu klaim dapat punya banyak baris, dan yang lama tidak pernah
	// ditimpa.
	history []inboxcompliance.HistoryEntry

	// assignments meniru `POOLDATA.CPNC_PENUGASAN` — append-only, karena penugasan yang
	// selesai DITANDAI, tidak dihapus (`D-66`).
	assignments []inboxcompliance.Assignment

	// surveyResults meniru `POOLDATA.T_SURVEYORLIST`, dikunci `PNCCASEID`.
	//
	// Hanya dibaca — tidak ada method yang menulisinya, sama seperti adapter SQL. Isinya
	// disemai uji lewat SeedSurveyResults.
	surveyResults map[string][]inboxcompliance.SurveyResult

	// documentChecklist menyimpan daftar periksa kelengkapan dokumen per klaim.
	documentChecklist map[string][]inboxcompliance.DocumentChecklistItem

	// documents meniru `POOLDATA.DATA_ATTACHFILE`, dikunci `IDPEGA`.
	//
	// Hanya dibaca, sama seperti adapter SQL. Disemai uji lewat SeedDocuments.
	documents map[string][]inboxcompliance.Document

	// attachmentRunno meniru `ATTACHFILE_SEQ`.
	attachmentRunno int64

	// rejectPrefill meniru kolom tabel datar `T_CLAIM_PNC` yang dibaca form Surat
	// Penolakan.
	rejectPrefill map[string]inboxcompliance.RejectPrefill
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	// 0, bukan 1: pencacah dinaikkan LEBIH DULU saat dipakai, sehingga nomor pertama yang
	// terbit tetap bernomor urut 1 — sama dengan `START WITH 1` pada sequence-nya.
	return &Store{
		rows:      rows,
		sequence:  0,
		decisions: map[string]inboxcompliance.Decision{},
	}
}

// List mengambil satu halaman antrean, meniru predikat dan urutan kueri Oracle.
func (s *Store) List(
	_ context.Context,
	q inboxcompliance.Query,
	page inboxcompliance.Pagination,
) (inboxcompliance.Page, error) {
	// Kunci diperlukan sejak pembacaan ikut menyentuh s.assignments.
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := page.Normalize()

	matched := make([]Row, 0, len(s.rows))
	for _, row := range s.rows {
		if row.tab() != q.Tab.Code {
			continue
		}

		// Kedua penyaring berikut HANYA berlaku pada tab Compliance, karena hanya tab itu
		// yang membaca workbasket dan status kerja. Tab Post Audit membaca tabel datar
		// yang tidak punya kolom status sama sekali — lihat catatan di
		// repo/sqlstore/inboxcompliance.sql.
		if q.Tab.Code == inboxcompliance.TabCompliance {
			if row.Resolved || row.Workbasket != q.Workbasket {
				continue
			}

			// Penyaring yang SAMA dengan `NOT EXISTS` pada kueri daftar: klaim yang
			// tahap Compliance-nya sudah selesai tidak lagi ditampilkan.
			if s.complianceSelesai(row.Item.Reference) {
				continue
			}
		}

		matched = append(matched, row)
	}

	// Kedua tab diurutkan BERBEDA, dan keduanya meniru kuerinya masing-masing:
	//
	//	Compliance  ORDER BY A.PXCREATEDATETIME DESC, A.PYID DESC
	//	Post Audit  ORDER BY CASEID DESC, TGL_KIRIM_POST_AUDIT DESC
	//
	// Perhatikan tab Post Audit: nomor case yang menjadi kunci PERTAMA, dan ia diurutkan
	// sebagai TEKS. Itu bukan penyederhanaan — layar Pega menampilkan
	// `CPL-3, CPL-2, CPL-19, CPL-17, …`, yang hanya masuk akal bila teksnya yang
	// dibandingkan. Mengurutkan angkanya akan menaruh `CPL-19` di atas `CPL-3`.
	sort.SliceStable(matched, func(i, j int) bool {
		if q.Tab.Code == inboxcompliance.TabPostAudit {
			if matched[i].Item.CaseID != matched[j].Item.CaseID {
				return matched[i].Item.CaseID > matched[j].Item.CaseID
			}
			return matched[i].sortKey(q.Tab.Code).After(matched[j].sortKey(q.Tab.Code))
		}

		left, right := matched[i].sortKey(q.Tab.Code), matched[j].sortKey(q.Tab.Code)
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].Item.CaseID > matched[j].Item.CaseID
	})

	result := inboxcompliance.Page{
		Items:      []inboxcompliance.WorkItem{},
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

	for _, row := range matched[offset:end] {
		result.Items = append(result.Items, row.Item)
	}

	return result, nil
}

// FindInQueue mencari satu klaim yang sedang menunggu di antrean Compliance.
//
// Penyaringnya ditiru dari kueri `find_compliance_claim`, termasuk kenyataan bahwa klaim
// yang sudah selesai maupun yang berada di antrean lain TIDAK ditemukan.
func (s *Store) FindInQueue(
	_ context.Context, q inboxcompliance.Query, reference string,
) (inboxcompliance.WorkItem, bool, error) {
	// Kunci diperlukan sejak pembacaan ikut menyentuh s.assignments.
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, row := range s.rows {
		if row.tab() != inboxcompliance.TabCompliance {
			continue
		}
		if row.Resolved || row.Workbasket != q.Workbasket {
			continue
		}
		if row.Item.Reference != reference {
			continue
		}

		// Sama dengan kueri find_compliance_claim, yang predikatnya WAJIB sama dengan
		// daftar: klaim yang sudah berpindah tidak dapat dibuka lagi dari sini — persis
		// seperti di Pega, yang assignment Compliance-nya memang sudah tidak ada.
		if s.complianceSelesai(reference) {
			return inboxcompliance.WorkItem{}, false, nil
		}
		return row.Item, true, nil
	}

	return inboxcompliance.WorkItem{}, false, nil
}

// CreatePostAudit menulis satu baris Post Audit ke dalam memori.
//
// Nomornya dibentuk dari pencacah di memori, meniru sequence basis data — termasuk titik
// mulainya, 100.001, supaya nomor contoh berbentuk sama dengan yang terbit di Oracle.
//
// Berbeda dari operasi baca, yang ini MENGUBAH keadaan, sehingga penyimpanan ini kini
// dilindungi mutex. Tanpa itu, dua permintaan bersamaan saat pengembangan lokal dapat
// menerbitkan nomor yang sama — cacat yang justru tidak akan terjadi di Oracle, sehingga ia
// hanya akan membingungkan.
func (s *Store) CreatePostAudit(
	_ context.Context, entry inboxcompliance.PostAuditEntry,
) (inboxcompliance.PostAuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++

	saved := entry

	// Bentuknya WAJIB sama dengan yang dirakit kueri `post_audit_next_sequence`:
	//
	//     'CPL' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(seq.NEXTVAL)
	//
	// Dua digit tahun diambil dari waktu terbit baris ini, padanan `SYSDATE` di Oracle.
	// Tanpa nol di depan pada nomor urutnya — `TO_CHAR` tanpa format mask memang tidak
	// memberinya, dan fake yang merapikannya akan meloloskan pengurutan yang di Oracle
	// justru berantakan (`CPL.26.10` di atas `CPL.26.9`).
	tahun := saved.SentAt.Format("06")
	if saved.SentAt.IsZero() {
		tahun = time.Now().Format("06")
	}
	saved.CaseID = "CPL." + tahun + "." + strconv.FormatInt(s.sequence, 10)

	s.rows = append(s.rows, Row{
		Tab: inboxcompliance.TabPostAudit,
		Item: inboxcompliance.WorkItem{
			CaseID:            saved.CaseID,
			ClaimNumber:       saved.ClaimNumber,
			Reference:         saved.ClaimNumber,
			InsuredName:       saved.InsuredName,
			PolicyNumber:      saved.PolicyNumber,
			ComplianceRemarks: saved.Remarks,
			PostAuditSentDate: &saved.SentAt,
		},
	})

	return saved, nil
}

// FindDecision mengambil keputusan Compliance yang sudah tersimpan atas satu klaim.
//
// Tidak ditemukan BUKAN galat: klaim yang baru masuk antrean memang belum diputuskan.
func (s *Store) FindDecision(
	_ context.Context, reference string,
) (inboxcompliance.Decision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	decision, exists := s.decisions[reference]
	return decision, exists, nil
}

// SaveDecision menyimpan keputusan Compliance, menimpa keputusan sebelumnya atas klaim yang
// sama.
//
// Menimpa, bukan menambah, karena satu klaim hanya punya satu keputusan Compliance yang
// berlaku — `.ClaimData.PilihanCompliance` adalah satu properti pada klaimnya, bukan daftar.
// Riwayat perubahannya ada di tempat lain (`InsertHistoryClaimPNC`), dan itu bukan tabel ini.
// Komentarnya TIDAK ditimpa seluruhnya melainkan digabung per nomor baris, dan itu bukan
// kerumitan yang dikarang: adapter SQL menyimpannya lewat `MERGE` per baris, yang menurut
// sifatnya tidak pernah menghapus baris lama. Menyimpan dua komentar setelah sebelumnya
// ada tiga meninggalkan baris ketiga di sana.
//
// Fake yang mengganti seluruhnya akan meluluskan uji yang gagal di basis data — persis
// kelas cacat yang seam ini ada untuk menangkapnya, bukan menyembunyikannya.
func (s *Store) SaveDecision(
	_ context.Context, decision inboxcompliance.Decision,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Komentar DITIMPA SELURUHNYA, tidak digabung dengan yang tersimpan.
	//
	// Itu meniru kolom `KOMENTAR_JSON` pada adapter SQL: satu kolom JSON ditulis utuh
	// setiap kali, sehingga baris yang tidak ikut dikirim memang HILANG.
	//
	// Versi sebelumnya menggabungkan per nomor baris, meniru `MERGE` pada tabel komentar
	// terpisah. Tabel itu sudah tidak ada sejak 2026-10-07, dan penggabungannya tertinggal
	// — membuat fake MENOLAK penghapusan yang di Oracle justru berhasil. Tombol "Hapus"
	// pada grid Pega menuntut penghapusan itu benar-benar terjadi.
	s.decisions[decision.Reference] = decision
	return nil
}

// ApplyDecisionToClaim meniru `UPDATE POOLDATA.T_CLAIM_PNC` pada adapter SQL.
//
// Ia menyentuh baris antrean yang ada di penyimpanan ini, sehingga uji dapat memeriksa
// klaimnya BENAR-BENAR berpindah status — bukan hanya bahwa method-nya terpanggil.
//
// Perilaku nil ditiru persis dari `COALESCE(:n, kolom)` pada kuerinya: nil berarti
// **jangan sentuh**, bukan kosongkan. Fake yang mengosongkannya akan meloloskan cacat yang
// di Oracle justru tidak terjadi — tanggal valid yang lenyap saat keputusan diubah.
func (s *Store) ApplyDecisionToClaim(
	_ context.Context, effect inboxcompliance.ClaimEffect,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.claimEffects == nil {
		s.claimEffects = map[string]inboxcompliance.ClaimEffect{}
	}

	// Ketiga kolomnya TIDAK disimpan ke WorkItem, karena WorkItem adalah bentuk baris
	// GRID dan tidak satu pun dari ketiganya digambar di sana. Menambahkannya ke WorkItem
	// akan membuat tipe itu mengaku membawa kolom yang tidak pernah dibaca layar.
	//
	// Perilaku nil ditiru persis dari `COALESCE(:n, kolom)` pada kuerinya: nil berarti
	// **jangan sentuh**, bukan kosongkan. Fake yang mengosongkannya akan meloloskan cacat
	// yang di Oracle tidak terjadi — tanggal valid yang lenyap saat keputusan diubah.
	sebelumnya := s.claimEffects[effect.Reference]
	if effect.ValidatedAt == nil {
		effect.ValidatedAt = sebelumnya.ValidatedAt
	}
	if effect.SentToPostAuditAt == nil {
		effect.SentToPostAuditAt = sebelumnya.SentToPostAuditAt
	}

	s.claimEffects[effect.Reference] = effect
	return nil
}

// AppendHistory meniru INSERT ke `POOLDATA.LIST_HISTORY_CLAIM_PNC`.
//
// Append-only, persis seperti adapter SQL: barisnya ditambahkan, tidak pernah ditimpa
// maupun dibuang. Fake yang menimpanya akan menyembunyikan keputusan yang diubah dua kali
// — padahal justru itu yang paling ingin terlihat di jejak audit.
func (s *Store) AppendHistory(
	_ context.Context, entry inboxcompliance.HistoryEntry,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history, entry)
	return nil
}

// History mengembalikan seluruh baris riwayat satu klaim, berurutan sesuai penulisannya.
func (s *Store) History(reference string) []inboxcompliance.HistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rows []inboxcompliance.HistoryEntry
	for _, e := range s.history {
		if e.Reference == reference {
			rows = append(rows, e)
		}
	}
	return rows
}

// SeedSurveyResults menyemai hasil investigasi satu klaim.
//
// Bukan bagian dari antarmuka Repo — ia alat uji, sepadan dengan mengisi
// `T_SURVEYORLIST` sebelum menjalankan kueri. Tabel itu ditulis modul lain (Investigator
// dan Survey), bukan oleh modul ini, sehingga fake pun tidak boleh punya method penulis
// yang menyaru sebagai bagian kontrak.
func (s *Store) SeedSurveyResults(
	reference string, results ...inboxcompliance.SurveyResult,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.surveyResults == nil {
		s.surveyResults = map[string][]inboxcompliance.SurveyResult{}
	}
	s.surveyResults[reference] = results
}

// FindSurveyResults meniru pembacaan `POOLDATA.T_SURVEYORLIST`.
//
// Klaim tanpa baris mengembalikan senarai KOSONG dan nil galat — keadaan normal bagi
// klaim yang belum pernah disurvei, bukan kegagalan.
func (s *Store) FindSurveyResults(
	_ context.Context, reference string,
) ([]inboxcompliance.SurveyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Salinan, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah isi
	// "tabel" dengan menulisi hasil bacaannya.
	stored := s.surveyResults[reference]
	results := make([]inboxcompliance.SurveyResult, len(stored))
	copy(results, stored)
	return results, nil
}

// SeedRejectPrefill menyemai isian pra-isi Surat Penolakan satu klaim.
func (s *Store) SeedRejectPrefill(
	reference string, prefill inboxcompliance.RejectPrefill,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rejectPrefill == nil {
		s.rejectPrefill = map[string]inboxcompliance.RejectPrefill{}
	}
	s.rejectPrefill[reference] = prefill
}

// FindRejectPrefill meniru pembacaan tabel datar `T_CLAIM_PNC`.
//
// Klaim tanpa semaian mengembalikan isian KOSONG tanpa galat — meniru `LEFT JOIN` dan
// `sql.ErrNoRows` pada adapter SQL, bukan memperlakukannya sebagai kelainan.
func (s *Store) FindRejectPrefill(
	_ context.Context, reference string,
) (inboxcompliance.RejectPrefill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rejectPrefill[reference], nil
}

// SeedDocuments menyemai dokumen satu klaim.
//
// Alat uji, bukan bagian antarmuka Repo — `DATA_ATTACHFILE` ditulis jalur unggah, bukan
// oleh pembacaan form ini.
func (s *Store) SeedDocuments(reference string, docs ...inboxcompliance.Document) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.documents == nil {
		s.documents = map[string][]inboxcompliance.Document{}
	}
	s.documents[reference] = docs
}

// FindDocuments meniru pembacaan `POOLDATA.DATA_ATTACHFILE`.
//
// Baris tanpa StorageID DIBUANG, meniru penyaring `IMAGEID IS NOT NULL` pada kuerinya.
// Tanpa itu, fake ini akan meloloskan baris yang di Oracle tidak pernah terbaca — dan
// uji yang hijau atasnya tidak membuktikan apa pun.
func (s *Store) FindDocuments(
	_ context.Context, reference string,
) ([]inboxcompliance.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	documents := []inboxcompliance.Document{}
	for _, doc := range s.documents[reference] {
		if doc.StorageID == "" {
			continue
		}
		documents = append(documents, doc)
	}
	return documents, nil
}

// SaveDocument meniru penulisan `DATA_ATTACHFILE` beserta pembentukan `DATAID`.
//
// Nomornya `YY` + sepuluh digit, sama bentuknya dengan `SET_ATTACHMENT_64BIT` — bukan
// bentuk bebas. Fake yang menerbitkan bentuk lain akan meloloskan pemanggil yang
// mengandaikan panjangnya, dan itu baru terlihat di Oracle.
func (s *Store) SaveDocument(
	_ context.Context, document inboxcompliance.Document,
) (inboxcompliance.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.documents == nil {
		s.documents = map[string][]inboxcompliance.Document{}
	}

	s.attachmentRunno++
	document.ID = fmt.Sprintf("%s%010d", time.Now().Format("06"), s.attachmentRunno)

	s.documents[document.ClaimReference] = append(
		s.documents[document.ClaimReference], document)
	return document, nil
}

// DeleteDocument meniru `DELETE … WHERE DATAID = :1 AND IDPEGA = :2`.
//
// Penyaring klaim IKUT ditiru. Fake yang menghapus hanya dengan id dokumen akan
// meloloskan penghapusan lintas klaim yang kueri sebenarnya tolak — dan itu justru
// pengetatan yang paling ingin dijaga di jalur ini.
func (s *Store) DeleteDocument(
	_ context.Context, reference, documentID string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sisa := make([]inboxcompliance.Document, 0, len(s.documents[reference]))
	terhapus := false
	for _, doc := range s.documents[reference] {
		if doc.ID == documentID {
			terhapus = true
			continue
		}
		sisa = append(sisa, doc)
	}
	s.documents[reference] = sisa
	return terhapus, nil
}

// MoveAssignment menutup tahap lama dan membuka tahap baru.
//
// Keduanya ditambahkan BERSAMAAN di bawah satu kunci, meniru satu transaksi pada adapter
// SQL. Fake yang menambahkannya satu per satu akan meloloskan keadaan separuh jalan yang
// di Oracle tidak mungkin terjadi — dan keadaan itulah yang paling berbahaya: klaim hilang
// dari antrean tanpa tiba di mana pun.
func (s *Store) MoveAssignment(
	_ context.Context, move inboxcompliance.AssignmentMove,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assignments = append(s.assignments, move.Closed, move.Opened)
	return nil
}

// Assignments mengembalikan seluruh penugasan satu klaim, untuk pengujian.
func (s *Store) Assignments(reference string) []inboxcompliance.Assignment {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rows []inboxcompliance.Assignment
	for _, a := range s.assignments {
		if a.Reference == reference {
			rows = append(rows, a)
		}
	}
	return rows
}

// complianceSelesai meniru penyaring `NOT EXISTS` pada kueri daftar.
//
// Ia WAJIB ada di fake, bukan hanya di SQL: tanpa itu, uji akan memperlihatkan klaim tetap
// di antrean setelah diputuskan — padahal di Oracle ia hilang. Fake yang menyaring berbeda
// dari kuerinya adalah fake yang membohongi ujinya sendiri.
//
// Pemanggil WAJIB sudah memegang kunci.
func (s *Store) complianceSelesai(reference string) bool {
	for _, a := range s.assignments {
		if a.Reference == reference &&
			a.Stage == inboxcompliance.StageCompliance &&
			a.Status == inboxcompliance.AssignmentDone {
			return true
		}
	}
	return false
}

// ClaimEffect mengembalikan akibat yang tersimpan pada satu klaim, untuk pengujian.
//
// Nilai kedua salah ketika klaimnya belum pernah disentuh — dibedakan dari "disentuh
// dengan nilai kosong", karena keduanya memang berbeda.
func (s *Store) ClaimEffect(reference string) (inboxcompliance.ClaimEffect, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	effect, ada := s.claimEffects[reference]
	return effect, ada
}

// `mergeComments` DIBUANG 2026-10-07.
//
// Ia menggabungkan komentar per nomor baris, meniru `MERGE` pada tabel komentar terpisah.
// Tabel itu sudah tidak ada — komentar kini satu kolom JSON yang ditulis utuh — sehingga
// penggabungannya membuat fake MENOLAK penghapusan yang di Oracle justru berhasil.
//
// Dicatat alih-alih dihapus diam-diam, karena ia tampak seperti pengaman: orang berikutnya
// yang melihat SaveDecision menimpa seluruh daftar dapat mengira itu kelalaian, lalu
// memasangnya kembali — dan tombol Hapus pada grid akan diam-diam berhenti bekerja.

// SeedDocumentChecklist menyemai daftar periksa kelengkapan dokumen satu klaim.
func (s *Store) SeedDocumentChecklist(
	reference string, items ...inboxcompliance.DocumentChecklistItem,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.documentChecklist == nil {
		s.documentChecklist = map[string][]inboxcompliance.DocumentChecklistItem{}
	}
	s.documentChecklist[reference] = items
}

// FindDocumentChecklist memenuhi inboxcompliance.Repo.
func (s *Store) FindDocumentChecklist(
	_ context.Context, reference string,
) ([]inboxcompliance.DocumentChecklistItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Salinan, bukan senarai aslinya — alasan yang sama dengan FindSurveyResults:
	// pemanggil tidak boleh dapat mengubah isi "tabel" dengan menulisi hasil bacaannya.
	stored := s.documentChecklist[reference]
	items := make([]inboxcompliance.DocumentChecklistItem, len(stored))
	copy(items, stored)
	return items, nil
}

// UpdateDocumentCategory memenuhi inboxcompliance.Repo.
func (s *Store) UpdateDocumentCategory(
	_ context.Context, reference, documentID, category string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Kedua penyaring ditiru: dokumen milik klaim LAIN tidak boleh terpindahkan, dan
	// tanpa pemeriksaan itu fake ini akan meloloskan cacat yang kueri aslinya tolak.
	for i := range s.documents[reference] {
		if s.documents[reference][i].ID == documentID {
			s.documents[reference][i].Category = category
			return true, nil
		}
	}
	return false, nil
}
