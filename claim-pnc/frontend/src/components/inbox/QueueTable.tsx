import { DataTable, type Column, type ServerPagination } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { InboxPagination, type PrevNextPageInfo } from './InboxPagination'

/** Bentuk hasil kueri daftar inbox yang dibaca QueueTable. */
export type QueueQuery<T> = {
  data?: { baris: T[]; paginasi: PrevNextPageInfo } | undefined
  isPending: boolean
  isError: boolean
  isFetching: boolean
  error: unknown
}

type Props<T> = {
  query: QueueQuery<T>
  columns: Column<T>[]
  rowKey: (row: T) => string
  title: string
  emptyMessage: string
  onMove: (page: number) => void
  describe: (error: unknown) => string
  /** Judul pesan saat daftar gagal dimuat. */
  failureTitle?: string | undefined
}

/**
 * QueueTable menggambar tabel antrean inbox beserta paginasinya.
 *
 * Kotak cari bawaan DataTable disembunyikan: hasilnya akan menyaring HANYA halaman yang
 * sedang terbuka, sehingga pengguna dapat diberi tahu "tidak ada" untuk baris yang
 * sebenarnya ada di halaman berikutnya. Layar lama pun tidak punya kotak cari.
 *
 * Paginasi hanya digambar bila ada baris sama sekali — "Menampilkan 0–0 dari 0" tidak
 * memberi tahu apa pun yang belum dikatakan pesan kosong tabel.
 */
export function QueueTable<T>(props: Readonly<Props<T>>) {
  const { query, onMove } = props
  return (
    <>
      <QueueGrid<T> {...props} />

      {query.data && query.data.paginasi.total > 0 && (
        <InboxPagination
          info={query.data.paginasi}
          visible={query.data.baris.length}
          onMove={onMove}
          loading={query.isFetching}
        />
      )}
    </>
  )
}

type PagedProps<T> = Omit<Props<T>, 'onMove'> & {
  /** Nama tabel bagi pembaca layar. */
  label: string
  onPageChange: (page: number) => void
  /** Ukuran halaman sebelum jawaban server pertama tiba. */
  defaultSize: number
}

/**
 * PagedQueueTable sama dengan QueueTable, tetapi memakai paginasi bernomor bawaan
 * DataTable (`pagination`) alih-alih "sebelumnya / berikutnya".
 */
export function PagedQueueTable<T>({
  query,
  label,
  onPageChange,
  defaultSize,
  ...rest
}: Readonly<PagedProps<T>>) {
  return (
    <QueueGrid<T>
      {...rest}
      query={query}
      label={label}
      pagination={serverPaging(query, onPageChange, defaultSize)}
    />
  )
}

/**
 * serverPaging menerjemahkan paginasi yang dikirim server menjadi paginasi bernomor
 * DataTable. Sebelum jawaban pertama tiba, halaman 1 dari 1 dengan `defaultSize` baris.
 */
export function serverPaging(
  query: { data?: { paginasi: PrevNextPageInfo } | undefined; isFetching: boolean },
  onPageChange: (page: number) => void,
  defaultSize: number,
  fallback: { page: number; totalPage: number } = { page: 1, totalPage: 1 },
): ServerPagination {
  return {
    page: query.data?.paginasi.halaman ?? fallback.page,
    size: query.data?.paginasi.ukuran ?? defaultSize,
    total: query.data?.paginasi.total ?? 0,
    totalPage: query.data?.paginasi.total_halaman ?? fallback.totalPage,
    onPageChange,
    isLoading: query.isFetching,
  }
}

/**
 * TabQueueTable adalah PagedQueueTable untuk satu tab antrean penyelia: judulnya nama tab,
 * nama aksesibelnya "Antrean <nama tab>", dan halamannya 50 baris sebelum server menjawab.
 */
export function TabQueueTable<T>({
  tab,
  ...rest
}: Readonly<
  Omit<PagedProps<T>, 'title' | 'label' | 'defaultSize'> & { tab: { nama: string } }
>) {
  return (
    <div className="mt-4">
      <PagedQueueTable<T>
        {...rest}
        title={tab.nama}
        label={`Antrean ${tab.nama}`}
        defaultSize={50}
      />
    </div>
  )
}

type GridProps<T> =Omit<Props<T>, 'onMove'> & {
  label?: string | undefined
  pagination?: ServerPagination | undefined
}

/** QueueGrid menggambar DataTable antrean tanpa paginasi "sebelumnya / berikutnya". */
function QueueGrid<T>({
  query,
  columns,
  rowKey,
  title,
  emptyMessage,
  describe,
  failureTitle = 'Antrean tidak dapat dimuat',
  label,
  pagination,
}: Readonly<GridProps<T>>) {
  return (
    <DataTable<T>
      columns={columns}
      rows={query.data?.baris ?? []}
      rowKey={rowKey}
      title={title}
      {...(label === undefined ? {} : { label })}
      hideSearch
      isLoading={query.isPending}
      error={
        query.isError ? (
          <ErrorMessage title={failureTitle} description={describe(query.error)} tone="gangguan" />
        ) : undefined
      }
      emptyMessage={emptyMessage}
      {...(pagination === undefined ? {} : { pagination })}
    />
  )
}
