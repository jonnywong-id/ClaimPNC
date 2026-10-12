// Package reportklaim adalah inti modul Report Klaim.
//
// # Layar apa ini
//
// Menu `MENU_ID 85` "Report Klaim" pada POOLDATA.M_MENU_APLIKASI_PNC, kelompok REPORT,
// yang menunjuk harness `PNCTATReport`. Judul yang dibaca pengguna di sistem lama adalah
// **"Report Claim"**.
//
// # Yang membuatnya berbeda dari modul lain
//
// Ia **bukan satu laporan**. `Harness/PNCTATReport-Harness.xml` memuat **28 panel
// laporan** yang berdiri sendiri-sendiri, masing-masing hanya berisi judul dan satu
// tombol Export. Tidak ada tabel hasil di layar sama sekali: setiap tombol memanggil satu
// activity yang menjalankan kueri, menyusun barisnya, lalu mengunduh **berkas CSV** lewat
// `pxConvertResultsToCSV`.
//
// Karena itu modul ini tidak punya "daftar" dalam arti modul inbox. Yang dimilikinya
// adalah **katalog laporan** — 28 entri yang menyebutkan judul panel, label tombolnya,
// penyaring yang ia pakai, nama berkas keluarannya, dan susunan kolomnya.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/PNCTATReport-Harness.xml   susunan 28 panel, judul, label tombol, penyaring
//	                                   bersama, dan parameter tetap tiap tombol
//	Activity/<24 activity>-Act.xml     kueri yang dijalankan, pemetaan kolom, susunan CSV
//	                                   beserta judul kolomnya, dan nama berkasnya
//	RDB List/<26 rule>-SQL.xml         SQL tiap laporan beserta penyaring yang diterimanya
//
// Daftar lengkap pasangan panel → activity → rule SQL ada di Catalog pada catalog.go;
// setiap entri menyebut berkas sumbernya satu per satu.
//
// # Penyaring bersama, dan nama propertinya yang menyesatkan
//
// Keempat penyaring di atas layar hidup pada halaman klipboard `TempLaporan` dan
// `TempAdjComp`. Nama propertinya **tidak mencerminkan isinya** — persis utang teknis
// alias kolom pada `03-CURRENT-ARCHITECTURE.md` §4.2:
//
//	label layar        properti Pega                     arti sebenarnya
//	Dari               TempLaporan.AnalystTransferDate   tanggal awal
//	Sampai             TempLaporan.DateOfLoss            tanggal akhir
//	Treaty             TempLaporan.StatusReceiver        LINI BISNIS, bukan treaty
//	Status Compliance  TempAdjComp.ComplianceStatus      status compliance
//
// Label layar **ditiru apa adanya** (`D-13`, keputusan Work Owner 2026-09-24): pengguna
// tidak perlu belajar ulang. Yang tidak ditiru adalah nama internalnya — di dalam kode ia
// bernama menurut isinya, dan ketidakcocokan itu dicatat di sini supaya penelusuran ke
// rule Pega tetap mungkin. Lihat BusinessLine pada filter.go.
//
// # Perilaku yang bergantung pada nama server TIDAK dibawa
//
// `Activity/PNCTATReport1_Act-Act.xml` bercabang pada
// `pxRequestor.pxReqServer=="pega.simasinsurtech.com"` untuk menambah pengecualian
// businesscode. Membandingkan nama server dilarang (`12-CROSSCUTTING.md` §3.4): ia tidak
// dapat diuji, tidak terlihat saat membaca kode, dan berubah diam-diam ketika server
// dipindahkan.
//
// Penggantinya adalah **portal yang sedang aktif** (`D-75`, `ADR-0030`) — entitas
// dinyatakan oleh pemanggil, bukan disimpulkan dari nama mesin. Lihat Filter.Entity.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	reportklaim/          katalog + aturan + seam       ← paket ini
//	reportklaim/usecase/  orkestrasi: pilih laporan, periksa penyaring, alirkan baris
//	reportklaim/repo/     pengisi seam penyimpanan      — sqlstore, memory
//	reportklaim/http/     lapisan transport modul ini   — handler, dto, rute, ekspor
//
// # Yang BUKAN urusan paket ini
//
// Isi klaimnya sendiri. Modul ini hanya MEMBACA — ia tidak menulis satu baris pun ke
// tabel mana pun, dan karena itu tidak pernah menjadi penulis sebuah tabel dalam arti
// `P-1`. Registrasi (`B-2`), akseptasi (`B-10`), komite (`B-7`), dan PLA/DLA (`B-9`)
// masing-masing punya modulnya sendiri; yang di sini hanyalah laporan atas hasilnya.
package reportklaim
