import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HomePage } from '@/modules/home/HomePage'
import { AutoClaimInboxPage } from '@/modules/inbox-auto-claim/AutoClaimInboxPage'
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
