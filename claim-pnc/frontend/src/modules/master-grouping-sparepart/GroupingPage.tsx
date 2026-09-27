import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, GroupingStatus, type Grouping } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCreateGrouping, useDecideGrouping, useGroupingList, useSaveGrouping } from './api'
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
 * # Enam kolom, bukan sebelas
 *
 * Grid Pega hanya menampilkan ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, dan
 * No Rangka; kelima isian sisanya hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru
 * apa adanya, termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * Satu kolom DITAMBAHKAN: **Nomor Grup**. Ia tidak digambar di layar lama sama sekali, dan
 * tanpa itu pengguna tidak punya satu pun cara melihat baris mana yang tergabung dengan baris
 * mana — padahal itulah seluruh gunanya layar ini. Selisih yang dicatat, bukan disembunyikan.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Di sistem lama status yang disimpan datang dari pemanggil (`Param.Approval`), dan layar
 * penyuntingannya selalu mengirim "0".
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalPNCMasterGroupingSparepartHE` dipakai
 * Inbox Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap grouping yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya. Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang
 * beberapa baris, lalu satu tombol untuk seluruh pilihan — sama dengan ketiga master alat
 * berat lain.
 *
 * # Tanpa isian Catatan pada keputusan
 *
 * Kedua tabel modul ini tidak punya kolom penampung alasan penolakan, dan layar persetujuan
 * Pega pun tidak punya isian catatan. Menggambar isian yang diam-diam membuang isinya lebih
 * buruk daripada tidak menggambarnya.
 */
export function GroupingPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Grouping | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useGroupingList(active.status)
  const create = useCreateGrouping()
  const save = useSaveGrouping()
  const decide = useDecideGrouping()

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

  function toggle(id: string) {
    setChosen((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
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

  function runDecision(status: string) {
    decide.mutate({ id_grouping: [...chosen], status }, { onSuccess: () => setChosen(new Set()) })
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — keenamnya berdampingan pada urutan yang
    sama: ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, No Rangka.

    Judulnya pun diambil dari grid, bukan dari form: grid menulis "No Sparepart" dan "Sisi
    Panel" sementara form menulis "Nomor Sparepart" dan "Sisi". Keduanya dipertahankan di
    tempatnya masing-masing (D-13).
  */
  const columns: Column<Grouping>[] = [
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: Grouping) => (chosen.has(row.id_grouping) ? 'dipilih' : ''),
            render: (row: Grouping) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_grouping)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_grouping)}
                />
                <span className="sr-only">
                  Pilih grouping {row.nomor_sparepart} pada {row.nama_panel}
                </span>
              </label>
            ),
          } satisfies Column<Grouping>,
        ]
      : []),
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
      // DITAMBAHKAN terhadap grid Pega; lihat catatan pada GroupingPage.
      key: 'grup',
      title: 'Nomor Grup',
      width: '8rem',
      value: (row) => row.nomor_grup,
      render: (row) => (
        <span className="tabular-nums text-sm text-slate-900">{row.nomor_grup || '—'}</span>
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
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
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
              // Centang dibuang saat berpindah tab: baris yang dipilih milik tab sebelumnya,
              // dan menyimpannya berarti keputusan dapat mengenai baris yang tidak sedang
              // dilihat siapa pun.
              setChosen(new Set())
              decide.reset()
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

      {decide.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Keputusan belum tersimpan"
            description={
              decide.error instanceof APIError
                ? decide.error.message
                : 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {decide.isSuccess && decide.data && (
        <p className="mt-4 rounded-kontrol border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
          {decide.data.jumlah_berubah} grouping dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          isBusy={decide.isPending}
          onApprove={() => runDecision(GroupingStatus.disetujui)}
          onReject={() => runDecision(GroupingStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

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

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang.
 *
 * Ia padanan `Section/ApprovalPNCMasterGroupingSparepartHE-Section.xml`.
 *
 * TANPA isian Catatan: kedua tabel modul ini tidak punya kolom penampungnya.
 *
 * Tombolnya mati selama belum ada yang dicentang — bukan disembunyikan. Tombol yang hilang
 * membuat pengguna mencari fiturnya; tombol yang mati menunjukkan apa yang harus dilakukan
 * lebih dulu.
 */
function DecisionBar({
  count,
  isBusy,
  onApprove,
  onReject,
  onClear,
}: {
  count: number
  isBusy: boolean
  onApprove: () => void
  onReject: () => void
  onClear: () => void
}) {
  return (
    <div className="mt-4 flex flex-wrap items-center gap-2 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <span className="min-w-0 flex-1 text-sm text-slate-700">
        {count === 0 ? (
          'Centang grouping yang akan diputuskan.'
        ) : (
          <>
            <span className="font-medium">{count} grouping</span> dipilih.
          </>
        )}
      </span>
      {count > 0 && (
        <Button tone="halus" onClick={onClear} disabled={isBusy}>
          Bersihkan
        </Button>
      )}
      {/*
        Namanya "Approve terpilih", bukan "Approve" saja.

        Bukan sekadar demi kejelasan kalimat: tab di atasnya juga bernama "Approve" dan
        "Reject" — caption Pega yang memang harus ditiru (D-13) — sehingga tombol bernama sama
        membuat dua kontrol yang sama sekali berbeda tidak dapat dibedakan dari namanya.
        Pembaca layar mengumumkan keduanya dengan kata yang sama persis.
      */}
      <Button tone="utama" onClick={onApprove} disabled={isBusy || count === 0}>
        {isBusy ? 'Menyimpan…' : 'Approve terpilih'}
      </Button>
      <Button tone="kedua" onClick={onReject} disabled={isBusy || count === 0}>
        Reject terpilih
      </Button>
    </div>
  )
}
