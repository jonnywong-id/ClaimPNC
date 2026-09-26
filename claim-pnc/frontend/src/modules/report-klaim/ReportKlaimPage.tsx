import { useMemo, useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useBusinessOptions, useExportReport, useReportCatalog } from './api'
import { EMPTY_FILTER, type ExportRequest, type Report, type ReportFilter } from './types'

/**
 * Report Klaim — menu `MENU_ID 85`, pengganti harness `PNCTATReport`.
 *
 * Judul yang dibaca pengguna di sistem lama adalah **"Report Claim"**, dan isinya bukan
 * satu laporan melainkan **28 panel** yang berdiri sendiri-sendiri. Setiap panel hanya
 * berisi judul dan tombol Export; tidak ada tabel hasil di layar sama sekali — yang
 * keluar adalah berkas CSV.
 *
 * # Susunan layar
 *
 * Penyaring bersama di atas, lalu kartu laporan — keputusan Work Owner 2026-09-24.
 *
 * Yang ditambahkan dari sistem lama hanya **pengelompokan kartu**. Harness lama menumpuk
 * ke-28 panel dalam satu kolom tanpa pengelompokan, sehingga mencari satu laporan berarti
 * membaca 28 judul berurutan. Pengelompokannya tidak mengubah satu pun laporan, satu pun
 * penyaring, dan satu pun kolom keluaran.
 *
 * # Label "Treaty" yang isinya bukan treaty
 *
 * Dropdown itu mengisi penyaring **lini bisnis**, bukan treaty. Labelnya tetap ditiru
 * (`D-13`, keputusan Work Owner 2026-09-24): pengguna sudah mengenalnya dengan nama itu
 * selama bertahun-tahun, dan memperbaikinya di layar berarti melatih ulang tanpa ada yang
 * meminta. Di dalam kode ia bernama menurut isinya.
 *
 * # Isian yang tidak berlaku DINONAKTIFKAN, bukan disembunyikan
 *
 * Setiap kartu menyebutkan penyaring mana yang benar-benar dipakai kuerinya. Isian yang
 * tidak berpengaruh dibuat pudar dan tidak dapat diisi ketika kartunya dipilih — bukan
 * dihilangkan, supaya letak isian tidak berpindah-pindah setiap kali kartu berganti.
 */
