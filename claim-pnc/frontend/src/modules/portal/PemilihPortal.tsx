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
    return <span className="text-sm text-slate-500">Memuat portal…</span>
  }
  if (isError || !data) {
    return <span className="text-sm text-red-700">Daftar portal tidak dapat dimuat.</span>
  }

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="portal" className="text-sm font-medium text-slate-700">
        Portal
      </label>
      <select
        id="portal"
        value={terpilih ?? data.utama}
        onChange={(peristiwa) => pilih(peristiwa.target.value)}
        className="rounded border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none"
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
        <caption className="mb-2 text-left text-xs uppercase tracking-wide text-slate-500">
          Portal entitas — sumber: POOLDATA.M_PORTAL_PNC
        </caption>
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-600">
            <th scope="col" className="py-2 pr-4 font-medium">
              Nama portal
            </th>
            <th scope="col" className="py-2 pr-4 font-medium">
              Alias
            </th>
            <th scope="col" className="py-2 font-medium">
              Status
            </th>
          </tr>
        </thead>
        <tbody>
          {data.portal.map((p) => (
            <tr key={p.alias} className="border-b border-slate-100">
              <td className="py-2 pr-4 text-slate-900">{p.nama}</td>
              <td className="py-2 pr-4 text-slate-600">{p.alias}</td>
              <td className="py-2">
                {p.siap ? (
                  <span className="text-emerald-700">Tersedia</span>
                ) : (
                  <span className="text-slate-500">Menunggu kredensial basis data</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
