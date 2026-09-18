import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'

import { ListIcon, LogoutIcon, ShieldIcon } from '@/components/Icon'
import { Button } from '@/components/Button'
import { useLogout } from '@/modules/login/api'
import { PortalPicker } from '@/modules/portal/PortalPicker'
import { UserKind } from '@/api/types'
import { useSession } from '@/app/session'

/** Satu entri navigasi. Daftarnya ada di bawah, bukan tersebar di layar-layar. */
type Entry = { ke: string; label: string; icon: ReactNode }

/**
 * Menu sementara.
 *
 * Kerangka portal yang sebenarnya — navigasi samping, jejak lokasi, dan peta rute dari
 * 74 harness — adalah `TKT-U1-001` dan `TKT-U1-004`, dan yang terakhir masih terhalang
 * tujuh harness yang hilang dari export.
 *
 * Yang lebih menentukan: `D-59` menetapkan satuan izin adalah MENU, sehingga daftar ini
 * seharusnya datang dari izin peran pengguna — bukan ditulis tetap di sini. Tabel 22
 * peran dan 51 izin menu adalah `TKT-F3-004`, yang dapat dibangun tetapi belum dapat
 * diisi karena penugasan operator ke peran tidak ada di basis data maupun di export.
 *
 * Sampai itu tiba, daftar ini tetap: setiap pengguna yang sudah masuk melihat menu yang
 * sama. Itu keadaan yang sama dengan seluruh aplikasi hari ini, dan dicatat terbuka di
 * docs/keputusan-implementasi.md — bukan disembunyikan sebagai fitur yang seolah sudah
 * berizin.
 */
const menu: Entry[] = [
  { ke: '/', label: 'Beranda', icon: <ShieldIcon className="h-4 w-4" /> },
  { ke: '/master/status-klaim', label: 'Master Status Klaim', icon: <ListIcon className="h-4 w-4" /> },
]

/**
 * PageShell membungkus layar yang berada di balik sesi: bilah atas, menu, dan isi.
 *
 * Ia hidup di `app/` dan bukan di salah satu modul, karena ia milik kerangka aplikasi —
 * modul tidak boleh saling mengimpor, dan menaruh menu di dalam salah satunya akan
 * memaksa modul lain mengimpornya.
 *
 * # Kenapa identitas pengguna dan tombol Keluar ada di sini
 *
 * Keduanya berlaku untuk SELURUH layar di balik sesi, bukan hanya beranda. Sebelumnya
 * keduanya hidup di dalam halaman beranda, sehingga pengguna yang sedang berada di layar
 * master tidak punya cara keluar tanpa kembali ke beranda lebih dulu.
 *
 * Pemindahan itu juga yang membuat keduanya tidak kembar: satu tombol Keluar di seluruh
 * aplikasi, satu tempat nama pengguna ditampilkan.
 */
export function PageShell({ anak }: { anak: ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-50">
      <TopBar />
      <main className="pb-16">{anak}</main>
    </div>
  )
}

function TopBar() {
  const pengguna = useSession((state) => state.user)
  const logout = useLogout()

  return (
    /*
      `sticky` supaya menu tetap terjangkau pada tabel panjang — layar master dapat
      memuat puluhan baris, dan menggulir kembali ke atas hanya untuk berpindah menu
      adalah gesekan yang tidak perlu.

      `backdrop-blur` membuat isi yang lewat di belakangnya tetap terbaca samar, sehingga
      bilahnya terasa melayang, bukan menutup.
    */
    <header className="sticky top-0 z-30 border-b border-slate-200 bg-white/85 shadow-lembut backdrop-blur-md">
      <div className="mx-auto max-w-6xl px-4">
        <div className="flex h-16 items-center justify-between gap-4">
          <Brand />

          <div className="flex items-center gap-2 sm:gap-3">
            <div className="hidden sm:block">
              <PortalPicker />
            </div>

            {pengguna && <UserChip nama={pengguna.nama} jenis={pengguna.jenis} identitas={pengguna.identitas} />}

            <Button
              tone="halus"
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
              aria-label="Keluar"
            >
              <LogoutIcon className="h-4 w-4" />
              {/*
                Teks tombol disembunyikan pada layar sempit, tetapi TETAP di DOM —
                `sr-only`, bukan dihapus. Tombol yang hanya berisi ikon tanpa teks tidak
                dapat dijelaskan pembaca layar, dan `aria-label` di atas menjaganya
                tetap bernama "Keluar" apa pun lebar layarnya.
              */}
              <span className="sr-only sm:not-sr-only">Keluar</span>
            </Button>
          </div>
        </div>

        <Navigation />
      </div>
    </header>
  )
}

