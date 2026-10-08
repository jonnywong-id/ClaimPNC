import { Button } from '@/components/Button'
import { DataTable, type Column, type ServerPagination } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ReloadIcon } from '@/components/Icon'

type Props<T> = {
  columns: Column<T>[]
  rowKey: (row: T) => string
  title: string
  label: string
  /** Keterangan di bawah judul dari jumlah baris; hanya digambar bila ada baris. */
  description: (total: number) => string
  searchLabel: string
  emptyMessage: string
  /** Kata kunci pencarian sisi server. */
  search: string
  onSearch: (value: string) => void
  /** Jumlah baris yang dilewati (paginasi berbasis offset). */
  offset: number
  onOffset: (offset: number) => void
  pageSize: number
  /** Hasil kueri: satu halaman baris beserta jumlah seluruh baris yang cocok. */
  query: {
    data?: { data: T[]; total: number } | undefined
    isPending: boolean
    isFetching: boolean
    isError: boolean
    error: unknown
    refetch: () => unknown
  }
  describe: (error: unknown) => string
}

/**
 * TaskQueueTable menggambar antrean tugas pribadi (Inbox Analyst Doctor, Inbox RCL):
 * pencarian sisi server, tombol "Muat ulang", dan paginasi berbasis offset.
 *
 * Pencariannya dikerjakan SERVER, bukan disaring di peramban: antreannya berhalaman, dan
 * menyaring halaman yang sedang terbuka saja akan memberi tahu "tidak ada" untuk baris yang
 * sebenarnya ada di halaman lain.
 *
 * Tombol "Muat ulang" ada karena antrean ini diisi oleh orang lain — tugas baru dapat masuk
 * kapan saja selama layar terbuka.
 */
export function TaskQueueTable<T>({
  columns,
  rowKey,
  title,
  label,
  description,
  searchLabel,
  emptyMessage,
  search,
  onSearch,
  offset,
  onOffset,
  pageSize,
  query,
  describe,
}: Readonly<Props<T>>) {
  const rows = query.data?.data ?? []
  const total = query.data?.total ?? 0

  return (
    <DataTable<T>
      columns={columns}
      rows={rows}
      rowKey={rowKey}
      title={title}
      label={label}
      {...(total > 0 ? { description: description(total) } : {})}
      isLoading={query.isPending}
      searchLabel={searchLabel}
      emptyMessage={emptyMessage}
      serverSearch={{ value: search, onChange: onSearch, matchCount: total }}
      actions={
        <Button
          tone="halus"
          onClick={() => { query.refetch() }}
          disabled={query.isFetching}
        >
          <ReloadIcon className="h-4 w-4" />
          {query.isFetching ? 'Memuat…' : 'Muat ulang'}
        </Button>
      }
      error={
        query.isError ? (
          <ErrorMessage
            title="Antrean tidak dapat dimuat"
            description={describe(query.error)}
            tone="gangguan"
          />
        ) : undefined
      }
      pagination={offsetPagination(offset, pageSize, total, onOffset, query.isFetching)}
    />
  )
}

/**
 * offsetPagination menerjemahkan paginasi berbasis offset (jumlah baris yang dilewati)
 * menjadi paginasi bernomor DataTable.
 */
export function offsetPagination(
  offset: number,
  pageSize: number,
  total: number,
  onOffset: (offset: number) => void,
  isLoading: boolean,
): ServerPagination {
  return {
    page: Math.floor(offset / pageSize) + 1,
    size: pageSize,
    total,
    totalPage: Math.max(1, Math.ceil(total / pageSize)),
    onPageChange: (nomor) => onOffset((nomor - 1) * pageSize),
    isLoading,
  }
}
