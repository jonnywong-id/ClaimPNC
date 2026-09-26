import { useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { ArchiveFormPanel } from './ArchiveFormPanel'
import { ArchiveSearchPanel } from './ArchiveSearchPanel'
import { BranchQueue } from './BranchQueue'
import { ClaimPicker } from './ClaimPicker'
import {
  ArchiveError,
  EMPTY_CLAIM_SEARCH,
  EMPTY_SEARCH,
  formFromClaim,
  type ArchiveFile,
  type ArchiveForm,
  type ClaimSearchForm,
  type SearchForm,
} from './types'
import {
  useArchiveSearch,
  useClaimSearch,
  useOpenArchive,
  usePendingBranch,
  useExportArchive,
  useSaveArchive,
  useSendToBranch,
} from './api'

/** Ketiga bagian layar, persis pembagian layar lama. */
const TABS = [
  { id: 'cari', label: 'Archive File Klaim' },
  { id: 'input', label: 'Input Data Archive' },
  { id: 'cabang', label: 'Kirim ke Cabang' },
] as const

type TabID = (typeof TABS)[number]['id']

/**
 * Archive Dokumen Klaim — menu `MENU_ID 77`, pengganti harness `PNCArchiveDokumen`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Layar lama menggambar ketiga bagiannya di SATU halaman yang saling menampakkan dan
 * menyembunyikan lewat penanda `FalgArchiveData.FlagASO` dan `.ContractNo`
 * (`Activity/FlagForArchiveData-Act.xml`). Di sini ketiganya menjadi tab.
 *
 * Yang berubah hanyalah CARA berpindah, bukan isinya: ketiga bagiannya tetap ada, tetap
 * berurutan sama, dan tetap membawa isian yang sama. Penanda tampil-sembunyi Pega itu
 * memang tab yang digambar dengan cara lain — dan menirunya apa adanya akan menghasilkan
 * halaman yang isinya berganti tanpa ada yang menunjukkan bahwa ia berganti.
 *
 * # Nama kolom tidak diterjemahkan
 *
 * `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang, dan
 * "Nama BOX", "Kode Filling", serta "TGL INPUT" adalah teks yang selama ini mereka baca.
 */
export function ArchiveDocumentPage() {
  const [tab, setTab] = useState<TabID>('cari')

  const [searchForm, setSearchForm] = useState<SearchForm>(EMPTY_SEARCH)
  const [submittedSearch, setSubmittedSearch] = useState<SearchForm | null>(null)
  const [searchPage, setSearchPage] = useState(1)

  const [claimForm, setClaimForm] = useState<ClaimSearchForm>(EMPTY_CLAIM_SEARCH)
  const [submittedClaim, setSubmittedClaim] = useState<ClaimSearchForm | null>(null)

  const [archiveForm, setArchiveForm] = useState<ArchiveForm | null>(null)
  const [savedMessage, setSavedMessage] = useState('')

  const [exportError, setExportError] = useState('')

  const [branchPage, setBranchPage] = useState(1)
  const [sendingID, setSendingID] = useState<number | null>(null)
  const [sendMessage, setSendMessage] = useState('')

  const portal = useSelectedPortal((state) => state.alias)

  const opened = useOpenArchive()
  const search = useArchiveSearch(
    submittedSearch ?? EMPTY_SEARCH,
    searchPage,
    submittedSearch !== null,
  )
  const claims = useClaimSearch(submittedClaim ?? EMPTY_CLAIM_SEARCH, submittedClaim !== null)
  const pending = usePendingBranch(branchPage, tab === 'cabang')
  const save = useSaveArchive()
  const send = useSendToBranch()
  const exportFile = useExportArchive()

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description="Berkas arsip milik satu badan hukum, sehingga entitasnya harus jelas sebelum daftarnya dibuka."
          tone="penolakan"
        />
      </PageFrame>
    )
  }

  if (opened.isError) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(opened.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  /** Mengubah mode pencarian MEMBERSIHKAN isian lain. */
  function changeSearch(patch: Partial<SearchForm>) {
    setSearchForm((previous) =>
      patch.mode !== undefined && patch.mode !== previous.mode
        ? { ...EMPTY_SEARCH, mode: patch.mode }
        : { ...previous, ...patch },
    )
  }

  function runSearch() {
    setSearchPage(1)
    setSubmittedSearch(searchForm)
  }

  function runClaimSearch() {
    setSubmittedClaim(claimForm)
  }

  function saveArchive() {
    if (!archiveForm) return

    setSavedMessage('')
    save.mutate(archiveForm, {
      onSuccess: (result) => {
        setSavedMessage(result.pesan)

        // Formulirnya DIKOSONGKAN setelah berhasil, dan klaim terpilih ikut dilepas.
        // Membiarkannya terisi mengundang penyimpanan kedua yang menghasilkan berkas
        // ganda untuk klaim yang sama — dan layar ini tidak punya tombol hapus.
        setArchiveForm(null)
      },
    })
  }

  /**
   * runExport mengunduh hasil pencarian sebagai berkas.
   *
   * Tautan sementara dibuat lalu SEGERA dilepas. Tanpa `revokeObjectURL`, setiap ekspor
   * menyisakan satu blob di memori peramban sampai tab ditutup — dan layar ini dipakai
   * sepanjang hari.
   */
  function runExport() {
    setExportError('')

    exportFile.mutate(submittedSearch ?? searchForm, {
      onSuccess: ({ blob, filename }) => {
        const url = URL.createObjectURL(blob)
        const anchor = document.createElement('a')
        anchor.href = url
        anchor.download = filename
        document.body.appendChild(anchor)
        anchor.click()
        anchor.remove()
        URL.revokeObjectURL(url)
      },
      onError: (error) => setExportError(messageOf(error)),
    })
  }

  function sendToBranch(id: number) {
    setSendMessage('')
    setSendingID(id)

    send.mutate(id, {
      onSuccess: (result) => {
        setSendMessage(`${result.pesan} Jawaban sistem Arsip: ${result.kode_layanan || '—'}.`)
      },
      onSettled: () => setSendingID(null),
    })
  }

  return (
    <PageFrame>
      <nav className="flex flex-wrap gap-2" aria-label="Bagian layar Archive Dokumen Klaim">
        {TABS.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => setTab(item.id)}
            aria-current={tab === item.id ? 'page' : undefined}
            className={[
              'rounded-kontrol px-4 py-2 text-sm font-medium transition-colors',
              'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
              tab === item.id
                ? 'bg-blue-600 text-white shadow-aksen'
                : 'bg-white text-slate-700 ring-1 ring-slate-300 hover:bg-slate-50',
            ].join(' ')}
          >
            {item.label}
          </button>
        ))}
      </nav>

      {tab === 'cari' && (
        <>
          <ArchiveSearchPanel
            form={searchForm}
            onChange={changeSearch}
            onSubmit={runSearch}
            busy={search.isFetching}
            fieldError={fieldErrorOf(search.error)}
          />

          {exportError && (
            <div className="mt-4">
              <ErrorMessage
                title="Berkas ekspor tidak dapat diambil"
                description={exportError}
                tone="gangguan"
              />
            </div>
          )}

          {submittedSearch !== null && (
            <div className="mt-4">
              <DataTable
                columns={archiveColumns()}
                rows={search.data?.berkas ?? []}
                rowKey={(row) => String(row.id)}
                title="ARCHIVE FILE KLAIM"
                label="Daftar berkas klaim yang sudah diarsipkan"
                isLoading={search.isFetching}
                error={search.isError && !isValidationError(search.error)
                  ? messageOf(search.error)
                  : ''}
                emptyMessage="Tidak ada berkas yang cocok. Pencocokan kata kunci PERSIS, bukan sebagian."
                hideSearch
                actions={
                  <Button
                    type="button"
                    tone="kedua"
                    onClick={runExport}
                    disabled={exportFile.isPending || (search.data?.halaman.total ?? 0) === 0}
                  >
                    {exportFile.isPending ? 'Menyiapkan…' : 'Export To Excel'}
                  </Button>
                }
                // Nilai cadangan dipakai saat halaman pertama masih dimuat, BUKAN
                // `undefined`: prop ini wajib menurut `exactOptionalPropertyTypes`, dan
                // melepasnya membuat bilah halaman hilang lalu muncul lagi setiap kali
                // pencarian dijalankan.
                pagination={{
                  page: searchPage,
                  size: search.data?.halaman.ukuran ?? 20,
                  total: search.data?.halaman.total ?? 0,
                  totalPage: search.data?.halaman.total_halaman ?? 1,
                  onPageChange: setSearchPage,
                  isLoading: search.isFetching,
                }}
              />
            </div>
          )}
        </>
      )}

      {tab === 'input' && (
        <div className="mt-4 space-y-4">
          <ClaimPicker
            types={opened.data?.tipe_input ?? []}
            form={claimForm}
            onChange={(patch) => setClaimForm((previous) => ({ ...previous, ...patch }))}
            onSubmit={runClaimSearch}
            claims={claims.data?.klaim ?? []}
            isLoading={claims.isFetching}
            error={
              claims.isError && !isValidationError(claims.error) ? messageOf(claims.error) : ''
            }
            onPick={(claim) => {
              setArchiveForm(formFromClaim(claim))
              setSavedMessage('')
            }}
            pickedNumber={archiveForm?.nomor_klaim ?? ''}
            fieldError={fieldErrorOf(claims.error)}
            searched={submittedClaim !== null}
          />

          {archiveForm ? (
            <ArchiveFormPanel
              form={archiveForm}
              onChange={(patch) =>
                setArchiveForm((previous) => (previous ? { ...previous, ...patch } : previous))
              }
              onSubmit={saveArchive}
              onCancel={() => {
                setArchiveForm(null)
                setSavedMessage('')
              }}
              documentTypes={opened.data?.tipe_dokumen ?? []}
              documentKinds={opened.data?.jenis_dokumen ?? []}
              busy={save.isPending}
              fieldError={saveFieldError(save.error)}
              successMessage=""
            />
          ) : (
            <p className="rounded-kartu border border-dashed border-slate-300 bg-white px-5 py-8 text-center text-sm text-slate-600">
              {savedMessage ||
                'Cari klaim di atas, lalu tekan Detail pada barisnya untuk mengisi berkas arsip.'}
            </p>
          )}
        </div>
      )}

      {tab === 'cabang' && (
        <div className="mt-4">
          <BranchQueue
            files={pending.data?.berkas ?? []}
            page={pending.data?.halaman}
            currentPage={branchPage}
            onPageChange={setBranchPage}
            scope={pending.data?.cakupan_cabang ?? opened.data?.cakupan_cabang}
            isLoading={pending.isFetching}
            error={pending.isError ? messageOf(pending.error) : ''}
            onSend={sendToBranch}
            sendingID={sendingID}
            sendError={send.isError ? messageOf(send.error) : ''}
            sendMessage={sendMessage}
          />
        </div>
      )}
    </PageFrame>
  )
}

