// Package simasbid memenuhi seam inboxsalvage.AuctionHouse dengan panggilan HTTP nyata.
//
// # Apa yang digantikan
//
// `Connect REST/SendData_SalvageSimasBid-ConnectREST.xml`, dipanggil langkah ke-19
// `Activity/SetStsSalvagePNC_act-Act.xml` lewat `Insert_salvageToSimasBid`.
//
// # Alamatnya TIDAK punya nilai bawaan, dan itu disengaja
//
// Satu-satunya alamat yang terbaca dari export menunjuk host **sandbox** di dalam ruleset
// produksi, dengan `pyUseAuthentication=false`. Keadaan itu persis yang `R-18` catat, dan
// `ADR-0025` menetapkan endpoint menjadi konfigurasi per lingkungan — bukan konstanta di
// dalam rule.
//
// Menjadikan alamat sandbox itu nilai bawaan berarti satu lingkungan yang lupa
// mengisinya akan diam-diam mengirim pengajuan nyata ke lingkungan uji pihak lain. Karena
// itu klien ini TIDAK dibentuk sama sekali bila `SIMASBID_ALAMAT` kosong, dan layar
// menyatakan pengiriman tidak aktif alih-alih berpura-pura mengirim.
//
// Nilai alamatnya tidak direproduksi di berkas mana pun yang di-commit (`D-69`).
//
// # Yang belum dapat dibuktikan
//
// Bentuk badan permintaan di bawah disusun dari properti yang disalin
// `Insert_salvageToSimasBid` ke `MyServicePage.ClaimData.Lelang`, lalu diserialkan utuh
// oleh `@GCNM.GetPageJSONString()`. Bentuk JAWABANNYA lebih lemah lagi dasarnya: yang
// terbaca hanyalah bahwa activity membacanya sebagai
// `MyServicePage.ResponseService.ResponseMessage` dan menilai keberhasilan dengan
// `@contains(…,"Sukses")`. Nama field JSON-nya sendiri tidak ada di export.
//
// Pembacaan jawaban di bawah karena itu MENERIMA beberapa bentuk sekaligus — lihat
// response. Bila SimasBid menuntut bentuk lain, yang berubah hanya berkas ini.
package simasbid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/inboxsalvage"
)

