import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'

import { useAdminScorecard, useExportAdmin } from './api'
import type { Grid, MetadataResponse, Metric, ScorecardResponse } from './types'

/**
 * Tab **KPI Admin** — kinerja tim ADMIN REGISTRASI klaim.
 *
 * # Satu grid, bernama "Data KPI", berkolom 21
 *
 * Koreksi 2026-10-09. Tab ini sebelumnya menggambar KARTU SKOR beserta grid "Rincian Klaim"
 * di bawahnya. Keduanya salah terhadap layar lama:
 *
 *   - Pega menggambar **satu tabel datar satu baris** berkolom 21, bukan kartu. Susunannya
 *     dibaca dari urutan `pyValue` grid pada `Section/ReportKPI_Section-Section.xml`, dan
 *     cocok satu-ke-satu dengan urutan SELECT `RDB List/GetDataKPIAdmin-SQL.xml`.
 *   - Grid rincian memang ada di section itu, tetapi ia milik blok PA yang TIDAK
 *     ditampilkan, dan kolomnya persis kolom berkas **Export Detail Data**. Jadi ia layout
 *     ekspor, bukan grid layar.
 *
 * Rinciannya tidak hilang: tombol Export Detail Data tetap mengunduhnya, dan kuerinya tidak
 * disentuh sama sekali.
 *
 * # SATU bilah penyaring
 *
 * `ReportKPI_Section` memuat DUA bilah di dalam layout `KPI Admin`: `Dari`+`Sampai` dan
 * `Periode KPI`, masing-masing dengan `Cari` dan `Export Detail Data` sendiri. Keduanya
 * bertanda `pyVisible: ALWAYS`, dan yang kedua berada di dalam wadah ber-`pyLoadDeferred`.
 *
 * **Work Owner memeriksa Pega yang berjalan pada 2026-10-09 dan bilah kedua TIDAK ADA.**
 * Wadah tertunda itu tidak pernah selesai dimuat; definisinya ada, tampilannya tidak — dan
 * yang ditiru adalah layar yang dipakai orang (`D-13`).
 *
 * Kueri, dropdown periode, dan kartu skor PA di sisi peladen **tetap ada** dan teruji; yang
 * hilang hanya jalan masuknya dari layar.
 */
export function ReportKPIAdmin({ metadata }: { metadata: MetadataResponse | undefined }) {
  const [draft, setDraft] = useState<AdminPeriod>(emptyAdminPeriod)
  const [applied, setApplied] = useState<AdminPeriod | null>(null)

  const filter = { ...(applied ?? emptyAdminPeriod), kelompok: GROUP_NONMBU }
  const searched = applied !== null

  const scorecard = useAdminScorecard(filter, searched)
  const exportFile = useExportAdmin()

  const violations = violationsOf(scorecard.error)
  const grid = adminScorecardGrid(metadata)

  // Judul blok dipakai sebagai `aria-label` SAJA, tidak digambar — seluruh bagian Admin
  // pada section Pega hanya memuat label isiannya, tanpa judul blok dan tanpa keterangan.
  const title = metadata?.kelompok_admin.find((item) => item.kode === GROUP_NONMBU)?.judul
  const rows = scorecard.data ? [scorecard.data] : []

  return (
    <section className="space-y-4" aria-label={title ?? GROUP_NONMBU}>
      <form
        className="rounded-kotak border border-slate-200 bg-white p-4"
        onSubmit={(event) => {
          event.preventDefault()
          setApplied(draft)
        }}
      >
        {/* SATU baris: isian, Cari, lalu Export Detail Data — seperti Pega. */}
        <div className="flex flex-wrap items-end gap-3">
          <FormField
            id="kpi-admin-dari"
            label="Dari"
            type="date"
            className="w-44"
            value={draft.dari}
            failure={violations['dari']}
            onChange={(event) => setDraft({ ...draft, dari: event.target.value })}
          />
          <FormField
            id="kpi-admin-sampai"
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
            Berkasnya mengikuti penyaring yang SEDANG ditampilkan, sehingga tombolnya
            dimatikan sebelum Cari ditekan — belum ada yang ditampilkan.
          */}
          <Button
            onClick={() => exportFile.mutate(filter)}
            disabled={!searched || exportFile.isPending}
          >
            Export Detail Data
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

      {searched && (
        <DataTable<ScorecardResponse>
          title={grid?.judul ?? 'Data KPI'}
          label="Penilaian KPI tim admin registrasi"
          columns={adminColumns(grid, scorecard.data)}
          rows={rows}
          rowKey={(row) => row.identitas.kategori}
          isLoading={scorecard.isPending}
          error={
            scorecard.isError ? (
              <ErrorMessage
                title="Data KPI tidak dapat diambil"
                description={messageOf(scorecard.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage={adminEmptyMessage}
          hideSearch
        />
      )}
    </section>
  )
}

/* ─────────────────────────── Penyusun kolom ─────────────────────────── */

/**
 * adminColumns menyusun ke-21 kolom: identitas, lalu metrik, lalu NOTE.
 *
 * Kolom metriknya datang dari JAWABAN, bukan dari metadata — metriknya berbeda antar
 * kelompok dan sebagiannya baru diketahui saat permintaan dijawab. Judul dan urutannya
 * disusun `BuildScorecard` di sisi peladen, sehingga layar tidak dapat menyusunnya berbeda
 * dari berkas ekspor.
 */
function adminColumns(
  grid: Grid | undefined,
  card: ScorecardResponse | undefined,
): Column<ScorecardResponse>[] {
  // `render` dipasang supaya sel kosong tergambar KOSONG, bukan sebagai tanda hubung.
  //
  // Yang terkena hari ini adalah kolom NOTE: rasio yang tidak dapat dihitung menghasilkan
  // kesimpulan kosong, dan Pega menggambarnya sebagai sel kosong — bukan sebagai tanda yang
  // terbaca seolah nilainya hilang.
  const identity: Column<ScorecardResponse>[] = (grid?.kolom ?? []).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => adminIdentityCell(row, column.kunci),
    render: (row) => adminIdentityCell(row, column.kunci),
  }))

  const metrics: Column<ScorecardResponse>[] = (card?.metrik ?? []).map((metric) => ({
    key: metric.kode,
    title: metric.judul,
    value: (row) => metricText(row.metrik.find((m) => m.kode === metric.kode)),

    // Seluruh kolom metrik berisi angka; dirapatkan ke kanan supaya desimalnya sejajar.
    alignRight: true,
  }))

  const trailing: Column<ScorecardResponse>[] = (grid?.kolom_akhir ?? []).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => adminIdentityCell(row, column.kunci),
    render: (row) => adminIdentityCell(row, column.kunci),
  }))

  return [...identity, ...metrics, ...trailing]
}

