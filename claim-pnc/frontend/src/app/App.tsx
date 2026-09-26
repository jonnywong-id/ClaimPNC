import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { ThresholdPage } from '@/modules/ambang-komite/ThresholdPage'
import { TieringPage } from '@/modules/ambang-komite/TieringPage'
import { HomePage } from '@/modules/home/HomePage'
import { InboxAdminPage } from '@/modules/inbox-admin/InboxAdminPage'
import { InboxCompliancePage } from '@/modules/inbox-compliance/InboxCompliancePage'
import { AutoClaimPage } from '@/modules/master-auto-claim/AutoClaimPage'
import { WorkshopPage } from '@/modules/master-bengkel/WorkshopPage'
import { PanelPage } from '@/modules/master-panel/PanelPage'
import { ClausePage } from '@/modules/master-pasal-kerugian/ClausePage'
import { ClauseAIPage } from '@/modules/master-pasal-ai/ClauseAIPage'
import { RejectionPage } from '@/modules/master-penolakan-klaim/RejectionPage'
import { SparepartPage } from '@/modules/master-sparepart/SparepartPage'
import { GroupingPage } from '@/modules/master-grouping-sparepart/GroupingPage'
import { PartCategoryPage } from '@/modules/master-kategori-sparepart/PartCategoryPage'
import { PartTypePage } from '@/modules/master-tipe-sparepart/PartTypePage'
import { ProgressStatus2Page } from '@/modules/master-status-progres/ProgressStatus2Page'
import { SupplierPage } from '@/modules/master-supplier/SupplierPage'
import { SurveyorLoginPage } from '@/modules/master-login/SurveyorLoginPage'
import { DetailPage as CauseOfLossDetailPage } from '@/modules/detail-penyebab-kerugian/DetailPage'
import { ReasMemberPage } from '@/modules/master-reas/ReasMemberPage'
import { InvestigatorInboxPage } from '@/modules/inbox-investigator/InvestigatorInboxPage'
import { ReceiveTKAInboxPage } from '@/modules/inbox-receive-tka/ReceiveTKAInboxPage'
import { ArchiveDocumentPage } from '@/modules/archive-dokumen-klaim/ArchiveDocumentPage'
import { ClaimHistoryPage } from '@/modules/riwayat-klaim/ClaimHistoryPage'
// Dua modul mengekspor komponen bernama sama, dan keduanya memang layar "penyebab
// kerugian" — yang satu varian Simas Online (MENU_ID 21), yang satu tingkat golongan
// (MENU_ID 20). Aliasnya di sini, bukan penggantian nama di modulnya, supaya nama di
// dalam tiap modul tetap sesuai layarnya sendiri.
import { CauseOfLossPage as SimasOnlineCauseOfLossPage } from '@/modules/master-col-simas-online/CauseOfLossPage'
import { DocumentTypePage } from '@/modules/daftar-tipe-dokumen/DocumentTypePage'
import { BusinessDocumentRulePage } from '@/modules/daftar-tipe-dokumen-bisnis/BusinessDocumentRulePage'
import { DetailDocumentTypePage } from '@/modules/daftar-detail-tipe-dokumen/DetailDocumentTypePage'
import { DocumentObjectPage } from '@/modules/daftar-objek-dokumen/DocumentObjectPage'
import { TravelDocumentDetailPage } from '@/modules/daftar-detail-dokumen-travel/TravelDocumentDetailPage'
import { TravelDocumentPage } from '@/modules/master-dokumen-travel/TravelDocumentPage'
import { AnalystDoctorPage } from '@/modules/inbox-analyst-doctor/AnalystDoctorPage'
import { CloseClaimPage } from '@/modules/inbox-close-claim/CloseClaimPage'
import { AcceptQueuePage } from '@/modules/inbox-accept-open-protection/AcceptQueuePage'
import { ProtectionListPage } from '@/modules/input-req-protection/ProtectionListPage'
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
import { SalvageInboxPage } from '@/modules/inbox-salvage/SalvageInboxPage'
import { RCLPUCLPage } from '@/modules/inbox-rcl-pucl/RCLPUCLPage'
import { LaporanHasilAIPage } from '@/modules/laporan-hasil-ai/LaporanHasilAIPage'
import { ReportKPIPage } from '@/modules/report-kpi/ReportKPIPage'
import { ReportKlaimPage } from '@/modules/report-klaim/ReportKlaimPage'
import { SendtoRCLPUCLPage } from '@/modules/inbox-rcl-pucl/SendtoRCLPUCLPage'
import { ClaimTreatyPropPage } from '@/modules/inbox-claim-treaty-prop/ClaimTreatyPropPage'
import { InboxXOLPage } from '@/modules/inbox-xol/InboxXOLPage'
import { InboxProgressClaimPage } from '@/modules/inbox-progress-claim/InboxProgressClaimPage'
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
        Master Pasal AI (MENU_ID 36). Kembaran layar di atas — section-nya Save-As darinya —
        tetapi sudah dipangkas menjadi layar PENCARIAN BACA-SAJA: hanya Cari dan Refresh,
        tanpa satu pun jalur tulis.

        Satu-satunya layar master yang paginasinya dikerjakan SERVER (25 baris). Itu bukan
        pilihan kami: grid Pega-nya ber-`pyPageMode = None`, dan jendelanya sudah dihitung
        activity lewat `FirstRow`/`LastRow` sejak dulu.

        Tabelnya `POOLDATA.MST_PASAL_AI`, dengan kolom `WP_PASAL`, `WP_AYAT`, dan
        `WP_KEJADIAN`. Ketiga nama itu baru terbaca setelah activity dan kedua Connect-SQL-nya
        diterima: properti yang mengikatnya di layar Pega bernama warisan — `.City`,
        `.CityID`, dan `.District` — sisa Save-As dari layar surveyor tahun 2017.
      */}
      <Route
        path="/master/pasal-ai"
        element={
          <SessionGuard>
            <Protected>
              <ClauseAIPage />
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
      {/*
        Master Login (MENU_ID 37). Satu butir menu, satu layar, TANPA tab — layar lamanya
        memang satu grid dengan dua tombol dan tidak punya penyaring status apa pun.

        Layar master paling sederhana di aplikasi ini, dan itu bukan kebetulan:
        POOLDATA.MST_LOGIN_SURVEYOR hanya punya tujuh kolom, dan tidak satu pun berupa
        APPROVAL, pencatat pelaku, stempel waktu, maupun penanda aktif. Akibatnya tidak ada
        alur persetujuan, tidak ada jejak siapa mengubah apa, dan tidak ada cara menyatakan
        sebuah login sudah tidak berlaku.

        Kuncinya DITURUNKAN dari Nama, bukan diterbitkan sequence — satu-satunya master di
        aplikasi ini yang begitu.
      */}
      {/*
        Master Reas (MENU_ID 35). Satu butir menu, satu layar, TANPA tab dan TANPA tombol
        simpan — harness lamanya memang satu grid dengan satu tombol Refresh, dan
        POOLDATA.T_REINSURER tidak punya kolom persetujuan.

        Satu-satunya layar master yang BACA-SAJA, dan itu keputusan berdasar bukti:
        satu-satunya penulis tabel itu di sistem lama adalah alur PLA/DLA lewat
        `Database/UPDATEREAS.prc` — dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`,
        bukan layar master ini.

        Rutenya berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan menu —
        `m_otorisasi_pnc.csv` membatasi MENU_ID 35 pada grup `IT` saja — adalah
        `TKT-F3-005` yang belum ada.
      */}
      <Route
        path="/master/reas"
        element={
          <SessionGuard>
            <Protected>
              <ReasMemberPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/detail-penyebab-kerugian"
        element={
          <SessionGuard>
            <Protected>
              <CauseOfLossDetailPage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/login"
        element={
          <SessionGuard>
            <Protected>
              <SurveyorLoginPage />
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
        Daftar Detail Tipe Dokumen (MENU_ID 41). JANGAN tertukar dengan dua tetangganya:
        /master/tipe-dokumen di atas (MENU_ID 40) adalah INDUKNYA, dan
        /master/tipe-dokumen-bisnis di bawah (MENU_ID 42) adalah tabel yang BERBEDA —
        ia justru merujuk ID baris layar ini lewat DOC_TYPE_DT_ID.
      */}
      <Route
        path="/master/detail-tipe-dokumen"
        element={
          <SessionGuard>
            <Protected>
              <DetailDocumentTypePage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/tipe-dokumen-bisnis"
        element={
          <SessionGuard>
            <Protected>
              <BusinessDocumentRulePage />
            </Protected>
          </SessionGuard>
        }
      />
      <Route
        path="/master/objek-dokumen"
        element={
          <SessionGuard>
            <Protected>
              <DocumentObjectPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Investigator (MENU_ID 48) — layar INBOX pertama, menggantikan harness
        `InboxInvestigator_Harness`.

        Isinya antrean bersama workbasket `InvestigatorPNC`. Ia inbox, bukan layar daftar:
        barisnya PEKERJAAN, hilang setelah dikerjakan, dan punya tenggat (`D-79`).

        Rutenya berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan menu — di
        sistem lama `When/IsInvestigator-When.xml` membatasinya pada access group
        `PncInvestigator` dan `Administrators` — adalah `TKT-F3-005` yang belum ada.
      */}
      <Route
        path="/inbox/investigator"
        element={
          <SessionGuard>
            <Protected>
              <InvestigatorInboxPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Inbox Receive TKA (MENU_ID 49) — layar INBOX kedua, menggantikan harness
        `InboxTKA_Harness`.

        Isinya klaim TKA yang tanggal penerimaan dokumen aslinya belum diisi. Berbeda dari
        Inbox Investigator yang baca-saja, layar ini MENULIS: pengguna mengisi tanggal
        langsung di dalam tabel lalu menekan Submit, dan barisnya hilang dari daftar — ciri
        kedua Inbox pada `D-79` yang di sini benar-benar terjadi lewat layar ini sendiri.

        Rutenya berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan menu adalah
        `TKT-F3-005` yang belum ada, dan untuk layar ini sistem lama tidak memberi petunjuk
        apa pun: tidak ada When rule yang menjaga MENU_ID 49.
      */}
      <Route
        path="/inbox/receive-tka"
        element={
          <SessionGuard>
            <Protected>
              <ReceiveTKAInboxPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        Dua layar berikut RUTENYA PERNAH ADA lalu hilang pada penggabungan cabang
        sebelumnya — `/riwayat-klaim` ada di commit efa135e, sementara modulnya ikut
        terbawa. Akibatnya keduanya menjadi layar yang lengkap
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
      {/*
        Inbox Compliance — MENU_ID 47, di bawah kelompok INBOX.

        Berbeda dari ketiga rute di atas, butir menunya SUDAH ada di
        POOLDATA.M_MENU_APLIKASI_PNC, sehingga layar ini dapat dicapai dari menu kiri
        begitu pemetaannya terdaftar di app/menu/registry.ts.
      */}
      <Route
        path="/inbox-compliance"
        element={
          <SessionGuard>
            <Protected>
              <InboxCompliancePage />
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
        Archive Dokumen Klaim — `MENU_ID 77`, di bawah kelompok VIEW.

        Satu rute untuk ketiga bagiannya. Bagian yang sedang dibuka adalah keadaan di
        dalam layar, bukan alamat tersendiri: layar lama pun menampakkan dan
        menyembunyikan ketiganya di satu halaman, dan memberi masing-masing alamat sendiri
        akan menjanjikan tautan-dalam yang isinya bergantung pada pencarian yang belum
        dijalankan.
      */}
      <Route
        path="/archive-dokumen-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ArchiveDocumentPage />
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

        Ia juga tujuan tombol "Lihat Detail Klaim" dan keenam inbox lain: rute ini
        menyatakan keadaan itu apa adanya alih-alih melempar pengguna ke beranda tanpa
        penjelasan. Kedua cabang menambahkannya sendiri-sendiri; di sini ia SATU rute.
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
        Input Req Protection — permintaan pembukaan proteksi beserta form inputnya.

        Layar lama dibatasi `When/IsReqProtection-When.xml` pada empat access group:
        PncAdmin, PncPICTeknik, PNCKomiteTeknik, dan Administrators. Pembatasan itu
        BELUM ada di sini; ia `TKT-F3-005`, yang bergantung pada tabel peran yang dapat
        dibangun tetapi belum dapat diisi.
      */}
      <Route
        path="/input-req-protection"
        element={
          <SessionGuard>
            <Protected>
              <ProtectionListPage />
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
        MENU_ID 82 "Laporan Hasil AI" — harness `Har_LaporanHasilAI`.

        Layar baca-saja yang menyandingkan penilaian AI dengan keputusan komitenya. Dua
        grid: ringkasan pencacah di atas, rincian baris di bawah.

        Tiga hal yang ditiru dari Pega dan mudah dikira kerusakan: lima kolom yang SELALU
        kosong, isian berlabel "Tgl Input" yang sebenarnya menyaring Tanggal Komite, dan
        "No Klaim" yang dikosongkan pada jenjang komite kedua ke atas. Ketiganya keputusan
        Work Owner 2026-09-26; alasannya ada di doc `LaporanHasilAIPage`.
      */}
      <Route
        path="/laporan-hasil-ai"
        element={
          <SessionGuard>
            <Protected>
              <LaporanHasilAIPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        MENU_ID 84 "Report KPI PNC" — harness `ReportKPIHarness`.

        CATATAN PEMULIHAN: rute ini sempat TERHAPUS pada 2026-09-25 oleh `git checkout`
        yang dijalankan sesi lain untuk membatalkan pemformatan ulang Prettier. Kodenya
        dipulihkan apa adanya; komentar aslinya tidak dapat dipulihkan utuh dan yang ada
        di sini ditulis ulang. Modulnya sendiri tidak pernah tersentuh.
      */}
      <Route
        path="/report-kpi"
        element={
          <SessionGuard>
            <Protected>
              <ReportKPIPage />
            </Protected>
          </SessionGuard>
        }
      />
      {/*
        MENU_ID 85 "Report Klaim" — harness `PNCTATReport`.

        Namanya menyesatkan: ia bukan layar laporan TAT melainkan halaman peluncur berisi
        28 panel laporan, yang REPORT TAT hanya salah satunya. Layarnya tidak menampilkan
        satu baris data pun — keluarannya berkas CSV.

        Tiga dari 28 panel TERHALANG dan tetap tampil bertanda sebabnya: Compliance
        (isinya properti klipboard Pega, bukan kolom), Adjuster (Report Definition-nya
        tidak ada di export), dan Mitra (penyaring barisnya menempuh DB Link `@ASMD`).
      */}
      <Route
        path="/report-klaim"
        element={
          <SessionGuard>
            <Protected>
              <ReportKlaimPage /></Protected>
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
        Inbox Salvage (`MENU_ID 71`), pengganti harness `InboxSalvage`.

        Ia layar pengelolaan barang sisa klaim, dan satu-satunya layar inbox yang MENULIS:
        tombol Tambah menyimpan pengajuan salvage ke `POOLDATA.PNC_SALVAGE` beserta detail
        itemnya. Kedua tabel itu dimiliki modul ini selama masa paralel, karena seluruh
        penulisnya di Pega adalah layar yang digantikannya (`P-1`).

        Daftar, halaman, dan kata kunci pencarian hidup di alamat — layar ini dibuka
        berpuluh kali sehari, dan pencariannya menyaring di server sehingga ia bagian dari
        apa yang sedang dilihat, bukan preferensi tampilan.
      */}
      <Route
        path="/inbox-salvage"
        element={
          <SessionGuard>
            <Protected>
              <SalvageInboxPage />
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
        Inbox Accept Open Protection — antrean akseptasi atas permintaan yang sama.

        Taruhannya lebih besar daripada layar di atas: di sini seseorang MENYETUJUI
        pembukaan proteksi. Layar lama membatasinya pada lima access group
        (`When/IsOpenProtectionPNC-When.xml`), dan memisahkan antrean PREMI khusus peran
        penagihan premi.

        Sampai TKT-F3-005 dikerjakan, yang tersisa sebagai kontrol hanyalah jejak
        DIAKSEP_OLEH — `D-59` menetapkan tidak ada pemisahan tugas formal.
      */}
      <Route
        path="/inbox-accept-open-protection"
        element={
          <SessionGuard>
            <Protected>
              <AcceptQueuePage />
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