/** PageFrame menggambar judul dan kerangka halaman. */
function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto w-full max-w-[100rem] px-4 py-6 sm:px-6 lg:px-8">
      <header>
        <h1 className="text-xl font-semibold text-slate-900">ARCHIVE FILE KLAIM</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pencatatan berkas fisik klaim yang diarsipkan, beserta pengirimannya ke sistem
          Arsip.
        </p>
      </header>

      <div className="mt-5">{children}</div>
    </div>
  )
}

/**
 * archiveColumns menyusun kolom grid ARCHIVE FILE KLAIM.
 *
 * Urutan dan namanya mengikuti harness lama apa adanya. Kolom "Tanggal Kirim Dok" ikut
 * digambar meski sistem lama tidak pernah mengisinya — di sistem baru ia terisi saat
 * berkasnya dikirim, dan kolom yang selalu kosong hanya memberi tahu pengguna bahwa
 * datanya hilang.
 *
 * TIDAK ADA kolom aksi, dan itu disengaja. Grid ini BACA-SAJA, sama seperti sistem lama:
 * penanda `flags` pada prosedur simpan hanya pernah disetel `"insert"` di seluruh export,
 * sehingga tidak ada satu pun jalur sunting dari layar ini (keputusan Work Owner
 * 2026-09-25). Pembetulan data ditempuh lewat DBA.
 */