/**
 * adminIdentityCell mengambil isi satu sel identitas.
 *
 * Perhatikan pasangan yang bersilangan: kolom **UNIT KERJA** diisi `identitas.kategori`,
 * sedangkan kolom **BUSINESS** diisi `identitas.unit_kerja`. Penamaan internal kami memang
 * tidak sejalan dengan judul layar lama, dan urutan SELECT kueri Pega yang memutuskannya.
 */
function adminIdentityCell(card: ScorecardResponse, key: string): string {
  switch (key) {
    case 'unit_kerja':
      return card.identitas.kategori
    case 'nama_koordinator':
      return card.identitas.nama_koordinator
    case 'business':
      return card.identitas.unit_kerja
    case 'nik':
      return card.identitas.nik
    case 'tanggal_efektif':
      return card.tanggal_efektif
    case 'note':
      return card.achievement ?? ''
    default:
      return ''
  }
}

/**
 * metricText menggambar satu angka menurut bentuknya.
 *
 * Bentuknya datang dari server, bukan disimpulkan dari nama metriknya: satu baris memuat
 * cacah, persen, nilai 1–5, dan desimal sekaligus.
 */
function metricText(metric: Metric | undefined): string {
  if (!metric || metric.nilai === null) return '—'

  switch (metric.bentuk) {
    case 'cacah':
      return String(metric.nilai)
    case 'persen':
      return `${round2(metric.nilai)}%`
    default:
      return String(round2(metric.nilai))
  }
}

/** round2 membulatkan ke dua desimal tanpa menambah nol di belakang. */
function round2(value: number): number {
  return Math.round(value * 100) / 100
}

/* ─────────────────────────── Bantuan kecil ─────────────────────────── */

/**
 * AdminPeriod adalah penyaring layar: HANYA periode.
 *
 * Kelompoknya bukan isian pengguna — hanya NON MBU yang digambar, dan ia menambahkan
 * kodenya sendiri saat memanggil peladen.
 */
export type AdminPeriod = { dari: string; sampai: string }

const emptyAdminPeriod: AdminPeriod = { dari: '', sampai: '' }

/**
 * Kode kelompok admin, apa adanya seperti yang diterima peladen.
 *
 * Kelompok `PA` tetap dikenal peladen dan kuerinya teruji, tetapi bilah penyaringnya
 * dicabut karena Pega tidak menampilkannya — lihat catatan di kepala berkas.
 */
const GROUP_NONMBU = 'NONMBU'

const adminEmptyMessage =
  'Tidak ada klaim pada penyaring ini. Periksa periodenya — yang disaring adalah tanggal ' +
  'registrasi klaim.'

/** adminScorecardGrid mengambil keterangan grid "Data KPI" dari metadata. */
function adminScorecardGrid(metadata: MetadataResponse | undefined): Grid | undefined {
  const tab = metadata?.tab.find((item) => item.kode === 'admin')
  return tab?.grid.find((item) => item.kode === 'kartu-skor')
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
