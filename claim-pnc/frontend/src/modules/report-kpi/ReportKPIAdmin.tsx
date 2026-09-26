import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField, type SelectOption } from '@/components/SelectField'

import { useAdminDetail, useAdminScorecard, useExportAdmin } from './api'
import type {
  AdminDetailRow,
  AdminFilterInput,
  Grid,
  MetadataResponse,
  Metric,
  ScorecardResponse,
} from './types'

/**
 * Tab **KPI Admin** — kinerja tim ADMIN REGISTRASI klaim.
 *
 * # Bentuknya berbeda dari tab Adjuster, dan itu mengikuti layar lama
 *
 * Tab Adjuster menggambar dua TABEL. Tab ini menggambar satu KARTU SKOR — satu baris hasil
 * hitungan dengan belasan metrik — lalu grid rincian klaim di bawahnya. Kuerinya pun
 * begitu: satu mengembalikan tepat satu baris, satu lagi mengembalikan banyak.
 *
 * # Dua kelompok, dan keduanya BUKAN penyaring atas bentuk yang sama
 *
 *   KLAIM NON MBU  membedakan LEADER dan MEMBER, mengukur satu tahap (registrasi)
 *   KLAIM PA       membedakan tahap REGISTRASI dan PEMBAYARAN, tanpa pembedaan orang
 *
 * Metrik dan kolomnya berbeda, dan yang menentukannya adalah keterangan dari server —
 * bukan percabangan di layar ini.
 */
