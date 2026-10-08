import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'

import { useInfo, useRun } from './api'
import type { ConnectionInfo, PolicyResult, Report } from './types'

/**
 * Konversi Coins — menu `MENU_ID 89` (`KonversiCoins`).
 *
 * Membaca daftar koasuransi polis (larik `CoinsList` di dokumen JSON_POLIS.DATA_JSONBLOB)
 * dari basis data LIVE untuk daftar nomor polis, lalu menulis ke basis data TEST:
 *
 *	JSON_POLIS.DATA_JSONBLOB $.CoinsList → T_COINSLIST (satu baris per anggota koasuransi)
 *
 * Setiap versi polis (PRODKE) ikut dikonversi. "Uji Coba" menjalankan seluruh tulisan di TEST
 * lalu membatalkannya, sehingga angka di laporan sama persis dengan yang akan terjadi.
 */
export function CoinsConversionPage() {
  const info = useInfo()
  const run = useRun()

  const [text, setText] = useState('')
  const [touched, setTouched] = useState(false)
  const [busy, setBusy] = useState<'test' | 'run' | null>(null)
  const [report, setReport] = useState<Report | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Daftar bawaan dari KONVERSI_POLIS mengisi kotak sekali, selama pengguna belum mengetik.
  useEffect(() => {
    if (!touched && info.data && info.data.polis.length > 0) setText(info.data.polis.join('\n'))
  }, [info.data, touched])

  const policies = text.split(/[\s,;]+/).filter((p) => p.trim() !== '')
  const unique = new Set(policies).size
  const ready = info.data?.live.lengkap && info.data?.test.lengkap && !info.data?.galat

  async function execute(dryRun: boolean) {
    if (!dryRun) {
      const ok = window.confirm(
        `Tulis hasil konversi ${unique} polis ke basis data TEST?\n\n` +
          'Baris koasuransi lama polis tersebut di T_COINSLIST TEST akan dihapus lalu diganti ' +
          'dengan isi CoinsList dari LIVE, untuk setiap versi (PRODKE).',
      )
      if (!ok) return
    }
    setBusy(dryRun ? 'test' : 'run')
    setError(null)
    try {
      setReport(await run(text, dryRun))
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
        <h1 className="text-xl font-semibold text-slate-900">Konversi Coins</h1>
        <p className="mt-1 text-sm text-slate-600">
          Salin daftar koasuransi polis (CoinsList pada JSON_POLIS.DATA_JSONBLOB) dari basis data LIVE
          menjadi tabel T_COINSLIST di basis data TEST.
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
            {busy === 'test' ? 'Menjalankan uji coba…' : 'Uji Coba'}
          </Button>
          <Button tone="utama" disabled={!ready || unique === 0 || busy !== null} onClick={() => execute(false)}>
            {busy === 'run' ? 'Mengonversi…' : 'Jalankan Konversi'}
          </Button>
        </div>
      </section>

      {error && <ErrorMessage tone="gangguan" title="Konversi gagal" description={error} />}
      {report && <ReportView report={report} />}
    </div>
  )
}

function ConnectionCard({ title, info, loading }: { title: string; info: ConnectionInfo | undefined; loading: boolean }) {
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

const statusLabel: Record<PolicyResult['status'], { text: string; cls: string }> = {
  berhasil: { text: 'Berhasil', cls: 'bg-green-100 text-green-800' },
  tidak_ditemukan: { text: 'Tidak ditemukan', cls: 'bg-amber-100 text-amber-800' },
  gagal: { text: 'Gagal', cls: 'bg-red-100 text-red-800' },
}

function ReportView({ report }: { report: Report }) {
  return (
    <section className="space-y-3">
      <div className="rounded-kontrol border border-slate-200 bg-white p-4 text-sm">
        <p className="font-semibold text-slate-900">
          {report.uji_coba ? 'Hasil Uji Coba — tidak ada yang tersimpan' : 'Hasil Konversi — tersimpan di TEST'}
        </p>
        <p className="mt-1 text-slate-700">
          {report.berhasil} berhasil · {report.tidak_ditemukan} tidak ditemukan · {report.gagal} gagal ·{' '}
          {report.total_baris} baris koasuransi
        </p>
      </div>

      <div className="overflow-x-auto rounded-kontrol border border-slate-200 bg-white">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-slate-600">
            <tr>
              <th className="px-3 py-2">Nomor Polis</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">PRODKE</th>
              <th className="px-3 py-2 text-right">Baris Coins</th>
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
                <td className="px-3 py-2">{(p.versi_prodke ?? []).join(', ')}</td>
                <td className="px-3 py-2 text-right">
                  {p.baris_ditulis}
                  {p.baris_dihapus > 0 && <span className="text-slate-500"> (ganti {p.baris_dihapus})</span>}
                </td>
                <td className="max-w-md px-3 py-2 text-xs">
                  {p.galat && <p className="text-red-700">{p.galat}</p>}
                  {p.versi_tanpa_coins > 0 && (
                    <p className="text-slate-600">{p.versi_tanpa_coins} versi tanpa koasuransi (CoinsList kosong)</p>
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
