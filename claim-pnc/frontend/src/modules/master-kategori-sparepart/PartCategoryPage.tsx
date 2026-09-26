import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PartCategoryStatus, type PartCategory } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  useCreatePartCategory,
  useDecidePartCategory,
  usePartCategoryList,
  useSavePartCategory,
} from './api'
import { PartCategoryForm, type PartCategoryFormValues } from './PartCategoryForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/MasterKategoriSparepartHE-Section.xml` merujuk tiga section yang ketiganya
 * membaca tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan kemunculannya
 * di dalam section itu:
 *
 *	MasterKategoriSparepartHEApprove   APPROVAL="1"
 *	MasterKategoriSparepartHEReject    APPROVAL="2"
 *	MasterKategoriSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Sparepart, Master Panel,
 * dan Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini,
 * dan `D-13` menuntut tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: PartCategoryStatus.disetujui,
    description:
      'Kategori yang sudah disetujui. Hanya yang ada di sini yang muncul sebagai pilihan ' +
      'di layar Master Sparepart dan Master Tipe Sparepart.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: PartCategoryStatus.ditolak,
    description:
      'Kategori yang ditolak. Namanya tetap terpakai — nama yang sama tidak dapat ' +
      'digunakan kategori lain sampai baris ini diubah namanya.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: PartCategoryStatus.menunggu,
    description: 'Kategori yang menunggu diputuskan.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab layar ini.
 *
 * Ketiganya bernilai **50** — berbeda dari Master Sparepart yang 30 dan Master Bengkel yang
 * 20. Tidak ada satu angka yang benar untuk seluruh layar; angkanya milik layar, bukan
 * milik komponen tabel.
 */
