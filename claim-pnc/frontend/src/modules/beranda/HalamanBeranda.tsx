import { Link } from 'react-router-dom'

import { gunakanKeluar } from '@/modules/masuk/api'
import { DaftarPortal, PemilihPortal } from '@/modules/portal/PemilihPortal'
import { JenisPengguna } from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

/**
 * Beranda sementara.
 *
 * Kerangka portal yang sebenarnya — navigasi samping, jejak lokasi, dan peta rute dari
 * 74 harness — adalah lingkup TKT-U1-001 dan TKT-U1-004; yang terakhir masih terhalang
 * 7 harness yang hilang dari export. Halaman ini membuktikan dua hal: sesi yang
 * diterbitkan benar-benar dikenali server, dan daftar portal terbaca dari
 * POOLDATA.M_PORTAL_PNC.
 */
export function HalamanBeranda() {
  const pengguna = gunakanSesi((keadaan) => keadaan.pengguna)
  const keluar = gunakanKeluar()

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Claim PNC</h1>
          <p className="text-sm text-slate-600">
            {pengguna ? `Selamat datang, ${pengguna.nama}.` : 'Anda sudah masuk.'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <PemilihPortal />
          <button
            type="button"
            onClick={() => keluar.mutate()}
            disabled={keluar.isPending}
            className="rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {keluar.isPending ? 'Keluar…' : 'Keluar'}
          </button>
        </div>
      </header>

      {pengguna && (
        <section className="mt-6">
          <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Identitas Anda
          </h2>
          <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-2">
            <Baris
              label={pengguna.jenis === JenisPengguna.karyawan ? 'NIK' : 'ID Login'}
              nilai={pengguna.identitas}
            />
            <Baris label="Nama" nilai={pengguna.nama} />
            <Baris
              label="Jenis pengguna"
              nilai={
                pengguna.jenis === JenisPengguna.karyawan
                  ? 'Karyawan'
                  : 'Non-karyawan (broker / surveyor independen)'
              }
            />
            <Baris label="Login" nilai={pengguna.login} />
            {/* Dua field berikut hanya terisi untuk karyawan: POOLDATA.M_LOGIN_PNC
                tidak memuat surel maupun perusahaan. */}
            {pengguna.email && <Baris label="Email" nilai={pengguna.email} />}
            {pengguna.perusahaan && <Baris label="Perusahaan" nilai={pengguna.perusahaan} />}
          </dl>
        </section>
      )}

      <section className="mt-8">
        <DaftarPortal />
      </section>

      {/*
        SATU-SATUNYA tautan sementara ke modul yang sudah ada. Ia berdiri sendiri dan
        dapat dihapus tanpa menyentuh modul mana pun: begitu daftar menu berbasis izin
        (TKT-F3-004 dan TKT-U1-001) tersedia, tautan ini digantikan olehnya.
      */}
      <section className="mt-8">
        <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">Modul</h2>
        <Link
          to="/master-rekening"
          className="mt-3 inline-block rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
        >
          Master Rekening
        </Link>
      </section>

      <p className="mt-8 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
        Menu belum tampil di sini. Daftar menu mengikuti izin peran, dan tabel 22 peran
        beserta 51 izin menu adalah TKT-F3-004 — masih menunggu daftar penugasan operator
        per peran dari DBA dan Work Owner.
      </p>
    </div>
  )
}

function Baris({ label, nilai }: { label: string; nilai: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-900">{nilai}</dd>
    </div>
  )
}
