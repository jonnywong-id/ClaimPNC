import type { ReactNode } from 'react'
import type { UseQueryResult } from '@tanstack/react-query'

import type { BusinessListResponse } from '@/api/types'
import { Button, type ButtonTone } from '@/components/Button'
import type { Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { AddIcon, EditIcon, ReloadIcon } from '@/components/Icon'

import { portalNotSelected, retryLoadMessage, type MessageContent } from './loadMessage'

/**
 * Kerangka bersama layar master: kepala layar, baris portal entitas, tombol Refresh dan
 * Tambah, dan keadaan daftar (portal belum dipilih, memuat, gagal, berhasil).
 *
 * Puluhan layar master menggambar bagian-bagian ini dengan markup yang sama persis; yang
 * berbeda hanya teksnya. Satu tempat berarti satu perbaikan berlaku untuk semuanya.
 */

/** Pesan galat dari {@link MessageContent}. */
export function MessageBox({ message }: Readonly<{ message: MessageContent }>) {
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}

/**
 * Portal belum dipilih, digambar sebagai pesan galat.
 *
 * `owner` dan `place` mengikuti kalimat layar masing-masing — lihat `portalNotSelected`.
 */
export function PortalNotSelected({
  owner,
  place,
}: Readonly<{ owner?: string | undefined; place?: 'bilah' | 'bagian' | undefined }>) {
  return <MessageBox message={portalNotSelected(owner, place)} />
}

/**
 * Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat badan
 * hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya diandaikan
 * pengguna (ADR-0030, R-20).
 */
export function EntityPortalLine({ portal }: Readonly<{ portal: string | null | undefined }>) {
  return (
    <p className="mt-3 text-xs text-slate-500">
      Portal entitas:{' '}
      <span className="font-medium text-slate-700">{portal ?? '—'}</span>
    </p>
  )
}

/**
 * Kepala layar bergaya "Master Data / <crumb>": jejak lokasi, judul, dan keterangan.
 * `children` digambar sesudah keterangan — biasanya {@link EntityPortalLine}.
 */
export function MasterDataHeader({
  crumb,
  title,
  description,
  children,
}: Readonly<{ crumb: string; title: string; description: ReactNode; children?: ReactNode }>) {
  return (
    <header className="mb-6">
      <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
        <ol className="flex items-center gap-1.5">
          <li>Master Data</li>
          <li aria-hidden="true" className="text-slate-300">
            /
          </li>
          <li className="text-slate-700">{crumb}</li>
        </ol>
      </nav>
      <h1 className="text-2xl font-semibold tracking-tight text-slate-900">{title}</h1>
      <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">{description}</p>
      {children}
    </header>
  )
}

/** Kepala layar ringkas: judul dan keterangan di kiri, tombol di kanan. */
export function ListHeader({
  title,
  description,
  children,
}: Readonly<{ title: ReactNode; description: ReactNode; children: ReactNode }>) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
      <div>
        <h1 className="text-xl font-semibold text-slate-900">{title}</h1>
        <p className="text-sm text-slate-600">{description}</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">{children}</div>
    </header>
  )
}

type Refetchable = { isFetching: boolean; refetch: () => unknown }

/** Tombol Refresh; `withIcon` menambah ikon putar seperti kepala layar bergaya jejak lokasi. */
export function RefreshButton({
  query,
  withIcon = false,
  label = 'Refresh',
}: Readonly<{ query: Refetchable; withIcon?: boolean | undefined; label?: string | undefined }>) {
  return (
    <Button tone="kedua" onClick={() => { query.refetch() }} disabled={query.isFetching}>
      {withIcon && (
        <ReloadIcon className={`h-4 w-4 ${query.isFetching ? 'animate-spin' : ''}`} />
      )}
      {query.isFetching ? 'Memuat…' : label}
    </Button>
  )
}

/** Tombol Tambah. */
export function AddButton({
  onClick,
  disabled,
  withIcon = false,
}: Readonly<{ onClick: () => void; disabled: boolean; withIcon?: boolean | undefined }>) {
  return (
    <Button tone="utama" onClick={onClick} disabled={disabled}>
      {withIcon && <AddIcon className="h-4 w-4" />}
      Tambah
    </Button>
  )
}

