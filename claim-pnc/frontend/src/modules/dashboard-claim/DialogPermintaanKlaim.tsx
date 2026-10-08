import { useEffect, useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

import { useAjukanPermintaanKlaim } from './api'
import type { BarisKlaim, HasilPermintaanKlaim, JenisPermintaanKlaim } from './types'

/**
 * Konfirmasi pengajuan **ReOpen** dan **Copy Klaim** atas baris yang dicentang.
 *
 * Menggantikan dua modal layar lama: `GCNMReopenConfirmation` dan
 * `GCNMCopyClaimConfirmation`, keduanya dipanggil sebagai `pyLocalAction` dari
 * `Section/InboxManagerReopen1_Sec-Section.xml`.
 *
 * # Isi modalnya tidak dapat ditiru, dan itu dinyatakan
 *
 * **Kedua flow action itu HILANG dari export** (`R-16`): tidak ada berkasnya di
 * `Flow Action/`. Yang terbaca hanya namanya, bukan isinya — sehingga tidak diketahui isian
 * apa yang dimintanya dan peringatan apa yang ditampilkannya.
 *
 * Yang dibangun di sini karena itu adalah konfirmasi yang **paling sedikit menduga**:
 * menyebutkan apa yang akan terjadi, atas berapa klaim, dengan satu isian Alasan yang
 * opsional. Menebak isian lain berarti menolak pengajuan yang sah karena isian karangan
 * kita sendiri tidak diisi.
 */
export function DialogPermintaanKlaim({
  jenis,
  baris,
  onTutup,
}: {
  jenis: JenisPermintaanKlaim
  baris: BarisKlaim[]
  onTutup: () => void
}) {
  const [alasan, setAlasan] = useState('')
  const ajukan = useAjukanPermintaanKlaim()
  const tutupRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    tutupRef.current?.focus()

    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  const judul = jenis === 'reopen' ? 'ReOpen Klaim' : 'Copy Klaim'
  const hasil = ajukan.data ?? null

  function kirim(event: React.FormEvent) {
    event.preventDefault()
    ajukan.mutate({
      jenis,
      alasan,
      baris: baris.map((row) => ({ klaim_id: row.klaim_id, nomor_klaim: row.nomor_klaim })),
    })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={judul}
    >
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">{judul}</h2>
            <p className="mt-1 text-sm text-slate-600">
              {baris.length} klaim dipilih.
            </p>
          </div>
          {/* Tombol polos, bukan Button: ia perlu `ref` untuk menerima fokus saat dibuka. */}
          <button
            ref={tutupRef}
            type="button"
            onClick={onTutup}
            aria-label="Tutup"
            className={[
              'rounded-kontrol px-3 py-1.5 text-sm text-slate-500',
              'transition ease-halus hover:bg-slate-100 hover:text-slate-700',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
            ].join(' ')}
          >
            ✕
          </button>
        </div>

        {hasil !== null ? (
          <Ringkasan hasil={hasil} jenis={jenis} onTutup={onTutup} />
        ) : (
          <form onSubmit={kirim} className="space-y-4">
            <div className="rounded-kontrol border border-slate-200 bg-slate-50 p-4 text-sm leading-relaxed text-slate-700">
              {jenis === 'reopen' ? (
                <>
                  Klaim yang dipilih akan diajukan untuk <strong>dibuka kembali</strong>.
                </>
              ) : (
                <>
                  Klaim baru akan diajukan sebagai <strong>salinan</strong> dari klaim yang
                  dipilih, beserta polis, objek, dan coverage-nya.
                </>
              )}{' '}
              Yang tercatat sekarang adalah <strong>permintaannya</strong> — klaimnya belum
              berubah dan barisnya masih akan tampil di daftar.
            </div>

            <DaftarKlaim baris={baris} />

            <Field
              id="permintaan-alasan"
              label="Alasan"
              value={alasan}
              onChange={(event) => setAlasan(event.target.value)}
              hint="Opsional. Tercatat pada jejak permintaan."
            />

            {ajukan.isError ? (
              <ErrorMessage
                tone="gangguan"
                title="Permintaan tidak dapat dikirim"
                description={pesanGalat(ajukan.error)}
              />
            ) : null}

            <div className="flex justify-end gap-2">
              <Button tone="kedua" type="button" onClick={onTutup}>
                Batal
              </Button>
              <Button tone="utama" type="submit" disabled={ajukan.isPending}>
                {ajukan.isPending ? 'Mengirim…' : `Ajukan ${judul}`}
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}

/** Daftar klaim yang akan diajukan, supaya pengguna melihat apa yang dicentangnya. */
function DaftarKlaim({ baris }: { baris: BarisKlaim[] }) {
  return (
    <div className="max-h-40 overflow-y-auto rounded-kontrol border border-slate-200">
      <ul className="divide-y divide-slate-100 text-sm">
        {baris.map((row) => (
          <li key={row.klaim_id || row.nomor_klaim} className="px-3 py-2">
            <span className="font-medium text-slate-900">{row.nomor_klaim}</span>
            <span className="ml-2 text-slate-500">{row.nama_tertanggung}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/**
 * Hasil per baris.
 *
 * Pengajuan massal adalah N permintaan terpisah, sehingga sebagiannya dapat berhasil. Yang
 * gagal disebut satu per satu beserta sebabnya — bukan diringkas menjadi satu "gagal", yang
 * akan membuat pengguna mengirim ulang seluruhnya.
 */
function Ringkasan({
  hasil,
  jenis,
  onTutup,
}: {
  hasil: HasilPermintaanKlaim[]
  jenis: JenisPermintaanKlaim
  onTutup: () => void
}) {
  const berhasil = hasil.filter((item) => item.berhasil)
  const gagal = hasil.filter((item) => !item.berhasil)

  return (
    <>
      {berhasil.length > 0 ? (
        <div className="rounded-kontrol border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
          <p className="font-medium">
            {berhasil.length} permintaan {jenis === 'reopen' ? 'ReOpen' : 'Copy Klaim'} tercatat.
          </p>
          <p className="mt-1 leading-relaxed">
            Klaimnya <strong>belum berubah</strong> — barisnya masih akan tampil di daftar.
            Pelaksanaannya dikerjakan di Pega selama masa paralel.
          </p>
        </div>
      ) : null}

      {gagal.length > 0 ? (
        <div className="mt-3 rounded-kontrol border border-rose-200 bg-rose-50 p-4 text-sm text-rose-900">
          <p className="font-medium">{gagal.length} permintaan tidak tercatat:</p>
          <ul className="mt-2 space-y-1">
            {gagal.map((item) => (
              <li key={item.klaim_id || item.nomor_klaim}>
                <span className="font-medium">{item.nomor_klaim}</span>
                {item.pesan ? <span> — {item.pesan}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="mt-4 flex justify-end">
        <Button tone="utama" onClick={onTutup}>
          Tutup
        </Button>
      </div>
    </>
  )
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
