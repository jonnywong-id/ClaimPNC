import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type BusinessChoice,
  type BusinessDocumentRule,
  type BusinessDocumentRuleInput,
} from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import {
  useAddBusinessDocumentRuleCoverage,
  useBusinessChoiceList,
  useBusinessDocumentRule,
  useBusinessDocumentRuleList,
  useBusinessList,
  useCreateBusinessDocumentRule,
  useDetailDocumentChoiceList,
  useDocumentTypeChoiceList,
  useObjectDocumentChoiceList,
  useUpdateBusinessDocumentRule,
} from './api'
import { BusinessDocumentRuleCreateForm } from './BusinessDocumentRuleCreateForm'
import {
  BusinessDocumentRuleForm,
  type BusinessDocumentRuleFields,
} from './BusinessDocumentRuleForm'

const CLOSED = 'closed'
const CREATE = 'create'

/** Bentuk form yang sedang terbuka; saat menyunting, ia ID barisnya. */
type FormState = typeof CLOSED | typeof CREATE | { editedID: string }

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa sambungan jaringan, lalu muat ulang.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih entitas di bagian atas layar untuk melihat daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description: 'Hubungi tim infrastruktur bila keadaan ini berlanjut.',
          tone: 'gangguan',
        }
      default:
        return { title: 'Daftar tidak dapat dimuat', description: error.message, tone: 'gangguan' }
    }
  }
  return {
    title: 'Daftar tidak dapat dimuat',
    description: 'Muat ulang halaman ini.',
    tone: 'gangguan',
  }
}

/**
 * Layar Daftar Tipe Dokumen Bisnis — MENU_ID 42, MENU_PROGRAM `DetTypeDocumenBisnis`.
 *
 * Menggantikan `Harness/DetTypeDocumenBisnis-Harness.xml` beserta ketiga section-nya.
 *
 * # Bentuknya BERTINGKAT DUA, seperti layar lamanya
 *
 * Grid pertama memuat lini bisnis yang sudah punya aturan; memilih salah satunya membuka
 * grid kedua berisi aturan dokumen milik bisnis itu. Bukan satu grid panjang: tabelnya
 * dapat memuat ribuan baris — satu per bisnis per dokumen per tahap — dan menampilkannya
 * sekaligus tidak pernah berguna bagi siapa pun.
 *
 * # Yang sengaja dibuat BERBEDA dari Pega
 *
 * | Berbeda | Sifatnya |
 * |---|---|
 * | Layar sempit menjadi kartu | tampilan (`D-12`) |
 * | Entitas yang dilihat disebut terang-terangan | keamanan (`R-20`) |
 * | Peringatan "wajib di sini belum berarti wajib di klaim" | penjelasan atas aturan yang sudah ada |
 * | "Pilih semua" tanpa pengecualian lima kode bisnis | `D-15` — kode bisnis tidak di dalam kode |
 * | Ketiga isian rujukan berupa dropdown | bentuk datanya hanya menyimpan kode |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada tombol hapus — layar lama pun tidak punya, dan `D-66` melarangnya. Tidak ada
 * validasi selain "Nama Bisnis belum di isi", karena hanya itu yang ada di layar lama.
 * Judul form "Update Data" dipertahankan pada mode ubah. Jaminan hanya DITAMBAHKAN,
 * karena tidak ada satu pun jalur hapus terhadap tabelnya di seluruh sistem lama.
 */
