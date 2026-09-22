import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PartTypeStatus, type PartType } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  useCreatePartType,
  useDecidePartType,
  usePartTypeList,
  usePartTypeOptions,
  useSavePartType,
} from './api'
import { PartTypeForm, type PartTypeFormValues } from './PartTypeForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/MasterTipeSparepartHE-Section.xml` merujuk tiga section yang ketiganya membaca
 * tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya:
 *
 *	MasterTipeSparepartHEApprove   APPROVAL="1"
 *	MasterTipeSparepartHEReject    APPROVAL="2"
 *	MasterTipeSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Kategori Sparepart, Master
 * Sparepart, Master Panel, dan Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang
 * dilihat petugas hari ini, dan `D-13` menuntut tata letak yang sama supaya pengguna tidak
 * perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: PartTypeStatus.disetujui,
    description:
      'Tipe yang sudah disetujui. Hanya yang ada di sini yang muncul sebagai pilihan Tipe ' +
      'di layar Master Sparepart.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: PartTypeStatus.ditolak,
    description:
      'Tipe yang ditolak. Namanya tetap terpakai — nama yang sama tidak dapat digunakan ' +
      'tipe lain, di kategori mana pun, sampai baris ini diubah namanya.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: PartTypeStatus.menunggu,
    description: 'Tipe yang menunggu diputuskan.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab layar ini.
 *
 * Ketiganya bernilai **50** — sama dengan Master Kategori Sparepart, berbeda dari Master
 * Sparepart yang 30 dan Master Bengkel yang 20. Tidak ada satu angka yang benar untuk
 * seluruh layar; angkanya milik layar, bukan milik komponen tabel.
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
          title: 'Daftar tipe sparepart tidak dapat dimuat',
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
 * Layar Master Tipe Sparepart.
 *
 * Pengganti `Harness/GCNMMasterSparepartType-Harness.xml` atas tabel
 * POOLDATA.GCNM_M_SPAREPART_TYPE (MENU_ID 34).
 *
 * # Apa yang dikelola layar ini
 *
 * **Penggolongan tingkat kedua suku cadang alat berat** — setiap tipe berinduk pada sebuah
 * kategori. Ia tabel acuan yang menyuapi satu layar lain: Master Sparepart memakai ID-nya
 * pada kolom TIPE_SPART.
 *
 * # Empat kolom, dan salah satunya milik tabel lain
 *
 * Grid Pega menampilkan "ID Tipe Sparepart", "Nama Tipe Sparepart", "ID Kategori
 * Sparepart", dan "Kategori Sparepart". Yang terakhir bukan kolom tabel ini — ia
 * PART_CATEGORY_NAME milik tabel kategori, dibaca lewat JOIN.
 *
 * Tidak ada kolom "User Update" maupun "Tanggal Update" seperti pada Master Sparepart:
 * tabel ini tidak punya kolomnya, sehingga siapa yang mengubah sebuah tipe TIDAK tersimpan
 * di mana pun.
 *
 * # Baris tanpa kategori TETAP terlihat di sini, dan itu berbeda dari Pega
 *
 * `BrowseMasterSparepartTypeClaimHE_sql` memakai inner join, sehingga tipe yang menunjuk
 * kategori yang sudah tidak ada HILANG dari layar lama — tidak dapat dilihat, tidak dapat
 * diperbaiki. Layar ini memakai LEFT JOIN dan menampilkannya dengan tanda "—" beserta
 * keterangan. Selisih perilaku ini disengaja; `claimpnc -periksa` melaporkan jumlahnya
 * lebih dulu supaya ia dapat dijelaskan sebelum uji kesetaraan dijalankan.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateTypeSparepart_act2` menetapkan APPROVAL := "0" tanpa syarat apa pun.
 * Akibatnya menyentuh layar LAIN — tipe yang sedang menunggu hilang dari dropdown Master
 * Sparepart — dan itu dinyatakan di muka pada form, bukan dibiarkan ditemukan.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterTipeSparepartHE` dipakai Inbox
 * Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap tipe yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya — dan selama tertahan, ia tidak dapat dipakai sparepart mana pun.
 *
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris,
 * lalu satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Kategori
 * Sparepart, Master Bengkel, Master Panel, dan Master Sparepart. Keputusan Work Owner
 * 2026-09-21.
 *
 * # Tanpa isian Catatan
 *
 * `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak punya kolom penampung alasan penolakan.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak
 * menggambarnya.
 */
export function PartTypePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<PartType | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = usePartTypeList(active.status)
  const options = usePartTypeOptions()
  const create = useCreatePartType()
  const save = useSavePartType()
  const decide = useDecidePartType()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.tipe_sparepart ?? []

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
    // Daftar pilihan disegarkan setiap kali form dibuka. Ia bercache panjang karena jarang
    // berubah, dan justru karena itu ia dapat basi tepat pada saat ia dipakai — kategori
    // yang ditolak sejak halaman dibuka akan tetap tampak dapat dipilih.
    void options.refetch()
  }

  function openEdit(row: PartType) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
    void options.refetch()
  }

  function toggle(id: string) {
    setChosen((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function submit(values: PartTypeFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan penolakan nama ganda adalah
    // kegagalan yang paling sering terjadi di layar ini.
    if (editing) {
      save.mutate({ id: editing.id_tipe_sparepart, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_tipe_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Empat kolom, mengikuti grid Pega apa adanya, ditambah kolom aksi dan — pada tab Waiting
    Approval — kolom centang.

    Tidak ada kolom status: ia sudah menjadi tab, dan menggambarnya lagi di setiap baris
    hanya mengulang hal yang sama di seluruh halaman.
  */
  const columns: Column<PartType>[] = [
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: PartType) => (chosen.has(row.id_tipe_sparepart) ? 'dipilih' : ''),
            render: (row: PartType) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_tipe_sparepart)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_tipe_sparepart)}
                />
                <span className="sr-only">Pilih {row.nama_tipe_sparepart}</span>
              </label>
            ),
          } satisfies Column<PartType>,
        ]
      : []),
    {
      key: 'id',
      title: 'ID Tipe Sparepart',
      width: '10rem',
      value: (row) => row.id_tipe_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Tipe Sparepart',
      value: (row) => row.nama_tipe_sparepart,
    },
    {
      key: 'id_kategori',
      title: 'ID Kategori Sparepart',
      width: '11rem',
      value: (row) => row.id_kategori_sparepart,
    },
    {
      key: 'kategori',
      title: 'Kategori Sparepart',
      // Nilai pencariannya tetap nama kategori apa adanya — termasuk saat kosong — supaya
      // pencarian bawaan DataTable tidak ikut menjaring tanda "—" yang hanya tampilan.
      value: (row) => row.nama_kategori_sparepart,
      render: (row) =>
        row.nama_kategori_sparepart !== '' ? (
          <span>{row.nama_kategori_sparepart}</span>
        ) : (
          /*
            Baris yatim: kategorinya tidak ada di master kategori.

            Di Pega baris ini tidak akan pernah terlihat — inner join-nya membuangnya. Di
            sini ia ditampilkan beserta alasannya, karena hanya dengan begitu petugas dapat
            memperbaikinya lewat tombol Ubah.
          */
          <span
            className="text-amber-700"
            title="Kategori induknya tidak ada di Master Kategori Sparepart. Buka Ubah lalu pilih kategori yang sah."
          >
            — kategori tidak ditemukan
          </span>
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
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari `pyCaption Master Tipe Sparepart` pada
              Section/MasterTipeSparepartHE (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Tipe Sparepart</h1>
          <p className="text-sm text-slate-600">
            Penggolongan tingkat kedua suku cadang alat berat. Setiap tipe berinduk pada satu
            kategori, dan menjadi pilihan Tipe di layar Master Sparepart.
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
        aria-label="Tab Master Tipe Sparepart"
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
          {decide.data.jumlah_berubah} tipe sparepart dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          isBusy={decide.isPending}
          onApprove={() => runDecision(PartTypeStatus.disetujui)}
          onReject={() => runDecision(PartTypeStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <PartTypeForm
            editing={editing}
            category={options.data?.kategori ?? []}
            isLoadingCategory={options.isPending || options.isFetching}
            isCategoryTruncated={options.data?.terpotong ?? false}
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
          <p className="text-sm text-slate-500">Memuat daftar tipe sparepart…</p>
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
            rowKey={(row) => row.id_tipe_sparepart}
            description="Sumber: POOLDATA.GCNM_M_SPAREPART_TYPE"
            searchLabel="Cari tipe atau kategori sparepart"
            pageSize={PAGE_SIZE}
            emptyMessage={`Belum ada tipe sparepart pada tab ${active.label}.`}
          />
        )}
      </section>
    </main>
  )
}

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang.
 *
 * Ia padanan `Section/ApprovalMasterTipeSparepartHE-Section.xml`, yang menggambar grid
 * bercentang dengan tombol Approve dan Reject.
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
          'Centang tipe yang akan diputuskan.'
        ) : (
          <>
            <span className="font-medium">{count} tipe</span> dipilih.
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