function Brand() {
  return (
    <div className="flex min-w-0 items-center gap-3">
      {/*
        Lambang memakai gradien, satu-satunya di aplikasi ini. Dipakai sekali saja
        supaya ia menjadi penanda, bukan hiasan yang berulang.
      */}
      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-kontrol bg-gradient-to-br from-blue-500 to-blue-700 text-white shadow-aksen">
        <ShieldIcon className="h-5 w-5" />
      </span>
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold leading-tight text-slate-900">Claim PNC</p>
        <p className="truncate text-xs leading-tight text-slate-500">Asuransi Sinar Mas</p>
      </div>
    </div>
  )
}

/**
 * ChipPengguna menampilkan siapa yang sedang masuk.
 *
 * Inisial dipakai sebagai avatar karena aplikasi ini tidak punya foto pengguna — dan
 * tidak akan punya: HCC/HCQ tidak mengirimkannya, dan menambahkan unggahan foto berarti
 * menyimpan data pribadi tanpa alasan bisnis.
 */
function UserChip({ nama, jenis, identitas }: { nama: string; jenis: string; identitas: string }) {
  return (
    <div className="flex items-center gap-2.5 rounded-kontrol border border-slate-200 bg-white py-1.5 pl-1.5 pr-3 shadow-lembut">
      <span
        aria-hidden="true"
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-xs font-semibold text-blue-700"
      >
        {initials(nama)}
      </span>
      <div className="hidden min-w-0 sm:block">
        {/*
          Nama berdiri sendiri di satu elemen. Uji beranda mencarinya dengan pencocokan
          persis, sehingga menggabungkannya dengan teks lain di elemen yang sama akan
          membuatnya tidak ditemukan.
        */}
        <p className="max-w-[10rem] truncate text-sm font-medium leading-tight text-slate-900">{nama}</p>
        <p className="max-w-[10rem] truncate text-xs leading-tight text-slate-500">
          {jenis === UserKind.karyawan ? `NIK ${identitas}` : identitas}
        </p>
      </div>
    </div>
  )
}

/** Mengambil paling banyak dua huruf pertama dari nama sebagai inisial. */
function initials(nama: string): string {
  const word = nama.trim().split(/\s+/).filter(Boolean)
  const start = word[0]?.[0] ?? '?'
  const end = word.length > 1 ? (word[word.length - 1]?.[0] ?? '') : ''
  return (start + end).toUpperCase()
}

function Navigation() {
  return (
    /*
      Menu digulir menyamping pada layar sempit alih-alih dilipat menjadi tombol
      hamburger. Dengan dua entri, hamburger justru menambah satu ketukan untuk
      menyembunyikan sesuatu yang sebenarnya muat. Bila menunya kelak berasal dari izin
      peran dan bertambah banyak, keputusan ini perlu ditinjau ulang.
    */
    <nav aria-label="Menu utama" className="-mb-px overflow-x-auto">
      <ul className="flex min-w-max items-center gap-1 pb-0">
        {menu.map((entri) => (
          <li key={entri.ke}>
            <NavLink
              to={entri.ke}
              end={entri.ke === '/'}
              className={({ isActive }) =>
                [
                  'flex items-center gap-2 border-b-2 px-3 py-3 text-sm font-medium',
                  'transition-[color,border-color,background-color] duration-150 ease-halus',
                  'focus:outline-none focus-visible:rounded-t focus-visible:ring-2 focus-visible:ring-blue-500/50',
                  isActive
                    ? 'border-blue-600 text-blue-700'
                    : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
                ].join(' ')
              }
            >
              {entri.icon}
              {entri.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}
