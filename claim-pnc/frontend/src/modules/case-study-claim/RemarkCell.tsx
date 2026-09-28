import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'

import { useSaveRemark } from './api'
import type { CaseStudyRow } from './types'

/**
 * Sel **Remark** beserta tombol **Save**-nya.
 *
 * # Kenapa komponen tersendiri
 *
 * Karena ia satu-satunya sel yang punya KEADAAN. Dua puluh tiga kolom lain hanya menggambar
 * apa yang datang dari server; yang ini menyimpan apa yang sedang diketik, tahu apakah ia
 * berbeda dari yang tersimpan, dan punya permintaan sendiri yang dapat gagal.
 *
 * Menaruh keadaan itu di halaman berarti satu peta berisi draf seluruh baris, dan setiap
 * ketikan menggambar ulang dua puluh baris grid sekaligus.
 *
 * # Ia meniru grid Pega, bukan menggantinya dengan dialog
 *
 * `Section/PNCStudyClaim-Section.xml:12814` menyetel sel ini `pyEditOptions=Editable`
 * dengan tombol Save di kolom sebelahnya — penyuntingan terjadi DI DALAM grid, baris demi
 * baris. Menggantinya dengan dialog akan mengubah alur kerja yang sudah dihafal petugas
 * (`D-13`).
 */
export function RemarkCell({ row }: { row: CaseStudyRow }) {
  const [draft, setDraft] = useState(row.remark)
  const save = useSaveRemark()

  // Draf disetel ulang ketika baris yang SAMA datang dengan catatan berbeda — yaitu
  // setelah penyimpanan berhasil dan daftarnya dimuat ulang.
  //
  // Tanpa ini, yang tampil tetap teks yang diketik, bukan teks yang benar-benar tersimpan.
  // Keduanya biasanya sama; yang membedakan justru kasus yang penting — server memangkas
  // spasi di ujung, dan kelak dapat menolak isi tertentu.
  useEffect(() => {
    setDraft(row.remark)
  }, [row.remark])

  const changed = draft.trim() !== row.remark.trim()

  return (
    <div className="flex min-w-[16rem] flex-col gap-1.5">
      <label className="sr-only" htmlFor={`remark-${row.nomor_klaim}`}>
        Remark untuk klaim {row.nomor_klaim}
      </label>
      <textarea
        id={`remark-${row.nomor_klaim}`}
        rows={2}
        value={draft}
        maxLength={2000}
        disabled={save.isPending}
        onChange={(event) => setDraft(event.target.value)}
        className={[
          'w-full rounded-kontrol border border-slate-300 bg-white px-2 py-1.5 text-sm',
          'transition-[border-color,box-shadow] duration-150',
          'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25',
          'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
          'resize-y',
        ].join(' ')}
      />

      <div className="flex items-center gap-2">
        {/*
          Tombol mati saat tidak ada perubahan.

          Bukan sekadar kerapian: penyimpanan ini MENULIS ke tabel klaim milik sistem lama,
          dan menekan Save tanpa perubahan berarti satu penulisan yang tidak dibutuhkan
          siapa pun — beserta satu baris di log yang membuat jejak perubahan yang
          sesungguhnya lebih sulit dicari.
        */}
        <Button
          tone={changed ? 'utama' : 'kedua'}
          disabled={!changed || save.isPending}
          onClick={() =>
            save.mutate({ nomor_klaim: row.nomor_klaim, catatan: draft.trim() })
          }
        >
          {save.isPending ? 'Menyimpan…' : 'Save'}
        </Button>

        {save.isSuccess && !changed && (
          <span className="text-xs text-emerald-700" role="status">
            Tersimpan
          </span>
        )}
      </div>

      {save.isError && (
        <p className="text-xs text-red-700" role="alert">
          {messageOf(save.error)}
        </p>
      )}
    </div>
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Catatan gagal disimpan. Coba lagi beberapa saat lagi.'
}
