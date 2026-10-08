import { useState, useMemo } from 'react'

import { WorkshopStatus, type Workshop } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useApprovalTabs } from '@/components/masterpage/useApprovalTabs'
import { useCrudForm } from '@/components/masterpage/useCrudForm'
import { ApprovalPanel } from '@/components/masterpage/ApprovalPanel'
import { AddButton, ListHeader, RefreshButton } from '@/components/masterpage/MasterPage'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { selectColumn } from '@/components/ApprovalControls'

import { useCreateWorkshop, useDecideWorkshop, useSaveWorkshop, useWorkshopList } from './api'
import { DocumentPanel } from './DocumentPanel'
import { WorkshopForm } from './WorkshopForm'
import { collectKnownValues } from '@/components/masterpage/knownValues'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * Caption-nya dibaca dari `Section/BrowseMasterHE-Section.xml` dan urutannya dari layar
 * Pega yang berjalan: **Approve · Reject · Waiting Approval**. Ketiganya membaca
 * `BrowseBengkelHE_RD` yang sama, berbeda hanya pada nilai APPROVAL-nya — yang di Pega
 * dikirim sebagai parameter report definition (`pyReportDefParams`, `APPROVAL="1"` pada
 * tab Approve).
 *
 * Urutannya bukan detail kosmetik: tab pertama adalah yang terbuka saat layar dibuka, dan
 * "Reject" di tengah berarti petugas yang terbiasa menekan tab kedua akan menekan tab yang
 * berbeda bila urutannya diubah.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: WorkshopStatus.disetujui,
    description: 'Bengkel yang sudah disetujui dan berlaku sebagai rekanan.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: WorkshopStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: WorkshopStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

/**
 * Banyaknya baris per halaman.
 *
 * Dibaca dari `pyPageSizeOther` pada `Section/BrowseMasterHEApprove-Section.xml`, yang
 * bernilai `20` karena `pyPageSize` di section yang sama bernilai `"Other"`. Ia angka
 * milik LAYAR INI — layar master lain memakai 15 — sehingga ia disebut di sini, bukan
 * dijadikan bawaan komponen tabel.
 */
const PAGE_SIZE = 20

/** Kolom yang nilai sahnya tidak ada di export; sarannya dikumpulkan dari data. */
const SUGGESTED_COLUMNS = [
  'status_bengkel',
  'jenis_pph',
  'status_disupply_asm',
  'status_eklaim',
  'status_auto_aksep',
  'status_payment',
  'status_autopayment',
  'status_tekno',
  'status_order',
] as const

/**
 * Keterangan di dalam grid saat tidak ada satu baris pun tergambar.
 *
 * # Gagal dan kosong sengaja tampil sama
 *
 * Layar ini TIDAK membedakan "tabelnya memang kosong" dari "tabelnya gagal dibaca".
 * Keduanya tampil identik. Pada saat tulisan ini dibuat, view `POOLDATA.BENGKEL_HE`
 * sedang rusak di Oracle (`ORA-04063`, karena `ACCOUNT_ID` pada tabel dasarnya tidak
 * lagi dikenali), dan layar ini menampilkannya sebagai data kosong.
 *
 * Keputusan Work Owner (2026-09-28), mengikuti keputusan yang sama untuk Master
 * Sparepart pada 2026-09-24. Yang menggantikan pembedaan itu ada di dua tempat yang
 * TIDAK dilihat pengguna:
 *
 *	log backend          setiap kegagalan tercatat lengkap dengan galat Oracle-nya
 *	claimpnc -periksa    menyebut objek dan galatnya, beserta kueri katalog penjawabnya
 *
 * Akibat yang diterima secara sadar: pengguna tidak punya cara membedakan tab yang
 * memang belum berisi dari basis data yang sedang rusak, sehingga kerusakan berikutnya
 * hanya terbaca dari log — bukan dari layar.
 *
 * Satu pengecualian yang dipertahankan: sebelum portal dipilih, kuerinya belum pernah
 * dijalankan sama sekali — tidak ada "hasil nol" untuk dilaporkan, dan yang dibutuhkan
 * pengguna adalah petunjuk tindakan, bukan keterangan data.
 */
function emptyMessageFor(hasPortal: boolean): string {
  if (!hasPortal) {
    return 'Pilih portal entitas di bagian atas halaman untuk menampilkan daftarnya.'
  }
  return 'Data tidak ada'
}

