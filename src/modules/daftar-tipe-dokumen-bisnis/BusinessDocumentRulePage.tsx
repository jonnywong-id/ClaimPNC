import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type BusinessChoice,
  type BusinessDocumentRuleInput,
  type BusinessDocumentRuleRowInput,
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
  useSaveBusinessDocumentRules,
} from './api'
import { BusinessDocumentRuleCreateForm } from './BusinessDocumentRuleCreateForm'
import { BusinessDocumentRuleEditForm } from './BusinessDocumentRuleEditForm'

const CLOSED = 'closed'

/**
 * Bentuk layar yang sedang terbuka.
 *
 * Ketiga modenya sama persis dengan ketiga tombol layar lama — Tambah, Ubah, Copy — dan
 * `businessID` menandai bisnis yang barisnya sedang dimuat. Tambah tidak memerlukannya.
 */
type FormState =
  | typeof CLOSED
  | { mode: 'tambah' }
  | { mode: 'ubah'; businessID: string; businessName: string }
  | { mode: 'copy'; businessID: string }

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
 * # Bentuknya: SATU daftar, tiga tombol
 *
 * Daftar memuat lini bisnis yang sudah punya aturan, masing-masing dengan **Ubah** dan
 * **Copy**; **Tambah** ada di kepala layar. Ubah membuka seluruh baris bisnis itu dalam
 * satu form yang dapat disunting sekaligus.
 *
 * Versi pertama modul ini memakai grid tingkat kedua dan tombol "Detail". Itu penyimpangan
 * yang saya buat sendiri dan sudah dicabut — lihat BusinessDocumentRuleEditForm.
 *
 * # Yang sengaja dibuat BERBEDA dari Pega
 *
 * | Berbeda | Sifatnya |
 * |---|---|
 * | Tabel baris digulung mendatar pada layar sempit | tampilan (`D-12`) |
 * | Entitas yang dilihat disebut terang-terangan | keamanan (`R-20`) |
 * | Peringatan "wajib di sini belum berarti wajib di klaim" | penjelasan atas aturan yang sudah ada |
 * | Lima kode pengecualian pemilihan massal datang dari konfigurasi | `D-15` |
 * | Satu penyimpanan Ubah = satu transaksi | `D-68` |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada tombol hapus — layar lama pun tidak punya, dan `D-66` melarangnya. Tidak ada
 * validasi selain "Nama Bisnis belum di isi", karena hanya itu yang ada di layar lama.
 * Judul form "Tambah Data" dan "Update Data" ditiru apa adanya. Jaminan hanya DITAMBAHKAN,
 * karena tidak ada satu pun jalur hapus terhadap tabelnya di seluruh sistem lama.
 */
