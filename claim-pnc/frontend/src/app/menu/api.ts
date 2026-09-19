import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { MenuListResponse } from '@/api/types'
import { useSession } from '@/app/session'

const ROUTE = '/api/menu'

/**
 * Hook peta menu pengguna.
 *
 * # Kenapa token ikut di dalam kunci cache
 *
 * Menu seseorang adalah kewenangannya. Tanpa token di kunci, pengguna berikutnya di
 * peramban yang sama akan melihat menu pengguna sebelumnya dari cache — dan ia akan
 * tampak seperti menu miliknya sendiri.
 *
 * # Kenapa TANPA portal
 *
 * Peta menu dan kewenangannya hidup di basis data portal utama dan tidak punya kolom
 * entitas. Berpindah portal mengubah data yang dibaca layar, bukan daftar layar yang
 * boleh dibuka.
 *
 * # Kenapa staleTime panjang
 *
 * Kewenangan menu nyaris tidak pernah berubah di tengah satu sesi kerja, dan menu
 * dirender di SETIAP layar. Memuatnya ulang pada setiap perpindahan halaman berarti
 * satu permintaan tambahan untuk jawaban yang sama. Perubahan izin tetap berlaku pada
 * sesi berikutnya — dan pada sesi ini pun, karena server memeriksa ulang setiap
 * permintaan; yang tertunda hanyalah tampilan menunya.
 */
export function useMenu() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['menu', token],
    queryFn: () => callAPI<MenuListResponse>(ROUTE, { token }),
    enabled: token !== null,
    staleTime: 30 * 60 * 1000,
  })
}
