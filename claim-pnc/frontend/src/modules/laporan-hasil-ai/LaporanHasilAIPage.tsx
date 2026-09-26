import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { formatDate } from '@/components/format'

import { useExportLaporanHasilAI, useLaporanHasilAI } from './api'
import { emptyFilter, isComplete, type FilterInput, type ReportRow, type ReportTally } from './types'

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
 * tombol "Cari Data" dan "Export To Excel", lalu DUA grid — ringkasan pencacah di atas,
 * rincian baris di bawah. Judul kolom TIDAK diterjemahkan: `D-13` menetapkan tampilan
 * meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Empat hal yang paling mudah disalahpahami di layar ini
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
 *
 * KEEMPAT — **kolom "Total" pada ringkasan BUKAN jumlah baris**. Ia `Diterima + Ditolak`;
 * yang menunggu tidak ikut. Kolom "Menunggu" ditambahkan supaya selisihnya terbaca — ia
 * satu-satunya hal di layar ini yang tidak ada di Pega.
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

  const summary = report.data?.ringkasan ?? []
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

      <form
        className="space-y-4 rounded-kotak border border-slate-200 bg-white p-4"
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

      {!searched ? (
        <p className="rounded-kotak border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
          Isi <strong>Tgl Input Dari</strong> dan <strong>Tgl Input Sampai</strong>, lalu
          tekan <strong>Cari Data</strong>. Layar lama pun menolak tanpa keduanya —
          penyaringnya disusun dari kedua isian itu, dan yang kosong membuat kuerinya gagal.
        </p>
      ) : (
        <>
          {/*
            Grid ringkasan digambar DataTable yang sama dengan grid rincian, bukan tabel
            mentah: tidak ada `<table>` di folder modules/ (aturan susunan nomor 5).

            Ia tanpa pencarian dan tanpa paginasi — isinya tepat dua baris, dan kotak
            pencarian di atas dua baris hanya menambah sesuatu yang tidak menjawab apa pun.
          */}
          <DataTable<ReportTally>
            title="Ringkasan"
            label="Pencacah keputusan AI dan keputusan komite"
            columns={summaryColumns}
            rows={summary}
            rowKey={(row) => row.keputusan}
            isLoading={report.isPending}
            hideSearch
            emptyMessage="Belum ada yang dapat diringkas pada rentang ini."
            error={
              report.isError ? (
                <ErrorMessage
                  title="Ringkasan tidak dapat diambil"
                  description={messageOf(report.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
          />

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
            isLoading={report.isPending}
            hideSearch
            emptyMessage="Tidak ada data pada rentang tanggal ini."
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
        </>
      )}
    </div>
  )
}

/**
 * Kolom grid ringkasan.
 *
 * Urutannya mengikuti layar lama — Keputusan, Total, Diterima, Ditolak — dengan "Menunggu"
 * di paling kanan. Ia ditaruh di akhir dengan sengaja: menyisipkannya di tengah akan
 * menggeser kolom yang sudah dihafal pembacanya.
 */
const summaryColumns: Column<ReportTally>[] = [
  { key: 'keputusan', title: 'Keputusan', value: (row) => row.keputusan },
  { key: 'total', title: 'Total', value: (row) => String(row.total), alignRight: true },
  { key: 'diterima', title: 'Diterima', value: (row) => String(row.diterima), alignRight: true },
  { key: 'ditolak', title: 'Ditolak', value: (row) => String(row.ditolak), alignRight: true },
  {
    key: 'menunggu',
    title: 'Menunggu',
    value: (row) => String(row.menunggu),
    alignRight: true,
  },
]

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
