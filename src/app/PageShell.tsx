import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'

import { CloseIcon, LogoutIcon, MenuIcon, ShieldIcon, SidebarIcon } from '@/components/Icon'
import { Button } from '@/components/Button'
import { useLogout } from '@/modules/login/api'
import { PortalPicker } from '@/modules/portal/PortalPicker'
import { UserKind } from '@/api/types'
import { useSession } from '@/app/session'

import { Sidebar, type GroupRequest } from './Sidebar'

/** Kunci penyimpanan peramban untuk pilihan menu diperkecil. */
const COLLAPSED_KEY = 'claimpnc.menu-diperkecil'

/**
 * readCollapsed membaca pilihan menu diperkecil yang diingat peramban.
 *
 * Bila penyimpanan tidak dapat dibaca, menu dibentangkan — keadaan bawaan yang paling
 * jelas bagi pengguna baru.
 */
function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

/**
 * PageShell membungkus layar yang berada di balik sesi: bilah atas, menu kiri, dan isi.
 *
 * Ia hidup di `app/` dan bukan di salah satu modul, karena ia milik kerangka aplikasi —
 * modul tidak boleh saling mengimpor, dan menaruh menu di dalam salah satunya akan
 * memaksa modul lain mengimpornya.
 *
 * # Menu pindah ke kiri, dan isinya kini datang dari basis data
 *
 * Sebelumnya daftar menu ditulis TETAP di berkas ini — tiga butir, sama bagi setiap
 * pengguna — dengan catatan bahwa ia seharusnya datang dari izin pengguna begitu
 * tabelnya tersedia. Tabelnya sudah tersedia: sejak 2026-09-19 menu dibaca dari
 * POOLDATA.M_MENU_APLIKASI_PNC dan disaring M_OTORISASI_PNC. Lihat `Sidebar`; berkas ini
 * tinggal menyediakan tempatnya.
 *
 * Bentuk kolom kiri dipilih karena jumlahnya: 80 butir dalam empat kelompok tidak muat
 * sebagai deret mendatar, dan memaksanya ke sana akan mengubah menu menjadi sesuatu yang
 * harus digulir menyamping untuk ditelusuri.
 *
 * # Kenapa identitas pengguna dan tombol Keluar tetap di bilah atas
 *
 * Keduanya berlaku untuk SELURUH layar di balik sesi dan bukan bagian dari peta menu.
 * Menaruhnya di dalam kolom menu akan membuat keduanya ikut tergulir bersama 80 butir.
 */
export function PageShell({ children }: { children: ReactNode }) {
  // Laci menu untuk layar sempit. `D-12` menetapkan surveyor memakai tablet dan ponsel
  // di lapangan, dan kolom selebar 16rem akan memakan hampir separuh layar ponsel.
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Menu kiri dapat DIPERKECIL menjadi rel ikon (permintaan Work Owner 2026-09-28),
  // memberi ruang lebih bagi grid lebar. Pilihannya diingat per peramban — kenyamanan
  // pribadi, bukan data yang perlu dibagikan.
  const [collapsed, setCollapsed] = useState(readCollapsed)
  const [requestedGroup, setRequestedGroup] = useState<GroupRequest | null>(null)

  function changeCollapsed(value: boolean) {
    setCollapsed(value)
    try {
      window.localStorage.setItem(COLLAPSED_KEY, value ? '1' : '0')
    } catch {
      // Penyimpanan peramban dapat ditolak (mode privat, kebijakan kantor). Menu tetap
      // berfungsi; hanya pilihannya tidak diingat.
    }
  }

  // Wadah gulir isi halaman. Posisi gulirnya dikembalikan ke atas setiap berpindah
  // halaman — sebelumnya jendela yang digulir, dan halaman baru terbuka di tengah bila
  // halaman sebelumnya sedang tergulir ke bawah.
  const scrollArea = useRef<HTMLDivElement>(null)
  const { pathname } = useLocation()
  useEffect(() => {
    scrollArea.current?.scrollTo?.({ top: 0 })
  }, [pathname])

  return (
    /*
      Isi halaman TIDAK lagi lewat di belakang bilah atas (permintaan Work Owner
      2026-09-29). Yang digulir kini wadah di BAWAH bilah atas, bukan jendela: batang
      gulirnya mulai di bawah bilah atas, dan baris tabel tidak pernah tampak samar di
      balik bilah yang tembus pandang.

      h-screen + pt-16 (border-box) menyisakan tepat tinggi layar dikurangi bilah atas
      untuk wadah gulir `flex-1`.
    */
    <div className="flex h-screen flex-col bg-slate-50 pt-16">
      <TopBar onOpenMenu={() => setDrawerOpen(true)} />

      <div ref={scrollArea} className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex max-w-[100rem]">
          {/* Kolom menu tetap pada layar lebar. `sticky` membuatnya tinggal di tempat saat
            isi halaman digulir — pada tabel panjang, menggulir kembali ke atas hanya
            untuk berpindah menu adalah gesekan yang tidak perlu. `top-0` kini diukur
            dari wadah gulir, yang sudah berada di bawah bilah atas. */}
          <aside
            className={[
              'sticky top-0 hidden h-[calc(100vh-4rem)] shrink-0 flex-col border-r border-slate-200 bg-white lg:flex',
              'transition-[width] duration-200 ease-halus',
              collapsed ? 'w-16' : 'w-64',
            ].join(' ')}
          >
            <div
              className={['flex px-3 pt-3', collapsed ? 'justify-center' : 'justify-end'].join(' ')}
            >
              <button
                type="button"
                onClick={() => changeCollapsed(!collapsed)}
                aria-label={collapsed ? 'Perbesar menu' : 'Perkecil menu'}
                aria-expanded={!collapsed}
                title={collapsed ? 'Perbesar menu' : 'Perkecil menu'}
                className={[
                  'flex h-9 w-9 items-center justify-center rounded-kontrol border border-slate-200 text-slate-600',
                  'transition-colors duration-150 ease-halus hover:bg-slate-100 hover:text-slate-900',
                  'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40',
                ].join(' ')}
              >
                <SidebarIcon className="h-5 w-5" />
              </button>
            </div>
            <div className="min-h-0 flex-1">
              <Sidebar
                collapsed={collapsed}
                requestedGroup={requestedGroup}
                onExpandGroup={(id) => {
                  changeCollapsed(false)
                  setRequestedGroup((previous) => ({ id, seq: (previous?.seq ?? 0) + 1 }))
                }}
              />
            </div>
          </aside>

          {drawerOpen && <MenuDrawer onClose={() => setDrawerOpen(false)} />}

          {/* min-w-0 mencegah tabel lebar memaksa seluruh halaman melebar — tanpa itu,
            gulir mendatar milik DataTable tidak berfungsi. */}
          <main className="min-w-0 flex-1 pb-16">{children}</main>
        </div>
      </div>
    </div>
  )
}