/**
 * Layar Master Bengkel.
 *
 * Pengganti `Harness/BengkelHE-Harness.xml` atas tabel POOLDATA.BENGKEL_HE (MENU_ID 28).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **bengkel rekanan** beserta syarat kerja samanya: ke mana pembayarannya dikirim,
 * berapa diskon jasa dan sparepart yang berlaku, pajak apa yang dipotong, berapa SLA-nya,
 * dan kanal mana saja yang boleh dipakai bengkel itu. Sebagian kolomnya menentukan hasil
 * hitungan uang pada klaim yang memakai bengkel tersebut — ia master komersial, bukan
 * sekadar daftar alamat.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateBengkelHE_act` step 7 menetapkan `APPROVAL := "0"` tanpa syarat apa
 * pun. Bengkel yang sudah disetujui lalu disunting kembali menunggu — dan itu benar untuk
 * master yang menentukan diskon, pajak, dan rekening tujuan pembayaran.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterBengkelHE` dipakai
 * `InboxManager_Sec`, dan keputusannya dijalankan `Activity/SetApprovalAllMaster` yang
 * melayani bengkel, panel, dan sparepart sekaligus.
 *
 * Inbox Manager belum dibangun. Menunda keputusannya sampai layar itu ada berarti setiap
 * bengkel yang ditambah tertahan di Waiting Approval tanpa satu pun cara menyelesaikannya
 * — dan alur ini tidak dapat dicoba sama sekali. Yang dipakai sebagai gantinya adalah
 * BENTUK yang sama persis: centang beberapa baris, lalu satu tombol untuk seluruh
 * pilihan. Memindahkannya ke Inbox Manager kelak hanya soal letak, bukan soal perilaku.
 */
