import { useEffect } from 'react'

import { gunakanPortalTerpilih } from '@/app/portal'

import { gunakanDaftarPortal } from './api'

/**
 * PemilihPortal menampilkan daftar entitas dan portal yang sedang aktif.
 *
 * Nama yang ditampilkan adalah kolom PORTAL_NAME dari POOLDATA.M_PORTAL_PNC — daftarnya
 * data, bukan konstanta di kode (ADR-0030). Menambah entitas berarti menambah baris
 * tabel; layar ini tidak perlu disentuh.
 *
 * Portal yang basis datanya belum siap tetap terlihat tetapi tidak dapat dipilih,
 * supaya pengguna tahu entitas itu direncanakan — bukan mengira ia tidak ada.
 */
export function PemilihPortal() {
  const { data, isPending, isError } = gunakanDaftarPortal()
  const terpilih = gunakanPortalTerpilih((keadaan) => keadaan.alias)
  const pilih = gunakanPortalTerpilih((keadaan) => keadaan.pilih)

  // Portal awal mengikuti portal utama yang disebut server, bukan tebakan di frontend.
  useEffect(() => {
    if (!terpilih && data?.utama) pilih(data.utama)
  }, [terpilih, data?.utama, pilih])

  if (isPending) {
    return (
      <span className="flex items-center gap-2 text-sm text-slate-500">
        <span aria-hidden="true" className="h-2 w-2 animate-pulse rounded-full bg-slate-300" />
        Memuat portal…
      </span>
    )
  }
  if (isError || !data) {
    return <span className="text-sm text-red-700">Daftar portal tidak dapat dimuat.</span>
  }

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="portal" className="text-sm font-medium text-slate-600">
        Portal
      </label>
      <select
        id="portal"
        value={terpilih ?? data.utama}
        onChange={(peristiwa) => pilih(peristiwa.target.value)}
        className={[
          'max-w-[12rem] truncate rounded-kontrol border border-slate-300 bg-white py-2 pl-3 pr-8',
          'text-sm font-medium text-slate-900',
          'transition-[border-color,box-shadow] duration-150 ease-halus',
          'hover:border-slate-400',
          'focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15',
        ].join(' ')}
      >
        {data.portal.map((p) => (
          <option key={p.alias} value={p.alias} disabled={!p.siap}>
            {p.nama}
            {p.siap ? '' : ' — belum tersedia'}
          </option>
        ))}
      </select>
    </div>
  )
}

/** DaftarPortal menampilkan seluruh entitas beserta kesiapannya sebagai tabel ringkas. */
export function DaftarPortal() {
  const { data, isPending, isError } = gunakanDaftarPortal()

  if (isPending) return <p className="text-sm text-slate-500">Memuat daftar portal…</p>
  if (isError || !data) {
    return <p className="text-sm text-red-700">Daftar portal tidak dapat dimuat.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[32rem] border-collapse text-sm">
        <caption className="mb-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          Portal entitas — sumber: POOLDATA.M_PORTAL_PNC
        </caption>
        <thead>
          <tr className="border-b border-slate-200 text-left">
            <th scope="col" className="py-2.5 pr-4 text-xs font-semibold uppercase tracking-wide text-slate-600">
              Nama portal
            </th>
            <th scope="col" className="py-2.5 pr-4 text-xs font-semibold uppercase tracking-wide text-slate-600">
              Alias
            </th>
            <th scope="col" className="py-2.5 text-xs font-semibold uppercase tracking-wide text-slate-600">
              Status
            </th>
          </tr>
        </thead>
        <tbody>
          {data.portal.map((p) => (
            <tr
              key={p.alias}
              className="border-b border-slate-100 transition-colors duration-150 ease-halus last:border-0 hover:bg-slate-50"
            >
              <td className="py-3 pr-4 text-slate-900">{p.nama}</td>
              <td className="py-3 pr-4 font-mono text-slate-600">{p.alias}</td>
              <td className="py-3">
                {/*
                  Kesiapan ditandai warna DAN titik DAN teks. Warna saja tidak terbaca
                  pengguna buta warna, dan "tersedia" versus "belum" adalah pembedaan
                  yang menentukan apakah portalnya dapat dipilih sama sekali.
                */}
                {p.siap ? (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 ring-1 ring-emerald-200">
                    <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                    Tersedia
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
                    <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-slate-400" />
                    Menunggu kredensial basis data
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
