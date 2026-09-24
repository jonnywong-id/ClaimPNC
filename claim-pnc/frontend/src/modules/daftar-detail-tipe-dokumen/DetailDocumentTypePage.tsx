import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DetailDocumentType } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateDetailDocumentType,
  useDetailDocumentType,
  useDetailDocumentTypeList,
  useDetailDocumentTypeReferences,
  useUpdateDetailDocumentType,
} from './api'
import {
  codeOf,
  DetailDocumentTypeForm,
  MANDATORY_YES,
  type DetailDocumentTypeFields,
} from './DetailDocumentTypeForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | DetailDocumentType

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
          title: 'Daftar tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/** Nama master yang gagal dibaca, diubah menjadi kata yang dikenali petugas. */
const REFERENCE_LABEL: Record<string, string> = {
  tipe_dokumen: 'Tipe Dokumen',
  penyebab_kerugian: 'Penyebab Kerugian',
  objek_dokumen: 'Objek Dokumen',
  bisnis: 'Bisnis',
}

/**
 * Layar Daftar Detail Tipe Dokumen.
 *
 * Pengganti `Harness/ListDetTypeDocument-Harness.xml` atas view
 * POOLDATA.V_LST_DET_TYPE_DOC beserta aturan per lini bisnisnya. Judul, susunan kolom, dan
 * tombolnya mengikuti layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak
 * perlu belajar ulang):
 *
 *   - Judul "Detail Tipe Dokumen"     — `Section/DetTypeDocument-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"   — section yang sama
 *   - Grid tiga kolom                 — `Section/BrowseListDetailTypeDocument-Section.xml`
 *   - Aksi "Ubah" per baris           — section yang sama
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri seluruh kolom |
 * | Entitas yang dilihat | tidak pernah disebut | disebut terang-terangan (`R-20`) |
 * | Batas 500 baris | `pyMaxRecords=500` | dihapus — pemotongan senyap, bukan aturan |
 * | Keterangan rujukan | hanya di dalam autocomplete | ikut tampil di grid dan di form |
 *
 * Kolom keterangan penyebab kerugian dan objek dokumen ikut ditampilkan meski grid Pega
 * tidak memuatnya. Sebabnya bukan hiasan: tanpa keduanya, satu-satunya cara mengetahui
 * rincian dokumen ini melekat pada objek apa adalah membuka barisnya satu per satu.
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi isian. Detail Dokumen kosong diterima, rujukan kosong diterima, dan
 * kode yang tidak ada di master pun diterima — persis seperti layar lama, yang tidak
 * memuat satu pun `pyRequired` bernilai true maupun Rule-Obj-Validate. Yang diperiksa
 * hanyalah panjang isian dan bentuk angka, dan keduanya bentuk kolom — bukan aturan
 * bisnis.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, dan ID baris ini dirujuk
 * `LST_TYPE_DOC_BUSINESS.DOC_TYPE_DT_ID` milik MENU_ID 42 beserta 34 rule pembaca lain —
 * menghapusnya memutus seluruhnya (`D-66`, `ADR-0012`).
 */
