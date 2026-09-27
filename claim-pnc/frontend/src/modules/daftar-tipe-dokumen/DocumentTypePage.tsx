import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DocumentType } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateDocumentType,
  useDocumentTypeList,
  useUpdateDocumentType,
} from './api'
import { DocumentTypeForm, type DocumentTypeFields } from './DocumentTypeForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | DocumentType

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
            'Daftar tipe dokumen dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
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
 * Layar Daftar Tipe Dokumen.
 *
 * Pengganti `Harness/ListDocumentTypeInbox-Harness.xml` atas tabel POOLDATA.LST_DOC_TYPE.
 * Judul, susunan kolom, dan tombolnya mengikuti layar lama (`D-13`: alur dan tata letak
 * ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Tombol "Tambah" dan "Refresh"   — `Section/ListDocumentType-Section.xml`
 *   - Grid tiga kolom: ID, Tipe Dokumen, Status Proses
 *                                     — `Section/BrowseListDocumentType-Section.xml`
 *   - Aksi "Update Data" per baris    — memanggil `CNMSetListDocumentType_act`
 *   - 50 baris per halaman            — `BrowseLstDocType_RD-RD.xml`, `pyPageSize=50`
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Entitas yang dilihat | tidak pernah disebut | disebut terang-terangan (`R-20`) |
 * | Arti "Status Proses" | tidak dijelaskan | diberi keterangan di form |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi. Nama kosong diterima dan nama ganda diterima, persis seperti layar
 * lama — keputusan Work Owner 2026-09-21. Ini berbeda dari Master Status Klaim, yang justru
 * diputuskan diperketat pada 2026-09-17; perbedaan keduanya disengaja dan dicatat di
 * docs/keputusan-implementasi.md.
 *
 * Tidak ada kotak cari. Layar lama tidak punya penyaring sama sekali — keputusan Work Owner
 * 2026-09-21, sama seperti Master Dokumen Travel dan berbeda dari Master Status Klaim.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, `PEGA_LST_DOC_TYPE.prc` hanya mengenal
 * INSERT dan UPDATE, dan ID-nya dirujuk dua master turunan beserta dokumen klaim yang sudah
 * terunggah (`ADR-0012`, `D-66`).
 */
export function DocumentTypePage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useDocumentTypeList()
  const create = useCreateDocumentType()
  const update = useUpdateDocumentType()

  const edited = typeof form === 'string' ? null : form
  const isSaving = create.isPending || update.isPending
  const saveError = edited ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: DocumentType) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: DocumentTypeFields) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    if (edited) {
      update.mutate({ id: edited.id, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  // `value` dipisah dari `render` mengikuti kontrak Column: yang diurutkan adalah teks
  // polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<DocumentType>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-28',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      // "Tipe Dokumen" — header kolom pada grid Pega
      // (`Section/BrowseListDocumentType-Section.xml`, pyCaption "Tipe Dokumen"). Ia
      // sengaja BERBEDA dari label isian di form, yang berbunyi "Jenis Dokumen"; kedua teks
      // itu memang tidak sama di layar lama, dan keduanya ditiru apa adanya (`D-13`).
      key: 'tipe_dokumen',
      title: 'Tipe Dokumen',
      value: (row) => row.tipe_dokumen,
      render: (row) =>
        row.tipe_dokumen ? (
          <span className="font-medium text-slate-900">{row.tipe_dokumen}</span>
        ) : (
          // Nama kosong memang dapat tersimpan — layar lama pun menerimanya. Ia ditandai
          // supaya barisnya tidak tampak seperti baris rusak atau sel yang gagal dimuat.
          <span className="text-slate-400" title="Baris ini tersimpan tanpa nama">
            (tanpa nama)
          </span>
        ),
    },
    {
      key: 'status_proses',
      title: 'Status Proses',
      width: 'w-56',
      value: (row) => row.status_proses,
      // Dirender sebagai TEKS BIASA, bukan lencana berwarna. Ia catatan bebas, bukan
      // status — merendernya sebagai lencana akan menyiratkan domain tertutup yang tidak
      // ada, dan membuat nilai yang tidak dikenali tampak seperti data rusak.
      render: (row) =>
        row.status_proses ? (
          <span className="text-slate-700">{row.status_proses}</span>
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-32',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya tombol,
      // bukan data.
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => openEdit(row)}
          aria-label={`Ubah tipe dokumen ${row.tipe_dokumen || row.id}`}
        >
          Update Data
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Daftar Tipe Dokumen</h1>
          <p className="text-sm text-slate-600">
            Kategori dokumen yang dapat dilampirkan pada klaim.
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

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (`ADR-0030`, `R-20`). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {form !== CLOSED && (
        <section className="mt-5">
          <DocumentTypeForm
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
            description="Daftar tipe dokumen dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar tipe dokumen…</p>
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
            rows={list.data.tipe_dokumen}
            rowKey={(row) => row.id}
            // Tanpa kotak cari, dan dipaginasi 50 baris per halaman — keduanya meniru grid
            // Pega apa adanya (`pyPageSize=50`, dan tidak ada satu pun penyaring di
            // sectionnya). Keputusan Work Owner 2026-09-21.
            searchable={false}
            pageSize={50}
            description={`${list.data.total} tipe dokumen terdaftar. Sumber: POOLDATA.LST_DOC_TYPE`}
            emptyMessage="Belum ada tipe dokumen pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
