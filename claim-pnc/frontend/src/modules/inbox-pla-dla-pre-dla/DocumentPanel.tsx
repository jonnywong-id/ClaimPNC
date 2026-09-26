import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { CloseIcon } from '@/components/Icon'

import type { Baris, Daftar, Dokumen } from './types'
import { formatTanggal, pesanGalat } from './pesan'

type Props = {
  /** Daftar yang sedang terbuka; kolom gridnya datang dari sini. */
  daftar: Daftar

  /** Baris antrean yang rinciannya sedang dibuka. */
  baris: Baris

  data: Dokumen[] | undefined
  isPending: boolean
  isError: boolean
  error: unknown

  onClose: () => void
}

/**
 * DocumentPanel adalah grid "Detail PLA List" / "Detail DLA List".
 *
 * # Kenapa panel, bukan grid kedua yang selalu tergambar
 *
 * Di Pega ia grid kedua yang selalu ada di bawah antrean, dan isinya kosong sampai satu
 * baris dipilih. Di sini ia panel yang terbuka atas permintaan, dan alasannya bukan
 * selera: gridnya menembak satu kueri per klaim, dan grid yang selalu tergambar akan
 * menembaknya setiap kali halaman berpindah meski tidak ada yang membukanya.
 *
 * Alur kerjanya tidak berubah — yang dilihat pengguna tetap rincian satu klaim yang ia
 * pilih.
 *
 * # Ia menampilkan SELURUH dokumen klaim itu, terkirim maupun belum
 *
 * Kueri lamanya pun tidak menyaring `ISKIRIM`. Kolom "Terkirim" dan "Tanggal Kirim" justru
 * ada supaya perbedaannya terlihat — dan pada klaim yang sebagian dokumennya sudah
 * dikirim, itulah satu-satunya cara mengetahui sisa pekerjaannya.
 */
export function DocumentPanel({
  daftar,
  baris,
  data,
  isPending,
  isError,
  error,
  onClose,
}: Props) {
  return (
    <section
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label={`Rincian ${daftar.nama} klaim ${baris.no_klaim}`}
    >
      <header className="mb-4 flex items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Detail {daftar.nama} List
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            Klaim {baris.no_klaim} — {baris.nama_tertanggung}
          </p>
        </div>

        <Button type="button" tone="halus" onClick={onClose}>
          <CloseIcon className="h-4 w-4" />
          <span className="sr-only">Tutup rincian</span>
        </Button>
      </header>

      <DataTable<Dokumen>
        columns={kolomDokumen(daftar)}
        rows={data ?? []}
        rowKey={(row) => `${row.no_advice}-${row.tanggal_dokumen}`}
        label={`Rincian ${daftar.nama}`}
        isLoading={isPending}
        error={
          isError ? (
            <ErrorMessage
              title="Rincian tidak dapat dimuat"
              description={pesanGalat(error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={
          `Klaim ini belum punya ${daftar.nama} sama sekali. ` +
          `Itu keadaan yang sah — barisnya masuk antrean karena dokumen jenis lain.`
        }
      />
    </section>
  )
}

/**
 * kolomDokumen menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * # Kenapa kolomnya tidak ditulis di sini
 *
 * Karena kedua grid rinciannya berbeda pada dua kolom — `Revisi` hanya ada di PLA,
 * `No Akseptasi` hanya ada di DLA — dan perbedaan itu datang dari membaca kueri lamanya,
 * bukan dari pilihan tampilan. Menuliskannya dengan tangan di sini berarti daftar yang
 * sama hidup di dua tempat.
 */
function kolomDokumen(daftar: Daftar): Column<Dokumen>[] {
  return daftar.kolom_rincian.map((kolom) => ({
    key: kolom.kunci,
    title: kolom.judul,
    value: (row) => nilaiSel(row, kolom.kunci),
    render: (row) => gambarSel(row, kolom.kunci, kolom.tanggal),
  }))
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: Dokumen, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/**
 * gambarSel menggambar satu sel.
 *
 * Kolom "Terkirim" digambar sebagai lencana, bukan sebagai teks mentah. Nilainya di basis
 * data adalah kosong, `"0"`, atau `"1"` — tidak satu pun terbaca manusia, dan dua yang
 * pertama berarti hal yang SAMA bagi pengguna meski berbeda di data.
 */
function gambarSel(row: Dokumen, kunci: string, tanggal: boolean) {
  const isi = nilaiSel(row, kunci)

  if (kunci === 'terkirim') {
    const sudah = isi === '1'
    return (
      <span
        className={[
          'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
          sudah
            ? 'bg-emerald-50 text-emerald-700'
            : 'bg-amber-50 text-amber-800',
        ].join(' ')}
      >
        {sudah ? 'Terkirim' : 'Belum'}
      </span>
    )
  }

  if (tanggal) return formatTanggal(isi)
  if (isi === '') return <span className="text-slate-400">—</span>
  return isi
}
