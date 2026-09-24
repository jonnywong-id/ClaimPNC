import { useMemo, useState } from 'react'

import { APIError } from '@/api/client'
import { SparepartStatus, type Sparepart } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useCreateSparepart, useDecideSparepart, useSaveSparepart, useSparepartList } from './api'
import { SparepartForm, type SparepartFormValues } from './SparepartForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/BrowseMasterSparepartHE-Section.xml` memuat tiga section yang ketiganya membaca
 * `BrowseSparepartHE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu:
 *
 *	BrowseMasterSparepartHEApprove   APPROVAL="1"
 *	BrowseMasterSparepartHEReject    APPROVAL="2"
 *	BrowseMasterSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Panel dan Master Bengkel.
 * Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini, dan `D-13` menuntut
 * tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: SparepartStatus.disetujui,
    description: 'Sparepart yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: SparepartStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: SparepartStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab Master Sparepart.
 *
 * Ketiganya bernilai **30** — berbeda dari Master Panel yang 15 pada satu tab dan 50 pada
 * dua tab lainnya, dan dari Master Bengkel yang 20. Tidak ada satu angka yang benar untuk
 * seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 30

/** Kolom penanda yang nilai sahnya tidak ada di export; pilihannya dikumpulkan dari data. */
const MARK_COLUMNS = [
  'jenis_sparepart',
  'satuan',
  'status_aktif',
  'status_sparepart',
] as const

/**
 * Kalimat yang mengisi badan grid ketika tidak ada satu pun baris yang tergambar.
 *
 * # Satu kalimat untuk SETIAP keadaan nol baris
 *
 * Grid Pega menggambar kepala kolomnya beserta satu pesan di bawahnya saat hasilnya nol,
 * dan pesannya berbunyi "data tidak ada". Layar ini mengikutinya apa adanya (`D-13`):
 * berhasil-tetapi-kosong dan gagal-dibaca memakai kalimat yang SAMA.
 *
 * Teksnya sendiri TIDAK dapat dibaca dari export — grid lama mengambilnya dari field value
 * `GridNoResultsOnLoad`, dan tidak ada satu pun direktori `Field Value/` di sana (`R-16`).
 * Ia datang dari Work Owner yang membaca layar Pega sungguhan (2026-09-24). Tanpa nama tab,
 * persis seperti Pega: pesannya sama di ketiga tab.
 *
 * # Yang hilang karenanya, dan di mana menggantinya
 *
 * Layar TIDAK LAGI membedakan "tabelnya memang kosong" dari "tabelnya gagal dibaca".
 * Keduanya tampil identik. Pada saat tulisan ini dibuat, view `POOLDATA.SPAREPART_HE`
 * sedang rusak di Oracle (`ORA-04063`), dan layar ini menampilkannya sebagai data kosong.
 *
 * Keputusan Work Owner, diambil setelah akibatnya disampaikan tiga kali (2026-09-24). Yang
 * menggantikan pembedaan itu ada di dua tempat yang TIDAK dilihat pengguna:
 *
 *	log backend          setiap kegagalan tercatat lengkap dengan galat Oracle-nya
 *	claimpnc -periksa    menyebut objek dan galatnya, beserta kueri katalog penjawabnya
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
 * Layar Master Sparepart.
 *
 * Pengganti `Harness/SparePart_HE-Harness.xml` atas tabel POOLDATA.SPAREPART_HE
 * (MENU_ID 31).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **suku cadang alat berat** beserta harga jual, dimensi, batas stok, dan
 * penggolongannya. Setiap sparepart menunjuk satu Kategori dan satu Tipe yang dibaca dari
 * dua tabel acuan.
 *
 * # Lima kolom, bukan dua puluh tiga
 *
 * Grid Pega hanya menampilkan ID, Nama, Harga Jual, User Update, dan Tanggal Update; sisanya
 * hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru apa adanya atas keputusan Work
 * Owner (2026-09-20), termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * Satu kolom Pega TIDAK digambar: sel yang di layar lama terikat pada `.TELP_BENGKEL` —
 * properti yang tidak ada di SPAREPART_HE, jadi selalu kosong. Ia sisa salin-tempel dari
 * grid Master Bengkel, dan menggambar kolom yang selalu kosong bukan kesetaraan melainkan
 * peniruan cacat.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateSparepartHE_act` menetapkan `APPROVAL := "0"` tanpa syarat apa pun.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterSparepartHE` dipakai Inbox
 * Manager, dan keputusannya dijalankan `Activity/SetApprovalAllMaster` yang melayani
 * bengkel, panel, dan sparepart sekaligus.
 *
 * Inbox Manager belum dibangun. Menunda keputusannya sampai layar itu ada berarti setiap
 * sparepart yang ditambah tertahan di Waiting Approval tanpa satu pun cara menyelesaikannya.
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris, lalu
 * satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Panel dan Master
 * Bengkel.
 *
 * # Tanpa isian Catatan, berbeda dari Master Panel
 *
 * `POOLDATA.SPAREPART_HE` tidak punya kolom penampung alasan penolakan — kedua puluh tiga
 * kolomnya terbaca lengkap dari `BrowseSparepartHE_RD`, dan tidak satu pun menampungnya.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak menggambarnya.
 */
export function SparepartPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Sparepart | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useSparepartList(active.status)
  const create = useCreateSparepart()
  const save = useSaveSparepart()
  const decide = useDecideSparepart()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.sparepart ?? []

  /*
    Pilihan nilai untuk keempat penanda yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia jawaban
    terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" — lihat
    ChoiceField.
  */
  const knownValues = useMemo(() => {
    const collected: Record<string, string[]> = {}
    for (const column of MARK_COLUMNS) {
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

  function openEdit(row: Sparepart) {
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

  function submit(values: SparepartFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form berisi dua puluh isian,
    // itu kehilangan yang tidak dapat dimaafkan.
    if (editing) {
      save.mutate({ id: editing.id_sparepart, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — kelimanya berdampingan pada urutan yang
    sama.

    Dua kolomnya di layar lama TIDAK punya judul sama sekali: selnya bertuliskan
    `.HARGA_JUAL` dan `.TGL_UPDATE_HARGA`, yakni nama propertinya sendiri yang bocor ke
    layar. Judulnya di sini diisi kata yang benar — "Harga Jual" dan "Tanggal Update" —
    karena nama properti yang bocor bukan tata letak yang layak ditiru, melainkan cacat.
  */
  const columns: Column<Sparepart>[] = [
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: Sparepart) => (chosen.has(row.id_sparepart) ? 'dipilih' : ''),
            render: (row: Sparepart) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_sparepart)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_sparepart)}
                />
                <span className="sr-only">Pilih {row.nama_sparepart}</span>
              </label>
            ),
          } satisfies Column<Sparepart>,
        ]
      : []),
    {
      key: 'id',
      title: 'ID Sparepart',
      width: '9rem',
      value: (row) => row.id_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Sparepart',
      width: '16rem',
      value: (row) => row.nama_sparepart,
    },
    {
      key: 'harga',
      title: 'Harga Jual',
      width: '10rem',
      alignRight: true,
      // Yang dicari dan diurutkan adalah teks aslinya, sedangkan yang dilihat pengguna
      // adalah bentuk berpemisah ribuan. Mengurutkan bentuk terformat akan menaruh
      // "1.000.000" sebelum "900.000".
      value: (row) => row.harga_jual,
      render: (row) => (
        <span className="tabular-nums text-sm text-slate-900">{rupiah(row.harga_jual)}</span>
      ),
    },
    {
      key: 'user',
      title: 'User Update',
      width: '10rem',
      value: (row) => row.user_update,
      render: (row) => (
        <span className="text-sm text-slate-900">{row.user_update || '—'}</span>
      ),
    },
    {
      key: 'tanggal',
      title: 'Tanggal Update',
      width: '12rem',
      value: (row) => row.tanggal_update_harga ?? '',
      render: (row) => (
        <span className="text-sm text-slate-900">{tanggalWIB(row.tanggal_update_harga)}</span>
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
          {/* Judulnya dibaca dari `pyCaption Master Sparepart HE` pada
              Harness/SparePart_HE — "HE" ikut, karena itulah yang tertulis di layar lama
              (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Sparepart HE</h1>
          <p className="text-sm text-slate-600">
            Suku cadang alat berat beserta harga jual, dimensi, dan batas stoknya.
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

      {/*
        Kedua tombol unggah layar Pega — "Upload Document" dan "Upload Data Master Sparepart"
        (`pyButtonLabel` pada Section/BrowseMasterSparepartHE) — SENGAJA TIDAK DIGAMBAR.

        Keduanya memanggil local action `UploadDocument` dan `PNCUploadMasterSparepartCSV`;
        tidak satu pun ada di export (`R-16`), sehingga susunan kolom CSV-nya, validasinya,
        dan — yang paling menentukan — apakah baris hasil unggah masuk antrean persetujuan,
        seluruhnya tidak diketahui.

        Sempat digambar dalam keadaan mati supaya ketiadaannya terbaca dari layar. Work Owner
        memilih menariknya sama sekali (2026-09-24), menyamakannya dengan Master Panel (§28);
        lihat docs/keputusan-implementasi.md §30.9.
      */}
      <nav
        aria-label="Tab Master Sparepart"
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
          {decide.data.jumlah_berubah} sparepart dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          isBusy={decide.isPending}
          onApprove={() => runDecision(SparepartStatus.disetujui)}
          onReject={() => runDecision(SparepartStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <SparepartForm
            editing={editing}
            knownValues={knownValues}
            isSaving={create.isPending || save.isPending}
            error={editing ? save.error : create.error}
            onSave={submit}
            onCancel={closeForm}
          />
        </section>
      )}

      {/*
        Tabelnya digambar dalam SETIAP keadaan — termuat, kosong, gagal, bahkan sebelum
        portal dipilih. Kolomnya karena itu selalu terlihat, persis seperti grid Pega yang
        menggambar kepala kolomnya beserta pyGridNoResultsMessage di bawahnya.

        Tidak ada lagi kotak galat yang menggantikan tabelnya: keterangan apa pun tinggal di
        dalam grid lewat emptyMessage. Lihat emptyMessageFor untuk alasan gagal dan kosong
        tetap dibedakan kalimatnya.
      */}
      <section className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_sparepart}
          description="Sumber: POOLDATA.SPAREPART_HE"
          searchLabel="Cari sparepart"
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
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang.
 *
 * Ia padanan `Section/ApprovalMasterSparepartHE-Section.xml` yang menyediakan Select All,
 * Deselect All, Approve, dan Reject.
 *
 * TANPA isian Catatan, berbeda dari Master Panel: `POOLDATA.SPAREPART_HE` tidak punya kolom
 * penampungnya. Lihat catatan pada SparepartPage.
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
          'Centang sparepart yang akan diputuskan.'
        ) : (
          <>
            <span className="font-medium">{count} sparepart</span> dipilih.
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

/**
 * rupiah menampilkan nilai harga dengan pemisah ribuan.
 *
 * Nilai yang BUKAN angka ditampilkan apa adanya, tidak dikosongkan: baris lama dapat memuat
 * apa saja di kolom itu — sistem lama tidak pernah memeriksanya — dan menyembunyikannya
 * berarti petugas tidak punya cara mengetahui bahwa isinya perlu diperbaiki.
 */
function rupiah(value: string): string {
  const clean = value.trim()
  if (clean === '') return '—'

  const number = Number(clean.replace(',', '.'))
  if (!Number.isFinite(number)) return clean

  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(number)
}

/**
 * tanggalWIB menampilkan stempel UTC dari server dalam waktu Jakarta.
 *
 * Konversi terjadi DI SINI, di satu tempat, lewat Intl — tidak ada satu pun penambahan 7 jam
 * manual (`F-5`, `08-TECHNICAL-STRATEGY.md` §4.4).
 */
function tanggalWIB(value: string | undefined): string {
  if (!value) return '—'

  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return value

  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Jakarta',
  }).format(at)
}
