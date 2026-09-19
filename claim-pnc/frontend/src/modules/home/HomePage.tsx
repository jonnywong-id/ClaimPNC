import { useLogout } from '@/modules/login/api'
import { PortalList, PortalSelector } from '@/modules/portal/PortalSelector'
import { UserKind } from '@/api/types'
import { useSession } from '@/app/session'

/**
 * Beranda sementara.
 *
 * Kerangka portal yang sebenarnya — navigasi samping, jejak lokasi, dan peta rute dari
 * 74 harness — adalah lingkup TKT-U1-001 dan TKT-U1-004; yang terakhir masih terhalang
 * 7 harness yang hilang dari export. Halaman ini membuktikan dua hal: sesi yang
 * diterbitkan benar-benar dikenali server, dan daftar portal terbaca dari
 * POOLDATA.M_PORTAL_PNC.
 */
export function HomePage() {
  const pengguna = useSession((state) => state.pengguna)
  const logout = useLogout()

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
          <PortalSelector />
          <button
            type="button"
            onClick={() => logout.mutate()}
            disabled={logout.isPending}
            className="rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {logout.isPending ? 'Keluar…' : 'Keluar'}
          </button>
        </div>
      </header>

      {pengguna && (
        <section className="mt-6">
          <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Identitas Anda
          </h2>
          <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-2">
            <Row
              label={pengguna.jenis === UserKind.employee ? 'NIK' : 'ID Login'}
              value={pengguna.identitas}
            />
            <Row label="Nama" value={pengguna.nama} />
            <Row
              label="Jenis pengguna"
              value={
                pengguna.jenis === UserKind.employee
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
        </section>
      )}

      <section className="mt-8">
        <PortalList />
      </section>

      <p className="mt-8 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
        Menu belum tampil di sini. Daftar menu mengikuti izin peran, dan tabel 22 peran
        beserta 51 izin menu adalah TKT-F3-004 — masih menunggu daftar penugasan operator
        per peran dari DBA dan Work Owner.
      </p>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-900">{value}</dd>
    </div>
  )
}
