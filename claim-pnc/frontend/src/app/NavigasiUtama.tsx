import { NavLink } from 'react-router-dom'

import { menuUtama } from './menu'

/**
 * NavigasiUtama menampilkan peta menu aplikasi.
 *
 * # Bentuknya berubah menurut lebar layar
 *
 * Di layar lebar ia kolom di sebelah kiri; di layar sempit ia deret mendatar yang dapat
 * digulir di bagian atas. Dua bentuk, bukan satu yang dipaksakan, karena kolom samping
 * pada lebar ponsel akan memakan hampir separuh layar — dan `D-12` menetapkan surveyor
 * memakai tablet dan ponsel di lapangan.
 *
 * Judul kelompok disembunyikan pada layar sempit: di dalam deret mendatar ia justru
 * memutus alurnya. Butirnya tetap terlihat seluruhnya, hanya kehilangan pengelompokannya.
 *
 * # Yang ditandai aktif
 *
 * `NavLink` menandai butir yang jalurnya sedang dibuka, dan penandanya bukan hanya warna
 * melainkan juga `aria-current` — pengguna pembaca layar perlu tahu ia sedang di mana,
 * dan warna saja tidak menyampaikan itu.
 *
 * Butir Beranda memakai `end` supaya ia tidak ikut aktif pada setiap jalur yang dimulai
 * dengan "/" — yaitu semua jalur.
 */
export function NavigasiUtama() {
  return (
    <nav
      aria-label="Menu utama"
      className="border-b border-slate-200 bg-slate-50 md:w-60 md:shrink-0 md:border-b-0 md:border-r"
    >
      <div className="flex gap-4 overflow-x-auto px-3 py-2 md:flex-col md:gap-3 md:overflow-visible md:px-3 md:py-4">
        {menuUtama.map((kelompok, indeks) => (
          <div key={kelompok.judul ?? `kelompok-${indeks}`} className="shrink-0">
            {kelompok.judul && (
              <h2 className="mb-1 hidden px-2 text-xs font-medium uppercase tracking-wide text-slate-500 md:block">
                {kelompok.judul}
              </h2>
            )}
            <ul className="flex gap-1 md:flex-col md:gap-0.5">
              {kelompok.butir.map((butir) => (
                <li key={butir.jalur} className="shrink-0">
                  <NavLink
                    to={butir.jalur}
                    end={butir.jalur === '/'}
                    className={({ isActive }) =>
                      'block whitespace-nowrap rounded px-3 py-2 text-sm ' +
                      (isActive
                        ? 'bg-slate-900 font-medium text-white'
                        : 'text-slate-700 hover:bg-slate-200')
                    }
                  >
                    {butir.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      {/* Keterbatasan yang sedang berlaku disebut di tempat ia paling mudah disalahpahami.
          Penguji bisnis yang melihat menu lengkap dapat mengira izin sudah ditegakkan di
          antarmuka — dan itu justru cacat sistem lama yang tidak boleh diulang.
          Ditampilkan hanya di layar lebar supaya deret mendatar tetap ringkas. */}
      <p className="hidden px-5 pb-4 text-xs leading-relaxed text-slate-500 md:block">
        Daftar menu masih tetap, belum disaring izin peran (<code>TKT-F3-004</code>).
        Kewenangan tetap diperiksa di server pada setiap permintaan.
      </p>
    </nav>
  )
}
