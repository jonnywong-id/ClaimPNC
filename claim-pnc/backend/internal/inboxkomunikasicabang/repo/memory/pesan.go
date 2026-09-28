package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// SampleBranches adalah daftar cabang contoh untuk pemilih tujuan.
//
// Isinya KARANGAN. `D-69` melarang data nasabah ditulis di berkas yang di-commit, dan
// kode cabang beserta alamat surelnya adalah data perusahaan yang tunduk pada aturan yang
// sama.
//
// Ketiganya sejajar dengan SampleBranchOfLogin: `1001` dan `1002` dimiliki login contoh,
// sehingga uji dapat mengirim pesan ke cabang yang benar-benar dapat membacanya kembali.
// `1003` sengaja TIDAK dimiliki siapa pun — ia saksi bahwa daftar cabang TIDAK disaring
// menurut cabang pemanggil.
func SampleBranches() []inboxkomunikasicabang.BranchOption {
	return []inboxkomunikasicabang.BranchOption{
		{Code: "1001", Name: "CABANG SURABAYA", Email: "cabang.1001@example.invalid"},
		{Code: "1002", Name: "CABANG BANDUNG", Email: "cabang.1002@example.invalid"},
		{Code: "1003", Name: "CABANG MEDAN", Email: "cabang.1003@example.invalid"},
	}
}

// Branches mengembalikan daftar cabang, diurutkan menurut nama.
//
// Urutannya ditiru dari `ORDER BY BRANCHNAME ASC` pada kueri SQL. Penyimpanan memori yang
// mengembalikan urutan berbeda akan membuat uji urutan lulus di sini dan gagal di Oracle.
func (s *Store) Branches(_ context.Context) ([]inboxkomunikasicabang.BranchOption, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := make([]inboxkomunikasicabang.BranchOption, len(s.branches))
	copy(result, s.branches)

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// SendMessage membuat percakapan BARU.
//
// Nomornya ditiru dari `MAX(KOMUNIKASIID)` pada kueri SQL — bukan dari penghitung yang
// berdiri sendiri. Nomor yang lahir dengan cara berbeda akan membuat uji lulus untuk
// perilaku yang tidak terjadi di Oracle.
func (s *Store) SendMessage(
	_ context.Context,
	command inboxkomunikasicabang.NewMessageCommand,
	origin string,
) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := s.nextConversationID()

	s.rows = append(s.rows, Row{
		ID:     id,
		CaseID: inboxkomunikasicabang.CaseOpen,

		// Asal dan tujuan disimpan MENTAH, sama seperti kolomnya. Penerjemahannya menjadi
		// "PUSAT"/"CABANG" terjadi saat baris dibaca, sehingga penerjemahnya ikut teruji.
		CommunicateFrom: strings.TrimSpace(origin),
		CommunicateTo:   strings.TrimSpace(command.RecipientCode()),

		Sender:     command.Sender.Login,
		SenderName: command.Sender.Name,
		Message:    command.Message,
		Status:     inboxkomunikasicabang.StatusNotAnswered,

		// Tanggal diisi dari perintahnya. Di Oracle kolom ini TIDAK disebut INSERT-nya dan
		// diisi basis data; di sini tidak ada basis data yang dapat mengisinya, sehingga
		// waktu perintah dipakai. Selisih itu disengaja dan tidak dapat dihindari — ia
		// dinyatakan pada catatan NewMessageCommand.CreatedAt.
		CreatedAt: command.CreatedAt.Format(replyTimeLayout),
	})

	s.history = append(s.history, ReplyHistory{
		Sender:         command.Sender.Login,
		Message:        command.Message,
		ConversationID: id,

		// Kode cabang TUJUAN — berbeda dari jalur balasan, yang mengisi kolom yang sama
		// dengan penanda kanal. Dua konvensi dalam satu kolom, dan keduanya direplikasi.
		Channel:   command.BranchCode,
		CreatedAt: command.CreatedAt.Format(replyTimeLayout),
	})

	return id, nil
}

// nextConversationID meniru `SELECT MAX(KOMUNIKASIID)` sesudah penyisipan.
//
// Ia membandingkan sebagai ANGKA bila seluruh nomornya berupa angka, dan sebagai TEKS bila
// tidak. Baris contoh bernomor `KOM-0001`; baris yang lahir dari layar akan bernomor angka
// di Oracle. Keduanya harus bekerja, dan mencampurnya tidak boleh menghasilkan nomor yang
// bentrok.
func (s *Store) nextConversationID() string {
	highest := 0
	for _, row := range s.rows {
		if n, err := strconv.Atoi(strings.TrimSpace(row.ID)); err == nil && n > highest {
			highest = n
		}
	}

	// Baris contoh yang bernomor `KOM-xxxx` tidak terbaca sebagai angka, sehingga nomor
	// pertama yang terbit adalah 1. Itu tidak bentrok: penyimpanan ini hanya dipakai untuk
	// pengujian dan pengembangan lokal, dan kedua bentuk nomor hidup berdampingan persis
	// seperti nomor klaim warisan hidup berdampingan dengan `PNCN.YY.xxxx` (`D-71`).
	return strconv.Itoa(highest + 1)
}
