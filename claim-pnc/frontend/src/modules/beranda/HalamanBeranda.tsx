import { Link } from 'react-router-dom'

<<<<<<< HEAD
import { gunakanKeluar } from '@/modules/masuk/api'
=======
>>>>>>> master
import { DaftarPortal, PemilihPortal } from '@/modules/portal/PemilihPortal'
import { JenisPengguna } from '@/api/tipe'
import { IkonDaftar } from '@/components/Ikon'
import { gunakanSesi } from '@/app/sesi'

/**
 * Beranda sementara.
 *
 * Kerangka portal yang sebenarnya — navigasi samping, jejak lokasi, dan peta rute dari
 * 74 harness — adalah lingkup TKT-U1-001 dan TKT-U1-004; yang terakhir masih terhalang
 * 7 harness yang hilang dari export. Halaman ini membuktikan dua hal: sesi yang
 * diterbitkan benar-benar dikenali server, dan daftar portal terbaca dari
 * POOLDATA.M_PORTAL_PNC.
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
export function HalamanBeranda() {
  const pengguna = gunakanSesi((keadaan) => keadaan.pengguna)

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          {pengguna ? `Selamat datang, ${pengguna.nama}.` : 'Anda sudah masuk.'}
        </h1>
        <p className="mt-1.5 text-sm text-slate-600">
          Pilih menu di atas untuk mulai bekerja.
        </p>
      </header>

      <div className="grid gap-6 lg:grid-cols-3">
        <section className="lg:col-span-2">
          <PintasanMenu />
        </section>

        {pengguna && (
          <section className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
            <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Identitas Anda
            </h2>
            <dl className="mt-4 space-y-3.5">
              <Baris
                label={pengguna.jenis === JenisPengguna.karyawan ? 'NIK' : 'ID Login'}
                nilai={pengguna.identitas}
                mono
              />
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

            {/* Pemilih portal disembunyikan dari bilah atas pada layar sempit karena
                ruangnya habis; di sini ia tetap terjangkau. */}
            <div className="mt-5 border-t border-slate-100 pt-5 sm:hidden">
              <PemilihPortal />
            </div>
          </section>
        )}
      </div>

      <section className="mt-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
        <DaftarPortal />
      </section>

<<<<<<< HEAD
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
=======
      <p className="mt-6 rounded-kartu border border-slate-200 bg-slate-100/70 p-4 text-sm text-slate-600">
        Menu di atas belum mengikuti izin peran. Daftar 22 peran beserta 51 izin menunya
        adalah TKT-F3-004 — masih menunggu daftar penugasan operator per peran dari DBA
        dan Work Owner.
>>>>>>> master
      </p>
    </div>
  )
}

/**
 * Pintasan ke modul yang sudah dapat dipakai.
 *
 * Sengaja hanya memuat yang benar-benar ada. Kartu untuk modul yang belum dibangun akan
 * terlihat seperti janji, dan pengguna yang menekannya menemukan halaman kosong.
 */
function PintasanMenu() {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Link
        to="/master/status-klaim"
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
          <IkonDaftar className="h-5 w-5" />
        </span>
        <span>
          <span className="block text-sm font-semibold text-slate-900">Master Status Klaim</span>
          <span className="mt-1 block text-sm text-slate-600">
            33 keadaan bisnis sebuah klaim — Register, Claim Committee, Paid, dan seterusnya.
          </span>
        </span>
      </Link>

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

function Baris({ label, nilai, mono = false }: { label: string; nilai: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={`mt-0.5 text-sm text-slate-900 ${mono ? 'font-mono' : ''}`}>{nilai}</dd>
    </div>
  )
}
