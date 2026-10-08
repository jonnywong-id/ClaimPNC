import { useId, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

/** Satu butir lini bisnis. Kode BOLEH kosong untuk nama yang diketik bebas. */
export type PickedBusiness = {
  id: string
  nama: string
}

/** Bentuk minimal hasil kueri lookup lini bisnis yang dibaca komponen ini. */
export type BusinessLookupResult = {
  data?: { bisnis: PickedBusiness[] } | undefined
  isError: boolean
  isFetching: boolean
}

type Props = {
  /** Lini bisnis yang sudah menempel pada baris yang sedang disunting. */
  value: PickedBusiness[]
  onChange: (value: PickedBusiness[]) => void
  disabled?: boolean

  /** Panjang minimum kata kunci sebelum pencarian dijalankan. */
  minKeyword: number

  /** Hook kueri lookup lini bisnis milik modul pemanggil. */
  useLookup: (keyword: string) => BusinessLookupResult

  /** Teks saat daftar masih kosong, mis. "Belum ada lini bisnis. Pasal tetap …". */
  emptyText: string
}

/**
 * BusinessListPicker adalah isian "Bisnis" — daftar lini bisnis berulang dengan
 * autocomplete ke `POOLDATA.BUSINESS`, dipakai bersama oleh
 * `detail-penyebab-kerugian/BusinessPicker` dan `master-pasal-kerugian/BusinessPicker`.
 *
 * Nama yang diketik bebas DITERIMA (padanan `pyAllowFreeFormInput=true` layar lama) dan
 * ditandai "tanpa kode" supaya pengguna tahu butir itu tidak terhubung ke master. Alasan
 * rinci per modul tetap dicatat di berkas BusinessPicker masing-masing modul.
 */
export function BusinessListPicker({
  value,
  onChange,
  disabled = false,
  minKeyword,
  useLookup,
  emptyText,
}: Readonly<Props>) {
  const fieldId = useId()
  const [keyword, setKeyword] = useState('')
  const [isOpen, setOpen] = useState(false)

  const clean = keyword.trim()
  const shortKeyword = clean.length > 0 && clean.length < minKeyword
  const lookup = useLookup(keyword)

  /** Butir yang kodenya sudah dipakai tidak muncul lagi di daftar pilihan. */
  function alreadyPicked(id: string): boolean {
    return id !== '' && value.some((business) => business.id === id)
  }

  function add(business: PickedBusiness) {
    onChange([...value, business])
    setKeyword('')
    setOpen(false)
  }

  function remove(position: number) {
    onChange(value.filter((_, index) => index !== position))
  }

  const results = (lookup.data?.bisnis ?? []).filter((business) => !alreadyPicked(business.id))

  // Isi daftar hasil pencarian lini bisnis menurut keadaan kueri.
  function renderResults(): ReactNode {
    if (lookup.isError) {
      return (
        <p className="px-3 py-3 text-sm text-red-700" role="alert">
          Pencarian gagal. Coba beberapa saat lagi.
        </p>
      )
    }
    if (lookup.isFetching) {
      return <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
    }
    return (
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
    )
  }

  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">Bisnis</span>

      {value.length === 0 ? (
        <p className="mt-1.5 text-sm text-slate-500">
          {emptyText}
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
            hint={`Ketik minimal ${minKeyword} huruf, lalu pilih dari daftar.`}
            onChange={(e) => {
              setKeyword(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
          />

          {isOpen && !shortKeyword && clean !== '' && (
            <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
              {renderResults()}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