export function ReportKlaimPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [filter, setFilter] = useState<ReportFilter>(EMPTY_FILTER)
  const [dipilih, setDipilih] = useState<string | null>(null)

  const catalog = useReportCatalog()
  const exportReport = useExportReport()

  const laporanTerpilih = useMemo(() => {
    if (dipilih === null || catalog.data === undefined) return null
    for (const kelompok of catalog.data.kelompok) {
      const hit = kelompok.laporan.find((l) => l.kode === dipilih)
      if (hit !== undefined) return hit
    }
    return null
  }, [catalog.data, dipilih])

  // Daftar bisnis hanya ditarik ketika kartu yang membutuhkannya sedang dipilih. Ia
  // dapat berisi ratusan baris, dan 27 dari 28 panel tidak memakainya.
  const perluBisnis = laporanTerpilih?.penyaring.bisnis === true
  const businessOptions = useBusinessOptions(perluBisnis)

  // Penyaring yang AKTIF adalah milik kartu yang sedang dipilih. Sebelum satu kartu pun
  // dipilih, seluruh isian aktif — pengguna sering mengisi rentang tanggal lebih dulu,
  // baru memilih laporannya.
  const aktif: Report['penyaring'] = laporanTerpilih?.penyaring ?? {
    rentang_tanggal: true,
    lini_bisnis: true,
    status_compliance: true,
    bisnis: false,
    rincian: true,
  }

  function unduh(laporan: Report, aksi: string) {
    setDipilih(laporan.kode)
    const request: ExportRequest = {
      kode: laporan.kode,
      aksi,
      filter,
      penyaring: laporan.penyaring,
    }
    exportReport.mutate(request)
  }

  if (portal === null) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Berkas laporan memuat data klaim milik satu badan hukum, dan aplikasi ini ' +
            'melayani empat. Pilih portal di bilah atas untuk membuka layar ini.'
          }
          tone="penolakan"
        />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-slate-900">
          {catalog.data?.judul ?? 'Report Claim'}
        </h1>
        <p className="mt-1 text-sm text-slate-600">
          Pilih periode dan penyaringnya, lalu tekan Export pada laporan yang dituju.
          Berkasnya terunduh dalam bentuk CSV.
        </p>
      </header>

      <FilterBar
        filter={filter}
        setFilter={setFilter}
        aktif={aktif}
        liniBisnis={catalog.data?.lini_bisnis ?? []}
        bisnis={businessOptions.data?.bisnis ?? []}
        bisnisMemuat={perluBisnis && businessOptions.isPending}
      />

      {exportReport.isError && (
        <ErrorMessage
          title="Berkas tidak dapat diunduh"
          description={messageOf(exportReport.error)}
          tone={toneOf(exportReport.error)}
        />
      )}

      {catalog.isPending && <p className="text-sm text-slate-500">Memuat daftar laporan…</p>}

      {catalog.isError && (
        <ErrorMessage
          title="Daftar laporan tidak dapat dimuat"
          description={messageOf(catalog.error)}
          tone="gangguan"
        />
      )}

      {catalog.data?.kelompok.map((kelompok) => (
        <section key={kelompok.kode} className="space-y-3">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
            {kelompok.judul}
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {kelompok.laporan.map((laporan) => (
              <ReportCard
                key={laporan.kode}
                laporan={laporan}
                sedangDiunduh={exportReport.isPending && dipilih === laporan.kode}
                onExport={(aksi) => unduh(laporan, aksi)}
                onFocus={() => setDipilih(laporan.kode)}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

type FilterBarProps = {
  filter: ReportFilter
  setFilter: (next: ReportFilter) => void
  aktif: Report['penyaring']
  liniBisnis: { nilai: string; nama: string }[]
  bisnis: { kode: string; nama: string }[]
  bisnisMemuat: boolean
}

/**
 * Penyaring bersama di atas layar.
 *
 * Keempatnya diambil apa adanya dari harness: "Dari", "Sampai", "Treaty", dan
 * "Status Compliance" — ditambah "Bisnis" yang di sistem lama berada DI DALAM panel
 * Klaim Per Bisnis, satu-satunya panel yang memakainya.
 *
 * Memindahkannya ke atas bersama yang lain adalah penyesuaian bentuk, bukan perubahan
 * perilaku: ia tetap hanya berlaku pada panel itu, dan pada panel lain ia dinonaktifkan.
 */
function FilterBar({ filter, setFilter, aktif, liniBisnis, bisnis, bisnisMemuat }: FilterBarProps) {
  const ubah = (bagian: Partial<ReportFilter>) => setFilter({ ...filter, ...bagian })

  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Field
          id="dari"
          label="Dari"
          type="date"
          value={filter.dari}
          disabled={!aktif.rentang_tanggal}
          onChange={(e) => ubah({ dari: e.target.value })}
        />
        <Field
          id="sampai"
          label="Sampai"
          type="date"
          value={filter.sampai}
          disabled={!aktif.rentang_tanggal}
          onChange={(e) => ubah({ sampai: e.target.value })}
        />
        <SelectField
          id="lini"
          label="Treaty"
          value={filter.lini}
          disabled={!aktif.lini_bisnis}
          options={liniBisnis
            .filter((l) => l.nilai !== '')
            .map((l) => ({ value: l.nilai, label: l.nama }))}
          emptyText="----- Pilih -----"
          onChange={(e) => ubah({ lini: e.target.value })}
        />
        <SelectField
          id="bisnis"
          label="Bisnis"
          value={filter.bisnis}
          disabled={!aktif.bisnis || bisnisMemuat}
          options={bisnis.map((b) => ({ value: b.kode, label: b.nama }))}
          emptyText={bisnisMemuat ? 'Memuat…' : '----- Pilih -----'}
          onChange={(e) => ubah({ bisnis: e.target.value })}
        />
      </div>

      <div className="mt-4 flex flex-wrap items-end gap-6">
        <Field
          id="status_compliance"
          label="Status Compliance"
          value={filter.status_compliance}
          disabled={!aktif.status_compliance}
          hint="Daftar pilihannya tidak ada di export Pega; isikan kodenya bila diketahui."
          onChange={(e) => ubah({ status_compliance: e.target.value })}
          className="max-w-xs"
        />
        <label
          className={`flex items-center gap-2 pb-2 text-sm ${
            aktif.rincian ? 'text-slate-700' : 'text-slate-400'
          }`}
        >
          <input
            type="checkbox"
            checked={filter.rincian}
            disabled={!aktif.rincian}
            onChange={(e) => ubah({ rincian: e.target.checked })}
            className="h-4 w-4 rounded border-slate-300"
          />
          Tampilkan kolom rincian
        </label>
      </div>
    </div>
  )
}

type CardProps = {
  laporan: Report
  sedangDiunduh: boolean
  onExport: (aksi: string) => void
  onFocus: () => void
}

/**
 * Satu kartu laporan.
 *
 * Kartu yang TIDAK tersedia tetap digambar, bertanda sebabnya. Menghilangkannya membuat
 * pengguna melaporkan laporan yang "hilang", dan membuat kemajuan migrasi tidak terbaca
 * dari layar — perlakuan yang sama dengan butir menu yang belum punya layar.
 */
function ReportCard({ laporan, sedangDiunduh, onExport, onFocus }: CardProps) {
  const mati = !laporan.tersedia

  return (
    <article
      onMouseEnter={onFocus}
      onFocusCapture={onFocus}
      className={`flex flex-col justify-between rounded-kartu border p-4 shadow-lembut ${
        mati ? 'border-slate-200 bg-slate-50' : 'border-slate-200 bg-white'
      }`}
    >
      <div>
        <h3 className={`text-sm font-semibold ${mati ? 'text-slate-500' : 'text-slate-900'}`}>
          {laporan.judul}
        </h3>
        {mati && (
          <p className="mt-2 text-xs leading-relaxed text-slate-500">
            {laporan.alasan}
            {laporan.penghalang !== undefined && laporan.penghalang !== '' && (
              <span className="ml-1 rounded bg-slate-200 px-1.5 py-0.5 font-medium text-slate-600">
                {laporan.penghalang}
              </span>
            )}
          </p>
        )}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {mati ? (
          <span className="text-xs font-medium uppercase tracking-wide text-slate-400">
            Belum tersedia
          </span>
        ) : (
          laporan.tombol.map((tombol) => (
            <Button
              key={tombol.kode === '' ? laporan.kode : tombol.kode}
              tone={laporan.tombol.length > 1 && tombol.kode === 'rejected' ? 'kedua' : 'utama'}
              disabled={sedangDiunduh}
              onClick={() => onExport(tombol.kode)}
            >
              {sedangDiunduh ? 'Menyiapkan…' : tombol.label}
            </Button>
          ))
        )}
      </div>
    </article>
  )
}

function messageOf(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * Penolakan dan gangguan dibedakan nadanya.
 *
 * Isian yang belum benar dapat diperbaiki pengguna; kegagalan sistem tidak. Menyamakan
 * tampilannya membuat pengguna mencoba berulang kali pada hal yang tidak akan berubah.
 */
function toneOf(failure: unknown): 'penolakan' | 'gangguan' {
  if (failure instanceof APIError && failure.status < 500) return 'penolakan'
  return 'gangguan'
}
