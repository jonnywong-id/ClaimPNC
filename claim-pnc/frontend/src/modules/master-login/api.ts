import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  SurveyorLoginInput,
  SurveyorLoginListResponse,
  SurveyorLoginResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/login'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat daftar orang yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan mereka milik badan hukum lain (R-20).
 *
 * TANPA penyaring status di kunci cache, berbeda dari layar master di rumpun sparepart:
 * POOLDATA.MST_LOGIN_SURVEYOR tidak punya kolom APPROVAL, sehingga layar ini tidak bertab
 * dan hanya ada satu daftar per portal.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-login', portal, token] as const
}

/**
 * Hook daftar Master Login.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti daftar Pega yang dimuat
 * penuh ke klipboard lalu dipaginasi 15 baris per halaman tanpa satu pun kotak pencarian.
 * Yang dipakai layar sekarang adalah pencarian bawaan `DataTable`, sama seperti seluruh
 * layar master lain — pencarian kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function useSurveyorLoginList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<SurveyorLoginListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook penambahan.
 *
 * Daftar dibuang dari cache setelah berhasil. Baris baru muncul di daftar yang sama —
 * tidak ada tab yang membedakannya — sehingga satu pembuangan sudah cukup.
 */
export function useCreateSurveyorLogin() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: SurveyorLoginInput) =>
      callAPI<SurveyorLoginResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-login'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Jalur URL-nya memakai LOGIN, bukan sebuah ID terpisah: tabelnya tidak punya kunci lain,
 * dan setiap pernyataan simpannya menyaring `where login = ...`.
 *
 * `encodeURIComponent` bukan formalitas di sini. LOGIN diturunkan dari Nama dengan membuang
 * spasi, titik, koma, dan tanda hubung — tetapi karakter LAIN tidak dibuang, termasuk
 * apostrof pada nama seperti "O'Brien" dan garis miring bila ada. Tanpa pengodean, keduanya
 * mengubah bentuk jalurnya.
 */
export function useSaveSurveyorLogin() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ login, input }: { login: string; input: SurveyorLoginInput }) =>
      callAPI<SurveyorLoginResponse>(`${ROUTE}/${encodeURIComponent(login)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-login'] })
    },
  })
}
