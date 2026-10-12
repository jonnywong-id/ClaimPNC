import { Link } from 'react-router-dom'

import { PortalList, PortalPicker } from '@/modules/portal/PortalPicker'
import { UserKind } from '@/api/types'
import { CardIcon, ListIcon, ReloadIcon } from '@/components/Icon'
import { useSession } from '@/app/session'

/**
 * Beranda.
 *
 * Navigasi samping sudah ada dan isinya dibaca dari POOLDATA.M_MENU_APLIKASI_PNC — lihat
 * `app/Sidebar.tsx`. Yang masih menjadi lingkup TKT-U1-004 adalah peta rute lengkap ke
 * 74 harness, dan itu masih terhalang 7 harness yang hilang dari export.
 *
 * Halaman ini karena itu bukan lagi penampung menu, melainkan titik mulai: sesi yang
 * diterbitkan benar-benar dikenali server, daftar portal terbaca dari
 * POOLDATA.M_PORTAL_PNC, dan modul yang sudah punya layar dapat dicapai satu ketukan.
 *
 * # Yang PINDAH ke bilah atas pada penataan ulang 2026-09-17
 *
 * Tombol Keluar, pemilih portal, dan nama pengguna. Ketiganya berlaku untuk seluruh
 * layar di balik sesi, bukan hanya beranda — sebelumnya pengguna yang sedang membuka
 * layar master tidak punya cara keluar tanpa kembali ke sini lebih dulu.
 *
 * Baris "Nama" pada kartu identitas ikut dihapus karena bilah atas sudah menampilkannya;
 * menyisakannya berarti nama yang sama muncul dua kali di satu layar.
 */
export function HomePage() {
  const pengguna = useSession((state) => state.user)

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          {pengguna ? `Selamat datang, ${pengguna.nama}.` : 'Anda sudah masuk.'}
        </h1>
        <p className="mt-1.5 text-sm text-slate-600">
          Pilih menu di samping kiri untuk mulai bekerja.
        </p>
      </header>

      <div className="grid gap-6 lg:grid-cols-3">
        <section className="lg:col-span-2">
          <MenuShortcut />
        </section>

        {pengguna && (
          <section className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
            <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Identitas Anda
            </h2>
            <dl className="mt-4 space-y-3.5">
              <Row
                label={pengguna.jenis === UserKind.karyawan ? 'NIK' : 'ID Login'}
                value={pengguna.identitas}
                mono
              />
              <Row
                label="Jenis pengguna"
                value={
                  pengguna.jenis === UserKind.karyawan
                    ? 'Karyawan'
                    : 'Non-karyawan (broker / surveyor independen)'
                }
              />
              <Row label="Login" value={pengguna.login} />
              {/* Dua field berikut hanya terisi untuk karyawan: POOLDATA.M_LOGIN_PNC
                  tidak memuat surel maupun perusahaan. */}
              {pengguna.email && <Row label="Email" value={pengguna.email} />}
              {pengguna.perusahaan && <Row label="Perusahaan" value={pengguna.perusahaan} />}
            </dl>

            {/* Pemilih portal disembunyikan dari bilah atas pada layar sempit karena
                ruangnya habis; di sini ia tetap terjangkau. */}
            <div className="mt-5 border-t border-slate-100 pt-5 sm:hidden">
              <PortalPicker />
            </div>
          </section>
        )}
      </div>

      <section className="mt-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
        <PortalList />
      </section>

      <p className="mt-6 rounded-kartu border border-slate-200 bg-slate-100/70 p-4 text-sm text-slate-600">
        Menu di samping kiri sudah disaring POOLDATA.M_OTORISASI_PNC terhadap login Anda.
        Yang belum ada adalah pemeriksaan kewenangan menu di server pada setiap endpoint
        (TKT-F3-005) — sampai itu ada, tautan yang dibuka langsung lewat alamat peramban
        tidak tertahan meski menunya tidak tampil.
      </p>
    </div>
  )
}

/**
 * Modul yang layarnya SUDAH ada, beserta rutenya.
 *
 * Rutenya ditulis sama persis dengan yang didaftarkan `AppRoute` dan dipetakan
 * `app/menu/registry.ts`. Daftar ini sengaja terpisah dari peta menu server: server
 * memutuskan butir menu mana yang boleh DILIHAT, halaman ini memajang yang sudah punya
 * layar — dan keduanya memang dapat berbeda.
 */
const SHORTCUTS = [
  {
    to: '/master/status-klaim',
    Icon: ListIcon,
    title: 'Master Status Klaim',
    description:
      '33 keadaan bisnis sebuah klaim — Register, Claim Committee, Paid, dan seterusnya.',
  },
  {
    to: '/master/rekening',
    Icon: CardIcon,
    title: 'Master Rekening',
    description:
      'Rekening tujuan pembayaran klaim, beserta antrean persetujuan komitenya.',
  },
  {
    to: '/master/status-progres-1',
    Icon: ReloadIcon,
    title: 'Master Status Progres 1',
    description: 'Tahapan progres yang dilalui klaim, dicatat terpisah dari alur kerja.',
  },
] as const

/**
 * Pintasan ke modul yang sudah dapat dipakai.
 *
 * Sengaja hanya memuat yang benar-benar ada. Kartu untuk modul yang belum dibangun akan
 * terlihat seperti janji, dan pengguna yang menekannya menemukan halaman kosong.
 */
function MenuShortcut() {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {SHORTCUTS.map(({ to, Icon, title, description }) => (
        <Link
          key={to}
          to={to}
          className={[
            'group flex flex-col gap-3 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut',
            'transition-[box-shadow,transform,border-color] duration-200 ease-halus',
            'hover:-translate-y-0.5 hover:border-blue-200 hover:shadow-angkat',
            'active:translate-y-0 active:shadow-lembut',
            'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/25',
          ].join(' ')}
        >
          <span
            aria-hidden="true"
            className={[
              'flex h-10 w-10 items-center justify-center rounded-kontrol bg-blue-50 text-blue-600',
              'transition-colors duration-200 ease-halus',
              'group-hover:bg-blue-600 group-hover:text-white',
            ].join(' ')}
          >
            <Icon className="h-5 w-5" />
          </span>
          <span>
            <span className="block text-sm font-semibold text-slate-900">{title}</span>
            <span className="mt-1 block text-sm text-slate-600">{description}</span>
          </span>
        </Link>
      ))}

      <div className="flex flex-col justify-center gap-2 rounded-kartu border border-dashed border-slate-300 bg-slate-50/60 p-5">
        <span className="text-sm font-medium text-slate-600">Modul berikutnya menyusul</span>
        <span className="text-sm text-slate-500">
          Registrasi klaim, komite, akseptasi, dan laporan dikerjakan bertahap sesuai
          urutan gelombang migrasi.
        </span>
      </div>
    </div>
  )
}

function Row({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={`mt-0.5 text-sm text-slate-900 ${mono ? 'font-mono' : ''}`}>{value}</dd>
    </div>
  )
}
