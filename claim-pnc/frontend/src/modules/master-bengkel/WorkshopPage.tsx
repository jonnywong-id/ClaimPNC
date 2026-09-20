import { useMemo, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, WorkshopStatus, type Workshop } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCreateWorkshop, useDecideWorkshop, useSaveWorkshop, useWorkshopList } from './api'
import { WorkshopForm, type WorkshopFormValues } from './WorkshopForm'

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

type TabId = (typeof TABS)[number]['id']

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
          title: 'Daftar bengkel tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar bengkel tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
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

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Workshop | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useWorkshopList(active.status)
  const create = useCreateWorkshop()
  const save = useSaveWorkshop()
  const decide = useDecideWorkshop()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.bengkel ?? []

  /*
    Saran nilai untuk kolom yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia
    jawaban terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" —
    lihat WorkshopForm bagian "Penanda sistem".
  */
  const knownValues = useMemo(() => {
    const collected: Record<string, string[]> = {}
    for (const column of SUGGESTED_COLUMNS) {
      const unique = new Set<string>()
      for (const row of rows) {
        const value = row[column]
        if (value !== '') unique.add(value)
      }
      collected[column] = [...unique].sort()
    }
    return collected
  }, [rows])

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

  function openEdit(row: Workshop) {
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

  function submit(values: WorkshopFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form tiga puluh tiga
    // isian, itu kehilangan yang tidak dapat dimaafkan.
    if (editing) {
      save.mutate({ id: editing.id_bengkel, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
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
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: Workshop) => (chosen.has(row.id_bengkel) ? 'dipilih' : ''),
            render: (row: Workshop) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_bengkel)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_bengkel)}
                />
                <span className="sr-only">Pilih {row.nama_bengkel}</span>
              </label>
            ),
          } satisfies Column<Workshop>,
        ]
      : []),
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
      width: '6rem',
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
          {/*
            "Master Bengkel HE", bukan "Master Bengkel".

            Itu judul yang tertulis di layar Pega — caption
            `Section/MasterBengkelHE-Section.xml`. Butir menunya memang bernama "Master
            Bengkel" (`M_MENU_APLIKASI_PNC.MENU_DESC`), dan keduanya memang berbeda di
            sistem lama; `D-13` menuntut teks LAYAR yang diikuti.
          */}
          <h1 className="text-xl font-semibold text-slate-900">Master Bengkel HE</h1>
          <p className="text-sm text-slate-600">
            Bengkel rekanan beserta syarat kerja samanya — rekening pembayaran, diskon, pajak,
            dan SLA.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
        </div>
      </header>

      <nav
        aria-label="Tab Master Bengkel"
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
          {decide.data.jumlah_berubah} bengkel dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          isBusy={decide.isPending}
          onApprove={() => runDecision(WorkshopStatus.disetujui)}
          onReject={() => runDecision(WorkshopStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <WorkshopForm
            editing={editing}
            knownValues={knownValues}
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
          <p className="text-sm text-slate-500">Memuat daftar bengkel…</p>
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
            rowKey={(row) => row.id_bengkel}
            description="Sumber: POOLDATA.BENGKEL_HE"
            searchLabel="Cari bengkel"
            emptyMessage={`Belum ada bengkel pada tab ${active.label}.`}
            pageSize={PAGE_SIZE}
          />
        )}
      </section>
    </main>
  )
}

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang.
 *
 * Ia padanan `Activity/SetApprovalAllMaster`: satu keputusan atas sekumpulan pengajuan,
 * bukan satu keputusan per baris.
 *
 * Tombolnya mati selama belum ada yang dicentang — bukan disembunyikan. Tombol yang
 * hilang membuat pengguna mencari fiturnya; tombol yang mati menunjukkan apa yang harus
 * dilakukan lebih dulu.
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
          'Centang bengkel yang akan diputuskan.'
        ) : (
          <>
            <span className="font-medium">{count} bengkel</span> dipilih.
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
        namanya. Pembaca layar mengumumkan keduanya dengan kata yang sama persis, dan
        pengguna yang menyebut "tombol Approve" pada perintah suara tidak punya cara
        memilih yang mana.

        Kata "terpilih" sekaligus menyebutkan sifatnya: ia mengenai SELURUH baris yang
        dicentang, bukan satu baris.
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
function LoginCell({ row }: { row: Workshop }) {
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
