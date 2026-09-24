import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { ThresholdPage } from '@/modules/ambang-komite/ThresholdPage'
import { TieringPage } from '@/modules/ambang-komite/TieringPage'
import { HomePage } from '@/modules/home/HomePage'
import { AnalystDoctorPage } from '@/modules/inbox-analyst-doctor/AnalystDoctorPage'
import { CloseClaimPage } from '@/modules/inbox-close-claim/CloseClaimPage'
import { OutstandingPage } from '@/modules/inbox-outstanding/OutstandingPage'
import { AutoClaimInboxPage } from '@/modules/inbox-auto-claim/AutoClaimInboxPage'
import { ClaimReportFormPage } from '@/modules/inbox-laporan-klaim/ClaimReportFormPage'
import { InboxKomitePage } from '@/modules/inbox-komite/InboxKomitePage'
import { ClaimReportInboxPage } from '@/modules/inbox-laporan-klaim/ClaimReportInboxPage'
import { AccountPage } from '@/modules/master-rekening/AccountPage'
import { DominantFactorPage } from '@/modules/master-dominan-factor/DominantFactorPage'
import { CauseOfLossPage } from '@/modules/master-penyebab-kerugian/CauseOfLossPage'
import { MaskingPage } from '@/modules/master-masking/MaskingPage'
import { ClaimStatusPage } from '@/modules/master-status-klaim/ClaimStatusPage'
import { ProgressStatusPage } from '@/modules/master-status-progres/ProgressStatusPage'
import { TechnicianPage } from '@/modules/master-pic-teknik/TechnicianPage'
import { RecoveryPage } from '@/modules/master-recovery/RecoveryPage'
import { SurveyorPage } from '@/modules/master-surveyors/SurveyorPage'
import { SurveyorTypePage } from '@/modules/master-tipe-surveyors/SurveyorTypePage'
import { XOLPage } from '@/modules/master-xol/XOLPage'
import { LoginPage } from '@/modules/login/LoginPage'
import { ClaimTreatyNonPropPage } from '@/modules/inbox-claim-treaty-non-prop/ClaimTreatyNonPropPage'
import { ManagerReceivePUCLPage } from '@/modules/inbox-manager-receive-pucl/ManagerReceivePUCLPage'
import { KomunikasiCabangPage } from '@/modules/inbox-komunikasi-cabang/KomunikasiCabangPage'
import { RCLPUCLPage } from '@/modules/inbox-rcl-pucl/RCLPUCLPage'
import { SendtoRCLPUCLPage } from '@/modules/inbox-rcl-pucl/SendtoRCLPUCLPage'
import { ClaimTreatyPropPage } from '@/modules/inbox-claim-treaty-prop/ClaimTreatyPropPage'
import { InboxXOLPage } from '@/modules/inbox-xol/InboxXOLPage'
import { InboxProgressClaimPage } from '@/modules/inbox-progress-claim/InboxProgressClaimPage'
import { APIError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { useSession } from '@/app/session'

import { PageShell } from './PageShell'
import { SessionGuard } from './SessionGuard'
import { SessionWarning } from './SessionWarning'
import { ViewClaimPlaceholder } from './ViewClaimPlaceholder'

/**
 * Sesi yang ditolak server di tengah pekerjaan dibersihkan di satu tempat ini.
 *
 * Tanpa penanganan terpusat, setiap layar harus mengingat memeriksanya sendiri — dan
 * satu layar yang lupa akan menampilkan halaman kosong alih-alih mengembalikan pengguna
 * ke layar masuk.
 */
function handleSessionError(error: unknown): void {
  if (!(error instanceof APIError)) return
  if (error.kode === ErrorCode.invalidSession || error.kode === ErrorCode.sessionExpired) {
    useSession.getState().clear()
  }
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({ onError: handleSessionError }),
    mutationCache: new MutationCache({ onError: handleSessionError }),
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
      },
      mutations: { retry: false },
    },
  })
}