function archiveColumns(): Column<ArchiveFile>[] {
  return [
    { key: 'nomor_klaim', title: 'NO KLAIM', value: (row) => row.nomor_klaim },
    { key: 'nomor_polis', title: 'NO POLIS', value: (row) => row.nomor_polis },
    { key: 'nama_tertanggung', title: 'TERTANGGUNG', value: (row) => row.nama_tertanggung },
    { key: 'tanggal_kejadian', title: 'DOL', value: (row) => dateText(row.tanggal_kejadian) },
    { key: 'pic_teknis', title: 'PIC Teknis', value: (row) => row.pic_teknis },
    {
      key: 'tanggal_terima_dokumen',
      title: 'Tgl Terima Dokumen',
      value: (row) => dateText(row.tanggal_terima_dokumen),
    },
    { key: 'tanggal_input', title: 'TGL INPUT', value: (row) => dateText(row.tanggal_input) },
    {
      key: 'jumlah_lembar',
      title: 'Jumlah Lembar',
      value: (row) => String(row.jumlah_lembar),
      alignRight: true,
    },
    {
      key: 'tipe_dokumen',
      title: 'Tipe Dokumen',
      // Kode ikut ditampilkan saat namanya kosong. Kosong total membuat pengguna
      // mengira barisnya rusak; kodenya setidaknya dapat dicari di master.
      value: (row) => row.tipe_dokumen || row.kode_tipe_dokumen || '—',
    },
    {
      key: 'jenis_dokumen',
      title: 'Jenis Dokumen',
      value: (row) => row.jenis_dokumen || row.kode_jenis_dokumen || '—',
    },
    { key: 'nama_box', title: 'Nama BOX', value: (row) => row.nama_box },
    { key: 'kode_filling', title: 'Kode Filling', value: (row) => row.kode_filling },
    { key: 'user_input', title: 'USER INPUT', value: (row) => row.user_input },
    {
      key: 'tanggal_kirim_dokumen',
      title: 'Tanggal Kirim Dok',
      value: (row) => dateText(row.tanggal_kirim_dokumen),
    },
  ]
}

