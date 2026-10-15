import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField } from '@/components/SelectField'

import { useExportPICLaporan, useExportPICTeknik, usePICTeknik } from './api'
import type {
  Grid,
  MetadataResponse,
  PICComponent,
  PICRow,
  PICScorecard,
} from './types'

/**
 * Tab KPI PIC Teknik — penilaian kinerja PIC Teknik, empat komponen per orang.
 *
 * # Satu TABEL datar, bukan kartu per petugas
 *
 * Koreksi 2026-10-09. Sebelumnya tab ini menggambar satu kartu per PIC, masing-masing berisi
 * tabel kecil empat baris, dengan alasan "yang dibaca penyelia adalah bagaimana orang ini".
 * Alasan itu masuk akal, tetapi bukan alasan yang berlaku: layar Pega menggambar SATU grid
 * datar berjudul **Data KPI PIC Teknik**, satu baris per pasangan PIC × komponen, berkolom
 * `PIC · KATEGORI · TOTAL DATA · JUMLAH TERCAPAI · TERCAPAI (%) · NILAI`.
 *
 * Susunan kolomnya dibaca dari `Section/ReportKPI_Section-Section.xml` — urutan `pyValue`
 * pada grid itu — bukan dari tangkapan layar, dan `D-13` menetapkan tampilan mengikuti Pega.
 *
 * # Hal yang WAJIB diketahui sebelum membaca angkanya
 *
 * Dua dari empat komponen — Update Status Progress dan SLA Klaim — tangganya MENURUN:
 * makin kecil persentasenya, makin TINGGI nilainya. Itu bukan kekeliruan di sini melainkan
 * perilaku Pega yang direplikasi (`P-5`), dan setiap baris semacam itu diberi penanda pada
 * `title` selnya.
 *
 * # Satu selisih yang BELUM dapat dipastikan
 *
 * Pada Pega, sebagian PIC tampil dengan dua atau tiga baris saja — bukan empat. Dugaan
 * terkuatnya: kueri komponennya ber-`GROUP BY pic`, sehingga PIC tanpa data sama sekali
 * tidak dikembalikan dan barisnya tidak terbentuk. Dugaan itu TIDAK dapat dibuktikan dari
 * export — badan `GetProgressPerPIC-Act.xml` tidak terbaca di sana — sehingga aturannya
 * tidak dibuat-buat di sini: keempat baris tetap digambar, yang tanpa data bertotal 0.
 */
