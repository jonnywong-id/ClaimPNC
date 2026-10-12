import { useEffect, useRef, useState } from 'react'

/**
 * Filter kolom grid Pega (`pyGridFiltering` + `pyColumnFiltering`): ikon corong di kepala kolom
 * membuka daftar nilai unik kolom itu dengan kotak centang, tautan "Clear Filter", dan isian
 * "Search Text" yang menyaring daftar nilainya. Baris yang tampil adalah baris yang nilainya
 * dicentang; tanpa centang kolom itu tidak menyaring.
 */

/** ColumnFilters memetakan judul kolom ke nilai yang dicentang. */
export type ColumnFilters = Record<string, string[]>

/** matchesFilters menyatakan satu baris lolos seluruh filter kolom yang aktif. */
export function matchesFilters(values: Record<string, string>, filters: ColumnFilters): boolean {
  return Object.entries(filters).every(([column, picked]) => picked.length === 0 || picked.includes(values[column] ?? ''))
}

export function ColumnFilter({
  column,
  values,
  picked,
  onChange,
}: {
  column: string
  /** Nilai kolom pada seluruh baris (sebelum disaring kolom ini). */
  values: string[]
  picked: string[]
  onChange: (picked: string[]) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const box = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  const unique = [...new Set(values)].sort((a, b) => a.localeCompare(b))
  const word = search.trim().toLowerCase()
  const shown = word ? unique.filter((v) => v.toLowerCase().includes(word)) : unique
  const active = picked.length > 0

  function toggle(v: string) {
    onChange(picked.includes(v) ? picked.filter((x) => x !== v) : [...picked, v])
  }

  return (
    <div ref={box} className="relative inline-block">
      <button
        type="button"
        aria-label={`Filter ${column}`}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className={`ml-1 rounded px-1 text-xs ${active ? 'bg-blue-100 text-blue-700' : 'text-slate-600 hover:bg-slate-200'}`}
      >
        <svg viewBox="0 0 16 16" className="inline h-3 w-3" fill="currentColor" aria-hidden="true">
          <path d="M1 2h14l-5.5 6.5V14l-3-1.5V8.5z" />
        </svg>
      </button>
      {open && (
        <div role="dialog" aria-label={`Filter ${column}`} className="absolute left-0 z-30 mt-1 w-64 rounded border border-slate-300 bg-white p-2 font-normal text-slate-800 shadow-angkat">
          <div className="text-center">
            <button type="button" className="text-sm text-sky-700 underline" onClick={() => onChange([])}>
              Clear Filter
            </button>
          </div>
          <ul className="mt-1 max-h-48 overflow-y-auto text-sm">
            {shown.map((v) => (
              <li key={v}>
                <label className="flex items-center gap-2 px-1 py-0.5">
                  <input type="checkbox" checked={picked.includes(v)} onChange={() => toggle(v)} />
                  <span>{v === '' ? '(kosong)' : v}</span>
                </label>
              </li>
            ))}
            {shown.length === 0 && <li className="px-1 py-0.5 text-xs text-slate-500">Data Tidak Ada</li>}
          </ul>
          <label className="mt-2 block text-xs font-semibold text-slate-700">
            Search Text
            <input
              aria-label={`Search Text ${column}`}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm font-normal"
            />
          </label>
        </div>
      )}
    </div>
  )
}