function dateText(iso: string | null): string {
  return iso ? formatDate(iso) : '—'
}

function isValidationError(error: unknown): boolean {
  return error instanceof APIError && error.kode === ArchiveError.validationFail
}

/**
 * fieldErrorOf memetakan `detail` galat validasi menjadi pesan per isian.
 *
 * Nama isiannya dibaca dari `field` MAUPUN `kolom`. Keduanya diperiksa karena kontrak
 * galat belum seragam antarmodul — penyeragamannya `TKT-F1-004` yang masih terhalang.
 * Modul ini mengirim `field`; membaca keduanya membuat layar tidak ikut rusak bila
 * kontraknya kelak berubah.
 */
function fieldErrorOf(error: unknown): Record<string, string> {
  if (!(error instanceof APIError)) return {}

  const result: Record<string, string> = {}
  for (const violation of error.detail) {
    const name = violation.field ?? violation.kolom
    if (name) result[name] = violation.pesan
  }
  return result
}

/**
 * saveFieldError menambahkan satu kunci `__umum` untuk galat yang TIDAK menunjuk isian.
 *
 * Tanpa itu, penolakan seperti "berkas sudah tidak ada" atau "identitas tidak terbaca"
 * hilang sama sekali dari layar: tidak ada isian yang dapat ditandai, dan formulirnya
 * tampak seperti tidak menanggapi tombol Simpan.
 */
function saveFieldError(error: unknown): Record<string, string> {
  const result = fieldErrorOf(error)
  if (error && Object.keys(result).length === 0) {
    result['__umum'] = messageOf(error)
  }
  return result
}

function messageOf(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  return 'Terjadi kesalahan pada sistem.'
}
