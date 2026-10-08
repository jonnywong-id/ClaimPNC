import { useState, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'

import { useDetailRCL, useKeputusanRCL } from './api'
import type { DetailRCL, KeputusanRCL } from './types'

/** Nilai `RCL_PUCL` yang menentukan isi layar. */
const MODE_RCL = '1'
const MODE_MSIG = '3'

/**
 * Layar kerja RCL Dokter — pengganti Flow Action `SendToRCLDokter`, yang menyertakan section
 * `RCLDokter` (`Section/RCLDokter-Section.xml`). Dibuka saat Nomor Case di Inbox RCL diklik.
 *
 * Seluruh isinya dari POOLDATA.TC_PNC_PUCL (keputusan Work Owner 2026-10-05). Section-nya punya
 * dua mode menurut `PUCLStatus.RCL_PUCL`:
 *
 *   RCL  ("1")  Catatan dari Analyst · Alasan Klaim Ditolak/RCL ·
 *               "Apakah anda setuju untuk menolak klaim ini?" · Setuju / Tidak Setuju
 *   MSIG ("3")  Catatan dari Analyst · Alasan Klaim MSIG · Back / Submit
 *
 * Tombol "Lihat Dokumen" pada section tidak dibawa: syarat tampilnya `1=2`, sehingga di Pega
 * ia tidak pernah muncul.
 *
 * # Tombol
 *
 * Setuju dan Submit langsung menjalankan keputusan (`SendToPUCL` dengan `StatusRCL` "SETUJU"
 * dan "MSIG"). Tidak Setuju dan Back membuka layar Alasan Dokter
 * (`Section/sendToAnalystTolakRCL_sect-Section.xml`); tombol Kirim di sana menjalankan
 * "TidakSetuju" atau "BackMSIG". Sesudah berhasil, klaim keluar dari antrean dokter dan layar
 * kembali ke Inbox RCL — seperti Pega yang menutup penugasan lalu kembali ke portal.
 */
export function RCLDokterPage() {
  const { nomor } = useParams<{ nomor: string }>()
  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)

  const key = nomor ? decodeURIComponent(nomor) : ''
  const detail = useDetailRCL(key || null)
  const data = detail.data ?? null

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            {data ? `Klaim ${data.nomor_case}` : 'RCL Dokter'}
          </h1>
          <p className="mt-1 text-sm text-slate-600">Layar kerja dokter RCL.</p>
        </div>
        <Button tone="kedua" onClick={() => navigate('/inbox-rcl')}>
          Kembali ke antrean
        </Button>
      </header>

      {portal === null ? (
        <div className="mt-4">
          <ErrorMessage
            title="Pilih entitas lebih dulu"
            description="Pilih portal di bilah atas untuk membuka klaim ini."
            tone="gangguan"
          />
        </div>
      ) : detail.isPending ? (
        <p className="mt-4 text-sm text-slate-500" role="status">
          Memuat klaim…
        </p>
      ) : detail.isError ? (
        <div className="mt-4">
          <ErrorMessage
            title="Klaim tidak dapat dibuka"
            description={pesanGalat(detail.error)}
            tone={detail.error instanceof APIError && detail.error.status === 404 ? 'penolakan' : 'gangguan'}
          />
        </div>
      ) : data ? (
        <WorkScreen detail={data} onDone={() => navigate('/inbox-rcl', { replace: true })} />
      ) : null}
    </div>
  )
}

/** Isi section `RCLDokter`, urutan dan label apa adanya — atau layar Alasan Dokter. */
function WorkScreen({ detail, onDone }: { detail: DetailRCL; onDone: () => void }) {
  const rcl = detail.mode === MODE_RCL
  const msig = detail.mode === MODE_MSIG

  const keputusan = useKeputusanRCL(detail.nomor_case)
  const [alasanTerbuka, setAlasanTerbuka] = useState(false)
  const [alasanDokter, setAlasanDokter] = useState(detail.alasan_dokter)

  function kirim(nilai: KeputusanRCL) {
    keputusan.mutate(
      // Alasan Dokter hanya ikut terkirim dari layar Alasan Dokter (Tidak Setuju/Back).
      { keputusan: nilai, alasanDokter: alasanTerbuka ? alasanDokter : '' },
      { onSuccess: onDone },
    )
  }

  const galat = keputusan.isError ? (
    <div className="mt-4">
      <ErrorMessage
        title="Keputusan tidak tersimpan"
        description={pesanGalat(keputusan.error)}
        tone={keputusan.error instanceof APIError && keputusan.error.status < 500 ? 'penolakan' : 'gangguan'}
      />
    </div>
  ) : null

  if (alasanTerbuka) {
    // Section `sendToAnalystTolakRCL_sect` — satu isian (tidak wajib) dan tombol Kirim.
    return (
      <div className="mt-4 rounded-sm border border-slate-300 bg-white p-5">
        <TextAreaField
          id="alasan-dokter"
          label="Alasan Dokter"
          value={alasanDokter}
          onChange={(event) => setAlasanDokter(event.target.value)}
          rows={5}
        />
        {galat}
        <Actions>
          <Button tone="kedua" disabled={keputusan.isPending} onClick={() => setAlasanTerbuka(false)}>
            Cancel
          </Button>
          <Button
            tone="utama"
            disabled={keputusan.isPending}
            onClick={() => kirim(rcl ? 'TidakSetuju' : 'BackMSIG')}
          >
            Kirim
          </Button>
        </Actions>
      </div>
    )
  }

  return (
    <div className="mt-4 rounded-sm border border-slate-300 bg-white p-5">
      <dl className="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
        <Info label="Nomor Case" value={detail.nomor_case} />
        <Info label="No Polis" value={detail.nomor_polis} />
        <Info label="Nama Tertanggung" value={detail.nama_tertanggung} />
      </dl>

      <div className="mt-5 space-y-4">
        <TextAreaField
          id="catatan-analyst"
          label="Catatan dari Analyst"
          value={detail.catatan_analyst}
          readOnly
          rows={4}
        />

        {rcl && (
          <TextAreaField
            id="alasan-rcl"
            label="Alasan Klaim Ditolak/RCL"
            value={detail.alasan}
            readOnly
            rows={4}
          />
        )}

        {msig && (
          <TextAreaField
            id="alasan-msig"
            label="Alasan Klaim MSIG"
            value={detail.alasan}
            readOnly
            rows={4}
          />
        )}

        {rcl && (
          <p className="text-[15px] font-semibold text-slate-800">
            Apakah anda setuju untuk menolak klaim ini?
          </p>
        )}
      </div>

      {galat}

      <Actions>
        {rcl && (
          <>
            <Button tone="utama" disabled={keputusan.isPending} onClick={() => kirim('SETUJU')}>
              Setuju
            </Button>
            <Button tone="kedua" disabled={keputusan.isPending} onClick={() => setAlasanTerbuka(true)}>
              Tidak Setuju
            </Button>
          </>
        )}
        {msig && (
          <>
            <Button tone="kedua" disabled={keputusan.isPending} onClick={() => setAlasanTerbuka(true)}>
              Back
            </Button>
            <Button tone="utama" disabled={keputusan.isPending} onClick={() => kirim('MSIG')}>
              Submit
            </Button>
          </>
        )}
      </Actions>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="mt-0.5 font-medium text-slate-800">{value || '—'}</dd>
    </div>
  )
}

function Actions({ children }: { children: ReactNode }) {
  return <div className="mt-6 flex flex-wrap justify-end gap-2">{children}</div>
}

function pesanGalat(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