// Config adalah alamat dan kredensial layanan balai lelang.
type Config struct {
	// URL adalah alamat lengkap layanan penyisip salvage.
	//
	// Tidak punya nilai bawaan. Lihat banner paket.
	URL string

	// User dan Password boleh kosong.
	//
	// Rule lama menetapkan `pyUseAuthentication=false`, dan itu TIDAK dijadikan keharusan
	// di sini: menyalin ketiadaan autentikasi berarti membawa serta kelemahannya. Yang
	// dibawa adalah kemampuannya mengirim tanpa kredensial bila memang begitu yang
	// disepakati, bukan larangan memakainya.
	User     string
	Password string

	// Timeout membatasi lama menunggu jawaban balai lelang. Nol berarti nilai baku.
	//
	// Ia WAJIB ada nilainya — `09-API-STRATEGY.md` §8.2 menetapkan batas waktu wajib di
	// setiap pemanggilan keluar, karena satu sistem yang menggantung akan menghabiskan
	// seluruh koneksi kita. Rule lama justru menetapkan `pyResponseTimeout=0`, yang
	// berarti menunggu tanpa batas.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi ini cukup untuk menghubungi balai lelang.
func (k Config) Complete() bool { return strings.TrimSpace(k.URL) != "" }

const defaultTimeout = 30 * time.Second

// maxResponseBody membatasi jawaban yang dibaca ke memori.
//
// Jawaban balai lelang adalah satu kalimat; batas ini ada supaya layanan yang keliru
// mengalirkan berkas tidak pernah membengkakkan memori aplikasi.
const maxResponseBody = 1 << 20 // 1 MiB

// Client menghubungi balai lelang SimasBid lewat HTTP.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient membentuk klien balai lelang.
func NewClient(k Config) *Client {
	timeout := k.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{cfg: k, http: &http.Client{Timeout: timeout}}
}

// lelang adalah isi `MyServicePage.ClaimData.Lelang` pada rule lama.
//
// Nama field JSON-nya mengikuti nama properti Pega apa adanya, karena nama itulah yang
// sampai ke SimasBid — `@GCNM.GetPageJSONString()` menyerialkan halaman dengan nama
// propertinya sendiri. Ia KONTRAK milik sistem lain, sehingga `D-80` tidak berlaku
// padanya; yang berbahasa Inggris dan mengikuti gaya kita adalah nama field Go di
// sebelah kiri.
type lelang struct {
	IDObject        string     `json:"IDObject"`
	ObjectName      string     `json:"ObjectName"`
	Location        string     `json:"Location"`
	FlagJabodetabek string     `json:"FlagJabodetabek"`
	ItemDescription string     `json:"ItemDescription"`
	Type            string     `json:"Type"`
	SalvageStatus   string     `json:"SalvageStatus"`
	Harga           string     `json:"Harga"`
	Flag            string     `json:"Flag"`
	DocumentList    []document `json:"DocumentList"`
}

// document adalah satu berkas pada `Lelang.DocumentList`.
type document struct {
	FileName string `json:"FileName"`
	Format   string `json:"Format"`
	Base64   string `json:"Base64"`
}

// claimData membungkus lelang, mengikuti susunan halaman aslinya.
type claimData struct {
	Lelang lelang `json:"Lelang"`
}

// request adalah badan permintaan utuh.
type request struct {
	ClaimData claimData `json:"ClaimData"`
}

// Nilai tetap pada badan permintaan, disalin dari `Insert_salvageToSimasBid`.
//
// Keduanya konstanta di sana — `Lelang.Type := "1"` dan `Lelang.SalvageStatus := "1"` —
// bukan turunan dari data pengajuan. Keduanya dinamai di sini supaya angkanya tidak
// tampak seperti pilihan yang dibuat modul ini.
const (
	auctionTypeInsert   = "1"
	auctionStatusActive = "1"
)

// response adalah jawaban balai lelang.
//
// # Kenapa bentuknya longgar
//
// Nama field JSON-nya tidak ada di export — lihat banner paket. Yang diketahui hanyalah
// activity lama membacanya sebagai `ResponseService.ResponseMessage`. Struktur di bawah
// karena itu menerima pesan pada DUA tempat: di dalam `ResponseService`, dan di akar.
//
// Longgar di sisi PEMBACAAN tidak melonggarkan PENILAIAN: yang memutuskan diterima atau
// tidak tetap inboxsalvage.AuctionReceipt.Accepted, satu tempat, dengan aturan yang sama
// untuk setiap pengisi seam.
type response struct {
	ResponseService struct {
		ResponseMessage string `json:"ResponseMessage"`
		ResponseID      string `json:"ResponseID"`
	} `json:"ResponseService"`

	ResponseMessage string `json:"ResponseMessage"`
	IDSimasBid      string `json:"IDSimasBid"`
}

// message memilih pesan yang benar-benar terisi.
func (r response) message() string {
	if nested := strings.TrimSpace(r.ResponseService.ResponseMessage); nested != "" {
		return nested
	}
	return strings.TrimSpace(r.ResponseMessage)
}

// auctionID memilih nomor yang benar-benar terisi.
func (r response) auctionID() string {
	if nested := strings.TrimSpace(r.ResponseService.ResponseID); nested != "" {
		return nested
	}
	return strings.TrimSpace(r.IDSimasBid)
}

// SendSalvage mengirim satu pengajuan salvage ke balai lelang.
//
// # Kode HTTP di luar 2xx BUKAN galat Go
//
// Ia jawaban yang sah dari sistem lain: pengajuan sampai ke sana dan ditolak. Yang
// dikembalikan karena itu adalah AuctionReceipt berisi pesannya — bukan error — supaya
// pemanggil mencatat penolakannya pada baris pengajuan alih-alih memperlakukannya sebagai
// jaringan yang putus.
//
// Yang menghasilkan galat Go hanyalah keadaan yang membuat jawabannya TIDAK PERNAH ADA:
// alamat kosong, permintaan tidak tersusun, sambungan gagal.
func (k *Client) SendSalvage(
	ctx context.Context,
	submission inboxsalvage.AuctionSubmission,
) (inboxsalvage.AuctionReceipt, error) {
	if !k.cfg.Complete() {
		return inboxsalvage.AuctionReceipt{}, inboxsalvage.ErrAuctionNotAvailable
	}

	body := request{ClaimData: claimData{Lelang: lelang{
		IDObject: submission.ItemID,
		// Nama barang yang kosong dikirim apa adanya, tidak diganti nomor klaim.
		// Menambalnya di sini membuat balai lelang menerima nama yang tidak pernah
		// diketik siapa pun, dan petugas tidak akan pernah tahu isiannya terlewat.
		ObjectName:      submission.ItemName,
		Location:        submission.Location,
		FlagJabodetabek: boolText(submission.InJabodetabek),
		ItemDescription: submission.ItemDescription,
		Type:            auctionTypeInsert,
		SalvageStatus:   auctionStatusActive,
		Harga:           submission.Price,
		Flag:            submission.EntityFlag,
		DocumentList:    documentsOf(submission.Documents),
	}}}

	content, err := json.Marshal(body)
	if err != nil {
		return inboxsalvage.AuctionReceipt{},
			fmt.Errorf("inboxsalvage/simasbid: menyusun permintaan: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, k.cfg.URL, bytes.NewReader(content))
	if err != nil {
		return inboxsalvage.AuctionReceipt{},
			fmt.Errorf("inboxsalvage/simasbid: menyusun permintaan HTTP: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	if k.cfg.User != "" {
		httpRequest.SetBasicAuth(k.cfg.User, k.cfg.Password)
	}

	result, err := k.http.Do(httpRequest)
	if err != nil {
		// Galatnya dibungkus TANPA menyertakan URL: alamat layanan dapat memuat token di
		// query string, dan galat ini berakhir di log.
		return inboxsalvage.AuctionReceipt{},
			fmt.Errorf("inboxsalvage/simasbid: menghubungi balai lelang: %w", err)
	}
	defer func() { _ = result.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(result.Body, maxResponseBody))
	if err != nil {
		return inboxsalvage.AuctionReceipt{},
			fmt.Errorf("inboxsalvage/simasbid: membaca jawaban balai lelang: %w", err)
	}

	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Jawaban yang tidak terbaca sebagai JSON tetap dikembalikan sebagai PESAN, bukan
		// sebagai galat: balai lelang yang menjawab halaman galat HTML sudah menerima
		// permintaannya, dan petugas perlu melihat apa yang dijawabnya.
		//
		// Ia tidak akan memuat kata "Sukses", sehingga penilaiannya tetap "ditolak".
		return inboxsalvage.AuctionReceipt{Message: summarize(raw, result.StatusCode)}, nil
	}

	message := parsed.message()
	if message == "" {
		message = summarize(raw, result.StatusCode)
	}

	return inboxsalvage.AuctionReceipt{
		AuctionID: parsed.auctionID(),
		Message:   message,
	}, nil
}

// documentsOf menerjemahkan lampiran domain menjadi bentuk kawat.
//
// Mengembalikan senarai KOSONG, bukan nil, supaya badan permintaan selalu memuat
// `"DocumentList": []` — bentuk yang sama apakah ada lampiran atau tidak. Penerima yang
// menguraikan JSON dengan skema ketat sering menolak field yang berubah dari senarai
// menjadi null.
func documentsOf(list []inboxsalvage.AuctionDocument) []document {
	result := make([]document, 0, len(list))
	for _, item := range list {
		result = append(result, document{
			FileName: item.FileName,
			Format:   item.Format,
			Base64:   item.Content,
		})
	}
	return result
}

// boolText menuliskan penanda Jabodetabek sebagai teks.
//
// Rule lama mengirimnya sebagai TEKS, bukan boolean JSON:
//
//	@if(Param.isjabodatabek=="","false",Param.isjabodatabek)
//
// Bentuk itu dipertahankan. Mengubahnya menjadi boolean adalah perubahan kontrak terhadap
// sistem yang tidak kita miliki.
func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// summarize menyusun pesan dari jawaban yang tidak terbaca.
//
// Isinya dipotong supaya satu halaman galat HTML tidak pernah masuk utuh ke kolom basis
// data maupun ke layar.
func summarize(raw []byte, status int) string {
	text := strings.TrimSpace(string(raw))
	const limit = 200
	if len(text) > limit {
		text = text[:limit] + "…"
	}
	if text == "" {
		return fmt.Sprintf("balai lelang menjawab HTTP %d tanpa isi", status)
	}
	return fmt.Sprintf("HTTP %d: %s", status, text)
}

var _ inboxsalvage.AuctionHouse = (*Client)(nil)