export function ReportKPIPICTeknik({
  metadata,
}: {
  metadata: MetadataResponse | undefined
}) {
  const [draft, setDraft] = useState<PICFilterInput>(emptyPICFilter)
  const [applied, setApplied] = useState<PICFilterInput | null>(null)

  const active = applied ?? emptyPICFilter
  const searched = applied !== null

  const result = usePICTeknik(active, searched)
  const exportFile = useExportPICTeknik()
  const exportLaporan = useExportPICLaporan()

  const violations = violationsOf(result.error)
  const components = metadata?.komponen_pic ?? []
  const grid = picGrid(metadata)

  const rows = searched && result.data ? flatten(result.data.kartu_skor, result.data.rekapitulasi) : []

  return (
    <div className="space-y-4">
      <form
        className="rounded-kotak border border-slate-200 bg-white p-4"
        onSubmit={(event) => {
          event.preventDefault()
          setApplied(draft)
        }}
      >
        {/*
          SATU baris: Dari, Sampai, Cari, Export Data KPI, Pilih Data KPI, Export Data KPI
          — seperti bilah
          penyaring Pega. Sebelumnya isiannya digambar sebagai kisi dua kolom dengan tombol
          di baris tersendiri, sehingga bilahnya dua kali lebih tinggi daripada layar lama.
        */}
        <div className="flex flex-wrap items-end gap-3">
          <FormField
            id="kpi-pic-dari"
            label="Dari"
            type="date"
            className="w-44"
            value={draft.dari}
            failure={violations['dari']}
            onChange={(event) => setDraft({ ...draft, dari: event.target.value })}
          />

          <FormField
            id="kpi-pic-sampai"
            label="Sampai"
            type="date"
            className="w-44"
            value={draft.sampai}
            failure={violations['sampai']}
            onChange={(event) => setDraft({ ...draft, sampai: event.target.value })}
          />

          <Button type="submit" tone="utama">
            Cari
          </Button>

          {/*
            Tombol ekspor PERTAMA — mengunduh penilaian yang sedang terlihat.

            Pega punya DUA tombol berlabel sama di tab ini, dan sampai 2026-10-09 kami
            hanya membangun yang kedua. Yang membedakan keduanya adalah letaknya: yang
            ini menempel pada Cari, yang satunya pada Pilih Data KPI.
          */}
          <Button
            onClick={() => exportLaporan.mutate(active)}
            disabled={!searched || exportLaporan.isPending}
          >
            Export Data KPI
          </Button>

          {/*
            Dropdown ini memilih BERKAS yang diunduh, bukan isi layar — karena itu ia
            berdiri di sebelah tombol Export, bukan di antara penyaring. Begitu pula
            letaknya di layar Pega.
          */}
          <SelectField
            id="kpi-pic-data"
            label="Pilih Data KPI"
            options={(metadata?.data_kpi ?? []).map((item) => ({
              value: item.kode,
              label: item.judul,
            }))}
            value={draft.dataKPI}
            onChange={(event) => setDraft({ ...draft, dataKPI: event.target.value })}
            className="max-w-xs"
          />

          <Button
            onClick={() => exportFile.mutate(active)}
            disabled={!searched || exportFile.isPending}
          >
            Export Data KPI
          </Button>
        </div>
      </form>

      {(exportFile.isError || exportLaporan.isError) && (
        <ErrorMessage
          title="Berkas tidak dapat diunduh"
          description={messageOf(exportFile.error ?? exportLaporan.error)}
          tone="gangguan"
        />
      )}

      {/*
        Sebelum Cari ditekan TIDAK ada apa pun di bawah bilah penyaring, seperti layar lama.
        Ajakan "Pilih lini bisnis dan periode, lalu tekan Cari" sempat digambar di sini; itu
        tambahan kami, dan bagian PIC Teknik pada section Pega tidak memuat satu pun teks
        selain label isiannya.
      */}
      {searched && (
        <DataTable<FlatPICRow>
          title={grid?.judul ?? 'Data KPI PIC Teknik'}
          label="Penilaian KPI per PIC Teknik"
          columns={picColumns(grid, components)}
          rows={rows}
          rowKey={(row) => `${row.pic}|${row.komponen}`}
          isLoading={result.isPending}
          error={
            result.isError ? (
              <ErrorMessage
                title="Penilaian tidak dapat diambil"
                description={messageOf(result.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage={emptyMessage}
          hideSearch
        />
      )}
    </div>
  )
}

/* ─────────────────────────── Perataan baris ─────────────────────────── */

/**
 * FlatPICRow adalah satu baris grid: satu PIC × satu komponen.
 *
 * Ia membawa `leader` supaya baris rekapitulasi dapat dibedakan tanpa membandingkan nama —
 * nama "Leader" kebetulan juga dapat menjadi nama orang.
 */
type FlatPICRow = PICRow & { pic: string; leader: boolean }

/**
 * flatten meratakan kartu skor menjadi baris grid.
 *
 * Baris rekapitulasi ikut diratakan ke tabel yang sama, bukan digambar terpisah: ia memang
 * satu baris grid di Pega, dengan kolom PIC berisi "Leader". Ia dihitung
 * `PNCReportKPI_act` — bukan tambahan kami — sehingga menghilangkannya berarti kehilangan
 * angka yang ada di layar lama.
 */
function flatten(cards: PICScorecard[], recap: PICScorecard): FlatPICRow[] {
  // Tanpa satu pun petugas, rekapitulasinya TIDAK digambar: ia rata-rata dari nol orang,
  // dan menggambarnya akan menampilkan baris "Leader" berisi angka yang tidak merangkum
  // apa pun — sekaligus menyembunyikan pesan "tidak ada petugas terdaftar".
  const semua = cards.length === 0 ? [] : [...cards, recap]

  const rows: FlatPICRow[] = []
  for (const card of semua) {
    for (const row of card.baris) {
      rows.push({ ...row, pic: card.pic, leader: card.leader })
    }
  }
  return rows
}

/* ─────────────────────────── Penyusun kolom ─────────────────────────── */

/**
 * picColumns menyusun keenam kolom grid dari metadata.
 *
 * Judulnya datang dari server, bukan ditulis di sini: judul yang sama dipakai berkas ekspor,
 * dan dua daftar yang kebetulan sejalan akan berselisih pada perubahan berikutnya.
 */
function picColumns(
  grid: Grid | undefined,
  components: PICComponent[],
): Column<FlatPICRow>[] {
  return (grid?.kolom ?? []).map((column) => {
    const base: Column<FlatPICRow> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => picCellText(row, column.kunci),

      // Keempat kolom angka dirapatkan ke kanan supaya desimalnya sejajar antar baris.
      alignRight: column.kunci !== 'pic' && column.kunci !== 'kpi',
    }

    // `render` DIHILANGKAN, bukan diisi undefined: `exactOptionalPropertyTypes` menolak
    // properti opsional yang ada tetapi bernilai undefined.
    if (column.kunci !== 'kpi') return base
    return { ...base, render: (row: FlatPICRow) => metricLabel(row, components) }
  })
}

/**
 * metricLabel menggambar sel KATEGORI beserta penanda tangga menurun.
 *
 * Penanda itu bukan hiasan: tanpa itu, baris yang menunjukkan 95% dengan nilai 1 terbaca
 * sebagai kerusakan, dan yang melaporkannya akan menghabiskan waktu menelusuri hal yang
 * memang disengaja.
 */
function metricLabel(row: FlatPICRow, components: PICComponent[]) {
  const descending = components.find((c) => c.kode === row.komponen)?.tangga_menurun === true
  if (!descending) return row.judul

  return (
    <>
      {row.judul}
      <span
        className="ml-1 text-amber-700"
        title="Tangga nilainya MENURUN: makin kecil persentasenya, makin tinggi nilainya. Direplikasi dari Pega."
      >
        ↓
      </span>
    </>
  )
}

/** picCellText mengambil isi satu sel; yang kosong digambar sebagai tanda hubung. */
function picCellText(row: FlatPICRow, key: string): string {
  switch (key) {
    case 'pic':
      return row.pic
    case 'kpi':
      return row.judul
    case 'total':
      return String(row.total)
    case 'tercapai':
      return String(row.tercapai)
    case 'persentase':
      return row.persentase === null ? '—' : String(round2(row.persentase))
    case 'nilai':
      return row.nilai === null ? '—' : String(row.nilai)
    default:
      return ''
  }
}

/* ─────────────────────────── Bantuan kecil ─────────────────────────── */

/** Isian penyaring tab KPI PIC Teknik sebagaimana dipegang layar. */
type PICFilterInput = {
  lini: string
  dari: string
  sampai: string
  dataKPI: string
}

/**
 * Penyaring awal tab KPI PIC Teknik.
 *
 * # Kenapa lininya TETAP, bukan dipilih pengguna
 *
 * Layar Pega tidak punya pilihan lini bisnis di tab ini, dan itu bukan kelalaian:
 * `Activity/PNCReportKPI_act-Act.xml` menyetel `NONMBU` sebagai bawaan, lalu punya tiga
 * cabang — PA, TRAVEL, BONDING — yang menyala bila `TempLaporan.StatusReceiver` bernilai
 * `002`, `005`, atau `003`.
 *
 * Properti itu TIDAK terikat satu kontrol pun di `ReportKPIHarness` maupun
 * `ReportKPI_Section`, sehingga ia tidak pernah terisi dan ketiga cabang tidak pernah
 * menyala. Dikuatkan `RDB List/GetDataPICGroup-SQL.xml`, yang mematok
 * `type_business='NONMBU'` di dalam teks kuerinya.
 *
 * Jadi tab ini **selalu NONMBU di Pega**, dan dropdown yang sempat ada di sini adalah
 * tambahan kami — dicabut atas keputusan Work Owner "ikuti Pega as-is" (2026-10-08).
 */
const emptyPICFilter: PICFilterInput = { lini: 'NONMBU', dari: '', sampai: '', dataKPI: '1' }

const emptyMessage =
  'Tidak ada petugas terdaftar pada lini bisnis ini. Daftarnya datang dari master ' +
  'MST_USER_TEKNIK, bukan dari data klaim.'

/** picGrid mengambil keterangan grid tab KPI PIC Teknik dari metadata. */
function picGrid(metadata: MetadataResponse | undefined): Grid | undefined {
  const tab = metadata?.tab.find((item) => item.kode === 'pic-teknik')
  return tab?.grid.find((item) => item.kode === 'kartu-skor-pic')
}

/** round2 membulatkan untuk TAMPILAN saja — nilai simpanannya tidak disentuh. */
function round2(value: number): number {
  return Math.round(value * 100) / 100
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
