import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DetailDocumentType } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateDetailDocumentType,
  useDetailDocumentTypeList,
  useDetailDocumentTypeReferences,
  useUpdateDetailDocumentType,
} from './api'
import { DetailDocumentTypeForm, type DetailDocumentTypeFields } from './DetailDocumentTypeForm'

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

/**
 * Nama master yang gagal dibaca, diubah menjadi kata yang dikenali petugas.
 *
 * Hanya memuat master yang BENAR-BENAR dipakai sebuah isian di form. Server masih ikut
 * melaporkan `penyebab_kerugian` karena endpoint pilihannya tidak berubah, tetapi layar
 * ini tidak lagi punya isian yang memakainya — menyebutnya dalam peringatan hanya akan
 * membingungkan petugas, karena tidak ada yang bisa dikerjakannya.
 */
const REFERENCE_LABEL: Record<string, string> = {
  tipe_dokumen: 'Tipe Dokumen',
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
 *
 * # Kolomnya tepat seperti Pega
 *
 * Grid ini memuat ID, Tipe Dokumen, Detail Dokumen, dan aksi Ubah — tidak lebih. Kolom
 * Objek Dokumen dan Penyebab Kerugian sempat ditambahkan di sini dengan alasan
 * memudahkan, lalu DICABUT atas koreksi Work Owner (2026-10-03): grid Pega tidak
 * memuatnya, dan menambah kolom yang tidak ada di layar lama adalah perubahan tata letak
 * yang `D-13` justru hendak dicegah.
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

  // Baris dari daftar dipakai langsung. Daftar membawa SELURUH kolom yang dibutuhkan
  // form maupun penyimpanan — termasuk keenam kolom yang tidak disunting tetapi harus
  // ikut ditulis kembali — sehingga tidak ada lagi alasan memuat ulang satu baris.
  const edited = openedRow
  const isSaving = create.isPending || update.isPending
  const saveError = openedRow ? update.error : create.error

  // Master yang gagal dibaca disebut server lewat `tidak_tersedia`. Kegagalan seluruh
  // permintaannya — jaringan, portal — diperlakukan sebagai keempatnya hilang, karena
  // memang tidak satu pun daftar yang di tangan.
  const unavailable = (
    references.isError ? Object.keys(REFERENCE_LABEL) : (references.data?.tidak_tersedia ?? [])
  ).filter((name) => name in REFERENCE_LABEL)

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
    const input = {
      id_tipe_dokumen: values.id_tipe_dokumen,
      detail_dokumen: values.detail_dokumen,
      // Keenam kolom berikut tidak punya isian di layar ini — layar Pega yang berjalan
      // pun hanya meminta dua isian di atas. Nilainya DIBAWA APA ADANYA dari baris yang
      // sedang disunting, bukan dikirim kosong: mengirimnya kosong akan menghapus isi
      // kolomnya pada setiap penyimpanan, padahal petugas tidak pernah diberi kesempatan
      // mengubahnya. Pada penambahan baru keenamnya memang belum punya isi.
      status_tertanggung: edited?.status_tertanggung ?? '',
      id_penyebab_kerugian: edited?.id_penyebab_kerugian ?? '',
      keterangan_penyebab_kerugian: edited?.keterangan_penyebab_kerugian ?? '',
      id_objek_dokumen: edited?.id_objek_dokumen ?? '',
      keterangan_objek_dokumen: edited?.keterangan_objek_dokumen ?? '',
      resiko: edited?.resiko ?? '',
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
      // NAMANYA saja — kodenya tidak ditampilkan, sama seperti grid Pega. Kode itu
      // sempat ikut tampil sebagai baris kecil di bawah namanya dan DICABUT atas
      // koreksi Work Owner (2026-10-03).
      value: (row) => row.nama_tipe_dokumen,
    },
    { key: 'detail_dokumen', title: 'Detail Dokumen', value: (row) => row.detail_dokumen },
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
    <main className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya diambil apa adanya dari caption layar Pega, supaya pengguna
              mengenalinya tanpa diberi tahu. */}
          <h1 className="text-xl font-semibold text-slate-900">Detail Tipe Dokumen</h1>
          {/* Keterangannya sengaja satu baris pendek, sama seperti layar master lain.
              Keterangan panjang membuat blok judul melebar sampai tombol Refresh dan
              Tambah terdorong turun ke baris berikutnya — dan di sana ia rata kiri,
              tidak lagi di kanan seperti layar lain. */}
          <p className="text-sm text-slate-600">
            Rincian dokumen di bawah setiap tipe dokumen klaim.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
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
                  .join(', ')} tidak tersedia untuk sementara. Isian lain tetap dapat disimpan seperti biasa.`}
                tone="gangguan"
              />
            </div>
          )}
          <DetailDocumentTypeForm
            edited={edited}
            documentTypes={references.data?.tipe_dokumen ?? []}
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
