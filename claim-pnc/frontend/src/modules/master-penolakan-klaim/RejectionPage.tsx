import { useState, type ReactNode } from 'react'
import type { UseQueryResult } from '@tanstack/react-query'

import type { CommitteeRejection, Rejection } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'
import { retryLoadMessage, type MessageContent } from '@/components/masterpage/loadMessage'
import {
  AddButton,
  RefreshButton,
  editColumn,
  renderListState,
} from '@/components/masterpage/MasterPage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'

import {
  useCreateRejection,
  useRejectionList,
  useRejectionParentList,
  useUpdateRejection,
} from './api'
import {
  useCommitteeRejectionList,
  useCreateCommitteeRejection,
  useUpdateCommitteeRejection,
} from './apiKomite'
import { RejectionForm, toInput, type RejectionFields } from './RejectionForm'
import { CommitteeRejectionForm, type CommitteeRejectionFields } from './CommitteeRejectionForm'

/**
 * Dua tab layar ini, mengikuti dua tombol layar lama.
 *
 * `Section/BrowseNoteRejectClaim-Section.xml` berpindah isi lewat
 * `Activity/SetStatusMasterRejectsKlaim-Act.xml`, yang menyetel
 * `FlgMasterPenolakan.FlagASO` menjadi 1 atau 2. Nilai di bawah sengaja dinamai menurut
 * ARTINYA, bukan menurut angkanya — angka itu detail penyimpanan klipboard Pega, bukan
 * kontrak apa pun.
 */
type Tab = 'klaim' | 'komite'

function loadMessage(error: unknown): MessageContent {
  return retryLoadMessage(error)
}

/**
 * formatDate menuliskan waktu UTC dari server sebagai tanggal WIB.
 *
 * Konversi zona waktu terjadi DI SINI, di tempat waktu ditampilkan — satu-satunya tempat
 * yang boleh melakukannya (`08-TECHNICAL-STRATEGY.md` §4.4). Tidak ada penambahan 7 jam
 * manual di mana pun; `Asia/Jakarta` disebut namanya supaya hasilnya tidak bergantung pada
 * zona waktu mesin pengguna.
 */
function formatDate(value: string): string {
  if (value === '') return '—'
  const moment = new Date(value)
  if (Number.isNaN(moment.getTime())) return '—'
  return moment.toLocaleDateString('id-ID', {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })
}

/** Nada lencana status, mengikuti arti ketiga nilai kolom STATUS. */
function statusClass(status: string): string {
  switch (status) {
    case '1':
      return 'bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200'
    case '2':
      return 'bg-red-50 text-red-800 ring-1 ring-red-200'
    default:
      return 'bg-slate-100 text-slate-700 ring-1 ring-slate-200'
  }
}

/**
 * Layar Master Penolakan Klaim.
 *
 * Pengganti `Harness/PNC_MasterTolakKlaim-Harness.xml` — satu butir menu (MENU_ID 25) yang
 * mengelola DUA master dengan tabel yang sama sekali berbeda:
 *
 *   Penolakan Klaim   POOLDATA.MST_PENOLAKAN_KLAIM_1 + _2
 *   Penolakan Komite  POOLDATA.MST_REJECTED_KOMITE
 *
 * Judul, susunan kolom, dan kedua tabnya mengikuti layar lama (`D-13`: alur dan tata letak
 * ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Dua tombol pemilih isi   — `Section/BrowseNoteRejectClaim-Section.xml`
 *   - Kolom tab Penolakan Klaim — Status Penolakan 1 · Status Penolakan 2 · Status Aproval
 *     · Note Approval
 *   - Kolom tab Penolakan Komite — ID Master · Note Komite Reject · tombol "ubah"
 *
 * # Yang SENGAJA tidak ada, dan itu bukan pekerjaan yang belum selesai
 *
 * **Tombol Hapus.** Seluruh export tidak memuat satu pun DELETE terhadap ketiga tabel ini,
 * layar lama pun tidak punya tombolnya, dan tidak satu pun tabelnya punya kolom penanda
 * terhapus yang dapat dipakai `D-66`.
 *
 * **Isian persetujuan.** Tanggal Approve, Approve By, Status Aproval, dan Note Approval
 * hanya DITAMPILKAN. Yang mengisinya adalah layar Inbox Manager —
 * `Section/Sec_PenolakanKlaimChecker-Section.xml` yang dipakai
 * `Harness/UserInbox_Harness-Harness.xml` (MENU_ID 58), modul tersendiri yang belum
 * dibangun. Keputusan Work Owner 2026-09-19.
 *
 * **Penyaring "hanya yang menunggu".** Kueri lama memuat
 * `{ASIS:MasterCheckerPenolakan.RemakApprove}` yang diisi `"WHERE STATUS='0'"` — tetapi
 * langkah pengisinya berprasyarat `Param.master=="1"`, dan layar ini tidak mengirim
 * parameter itu. Menyaringnya di sini akan menyembunyikan baris yang sudah diputuskan.
 */
