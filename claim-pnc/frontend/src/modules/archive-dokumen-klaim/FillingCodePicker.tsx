import { useState } from 'react'

import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { CloseIcon, SearchIcon } from '@/components/Icon'

import { useFillingCodes } from './api'
import type { FillingCode } from './types'

type Props = {
  open: boolean
  onClose: () => void

  /** Dipanggil saat satu kode dipilih; membawa kode DAN nama boks pasangannya. */
  onPick: (code: FillingCode) => void
}

/**
 * FillingCodePicker adalah pengganti tombol "Pilih Kode" pada layar lama.
 *
 * # Apa yang diketahui, dan apa yang tidak
 *
 * Bentuk layarnya terbaca: `Section/KodeArchiveDoc-Section.xml` memuat tiga kolom — Kode
 * Archive, Desc Archive, Nama Box — beserta tombol Cari Kode, Input Kode, Generated Kode,
 * dan Pilih.
 *
 * SUMBER DATANYA TIDAK. Activity `SetKodeandSearchArchiveDoc` yang mengisinya tidak ada di
 * export, dan tidak ada satu pun tabel master kode arsip di seluruh 2.634 berkas
 * (`R-16`).
 *
 * # Apa yang dipakai sebagai gantinya, dan kenapa itu masuk akal
 *
 * Daftarnya disusun dari kode filling yang SUDAH PERNAH DIPAKAI berkas arsip. Dua hal
 * membuat rekonstruksi ini beralasan alih-alih tebakan buta: layar lama menyediakan
 * tombol "Input Kode" dan "Generated Kode" di samping "Cari Kode" — yang berarti kode
 * memang dapat DIBUAT dari layar ini, bukan hanya dipilih dari master tetap.
 *
 * Konsekuensinya disebut terang DI LAYAR, bukan hanya di dokumen: kode yang belum pernah
 * dipakai tidak muncul di daftar, dan karena itu isian Kode Filling tetap dapat diketik
 * langsung. Tanpa keterangan itu, pengguna akan menyimpulkan kode barunya ditolak.
 *
 * Kolom "Desc Archive" tidak digambar — isinya berasal dari master yang tidak kita miliki.
 * Yang menggantikannya adalah jumlah pemakaian, keterangan yang benar-benar ada.
 */
export function FillingCodePicker({ open, onClose, onPick }: Props) {
  const [keyword, setKeyword] = useState('')

  const codes = useFillingCodes(keyword, open)

  if (!open) return null

  const rows: FillingCode[] = codes.data?.kode ?? []

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 p-4 sm:p-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-pilih-kode"
    >
      <div className="w-full max-w-2xl rounded-kartu border border-slate-200 bg-white shadow-angkat">
        <header className="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 id="judul-pilih-kode" className="text-lg font-semibold text-slate-900">
              Pilih Kode Filling
            </h2>
            <p className="mt-1 text-sm text-slate-600">
              Daftar ini disusun dari kode yang sudah pernah dipakai berkas arsip — bukan
              dari master. Kode baru tetap boleh diketik langsung di isian Kode Filling.
            </p>
          </div>

          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup pemilih kode"
            className="rounded-kontrol p-2 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20"
          >
            <CloseIcon className="h-4 w-4" />
          </button>
        </header>

        <div className="p-5">
          <Field
            id="cari-kode-filling"
            label="Cari Kode"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            icon={<SearchIcon className="h-4 w-4" />}
            placeholder="FIL-2024"
            hint="Mencari pada Kode Archive maupun Nama Box."
          />

          <div className="mt-4 max-h-80 overflow-y-auto rounded-kontrol border border-slate-200">
            <table className="w-full text-left text-sm">
              <caption className="sr-only">Daftar kode filling yang sudah dipakai</caption>
              <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-600">
                <tr>
                  <th scope="col" className="px-4 py-2.5 font-medium">
                    Kode Archive
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-medium">
                    Nama Box
                  </th>
                  <th scope="col" className="px-4 py-2.5 text-right font-medium">
                    Dipakai
                  </th>
                  <th scope="col" className="px-4 py-2.5">
                    <span className="sr-only">Aksi</span>
                  </th>
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-200">
                {codes.isPending && (
                  <tr>
                    <td colSpan={4} className="px-4 py-6 text-center text-slate-500">
                      Memuat kode…
                    </td>
                  </tr>
                )}

                {!codes.isPending && rows.length === 0 && (
                  <tr>
                    <td colSpan={4} className="px-4 py-6 text-center text-slate-500">
                      Belum ada kode yang cocok. Ketik kode baru langsung di isian Kode
                      Filling.
                    </td>
                  </tr>
                )}

                {rows.map((code) => (
                  <tr key={`${code.kode}|${code.nama_box}`} className="hover:bg-slate-50">
                    <td className="px-4 py-2.5 font-medium text-slate-900">{code.kode}</td>
                    <td className="px-4 py-2.5 text-slate-700">{code.nama_box || '—'}</td>
                    <td className="px-4 py-2.5 text-right tabular-nums text-slate-600">
                      {code.jumlah_pemakaian}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <Button type="button" tone="kedua" onClick={() => onPick(code)}>
                        Pilih
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <footer className="flex justify-end border-t border-slate-200 p-5">
          <Button type="button" tone="halus" onClick={onClose}>
            Tutup
          </Button>
        </footer>
      </div>
    </div>
  )
}
