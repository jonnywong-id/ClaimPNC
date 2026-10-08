import { useState } from 'react'

import { SelectField, type SelectOption } from '@/components/SelectField'
import { failureNotice, violationsOf } from '@/components/shared/queryFeedback'
import { PeriodFields, ReportActions } from '@/components/shared/ReportFilterParts'

import { useExportPICTeknik, usePICTeknik } from './api'
import type {
  MetadataResponse,
  PICComponent,
  PICFilterInput,
  PICRow,
  PICScorecard,
} from './types'

/**
 * Tab KPI PIC Teknik — penilaian kinerja PIC Teknik, empat komponen per orang.
 *
 * # Kenapa satu kartu per petugas, bukan satu tabel besar
 *
 * Layar lama menggambar satu tabel kecil per PIC, empat baris masing-masing. Bentuk itu
 * dipertahankan (`D-13`), dan alasannya masih berlaku: yang dibaca penyelia adalah
 * "bagaimana orang ini", bukan "bandingkan kolom Nilai seluruh orang". Satu tabel besar
 * berisi dua puluh petugas × empat baris memaksa mata mencari batas antar orang.
 *
 * # Hal yang WAJIB diketahui sebelum membaca angkanya
 *
 * Dua dari empat komponen — Update Status Progress dan SLA Klaim — tangganya MENURUN:
 * makin kecil persentasenya, makin TINGGI nilainya. Itu bukan kekeliruan di sini melainkan
 * perilaku Pega yang direplikasi (`P-5`), dan setiap baris semacam itu diberi penanda.
 *
 * Tanpa penanda, pembaca yang melihat "95% → nilai 1" akan melaporkannya sebagai kerusakan.
 */
