import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'

import { SessionWarning } from './SessionWarning'

/**
 * Kerangka halaman di balik sesi: peringatan sesi, navigasi antarmodul, dan isinya.
 *
 * # Kenapa navigasinya di sini dan bukan di beranda
 *
 * Beranda adalah sebuah MODUL, sama seperti registrasi. Menaruh daftar menu di dalamnya
 * berarti satu modul harus tahu keberadaan seluruh modul lain — persis ketergantungan
 * yang aturan susunan frontend larang. Kerangka bukan modul; ia bagian aplikasi, dan di
 * sinilah tempat yang benar untuk menyatukan mereka.
 *
 * # Ini BUKAN menu yang sebenarnya
 *
 * Daftar menu yang mengikuti izin peran adalah `TKT-F3-004` dan `TKT-U1-004`, keduanya
 * masih terhalang artefak. Yang ada di sini adalah dua tautan tetap, dan setiap pengguna
 * melihat keduanya. Kewenangan atas isinya tetap diperiksa server pada setiap endpoint
 * (`ADR-0023`) — tautan yang terlihat bukan tautan yang dibolehkan.
 */
export function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-white">
      <SessionWarning />

      <nav aria-label="Menu utama" className="border-b border-slate-200 bg-slate-50">
        <ul className="mx-auto flex max-w-5xl gap-1 px-4">
          <Menu to="/" label="Beranda" />
          <Menu to="/registrasi" label="Registrasi Klaim" />
        </ul>
      </nav>

      {children}
    </div>
  )
}

function Menu({ to, label }: { to: string; label: string }) {
  return (
    <li>
      <NavLink
        to={to}
        end={to === '/'}
        className={({ isActive }) =>
          isActive
            ? 'inline-block border-b-2 border-slate-900 px-3 py-3 text-sm font-medium text-slate-900'
            : 'inline-block border-b-2 border-transparent px-3 py-3 text-sm text-slate-600 hover:text-slate-900'
        }
      >
        {label}
      </NavLink>
    </li>
  )
}
