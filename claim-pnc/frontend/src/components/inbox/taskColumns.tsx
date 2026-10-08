import type { Column } from '@/components/DataTable'

/** Satu kolom layar yang bentuknya ditetapkan server (judul dari export Pega). */
export type TaskScreenColumn = {
  kunci: string
  judul: string
}

/** Cara menggambar satu kolom, dikunci dengan `kunci` yang dikirim server. */
export type TaskRenderer<T, K extends TaskScreenColumn> = (
  k: K,
  open: (caseNumber: string) => void,
) => Column<T>

/**
 * buildTaskColumns menyusun kolom tabel dari judul yang dikirim server.
 *
 * Yang datang dari server hanyalah KUNCI dan JUDULNYA; cara menggambarnya tetap milik layar.
 * Kunci yang tidak dikenal dilewati, bukan digambar kosong: kolom baru menuntut keputusan
 * tampilan yang belum diambil, dan kolom kosong tanpa isi hanya menambah lebar tabel.
 */
export function buildTaskColumns<T, K extends TaskScreenColumn>(
  kolom: K[],
  renderers: Record<string, TaskRenderer<T, K> | undefined>,
  open: (caseNumber: string) => void,
): Column<T>[] {
  const result: Column<T>[] = []
  for (const k of kolom) {
    const built = renderers[k.kunci]?.(k, open)
    if (built) result.push(built)
  }
  return result
}

/**
 * commonTaskRenderers adalah cara menggambar tiga kolom yang dimiliki setiap antrean tugas
 * pribadi: Nomor Case (dapat diklik), No Polis, dan Nama Tertanggung.
 */
export function commonTaskRenderers<
  T extends { nomor_case: string; nomor_polis: string; nama_tertanggung: string },
  K extends TaskScreenColumn,
>(): Record<string, TaskRenderer<T, K>> {
  return {
    nomor_case: (k, open) => caseNumberColumn<T>(k, open),
    nomor_polis: (k) => textColumn<T>(k, '11rem', (row) => row.nomor_polis),
    nama_tertanggung: (k) => textColumn<T>(k, '14rem', (row) => row.nama_tertanggung),
  }
}

/** PlainText menggambar teks satu sel; kosong menjadi tanda pisah. */
export function PlainText({ value }: Readonly<{ value: string }>) {
  if (!value) return <span className="text-slate-400">—</span>
  return <span className="truncate">{value}</span>
}

/** textColumn menyusun kolom teks polos selebar `width`. */
export function textColumn<T>(
  k: TaskScreenColumn,
  width: string,
  pick: (row: T) => string,
): Column<T> {
  return {
    key: k.kunci,
    title: k.judul,
    width,
    value: pick,
    render: (row) => <PlainText value={pick(row)} />,
  }
}

/**
 * caseNumberColumn menyusun kolom Nomor Case yang dapat diklik untuk membuka klaimnya.
 *
 * Klaim yang belum bernomor tidak dapat dibuka, dan itu dinyatakan — bukan digambar
 * sebagai tombol yang tidak menuju ke mana pun.
 */
export function caseNumberColumn<T extends { nomor_case: string }>(
  k: TaskScreenColumn,
  open: (caseNumber: string) => void,
): Column<T> {
  return {
    key: k.kunci,
    title: k.judul,
    width: '11rem',
    value: (row) => row.nomor_case,
    render: (row) =>
      row.nomor_case ? (
        <button
          type="button"
          className="rounded-md font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          onClick={() => open(row.nomor_case)}
        >
          {row.nomor_case}
        </button>
      ) : (
        <span className="text-slate-400">belum bernomor</span>
      ),
  }
}
