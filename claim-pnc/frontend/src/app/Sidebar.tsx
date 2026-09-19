import { useEffect, useMemo, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'

import type { MenuItem } from '@/api/types'
import { ChevronIcon, ShieldIcon } from '@/components/Icon'

import { useMenu } from './menu/api'
import { routeFor } from './menu/registry'

/** Beranda tidak ada di M_MENU_APLIKASI_PNC — lihat catatan di Sidebar. */
const HOME_PATH = '/'

type Props = {
  /** Dipanggil setelah pengguna memilih sebuah butir; dipakai menutup laci di layar sempit. */
  onNavigate?: (() => void) | undefined
}

/**
 * Sidebar menampilkan peta menu aplikasi di sisi kiri.
 *
 * # Isinya datang dari basis data, bukan dari kode
 *
 * Susunannya dibaca dari POOLDATA.M_MENU_APLIKASI_PNC — berurutan menurut MENU_SEQUENCE,
 * bersarang menurut MENU_ID_LEADER — dan disaring POOLDATA.M_OTORISASI_PNC terhadap
 * login pengguna beserta group yang diikutinya. Menambah menu karena itu adalah
 * menambah BARIS TABEL, bukan menyunting berkas ini.
 *
 * Itu perubahan mendasar dari keadaan sebelumnya, ketika daftar menunya ditulis tetap di
 * dalam `PageShell` dan sama bagi setiap pengguna.
 *
 * # Dua keadaan yang dibedakan, dan kenapa
 *
 *	tidak diotorisasi  → tidak muncul sama sekali
 *	belum ada modulnya → muncul, tidak dapat diklik, bertanda "belum tersedia"
 *
 * Keputusan Work Owner 2026-09-18. Alasannya: kewenangan adalah fakta tentang HAK —
 * menampilkan pintu yang pasti tertutup hanya menawarkan sesuatu yang tidak ada. Modul
 * yang belum dibangun adalah fakta tentang KEMAJUAN — menyembunyikannya membuat 72 dari
 * 75 layar lenyap tanpa jejak, dan pengguna melaporkan menu yang "hilang".
 *
 * # Ini BUKAN kendali akses
 *
 * `D-59` menetapkan penyembunyian menu hanyalah kenyamanan tampilan; yang menggerbang
 * adalah pemeriksaan di server pada setiap endpoint. Sidebar ini tidak menambah maupun
 * mengurangi kewenangan siapa pun.
 */
export function Sidebar({ onNavigate }: Props) {
  const menu = useMenu()
  const location = useLocation()

  const groups = menu.data?.menu ?? []

  // Kelompok yang memuat layar yang sedang dibuka dibentangkan; sisanya terlipat.
  //
  // Tanpa ini, seluruh 80 butir terbentang sekaligus dan pengguna harus menggulir jauh
  // untuk menemukan kelompok berikutnya — kelompok MASTER saja berisi 39 butir.
  const activeGroupID = useMemo(() => {
    for (const group of groups) {
      for (const item of group.submenu) {
        if (routeFor(item.program) === location.pathname) return group.id
      }
    }
    return null
  }, [groups, location.pathname])

  const [openID, setOpenID] = useState<number | null>(null)

  // Kelompok aktif dibentangkan setiap kali layar berpindah ke kelompok lain. Ia
  // disimpan sebagai state supaya pengguna tetap dapat membuka kelompok lain untuk
  // menelusuri, tanpa pilihannya langsung dibatalkan render berikutnya.
  useEffect(() => {
    if (activeGroupID !== null) setOpenID(activeGroupID)
  }, [activeGroupID])

  return (
    <nav
      aria-label="Menu utama"
      className="flex h-full min-h-0 flex-col gap-1 overflow-y-auto px-3 py-4"
    >
      <NavLink
        to={HOME_PATH}
        end
        onClick={onNavigate}
        className={({ isActive }) => entryClass(isActive)}
      >
        <ShieldIcon className="h-4 w-4 shrink-0" />
        Beranda
      </NavLink>

      {/*
        Beranda sengaja TIDAK diambil dari tabel menu: ia bukan pengganti harness Pega
        mana pun, melainkan layar milik aplikasi baru ini. Menambahkannya ke
        M_MENU_APLIKASI_PNC berarti mengarang baris master.
      */}

      {menu.isPending && <p className="px-3 py-2 text-sm text-slate-500">Memuat menu…</p>}

      {menu.isError && (
        // role="status" (sopan), bukan "alert" (memotong): menu yang gagal dimuat
        // adalah keadaan, bukan sesuatu yang harus menyela apa pun yang sedang dibaca
        // pengguna di isi halaman.
        <p role="status" className="px-3 py-2 text-sm text-amber-800">
          Menu tidak dapat dimuat. Muat ulang halaman, lalu coba lagi.
        </p>
      )}

      {menu.isSuccess && groups.length === 0 && (
        <p className="px-3 py-2 text-sm text-slate-500">
          Belum ada menu yang diberikan untuk pengguna ini. Hubungi administrator Claim PNC.
        </p>
      )}

      {groups.map((group) => (
        <MenuGroup
          key={group.id}
          group={group}
          open={openID === group.id}
          onToggle={() => setOpenID(openID === group.id ? null : group.id)}
          onNavigate={onNavigate}
        />
      ))}
    </nav>
  )
}

function MenuGroup({
  group,
  open,
  onToggle,
  onNavigate,
}: {
  group: MenuItem
  open: boolean
  onToggle: () => void
  onNavigate?: (() => void) | undefined
}) {
  const panelID = `menu-group-${group.id}`

  return (
    <div className="mt-2">
      {/*
        Tombol, bukan judul yang dapat diklik: yang dilakukannya adalah membuka dan
        menutup, dan hanya tombol yang dapat dijalankan dengan papan ketik serta
        membawa aria-expanded.
      */}
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        aria-controls={panelID}
        className={[
          'flex w-full items-center justify-between gap-2 rounded-kontrol px-3 py-2',
          'text-xs font-semibold uppercase tracking-wide text-slate-500',
          'transition-colors duration-150 ease-halus hover:bg-slate-100 hover:text-slate-700',
          'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40',
        ].join(' ')}
      >
        {group.nama}
        <ChevronIcon
          className={`h-4 w-4 shrink-0 transition-transform duration-150 ease-halus ${
            open ? 'rotate-90' : ''
          }`}
        />
      </button>

      {open && (
        <ul id={panelID} className="mt-1 flex flex-col gap-0.5">
          {group.submenu.map((item) => (
            <li key={item.id}>
              <MenuEntry item={item} onNavigate={onNavigate} />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function MenuEntry({ item, onNavigate }: { item: MenuItem; onNavigate?: (() => void) | undefined }) {
  const path = routeFor(item.program)

  if (path === null) {
    // Bukan tautan dan bukan tombol: tidak ada yang terjadi bila ditekan, dan elemen
    // yang dapat difokus tetapi tidak melakukan apa pun justru menjebak pengguna papan
    // ketik. Keterangannya ditulis sebagai teks, bukan hanya warna abu-abu — warna saja
    // tidak terbaca pengguna buta warna.
    return (
      <span className="flex items-center justify-between gap-2 rounded-kontrol px-3 py-2 text-sm text-slate-400">
        <span className="truncate">{item.nama}</span>
        <span className="shrink-0 rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-500">
          belum tersedia
        </span>
      </span>
    )
  }

  return (
    <NavLink to={path} onClick={onNavigate} className={({ isActive }) => entryClass(isActive)}>
      <span className="truncate">{item.nama}</span>
    </NavLink>
  )
}

function entryClass(isActive: boolean): string {
  return [
    'flex items-center gap-2 rounded-kontrol px-3 py-2 text-sm',
    'transition-[color,background-color] duration-150 ease-halus',
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40',
    isActive
      ? 'bg-blue-50 font-medium text-blue-700'
      : 'text-slate-700 hover:bg-slate-100 hover:text-slate-900',
  ].join(' ')
}
