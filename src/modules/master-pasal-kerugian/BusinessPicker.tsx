import { useId, useState } from 'react'

import type { ClauseBusiness } from '@/api/types'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

import { MIN_LOOKUP_KEYWORD, useClauseBusinessLookup } from './api'

type Props = {
  /** Lini bisnis yang sudah menempel pada pasal ini. */
  value: ClauseBusiness[]
  onChange: (value: ClauseBusiness[]) => void
  disabled?: boolean
}

/**
 * BusinessPicker adalah isian "Bisnis" — daftar lini bisnis tempat sebuah pasal berlaku.
 *
 * # Bentuknya mengikuti layar lama, bukan diringkas menjadi satu dropdown
 *
 * `Section/BrowsePasalDeatailMaster-Section.xml` memakai **repeating grid** atas
 * `TempPasalCol.BISNISID` dengan ikon tambah/hapus baris (`pzPegaDefaultGridIcons`), dan
 * setiap barisnya adalah **autocomplete** ke `POOLDATA.BUSINESS` lewat
 * `Report Definition/BrowseBusiness_RD-RD.xml`. Satu pasal karena itu dapat menyebut
 * banyak lini bisnis, dan urutannya ditentukan pengguna.
 *
 * Dropdown bertanda centang akan lebih ringkas, tetapi ia mengubah cara kerja pengguna —
 * dan `D-13` menetapkan alur dan tata letak ditiru supaya petugas tidak perlu belajar
 * ulang.
 *
 * # Nama yang diketik bebas DITERIMA, dan itu keputusan yang disengaja
 *
 * Autocomplete layar lama ber-`pyAllowFreeFormInput=true`: nama yang tidak ada di master
 * tetap tersimpan, dengan kode kosong. Modul Master Auto Claim memilih sebaliknya — di
 * sana nilai yang tersimpan SELALU berasal dari baris yang dipilih — tetapi keputusan Work
 * Owner 2026-09-19 untuk modul ini adalah "jalankan as is".
 *
 * Yang ditambahkan di sini hanyalah KEJELASANNYA: butir tanpa kode ditandai terang-terangan
 * di layar, sehingga pengguna tahu butir itu tidak terhubung ke master mana pun. Di Pega,
 * butir semacam itu tampak sama persis dengan yang dipilih dari daftar.
 *
 * # Kenapa ia TIDAK di components/
 *
 * Hanya modul ini yang memakainya. Aturan susunan frontend menaikkan kebutuhan bersama ke
 * `components/`, dan yang belum bersama tetap di modulnya — menaikkannya lebih dulu berarti
 * menebak bentuk yang dibutuhkan modul lain sebelum modul itu ada.
 */
export function BusinessPicker({ value, onChange, disabled = false }: Props) {
  const fieldId = useId()
  const [keyword, setKeyword] = useState('')
  const [isOpen, setOpen] = useState(false)

  const clean = keyword.trim()
  const shortKeyword = clean.length > 0 && clean.length < MIN_LOOKUP_KEYWORD
  const lookup = useClauseBusinessLookup(keyword)

  /** Butir yang kodenya sudah dipakai tidak muncul lagi di daftar pilihan. */
  function alreadyPicked(id: string): boolean {
    return id !== '' && value.some((business) => business.id === id)
  }

  function add(business: ClauseBusiness) {
    onChange([...value, business])
    setKeyword('')
    setOpen(false)
  }

  function remove(position: number) {
    onChange(value.filter((_, index) => index !== position))
  }

  const results = (lookup.data?.bisnis ?? []).filter((business) => !alreadyPicked(business.id))

  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">Bisnis</span>

      {value.length === 0 ? (
        <p className="mt-1.5 text-sm text-slate-500">
          Belum ada lini bisnis. Pasal tetap dapat disimpan tanpa ini.
        </p>
      ) : (
        <ul className="mt-1.5 space-y-1.5">
          {value.map((business, position) => (
            <li
              // Kode BOLEH kosong dan BOLEH kembar — tidak ada yang mencegahnya di layar
              // lama — sehingga posisi ikut menjadi kunci. Memakai kode saja akan membuat
              // dua butir bernama sama saling menimpa saat salah satunya dihapus.
              key={`${business.id}-${business.nama}-${position}`}
              className="flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2"
            >
              <span className="min-w-0 flex-1 text-sm text-slate-900">
                {business.nama === '' ? (
                  <span className="text-slate-400">(tanpa nama)</span>
                ) : (
                  business.nama
                )}
                {business.id === '' ? (
                  <span
                    className="ml-2 rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-800 ring-1 ring-amber-200"
                    title="Nama ini diketik bebas dan tidak terhubung ke master lini bisnis."
                  >
                    tanpa kode
                  </span>
                ) : (
                  <span className="ml-2 text-xs text-slate-500">{business.id}</span>
                )}
              </span>
              {!disabled && (
                <Button
                  tone="halus"
                  onClick={() => remove(position)}
                  aria-label={`Hapus ${business.nama === '' ? 'butir tanpa nama' : business.nama} dari daftar bisnis`}
                >
                  Hapus
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      {!disabled && (
        <div className="mt-2">
          <Field
            id={fieldId}
            label="Tambah lini bisnis"
            type="search"
            value={keyword}
            autoComplete="off"
            placeholder="Cari lini bisnis…"
            hint={`Ketik minimal ${MIN_LOOKUP_KEYWORD} huruf, lalu pilih dari daftar.`}
            onChange={(e) => {
              setKeyword(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
          />

          {isOpen && !shortKeyword && clean !== '' && (
            <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
              {lookup.isError ? (
                <p className="px-3 py-3 text-sm text-red-700" role="alert">
                  Pencarian gagal. Coba beberapa saat lagi.
                </p>
              ) : lookup.isFetching ? (
                <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
              ) : (
                <ul>
                  {results.map((business) => (
                    <li key={business.id}>
                      <button
                        type="button"
                        onClick={() => add(business)}
                        className={[
                          'flex w-full items-baseline gap-2 px-3 py-2 text-left text-sm',
                          'transition-colors duration-150 ease-halus',
                          'hover:bg-blue-50 focus:bg-blue-50 focus:outline-none',
                        ].join(' ')}
                      >
                        <span className="min-w-0 flex-1 text-slate-900">{business.nama}</span>
                        <span className="shrink-0 text-xs text-slate-500">{business.id}</span>
                      </button>
                    </li>
                  ))}

                  {/*
                    Pilihan terakhir: menambahkan nama yang diketik apa adanya. Ia padanan
                    `pyAllowFreeFormInput=true` pada autocomplete layar lama, dan ia
                    sengaja diletakkan PALING BAWAH serta diberi keterangan — supaya
                    memilih dari master tetap menjadi jalan yang paling mudah ditempuh.
                  */}
                  <li className="border-t border-slate-200">
                    <button
                      type="button"
                      onClick={() => add({ id: '', nama: clean })}
                      className={[
                        'w-full px-3 py-2 text-left text-sm',
                        'transition-colors duration-150 ease-halus',
                        'hover:bg-amber-50 focus:bg-amber-50 focus:outline-none',
                      ].join(' ')}
                    >
                      <span className="text-slate-900">Tambahkan “{clean}” apa adanya</span>
                      <span className="block text-xs text-slate-500">
                        Tidak terhubung ke master lini bisnis.
                      </span>
                    </button>
                  </li>
                </ul>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
