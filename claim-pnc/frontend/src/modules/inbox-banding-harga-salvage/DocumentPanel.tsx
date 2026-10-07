import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { documentDownloadURL, useBandingHargaSalvageDocuments } from './api'
import type { AppealDocument, DocumentScope } from './types'

/**
 * Dialog **"Lihat File"** — dokumen pendukung satu banding harga.
 *
 * # Apa yang digantikan
 *
 *	Flow Action/DokumenBandingSalvage-FA.xml      dialognya
 *	Activity/LihatDokRequestSalvage-Act.xml       pemasok datanya
 *	Section/DokBandingHargaSalvage-Section.xml    grid di dalamnya
 *
 * Ketiga kolomnya mengikuti section lama apa adanya (`D-13`): Kategori, Nama, Tanggal.
 *
 * # Dua hal yang mudah disalahpahami
 *
 * PERTAMA — kolom "Kategori" SELALU bertuliskan "BandingHarga". Ia konstanta yang ditetapkan
 * activity lama untuk setiap baris, bukan isi kolom basis data. Nilainya tetap datang dari
 * server, bukan dituliskan di sini, supaya ia hidup di satu tempat.
 *
 * KEDUA — dokumen yang sudah ditandai ditolak TIDAK muncul. Kueri lama menyaring
 * `IDBALAILELANG IS NULL`, dan penyaring itu ditiru. Akibatnya dokumen banding yang pernah
 * ditolak tidak dapat dibuka lagi dari layar ini.
 *
 * # Unduhannya tautan biasa, bukan permintaan JavaScript
 *
 * Yang diserahkan server adalah isi berkas beserta header `Content-Disposition`. Menariknya
 * lewat `fetch` berarti seluruh berkas ditahan di memori peramban lebih dulu — tanpa satu pun
 * manfaat, karena yang dibutuhkan hanyalah peramban menyimpannya.
 */
type Props = {
  /** Banding yang dokumennya dibuka. `null` berarti dialog ini tertutup. */
  scope: DocumentScope | null
  onClose: () => void
}

export function DocumentPanel({ scope, onClose }: Readonly<Props>) {
  const documents = useBandingHargaSalvageDocuments(scope)

  if (scope === null) return null

  return (
    <section
      className="mt-4 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut"
      aria-label="Dokumen banding harga"
    >
      <header className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">
            Dokumen Banding Harga — {scope.no_klaim}
          </h2>
          <p className="mt-1 text-xs text-slate-600">
            {scope.nama_barang || 'Barang tanpa nama'} · Detail Object{' '}
            {scope.detail_object}
          </p>
        </div>
        <Button tone="halus" onClick={onClose} aria-label="Tutup dokumen banding">
          Tutup
        </Button>
      </header>

      <div className="mt-3">
        <DataTable<AppealDocument>
          columns={columns(scope)}
          rows={documents.data?.baris ?? []}
          rowKey={(row) => row.id}
          label={`Dokumen banding ${scope.no_klaim}`}
          hideSearch
          isLoading={documents.isPending}
          error={
            documents.isError ? (
              <ErrorMessage
                title="Dokumen tidak dapat dimuat"
                description={messageOf(documents.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage={
            'Banding ini tidak punya dokumen pendukung yang dapat dibuka. Dokumen yang ' +
            'sudah Anda tolak sebelumnya memang tidak ditampilkan di sini.'
          }
        />
      </div>
    </section>
  )
}

/**
 * columns menyusun ketiga kolom dialog, mengikuti urutan section lama.
 *
 * Kolom "Nama" digambar sebagai TAUTAN UNDUH, bukan teks. Di layar lama pun begitu: sel itu
 * memakai kontrol `PNCDownloadFile`.
 */
function columns(scope: DocumentScope): Column<AppealDocument>[] {
  return [
    {
      key: 'kategori',
      title: 'Kategori',
      value: (row) => row.kategori,
    },
    {
      key: 'nama',
      title: 'Nama',
      value: (row) => row.nama,
      render: (row) => (
        <a
          href={documentDownloadURL(scope, row.id)}
          /*
            `download` memberi tahu peramban untuk MENYIMPAN, bukan menampilkan. Tanpa itu,
            berkas yang tipe isinya dikenali peramban — PDF, gambar — akan dibuka di tab
            yang sama dan pengguna kehilangan layar yang sedang ia kerjakan.

            Server tetap mengirim `Content-Disposition: attachment`; keduanya berlapis
            karena atribut ini dapat diabaikan pada unduhan lintas-asal.
          */
          download={row.nama || undefined}
          className={[
            'text-blue-700 underline underline-offset-2',
            'hover:text-blue-900 focus:outline-none focus-visible:ring-2',
            'focus-visible:ring-blue-500/40 rounded-sm',
          ].join(' ')}
        >
          {row.nama || 'Berkas tanpa nama'}
        </a>
      ),
    },
    {
      key: 'tanggal_unggah',
      title: 'Tanggal',
      value: (row) => row.tanggal_unggah ?? '',
      render: (row) =>
        row.tanggal_unggah ? (
          formatDate(row.tanggal_unggah)
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
  ]
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