export function DetailDocumentTypePage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useDetailDocumentTypeList()
  const references = useDetailDocumentTypeReferences()
  const create = useCreateDetailDocumentType()
  const update = useUpdateDetailDocumentType()

  const openedRow = typeof form === 'string' ? null : form

  // Baris yang dibuka dimuat ULANG dari server supaya daftar bisnisnya ikut terbawa.
  // Daftar sengaja tidak membawanya — grid hanya menampilkan tiga kolom — sehingga baris
  // yang diambil dari daftar SELALU punya `bisnis: []`. Memakainya langsung akan membuat
  // form tampak seolah seluruh lini bisnisnya sudah dihapus, dan menyimpannya benar-benar
  // menghapusnya.
  const detail = useDetailDocumentType(openedRow?.id ?? null)

  const edited = openedRow === null ? null : (detail.data?.detail_tipe_dokumen ?? openedRow)
  const isSaving = create.isPending || update.isPending
  const saveError = openedRow ? update.error : create.error

  // Master yang gagal dibaca disebut server lewat `tidak_tersedia`. Kegagalan seluruh
  // permintaannya — jaringan, portal — diperlakukan sebagai keempatnya hilang, karena
  // memang tidak satu pun daftar yang di tangan.
  const unavailable = references.isError
    ? Object.keys(REFERENCE_LABEL)
    : (references.data?.tidak_tersedia ?? [])

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: DetailDocumentType) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: DetailDocumentTypeFields) {
    const causes = references.data?.penyebab_kerugian ?? []
    const objects = references.data?.objek_dokumen ?? []

    const input = {
      id_tipe_dokumen: values.id_tipe_dokumen,
      detail_dokumen: values.detail_dokumen,
      status_tertanggung: values.status_tertanggung,
      // Kode dicarikan dari KETERANGAN yang diketik atau dipilih petugas — itulah yang
      // dilihatnya di layar, dan itulah yang dikerjakan `pyPropertyTarget` pada
      // autocomplete Pega saat sebuah pilihan dipilih.
      //
      // Keterangan yang tidak ada di master memang tidak punya kode, dan barisnya TETAP
      // dikirim tanpa kode: `pyAllowFreeFormInput=true` mengizinkannya, dan kolom
      // keterangannya yang menyimpan isian itu.
      //
      // Satu penyimpangan kecil yang disengaja: Pega hanya mengisi kodenya saat pilihan
      // benar-benar DIPILIH dari daftar, sehingga keterangan yang diketik ulang persis
      // sama akan meninggalkan kode LAMA yang tidak lagi cocok. Di sini kodenya selalu
      // dicocokkan ulang terhadap keterangannya, sehingga keduanya tidak pernah
      // menyimpang. Perbedaannya tidak terlihat pengguna, dan yang dihasilkan lebih
      // konsisten.
      id_penyebab_kerugian: codeOf(causes, values.keterangan_penyebab_kerugian) ?? '',
      keterangan_penyebab_kerugian: values.keterangan_penyebab_kerugian,
      id_objek_dokumen: codeOf(objects, values.keterangan_objek_dokumen) ?? '',
      keterangan_objek_dokumen: values.keterangan_objek_dokumen,
      resiko: values.resiko,
      bisnis: values.bisnis
        // Baris yang bisnisnya belum dipilih dibuang di sini supaya tidak terkirim sebagai
        // aturan yang tidak menunjuk lini bisnis mana pun. Server pun membuangnya, tetapi
        // membuangnya lebih awal membuat permintaannya menyatakan apa yang benar-benar
        // dimaksud.
        .filter((row) => row.id_bisnis.trim() !== '')
        .map((row) => ({
          id_bisnis: row.id_bisnis.trim(),
          status_wajib: row.status_wajib === MANDATORY_YES,
          // Isian kosong dibaca sebagai nol — pada isian yang kosongnya sah, "belum
          // diisi" tidak boleh menjadi penolakan. Bentuknya sudah dijaga skema form
          // berupa angka saja, sehingga Number() di sini tidak pernah menghasilkan NaN.
          minimum_dokumen: Number(row.minimum_dokumen || '0'),
        })),
    }

    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    if (openedRow) {
      update.mutate({ id: openedRow.id, input }, { onSuccess: closeForm })
      return
    }
    create.mutate(input, { onSuccess: closeForm })
  }

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<DetailDocumentType>[] = [
    { key: 'id', title: 'ID', width: 'w-28', value: (row) => row.id },
    {
      key: 'tipe_dokumen',
      title: 'Tipe Dokumen',
      width: 'w-56',
      // Kode DAN namanya sama-sama dapat dicari. Petugas yang hafal kodenya dan petugas
      // yang hafal namanya sama-sama dilayani satu kotak cari.
      value: (row) => `${row.id_tipe_dokumen} ${row.nama_tipe_dokumen}`,
      render: (row) => (
        <span>
          <span className="block">{row.nama_tipe_dokumen || '—'}</span>
          <span className="block text-xs text-slate-500">{row.id_tipe_dokumen}</span>
        </span>
      ),
    },
    { key: 'detail_dokumen', title: 'Detail Dokumen', value: (row) => row.detail_dokumen },
    {
      key: 'objek_dokumen',
      title: 'Objek Dokumen',
      width: 'w-48',
      value: (row) => row.keterangan_objek_dokumen,
    },
    {
      key: 'penyebab_kerugian',
      title: 'Penyebab Kerugian',
      width: 'w-48',
      value: (row) => row.keterangan_penyebab_kerugian,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-24',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya
      // tombol, bukan data.
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => openEdit(row)}
          aria-label={`Ubah ${row.detail_dokumen || row.id}`}
        >
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya diambil apa adanya dari caption layar Pega, supaya pengguna
              mengenalinya tanpa diberi tahu. */}
          <h1 className="text-xl font-semibold text-slate-900">Detail Tipe Dokumen</h1>
          <p className="text-sm text-slate-600">
            Rincian dokumen di bawah setiap tipe dokumen klaim: melekat pada objek apa, dipicu
            penyebab kerugian mana, dan wajib pada lini bisnis mana.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openCreate} disabled={form !== CLOSED}>
            Tambah
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
          empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
          hanya diandaikan pengguna (`ADR-0030`, `R-20`). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {form !== CLOSED && (
        <section className="mt-5">
          {/* Master yang gagal dibaca TIDAK menutup form dan tidak menghalangi
              penyimpanan: keempat kodenya memang boleh diketik sendiri. Yang hilang hanya
              sarannya, dan form itu sendiri yang mengatakannya. */}
          {unavailable.length > 0 && (
            <div className="mb-3">
              <ErrorMessage
                title="Sebagian daftar pilihan tidak dapat dimuat"
                description={`Saran untuk isian ${unavailable
                  .map((name) => REFERENCE_LABEL[name] ?? name)
                  .join(', ')} tidak tersedia untuk sementara. Kodenya tetap dapat diketik sendiri, dan seluruh isian tetap dapat disimpan.`}
                tone="gangguan"
              />
            </div>
          )}
          <DetailDocumentTypeForm
            edited={edited}
            documentTypes={references.data?.tipe_dokumen ?? []}
            causesOfLoss={references.data?.penyebab_kerugian ?? []}
            objectDocuments={references.data?.objek_dokumen ?? []}
            businesses={references.data?.bisnis ?? []}
            isLoadingBusinessRules={openedRow !== null && detail.isPending}
            isSaving={isSaving}
            error={saveError}
            onSave={save}
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
          <p className="text-sm text-slate-500">Memuat daftar detail tipe dokumen…</p>
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
            rows={list.data.detail_tipe_dokumen}
            rowKey={(row) => row.id}
            description="Sumber: POOLDATA.V_LST_DET_TYPE_DOC"
            emptyMessage="Belum ada detail tipe dokumen pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
