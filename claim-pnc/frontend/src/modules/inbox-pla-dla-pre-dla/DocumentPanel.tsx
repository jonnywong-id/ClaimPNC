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

  /**
   * Tombol **"Upload File Penunjang"** — di ATAS grid, persis letaknya di Pega.
   *
   * Belum dibangun; menekannya menjawab alasannya.
   */
  onUpload: () => void

  /**
   * Tombol **"SEND"** — satu per BARIS di dalam grid, persis letaknya di Pega.
   *
   * Belum dibangun; menekannya menjawab alasannya.
   */
  onSend: (dokumen: Dokumen) => void

  /** Sedang menunggu jawaban salah satu tombol di atas. */
  busy: boolean

  /** Pesan hasil pengiriman terakhir; kosong berarti belum ada. */
  pesanKirim?: string
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
 *
 * # "SEND" adalah tombol PER BARIS
 *
 * Di Pega ia sel di dalam grid ini (`Embed-Display-Table-Cell`), bukan tombol di bawahnya.
 * Perbedaannya bukan tata letak: satu tombol di bawah grid mengirim SELURUH dokumen
 * klaim, tombol per baris mengirim SATU dokumen — dan keduanya menghasilkan surat yang
 * berbeda ke reasuradur yang berbeda.
 *
 * # Syarat tampilnya SENGAJA tidak dibawa
 *
 * Pega menyembunyikan tombol ini pada dokumen yang sudah terkirim (`.MARKETING != '1'`,
 * dengan `MARKETING` sebagai alias untuk `ISKIRIM`). Penyembunyian itu **tidak dibawa**
 * atas keputusan Work Owner 2026-09-27: tombolnya digambar pada setiap baris.
 *
 * Akibatnya kolom "Terkirim" menjadi satu-satunya penanda dokumen mana yang sudah
 * dikirim. Hari ini tidak berbahaya — tombolnya menjawab alasan dan tidak mengirim apa
 * pun. Ia menjadi berbahaya pada hari Send benar-benar dibangun, dan itu dicatat sebagai
 * selisih terencana supaya tidak terlewat pada hari itu.
 */
export function DocumentPanel({
  daftar,
  baris,
  data,
  isPending,
  isError,
  error,
  onClose,
  onUpload,
  onSend,
  busy,
  pesanKirim,
}: Props) {
  return (
    <section
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label={`Rincian ${daftar.nama} klaim ${baris.no_klaim}`}
    >
      <header className="mb-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Detail {daftar.nama} List
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            Klaim {baris.no_klaim} — {baris.nama_tertanggung}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/*
            "Upload File Penunjang" ada di ATAS grid rincian di Pega — di antara grid
            antrean dan grid rincian. Letaknya dipertahankan.

            Ia TIDAK dinonaktifkan meski belum dibangun. Tombol yang mati tidak
            menjelaskan apa pun; yang dibutuhkan pengguna adalah tahu MENGAPA ia belum
            dapat dipakai, dan itu hanya sampai bila tombolnya dapat ditekan.
          */}
          <Button type="button" tone="kedua" onClick={onUpload} disabled={busy}>
            Upload File Penunjang
          </Button>

          <Button type="button" tone="halus" onClick={onClose}>
            <CloseIcon className="h-4 w-4" />
            <span className="sr-only">Tutup rincian</span>
          </Button>
        </div>
      </header>

      <DataTable<Dokumen>
        columns={kolomDokumen(daftar, onSend, busy)}
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

      <p className="mt-3 text-xs text-slate-500">
        Tombol <strong>Send</strong> mengirim <strong>satu</strong> {daftar.nama} beserta
        lampirannya ke reasuradur lewat surel, lalu menandainya terkirim. Surat yang sudah
        terkirim tidak dapat ditarik kembali — periksa kolom <strong>Terkirim</strong>
        sebelum menekannya.
      </p>

      {pesanKirim != null && pesanKirim !== '' && (
        <p
          role="status"
          className="mt-2 rounded-kontrol bg-emerald-50 px-3 py-2 text-sm text-emerald-800"
        >
          {pesanKirim}
        </p>
      )}
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
function kolomDokumen(
  daftar: Daftar,
  onSend: (dokumen: Dokumen) => void,
  busy: boolean,
): Column<Dokumen>[] {
  const kolom: Column<Dokumen>[] = daftar.kolom_rincian.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => gambarSel(row, item.kunci, item.tanggal),
  }))

  // Kolom "SEND" — sel tombol di dalam grid, persis seperti di Pega.
  //
  // Tombolnya ada pada SETIAP baris, termasuk dokumen yang sudah terkirim. Syarat tampil
  // Pega (`.MARKETING != '1'`) sengaja tidak dibawa — lihat catatan di kepala berkas.
  kolom.push({
    key: 'aksi',
    title: '',
    width: '6rem',
    noSort: true,
    alignRight: true,
    value: () => '',
    render: (row) => (
      <Button type="button" onClick={() => onSend(row)} disabled={busy}>
        {busy ? '…' : 'Send'}
      </Button>
    ),
  })

  return kolom
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
