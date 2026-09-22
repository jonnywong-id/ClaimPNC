import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HomePage } from '@/modules/home/HomePage'
import { AccountPage } from '@/modules/master-rekening/AccountPage'
import { ClaimStatusPage } from '@/modules/master-status-klaim/ClaimStatusPage'
import { ProgressStatusPage } from '@/modules/master-status-progres/ProgressStatusPage'
import { ProgressStatus2Page } from '@/modules/master-status-progres/ProgressStatus2Page'
import { AutoClaimPage } from '@/modules/master-auto-claim/AutoClaimPage'
import { WorkshopPage } from '@/modules/master-bengkel/WorkshopPage'
import { PanelPage } from '@/modules/master-panel/PanelPage'
import { SparepartPage } from '@/modules/master-sparepart/SparepartPage'
import { GroupingPage } from '@/modules/master-grouping-sparepart/GroupingPage'
import { PartCategoryPage } from '@/modules/master-kategori-sparepart/PartCategoryPage'
import { PartTypePage } from '@/modules/master-tipe-sparepart/PartTypePage'
import { ClausePage } from '@/modules/master-pasal-kerugian/ClausePage'
import { SupplierPage } from '@/modules/master-supplier/SupplierPage'
import { RejectionPage } from '@/modules/master-penolakan-klaim/RejectionPage'
import { LoginPage } from '@/modules/login/LoginPage'
import { ClaimHistoryPage } from '@/modules/riwayat-klaim/ClaimHistoryPage'
import { ClaimReportPage } from '@/modules/pelaporan-klaim/ClaimReportPage'
import { APIError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { useSession } from '@/app/session'

import { PageShell } from './PageShell'
import { SessionGuard } from './SessionGuard'
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
        Master Grouping Sparepart (MENU_ID 32). Master keempat dari keluarga alat berat.

        Yang dikelolanya BUKAN penggolongan suku cadang melainkan penautan suku cadang ke
        panel bodi pada sebuah kendaraan — baris yang menunjuk kendaraan yang sama
        dikumpulkan di bawah satu Nomor Grup.

        Tiga hal membedakannya dari ketiga master alat berat lain: ia memakai DUA tabel yang
        digabungkan INNER JOIN, kunci alaminya EMPAT KOLOM BERSAMA-SAMA alih-alih kolom yang
        masing-masing unik, dan lima isiannya DITURUNKAN dari Master Sparepart alih-alih
        diketik.
      */}
      <Route
        path="/master/grouping-sparepart"
        element={
          <SessionGuard>
            <Protected>
              <GroupingPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Kategori Sparepart (MENU_ID 33). Penggolongan suku cadang yang menjadi
        pilihan Kategori di layar Master Sparepart — layar ini MENULIS tabel yang layar itu
        hanya baca (P-1, satu tabel satu penulis).

        Tabelnya hanya punya TIGA kolom, dan itu menentukan seluruh bentuk layarnya: satu
        isian yang dapat diketik, tanpa kolom pencatat pelaku, tanpa stempel waktu, dan
        tanpa isian Catatan pada penolakan.

        Tombol Approve dan Reject ada DI DALAM layar ini dengan alasan yang sama seperti
        Master Bengkel, Panel, dan Sparepart: `Section/ApprovalMasterKategoriSparepartHE`
        di Pega dipakai Inbox Manager, dan Inbox Manager belum dibangun. Di sini akibat
        menundanya lebih berat — kategori yang tertahan tidak dapat dipakai sparepart mana
        pun.
      */}
      <Route
        path="/master/kategori-sparepart"
        element={
          <SessionGuard>
            <Protected>
              <PartCategoryPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Master Tipe Sparepart (MENU_ID 34). Penggolongan tingkat kedua di bawah kategori,
        dan master pertama di rumpun sparepart yang menyimpan KUNCI ASING — setiap tipe
        berinduk pada satu kategori yang dipilih dari dropdown.

        Tombol Approve dan Reject ada DI DALAM layar ini dengan alasan yang sama seperti
        Master Kategori Sparepart: `Section/ApprovalMasterTipeSparepartHE` di Pega dipakai
        Inbox Manager, dan Inbox Manager belum dibangun. Tipe yang tertahan tidak dapat
        dipakai sparepart mana pun.
      */}
      <Route
        path="/master/tipe-sparepart"
        element={
          <SessionGuard>
            <Protected>
              <PartTypePage />
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
        Pelaporan Klaim — modul proses klaim yang pertama, menggantikan harness
        `InboxRCVApp_Harness` yang di menu Pega berjudul "Inbox Laporan Klaim".

        Rutenya berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan menu —
        sistem lama membatasinya pada tujuh peran lewat When rule `IsReceivePNC` — adalah
        `TKT-F3-005` yang belum ada.
      */}
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
      {/*
        View History Claim — pencarian riwayat klaim, menggantikan harness
        `PNCSearchKlaim` (`MENU_ID 76`).

        Selain penjaga sesi, layar ini dijaga GERBANG PROTEKSI DATA di server: pengguna
        wajib terdaftar di Master Proteksi Data, dan satu jatah pencarian terpakai setiap
        kali layar dibuka. Penjaga di sini tetap sekadar kenyamanan tampilan.
      */}
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
        Master rekening berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan
        menu — siapa yang boleh membuka layar master mana — adalah TKT-F3-005 yang
        belum ada; sampai itu ada, setiap pengguna yang dapat masuk dapat membukanya.
      */}
      <Route
        path="/master-rekening"
        element={
          <SessionGuard>
            <div className="min-h-screen bg-white">
              <SessionWarning />
              <AccountPage />
            </div>
          </SessionGuard>
        }
      />
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
