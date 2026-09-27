import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DocumentObject } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  useBusinessList,
  useCreateDocumentObject,
  useDocumentObject,
  useDocumentObjectList,
  useUpdateDocumentObject,
} from './api'
import { DocumentObjectForm, type DocumentObjectFields } from './DocumentObjectForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | DocumentObject

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

/**
 * Layar Daftar Objek Dokumen (MENU_ID 43).
 *
 * Pengganti `Harness/ListDocumentObject-Harness.xml`. Judul, susunan kolom, dan kedua
 * tombolnya mengikuti layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak
 * perlu belajar ulang):
 *
 *   - Judul "Daftar Objek Dokumen"          — `Section/BrowseDocumentObject-Section.xml:6715`
 *   - Tombol "Tambah" dan "Refresh"          — `Section/ListDocumentObject-Section.xml`
 *   - Grid dua kolom: ID dan keterangannya   — `Section/BrowseDocumentObject-Section.xml`
 *   - Aksi baris "Ubah"                      — `pyButtonLabel` pada section yang sama
 *
 * Yang SENGAJA tidak ada:
 *
 *   - Tombol hapus. Seluruh grid di layar lama ber-`pyGridDeleteActivityExists=false`, dan
 *     `D-66` melarang penghapusan fisik data bernilai bisnis. Barisnya pun dirujuk
 *     `LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID` pada data yang sudah berjalan — menghapusnya
 *     akan memutus rujukan itu.
 *   - Kolom ID lama (OLD_ID). Server mengirimkannya karena Report Definition memuatnya,
 *     tetapi grid Pega hanya memuat DUA kolom. Ketiadaannya di layar adalah keputusan,
 *     bukan kelalaian.
 */
export function DocumentObjectPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useDocumentObjectList()
  const businesses = useBusinessList()
  const create = useCreateDocumentObject()
  const update = useUpdateDocumentObject()

  const openedRow = typeof form === 'string' ? null : form

  // Baris yang dibuka dimuat ULANG dari server supaya pemetaan bisnisnya ikut terbawa.
  // Daftar sengaja tidak membawanya — grid hanya menampilkan ID dan keterangannya —
  // sehingga baris yang diambil dari daftar SELALU punya `bisnis: []`. Memakainya langsung
  // akan membuat form tampak seolah seluruh bisnisnya sudah dihapus, dan menyimpannya
  // benar-benar mencabutnya.
  const detail = useDocumentObject(openedRow?.id ?? null)

  const edited = openedRow === null ? null : (detail.data?.objek_dokumen ?? openedRow)
  const isSaving = create.isPending || update.isPending
  const saveError = openedRow ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: DocumentObject) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: DocumentObjectFields) {
    const input = {
      objek_dokumen: values.objek_dokumen,
      // NAMA yang dikirim, bukan ID: itulah yang diketik dan dilihat petugas, dan nama yang
      // diketik bebas memang tidak punya ID. Server yang menyelesaikannya menjadi ID dengan
      // mencocokkan ke master.
      //
      // Baris yang dibiarkan kosong dibuang di sini supaya tidak terkirim sebagai pemetaan
      // ke bisnis bernama kosong — server pun membuangnya, tetapi membuangnya lebih awal
      // membuat permintaannya menyatakan apa yang benar-benar dimaksud.
      bisnis: values.bisnis.map((b) => b.nama.trim()).filter((nama) => nama !== ''),
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

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan adalah
  // teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<DocumentObject>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-28',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      key: 'objek_dokumen',
      // Judul kolom diambil apa adanya dari sel ber-`pyCellHeader=true` di layar Pega. Ia
      // memang sama dengan judul layarnya, dan itu dibiarkan — `D-13` menetapkan teks layar
      // ditiru, bukan dirapikan.
      title: 'Daftar Objek Dokumen',
      value: (row) => row.objek_dokumen,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-24',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya tombol,
      // bukan data.
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => openEdit(row)}
          aria-label={`Ubah objek dokumen ${row.objek_dokumen || row.id}`}
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
          {/* Judulnya diambil apa adanya dari caption layar Pega, supaya pengguna
              mengenalinya tanpa diberi tahu. */}
          <h1 className="text-xl font-semibold text-slate-900">Daftar Objek Dokumen</h1>
          <p className="text-sm text-slate-600">
            Objek yang melekat pada dokumen klaim, beserta bisnis yang memakainya.
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
          {/* Kegagalan memuat daftar bisnis TIDAK menutup form dan tidak menghalangi
              penyimpanan: nama bisnis memang boleh diketik sendiri. Yang hilang hanya
              sarannya, dan form itu sendiri yang mengatakannya. */}
          {businesses.isError && (
            <div className="mb-3">
              <ErrorMessage
                title="Daftar bisnis tidak dapat dimuat"
                description="Saran nama bisnis tidak tersedia untuk sementara. Namanya tetap dapat diketik sendiri, dan seluruh isian tetap dapat disimpan."
                tone="gangguan"
              />
            </div>
          )}
          <DocumentObjectForm
            edited={edited}
            businesses={businesses.data?.bisnis ?? []}
            isLoadingBusinessMapping={openedRow !== null && detail.isPending}
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
          <p className="text-sm text-slate-500">Memuat daftar objek dokumen…</p>
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
            rows={list.data.objek_dokumen}
            rowKey={(row) => row.id}
            description={`${list.data.total} objek dokumen terdaftar. Sumber: POOLDATA.V_LST_DOC_OBJ`}
            emptyMessage="Belum ada objek dokumen pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
