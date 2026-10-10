import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, GroupingStatus, type Grouping } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCreateGrouping, useGroupingList, useSaveGrouping } from './api'
import { GroupingForm, type GroupingFormValues } from './GroupingForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/PNCMasterGroupingSparepartHE-Section.xml` memuat tiga section yang ketiganya
 * membaca `GetDataMasterGrouping` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu:
 *
 *	MasterGroupingSparepartHEApprove   APPROVAL="1"
 *	MasterGroupingSparepartHEReject    APPROVAL="2"
 *	MasterGroupingSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Sparepart, Master Panel, dan
 * Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini, dan
 * `D-13` menuntut tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: GroupingStatus.disetujui,
    description: 'Grouping yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: GroupingStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: GroupingStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Ukuran halaman diambil dari `pyPageSizeOther` pada ketiga section tab.
 *
 * Ketiganya bernilai **15** — berbeda dari Master Sparepart yang 30, Master Bengkel yang 20,
 * dan Master Panel yang 15 pada satu tab dan 50 pada dua tab lainnya. Tidak ada satu angka
 * yang benar untuk seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 15

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar grouping tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar grouping tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Grouping Sparepart.
 *
 * Pengganti `Harness/GroupingSparePart_HE-Harness.xml` atas
 * POOLDATA.SPAREPART_HE_VIN_KEY beserta pendampingnya SPAREPART_HE_VIN_GROUP (MENU_ID 32).
 *
 * # Apa yang dikelola layar ini
 *
 * Penautan sebuah **suku cadang alat berat** ke sebuah **panel bodi** pada sebuah
 * **kendaraan**. Baris yang menunjuk kendaraan yang sama dikumpulkan di bawah satu **Nomor
 * Grup** — itulah arti kata "grouping" pada namanya, bukan penggolongan suku cadang.
 *
 * # Enam kolom, dan HANYA enam
 *
 * Grid Pega menampilkan ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, dan No
 * Rangka; kelima isian sisanya hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru apa
 * adanya, termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * **Nomor grup TIDAK digambar.** Ia memang ada sebagai kolom basis data
 * (`NO_GROUP_RANGKA`) dan tetap diterbitkan penyimpanan, tetapi layar Pega tidak
 * menampilkannya di mana pun — ia tersimpan pada properti `pyID` yang tidak pernah dirender.
 * Sempat digambar sebagai kolom tambahan di sini; **dicabut atas keputusan Work Owner
 * 2026-10-04**, dengan alasan yang sama yang berlaku untuk seluruh layar: ikuti Pega.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Di sistem lama status yang disimpan datang dari pemanggil (`Param.Approval`), dan layar
 * penyuntingannya selalu mengirim "0".
 *
 * # TIDAK ADA keputusan Approve/Reject di layar ini — dan itu memang Pega
 *
 * Ketiga tabnya BENAR ada di Pega: `Section/PNCMasterGroupingSparepartHE` merujuk tepat tiga
 * caption — `pyCaption Approve`, `pyCaption Reject`, dan `pyCaption Waiting Approval`.
 *
 * Yang TIDAK ada di sana adalah cara memutuskannya. Seluruh tombol yang dirujuk layar ini
 * hanya **SIMPAN** dan **Ubah**, dan `pySelected` maupun `pxCheckbox` **nol kemunculan** di
 * kelima section-nya. Keputusan hidup di layar lain —
 * `Section/ApprovalPNCMasterGroupingSparepartHE`, yang disertakan `Harness/UserInbox_Harness`
 * dan `Section/InboxManager_Sec`, yakni **Inbox Manager**.
 *
 * Tab Waiting Approval di sini karena itu murni **daftar**: ia memperlihatkan apa yang sedang
 * menunggu, dan penyuntingannya memakai tombol Ubah yang sama dengan kedua tab lain.
 *
 * Sempat ada centang borongan beserta tombol "Approve terpilih" dan "Reject terpilih" di sini,
 * meniru Master Bengkel, Master Panel, dan Master Sparepart. **Dicabut atas keputusan Work
 * Owner 2026-10-04.** Catatan lama yang menyebut layar persetujuan Pega menyediakan "Select
 * All, Deselect All, Approve, Reject" **salah untuk modul ini** — itu benar untuk ketiga master
 * alat berat lain yang memakai `Activity/SetApprovalAllMaster`, dan ditulis di sini tanpa
 * diperiksa ulang. Layar persetujuan modul ini memutuskan **satu baris pada satu waktu**:
 * tombolnya `Approve`, `Reject`, dan `DETAILS`, tanpa satu pun centang.
 */
