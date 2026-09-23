import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HomePage } from '@/modules/home/HomePage'
import { InboxAdminPage } from '@/modules/inbox-admin/InboxAdminPage'
import { AutoClaimPage } from '@/modules/master-auto-claim/AutoClaimPage'
import { WorkshopPage } from '@/modules/master-bengkel/WorkshopPage'
import { PanelPage } from '@/modules/master-panel/PanelPage'
import { ClausePage } from '@/modules/master-pasal-kerugian/ClausePage'
import { RejectionPage } from '@/modules/master-penolakan-klaim/RejectionPage'
import { SparepartPage } from '@/modules/master-sparepart/SparepartPage'
import { ProgressStatus2Page } from '@/modules/master-status-progres/ProgressStatus2Page'
import { SupplierPage } from '@/modules/master-supplier/SupplierPage'
import { ClaimReportPage } from '@/modules/pelaporan-klaim/ClaimReportPage'
import { ClaimHistoryPage } from '@/modules/riwayat-klaim/ClaimHistoryPage'
// Dua modul mengekspor komponen bernama sama, dan keduanya memang layar "penyebab
// kerugian" — yang satu varian Simas Online (MENU_ID 21), yang satu tingkat golongan
// (MENU_ID 20). Aliasnya di sini, bukan penggantian nama di modulnya, supaya nama di
// dalam tiap modul tetap sesuai layarnya sendiri.
import { CauseOfLossPage as SimasOnlineCauseOfLossPage } from '@/modules/master-col-simas-online/CauseOfLossPage'
import { DocumentTypePage } from '@/modules/daftar-tipe-dokumen/DocumentTypePage'
import { TravelDocumentDetailPage } from '@/modules/daftar-detail-dokumen-travel/TravelDocumentDetailPage'
import { TravelDocumentPage } from '@/modules/master-dokumen-travel/TravelDocumentPage'
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
import { APIError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { useSession } from '@/app/session'

import { PageShell } from './PageShell'
import { SessionGuard } from './SessionGuard'
import { ViewClaimPlaceholder } from './ViewClaimPlaceholder'
import { SessionWarning } from './SessionWarning'

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
      {/*
        Delapan rute berikut RUTENYA PERNAH ADA lalu hilang pada penggabungan cabang
        sebelumnya — seluruhnya terbaca di commit c1e3194, sementara modul beserta ujinya
        ikut terbawa ke sini. Akibatnya kedelapan layar itu lengkap tetapi tidak dapat
        dibuka sama sekali: setiap alamatnya jatuh ke rute `*`.

        Komentar di tiap blok di bawah dipulihkan APA ADANYA dari commit itu, bukan
        ditulis ulang — isinya merekam alasan rancangan yang tidak dapat disimpulkan
        kembali dari kode.

        Kedelapannya BELUM punya butir menu, sehingga belum dapat dicapai dari menu kiri.
        Menambahkannya ke menu menempuh POOLDATA.M_MENU_APLIKASI_PNC, bukan berkas ini.
      */}
      <Route
        path="/master/status-progres-2"
        element={
          <SessionGuard>
            <Protected>
              <ProgressStatus2Page />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Satu rute untuk DUA master — Penolakan Klaim dan Penolakan Komite — karena
        keduanya satu layar dan satu butir menu di Pega (MENU_ID 25). Pemilihannya tab di
        dalam layar, bukan dua rute.
      */}
      <Route
        path="/master/penolakan-klaim"
        element={
          <SessionGuard>
            <Protected>
              <RejectionPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Satu rute untuk EMPAT tab — Master Auto Klaim, Komite Approval, Waiting
        Approval, dan Reject — karena keempatnya satu layar dan satu butir menu di Pega
        (MENU_ID 26). Keempatnya hanya berbeda saringan atas tabel yang sama.
      */}
      <Route
        path="/master/auto-claim"
        element={
          <SessionGuard>
            <Protected>
              <AutoClaimPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Satu rute untuk TIGA tab — Approve, Waiting Approval, dan Reject — karena
        ketiganya satu layar dan satu butir menu di Pega (MENU_ID 28). Ketiganya hanya
        berbeda saringan atas tabel yang sama.

        Tombol Approve dan Reject ada DI DALAM layar ini, padahal di Pega keduanya ada di
        Inbox Manager (`Section/ApprovalMasterBengkelHE`). Inbox Manager belum dibangun,
        dan menunda keputusannya berarti setiap bengkel yang ditambah tertahan tanpa satu
        pun cara menyelesaikannya. Bentuk keputusannya sama persis — centang beberapa
        baris, satu tombol untuk seluruh pilihan.
      */}
      <Route
        path="/master/bengkel"
        element={
          <SessionGuard>
            <Protected>
              <WorkshopPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Panel (MENU_ID 30). Layar master pertama yang mengelola BARIS ANAK —
        daftar lokasi pada setiap panel, tersimpan di POOLDATA.LOKASI_PANEL_HE.

        Tombol Approve dan Reject ada DI DALAM layar ini dengan alasan yang sama seperti
        Master Bengkel: `Section/ApprovalMasterPanelHE` di Pega dipakai Inbox Manager,
        dan Inbox Manager belum dibangun.
      */}
      <Route
        path="/master/panel"
        element={
          <SessionGuard>
            <Protected>
              <PanelPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Sparepart (MENU_ID 31). Master ketiga dari keluarga alat berat, setelah
        Master Bengkel dan Master Panel; ketiganya berbagi satu activity persetujuan yang
        sama di Pega (`Activity/SetApprovalAllMaster`).

        Dua hal membedakannya: tabelnya PUNYA kolom pencatat pelaku (USER_UPDATE) dan
        stempel waktu (TGL_UPDATE_HARGA), dan ia TIDAK punya kolom alasan penolakan —
        sehingga layarnya tidak menggambar isian Catatan sama sekali.
      */}
      <Route
        path="/master/sparepart"
        element={
          <SessionGuard>
            <Protected>
              <SparepartPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Pasal Kerugian (MENU_ID 27). Layar pertama yang MENGHAPUS data secara
        permanen — `D-66` menetapkan soft delete menyeluruh, tetapi tabelnya tidak punya
        kolom penanda terhapus dan Work Owner memilih "jalankan as is" pada 2026-09-19.
      */}
      <Route
        path="/master/pasal-kerugian"
        element={
          <SessionGuard>
            <Protected>
              <ClausePage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Supplier (MENU_ID 29). TANPA tab — layar lamanya memang satu grid dengan
        tiga tombol, tanpa penyaring status apa pun.

        Dua hal yang membedakannya dari master lain, dan keduanya menyentuh uang:

        Seluruh isinya tinggal di SATU kolom JSONDATA. `M_SUPPLIER` hanya punya ID, OLDID,
        dan JSONDATA — tidak ada kembaran berkolom bernama seperti POOLDATA.BENGKEL_HE.

        Menonaktifkan supplier berlaku SEKETIKA, tanpa persetujuan siapa pun, sedangkan
        mengaktifkannya harus menunggu (`EditMasterSupplier_post` step 12). Sisi pemutus
        antreannya TIDAK ADA di export sama sekali (`R-16`), sehingga layar ini berhenti
        pada menyisipkan permintaannya — persis seperti sistem lama.
      */}
      <Route
        path="/master/supplier"
        element={
          <SessionGuard>
            <Protected>
              <SupplierPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/dokumen-travel"
        element={
          <SessionGuard>
            <Protected>
              <TravelDocumentPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/tipe-dokumen"
        element={
          <SessionGuard>
            <Protected>
              <DocumentTypePage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Tiga layar berikut RUTENYA PERNAH ADA lalu hilang pada penggabungan cabang
        sebelumnya — `/pelaporan-klaim` dan `/riwayat-klaim` ada di commit efa135e,
        sementara modulnya ikut terbawa. Akibatnya ketiganya menjadi layar yang lengkap
        beserta ujinya tetapi tidak dapat dibuka sama sekali: setiap alamatnya jatuh ke
        rute `*`.

        Ketiganya dipulihkan karena ujinya sendiri menyatakan maksud itu — masing-masing
        membuka alamat di atas lewat MemoryRouter dan menuntut layarnya muncul.

        Ketiganya BELUM punya butir menu, sehingga belum dapat dicapai dari menu kiri.
        Menambahkannya ke menu menempuh POOLDATA.M_MENU_APLIKASI_PNC, bukan berkas ini.
      */}
      <Route
        path="/inbox-admin"
        element={
          <SessionGuard>
            <Protected>
              <InboxAdminPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/pelaporan-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ClaimReportPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/riwayat-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ClaimHistoryPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Tujuan tombol "Lihat Detail Klaim" pada Inbox Admin. Layar rinciannya sendiri —
        `MENU_ID 75` "View Claim" — belum dibangun; yang dipasang di sini penampung yang
        MENAMPILKAN kunci yang diterimanya, sehingga menyalakan layar itu kelak tidak
        menuntut perubahan kontrak. Berkas penampungnya sudah ada; hanya rutenya yang
        hilang pada penggabungan sebelumnya.
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
        Daftar Detail Dokumen Travel (MENU_ID 39). JANGAN tertukar dengan
        /master/dokumen-travel di atas (MENU_ID 22) — yang ini master TURUNANNYA,
        atas V_LST_DOC_TRAVEL, dan merujuk DOCID milik yang itu.
      */}
      <Route
        path="/master/daftar-detail-dokumen-travel"
        element={
          <SessionGuard>
            <Protected>
              <TravelDocumentDetailPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/col-simas-online"
        element={
          <SessionGuard>
            <Protected>
              <SimasOnlineCauseOfLossPage />
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