export function BusinessDocumentRulePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [selectedBusiness, setSelectedBusiness] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(CLOSED)
  const [coverageDraft, setCoverageDraft] = useState('')

  // Baris yang disalin dari sebuah bisnis, menunggu bisnis tujuan dipilih.
  //
  // Ia dipisahkan dari FormState supaya Copy dan Tambah memakai form yang SAMA — di Pega
  // pun keduanya satu layar, dan bedanya hanya pada baris yang sudah terisi.
  const [copiedRules, setCopiedRules] = useState<BusinessDocumentRuleInput[] | undefined>(
    undefined,
  )

  const businesses = useBusinessList()
  const rules = useBusinessDocumentRuleList(selectedBusiness)

  const businessChoices = useBusinessChoiceList()
  const documentTypes = useDocumentTypeChoiceList()
  const detailDocuments = useDetailDocumentChoiceList()
  const objectDocuments = useObjectDocumentChoiceList()

  const editedID = typeof form === 'object' ? form.editedID : null
  const edited = useBusinessDocumentRule(editedID)

  const create = useCreateBusinessDocumentRule()
  const update = useUpdateBusinessDocumentRule()
  const addCoverage = useAddBusinessDocumentRuleCoverage()

  function closeForm() {
    create.reset()
    update.reset()
    addCoverage.reset()
    setCoverageDraft('')
    setCopiedRules(undefined)
    setForm(CLOSED)
  }

  function openCreate() {
    create.reset()
    update.reset()
    setCopiedRules(undefined)
    setForm(CREATE)
  }

  /**
   * Menyalin seluruh aturan sebuah bisnis ke form Tambah, tanpa ID-nya.
   *
   * Inilah guna tombol Copy di layar lama: `UpdateDetailTypeDocumentBusiness_act` memuat
   * baris bisnis asal dengan pemetaan yang sama seperti Ubah, tetapi cabang salinnya
   * TIDAK PERNAH mengisi `.ID` — sehingga penyimpanan berikutnya menerbitkan baris baru.
   *
   * Aturan dokumen memang tidak dapat dipindahkan antar lini bisnis (`BUSINESSID` tidak
   * ikut diubah saat menyunting), sehingga menyalin adalah satu-satunya cara memakai
   * ulang susunan yang sudah ada.
   */
  function openCopy() {
    if (rules.data === undefined) return
    create.reset()
    update.reset()
    setCopiedRules(
      rules.data.tipe_dokumen_bisnis.map((row) => ({
        id_tipe_dokumen: row.id_tipe_dokumen,
        id_object_dokumen: row.id_object_dokumen,
        id_detail_dokumen: row.id_detail_dokumen,
        detail_dokumen: row.detail_dokumen,
        status_wajib: row.status_wajib,
        minimum_dokumen: row.minimum_dokumen,
      })),
    )
    setForm(CREATE)
  }

  function openEdit(row: BusinessDocumentRule) {
    create.reset()
    update.reset()
    addCoverage.reset()
    setCoverageDraft('')
    setForm({ editedID: row.id })
  }

  function saveCreate(businessIDs: string[], draft: BusinessDocumentRuleInput[]) {
    create.mutate({ bisnis: businessIDs, dokumen: draft }, { onSuccess: closeForm })
  }

  function saveUpdate(values: BusinessDocumentRuleFields) {
    if (editedID === null) return
    update.mutate(
      {
        id: editedID,
        input: {
          id_tipe_dokumen: values.id_tipe_dokumen,
          id_object_dokumen: values.id_object_dokumen,
          id_detail_dokumen: values.id_detail_dokumen,
          detail_dokumen: values.detail_dokumen,
          status_wajib: values.status_wajib,
          // Diubah menjadi angka di sini, bukan di dalam form — lihat komentar skema di
          // BusinessDocumentRuleForm.
          minimum_dokumen: Number(values.minimum_dokumen.replace(/[^0-9]/g, '')) || 0,
        },
      },
      { onSuccess: closeForm },
    )
  }

  const businessColumns: Column<BusinessChoice>[] = [
    { key: 'id', title: 'ID', width: 'w-28', value: (row) => row.id },
    {
      key: 'nama_bisnis',
      title: 'Nama Bisnis',
      value: (row) => row.nama_bisnis,
      render: (row) =>
        row.nama_bisnis || <span className="text-slate-400">(tanpa nama)</span>,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-32',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => setSelectedBusiness(row.id)}
          aria-label={`Lihat dokumen bisnis ${row.nama_bisnis || row.id}`}
        >
          Detail
        </Button>
      ),
    },
  ]

  const ruleColumns: Column<BusinessDocumentRule>[] = [
    { key: 'tipe_dokumen', title: 'Tipe Dokumen', width: 'w-44', value: (row) => row.tipe_dokumen },
    {
      key: 'object_dokumen',
      title: 'Object Dokumen',
      value: (row) => row.object_dokumen,
      render: (row) => row.object_dokumen || <span className="text-slate-400">—</span>,
    },
    {
      key: 'detail_dokumen',
      title: 'Detail Dokumen',
      value: (row) => row.detail_dokumen,
      render: (row) =>
        row.detail_dokumen === '-' ? (
          <span title="Disembunyikan dari seluruh layar unggah dokumen">
            <span className="text-slate-400">— disembunyikan</span>
          </span>
        ) : (
          row.detail_dokumen || <span className="text-slate-400">—</span>
        ),
    },
    {
      key: 'status_wajib',
      title: 'Status Wajib',
      width: 'w-32',
      value: (row) => (row.status_wajib ? 'Ya' : 'Tidak'),
    },
    {
      key: 'minimum_dokumen',
      title: 'Minimum Dokumen',
      width: 'w-36',
      alignRight: true,
      value: (row) => String(row.minimum_dokumen),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-32',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => openEdit(row)}
          aria-label={`Ubah aturan ${row.detail_dokumen || row.id}`}
        >
          Update Data
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya ditiru dari `DetTypeDocumenBisnis_Portal-Section.xml` apa adanya. */}
          <h1 className="text-xl font-semibold text-slate-900">Detail Tipe Dokumen Bisnis</h1>
          <p className="text-sm text-slate-600">
            Dokumen yang harus diunggah, per lini bisnis dan per tahap klaim.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            tone="kedua"
            onClick={() => {
              void businesses.refetch()
              if (selectedBusiness !== null) void rules.refetch()
            }}
            disabled={businesses.isFetching}
          >
            {businesses.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openCreate} disabled={form !== CLOSED}>
            Tambah
          </Button>
        </div>
      </header>

      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">
          {businesses.data?.portal ?? portal ?? '—'}
        </span>
      </p>

      {form === CREATE && (
        <section className="mt-5">
          <BusinessDocumentRuleCreateForm
            businesses={businessChoices.data?.bisnis ?? []}
            mayBulkSelect={businessChoices.data?.boleh_pilih_semua ?? false}
            initialRules={copiedRules}
            documentTypes={documentTypes.data?.pilihan ?? []}
            detailDocuments={detailDocuments.data?.pilihan ?? []}
            objectDocuments={objectDocuments.data?.pilihan ?? []}
            isSaving={create.isPending}
            error={create.error}
            onSave={saveCreate}
            onCancel={closeForm}
          />
        </section>
      )}

      {editedID !== null && (
        <section className="mt-5 space-y-4">
          {edited.isPending ? (
            <p className="text-sm text-slate-500">Memuat aturan dokumen…</p>
          ) : edited.isError ? (
            (() => {
              const message = loadMessage(edited.error)
              return (
                <ErrorMessage
                  title={message.title}
                  description={message.description}
                  tone={message.tone}
                />
              )
            })()
          ) : (
            <>
              <BusinessDocumentRuleForm
                edited={edited.data.tipe_dokumen_bisnis}
                documentTypes={documentTypes.data?.pilihan ?? []}
                detailDocuments={detailDocuments.data?.pilihan ?? []}
                objectDocuments={objectDocuments.data?.pilihan ?? []}
                isSaving={update.isPending}
                error={update.error}
                onSave={saveUpdate}
                onCancel={closeForm}
              />

              {/*
                Jenis Klaim berada di luar form penyuntingan, dan itu bukan pilihan tata
                letak melainkan cerminan cara ia tersimpan: di sistem lama ia ditulis lewat
                pemanggilan procedure TERSENDIRI, satu jaminan per panggilan, dan tidak ada
                jalur yang mengganti seluruh daftarnya sekaligus. Menaruhnya di dalam form
                akan menyiratkan bahwa Batal dapat membatalkannya — padahal setiap jaminan
                tersimpan seketika dan tidak dapat dibuang.
              */}
              <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
                <h2 className="text-base font-semibold text-slate-900">Jenis Klaim</h2>
                <p className="mt-1 text-sm text-slate-600">
                  Jaminan yang membuat dokumen ini benar-benar wajib saat klaim diregistrasi.
                  Selama daftar ini kosong, dokumen tetap terbaca tidak wajib meski ditandai
                  wajib di atas.
                </p>

                <ul className="mt-3 flex flex-wrap gap-2">
                  {edited.data.tipe_dokumen_bisnis.jenis_klaim.length === 0 ? (
                    <li className="text-sm text-slate-400">Belum ada jenis klaim.</li>
                  ) : (
                    edited.data.tipe_dokumen_bisnis.jenis_klaim.map((coverage) => (
                      <li
                        key={coverage}
                        className="rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-sm text-slate-700"
                      >
                        {coverage}
                      </li>
                    ))
                  )}
                </ul>

                <div className="mt-4 flex flex-wrap items-end gap-2">
                  <label className="text-sm">
                    <span className="block font-medium text-slate-700">Kode Jenis Klaim</span>
                    <input
                      type="text"
                      autoComplete="off"
                      aria-label="Kode Jenis Klaim"
                      className="mt-1 w-48 rounded border border-slate-300 bg-white px-3 py-2"
                      value={coverageDraft}
                      onChange={(event) => setCoverageDraft(event.target.value)}
                    />
                  </label>
                  <Button
                    tone="kedua"
                    // Namanya dibedakan dari tombol Tambah di kepala layar. Keduanya
                    // berbunyi "Tambah" dan melakukan hal yang sangat berbeda — yang satu
                    // membuka form baris baru, yang lain menyimpan jaminan seketika dan
                    // tidak dapat dibatalkan.
                    aria-label="Tambah jenis klaim"
                    disabled={addCoverage.isPending}
                    onClick={() =>
                      addCoverage.mutate(
                        { id: editedID, coverageID: coverageDraft },
                        { onSuccess: () => setCoverageDraft('') },
                      )
                    }
                  >
                    {addCoverage.isPending ? 'Menambah…' : 'Tambah'}
                  </Button>
                </div>

                <p className="mt-2 text-xs text-slate-500">
                  Jenis klaim hanya dapat ditambahkan, tidak dapat dibuang — sistem lama pun tidak
                  punya jalur menghapusnya.
                </p>

                {addCoverage.error !== null &&
                  (() => {
                    const message = loadMessage(addCoverage.error)
                    return (
                      <div className="mt-3">
                        <ErrorMessage
                          title={message.title}
                          description={message.description}
                          tone={message.tone}
                        />
                      </div>
                    )
                  })()}
              </section>
            </>
          )}
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Pilih entitas di bagian atas layar untuk melihat daftarnya."
            tone="penolakan"
          />
        ) : businesses.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar bisnis…</p>
        ) : businesses.isError ? (
          (() => {
            const message = loadMessage(businesses.error)
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
            columns={businessColumns}
            rows={businesses.data.bisnis}
            rowKey={(row) => row.id}
            searchable={false}
            pageSize={50}
            description={`${businesses.data.total} lini bisnis sudah punya aturan dokumen. Sumber: POOLDATA.LST_TYPE_DOC_BUSINESS`}
            emptyMessage="Belum ada lini bisnis yang punya aturan dokumen pada entitas ini."
          />
        )}
      </section>

      {selectedBusiness !== null && (
        <section className="mt-8">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 pb-2">
            <h2 className="text-base font-semibold text-slate-900">
              Dokumen bisnis{' '}
              <span className="font-mono text-slate-600">{selectedBusiness}</span>
            </h2>
            <div className="flex flex-wrap gap-2">
              {/*
                Copy menyalin SELURUH aturan bisnis ini ke form Tambah tanpa ID-nya, lalu
                petugas memilih bisnis tujuannya. Ia satu-satunya cara memakai ulang
                susunan yang sudah ada, karena sebuah aturan tidak dapat dipindahkan antar
                lini bisnis.
              */}
              <Button
                tone="kedua"
                onClick={openCopy}
                disabled={
                  form !== CLOSED ||
                  rules.data === undefined ||
                  rules.data.tipe_dokumen_bisnis.length === 0
                }
              >
                Copy
              </Button>
              <Button tone="halus" onClick={() => setSelectedBusiness(null)}>
                Tutup
              </Button>
            </div>
          </div>

          <div className="mt-4">
            {rules.isPending ? (
              <p className="text-sm text-slate-500">Memuat dokumen…</p>
            ) : rules.isError ? (
              (() => {
                const message = loadMessage(rules.error)
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
                columns={ruleColumns}
                rows={rules.data.tipe_dokumen_bisnis}
                rowKey={(row) => row.id}
                searchable={false}
                pageSize={50}
                description={`${rules.data.total} aturan dokumen pada bisnis ini.`}
                emptyMessage="Bisnis ini belum punya aturan dokumen. Tekan Tambah untuk membuatnya."
              />
            )}
          </div>
        </section>
      )}
    </main>
  )
}