export function AppRoute() {
  return (
    <Routes>
      <Route path="/masuk" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <SessionGuard>
            <Protected>
              <HomePage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Modul berikutnya menempel sebagai satu baris di sini. Penjaga sesi adalah
        KENYAMANAN TAMPILAN; penegakan yang sebenarnya ada di server, yang memeriksa
        sesi pada setiap endpoint.
      */}
      <Route
        path="/master/status-progres-1"
        element={
          <SessionGuard>
            <Protected>
              <ProgressStatusPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/status-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ClaimStatusPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Laporan Klaim — modul bisnis pertama pada kelompok menu INBOX. Ia memakai
        kerangka yang sama dengan layar master, sehingga bilah atas, menu, dan pemilih
        portal tersedia di dalamnya.
      */}
      <Route
        path="/inbox/laporan-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ClaimReportInboxPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Auto Claim. Ia layar INBOX pertama yang dibangun — barisnya pekerjaan yang
        menunggu diproses, bukan data acuan (`D-79`).
      */}
      <Route
        path="/inbox-auto-claim"
        element={
          <SessionGuard>
            <Protected>
              <AutoClaimInboxPage />
              </Protected>
          </SessionGuard>
        }
      />

             {/*  
        Master Dominan Factor juga membaca basis data ENTITAS yang sedang dipilih.
        Akibat salah entitas di sini halus tetapi luas: keterangan faktor ikut terbaca
        laporan Outstanding per Cabang lewat LISTAGG, sehingga yang keliru bukan satu
        layar melainkan isi laporan yang dibaca manajemen.
      */}
      <Route
        path="/master/dominan-factor"
        element={
          <SessionGuard>
            <Protected>
              <DominantFactorPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Penyebab Kerugian — TINGKAT GOLONGAN saja (MENU_ID 20). Rinciannya
        (MENU_ID 38) butir menu tersendiri dan belum punya layar.

        Ia membaca basis data ENTITAS yang sedang dipilih. Akibat salah entitas di sini
        menjangkau lebih jauh daripada satu layar: keterangannya dibaca 19 rule Pega dan
        menjadi kolom PENGELOMPOKAN pada dasbor klaim per penyebab kerugian serta laporan
        XOL per bisnis.
      */}
      <Route
        path="/master/penyebab-kerugian"
        element={
          <SessionGuard>
            <Protected>
              <CauseOfLossPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Tipe Surveyors membaca basis data ENTITAS yang sedang dipilih, bukan basis
        data portal utama. Penjaga portalnya ada di server — layar hanya menuntun
        pengguna memilih lebih dulu.
      */}
      <Route
        path="/master/tipe-surveyor"
        element={
          <SessionGuard>
            <Protected>
              <SurveyorTypePage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Progress Claim — pemantauan progres klaim berjalan, menggantikan harness
        `ProgressClaim_Harness` (`MENU_ID 65`). Bagiannya bertumpuk, bukan bertab: itulah
        bentuknya di Pega.

        Bagian "Approval Progress Klaim" milik sistem lama tidak dibawa — ia satu-satunya
        bagian yang menulis, dan tabelnya masih dimiliki Pega selama masa berjalan
        paralel (keputusan Work Owner 2026-09-21).
      */}
      <Route
        path="/inbox-progress-claim"
        element={
          <SessionGuard>
            <Protected>
              <InboxProgressClaimPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Tujuan tombol "Lihat Detail Klaim". Layar sebenarnya adalah `MENU_ID 75`
        "View Claim" yang belum dibangun; rute ini menyatakan keadaan itu apa adanya
        alih-alih melempar pengguna ke beranda tanpa penjelasan.
      */}
      <Route
        path="/view-claim/:referensi"
        element={
          <SessionGuard>
            <Protected>
              <ViewClaimPlaceholder />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Surveyors — daftar ORANGNYA, anak dari Master Tipe Surveyors di atas.
        Membaca basis data ENTITAS yang sedang dipilih, dan barisnya memuat nama, alamat,
        telepon, surel, serta nama login aplikasi seseorang.
      */}
      <Route
        path="/master/surveyor"
        element={
          <SessionGuard>
            <Protected>
              <SurveyorPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master PIC Teknik juga membaca basis data ENTITAS yang sedang dipilih. Selain itu
        ia menembak direktori pegawai untuk mencari nama, dan alamat layanannya pun dibaca
        per entitas — dua alasan yang membuat portal wajib dipilih lebih dulu.
      */}
      <Route
        path="/master/pic-teknik"
        element={
          <SessionGuard>
            <Protected>
              <TechnicianPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Recovery juga membaca basis data ENTITAS yang sedang dipilih — dan di layar
        ini akibat salah entitas paling berat, karena yang ditampilkan memuat NOMOR
        REKENING VIRTUAL. Penjaga portalnya ada di server; layar hanya menuntun pengguna
        memilih lebih dulu.

        Berbeda dari butir master lain: layarnya FORM ENTRI, bukan pengelola data acuan.
        Sistem lama tidak punya cara membaca kembali batch yang sudah tercatat, dan itu
        ditiru apa adanya (keputusan Work Owner 2026-09-19).
      */}
      <Route
        path="/master/recovery"
        element={
          <SessionGuard>
            <Protected>
              <RecoveryPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Masking juga membaca basis data ENTITAS yang sedang dipilih, dan di layar
        inilah akibat salah entitas paling berat di antara seluruh butir master: yang
        ditampilkan adalah daftar SIAPA yang boleh membuka nomor KTP, surel, dan nomor
        telepon nasabah tanpa disamarkan (`R-20`). Penjaga portalnya ada di server; layar
        hanya menuntun pengguna memilih lebih dulu.

        Kewenangan menu — siapa yang boleh membuka layar ini — adalah TKT-F3-005 yang
        belum ada. Sampai itu ada, setiap pengguna yang dapat masuk dapat membukanya, dan
        itu berarti dapat memberi dirinya sendiri kewenangan membuka data pribadi. Dicatat
        terbuka di docs/keputusan-implementasi.md, bukan disembunyikan.
      */}
      <Route
        path="/master/masking"
        element={
          <SessionGuard>
            <Protected>
              <MaskingPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master XOL juga membaca basis data ENTITAS yang sedang dipilih, dan di layar ini
        akibat salah entitas menjalar paling jauh di antara butir master: struktur treaty
        menentukan pembagian klaim ke para reasuradur, sehingga limit dan share satu badan
        hukum yang tersimpan di badan hukum lain akan mengubah nilai yang dihitung PLA dan
        DLA sesudahnya (`R-20`). Penjaga portalnya ada di server; layar hanya menuntun
        pengguna memilih lebih dulu.

        Berbeda dari butir master lain: menyimpan di sini SEKALIGUS mengajukan struktur
        treaty ke komite — perilaku yang ditiru apa adanya dari layar lama.
      */}
      <Route
        path="/master/xol"
        element={
          <SessionGuard>
            <Protected>
              <XOLPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox XOL — akumulasi klaim per perjanjian Excess of Loss, pengganti harness
        `Inbox_XOL_Harness` (`MENU_ID 53`).

        Layar ini MEMBACA SAJA: keempat tabel yang ditulis sistem lama masih dimiliki
        Pega selama masa paralel (`P-1`), keputusan Work Owner 2026-09-20.

        Di sistem lama kedua tabnya dijaga access group yang berbeda — PncPICTeknik dan
        CaseManager. Pembedaan itu belum dapat ditegakkan (`TKT-F3-004`), sehingga setiap
        pengguna yang dapat masuk melihat keduanya.
      */}
      <Route
        path="/inbox-xol"
        element={
          <SessionGuard>
            <Protected>
              <InboxXOLPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Claim Treaty Prop — antrean klaim treaty proporsional, pengganti harness
        `InboxClaimTreaty_Harness` (`MENU_ID 54`).

        Layar ini MEMBACA SAJA: pembuatan klaim treaty menulis objek kerja di tabel yang
        selama masa paralel masih dimiliki Pega (`P-1`), keputusan Work Owner 2026-09-21.

        Di sistem lama ketiga antreannya dipisahkan KEADAAN pemanggil — apakah ia
        memegang akun antrean teknik, dan apakah Operator ID-nya terdaftar di
        POOLDATA.EMAILKOMITE. Pembedaan itu belum dapat ditegakkan (`TKT-F3-004`),
        sehingga setiap pengguna yang dapat masuk melihat ketiganya.
      */}
      <Route
        path="/inbox-claim-treaty-prop"
        element={
          <SessionGuard>
            <Protected>
              <ClaimTreatyPropPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Claim Treaty Non Prop — antrean klaim treaty NON-proporsional, pengganti
        harness `InboxClaimNonProp_Harness` (`MENU_ID 55`).

        Rutenya sengaja terpisah dari layar saudaranya di atas: keduanya membaca tabel,
        kolom, dan penanda objek kerja yang berbeda. Menyatukannya akan menampilkan
        antrean lini bisnis yang salah tanpa satu pun tanda di layar.

        Layar ini MEMBACA SAJA, dengan satu pengecualian yang tetap hanya membaca: tombol
        ekspor berfungsi penuh, karena menghasilkan berkas tidak menyentuh kepemilikan
        tabel (`P-1`). Pembuatan klaim tetap menolak dengan alasan.
      */}
      <Route
        path="/inbox-claim-treaty-non-prop"
        element={
          <SessionGuard>
            <Protected>
              <ClaimTreatyNonPropPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Outstanding menggantikan butir menu Pega "Inbox Outstanding", yang menunjuk
        `InboxOutstanding_Harness` — harness yang TIDAK ADA di export (`K-33`).

        Layar ini menampilkan SELURUH klaim yang masih berjalan pada satu entitas, bukan
        pekerjaan pemanggil, sehingga ia layar pemantauan dan bukan Inbox menurut `D-79`.

        Yang membatasi apa yang terlihat hanyalah lini bisnis pengguna — dan batas itu
        belum berlaku bagi pengguna yang kolom LINEBUSINESS-nya belum diisi. Pemeriksaan
        kewenangan menu adalah `TKT-F3-005` yang belum ada.
      */}
      <Route
        path="/inbox-outstanding"
        element={
          <SessionGuard>
            <Protected>
              <OutstandingPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Close Claim (`MENU_ID 59`) — KEBALIKAN TEPAT dari rute tepat di atasnya.

        Keduanya menyaring dua nilai `PYSTATUSWORK` yang SAMA dengan arah yang berlawanan:
        yang di atas `NOT IN`, yang ini `IN`. Rutenya karena itu terpisah dan tidak boleh
        disatukan — menunjuk keduanya ke satu layar akan menampilkan kebalikan dari yang
        diminta pengguna, tanpa satu pun tanda di layar.

        Harness-nya juga TIDAK ADA di export (`K-33`); yang dipakai adalah kueri, activity,
        dan section yang memang ada.

        Ia satu-satunya layar inbox yang MENULIS. Yang ditulisnya bukan klaim melainkan
        permintaan atas klaim — `P-1` menetapkan klaim masih ditulis Pega selama masa
        paralel. Pemeriksaan kewenangan menu tetap `TKT-F3-005` yang belum ada, dan di layar
        ini taruhannya lebih besar: kedua tombolnya menyentuh klaim yang sudah tutup.
      */}
      <Route
        path="/inbox-close-claim"
        element={
          <SessionGuard>
            <Protected>
              <CloseClaimPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Analyst Doctor — antrean penilaian medis milik SATU petugas, pengganti harness
        `inboxAnalystDoctor_Harness` (`MENU_ID 60`).

        Layar ini MEMBACA SAJA. Menyelesaikan tugasnya berarti menjalankan Flow Action
        `SendAnalystDoctor`, yang memindahkan penugasan — dan penugasan masih dimiliki Pega
        selama masa paralel (`P-1`).

        Pemeriksaan kewenangan menu tetap `TKT-F3-005` yang belum ada. Di layar ini
        akibatnya diredam penyaring identitas di server: antreannya disaring dengan Operator
        ID pemanggil, sehingga pengguna lain melihat layar kosong, bukan tugas medis orang
        lain. Itu peredam, bukan kendali — dan barisnya menyangkut data medis yang `FR-R2`
        batasi.
      */}
      <Route
        path="/inbox-analyst-doctor"
        element={
          <SessionGuard>
            <Protected>
              <AnalystDoctorPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Manager Receive / PUCL — pandangan penyelia atas DUA antrean sekaligus,
        pengganti harness `ReceiveDoucument_Harness` (`MENU_ID 56`).

        Layar ini MEMBACA SAJA, dengan satu pengecualian yang tetap hanya membaca: tombol
        ekspor berfungsi penuh, karena menghasilkan berkas tidak menyentuh kepemilikan
        tabel (`P-1`). Tindakan yang di Pega menulis — antara lain mencetak surat PUCL/RCL
        — menolak dengan alasan.

        Ia BERSAUDARA dekat dengan Inbox Outstanding tepat di atasnya, dan keduanya mudah
        tertukar: sama-sama layar pemantauan yang tidak menyaring menurut pemanggil. Yang
        membedakan adalah ISI antreannya — yang di atas seluruh klaim berjalan pada satu
        entitas, yang ini berkas penerimaan dokumen ditambah klaim RCL/PUCL. Rutenya karena
        itu terpisah.

        Berbeda dari seluruh layar inbox lain di berkas ini, TIDAK SATU PUN tabnya
        menyaring menurut pengguna yang login: Report Definition-nya menyaring unit
        organisasi, dan parameternya tidak pernah diisi di Pega. Sampai `TKT-F3-004`
        selesai, setiap pengguna yang dapat masuk melihat seluruh antrean portalnya —
        itulah sebabnya setiap pembukaannya dicatat di sisi peladen.
      */}
      <Route
        path="/inbox-manager-receive-pucl"
        element={
          <SessionGuard>
            <Protected>
              <ManagerReceivePUCLPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox RCL/PUCL (`MENU_ID 61`) — klaim yang ditolak atau diproses ulang.

        Ia BERSAUDARA dekat dengan layar tepat di atasnya, dan keduanya membaca antrean
        bersama yang SAMA. Yang membedakan adalah seberapa halus antrean itu dipartisi:

          Inbox Manager Receive / PUCL (56)  satu tab RCL/PUCL, tanpa penyaring halus —
                                             pandangan penyelia, superset layar ini
          layar ini (61)                     tiga tab menurut perjalanan surat PUCL,
                                             untuk petugas yang mengerjakannya

        Rutenya terpisah, dan tidak boleh disatukan: Pega pun punya dua menu dan dua
        harness untuk keduanya, ditujukan pada peran yang berbeda.

        Seperti saudaranya, TIDAK SATU PUN tabnya menyaring menurut pengguna yang login —
        penyaringnya akun antrean bersama, bukan orang. Di Pega, butir menunya dijaga
        `When/IsRCLPUCL-When.xml`: `(Administrators OR PncRCLPUCL) AND NOT ViewClaimPNC`.
        Aturan itu belum ditegakkan (`TKT-F3-004`), dan sampai saat itu setiap pembukaan
        dicatat di sisi peladen — termasuk rentang tanggal laporan hariannya.
      */}
      <Route
        path="/inbox-rcl-pucl"
        element={
          <SessionGuard>
            <Protected>
              <RCLPUCLPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Komunikasi Cabang (`MENU_ID 70`), pengganti harness `InboxKomunikasiCabang`.

        Ia SATU rute, bukan dua seperti RCL/PUCL: layar "Detail Komunikasi" di Pega bukan
        layar tujuan melainkan flow action yang menyisipkan section ke halaman yang sama,
        dan petugas kembali ke daftarnya begitu selesai membaca. Nomor percakapan yang
        sedang dibuka hidup di parameter alamat, sehingga alamatnya tetap dapat disalin.

        Berbeda dari modul inbox lain, daftar layar ini DISARING menurut cabang pemanggilnya
        — batas itu diselesaikan di sisi peladen dari login, bukan dari pilihan di layar.
        Petugas yang cabangnya tidak dapat diturunkan dilayani sebagai kantor pusat (`P-5`),
        dan layarnya menyatakan keadaan itu apa adanya.
      */}
      <Route
        path="/inbox-komunikasi-cabang"
        element={
          <SessionGuard>
            <Protected>
              <KomunikasiCabangPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Layar kerja satu klaim RCL/PUCL — section `SendtoRCLPUCL`, yang di Pega dibuka Open
        Assignment saat Nomor Case diklik.
        Ia rute TERSENDIRI, bukan panel di dalam antrean, karena di Pega pun ia layar tujuan:
        klaimnya terbuka pada tahap alur kerjanya untuk dikerjakan. Alamatnya karena itu dapat
        disalin dan dibuka kembali — dan `pzInsKey` di dalamnya wajib terkodekan, sebab kunci
        itu memuat spasi.
        Rute ini TIDAK dipakai modul lain. Enam inbox lain menuju `/view-claim/:referensi`,
        layar "View Claim" yang belum dibangun; RCL/PUCL berbeda karena layar tujuannya sudah
        diketahui — ketiga rule section-nya diterima 2026-09-24.
      */}
      <Route
        path="/inbox-rcl-pucl/klaim/:referensi"
        element={
          <SessionGuard>
            <Protected>
              <SendtoRCLPUCLPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Form Input Receive Document. Ia berdiri sebagai rute tersendiri, bukan modal di
        atas daftar: alamatnya dapat disalin dan dibuka kembali, dan itu yang dibutuhkan
        petugas yang menerima nomor berkas lewat telepon.
      */}
      <Route
        path="/inbox/laporan-klaim/:id"
        element={
          <SessionGuard>
            <Protected>
              <ClaimReportFormPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master rekening berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan
        menu — siapa yang boleh membuka layar master mana — adalah TKT-F3-005 yang
        belum ada; sampai itu ada, setiap pengguna yang dapat masuk dapat membukanya.
      */}
      <Route
        path="/master/rekening"
        element={
          <SessionGuard>
            <Protected>
              <AccountPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/ambang-komite"
        element={
          <SessionGuard>
            <Protected>
              <ThresholdPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Penjenjangan berada di bawah /komite, bukan /master, karena ia bukan data acuan
        melainkan aturan bisnis modul B-7. Tangga ambangnya milik F-4, cara membacanya
        milik B-7 — dan batas itu ikut terlihat di alamat halamannya.
      */}
      <Route
        path="/komite/penjenjangan"
        element={
          <SessionGuard>
            <Protected>
              <TieringPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Komite — menggantikan harness `InboxKomite_Harness`, MENU_ID 52.

        Ia berada di bawah /komite bersama penjenjangan, bukan di bawah /master: isinya
        pekerjaan dan keputusan, bukan data acuan. Batas kepemilikan itu ikut terlihat di
        alamat halamannya.

        Pemeriksaan kewenangan menu — di data contoh, MENU_ID 52 hanya diberikan kepada
        grup `IT` — adalah `TKT-F3-005` yang belum ada. Sampai itu ada, setiap pengguna
        yang dapat masuk dapat membukanya; yang membatasi isinya adalah penyaring pemilik
        di server, bukan rute ini.
      */}
      <Route
        path="/komite/inbox"
        element={
          <SessionGuard>
            <Protected>
              <InboxKomitePage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Jalur lama `/master-rekening` dipertahankan sebagai pengalihan, bukan dihapus.
        Ia sudah dipakai dan sudah tersimpan di riwayat peramban; membiarkannya mati
        akan menjawab tautan yang pernah sah dengan halaman beranda tanpa penjelasan.
      */}
      <Route path="/master-rekening" element={<Navigate to="/master/rekening" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

/**
 * Protected membungkus SELURUH layar di balik sesi dengan kerangka yang sama: bilah
 * atas, menu, identitas pengguna, tombol keluar, dan peringatan sesi.
 *
 * Satu pembungkus untuk semuanya, bukan satu per layar. Itu yang membuat tombol Keluar
 * dan nama pengguna hanya ada di satu tempat — sebelumnya keduanya hidup di dalam
 * halaman beranda, sehingga layar lain tidak punya cara keluar.
 */
function Protected({ children }: { children: ReactNode }) {
  return (
    <PageShell>
      <SessionWarning />
      {children}
    </PageShell>
  )
}

export function App() {
  // Klien dibuat sekali seumur hidup aplikasi; membuatnya ulang tiap render akan
  // membuang seluruh cache pada setiap perubahan state.
  const [client] = useState(createQueryClient)

  return (
    <QueryClientProvider client={client}>
      <BrowserRouter>
        <AppRoute />
      </BrowserRouter>
    </QueryClientProvider>
  )
}