export function ReportKPIPICTeknik({
  metadata,
}: Readonly<{
  metadata: MetadataResponse | undefined
}>) {
  const [draft, setDraft] = useState<PICFilterInput>(emptyPICFilter)
  const [applied, setApplied] = useState<PICFilterInput | null>(null)

  const active = applied ?? emptyPICFilter
  const searched = applied !== null

  const result = usePICTeknik(active, searched)
  const exportFile = useExportPICTeknik()

  const violations = violationsOf(result.error)
  const components = metadata?.komponen_pic ?? []

  // Isi bagian hasil menurut keadaan pencarian dan kueri.
  function renderResult() {
    if (!searched) {
      return (
        <p className="rounded-kotak border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
          Pilih lini bisnis dan periode, lalu tekan <strong>Cari</strong>. Layar lama pun
          menolak tanpa periode — pesannya berbunyi &quot;Periode tanggal masih kosong&quot;.
        </p>
      )
    }
    if (result.isError) {
      return failureNotice('Penilaian tidak dapat diambil', result.error)
    }
    if (result.isPending) {
      return (
        <p className="rounded-kotak border border-slate-200 bg-white px-4 py-6 text-sm text-slate-500">
          Menghitung penilaian…
        </p>
      )
    }
    return (
      <>
        {result.data.kartu_skor.length === 0 ? (
          <p className="rounded-kotak border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
            Tidak ada petugas terdaftar pada lini bisnis ini.
          </p>
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            {result.data.kartu_skor.map((card) => (
              <Scorecard key={card.pic} card={card} components={components} />
            ))}
          </div>
        )}

        {/*
          Rekapitulasi digambar TERPISAH di bawah, bukan sebagai kartu kesekian di dalam
          kisi. Ia bukan orang, dan menaruhnya berdampingan dengan kartu petugas membuat
          pembacanya mengira "Leader" adalah nama seseorang.
        */}
        <Scorecard
          card={result.data.rekapitulasi}
          components={components}
          summary
        />
      </>
    )
  }

  return (
    <div className="space-y-6">
      <form
        className="space-y-4 rounded-kotak border border-slate-200 bg-white p-4"
        onSubmit={(event) => {
          event.preventDefault()
          setApplied(draft)
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <SelectField
            id="kpi-pic-lini"
            label="Lini Bisnis"
            options={lineOptions(metadata)}
            emptyText="--Pilih--"
            value={draft.lini}
            error={violations['lini_bisnis']}
            onChange={(event) => setDraft({ ...draft, lini: event.target.value })}
          />

          <PeriodFields
            idPrefix="kpi-pic"
            fromLabel="Periode — Dari"
            toLabel="Periode — Sampai"
            value={draft}
            violations={violations}
            onChange={(patch) => setDraft({ ...draft, ...patch })}
          />
        </div>

        {lineNote(metadata, draft.lini) && (
          <p className="text-sm text-slate-600">{lineNote(metadata, draft.lini)}</p>
        )}

        <ReportActions
          searchLabel="Cari"
          exportLabel="Export Data"
          onExport={() => exportFile.mutate(active)}
          exportDisabled={!searched || exportFile.isPending}
        />
      </form>

      {exportFile.isError && failureNotice('Berkas tidak dapat diunduh', exportFile.error)}

      {renderResult()}

    </div>
  )
}

/* ─────────────────────────── Kartu skor ─────────────────────────── */

function Scorecard({
  card,
  components,
  summary,
}: Readonly<{
  card: PICScorecard
  components: PICComponent[]
  summary?: boolean
}>) {
  return (
    <section
      className={[
        'rounded-kotak border bg-white p-4',
        summary ? 'border-slate-400' : 'border-slate-200',
      ].join(' ')}
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">
            {summary ? 'Rekapitulasi seluruh PIC' : card.pic}
          </h3>
          {!summary && card.leader && (
            <p className="text-xs text-slate-500">Leader tim</p>
          )}
        </div>

        {card.nilai_berbobot !== null && (
          <span className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-700">
            Nilai berbobot {round2(card.nilai_berbobot)} dari 15
          </span>
        )}
      </header>

      <table className="w-full text-sm">
        <caption className="sr-only">
          Penilaian KPI {summary ? 'seluruh PIC' : card.pic}
        </caption>
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs uppercase text-slate-500">
            <th scope="col" className="py-2">
              KPI
            </th>
            <th scope="col" className="py-2 text-right">
              Total
            </th>
            <th scope="col" className="py-2 text-right">
              Tercapai
            </th>
            <th scope="col" className="py-2 text-right">
              Persentase
            </th>
            <th scope="col" className="py-2 text-right">
              Nilai
            </th>
          </tr>
        </thead>
        <tbody>
          {card.baris.map((row) => (
            <MetricRow
              key={row.komponen}
              row={row}
              descending={isDescending(components, row.komponen)}
            />
          ))}
        </tbody>
      </table>
    </section>
  )
}

/**
 * Satu baris penilaian.
 *
 * Baris bertangga menurun diberi penanda "↓" beserta penjelasannya pada `title`. Penanda itu
 * bukan hiasan: tanpa itu, baris yang menunjukkan 95% dengan nilai 1 terbaca sebagai
 * kerusakan, dan yang melaporkannya akan menghabiskan waktu menelusuri hal yang memang
 * disengaja.
 */
function MetricRow({ row, descending }: Readonly<{ row: PICRow; descending: boolean }>) {
  return (
    <tr className="border-b border-slate-100 last:border-0">
      <td className="py-2 text-slate-800">
        {row.judul}
        {descending && (
          <span
            className="ml-1 text-amber-700"
            title="Tangga nilainya MENURUN: makin kecil persentasenya, makin tinggi nilainya. Direplikasi dari Pega."
          >
            ↓
          </span>
        )}
      </td>
      <td className="py-2 text-right tabular-nums text-slate-700">{row.total}</td>
      <td className="py-2 text-right tabular-nums text-slate-700">{row.tercapai}</td>
      <td className="py-2 text-right tabular-nums text-slate-700">
        {row.persentase === null ? '—' : `${round2(row.persentase)}%`}
      </td>
      <td className="py-2 text-right tabular-nums font-medium text-slate-900">
        {row.nilai ?? '—'}
      </td>
    </tr>
  )
}

/* ─────────────────────────── Bantuan kecil ─────────────────────────── */

const emptyPICFilter: PICFilterInput = { lini: '', dari: '', sampai: '' }

/** lineOptions menyusun isi dropdown lini bisnis. */
function lineOptions(metadata: MetadataResponse | undefined): SelectOption[] {
  return (metadata?.lini_bisnis ?? []).map((line) => ({
    value: line.kode,
    label: line.judul,
  }))
}

/** lineNote mengambil keterangan penyaring lini yang sedang dipilih. */
function lineNote(
  metadata: MetadataResponse | undefined,
  code: string,
): string | undefined {
  return (metadata?.lini_bisnis ?? []).find((line) => line.kode === code)?.keterangan
}

/** isDescending menyatakan sebuah komponen bertangga menurun. */
function isDescending(components: PICComponent[], code: string): boolean {
  return components.find((component) => component.kode === code)?.tangga_menurun === true
}

/** round2 membulatkan untuk TAMPILAN saja — nilai simpanannya tidak disentuh. */
function round2(value: number): number {
  return Math.round(value * 100) / 100
}