export function BusinessDocumentRulePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [form, setForm] = useState<FormState>(CLOSED)
  const [coverageRuleID, setCoverageRuleID] = useState<string | null>(null)
  const [coverageDraft, setCoverageDraft] = useState('')

  const businesses = useBusinessList()

  // Baris dimuat untuk Ubah maupun Copy — keduanya memerlukan isi bisnis yang sama, dan
  // bedanya hanya pada apa yang dilakukan terhadapnya.
  const loadedBusiness = form !== CLOSED && 'businessID' in form ? form.businessID : null
  const rules = useBusinessDocumentRuleList(loadedBusiness)

  const businessChoices = useBusinessChoiceList()
  const documentTypes = useDocumentTypeChoiceList()
  const detailDocuments = useDetailDocumentChoiceList()
  const objectDocuments = useObjectDocumentChoiceList()

  const coverageRule = useBusinessDocumentRule(coverageRuleID)

  const create = useCreateBusinessDocumentRule()
  const save = useSaveBusinessDocumentRules()
  const addCoverage = useAddBusinessDocumentRuleCoverage()

  function closeForm() {
    create.reset()
    save.reset()
    addCoverage.reset()
    setCoverageDraft('')
    setCoverageRuleID(null)
    setForm(CLOSED)
  }

  function open(next: FormState) {
    create.reset()
    save.reset()
    addCoverage.reset()
    setCoverageDraft('')
    setCoverageRuleID(null)
    setForm(next)
  }

  function saveCreate(businessIDs: string[], draft: BusinessDocumentRuleInput[]) {
    create.mutate({ bisnis: businessIDs, dokumen: draft }, { onSuccess: closeForm })
  }

  function saveEdit(rows: BusinessDocumentRuleRowInput[]) {
    if (form === CLOSED || form.mode !== 'ubah') return
    save.mutate({ businessID: form.businessID, rows }, { onSuccess: closeForm })
  }

  const businessColumns: Column<BusinessChoice>[] = [
    { key: 'id', title: 'ID', width: 'w-28', value: (row) => row.id },
    {
      key: 'nama_bisnis',
      title: 'Nama Bisnis',
      value: (row) => row.nama_bisnis,
      render: (row) => row.nama_bisnis || <span className="text-slate-400">(tanpa nama)</span>,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-44',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <div className="flex flex-wrap justify-end gap-1">
          <Button
            tone="kedua"
            disabled={form !== CLOSED}
            onClick={() =>
              open({ mode: 'ubah', businessID: row.id, businessName: row.nama_bisnis })
            }
            aria-label={`Ubah dokumen bisnis ${row.nama_bisnis || row.id}`}
          >
            Ubah
          </Button>
          {/*
            Copy menyalin SELURUH aturan bisnis ini ke form Tambah tanpa ID-nya, lalu
            petugas mengisi bisnis tujuannya. Ia satu-satunya cara memakai ulang susunan
            yang sudah ada, karena sebuah aturan tidak dapat dipindahkan antar lini bisnis.
          */}
          <Button
            tone="kedua"
            disabled={form !== CLOSED}
            onClick={() => open({ mode: 'copy', businessID: row.id })}
            aria-label={`Copy dokumen bisnis ${row.nama_bisnis || row.id}`}
          >
            Copy
          </Button>
        </div>
      ),
    },
  ]

  // Baris sedang dimuat untuk Ubah atau Copy — keduanya menunggu data yang sama.
  const waitingForRules =
    form !== CLOSED && 'businessID' in form && (rules.isPending || rules.isError)

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
              businesses.refetch()
              if (loadedBusiness !== null) rules.refetch()
            }}
            disabled={businesses.isFetching}
          >
            {businesses.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={() => open({ mode: 'tambah' })} disabled={form !== CLOSED}>
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

      {waitingForRules && (
        <section className="mt-5">
          {rules.isError ? (
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
            <p className="text-sm text-slate-500">Memuat dokumen…</p>
          )}
        </section>
      )}

      {form !== CLOSED && form.mode === 'tambah' && (
        <section className="mt-5">
          <BusinessDocumentRuleCreateForm
            businesses={businessChoices.data?.bisnis ?? []}
            mayBulkSelect={businessChoices.data?.boleh_pilih_semua ?? false}
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

      {form !== CLOSED && form.mode === 'copy' && rules.data !== undefined && (
        <section className="mt-5">
          {/*
            `key` memaksa form dibentuk ulang saat bisnis asal berganti. Tanpa itu, baris
            awal yang dibentuk sekali di dalam useState akan tetap milik salinan
            sebelumnya.
          */}
          <BusinessDocumentRuleCreateForm
            key={`copy-${form.businessID}`}
            businesses={businessChoices.data?.bisnis ?? []}
            mayBulkSelect={businessChoices.data?.boleh_pilih_semua ?? false}
            initialRules={rules.data.tipe_dokumen_bisnis.map((row) => ({
              id_tipe_dokumen: row.id_tipe_dokumen,
              id_object_dokumen: row.id_object_dokumen,
              id_detail_dokumen: row.id_detail_dokumen,
              detail_dokumen: row.detail_dokumen,
              status_wajib: row.status_wajib,
              minimum_dokumen: row.minimum_dokumen,
            }))}
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

      {form !== CLOSED && form.mode === 'ubah' && rules.data !== undefined && (
        <section className="mt-5 space-y-4">
          <BusinessDocumentRuleEditForm
            key={`ubah-${form.businessID}`}
            businessID={form.businessID}
            businessName={form.businessName}
            rules={rules.data.tipe_dokumen_bisnis}
            documentTypes={documentTypes.data?.pilihan ?? []}
            detailDocuments={detailDocuments.data?.pilihan ?? []}
            objectDocuments={objectDocuments.data?.pilihan ?? []}
            isSaving={save.isPending}
            error={save.error}
            onSave={saveEdit}
            onCancel={closeForm}
            onOpenCoverage={(ruleID) => {
              addCoverage.reset()
              setCoverageDraft('')
              setCoverageRuleID((current) => (current === ruleID ? null : ruleID))
            }}
          />

          {/*
            Jenis Klaim berada di luar form, dan itu bukan pilihan tata letak melainkan
            cerminan cara ia tersimpan: di sistem lama ia ditulis lewat pemanggilan
            procedure TERSENDIRI, satu jaminan per panggilan, dan tidak ada jalur yang
            mengganti seluruh daftarnya sekaligus. Menaruhnya di dalam form akan
            menyiratkan bahwa Batal dapat membatalkannya — padahal setiap jaminan tersimpan
            seketika dan tidak dapat dibuang.
          */}
          {coverageRuleID !== null && (
            <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
              <h2 className="text-base font-semibold text-slate-900">
                Jenis Klaim{' '}
                <span className="font-mono text-sm font-normal text-slate-500">
                  baris {coverageRuleID}
                </span>
              </h2>
              <p className="mt-1 text-sm text-slate-600">
                Jaminan yang membuat dokumen ini benar-benar wajib saat klaim diregistrasi. Selama
                daftar ini kosong, dokumen tetap terbaca tidak wajib meski ditandai wajib di atas.
              </p>

              {coverageRule.isPending ? (
                <p className="mt-3 text-sm text-slate-500">Memuat jenis klaim…</p>
              ) : coverageRule.isError ? (
                (() => {
                  const message = loadMessage(coverageRule.error)
                  return (
                    <div className="mt-3">
                      <ErrorMessage
                        title={message.title}
                        description={message.description}
                        tone={message.tone}
                      />
                    </div>
                  )
                })()
              ) : (
                <ul className="mt-3 flex flex-wrap gap-2">
                  {coverageRule.data.tipe_dokumen_bisnis.jenis_klaim.length === 0 ? (
                    <li className="text-sm text-slate-400">Belum ada jenis klaim.</li>
                  ) : (
                    coverageRule.data.tipe_dokumen_bisnis.jenis_klaim.map((coverage) => (
                      <li
                        key={coverage}
                        className="rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-sm text-slate-700"
                      >
                        {coverage}
                      </li>
                    ))
                  )}
                </ul>
              )}

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
                  // Namanya dibedakan dari tombol Tambah di kepala layar. Keduanya berbunyi
                  // "Tambah" dan melakukan hal yang sangat berbeda — yang satu membuka form
                  // baris baru, yang lain menyimpan jaminan seketika dan tidak dapat
                  // dibatalkan.
                  aria-label="Tambah jenis klaim"
                  disabled={addCoverage.isPending}
                  onClick={() =>
                    addCoverage.mutate(
                      { id: coverageRuleID, coverageID: coverageDraft },
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
            /*
              Pencarian dikerjakan di peramban, dan itu sah di sini: daftar bisnis dimuat
              SEKALIGUS, bukan dipaginasi server. Pada layar yang dipaginasi server,
              menyaring di peramban hanya menyentuh halaman yang terbuka dan hasilnya
              berbohong — itu sebabnya `manualFiltering` ada. Di sini seluruh barisnya sudah
              di tangan, sehingga hasilnya utuh.

              Layar lama memang tidak punya kotak cari. Penambahan ini SELISIH TERENCANA
              atas permintaan Work Owner (2026-10-05): daftarnya memuat ratusan lini bisnis,
              dan menggulungnya satu per satu untuk menemukan satu nama adalah pekerjaan
              yang tidak dibayar manfaat apa pun.
            */
            searchLabel="Cari nama bisnis atau kodenya"
            pageSize={50}
            description={`${businesses.data.total} lini bisnis sudah punya aturan dokumen. Sumber: POOLDATA.LST_TYPE_DOC_BUSINESS`}
            emptyMessage="Belum ada lini bisnis yang punya aturan dokumen pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
