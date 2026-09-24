import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Recovery } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { AddIcon, ReloadIcon } from '@/components/Icon'

import { RecoveryForm, SavedRecoveryTable } from './RecoveryForm'
import { VirtualAccountPanel } from './VirtualAccountPanel'
import { useRecoveryForm, useRecoveryPrincipals } from './api'

/**
 * Layar Master Recovery.
 *
 * Menggantikan harness `MasterRecovery` beserta kedua section-nya:
 * `OutstandingMasterRecovery` (form entri) dan `DetailMasterRecovery` (panel penerbitan
 * Virtual Account). Butir menunya `MENU_ID 16` pada POOLDATA.M_MENU_APLIKASI_PNC.
 *
 * # Apa yang dikerjakan di sini
 *
 * Mencatat **pemulihan dana klaim dari pihak penjamin** — satu baris per batch ke
 * POOLDATA.MST_RECOVERY_ASM_PENJAMINAN, beserta bukti bayar dan daftar polis yang
 * tercakup.
 *
 * # Kenapa TIDAK ada daftar data tersimpan
 *
 * Bukan kelalaian, dan bukan penyederhanaan. Tidak ada satu pun kueri di seluruh export
 * Pega yang MEMBACA tabel itu — layarnya form entri, bukan daftar. Keputusan Work Owner
 * 2026-09-19 menetapkan itu ditiru apa adanya, sehingga rute daftar pun tidak dibuat di
 * server.
 *
 * Yang ada di bawah hanyalah rekap batch yang disimpan SEJAK LAYAR DIBUKA, dan layar
 * mengatakannya terus terang — supaya tidak ada yang mengira itu isi tabelnya.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Panel VA | selalu tampil, disembunyikan sakelar `FlagASO` | dibuka tombol, tertutup secara baku |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 * | Hasil simpan | tidak ditampilkan | nomor batch dan sisa yang benar-benar tersimpan |
 */
export function RecoveryPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const form = useRecoveryForm()
  const principal = useRecoveryPrincipals()

  const [vaOpen, setVAOpen] = useState(false)
  const [saved, setSaved] = useState<Recovery[]>([])
  const [lastPolicyMissing, setLastPolicyMissing] = useState(false)

  function handleSaved(row: Recovery, policyResolved: boolean) {
    // Terbaru di puncak: yang baru saja disimpan adalah yang paling ingin dilihat petugas.
    setSaved((previous) => [row, ...previous])
    setLastPolicyMissing(row.nomor_polis !== '' && !policyResolved)
  }

  const loadFailed = form.isError || principal.isError
  const loadError = form.error ?? principal.error

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Recovery</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master Recovery</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Pencatatan pemulihan dana klaim dari pihak penjamin. Setiap batch menyebut
          principal, tahun, nilai klaim, pembayaran, dan sisanya — beserta bukti bayar dan
          daftar polis yang tercakup.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan di layar ini akibat salah
            entitas paling berat: nomor rekening virtual milik badan hukum lain dapat
            mengarahkan dana ke tempat yang keliru (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">
            {form.data?.portal ?? portal ?? '—'}
          </span>
        </p>
      </header>

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Data recovery dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
        <div className="space-y-6">
          <div className="flex flex-wrap items-center justify-end gap-2">
            <Button
              tone="kedua"
              onClick={() => {
                void form.refetch()
                void principal.refetch()
              }}
              disabled={form.isFetching || principal.isFetching}
            >
              <ReloadIcon
                className={`h-4 w-4 ${form.isFetching || principal.isFetching ? 'animate-spin' : ''}`}
              />
              {form.isFetching || principal.isFetching ? 'Memuat…' : 'Refresh'}
            </Button>
            <Button tone="halus" onClick={() => setVAOpen((open) => !open)}>
              <AddIcon className="h-4 w-4" />
              {vaOpen ? 'Tutup panel VA' : 'Terbitkan VA baru'}
            </Button>
          </div>

          {loadFailed && <LoadErrorMessage error={loadError} />}

          {vaOpen && (
            <VirtualAccountPanel
              onClose={() => setVAOpen(false)}
              onIssued={() => {
                // Panel dibiarkan TERBUKA supaya nomor yang baru terbit tetap terlihat dan
                // dapat disalin. Daftar principal sudah dimuat ulang oleh hook-nya, jadi
                // principal baru itu langsung dapat dipilih pada form di bawah.
                void principal.refetch()
              }}
            />
          )}

          {saved.length > 0 && (
            <div className="rounded-kartu border border-emerald-200 bg-emerald-50 p-4">
              <p className="text-sm font-medium text-emerald-900">
                Batch {saved[0]?.nomor_batch} tersimpan
              </p>
              <p className="mt-1 text-xs text-emerald-800">
                Sisa yang tercatat: Rp {saved[0]?.sisa.toLocaleString('id-ID')} — angka ini
                dihitung server, dan itulah yang tersimpan.
              </p>
              {lastPolicyMissing && (
                /*
                  Diberitahukan terpisah karena batch-nya TETAP tersimpan. Sistem lama pun
                  meneruskan dengan keempat kolom identitas kosong tanpa mengatakan apa pun
                  — petugas tidak punya cara mengetahuinya sampai membaca laporan.
                */
                <p className="mt-2 text-xs font-medium text-amber-800">
                  Identitas polis tidak ditemukan, sehingga lini bisnis, cabang, agen, dan
                  marketing tersimpan kosong. Batch-nya sendiri sudah tercatat.
                </p>
              )}
            </div>
          )}

          <RecoveryForm
            nextBatch={form.data?.nomor_batch_perkiraan}
            year={form.data?.tahun ?? []}
            principal={principal.data?.principal ?? []}
            onSaved={handleSaved}
          />

          <SavedRecoveryTable rows={saved} />
        </div>
      )}
    </div>
  )
}

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat salah
 * — ia baru membuka layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Layar belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data recovery dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return { title: 'Layar gagal dimuat', description: error.message, tone: 'gangguan' }
    }
  }

  return {
    title: 'Layar gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
