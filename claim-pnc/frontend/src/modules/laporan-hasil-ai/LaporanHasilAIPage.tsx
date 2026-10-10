import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { formatDate } from '@/components/format'

import { useExportLaporanHasilAI, useLaporanHasilAI } from './api'
import { emptyFilter, isComplete, type FilterInput, type ReportRow } from './types'

/**
 * Laporan Hasil AI — butir menu `MENU_ID 82`, pengganti harness `Har_LaporanHasilAI`.
 *
 * # Apa yang dilaporkan layar ini
 *
 * Hasil penilaian **AI** atas sebuah klaim, disandingkan dengan **keputusan komite** yang
 * menyusul. Gunanya membandingkan keduanya: seberapa sering komite sependapat dengan AI,
 * dan pada kasus seperti apa keduanya berbeda.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/SecLaporanHasilAI-Section.xml`: judul, dua isian tanggal beserta
 * tombol "Cari Data" dan "Export To Excel", lalu SATU grid rincian. Judul kolom TIDAK
 * diterjemahkan: `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang selama ini
 * dibaca pengguna.
 *
 * # SATU grid, bukan dua
 *
 * Export section-nya memuat grid KEDUA terikat `TempTotal.pxResults` — pencacah
 * "Keputusan · Total · Diterima · Ditolak" — dan activity-nya benar-benar mengisinya dua
 * baris. Layar ini sempat membangunnya karena itu.
 *
 * **Pega yang berjalan tidak memilikinya** (Work Owner, 2026-10-03), dan layar yang
 * berjalan mengalahkan export yang kuerinya sendiri bertanda `work in progress`. Grid
 * ringkasan beserta kueri agregat di belakangnya dibuang seluruhnya.
 *
 * # Tiga hal yang paling mudah disalahpahami di layar ini
 *
 * PERTAMA — **lima kolom SELALU kosong**: Object Name, Note AI Terima, Note AI Tolak,
 * Coverage Final, dan Kategori Kronologi. Itu bukan kerusakan dan bukan data yang hilang;
 * kueri layar lamanya memang tidak memilih kolomnya (`pyMemo = "work in progress"`), dan
 * Work Owner memutuskan pada 2026-09-26 untuk menirunya apa adanya.
 *
 * KEDUA — **isian berlabel "Tgl Input" sebenarnya menyaring Tanggal Komite**. Labelnya
 * menyesatkan sejak di Pega, dan dipakai apa adanya atas keputusan yang sama.
 *
 * KETIGA — **"No Klaim" kosong pada jenjang komite kedua ke atas**, sehingga satu klaim
 * terbaca sebagai satu kelompok. Padanan `CASE WHEN B.KOMITEKE = '1' THEN … ELSE '' END`.
 */
