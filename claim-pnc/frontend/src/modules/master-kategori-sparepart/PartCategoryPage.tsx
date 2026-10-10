import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PartCategoryStatus, type PartCategory } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

// `useDecidePartCategory` sengaja TIDAK diimpor. Endpoint-nya masih ada dan hook-nya masih
// diekspor — keduanya padanan `Activity/UpdateKategoriSparepart_act` yang memang ada di
// Pega — tetapi pemakainya adalah Inbox Manager, bukan layar ini. Lihat PartCategoryPage.
import { useCreatePartCategory, usePartCategoryList, useSavePartCategory } from './api'
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
 * # TANPA tombol Approve dan Reject — dan itu memang Pega
 *
 * Versi pertama layar ini (2026-09-21) menggambar centang pada tab Waiting Approval beserta
 * tombol "Approve terpilih" dan "Reject terpilih". Keduanya **dihapus pada 2026-10-04**
 * setelah ketiga section tab Pega dibaca ulang, dan hasilnya tidak menyisakan ruang tafsir:
 *
 *	MasterKategoriSparepartHEApprove   tombol: UBAH, Save
 *	MasterKategoriSparepartHEReject    tombol: UBAH, Save
 *	MasterKategoriSparepartHEApproval  tombol: UBAH, Save
 *
 * Ketiganya IDENTIK. Tidak satu pun memuat Approve, Reject, maupun kontrol centang.
 *
 * Keputusannya ada di layar LAIN — `Section/ApprovalMasterKategoriSparepartHE`, yang dipakai
 * Inbox Manager — dan bentuknya pun berbeda dari yang sempat dibangun di sini:
 *
 *	pySelected           0 kemunculan   -> tidak ada pemilihan borongan
 *	SetApprovalAllMaster 0 kemunculan   -> kategori TIDAK ikut activity borongan itu
 *	tombol               Approve · Reject · DETAILS, per baris
 *
 * Jadi keputusan kategori di Pega bersifat **per baris** lewat
 * `Activity/UpdateKategoriSparepart_act`, bukan borongan bercentang seperti Master Bengkel,
 * Panel, dan Sparepart. Yang dibangun sebelumnya salah pada DUA hal sekaligus: salah tempat,
 * dan salah model.
 *
 * **Akibat yang disadari dan diterima:** dari layar ini sebuah kategori tidak dapat
 * disetujui maupun ditolak. Itu persis keadaan di Pega, dan ia berarti kategori baru
 * tertahan di Waiting Approval — tidak dapat dipakai sparepart mana pun — sampai Inbox
 * Manager dibangun. Keputusan Work Owner 2026-10-04: ikuti Pega.
 *
 * Akibat itu TIDAK dijelaskan di layar. Satu paragraf keterangan sempat dipasang pada tab
 * Waiting Approval dan **dihapus atas permintaan Work Owner** pada hari yang sama: layar
 * Pega tidak memuat kalimat semacam itu, dan menambahkannya adalah menambah sesuatu yang
 * tidak ada di sana. Ketiga tab karena itu kini murni daftar.
 *
 * Endpoint `POST /keputusan` di server TIDAK dihapus: ia padanan
 * `UpdateKategoriSparepart_act` yang memang ada di Pega, dan Inbox Manager akan memanggilnya.
 * Yang dihapus hanyalah pemakaiannya di layar ini.
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

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = usePartCategoryList(active.status)
  const create = useCreatePartCategory()
  const save = useSavePartCategory()

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

  /*
    Dua kolom, mengikuti grid Pega apa adanya, ditambah kolom aksi.

    Susunannya SAMA pada ketiga tab, dan itu bukan penyederhanaan melainkan peniruan:
    ketiga section tab Pega isinya identik — grid dua kolom, tombol UBAH, tombol Save.
    Tidak ada kolom centang pada tab mana pun; lihat catatan pada PartCategoryPage.

    Tidak ada kolom status: ia sudah menjadi tab, dan menggambarnya lagi di setiap baris
    hanya mengulang hal yang sama di seluruh halaman.
  */
  const columns: Column<PartCategory>[] = [
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
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
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
