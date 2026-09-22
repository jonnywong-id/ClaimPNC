import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type TravelDocument } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateTravelDocument,
  useTravelDocumentList,
  useUpdateTravelDocument,
} from './api'
import { TravelDocumentForm, type TravelDocumentFields } from './TravelDocumentForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | TravelDocument

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
            'Daftar dokumen Travel dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
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

/**
 * Layar Master Dokumen Travel.
 *
 * Pengganti `Harness/BrowseMasterDocumentTravel_Harness-Harness.xml` atas tabel
 * POOLDATA.M_DOCTRAVEL. Judul, susunan kolom, dan tombolnya mengikuti layar lama
 * (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Judul "Master Dokumen Travel"  — `Section/BrowseMasterDocumentTravel-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"  — section yang sama
 *   - Grid dua kolom: ID dan Judul Dokumen — `BrowseMstDocTravel_RD-RD.xml`
 *   - Aksi "Ubah" per baris          — memanggil `SetMstDocTravelValue_act(docid)`
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri kedua kolom |
 * | Entitas yang dilihat | tidak pernah disebut | disebut terang-terangan (`R-20`) |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi. Judul kosong diterima dan judul ganda diterima, persis seperti
 * layar lama — keputusan Work Owner 2026-09-21. Ini berbeda dari Master Status Klaim,
 * yang justru diputuskan diperketat pada 2026-09-17; perbedaan keduanya disengaja dan
 * dicatat di docs/keputusan-implementasi.md.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, `DOCTRAVEL_CVG.prc` hanya mengenal
 * INSERT dan UPDATE, dan DOCID dirujuk baris V_LST_DOC_TRAVEL beserta dokumen klaim yang
 * sudah terunggah (`ADR-0012`).
 */
export function TravelDocumentPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useTravelDocumentList()
  const create = useCreateTravelDocument()
  const update = useUpdateTravelDocument()

  const edited = typeof form === 'string' ? null : form
  const isSaving = create.isPending || update.isPending
  const saveError = edited ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: TravelDocument) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: TravelDocumentFields) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    if (edited) {
      update.mutate({ id: edited.id, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<TravelDocument>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-32',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      // "Judul Dokumen Travel" — header kolom kedua pada grid Pega
      // (`Section/BrowseMasterDocumentTravel-Section.xml`, Embed-Display-Table-Cell
      // indeks 2). Ia sengaja BERBEDA dari label isian di form, yang berbunyi "Judul
      // Dokumen" saja; kedua teks itu memang tidak sama di layar lama, dan keduanya
      // ditiru apa adanya (`D-13`, keputusan Work Owner 2026-09-21).
      key: 'judul',
      title: 'Judul Dokumen Travel',
      value: (row) => row.judul,
      render: (row) =>
        row.judul ? (
          <span className="font-medium text-slate-900">{row.judul}</span>
        ) : (
          // Judul kosong memang dapat tersimpan — layar lama pun menerimanya. Ia
          // ditandai supaya barisnya tidak tampak seperti baris rusak atau sel yang
          // gagal dimuat.
          <span className="text-slate-400" title="Baris ini tersimpan tanpa judul">
            (tanpa judul)
          </span>
        ),
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
          aria-label={`Ubah dokumen ${row.judul || row.id}`}
        >
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Dokumen Travel</h1>
          <p className="text-sm text-slate-600">
            Daftar jenis dokumen yang dapat diminta pada klaim lini Travel.
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
          <TravelDocumentForm
            edited={edited}
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
            description="Daftar dokumen Travel dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar dokumen travel…</p>
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
            rows={list.data.dokumen_travel}
            rowKey={(row) => row.id}
            // Tanpa kotak cari, dan dipaginasi 50 baris per halaman — keduanya meniru
            // grid Pega apa adanya (`pyPageSize=50`, dan tidak ada satu pun
            // `pySortFilterProperty` yang terisi). Keputusan Work Owner 2026-09-21.
            //
            // Berbeda dari Master Status Klaim, yang justru ditambahi kotak cari pada
            // 2026-09-17. Perbedaan itu disengaja dan dicatat di
            // docs/keputusan-implementasi.md.
            searchable={false}
            pageSize={50}
            description={`${list.data.total} dokumen terdaftar. Sumber: POOLDATA.M_DOCTRAVEL`}
            emptyMessage="Belum ada dokumen travel pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