export function GroupingPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Grouping | null>(null)

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useGroupingList(active.status)
  const create = useCreateGrouping()
  const save = useSaveGrouping()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.grouping ?? []

  function closeForm() {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(null)
  }

  function openAdd() {
    create.reset()
    save.reset()
    setEditing(null)
    setAdding(true)
  }

  function openEdit(row: Grouping) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
  }

  function submit(values: GroupingFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan membuang
    // isian pengguna saat penyimpanan gagal.
    if (editing) {
      save.mutate({ id: editing.id_grouping, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — keenamnya berdampingan pada urutan yang
    sama: ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, No Rangka.

    Judulnya pun diambil dari grid, bukan dari form: grid menulis "No Sparepart" dan "Sisi
    Panel" sementara form menulis "Nomor Sparepart" dan "Sisi". Keduanya dipertahankan di
    tempatnya masing-masing (D-13).

    Susunannya SAMA pada ketiga tab, termasuk Waiting Approval — ketiga section tab Pega
    menyebut keenam judul yang sama persis, dan tidak satu pun punya kolom centang.

    Kolom "Aksi" adalah satu-satunya judul yang tidak menyalin Pega; ia berlaku di seluruh
    modul aplikasi ini.
  */
  const columns: Column<Grouping>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '6rem',
      value: (row) => row.id_grouping,
    },
    {
      key: 'nomor',
      title: 'No Sparepart',
      width: '10rem',
      value: (row) => row.nomor_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Sparepart',
      width: '14rem',
      value: (row) => row.nama_sparepart,
      render: (row) => (
        <span className="text-sm text-slate-900">{row.nama_sparepart || '—'}</span>
      ),
    },
    {
      key: 'panel',
      title: 'Nama Panel',
      width: '12rem',
      value: (row) => row.nama_panel,
    },
    {
      key: 'sisi',
      title: 'Sisi Panel',
      width: '7rem',
      // Yang dicari dan diurutkan adalah sebutannya, bukan sandinya: pengguna mencari "KIRI",
      // bukan "1".
      value: (row) => row.sisi_label,
      render: (row) => <span className="text-sm text-slate-900">{row.sisi_label || '—'}</span>,
    },
    {
      key: 'rangka',
      title: 'No Rangka',
      width: '13rem',
      value: (row) => row.no_rangka,
      render: (row) => (
        <span className="font-mono text-xs text-slate-900">{row.no_rangka || '—'}</span>
      ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari Section/MasterGroupingSparepartHE — "HE" ikut, karena itulah
              yang tertulis di layar lama (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Grouping Sparepart HE</h1>
          <p className="text-sm text-slate-600">
            Penautan suku cadang ke panel bodi pada sebuah kendaraan, dikelompokkan menurut nomor
            rangka.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* Kedua tombolnya diambil dari `pyLabel` pada Section/MasterGroupingSparepartHE:
              "Tambah" dan "Refresh". */}
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      <nav
        aria-label="Tab Master Grouping Sparepart"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-current={tab === t.id ? 'page' : undefined}
            onClick={() => {
              closeForm()
              setTab(t.id)
            }}
            className={[
              'rounded-t px-3 py-2 text-sm font-medium',
              'transition-colors duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
              tab === t.id
                ? 'border-b-2 border-blue-600 text-blue-700'
                : 'text-slate-500 hover:text-slate-800',
            ].join(' ')}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        {active.description}{' '}
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {isFormOpen && (
        <section className="mt-5">
          <GroupingForm
            editing={editing}
            isSaving={create.isPending || save.isPending}
            error={editing ? save.error : create.error}
            onSave={submit}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar grouping…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(row) => row.id_grouping}
            description="Sumber: POOLDATA.SPAREPART_HE_VIN_KEY + SPAREPART_HE_VIN_GROUP"
            searchLabel="Cari grouping"
            pageSize={PAGE_SIZE}
            emptyMessage={`Belum ada grouping pada tab ${active.label}.`}
          />
        )}
      </section>
    </main>
  )
}
