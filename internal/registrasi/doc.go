// Package registrasi adalah inti modul Registrasi Klaim (`B-2`) beserta alur yang
// menggerakkannya (`Flow/Register_Flow.xml`).
//
// # Isi paket ini
//
// Tiga hal, dan ketiganya adalah aturan bisnis murni:
//
//  1. ALUR — tahap klaim, percabangan, dan perpindahannya. Disalin dari
//     `Flow/Register_Flow.xml` sebagai DATA (lihat alur.go), bukan sebagai rangkaian
//     if-else yang tersebar. Bentuk data dipilih supaya diagram lama dan definisi baru
//     dapat dibandingkan baris per baris saat uji kesetaraan `S-8` dijalankan.
//  2. KLAIM — data yang dicatat petugas pada tahap Input Register, beserta empat konsep
//     status yang `ADR-0018` tetapkan tidak boleh digabung.
//  3. VALIDASI — gerbang terberat di seluruh sistem: 137 langkah
//     `Activity/InputRegister_act-Act.xml` yang dibaca ulang dan dinyatakan sebagai
//     aturan yang dapat diuji satu per satu.
//
// Paket ini juga mendeklarasikan SEAM-nya sendiri: `ClaimRepo`, `TaskRepo`,
// `PolicyRepo`, `NumberIssuer`, `Parameter`, `Kurs`, dan `Notifier`. Antarmuka
// dideklarasikan di paket yang memakainya, dan dipenuhi subpaket di bawahnya.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul (`platform/waktu`).
//
// # Susunan subpaket
//
//	registrasi/          aturan modul + seam          ← paket ini
//	registrasi/usecase/  orkestrasi: mulai, simpan, kembalikan, ambil tugas
//	registrasi/repo/     pengisi seam penyimpanan     — sqlstore, memori
//	registrasi/http/     lapisan transport modul ini  — handler, dto, rute
//
// # Yang BUKAN urusan paket ini
//
// Isi tahap selain Input Register. Alur Register melewati sembilan tahap; yang
// aturannya dimiliki modul ini hanya `Input Register`. Tahap `Input Estimasi` milik
// `B-5`, `Choose Surveyor` milik `B-8`, `RCL/PUCL`, `Compliance`, `Investigator`, dan
// `Analyst Doctor` milik `B-11`. Modul ini mengetahui KEBERADAAN tahap-tahap itu — ia
// harus, karena ia yang memindahkan klaim ke sana — tetapi tidak mengetahui isinya.
//
// Juga bukan urusan paket ini: snapshot polis (`B-1`, milik GISFW menurut `ADR-0006`),
// objek & coverage (`B-3`), spreading (`B-4`), pengiriman surel (`S-3`), dan otorisasi
// menu (`ADR-0023`).
package registrasi