export function RejectionPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [tab, setTab] = useState<Tab>('klaim')

  // Isi tab aktif, atau peringatan bila portal belum dipilih.
  function renderTab() {
    if (portal === null) {
      return (
        <section className="mt-6">
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        </section>
      )
    }
    if (tab === 'klaim') {
      return <RejectionTab />
    }
    return <CommitteeTab />
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Penolakan Klaim</h1>
          <p className="text-sm text-slate-600">
            Daftar baku alasan penolakan klaim, beserta catatan penolakan komite.
          </p>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
      </p>

      {/* Dua tab menggantikan dua tombol layar lama. Ia `role="tablist"` supaya pembaca
          layar menyebutnya sebagai pemilih, bukan sebagai dua tombol lepas. */}
      <div role="tablist" aria-label="Pilih master" className="mt-5 flex flex-wrap gap-2">
        <TabButton active={tab === 'klaim'} onClick={() => setTab('klaim')}>
          Penolakan Klaim
        </TabButton>
        <TabButton active={tab === 'komite'} onClick={() => setTab('komite')}>
          Penolakan Komite
        </TabButton>
      </div>

      {renderTab()}
    </main>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: Readonly<{
  active: boolean
  onClick: () => void
  children: string
}>) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={
        'rounded-kontrol px-3.5 py-2 text-sm font-medium transition-colors duration-150 ' +
        'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20 ' +
        (active
          ? 'bg-blue-600 text-white shadow-aksen'
          : 'bg-white text-slate-700 ring-1 ring-slate-300 hover:bg-slate-50')
      }
    >
      {children}
    </button>
  )
}

/**
 * Isi satu tab: tombol Refresh dan Tambah, form (bila terbuka), lalu daftar. Dipakai bersama
 * oleh kedua tab supaya susunannya hanya ditulis sekali.
 */
function TabBody<TData, TError>({
  list,
  onAdd,
  addDisabled,
  form,
  loadingText,
  renderTable,
}: Readonly<{
  list: UseQueryResult<TData, TError>
  onAdd: () => void
  addDisabled: boolean
  form: ReactNode
  loadingText: string
  renderTable: (data: TData) => ReactNode
}>) {
  return (
    <>
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <RefreshButton query={list} />
        <AddButton onClick={onAdd} disabled={addDisabled} />
      </div>

      {form ? <section className="mt-5">{form}</section> : null}

      {/* Isi bagian daftar menurut keadaan kueri. */}
      <section className="mt-6">
        {renderListState({ query: list, loadingText, toMessage: loadMessage, render: renderTable })}
      </section>
    </>
  )
}

