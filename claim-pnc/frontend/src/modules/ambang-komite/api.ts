import { useQuery } from '@tanstack/react-query'

import { panggilAPI } from '@/api/klien'
import type {
  KomiteIntegrityResponse,
  KomiteThresholdListResponse,
  KomiteTieringResponse,
} from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

const MASTER_PATH = '/api/master/ambang-komite'
const TIERING_PATH = '/api/komite/penjenjangan'

/** Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran. */
const keys = {
  list: (token: string | null) => ['ambang-komite', token] as const,
  integrity: (token: string | null) => ['ambang-komite-integritas', token] as const,
  tiering: (token: string | null, value: string, line: string, applicant: string) =>
    ['komite-penjenjangan', token, value, line, applicant] as const,
}

/**
 * Hook daftar master ambang komite.
 *
 * Menggantikan pembacaan `POOLDATA.EMAILKOMITE` yang di sistem lama tersebar di **17
 * kueri berbeda** — `EmailKomiteBerjenjang_sql` beserta varian PA, Travel, Bonding,
 * Simasnet, Adjuster, dan Salvage — masing-masing dengan kombinasi penyaring sendiri.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu keputusan yang diambil
 * dengan angka: isinya 30 baris dan bertambah beberapa baris per tahun. Layar yang
 * datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 */
export function useThresholdList() {
  const token = gunakanSesi((state) => state.token)

  return useQuery({
    queryKey: keys.list(token),
    queryFn: () => panggilAPI<KomiteThresholdListResponse>(MASTER_PATH, { token }),
    enabled: token !== null,
    // Master nyaris tidak pernah berubah dalam satu sesi kerja, dan aplikasi ini bahkan
    // tidak dapat mengubahnya — ia dibaca saja selama masa paralel.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook pemeriksaan integritas master ambang.
 *
 * Tidak ada padanannya di sistem lama: `LIMIT_TOP` di sana tersimpan tetapi tidak pernah
 * dipakai satu kueri pun. `D-47` memberinya peran — memeriksa apakah tangganya tersusun
 * rapi — dan inilah yang menjalankannya.
 */
export function useIntegrity() {
  const token = gunakanSesi((state) => state.token)

  return useQuery({
    queryKey: keys.integrity(token),
    queryFn: () => panggilAPI<KomiteIntegrityResponse>(`${MASTER_PATH}/integritas`, { token }),
    enabled: token !== null,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook perhitungan penjenjangan.
 *
 * Dijalankan hanya ketika `enabled` bernilai benar, supaya layar tidak memanggil server
 * pada setiap huruf yang diketik pengguna di kolom nilai. Yang memicunya adalah tombol
 * Hitung, bukan perubahan isian.
 *
 * `value` sudah berbentuk **kanonik** — hasil `parseRupiah`, bukan apa yang diketik
 * pengguna. Server menolak pemisah ribuan dengan sengaja: artinya berbeda antar bahasa,
 * sehingga penafsirannya terjadi di layar yang tahu bahasanya.
 */
export function useTiering(value: string, line: string, applicant: string, enabled: boolean) {
  const token = gunakanSesi((state) => state.token)

  return useQuery({
    queryKey: keys.tiering(token, value, line, applicant),
    queryFn: () => {
      const params = new URLSearchParams({ nilai: value, lini: line })
      // Penginput hanya berpengaruh pada mode satu-penyetuju, tempat ia dikecualikan
      // supaya tidak menyetujui pengajuannya sendiri. Dikirim hanya bila terisi, supaya
      // alamat halaman tetap bersih pada portal yang tidak memakainya.
      if (applicant !== '') params.set('penginput', applicant)

      return panggilAPI<KomiteTieringResponse>(`${TIERING_PATH}?${params}`, { token })
    },
    enabled: token !== null && enabled && value !== '' && line !== '',
    // Hasilnya murni turunan dari master dan nilai yang diminta; selama keduanya sama,
    // jawabannya tidak akan berubah.
    staleTime: 5 * 60 * 1000,
  })
}