/**
 * Isi bagian daftar menurut keadaan portal dan kueri: portal belum dipilih, sedang memuat,
 * gagal memuat, atau `render` atas data yang berhasil dimuat.
 */
export function renderListState<TData, TError>({
  portal,
  query,
  loadingText,
  toMessage,
  portalOwner,
  render,
}: Readonly<{
  /** null = portal belum dipilih; dibiarkan kosong bila portal sudah diperiksa pemanggil. */
  portal?: string | null | undefined
  query: UseQueryResult<TData, TError>
  loadingText: string
  toMessage: (error: TError) => MessageContent
  portalOwner?: string | undefined
  render: (data: TData) => ReactNode
}>): ReactNode {
  if (portal === null) {
    return <PortalNotSelected owner={portalOwner} place="bagian" />
  }
  if (query.isPending) {
    return <p className="text-sm text-slate-500">{loadingText}</p>
  }
  if (query.isError) {
    return <MessageBox message={toMessage(query.error)} />
  }
  return render(query.data)
}

/**
 * Peringatan bahwa daftar bisnis gagal dimuat, digambar di atas form yang memetakan bisnis.
 *
 * Kegagalan memuat daftar bisnis TIDAK menutup form dan tidak menghalangi penyimpanan: nama
 * bisnis memang boleh diketik sendiri. Yang hilang hanya sarannya, dan form itu sendiri
 * yang mengatakannya.
 */
export function BusinessListWarning({ show }: Readonly<{ show: boolean }>) {
  if (!show) return null
  return (
    <div className="mb-3">
      <ErrorMessage
        title="Daftar bisnis tidak dapat dimuat"
        description="Saran nama bisnis tidak tersedia untuk sementara. Namanya tetap dapat diketik sendiri, dan seluruh isian tetap dapat disimpan."
        tone="gangguan"
      />
    </div>
  )
}

/**
 * Nama bisnis yang dikirim pada pemetaan bisnis.
 *
 * NAMA yang dikirim, bukan ID: itulah yang diketik dan dilihat petugas, dan nama yang
 * diketik bebas memang tidak punya ID. Server yang menyelesaikannya menjadi ID dengan
 * mencocokkan ke master.
 *
 * Baris yang dibiarkan kosong dibuang di sini supaya tidak terkirim sebagai pemetaan ke
 * bisnis bernama kosong — server pun membuangnya, tetapi membuangnya lebih awal membuat
 * permintaannya menyatakan apa yang benar-benar dimaksud.
 */
export function businessNames(rows: readonly { nama: string }[]): string[] {
  return rows.map((b) => b.nama.trim()).filter((nama) => nama !== '')
}

/**
 * Kolom aksi "Ubah" per baris.
 *
 * Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya tombol, bukan
 * data. Posisinya paling kanan, sama seperti kolom tanpa judul yang memuat tombol ubah
 * pada grid Pega.
 */
export function editColumn<T>({
  onEdit,
  width,
  tone = 'kedua',
  ariaLabel,
  disabled,
  withIcon = false,
  label = 'Ubah',
  title = 'Aksi',
}: Readonly<{
  onEdit: (row: T) => void
  width: string
  tone?: ButtonTone | undefined
  ariaLabel?: ((row: T) => string) | undefined
  disabled?: boolean | undefined
  withIcon?: boolean | undefined
  label?: string | undefined
  /** Judul kolom; beberapa layar lama tidak memberinya judul sama sekali. */
  title?: string | undefined
}>): Column<T> {
  return {
    key: 'aksi',
    title,
    width,
    noSort: true,
    alignRight: true,
    value: () => '',
    render: (row) => (
      <Button
        tone={tone}
        onClick={() => onEdit(row)}
        aria-label={ariaLabel?.(row)}
        disabled={disabled}
      >
        {withIcon && <EditIcon className="h-3.5 w-3.5" />}
        {label}
      </Button>
    ),
  }
}

/**
 * Kerangka layar master ringkas: kepala layar dengan Refresh dan Tambah, baris portal
 * entitas, form (bila terbuka), lalu bagian daftar.
 */