/** Tab pertama — POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2. */
function RejectionTab() {
  const list = useRejectionList()
  const parents = useRejectionParentList()
  const create = useCreateRejection()
  const update = useUpdateRejection()

  const form = useCrudForm(create, update, (row: Rejection, input) => ({ id: row.id, input }))
  const edited = form.openedRow
  const saving = edited === null ? create : update
  const { openEdit } = form

  function save(values: RejectionFields) {
    form.submit(toInput(values))
  }

  // Susunan kolom mengikuti grid Pega apa adanya, terbaca dari label pada
  // `Section/BrowseNoteRejectClaim-Section.xml`:
  //
  //	Status Penolakan 1  -> .City    (NOTE_ST, nama induk)
  //	Status Penolakan 2  -> .CityID  (NOTE_ND, nama baris ini)
  //	Status Aproval      -> derivasi STATUS
  //	Note Approval       -> .NoteKasir (NOTEAPPROVED)
  //
  // Perhatikan urutannya: induk mendahului nama barisnya sendiri, sama seperti pada Master
  // Status Progres 2. Itu berlawanan dengan dugaan yang wajar.
  const columns: Column<Rejection>[] = [
    {
      key: 'status_1',
      title: 'Status Penolakan 1',
      // ID induk ikut ke `value` supaya pencarian menemukan baris lewat kodenya maupun
      // lewat namanya.
      value: (row) => `${row.nama_status_1} ${row.id_status_1}`,
      render: (row) => (
        <span>
          {row.nama_status_1}
          <span className="ml-2 text-xs text-slate-500">{row.id_status_1}</span>
        </span>
      ),
    },
    { key: 'nama', title: 'Status Penolakan 2', value: (row) => row.nama },
    {
      key: 'status',
      title: 'Status Aproval',
      width: 'w-36',
      value: (row) => row.status_label,
      render: (row) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusClass(row.status)}`}
        >
          {row.status_label}
        </span>
      ),
    },
    {
      key: 'catatan_persetujuan',
      title: 'Note Approval',
      value: (row) => row.catatan_persetujuan,
      render: (row) => (
        <span>
          {row.catatan_persetujuan === '' ? (
            <span className="text-slate-400">—</span>
          ) : (
            row.catatan_persetujuan
          )}
          {row.disetujui_oleh !== '' && (
            <span className="block text-xs text-slate-500">
              {row.disetujui_oleh} · {formatDate(row.disetujui_pada)}
            </span>
          )}
        </span>
      ),
    },
    editColumn<Rejection>({
      onEdit: openEdit,
      width: 'w-24',
      title: '',
      ariaLabel: (row) => `Ubah ${row.nama}`,
    }),
  ]

  return (
    <TabBody
      list={list}
      onAdd={form.openCreate}
      addDisabled={form.isOpen && edited === null}
      form={
        form.isOpen && (
          <RejectionForm
            edited={edited}
            parents={parents.data?.status_1 ?? []}
            isSaving={saving.isPending}
            error={saving.error}
            onSave={save}
            onCancel={form.closeForm}
          />
        )
      }
      loadingText="Memuat daftar penolakan klaim…"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.penolakan_klaim}
          rowKey={(row) => row.id}
          description="Sumber: POOLDATA.MST_PENOLAKAN_KLAIM_2 · persetujuan diisi lewat Inbox Manager"
          emptyMessage="Belum ada penolakan klaim pada entitas ini."
        />
      )}
    />
  )
}

/** Tab kedua — POOLDATA.MST_REJECTED_KOMITE. */
function CommitteeTab() {
  const list = useCommitteeRejectionList()
  const create = useCreateCommitteeRejection()
  const update = useUpdateCommitteeRejection()

  const form = useCrudForm(
    create,
    update,
    (row: CommitteeRejection, input: CommitteeRejectionFields) => ({ id: row.id, input }),
  )
  const edited = form.openedRow
  const saving = edited === null ? create : update
  const { openEdit } = form

  // Dua kolom, persis seperti kueri lama — ditambah tombol "ubah" yang di Pega dikirim
  // sebagai kolom ketiga beralias `NOKTP`. Label tombolnya dibawa; caranya tidak.
  const columns: Column<CommitteeRejection>[] = [
    { key: 'id', title: 'ID Master', width: 'w-28', value: (row) => row.id },
    { key: 'catatan', title: 'Note Komite Reject', value: (row) => row.catatan },
    editColumn<CommitteeRejection>({
      onEdit: openEdit,
      width: 'w-24',
      title: '',
      ariaLabel: (row) => `Ubah ${row.catatan}`,
    }),
  ]

  return (
    <TabBody
      list={list}
      onAdd={form.openCreate}
      addDisabled={form.isOpen && edited === null}
      form={
        form.isOpen && (
          <CommitteeRejectionForm
            edited={edited}
            isSaving={saving.isPending}
            error={saving.error}
            onSave={form.submit}
            onCancel={form.closeForm}
          />
        )
      }
      loadingText="Memuat daftar penolakan komite…"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.penolakan_komite}
          rowKey={(row) => row.id}
          description="Sumber: POOLDATA.MST_REJECTED_KOMITE"
          emptyMessage="Belum ada penolakan komite pada entitas ini."
        />
      )}
    />
  )
}
