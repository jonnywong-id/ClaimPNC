import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { panggilAPI } from '@/api/klien'
import type { ResponsDaftarStatusKlaim, ResponsStatusKlaim } from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

const JALUR = '/api/master/status-klaim'

/** Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran. */
const kunci = {
  daftar: (token: string | null) => ['master-status-klaim', token] as const,
}

/**
 * Hook daftar Master Status Klaim.
 *
 * Menggantikan Report Definition `BrowseVStsClaim_RD` yang mengisi grid layar
 * `StatusClaimInbox`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu keputusan yang diambil
 * dengan angka: isinya 33 baris dan bertambah beberapa baris per tahun. Layar yang
 * datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 */
export function gunakanDaftarStatusKlaim() {
  const token = gunakanSesi((keadaan) => keadaan.token)

  return useQuery({
    queryKey: kunci.daftar(token),
    queryFn: () => panggilAPI<ResponsDaftarStatusKlaim>(JALUR, { token }),
    enabled: token !== null,
    // Master nyaris tidak pernah berubah dalam satu sesi kerja. Lima menit menahan
    // pemuatan ulang yang tidak perlu, sementara tombol Muat ulang tetap tersedia bagi
    // pengguna yang tahu datanya baru saja diubah orang lain.
    staleTime: 5 * 60 * 1000,
  })
}

type IsianSimpan = {
  /** Kosong berarti menambah; terisi berarti mengubah status dengan kode itu. */
  kode?: string
  label: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempStsClaim` untuk keduanya, dan membedakannya dengan ada-tidaknya `LSC_ID`.
 * Perbedaannya hanya pada metode dan jalur, dan itu satu baris.
 *
 * Kode TIDAK pernah dikirim di badan permintaan. Pada penambahan ia dibuat server; pada
 * pengubahan ia berada di jalur URL.
 */
export function gunakanSimpanStatusKlaim() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: ({ kode, label }: IsianSimpan) =>
      panggilAPI<ResponsStatusKlaim>(kode ? `${JALUR}/${encodeURIComponent(kode)}` : JALUR, {
        metode: kode ? 'PUT' : 'POST',
        badan: { label },
        token,
      }),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan,
      // kode barunya hanya diketahui server — menebaknya di klien akan menampilkan
      // kode yang salah sampai muat ulang berikutnya.
      void klien.invalidateQueries({ queryKey: kunci.daftar(token) })
    },
  })
}