export function ReportKPIAdmin({ metadata }: { metadata: MetadataResponse | undefined }) {
  const [draft, setDraft] = useState<AdminFilterInput>(emptyAdminFilter)
  const [applied, setApplied] = useState<AdminFilterInput | null>(null)
  const [page, setPage] = useState(1)

  const active = applied ?? emptyAdminFilter
  const searched = applied !== null

  const scorecard = useAdminScorecard(active, searched)
  const detail = useAdminDetail(active, page, searched)
  const exportFile = useExportAdmin()

  const violations = violationsOf(scorecard.error ?? detail.error)
  const grid = adminDetailGrid(metadata)

  const groupOptions: SelectOption[] = (metadata?.kelompok_admin ?? []).map((item) => ({
    value: item.kode,
    label: item.judul,
  }))

  const note = metadata?.kelompok_admin.find(
    (item) => item.kode === draft.kelompok,
  )?.keterangan

  return (
    <div className="space-y-6">
      <form
        className="space-y-4 rounded-kotak border border-slate-200 bg-white p-4"
        onSubmit={(event) => {
          event.preventDefault()
          setApplied(draft)
          setPage(1)
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <SelectField
            id="kpi-admin-kelompok"
            label="Pilih Data KPI"
            options={groupOptions}
            emptyText="--Pilih--"
            value={draft.kelompok}
            error={violations['kelompok']}
            onChange={(event) => setDraft({ ...draft, kelompok: event.target.value })}
          />

          <FormField
            id="kpi-admin-dari"
            label="Periode — Dari"
            type="date"
            value={draft.dari}
            failure={violations['dari']}
            onChange={(event) => setDraft({ ...draft, dari: event.target.value })}
          />

          <FormField
            id="kpi-admin-sampai"
            label="Periode — Sampai"
            type="date"
            value={draft.sampai}
            failure={violations['sampai']}
            onChange={(event) => setDraft({ ...draft, sampai: event.target.value })}
          />
        </div>

        {note && <p className="text-sm text-slate-600">{note}</p>}

        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" tone="utama">
            Cari
          </Button>
          <Button
            onClick={() => exportFile.mutate(active)}
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

      {!searched ? (
        <p className="rounded-kotak border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
          Pilih data KPI dan periode, lalu tekan <strong>Cari</strong>. Layar lama pun
          menolak tanpa periode — pesannya berbunyi &quot;Periode tanggal masih kosong&quot;.
        </p>
      ) : (
        <>
          {scorecard.isError ? (
            <ErrorMessage
              title="Kartu skor tidak dapat diambil"
              description={messageOf(scorecard.error)}
              tone="gangguan"
            />
          ) : (
            <Scorecard
              card={scorecard.data}
              loading={scorecard.isPending}
              coordinatorInQuery={metadata?.koordinator_di_kueri ?? ''}
            />
          )}

          <DataTable<AdminDetailRow>
            title="Rincian Klaim"
            label="Rincian klaim yang ditangani tim admin"
            columns={adminColumns(grid, active.kelompok)}
            rows={detail.data?.baris ?? []}
            rowKey={(row) => row.no_klaim}
            isLoading={detail.isPending}
            error={
              detail.isError ? (
                <ErrorMessage
                  title="Rincian tidak dapat diambil"
                  description={messageOf(detail.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage={adminEmptyMessage}
            hideSearch
            pagination={{
              page: detail.data?.paginasi.halaman ?? 1,
              size: detail.data?.paginasi.ukuran ?? 50,
              total: detail.data?.paginasi.total ?? 0,
              totalPage: detail.data?.paginasi.total_halaman ?? 0,
              onPageChange: setPage,
              isLoading: detail.isFetching,
            }}
          />
        </>
      )}
    </div>
  )
}

/* ─────────────────────────── Kartu skor ─────────────────────────── */

/**
 * Kartu skor — kepala identitas, lalu metrik berurutan, lalu kesimpulan.
 *
 * Ia digambar sebagai KARTU, bukan tabel satu baris. Tabel dengan empat belas kolom dan
 * satu baris memaksa pengguna menggulir menyamping untuk membaca satu penilaian — dan
 * kartu skor justru dibaca dari atas ke bawah.
 */
function Scorecard({
  card,
  loading,
  coordinatorInQuery,
}: {
  card: ScorecardResponse | undefined
  loading: boolean
  coordinatorInQuery: string
}) {
  if (loading && !card) {
    return (
      <div className="rounded-kotak border border-slate-200 bg-white p-4 text-sm text-slate-500">
        Menghitung kartu skor…
      </div>
    )
  }
  if (!card) return null

  return (
    <section className="space-y-4 rounded-kotak border border-slate-200 bg-white p-4">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            {card.identitas.kategori}
          </h2>
          <p className="text-sm text-slate-600">
            {card.identitas.nama_koordinator} · NIK {card.identitas.nik} ·{' '}
            {card.identitas.unit_kerja}
          </p>
          <p className="text-sm text-slate-500">
            Tanggal efektif: {card.tanggal_efektif}
          </p>
        </div>

        {card.achievement && (
          <span
            className={[
              'rounded-full px-3 py-1 text-sm font-medium',
              // Warna BUKAN satu-satunya pembeda: teksnya sendiri menyebut hasilnya.
              card.achievement === 'TERCAPAI TARGET'
                ? 'bg-blue-50 text-blue-800'
                : 'bg-amber-100 text-amber-900',
            ].join(' ')}
          >
            {card.achievement}
          </span>
        )}
      </header>

      {/*
        Keterangan nama koordinator. Ia digambar karena penguji yang membandingkan layar
        ini dengan teks kueri Pega akan menemukan nama yang BERBEDA — dan tanpa keterangan
        ini ia akan melaporkannya sebagai kekeliruan.
      */}
      {coordinatorInQuery !== '' &&
        coordinatorInQuery !== card.identitas.nama_koordinator && (
          <p className="text-xs text-slate-500">
            Catatan: teks kueri Pega menyebut nama koordinator{' '}
            <strong>{coordinatorInQuery}</strong>, tetapi activity-nya menimpanya dengan
            nama di atas. Yang ditampilkan adalah yang benar-benar dilihat pengguna di
            sistem lama.
          </p>
        )}

      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
        {card.metrik.map((metric) => (
          <div key={metric.kode} className="flex items-baseline justify-between gap-3">
            <dt className="text-sm text-slate-600">{metric.judul}</dt>
            <dd className="text-sm font-semibold tabular-nums text-slate-900">
              {metricText(metric)}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

/**
 * metricText menggambar satu angka menurut bentuknya.
 *
 * Bentuknya datang dari server, bukan disimpulkan dari nama metriknya: satu kartu memuat
 * cacah, persen, nilai 1–5, dan desimal sekaligus.
 */
function metricText(metric: Metric): string {
  if (metric.nilai === null) return '—'

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

const emptyAdminFilter: AdminFilterInput = { kelompok: '', dari: '', sampai: '' }

const adminEmptyMessage =
  'Tidak ada klaim pada penyaring ini. Periksa periodenya — yang disaring adalah tanggal ' +
  'registrasi klaim. Perhatikan pula bahwa hanya klaim yang dibuat oleh tim admin ' +
  'kelompok ini yang dihitung; daftar petugasnya tertulis di dalam kueri, bukan di master.'

/** adminDetailGrid mengambil keterangan grid rincian dari metadata. */
function adminDetailGrid(metadata: MetadataResponse | undefined): Grid | undefined {
  const tab = metadata?.tab.find((item) => item.kode === 'admin')
  return tab?.grid.find((item) => item.kode === 'rincian-klaim')
}

/**
 * adminColumns menyusun kolom grid rincian untuk kelompok yang sedang dipilih.
 *
 * Penyaring kelompoknya TIDAK boleh dilewati: judul "Tgl Terima Dokumen" ada di kedua
 * kelompok tetapi menunjuk kolom basis data yang berbeda, sehingga tanpa penyaring layar
 * akan menggambar judul itu dua kali — dan salah satunya berisi tanggal yang salah.
 */
function adminColumns(
  grid: Grid | undefined,
  group: string,
): Column<AdminDetailRow>[] {
  return (grid?.kolom ?? [])
    .filter((column) => !column.hanya_kelompok || column.hanya_kelompok === group)
    .map((column) => ({
      key: column.kunci,
      title: column.judul,
      value: (row) => adminCellText(row, column.kunci),
      alignRight: column.kunci.startsWith('aging_'),
    }))
}

/** adminCellText mengambil isi satu sel; yang kosong digambar sebagai tanda hubung. */
function adminCellText(row: AdminDetailRow, key: string): string {
  const value = row[key as keyof AdminDetailRow]

  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'number') return String(round2(value))
  return value
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