const PAGE_SIZE = 50

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian ' +
            'atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar kategori sparepart tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Kategori Sparepart.
 *
 * Pengganti `Harness/GCNMCatSparepart-Harness.xml` atas tabel
 * POOLDATA.GCNM_M_SPAREPART_CATEGORY (MENU_ID 33).
 *
 * # Apa yang dikelola layar ini
 *
 * **Penggolongan suku cadang alat berat** — satu baris per kategori, dan setiap kategori
 * hanya punya nama. Ia tabel acuan yang menyuapi dua layar lain: Master Sparepart memakai
 * ID-nya pada kolom KATEGORI_SPART, dan Master Tipe Sparepart menjadikannya induk setiap
 * tipe.
 *
 * # Dua kolom, dan memang hanya itu yang ada
 *
 * Grid Pega menampilkan "ID Kategori Sparepart" dan "Nama Kategori Sparepart", dan tabelnya
 * memang hanya punya tiga kolom — yang ketiga adalah status, yang sudah menjadi tab. Tidak
 * ada kolom "User Update" maupun "Tanggal Update" seperti pada Master Sparepart: tabel ini
 * tidak punya kolomnya, sehingga siapa yang mengubah sebuah kategori TIDAK tersimpan di
 * mana pun.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateKategoriSparepart_act2` menetapkan APPROVAL := "0" tanpa syarat apa pun.
 * Akibatnya menyentuh layar LAIN — kategori yang sedang menunggu hilang dari dropdown
 * Master Sparepart — dan itu dinyatakan di muka pada form, bukan dibiarkan ditemukan.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterKategoriSparepartHE` dipakai
 * Inbox Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap kategori yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya — dan selama tertahan, ia tidak dapat dipakai sparepart mana pun.
 *
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris,
 * lalu satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Bengkel, Master
 * Panel, dan Master Sparepart. Keputusan Work Owner 2026-09-21.
 *
 * # Tanpa isian Catatan
 *
 * `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya kolom penampung alasan penolakan.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak
 * menggambarnya — perlakuan yang sama dengan Master Sparepart.
 */
export function PartCategoryPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<PartCategory | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = usePartCategoryList(active.status)
  const create = useCreatePartCategory()
  const save = useSavePartCategory()
  const decide = useDecidePartCategory()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.kategori_sparepart ?? []

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

  function openEdit(row: PartCategory) {
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

  function submit(values: PartCategoryFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan penolakan nama ganda adalah
    // kegagalan yang paling sering terjadi di layar ini.
    if (editing) {
      save.mutate(
        { id: editing.id_kategori_sparepart, input: values },
        { onSuccess: closeForm },
      )
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_kategori_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Dua kolom, mengikuti grid Pega apa adanya, ditambah kolom aksi dan — pada tab Waiting
    Approval — kolom centang.

    Tidak ada kolom status: ia sudah menjadi tab, dan menggambarnya lagi di setiap baris
    hanya mengulang hal yang sama di seluruh halaman.
  */
  const columns: Column<PartCategory>[] = [
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: PartCategory) =>
              chosen.has(row.id_kategori_sparepart) ? 'dipilih' : '',
            render: (row: PartCategory) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_kategori_sparepart)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_kategori_sparepart)}
                />
                <span className="sr-only">Pilih {row.nama_kategori_sparepart}</span>
              </label>
            ),
          } satisfies Column<PartCategory>,
        ]
      : []),
    {
      key: 'id',
      title: 'ID Kategori Sparepart',
      width: '12rem',
      value: (row) => row.id_kategori_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Kategori Sparepart',
      value: (row) => row.nama_kategori_sparepart,
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
    <main className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari `pyCaption Master Kategori Sparepart` pada
              Section/MasterKategoriSparepartHE (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Kategori Sparepart</h1>
          <p className="text-sm text-slate-600">
            Penggolongan suku cadang alat berat yang menjadi pilihan Kategori di layar Master
            Sparepart dan Master Tipe Sparepart.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      <nav
        aria-label="Tab Master Kategori Sparepart"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-current={tab === t.id ? 'page' : undefined}
            onClick={() => {
              closeForm()
              // Centang dibuang saat berpindah tab: baris yang dipilih milik tab
              // sebelumnya, dan menyimpannya berarti keputusan dapat mengenai baris yang
              // tidak sedang dilihat siapa pun.
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
          {decide.data.jumlah_berubah} kategori sparepart dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          isBusy={decide.isPending}
          onApprove={() => runDecision(PartCategoryStatus.disetujui)}
          onReject={() => runDecision(PartCategoryStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <PartCategoryForm
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
          <p className="text-sm text-slate-500">Memuat daftar kategori sparepart…</p>
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
            rowKey={(row) => row.id_kategori_sparepart}
            description="Sumber: POOLDATA.GCNM_M_SPAREPART_CATEGORY"
            searchLabel="Cari kategori sparepart"
            pageSize={PAGE_SIZE}
            emptyMessage={`Belum ada kategori sparepart pada tab ${active.label}.`}
          />
        )}
      </section>
    </main>
  )
}

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang.
 *
 * Ia padanan `Section/ApprovalMasterKategoriSparepartHE-Section.xml`, yang menggambar grid
 * bercentang dengan tombol Approve, Reject, dan DETAILS.
 *
 * DETAILS tidak ditiru: di layar ini setiap baris sudah punya tombol Ubah yang membuka
 * seluruh isinya — dan isinya hanya satu isian. Tombol kedua yang membuka hal yang sama
 * hanya akan menambah pilihan tanpa menambah kemampuan.
 *
 * TANPA isian Catatan: tabelnya tidak punya kolom penampungnya.
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
          'Centang kategori yang akan diputuskan.'
        ) : (
          <>
            <span className="font-medium">{count} kategori</span> dipilih.
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
        "Reject" — caption Pega yang memang harus ditiru (D-13) — sehingga tombol bernama
        sama membuat dua kontrol yang sama sekali berbeda tidak dapat dibedakan dari
        namanya. Pembaca layar mengumumkan keduanya dengan kata yang sama persis.
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
