package memory

import "claim-pnc/internal/masterautoclaim"

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
func SampleList() []masterautoclaim.AutoClaim {
	return []masterautoclaim.AutoClaim{
		{
			Initial:         "AGN001",
			ReceiverName:    "MITRA CONTOH SEJAHTERA",
			BankName:        "BANK CONTOH NIAGA",
			AccountNumber:   "1000000001",
			MaxPercent:      "100",
			ReporterPIC:     "PIC Contoh Satu",
			ReporterEmail:   "pic.satu@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 1, Jakarta",
			SubmittedBy:     SampleCommittee,
			Committee:       SampleCommittee,
			Status:          masterautoclaim.StatusApproved,
			ClientID:        "CLI001",
			ClientName:      "TERTANGGUNG CONTOH PERTAMA",
		},
		{
			Initial:         "AGN002",
			ReceiverName:    "KOPERASI CONTOH BERSAMA",
			BankName:        "BANK CONTOH MANDIRI",
			AccountNumber:   "1000000002",
			MaxPercent:      "80",
			ReporterPIC:     "PIC Contoh Dua",
			ReporterEmail:   "pic.dua@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 2, Bandung",
			SubmittedBy:     SampleCommittee,
			Committee:       SampleCommittee,
			Status:          masterautoclaim.StatusApproved,
			ClientID:        "CLI002",
			ClientName:      "TERTANGGUNG CONTOH KEDUA",
		},
		{
			Initial:         "AGN003",
			ReceiverName:    "MULTIFINANCE CONTOH ABADI",
			BankName:        "BANK CONTOH NIAGA",
			AccountNumber:   "1000000003",
			MaxPercent:      "75,5",
			ReporterPIC:     "PIC Contoh Tiga",
			ReporterEmail:   "pic.tiga@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 3, Surabaya",
			SubmittedBy:     "PICTEKNIKS",
			Committee:       SampleCommittee,
			Status:          masterautoclaim.StatusPending,
			ClientID:        "CLI003",
			ClientName:      "TERTANGGUNG CONTOH KETIGA",
		},
		{
			Initial:         "AGN004",
			ReceiverName:    "BPR CONTOH LESTARI",
			BankName:        "BANK CONTOH RAKYAT",
			AccountNumber:   "1000000004",
			MaxPercent:      "60",
			ReporterPIC:     "PIC Contoh Empat",
			ReporterEmail:   "pic.empat@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 4, Medan",
			SubmittedBy:     "PICTEKNIKS",
			Committee:       SampleCommittee,
			Status:          masterautoclaim.StatusPending,
			ClientID:        "",
			ClientName:      "",
		},
		{
			// Baris TANPA penyetuju komite, sengaja ada. Ia yang memperlihatkan akibat
			// POOLDATA.EMAILKOMITE kosong: barisnya muncul di Waiting Approval tetapi
			// TIDAK PERNAH muncul di tab Komite Approval siapa pun, sehingga tertahan di
			// sana selamanya. Lihat usecase.Service.Create.
			Initial:         "AGN005",
			ReceiverName:    "DEALER CONTOH NUSANTARA",
			BankName:        "BANK CONTOH MANDIRI",
			AccountNumber:   "1000000005",
			MaxPercent:      "50",
			ReporterPIC:     "PIC Contoh Lima",
			ReporterEmail:   "pic.lima@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 5, Semarang",
			SubmittedBy:     "PICTEKNIKS",
			Committee:       "",
			Status:          masterautoclaim.StatusPending,
			ClientID:        "CLI005",
			ClientName:      "TERTANGGUNG CONTOH KELIMA",
		},
		{
			Initial:         "AGN006",
			ReceiverName:    "AGEN CONTOH PRATAMA",
			BankName:        "BANK CONTOH RAKYAT",
			AccountNumber:   "1000000006",
			MaxPercent:      "100",
			ReporterPIC:     "PIC Contoh Enam",
			ReporterEmail:   "pic.enam@contoh.example",
			ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
			ReceiverAddress: "Jalan Contoh Nomor 6, Makassar",
			SubmittedBy:     SampleCommittee,
			Committee:       SampleCommittee,
			Status:          masterautoclaim.StatusRejected,
			ClientID:        "CLI006",
			ClientName:      "TERTANGGUNG CONTOH KEENAM",
		},
	}
}

// SampleBusinessSources adalah lookup Sumber Bisnis contoh, meniru POOLDATA.AGENT.
//
// Keenam yang pertama sengaja sama dengan yang sudah ada di SampleList supaya penambahan
// atas kode yang sudah dipakai dapat dicoba — dan pesan "sudah ada di master" terlihat.
// Tiga terakhir belum dipakai, sehingga penambahan yang berhasil juga dapat dicoba.
func SampleBusinessSources() []masterautoclaim.BusinessSource {
	return []masterautoclaim.BusinessSource{
		{ID: "AGN001", Name: "MITRA CONTOH SEJAHTERA"},
		{ID: "AGN002", Name: "KOPERASI CONTOH BERSAMA"},
		{ID: "AGN003", Name: "MULTIFINANCE CONTOH ABADI"},
		{ID: "AGN004", Name: "BPR CONTOH LESTARI"},
		{ID: "AGN005", Name: "DEALER CONTOH NUSANTARA"},
		{ID: "AGN006", Name: "AGEN CONTOH PRATAMA"},
		{ID: "AGN007", Name: "PT. LEASING CONTOH UTAMA"},
		{ID: "AGN008", Name: "PT CONTOH ARTHA FINANSIAL"},
		{ID: "AGN009", Name: "BROKER CONTOH MANDIRI"},
	}
}

// SampleClients adalah lookup Client contoh, meniru POOLDATA.CLIENT.
func SampleClients() []masterautoclaim.Client {
	return []masterautoclaim.Client{
		{ID: "CLI001", Name: "TERTANGGUNG CONTOH PERTAMA"},
		{ID: "CLI002", Name: "TERTANGGUNG CONTOH KEDUA"},
		{ID: "CLI003", Name: "TERTANGGUNG CONTOH KETIGA"},
		{ID: "CLI005", Name: "TERTANGGUNG CONTOH KELIMA"},
		{ID: "CLI006", Name: "TERTANGGUNG CONTOH KEENAM"},
		{ID: "CLI007", Name: "PT. TERTANGGUNG CONTOH KETUJUH"},
	}
}

// SampleBanks adalah daftar bank contoh, meniru GENERAL.LST_BANK_GROUP.
//
// Nama-namanya sengaja BUKAN nama bank yang sesungguhnya. Bank contoh yang tidak ada di
// daftar ini dipakai menguji penolakan "harus dipilih dari daftar".
func SampleBanks() []masterautoclaim.Bank {
	return []masterautoclaim.Bank{
		{Code: "001", Name: "BANK CONTOH NIAGA"},
		{Code: "002", Name: "BANK CONTOH MANDIRI"},
		{Code: "003", Name: "BANK CONTOH RAKYAT"},
		{Code: "004", Name: "BANK CONTOH SYARIAH"},
	}
}