export function WorkshopPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')

  // Bengkel yang panel dokumennya sedang terbuka. Null berarti tertutup.
  const [documentFor, setDocumentFor] = useState<Workshop | null>(null)

  const list = useWorkshopList(active.status)
  const create = useCreateWorkshop()
  const save = useSaveWorkshop()
  const form = useCrudForm(create, save, (row: Workshop, input) => ({ id: row.id_bengkel, input }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd } = form
  const decide = useDecideWorkshop()

  const rows = list.data?.bengkel ?? []

  /*
    Saran nilai untuk kolom yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia
    jawaban terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" —
    lihat WorkshopForm bagian "Penanda sistem".
  */
  const knownValues = useMemo(() => collectKnownValues(rows, SUGGESTED_COLUMNS), [rows])

  /*
    Panel dokumen dan form saling menutup.

    Keduanya menggarap baris yang sama dan keduanya digambar di tempat yang sama. Dibiarkan
    terbuka bersamaan, pengguna melihat dua panel bertumpuk yang tidak jelas mana yang
    sedang dikerjakannya.
  */
  function openDocument(row: Workshop) {
    closeForm()
    setDocumentFor(row)
  }

  function openEdit(row: Workshop) {
    form.openEdit(row)
    // Panel dokumen ikut ditutup — lihat alasannya pada openDocument.
    setDocumentFor(null)
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_bengkel: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA.

    `Section/BrowseMasterHEApprove-Section.xml` menyetel `pyColumnCount = 7`, dan keenam
    kolom datanya muncul pada urutan ini di dalam definisi gridnya:

      ID_BENGKEL       ID Bengkel        <- kolom pertama, pySortType=DESC pySortOrder=1
      NAMA_BENGKEL     Nama Bengkel
      ALM_BENGKEL      Alamat Bengkel
      TELP_BENGKEL     Telp Bengkel
      NOHP_BENGKEL     No HP Bengkel
      LOGIN_APLIKASI   Login Aplikasi
      (aksi)           Ubah

    Ketiga puluh empat kolom selebihnya ADA di tabel tetapi TIDAK di grid — keseluruhannya
    hanya muncul di form. Menampilkan lebih banyak "supaya informatif" berarti membuat
    layar yang berbeda dari yang dipakai petugas hari ini (`D-13`).

    SATU kolom ditambahkan, dan hanya pada tab Waiting Approval: kotak centang untuk
    keputusan borongan. Ia bagian dari keputusan yang sudah dicatat — lihat doc comment
    WorkshopPage — dan tidak muncul di kedua tab lain.
  */
  const columns: Column<Workshop>[] = [
    ...selectColumn<Workshop>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_bengkel,
      nameOf: (row) => row.nama_bengkel,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
    {
      key: 'id_bengkel',
      title: 'ID Bengkel',
      width: '10rem',
      value: (row) => row.id_bengkel,
    },
    {
      key: 'nama_bengkel',
      title: 'Nama Bengkel',
      value: (row) => row.nama_bengkel,
    },
    {
      key: 'alamat_bengkel',
      title: 'Alamat Bengkel',
      value: (row) => row.alamat_bengkel,
    },
    {
      key: 'telp_bengkel',
      title: 'Telp Bengkel',
      width: '10rem',
      value: (row) => row.telp_bengkel,
    },
    {
      key: 'nohp_bengkel',
      title: 'No HP Bengkel',
      width: '10rem',
      value: (row) => row.nohp_bengkel,
    },
    {
      key: 'login_aplikasi',
      title: 'Login Aplikasi',
      width: '11rem',
      value: (row) => row.login_aplikasi,
      render: (row) => <LoginCell row={row} />,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '11rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <div className="flex justify-end gap-2">
          <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
            Ubah
          </Button>
          {/*
            Padanan dua tombol layar lama sekaligus — "Upload Document" dan tombol lihat
            dokumen pada layar persetujuan. Letaknya PER BARIS, bukan tingkat layar:
            yang disimpannya adalah DOKUMENID, kolom milik satu baris. Alasannya lengkap
            di DocumentPanel.
          */}
          <Button tone="halus" onClick={() => openDocument(row)}>
            Dokumen
          </Button>
        </div>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      {/* "Master Bengkel HE", bukan "Master Bengkel".

          Itu judul yang tertulis di layar Pega — caption
          `Section/MasterBengkelHE-Section.xml`. Butir menunya memang bernama "Master
          Bengkel" (`M_MENU_APLIKASI_PNC.MENU_DESC`), dan keduanya memang berbeda di
          sistem lama; `D-13` menuntut teks LAYAR yang diikuti. */}
      <ListHeader
        title="Master Bengkel HE"
        description="Bengkel rekanan beserta syarat kerja samanya — rekening pembayaran, diskon, pajak, dan SLA."
      >
        <RefreshButton query={list} />
        <AddButton onClick={openAdd} disabled={isFormOpen} />
      </ListHeader>

      <ApprovalPanel
        tabLabel="Tab Master Bengkel"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={closeForm}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="bengkel"
        barNoun="bengkel"
        chosenCount={chosen.size}
        onApprove={() => runDecision(WorkshopStatus.disetujui)}
        onReject={() => runDecision(WorkshopStatus.ditolak)}
        onClear={() => setChosen(new Set())}
      />

      {documentFor && (
        <DocumentPanel workshop={documentFor} onClose={() => setDocumentFor(null)} />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <WorkshopForm
            editing={editing}
            knownValues={knownValues}
            isSaving={form.isSaving}
            error={form.saveError}
            onSave={form.submit}
            onCancel={closeForm}
          />
        </section>
      )}

      {/*
        Tabelnya digambar dalam SETIAP keadaan — termuat, kosong, gagal, bahkan sebelum
        portal dipilih. Kolomnya karena itu selalu terlihat, persis seperti grid Pega yang
        menggambar kepala kolomnya beserta pyGridNoResultsMessage di bawahnya.

        Tidak ada lagi kotak galat yang menggantikan tabelnya: keterangan apa pun tinggal
        di dalam grid lewat emptyMessage. Lihat emptyMessageFor untuk alasan gagal dan
        kosong sengaja tampil sama.
      */}
      <section className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_bengkel}
          description="Sumber: POOLDATA.BENGKEL_HE"
          searchLabel="Cari bengkel"
          pageSize={PAGE_SIZE}
          isLoading={portal !== null && list.isPending}
          showHeaderWhenEmpty
          emptyMessage={emptyMessageFor(portal !== null)}
        />
      </section>
    </main>
  )
}

/**
 * LoginCell menggambar kolom Login Aplikasi.
 *
 * Isinya nilai kolom apa adanya — persis seperti grid Pega, yang juga mengosongkannya
 * bila belum ada.
 *
 * # Satu keterangan yang DITAMBAHKAN, dan hanya pada satu keadaan
 *
 * Bengkel berstatus **rekanan** yang login aplikasinya kosong ditandai. Di Pega keadaan
 * itu terlihat sebagai sel kosong yang tidak berbeda dari sel kosong milik bengkel
 * non-rekanan — padahal artinya jauh berbeda: yang satu memang tidak diberi login, yang
 * lain **seharusnya punya tetapi tidak**, dan bengkelnya tidak akan pernah dapat masuk.
 *
 * Kolom "Rekanan" tersendiri sempat ada di layar ini lalu dicabut, karena grid Pega tidak
 * punya kolom itu. Keterangan ini yang menggantikannya: ia menyampaikan hal yang sama
 * pada baris yang benar-benar memerlukannya, tanpa menambah kolom.
 */
function LoginCell({ row }: Readonly<{ row: Workshop }>) {
  if (row.login_aplikasi !== '') {
    return <span>{row.login_aplikasi}</span>
  }
  if (!row.rekanan) {
    return <span className="text-slate-400">—</span>
  }
  return (
    <span className="inline-flex flex-col">
      <span className="text-slate-400">—</span>
      <span className="text-xs text-amber-700">rekanan tanpa login</span>
    </span>
  )
}
