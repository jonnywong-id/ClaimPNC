import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Recovery } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { AddIcon, ReloadIcon } from '@/components/Icon'

import { TabBar } from '@/components/TabBar'

import { OutstandingTable } from './OutstandingTable'
import { RecoveryForm } from './RecoveryForm'
import { VirtualAccountPanel } from './VirtualAccountPanel'
import { useRecoveryForm, useRecoveryPrincipals, useRefreshRecoveryList } from './api'

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
 * # Susunannya mengikuti layar lama (diperbaiki 2026-09-29)
 *
 * Layar Pega berbentuk: judul di kiri, tombol **Tambah** dan **Refresh** di kanannya, satu
 * tab **Outstanding**, lalu grid berisi batch yang sudah tercatat. Entri dibuka lewat
 * Tambah, bukan tergelar sejak layar dibuka.
 *
 * Susunan itu ditiru di sini (`D-13`), menggantikan bentuk sebelumnya yang menaruh form
 * sebagai isi utama dan daftar sebagai pelengkap di bawahnya.
 *
 * # Daftar Outstanding sempat tidak ada, dan itu KELIRU
 *
 * Bentuk sebelumnya tidak punya daftar sama sekali — hanya rekap batch yang disimpan sejak
 * layar dibuka. Dasarnya kesimpulan saya bahwa layar lama adalah form entri tanpa daftar,
 * yang ditarik dari tidak adanya kueri pembaca di export.
 *
 * Kesimpulan itu salah, dan cara menariknya yang salah: export-nya sendiri tidak lengkap
 * (`R-16`). Grid Outstanding ada, terbaca jelas di
 * `Section/OutstandingMasterRecovery-Section.xml`, dan berisi data di Pega yang berjalan.
 * Yang hilang adalah rule pemuatnya, bukan fiturnya.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Panel VA | selalu tampil, disembunyikan sakelar `FlagASO` | dibuka tombol, tertutup secara baku |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 * | Hasil simpan | tidak ditampilkan | nomor batch dan sisa yang benar-benar tersimpan |
 * | Entri | jendela modal | panel yang terbuka di tempat, di atas daftar |
 *
 * Baris terakhir adalah satu-satunya penyimpangan susunan yang disengaja: form ini panjang
 * — belasan isian, unggahan bukti bayar, dan grid data klaim — dan memaksanya ke dalam
 * jendela modal membuat isinya harus digulir di dalam gulungan halaman. Yang ditiru adalah
 * ALUR-nya (daftar dulu, entri dibuka lewat Tambah), bukan wadahnya.
 */
export function RecoveryPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const form = useRecoveryForm()
  const principal = useRecoveryPrincipals()
  const refreshList = useRefreshRecoveryList()

  const [vaOpen, setVAOpen] = useState(false)
  const [entryOpen, setEntryOpen] = useState(false)
  const [lastSaved, setLastSaved] = useState<Recovery | null>(null)
  const [lastPolicyMissing, setLastPolicyMissing] = useState(false)

  function handleSaved(row: Recovery, policyResolved: boolean) {
    setLastSaved(row)
    setLastPolicyMissing(row.nomor_polis !== '' && !policyResolved)

    // Panel entri ditutup setelah berhasil, sehingga yang terlihat berikutnya adalah
    // DAFTAR berisi batch yang baru saja tersimpan — bukti bahwa ia benar-benar tercatat,
    // bukan sekadar pesan yang mengatakannya. Daftarnya dimuat ulang oleh hook simpan.
    setEntryOpen(false)
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
          {loadFailed && <LoadErrorMessage error={loadError} />}

          {lastSaved !== null && (
            <div className="rounded-kartu border border-emerald-200 bg-emerald-50 p-4">
              <p className="text-sm font-medium text-emerald-900">
                Batch {lastSaved.nomor_batch} tersimpan
              </p>
              <p className="mt-1 text-xs text-emerald-800">
                Sisa yang tercatat: Rp {lastSaved.sisa.toLocaleString('id-ID')} — angka ini
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

          {vaOpen && (
            <VirtualAccountPanel
              onClose={() => setVAOpen(false)}
              onIssued={() => {
                // Panel dibiarkan TERBUKA supaya nomor yang baru terbit tetap terlihat dan
                // dapat disalin. Daftar principal sudah dimuat ulang oleh hook-nya, jadi
                // principal baru itu langsung dapat dipilih pada form entri.
                principal.refetch()
              }}
            />
          )}

          {entryOpen && (
            <RecoveryForm
              nextBatch={form.data?.nomor_batch_perkiraan}
              year={form.data?.tahun ?? []}
              principal={principal.data?.principal ?? []}
              onSaved={handleSaved}
            />
          )}

          {/* Satu tab saja, sama seperti layar lama. Ia tetap digambar meski tunggal:
              bilahnya bagian dari susunan yang dikenali petugas, dan menghilangkannya
              membuat layar ini satu-satunya master yang berbeda bentuk. */}
          <TabBar
            tabs={[{ kode: 'outstanding', nama: 'Outstanding' }]}
            active="outstanding"
            onSelect={() => undefined}
            label="Daftar Master Recovery"
          />

          <OutstandingTable
            actions={
              <>
                <Button
                  tone="kedua"
                  onClick={() => {
                    form.refetch()
                    principal.refetch()
                    refreshList()
                  }}
                  disabled={form.isFetching || principal.isFetching}
                >
                  <ReloadIcon
                    className={`h-4 w-4 ${form.isFetching || principal.isFetching ? 'animate-spin' : ''}`}
                  />
                  {form.isFetching || principal.isFetching ? 'Memuat…' : 'Refresh'}
                </Button>
                <Button tone="halus" onClick={() => setVAOpen((open) => !open)}>
                  {vaOpen ? 'Tutup panel VA' : 'Terbitkan VA baru'}
                </Button>
                <Button tone="utama" onClick={() => setEntryOpen((open) => !open)}>
                  <AddIcon className="h-4 w-4" />
                  {entryOpen ? 'Tutup form' : 'Tambah'}
                </Button>
              </>
            }
          />
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
function LoadErrorMessage({ error }: Readonly<{ error: unknown }>) {
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
