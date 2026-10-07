import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'

import { useJalankan, useKeterangan } from './api'
import type { HasilPolis, InfoKoneksi, Laporan } from './types'

/**
 * Konversi Coverage — menu `MENU_ID 87`.
 *
 * Membaca dokumen coverage (kolom BLOB JSON) dari basis data LIVE untuk daftar nomor
 * polis, lalu menulis ke basis data TEST:
 *
 *	T_ANEKALIST.COVERAGELIST     → T_COVERAGELIST_ANEKA  + T_SPREADINGLIST
 *	T_CARGOLIST.COVERAGEDATA     → T_COVERAGELIST_CARGO  + T_SPREADINGLIST
 *	T_PROPERTYLIST.COVERAGELIST  → T_COVERAGELIST_FIRE   + T_SPREADINGLIST
 *	T_PERSONLIST.COVERAGEDATA    → T_COVERAGELIST_PERSON + T_SPREADINGLIST
 *
 * sekaligus menyalin kolom BLOB-nya ke baris objek yang sama di TEST.
 *
 * "Uji Coba" menjalankan seluruh tulisan di TEST lalu membatalkannya, sehingga angka di
 * laporan sama persis dengan yang akan terjadi — termasuk galat bentuk data.
 */
export function CoverageConversionPage() {
  const info = useKeterangan()
  const run = useJalankan()

  const [text, setText] = useState('')
  const [touched, setTouched] = useState(false)
  const [busy, setBusy] = useState<'uji' | 'jalan' | null>(null)
  const [report, setReport] = useState<Laporan | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Daftar bawaan dari KONVERSI_POLIS mengisi kotak sekali, selama pengguna belum mengetik.
  useEffect(() => {
    if (!touched && info.data && info.data.polis.length > 0) setText(info.data.polis.join('\n'))
  }, [info.data, touched])

  const policies = text.split(/[\s,;]+/).filter((p) => p.trim() !== '')
  const unique = new Set(policies).size
  const ready = info.data?.live.lengkap && info.data?.test.lengkap && !info.data?.galat

  async function execute(ujiCoba: boolean) {
    if (!ujiCoba) {
      const ok = window.confirm(
        `Tulis hasil konversi ${unique} polis ke basis data TEST?\n\n` +
          'Coverage dan spreading lama polis tersebut di TEST akan dihapus lalu diganti, ' +
          'dan kolom BLOB objeknya ditimpa dengan isi LIVE.',
      )
      if (!ok) return
    }
    setBusy(ujiCoba ? 'uji' : 'jalan')
    setError(null)
    try {
      setReport(await run(text, ujiCoba))
    } catch (e) {
      setReport(null)
      setError(e instanceof APIError ? e.message : 'Konversi gagal dijalankan.')
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-xl font-semibold text-slate-900">Konversi Coverage</h1>
        <p className="mt-1 text-sm text-slate-600">
          Salin dokumen coverage polis dari basis data LIVE menjadi tabel coverage dan spreading
          di basis data TEST.
        </p>
      </header>

      <section className="grid gap-4 md:grid-cols-2">
        <ConnectionCard title="LIVE (hanya dibaca)" info={info.data?.live} loading={info.isLoading} />
        <ConnectionCard title="TEST (ditulis)" info={info.data?.test} loading={info.isLoading} />
      </section>

      {info.data?.galat && (
        <ErrorMessage tone="penolakan" title="Konfigurasi ditolak" description={info.data.galat} />
      )}
      {info.isError && (
        <ErrorMessage
          tone="gangguan"
          title="Keterangan tidak dapat dimuat"
          description={info.error instanceof APIError ? info.error.message : 'Coba muat ulang halaman.'}
        />
      )}

      <section className="space-y-3 rounded-kontrol border border-slate-200 bg-white p-4">
        <TextAreaField
          id="daftar-polis"
          label="Daftar Nomor Polis"
          rows={8}
          value={text}
          onChange={(e) => {
            setTouched(true)
            setText(e.target.value)
          }}
          placeholder="Satu nomor polis per baris (boleh juga dipisah koma)"
          hint={`${unique} nomor polis unik · maksimal ${info.data?.maks_polis ?? 500} per kali jalan · seluruh versi (PRODKE) ikut dikonversi`}
        />
        <div className="flex flex-wrap gap-2">
          <Button tone="kedua" disabled={!ready || unique === 0 || busy !== null} onClick={() => execute(true)}>
            {busy === 'uji' ? 'Menjalankan uji coba…' : 'Uji Coba'}
          </Button>
          <Button tone="utama" disabled={!ready || unique === 0 || busy !== null} onClick={() => execute(false)}>
            {busy === 'jalan' ? 'Mengonversi…' : 'Jalankan Konversi'}
          </Button>
        </div>
      </section>

      {error && <ErrorMessage tone="gangguan" title="Konversi gagal" description={error} />}
      {report && <ReportView report={report} />}
    </div>
  )
}

function ConnectionCard({ title, info, loading }: { title: string; info: InfoKoneksi | undefined; loading: boolean }) {
  return (
    <div className="rounded-kontrol border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold text-slate-800">{title}</h2>
      {loading || !info ? (
        <p className="mt-1 text-sm text-slate-500">Memuat…</p>
      ) : info.lengkap ? (
        <p className="mt-1 break-all font-mono text-sm text-slate-700">{info.alamat}</p>
      ) : (
        <p className="mt-1 text-sm text-red-700">
          Belum dikonfigurasi. Isi di <code>backend/.env</code>: {info.kurang.join(', ')}
        </p>
      )}
    </div>
  )
}

const statusLabel: Record<HasilPolis['status'], { text: string; cls: string }> = {
  berhasil: { text: 'Berhasil', cls: 'bg-green-100 text-green-800' },
  tidak_ditemukan: { text: 'Tidak ditemukan', cls: 'bg-amber-100 text-amber-800' },
  gagal: { text: 'Gagal', cls: 'bg-red-100 text-red-800' },
}

function ReportView({ report }: { report: Laporan }) {
  return (
    <section className="space-y-3">
      <div className="rounded-kontrol border border-slate-200 bg-white p-4 text-sm">
        <p className="font-semibold text-slate-900">
          {report.uji_coba ? 'Hasil Uji Coba — tidak ada yang tersimpan' : 'Hasil Konversi — tersimpan di TEST'}
        </p>
        <p className="mt-1 text-slate-700">
          {report.berhasil} berhasil · {report.tidak_ditemukan} tidak ditemukan · {report.gagal} gagal ·{' '}
          {report.total_coverage} baris coverage · {report.total_spreading} baris spreading
        </p>
      </div>

      <div className="overflow-x-auto rounded-kontrol border border-slate-200 bg-white">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-slate-600">
            <tr>
              <th className="px-3 py-2">Nomor Polis</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Tabel Objek</th>
              <th className="px-3 py-2">PRODKE</th>
              <th className="px-3 py-2 text-right">Objek</th>
              <th className="px-3 py-2 text-right">BLOB Diperbarui</th>
              <th className="px-3 py-2 text-right">Coverage</th>
              <th className="px-3 py-2 text-right">Spreading</th>
              <th className="px-3 py-2">Keterangan</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {report.polis.map((p) => (
              <tr key={p.nomor_polis} className="align-top">
                <td className="px-3 py-2 font-mono">{p.nomor_polis}</td>
                <td className="px-3 py-2">
                  <span className={`rounded px-2 py-0.5 text-xs font-medium ${statusLabel[p.status].cls}`}>
                    {statusLabel[p.status].text}
                  </span>
                </td>
                <td className="px-3 py-2">{(p.lini ?? []).join(', ')}</td>
                <td className="px-3 py-2">{(p.versi_prodke ?? []).join(', ')}</td>
                <td className="px-3 py-2 text-right">{p.jumlah_objek}</td>
                <td className="px-3 py-2 text-right">{p.objek_diperbarui}</td>
                <td className="px-3 py-2 text-right">
                  {p.coverage_ditulis}
                  {p.coverage_dihapus > 0 && <span className="text-slate-500"> (ganti {p.coverage_dihapus})</span>}
                </td>
                <td className="px-3 py-2 text-right">
                  {p.spreading_ditulis}
                  {p.spreading_dihapus > 0 && <span className="text-slate-500"> (ganti {p.spreading_dihapus})</span>}
                </td>
                <td className="max-w-md px-3 py-2 text-xs">
                  {p.galat && <p className="text-red-700">{p.galat}</p>}
                  {p.dokumen_kosong > 0 && <p className="text-slate-600">{p.dokumen_kosong} objek tanpa dokumen coverage</p>}
                  {(p.objek_tidak_ada_di_test ?? []).length > 0 && (
                    <p className="text-amber-700">Tidak ada di TEST: {p.objek_tidak_ada_di_test!.join('; ')}</p>
                  )}
                  {(p.peringatan ?? []).map((w) => (
                    <p key={w} className="text-slate-600">
                      {w}
                    </p>
                  ))}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