export function MasterListLayout<TData extends { portal?: string | undefined }, TError>({
  maxWidth = 'max-w-5xl',
  title,
  description,
  addFirst = false,
  query,
  portal,
  crud,
  form,
  loadingText,
  toMessage = retryLoadMessage,
  portalOwner,
  renderTable,
}: Readonly<{
  maxWidth?: string | undefined
  title: ReactNode
  description: ReactNode
  /** Tombol Tambah digambar di depan Refresh, mengikuti urutan layar lama. */
  addFirst?: boolean | undefined
  query: UseQueryResult<TData, TError>
  portal: string | null
  /** Keadaan form tambah/ubah — lihat useCrudForm. */
  crud: { isOpen: boolean; openCreate: () => void }
  form: ReactNode
  loadingText: string
  /** Bawaannya gaya "coba beberapa saat lagi" — lihat retryLoadMessage. */
  toMessage?: ((error: TError) => MessageContent) | undefined
  portalOwner?: string | undefined
  renderTable: (data: TData) => ReactNode
}>) {
  const refresh = <RefreshButton query={query} />
  const add = <AddButton onClick={crud.openCreate} disabled={crud.isOpen} />
  return (
    <main className={`mx-auto ${maxWidth} px-4 py-8`}>
      <ListHeader title={title} description={description}>
        {addFirst ? add : refresh}
        {addFirst ? refresh : add}
      </ListHeader>

      <EntityPortalLine portal={query.data?.portal ?? portal} />

      {crud.isOpen && <section className="mt-5">{form}</section>}

      <section className="mt-6">
        {renderListState({
          portal,
          query,
          loadingText,
          toMessage,
          portalOwner,
          render: renderTable,
        })}
      </section>
    </main>
  )
}

/** Tombol Refresh lalu Tambah, keduanya berikon — dipasang pada `actions` DataTable. */
export function RefreshAddActions({
  query,
  onAdd,
  addDisabled,
  refreshLabel,
}: Readonly<{
  query: Refetchable
  onAdd: () => void
  addDisabled: boolean
  refreshLabel?: string | undefined
}>) {
  return (
    <>
      <RefreshButton query={query} withIcon label={refreshLabel} />
      <AddButton onClick={onAdd} disabled={addDisabled} withIcon />
    </>
  )
}

/**
 * Kerangka layar master bergaya jejak lokasi "Master Data / …": kepala layar beserta
 * portal entitas, pemberitahuan (bila ada), form (bila terbuka), lalu isi layar — atau
 * pesan portal belum dipilih.
 */
export function MasterDataLayout({
  maxWidth = 'max-w-6xl',
  crumb,
  title,
  description,
  headerNote,
  portalLabel,
  notice,
  form,
  afterForm,
  portal,
  portalOwner,
  children,
}: Readonly<{
  maxWidth?: string | undefined
  crumb: string
  title: string
  description: ReactNode
  /** Digambar di kepala layar, di antara keterangan dan portal entitas. */
  headerNote?: ReactNode
  /** Nama entitas yang ditampilkan — biasanya `list.data?.portal ?? portal`. */
  portalLabel: string | null | undefined
  /** Digambar di antara kepala layar dan form. */
  notice?: ReactNode
  /** Form yang sedang terbuka, atau `false`/`null` bila tertutup. */
  form: ReactNode
  /** Digambar di antara form dan isi layar. */
  afterForm?: ReactNode
  portal: string | null
  portalOwner?: string | undefined
  children: ReactNode
}>) {
  return (
    <div className={`mx-auto ${maxWidth} px-4 py-8 sm:px-6`}>
      <MasterDataHeader crumb={crumb} title={title} description={description}>
        {headerNote}
        <EntityPortalLine portal={portalLabel} />
      </MasterDataHeader>

      {notice}

      {form && <div className="mb-6">{form}</div>}

      {afterForm}

      {portal === null ? <PortalNotSelected owner={portalOwner} place="bilah" /> : children}
    </div>
  )
}

/**
 * Prop bersama form yang memetakan bisnis: saran nama bisnis, tanda pemetaan sedang dimuat,
 * dan keadaan simpan dari useCrudForm.
 */
export function businessFormProps(
  businesses: Readonly<{ data?: BusinessListResponse | undefined }>,
  isLoadingBusinessMapping: boolean,
  form: Readonly<{ isSaving: boolean; saveError: unknown; closeForm: () => void }>,
) {
  return {
    businesses: businesses.data?.bisnis ?? [],
    isLoadingBusinessMapping,
    isSaving: form.isSaving,
    error: form.saveError,
    onCancel: form.closeForm,
  }
}
