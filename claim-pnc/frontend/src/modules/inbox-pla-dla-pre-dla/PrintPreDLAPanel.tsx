import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { CloseIcon } from '@/components/Icon'
import { adviceCell } from '@/components/inbox/adviceCell'

import type { Baris, Daftar, DokumenPreDLA } from './types'
import { pesanGalat } from './pesan'

type Props = {
  /** Tab Pre DLA; kolom panelnya datang dari sini (`kolom_cetak`). */
  daftar: Daftar

  /** Baris antrean yang panelnya sedang dibuka. */
  baris: Baris

  data: DokumenPreDLA[] | undefined
  isPending: boolean
  isError: boolean
  error: unknown

  onClose: () => void

  /**
   * Tombol **"Kirim Pre DLA"** — satu per BARIS di dalam grid, persis letaknya di Pega.
   *
   * Menandai SATU Pre-DLA sebagai terkirim.
   */
  onKirim: (dokumen: DokumenPreDLA) => void

  /** Sedang menunggu jawaban penandaan. */
  busy: boolean

  /** Pesan hasil penandaan terakhir; kosong berarti belum ada. */
  pesanKirim?: string
}

/**
 * PrintPreDLAPanel adalah panel **"Print Pre DLA"**.
 *
 * Pengganti `Flow Action/PNCInboxPrintPreDLA-FA.xml` dan
 * `Section/PrintPreDLA-Section.xml`.
 *
 * # Ia tidak mencetak apa pun, dan itu bukan kekurangan sistem baru
 *
 * Tombolnya bernama "Print Pre DLA", tetapi flow action-nya membuka MODAL berisi daftar
 * Pre-DLA beserta tanggal kirimnya — di Pega pun tidak ada berkas yang dihasilkan.
 * Namanya dipertahankan (`D-13`) karena itulah nama yang dikenal pengguna, tetapi
 * kalimat di bawah gridnya menyebutkan apa yang sebenarnya ia lakukan.
 *
 * # Ia menampilkan LEBIH SEDIKIT baris daripada tab Pre DLA
 *
 * Kueri lamanya digabung ke kedua tabel lampiran Pega dan mencocokkan nama berkasnya
 * dengan nomor Pre-DLA. Pre-DLA yang dokumennya belum terlampir karena itu TIDAK muncul
 * di sini, meski barisnya ada di antrean di atas.
 *
 * Itu perilaku Pega yang dibawa apa adanya (`P-5`), dan ia dinyatakan di panelnya sendiri
 * — bukan hanya di kaki halaman. Petugas yang melihat antrean menyebut tiga Pre-DLA lalu
 * membuka panel berisi satu baris akan menyimpulkan datanya hilang, dan kaki halaman
 * terlalu jauh dari tempat kesimpulan itu terbentuk.
 *
 * # "Kirim Pre DLA" adalah tombol PER BARIS
 *
 * Sel tombolnya berada di dalam blok berulang panel ini
 * (`Section/PrintPreDLA-Section.xml`, `REPEATING` 42.697 → `ACTION` 111.588), dan ia
 * mengirim `tmpnodla = .NO_DLA` — nomor baris itu sendiri. Satu tombol di bawah grid akan
 * menandai seluruhnya sekaligus, dan itu operasi yang sama sekali berbeda.
 *
 * # Kenapa tanggal kirimnya diisi SERVER
 *
 * Pega mengirim `tmptglkirim = .TglDLA`, yaitu tanggal kirim yang baru saja ditampilkan.
 * Pada Pre-DLA yang belum pernah dikirim, kolom itu KOSONG — sehingga menekan tombolnya
 * menandai baris terkirim dengan tanggal kirim kosong. Itu kehilangan informasi yang
 * tidak dapat dipulihkan, dan tidak ditiru: tanggalnya diisi basis data.
 *
 * Karena itu layar TIDAK menebak nilainya sendiri; ia memuat ulang panelnya.
 *
 * # Kenapa panel di bawah grid, bukan modal seperti di Pega
 *
 * Flow action-nya `pyModalDisplay = Full screen`, dan itu menutupi antreannya. Di sini ia
 * digambar di bawah antrean, sama dengan panel rincian PLA dan DLA — sehingga pengguna
 * tetap melihat klaim mana yang sedang ia buka, dan susunan layarnya tetap satu pola
 * untuk ketiga tab.
 */
