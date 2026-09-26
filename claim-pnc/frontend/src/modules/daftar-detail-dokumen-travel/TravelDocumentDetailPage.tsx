import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type TravelDocumentDetail } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateTravelDocumentDetail,
  useTravelDocumentChoiceList,
  useTravelDocumentDetail,
  useTravelDocumentDetailList,
  useTravelPlanList,
  useUpdateTravelDocumentDetail,
} from './api'
import {
  MANDATORY_YES,
  TravelDocumentDetailForm,
  type TravelDocumentDetailFields,
} from './TravelDocumentDetailForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | TravelDocumentDetail

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
 * Layar Daftar Detail Dokumen Travel.
 *
 * Pengganti `Harness/ListDocumentTravel-Harness.xml` atas view POOLDATA.V_LST_DOC_TRAVEL
 * beserta pembatasan plan dan jaminannya. Judul, susunan kolom, dan tombolnya mengikuti
 * layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar
 * ulang):
 *
 *   - Judul "Detail Dokumen Travel"   — `Section/LSTDocumentTravel-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"   — section yang sama
 *   - Grid lima kolom                 — `BrowseLstDocTravel_RD-RD.xml`, urut ID lalu DOCID
 *   - Aksi "Ubah" per baris           — `Section/BrowseDocumentTravel-Section.xml`
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
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi. ID Dokumen kosong diterima, nama dokumen kosong diterima, dan kode
 * yang tidak ada di master pun diterima — persis seperti layar lama, yang tidak memuat
 * satu pun `pyRequired` bernilai true maupun Validate rule. Keputusan Work Owner
 * 2026-09-21, sama dengan modul Master Dokumen Travel.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, dan baris ini menentukan dokumen
 * apa yang diminta pada klaim yang sedang berjalan (`Activity/TravelDocument_act-Act.xml`)
 * — menghapusnya mengubah kelengkapan klaim yang sudah telanjur dinilai (`D-66`,
 * `ADR-0012`).
 */
export function TravelDocumentDetailPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useTravelDocumentDetailList()
  const documents = useTravelDocumentChoiceList()
  const travelPlans = useTravelPlanList()
  const create = useCreateTravelDocumentDetail()
  const update = useUpdateTravelDocumentDetail()

  const openedRow = typeof form === 'string' ? null : form

  // Baris yang dibuka dimuat ULANG dari server supaya pembatasan plan-nya ikut terbawa.
  // Daftar sengaja tidak membawanya — grid hanya menampilkan lima kolom — sehingga baris
  // yang diambil dari daftar SELALU punya `jaminan: []`. Memakainya langsung akan membuat
  // form tampak seolah seluruh pembatasannya sudah dihapus, dan menyimpannya benar-benar
  // menghapusnya.
  const detail = useTravelDocumentDetail(openedRow?.id ?? null)

  const edited = openedRow === null ? null : (detail.data?.detail_dokumen_travel ?? openedRow)
  const isSaving = create.isPending || update.isPending
  const saveError = openedRow ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: TravelDocumentDetail) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: TravelDocumentDetailFields) {
    const plans = travelPlans.data?.plan ?? []
    const coverages = travelPlans.data?.jaminan ?? []

    const input = {
      id_dokumen: values.id_dokumen,
      nama_dokumen: values.nama_dokumen,
      status_wajib: values.status_wajib === MANDATORY_YES,
      // Isian kosong dibaca sebagai nol — pada layar tanpa validasi, "belum diisi" tidak
      // boleh menjadi penolakan. Bentuknya sudah dijaga skema form berupa angka saja,
      // sehingga Number() di sini tidak pernah menghasilkan NaN.
      minimal_unggah: Number(values.minimal_unggah || '0'),
      jaminan: values.jaminan
        // Baris yang dibiarkan kosong seluruhnya dibuang di sini supaya tidak terkirim
        // sebagai pembatasan yang tidak membatasi apa pun. Server pun membuangnya, tetapi
        // membuangnya lebih awal membuat permintaannya menyatakan apa yang benar-benar
        // dimaksud.
        .filter((row) => row.nama_plan.trim() !== '' || row.nama_jaminan.trim() !== '')
        .map((row) => {
          // Kode dicarikan dari NAMA yang diketik atau dipilih petugas — itulah yang
          // dilihatnya di layar. Nama yang tidak ada di master memang tidak punya kode,
          // dan barisnya TETAP dikirim tanpa kode: autocomplete Pega pun menyimpan
          // namanya saja pada keadaan itu (`pyAllowFreeFormInput=true`).
          const planName = row.nama_plan.trim()
          const coverageName = row.nama_jaminan.trim()
          const plan = plans.find((item) => item.nama === planName)
          const coverage = coverages.find(
            (item) =>
              item.nama === coverageName && (plan === undefined || item.id_plan === plan.id),
          )

          return {
            id_plan: plan?.id ?? '',
            nama_plan: planName,
            id_jaminan: coverage?.id ?? '',
            nama_jaminan: coverageName,
          }
        }),
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
  const columns: Column<TravelDocumentDetail>[] = [
    { key: 'id', title: 'ID', width: 'w-24', value: (row) => row.id },
    { key: 'id_dokumen', title: 'ID Dokumen', width: 'w-32', value: (row) => row.id_dokumen },
    { key: 'nama_dokumen', title: 'Nama Dokumen', value: (row) => row.nama_dokumen },
    {
      key: 'status_wajib',
      title: 'Status Wajib',
      width: 'w-32',
      // Teks "Ya"/"Tidak" mengikuti `Activity/BrowseDocTravel-Act.xml`, yang mengubah
      // STSWAJIB menjadi kedua kata itu sebelum menampilkannya. Yang tersimpan tetap 1
      // atau 0.
      value: (row) => (row.status_wajib ? 'Ya' : 'Tidak'),
      render: (row) => (
        <span
          className={
            row.status_wajib
              ? 'rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700'
              : 'rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600'
          }
        >
          {row.status_wajib ? 'Ya' : 'Tidak'}
        </span>
      ),
    },
    {
      key: 'minimal_unggah',
      title: 'Minimal Unggah',
      width: 'w-36',
      alignRight: true,
      value: (row) => String(row.minimal_unggah),
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
          aria-label={`Ubah ${row.nama_dokumen || row.id}`}
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
          <h1 className="text-xl font-semibold text-slate-900">Detail Dokumen Travel</h1>
          <p className="text-sm text-slate-600">
            Dokumen apa saja yang diminta pada klaim Travel, wajib atau tidak, dan paling sedikit
            berapa berkas yang harus diunggah.
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
          {/* Kegagalan memuat daftar plan TIDAK menutup form dan tidak menghalangi
              penyimpanan: nama plan dan jaminan memang boleh diketik sendiri. Yang hilang
              hanya sarannya, dan form itu sendiri yang mengatakannya. */}
          {travelPlans.isError && (
            <div className="mb-3">
              <ErrorMessage
                title="Daftar plan dan jaminan tidak dapat dimuat"
                description="Sarannya tidak tersedia untuk sementara. Namanya tetap dapat diketik sendiri, dan seluruh isian tetap dapat disimpan — hanya kodenya yang tidak ikut terisi."
                tone="gangguan"
              />
            </div>
          )}
          {documents.isError && (
            <div className="mb-3">
              <ErrorMessage
                title="Daftar dokumen travel tidak dapat dimuat"
                description="Saran ID Dokumen tidak tersedia untuk sementara. Kodenya tetap dapat diketik sendiri."
                tone="gangguan"
              />
            </div>
          )}
          <TravelDocumentDetailForm
            edited={edited}
            documents={documents.data?.dokumen ?? []}
            plans={travelPlans.data?.plan ?? []}
            coverages={travelPlans.data?.jaminan ?? []}
            isLoadingCoverageMapping={openedRow !== null && detail.isPending}
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
          <p className="text-sm text-slate-500">Memuat daftar detail dokumen travel…</p>
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
            rows={list.data.detail_dokumen_travel}
            rowKey={(row) => row.id}
            description="Sumber: POOLDATA.V_LST_DOC_TRAVEL"
            emptyMessage="Belum ada detail dokumen travel pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