export function LaporanHasilAIPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Penyaring dipegang DUA kali: satu yang sedang diketik, satu yang sudah dikirim.
  //
  // Pemisahan ini mengikuti layar lama, yang punya tombol "Cari Data" — bukan menyaring
  // saat mengetik. Alasannya masih berlaku: tanggal yang setengah diketik hanya
  // menghasilkan penolakan validasi beruntun.
  const [draft, setDraft] = useState<FilterInput>(emptyFilter)
  const [applied, setApplied] = useState<FilterInput | null>(null)
  const [page, setPage] = useState(1)

  const active = applied ?? emptyFilter
  const searched = applied !== null

  const report = useLaporanHasilAI(active, page, searched)
  const exportFile = useExportLaporanHasilAI()

  // Pelanggaran per isian datang dari server, bukan dihitung ulang di layar.
  //
  // Aturannya hidup di domain (`internal/laporanhasilai/laporanhasilai.go`), dan
  // menyalinnya ke sini berarti dua tempat yang dapat berselisih — dengan yang di layar
  // selalu menang lebih dulu, sehingga selisihnya tidak pernah terlihat.
  const violations = violationsOf(report.error ?? exportFile.error)

  const rows = report.data?.baris ?? []

  return (
    /*
      Akar halaman mengikuti Report Klaim PERSIS: `mx-auto max-w-6xl space-y-6 p-4 sm:p-6`.

      `<main>` pada PageShell ber-`min-w-0 flex-1 pb-16` — TANPA padding horizontal sama
      sekali. Setiap halaman karena itu menyediakan paddingnya sendiri, dan halaman yang
      lupa akan menempelkan judulnya ke sisi sidebar. Itu yang terjadi di sini sebelum
      2026-09-26.
    */
    <div className="mx-auto max-w-6xl space-y-6 p-4 sm:p-6">
      <header className="space-y-1">
        <h1 className="text-xl font-semibold text-slate-900">Laporan Hasil Data AI</h1>
        <p className="text-sm text-slate-600">
          Penilaian AI atas klaim, disandingkan dengan keputusan komitenya
          {portal ? ` — entitas ${portal}` : ''}.
        </p>
      </header>

      {/*
        Kartu penyaring memakai `rounded-kartu`, BUKAN `rounded-kotak`.

        `rounded-kotak` tidak punya token radius sama sekali — `src/styles.css` hanya
        mendefinisikan `--radius-kartu` dan `--radius-kontrol`. Kelasnya karena itu INERT:
        ia tertulis, lolos `tsc`, lolos seluruh uji, dan tidak menghasilkan satu piksel pun
        lengkung. Kartu ini tergambar bersudut tajam sementara 203 kartu lain di aplikasi
        melengkung.

        Sekelas dengan jebakan `w-24` pada prop `width` DataTable: nama kelas yang tidak
        dikenal Tailwind tidak pernah mengeluh tentang dirinya sendiri.
      */}
      <form
        className="space-y-4 rounded-kartu border border-slate-200 bg-white p-4"
        onSubmit={(event) => {
          event.preventDefault()
          setApplied(draft)
          setPage(1)
        }}
      >
        {/*
          Bilah penyaring mengikuti Report Klaim: `flex flex-wrap` beserta `max-w-xs` pada
          tiap isian — BUKAN grid berkolom tetap.

          Sebabnya terukur. Dengan `lg:grid-cols-3` dan hanya DUA isian, tiap kolom
          selebar ±360 px sehingga kotak pemilih tanggal melar hampir dua kali lebar
          wajarnya, lalu menyisakan satu kolom kosong di kanan. Dengan flex, keduanya
          menyusut ke lebar intrinsiknya dan berhenti di 320 px.

          SATU penyimpangan dari Report Klaim: `items-start`, bukan `items-end`. Pesan
          galat digambar DI BAWAH isian, dan pada layar ini galatnya sering hanya mengenai
          salah satu tanggal — merapatkan dasar kedua isian akan menggeser isian yang
          benar ke bawah, seolah ia yang bermasalah.
        */}
        <div className="flex flex-wrap items-start gap-6">
          {/*
            Label keduanya disalin APA ADANYA dari layar lama, termasuk yang menyesatkan.
            Lihat catatan KEDUA pada doc komponen ini.
          */}
          <FormField
            id="laporan-ai-dari"
            label="Tgl Input Dari"
            type="date"
            className="max-w-xs"
            value={draft.dari}
            failure={violations['dari']}
            onChange={(event) => setDraft({ ...draft, dari: event.target.value })}
          />

          <FormField
            id="laporan-ai-sampai"
            label="Tgl Input Sampai"
            type="date"
            className="max-w-xs"
            value={draft.sampai}
            failure={violations['sampai']}
            onChange={(event) => setDraft({ ...draft, sampai: event.target.value })}
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" tone="utama">
            Cari Data
          </Button>
          {/*
            Ekspor memakai penyaring yang SUDAH DIKIRIM, bukan yang sedang diketik.
            Berkas yang isinya berbeda dari yang terlihat di layar adalah berkas yang tidak
            dapat dicocokkan dengan apa pun.

            Ia dinonaktifkan sebelum pencarian pertama: di layar lama pun tombolnya
            memanggil activity yang sama, sehingga menekannya tanpa tanggal hanya
            menghasilkan galat.
          */}
          <Button
            type="button"
            onClick={() => exportFile.mutate(active)}
            disabled={!searched || !isComplete(active) || exportFile.isPending}
          >
            Export To Excel
          </Button>
        </div>
      </form>

      {exportFile.isError && (
        <ErrorMessage
          title="Berkas tidak dapat diunduh"
          description={messageOf(exportFile.error)}
          tone="gangguan"
        />
      )}

      {/*
        KEDUA GRID DIGAMBAR SEJAK AWAL, sebelum satu tanggal pun diisi.

        Itu mengikuti Pega, bukan penyederhanaan: harness-nya merender section pada saat
        dimuat, dan grid yang page list-nya masih kosong digambar lengkap dengan judul
        kolomnya ditambah `pyGridNoResultsMessage` di bawahnya — rule yang memang terdaftar
        di `Harness/Har_LaporanHasilAI-Harness.xml`.

        Sebelum 2026-10-03 layar ini menyembunyikan keduanya dan hanya menampilkan satu
        paragraf ajakan. Akibatnya pengguna tidak dapat melihat kolom apa saja yang akan
        didapatnya sebelum mencari — padahal di Pega ia terbaca sejak layar terbuka. Work
        Owner meminta perilaku Pega yang diikuti.

        Yang TIDAK berubah: permintaan ke server tetap ditahan sampai tombol ditekan
        (`enabled` pada `useLaporanHasilAI`). Pega pun tidak menjalankan kuerinya saat
        memuat layar — grid kosongnya digambar tanpa satu pun pembacaan.
      */}

      <DataTable<ReportRow>
        title="Rincian"
        label="Penilaian AI beserta keputusan komitenya"
        description={
          'Satu baris adalah satu penilaian AI pada satu objek pertanggungan, ' +
          'sehingga satu klaim dapat muncul beberapa kali.'
        }
        columns={detailColumns}
        rows={rows}
        rowKey={(row) => row.id}
        isLoading={report.isLoading}
        hideSearch
        showHeaderWhenEmpty
        emptyMessage={emptyMessage}
        error={
          report.isError ? (
            <ErrorMessage
              title="Rincian tidak dapat diambil"
              description={messageOf(report.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        pagination={{
          page: report.data?.paginasi.halaman ?? 1,
          size: report.data?.paginasi.ukuran ?? 50,
          total: report.data?.paginasi.total ?? 0,
          totalPage: report.data?.paginasi.total_halaman ?? 0,
          onPageChange: setPage,
          isLoading: report.isFetching,
        }}
      />
    </div>
  )
}

/**
 * Pesan yang menempati kedua grid ketika tidak ada satu baris pun.
 *
 * # SATU pesan, bukan dua
 *
 * Ia sama sebelum maupun sesudah pencarian, karena Pega pun begitu:
 * `pyGridNoResultsMessage` adalah satu section tunggal yang digambar setiap kali page
 * list-nya kosong, tanpa membedakan sebabnya.
 *
 * # Kenapa sependek ini
 *
 * Versi sebelumnya memuat tiga kalimat ajakan beserta alasan kedua tanggal wajib. Di
 * dalam grid, teks sepanjang itu membentang selebar sepuluh kolom dan menjadi paragraf
 * yang justru tidak terbaca. Work Owner memintanya dipendekkan mengikuti Pega
 * (2026-10-03).
 *
 * Ajakannya tidak hilang, hanya pindah ke saat yang tepat: menekan "Cari Data" dengan
 * isian kosong tetap dijawab 422 beserta pesan per isian — "Tgl Input Dari belum diisi."
 * tepat di bawah isiannya. Itu tempat yang lebih berguna daripada paragraf yang dibaca
 * sebelum kesalahannya terjadi.
 */
const emptyMessage = 'Data Tidak Ada'

/**
 * Kolom grid rincian — kesepuluhnya, pada urutan yang tergambar di layar lama.
 *
 * Kelima kolom yang SELALU kosong tetap digambar. Menghapusnya akan membuat layar baru
 * berbeda dari layar yang dihafal pengguna, dan menyembunyikan bahwa kuerinya belum selesai
 * — dua hal yang keduanya tidak diinginkan.
 */
const detailColumns: Column<ReportRow>[] = [
  { key: 'no_klaim', title: 'No Klaim', value: (row) => row.no_klaim },
  { key: 'nama_object', title: 'Object Name', value: (row) => row.nama_object },
  { key: 'komite_status', title: 'Komite Status', value: (row) => row.komite_status },
  {
    key: 'tanggal_komite',
    title: 'Tanggal Komite',
    value: (row) => row.tanggal_komite ?? '',
    render: (row) => dateCell(row.tanggal_komite),
  },
  { key: 'ai_status', title: 'AI Status', value: (row) => row.ai_status },
  {
    key: 'tanggal_ai',
    title: 'Tanggal AI',
    value: (row) => row.tanggal_ai ?? '',
    render: (row) => dateCell(row.tanggal_ai),
  },
  { key: 'note_ai_terima', title: 'Note AI Terima', value: (row) => row.note_ai_terima },
  { key: 'note_ai_tolak', title: 'Note AI Tolak', value: (row) => row.note_ai_tolak },
  { key: 'coverage_final', title: 'Coverage Final', value: (row) => row.coverage_final },
  {
    key: 'kategori_kronologi',
    title: 'Kategori Kronologi',
    value: (row) => row.kategori_kronologi,
  },
]

/**
 * dateCell menggambar tanggal memakai pemformat baku aplikasi.
 *
 * # Kenapa TIDAK dd/mm/yyyy seperti berkas CSV-nya
 *
 * Layar lama menggambar kedua tanggal APA ADANYA — teks waktu Pega seperti
 * `20260903T000000.000 GMT`. Itu artefak layar yang belum selesai, bukan bentuk yang
 * dimaksudkan siapa pun; berkas CSV-nya justru menuliskan tanggal yang sudah disusun ulang.
 *
 * Yang dipakai di layar karena itu pemformat baku aplikasi (`components/format.ts`),
 * supaya tidak ada dua gaya tanggal dalam satu aplikasi. Berkas CSV tetap dd/mm/yyyy,
 * persis seperti yang ditulis activity lamanya.
 */
function dateCell(value: string | null) {
  if (!value) return <span className="text-slate-400">—</span>
  return formatDate(value)
}

function violationsOf(error: unknown): Record<string, string> {
  if (error instanceof APIError) return error.violations()
  return {}
}

function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