/**
 * MenuDrawer menampilkan menu yang sama sebagai laci di layar sempit.
 *
 * Isinya `Sidebar` yang sama persis, bukan salinan: dua daftar menu yang berbeda akan
 * berbeda isinya cepat atau lambat.
 */
function MenuDrawer({ onClose }: { onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-40 lg:hidden">
      {/* Latar gelap ikut menutup laci saat ditekan. Ia `aria-hidden` dan tidak dapat
          difokus: penutup yang sama sudah tersedia sebagai tombol bernama di dalam
          lacinya, sehingga pengguna papan ketik tidak kehilangan apa pun. */}
      <div
        aria-hidden="true"
        onClick={onClose}
        className="absolute inset-0 bg-slate-900/40 backdrop-blur-[2px]"
      />
      <div className="absolute inset-y-0 left-0 flex w-72 max-w-[85%] flex-col bg-white shadow-angkat">
        <div className="flex items-center justify-between border-b border-slate-200 px-3 py-3">
          <span className="text-sm font-semibold text-slate-900">Menu</span>
          <Button tone="halus" onClick={onClose} aria-label="Tutup menu">
            <CloseIcon className="h-4 w-4" />
          </Button>
        </div>
        <div className="min-h-0 flex-1">
          <Sidebar onNavigate={onClose} />
        </div>
      </div>
    </div>
  )
}

function TopBar({ onOpenMenu }: { onOpenMenu: () => void }) {
  const user = useSession((state) => state.user)
  const logout = useLogout()

  return (
    /*
      `fixed`, bukan `sticky` (permintaan Work Owner 2026-09-28: bilah atas tetap di
      tempat saat layar digulir). `sticky` berhenti bekerja begitu salah satu wadah di
      atasnya memakai overflow; `fixed` tidak bergantung pada wadah apa pun. Tempatnya
      dicadangkan `pt-16` pada PageShell, dan sejak 2026-09-29 isi halaman digulir di
      wadah tersendiri di bawahnya sehingga tidak pernah lewat di belakang bilah ini.
    */
    <header className="fixed inset-x-0 top-0 z-30 h-16 border-b border-slate-200 bg-white/85 shadow-lembut backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-[100rem] items-center justify-between gap-4 px-4">
        <div className="flex min-w-0 items-center gap-3">
          <Button tone="halus" onClick={onOpenMenu} aria-label="Buka menu" className="lg:hidden">
            <MenuIcon className="h-4 w-4" />
          </Button>
          <Brand />
        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          <div className="hidden sm:block">
            <PortalPicker />
          </div>

          {user && <UserChip nama={user.nama} jenis={user.jenis} identitas={user.identitas} />}

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
 * UserChip menampilkan siapa yang sedang masuk.
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
        <p className="max-w-[10rem] truncate text-sm font-medium leading-tight text-slate-900">
          {nama}
        </p>
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
