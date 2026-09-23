import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import type { JenisPermintaan, KlaimTutup } from './types'

/**
 * Dialog konfirmasi ReOpen dan Copy Klaim.
 *
 * # Ia menggantikan dua rule yang TIDAK ADA di export
 *
 * Di Pega, kedua tombol memanggil `SaveReOpenAct` lalu membuka modal
 * `GCNMReopenConfirmation` atau `GCNMCopyClaimConfirmation`. **Ketiganya tidak ada di
 * export** — penelusuran seluruh berkas menghasilkan nol, termasuk untuk rule mana pun yang
 * menulis status `1164` atau menyentuh `PYREOPENCOUNT`.
 *
 * Isi dialog ini karena itu TIDAK menyalin tampilan modal lama — tidak ada yang dapat
 * disalin. Yang ditampilkan adalah akibat yang Work Owner tetapkan pada 2026-09-23, ditulis
 * apa adanya supaya pengguna membaca apa yang benar-benar akan terjadi.
 *
 * # Kenapa konfirmasi, bukan langsung kirim
 *
 * Keduanya tidak dapat dibatalkan lewat layar ini: begitu permintaannya tercatat, yang
 * membatalkannya adalah pelaksana di sisi Pega. Copy Klaim bahkan MENERBITKAN KLAIM BARU
 * bernomor sendiri — satu penekanan tombol yang tidak disengaja meninggalkan klaim yang
 * harus dibereskan seseorang.
 */
export function RequestDialog({
  jenis,
  klaim,
  onBatal,
  onKirim,
  sedangMengirim,
  galat,
}: {
  jenis: JenisPermintaan
  klaim: KlaimTutup
  onBatal: () => void
  onKirim: (alasan: string) => void
  sedangMengirim: boolean
  galat: string | null
}) {
  const [alasan, setAlasan] = useState('')

  // Escape menutup dialog, seperti dialog mana pun yang dikenal pengguna.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !sedangMengirim) onBatal()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onBatal, sedangMengirim])

  const reopen = jenis === 'reopen'
  const judul = reopen ? 'Buka kembali klaim ini?' : 'Salin klaim ini menjadi klaim baru?'

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-permintaan"
    >
      <div className="max-h-full w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-permintaan" className="text-lg font-semibold text-slate-900">
          {judul}
        </h2>

        <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt className="text-slate-500">No Klaim</dt>
          <dd className="font-mono text-slate-900">{klaim.nomor_klaim || '—'}</dd>
          <dt className="text-slate-500">No Polis</dt>
          <dd className="text-slate-900">{klaim.nomor_polis || '—'}</dd>
          <dt className="text-slate-500">Tertanggung</dt>
          <dd className="text-slate-900">{klaim.nama_tertanggung || '—'}</dd>
        </dl>

        {/*
          Akibatnya ditulis lengkap, bukan diringkas menjadi "lanjutkan?".

          Dua di antaranya tidak terlihat di layar mana pun setelah tombol ditekan: klaimnya
          tidak berubah seketika, dan Copy Klaim menerbitkan klaim BARU. Pengguna yang tidak
          diberi tahu keduanya akan menyangka tombolnya gagal.
        */}
        <div className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
          <p className="font-medium text-slate-900">Yang akan terjadi</p>
          {reopen ? (
            <ul className="mt-2 list-disc space-y-1 pl-5">
              <li>Status alur kerja klaim kembali menjadi berjalan.</li>
              <li>
                Status klaim menjadi <span className="font-medium">Reopen Claim</span>.
              </li>
              <li>Pencacah pembukaan kembali bertambah satu, beserta waktunya.</li>
            </ul>
          ) : (
            <ul className="mt-2 list-disc space-y-1 pl-5">
              <li>Klaim baru terbit dengan nomornya sendiri, mulai dari tahap registrasi.</li>
              <li>Yang disalin: data polis, objek pertanggungan, dan coverage.</li>
              <li>
                Yang <span className="font-medium">tidak</span> disalin: nilai estimasi,
                usulan, akseptasi, dan pembayaran.
              </li>
            </ul>
          )}

          <p className="mt-3 border-t border-slate-200 pt-3">
            Permintaan ini dicatat beserta nama Anda dan waktunya.{' '}
            <span className="font-medium text-slate-900">Klaim belum berubah sekarang</span> —
            ia berubah setelah permintaannya dijalankan.
          </p>
        </div>

        <div className="mt-4">
          <label htmlFor="alasan-permintaan" className="block text-sm font-medium text-slate-700">
            Alasan <span className="font-normal text-slate-500">(opsional)</span>
          </label>
          <textarea
            id="alasan-permintaan"
            rows={3}
            maxLength={1500}
            value={alasan}
            onChange={(event) => setAlasan(event.target.value)}
            disabled={sedangMengirim}
            placeholder="Mis. dokumen susulan diterima dari tertanggung"
            className="mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-slate-900 shadow-lembut transition-[border-color,box-shadow] duration-150 focus:outline-none focus-visible:border-blue-500 focus-visible:ring-4 focus-visible:ring-blue-500/25 disabled:bg-slate-50"
          />
          {/*
            Alasan TIDAK diwajibkan — Work Owner memilih efek reopen tanpa mewajibkannya
            (2026-09-23). Keterangan di bawah menjelaskan mengapa ia tetap berguna, alih-alih
            memaksanya lewat validasi yang tidak pernah diminta.
          */}
          <p className="mt-1 text-xs text-slate-500">
            Membantu pelaksana dan siapa pun yang kelak menelusuri mengapa klaim ini dibuka
            kembali.
          </p>
        </div>

        {galat && (
          <div className="mt-4">
            <ErrorMessage title="Permintaan tidak dapat dicatat" description={galat} tone="gangguan" />
          </div>
        )}

        <div className="mt-6 flex justify-end gap-2">
          {/*
            Fokus jatuh ke BATAL saat dialog terbuka, bukan ke tombol kirim: tombol kirim
            yang terfokus akan terpicu oleh satu ketukan Enter dari pengguna yang belum
            sempat membaca isinya.

            Dipasang lewat autoFocus, bukan lewat ref — `Button` tidak meneruskan ref,
            sehingga ref yang dipasang di sini akan diam-diam tidak berfungsi.
          */}
          <Button autoFocus tone="halus" onClick={onBatal} disabled={sedangMengirim}>
            Batal
          </Button>
          <Button tone="utama" onClick={() => onKirim(alasan)} disabled={sedangMengirim}>
            {sedangMengirim ? 'Mengirim…' : reopen ? 'Ajukan buka kembali' : 'Ajukan salin klaim'}
          </Button>
        </div>
      </div>
    </div>
  )
}