export function PrintPreDLAPanel({
  daftar,
  baris,
  data,
  isPending,
  isError,
  error,
  onClose,
  onKirim,
  busy,
  pesanKirim,
}: Readonly<Props>) {
  const rows = data ?? []

  return (
    <section
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label={`Print Pre DLA klaim ${baris.no_klaim}`}
    >
      <header className="mb-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">Print Pre DLA</h2>
          <p className="mt-1 text-sm text-slate-600">
            Klaim {baris.no_klaim} — {baris.nama_tertanggung}
          </p>
        </div>

        <Button type="button" tone="halus" onClick={onClose}>
          <CloseIcon className="h-4 w-4" />
          <span className="sr-only">Tutup panel Print Pre DLA</span>
        </Button>
      </header>

      <DataTable<DokumenPreDLA>
        columns={kolomCetak(daftar, onKirim, busy)}
        rows={rows}
        rowKey={(row) => `${row.no_advice}-${row.kunci_lampiran}`}
        label="Print Pre DLA"
        isLoading={isPending}
        error={
          isError ? (
            <ErrorMessage
              title="Panel Print Pre DLA tidak dapat dimuat"
              description={pesanGalat(error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={
          'Belum ada Pre-DLA klaim ini yang dokumennya terlampir. Panel ini hanya ' +
          'menampilkan Pre-DLA yang berkasnya sudah ada — barisnya tetap ada di ' +
          'antrean di atas.'
        }
      />

      {/*
        Selisih jumlah baris dinyatakan DI SINI, bukan hanya di kaki halaman.

        Ia hanya digambar ketika panelnya benar-benar berisi: pada panel kosong,
        `emptyMessage` di atas sudah mengatakan hal yang sama, dan dua kalimat yang sama
        berdampingan membuat keduanya lebih sulit dibaca.
      */}
      {rows.length > 0 && (
        <p className="mt-3 text-xs text-slate-500">
          Menampilkan {rows.length} Pre-DLA yang dokumennya sudah terlampir. Pre-DLA yang
          berkasnya belum ada tidak muncul di sini — sama seperti di Pega.
        </p>
      )}

      <p className="mt-3 text-xs text-slate-500">
        <strong>Kirim Pre DLA</strong> menandai <strong>satu</strong> Pre-DLA sebagai
        terkirim beserta tanggalnya. Ia tidak mengirim surat — surat ke reasuradur dikirim
        tombol <strong>Send</strong> di grid rincian PLA dan DLA.
      </p>

      {pesanKirim != null && pesanKirim !== '' && (
        <output
          className="block mt-2 rounded-kontrol bg-emerald-50 px-3 py-2 text-sm text-emerald-800"
        >
          {pesanKirim}
        </output>
      )}
    </section>
  )
}

/**
 * kolomCetak menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * Alasannya sama dengan grid antrean dan grid rincian: kolomnya adalah hasil pembacaan
 * `Section/PrintPreDLA-Section.xml`, dan tempat pembacaan itu tercatat adalah backend.
 * Menuliskannya dengan tangan di sini berarti daftar yang sama hidup di dua tempat.
 */
function kolomCetak(
  daftar: Daftar,
  onKirim: (dokumen: DokumenPreDLA) => void,
  busy: boolean,
): Column<DokumenPreDLA>[] {
  const kolom: Column<DokumenPreDLA>[] = daftar.kolom_cetak.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => gambarSel(row, item.kunci, item.tanggal),
  }))

  // Kolom "Kirim Pre DLA" — sel tombol di dalam grid, persis seperti di Pega.
  //
  // Selnya KOSONG pada Pre-DLA yang sudah terkirim. Berbeda dari tombol "Send" pada grid
  // rincian — yang syarat tampilnya sengaja dicabut Work Owner — di sini menekan tombol
  // pada baris terkirim tidak melakukan apa pun selain menghasilkan penolakan, sehingga
  // menggambarnya hanya menjanjikan sesuatu yang tidak terjadi.
  kolom.push({
    key: 'aksi',
    title: '',
    width: '9rem',
    noSort: true,
    alignRight: true,
    value: () => '',
    render: (row) =>
      row.terkirim === '1' ? null : (
        <Button type="button" onClick={() => onKirim(row)} disabled={busy}>
          {busy ? '…' : 'Kirim Pre DLA'}
        </Button>
      ),
  })

  return kolom
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: DokumenPreDLA, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/**
 * gambarSel menggambar satu sel.
 *
 * Kolom "Terkirim" digambar sebagai lencana, sama dengan grid rincian PLA dan DLA.
 * Nilainya di sini selalu `"0"` atau `"1"` — kuerinya sendiri yang menggantinya lewat
 * `NVL` — dan tidak satu pun terbaca manusia.
 */
function gambarSel(row: DokumenPreDLA, kunci: string, tanggal: boolean) {
  return adviceCell(nilaiSel(row, kunci), kunci, tanggal)
}
