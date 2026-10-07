package memory

import (
	_ "embed"

	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// # PERINGATAN — SELURUH ISI BERKAS INI BUKAN DATA PRODUKSI
//
// Isi sebenarnya POOLDATA.M_AUTO_CLAIM_PNC, POOLDATA.AGENT, POOLDATA.CLIENT, dan
// POOLDATA.EMAILKOMITE tidak ada di export: tidak ada berkas CSV-nya di `Database/`
// seperti halnya `v_sts_claim.csv`, `emailkomite.csv`, dan `m_portal_pnc.csv`. DDL
// keempatnya pun belum diterima (R-08).
//
// Nama, kode, nomor rekening, dan alamat di bawah karena itu SUSUNAN SENDIRI dan sengaja
// dibuat terbaca sebagai contoh. Tidak satu pun diambil dari data nasabah, dan itu
// mengikat: `D-69` melarang nomor polis, nama tertanggung, NPWP, dan nomor rekening
// nyata ditulis di berkas yang di-commit.
//
// Daftar ini TIDAK boleh dipakai sebagai dasar uji kesetaraan gerbang 1, dan harus
// diganti isi tabel yang sebenarnya begitu DBA mengirimkannya.

// SampleCommittee adalah operator komite contoh.
//
// Bentuknya meniru isi POOLDATA.EMAILKOMITE.OPERATOR_ID — huruf besar tanpa spasi —
// karena itulah yang dibandingkan penyaring tab Komite Approval. Nilainya harus sama
// dengan login pengguna contoh supaya tabnya dapat dicoba; lihat
// `provider/fake.go`, yang memuat `adminpnc`.
const SampleCommittee = "ADMINPNC"

// SampleList adalah isi awal master auto claim untuk pengembangan.
//
// Keenam baris sengaja tersebar di ketiga status supaya keempat tab layar dapat dicoba
// tanpa memasukkan data lebih dulu:
//
//	APPROVAL="1"  2 baris  -> tab Master Auto Klaim
//	APPROVAL="0"  3 baris  -> tab Waiting Approval, 2 di antaranya ber-KOMITE contoh
//	                          sehingga muncul juga di tab Komite Approval
//	APPROVAL="2"  1 baris  -> tab Reject
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Baris TANPA penyetuju komite, sengaja ada. Ia yang memperlihatkan akibat
// POOLDATA.EMAILKOMITE kosong: barisnya muncul di Waiting Approval tetapi
// TIDAK PERNAH muncul di tab Komite Approval siapa pun, sehingga tertahan di
// sana selamanya. Lihat usecase.Service.Create.
func SampleList() []masterautoclaim.AutoClaim {
	return sampledata.Must[[]masterautoclaim.AutoClaim](sampleJSON, "SampleList")
}

// SampleBusinessSources adalah lookup Sumber Bisnis contoh, meniru POOLDATA.AGENT.
//
// Keenam yang pertama sengaja sama dengan yang sudah ada di SampleList supaya penambahan
// atas kode yang sudah dipakai dapat dicoba — dan pesan "sudah ada di master" terlihat.
// Tiga terakhir belum dipakai, sehingga penambahan yang berhasil juga dapat dicoba.
func SampleBusinessSources() []masterautoclaim.BusinessSource {
	return sampledata.Must[[]masterautoclaim.BusinessSource](sampleJSON, "SampleBusinessSources")
}

// SampleClients adalah lookup Client contoh, meniru POOLDATA.CLIENT.
func SampleClients() []masterautoclaim.Client {
	return sampledata.Must[[]masterautoclaim.Client](sampleJSON, "SampleClients")
}

// SampleBanks adalah daftar bank contoh, meniru GENERAL.LST_BANK_GROUP.
//
// Nama-namanya sengaja BUKAN nama bank yang sesungguhnya. Bank contoh yang tidak ada di
// daftar ini dipakai menguji penolakan "harus dipilih dari daftar".
func SampleBanks() []masterautoclaim.Bank {
	return sampledata.Must[[]masterautoclaim.Bank](sampleJSON, "SampleBanks")
}
