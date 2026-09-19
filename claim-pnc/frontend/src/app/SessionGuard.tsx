import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useSession } from '@/app/session'

/**
 * SessionGuard menahan rute yang menuntut pengguna sudah masuk.
 *
 * Ini KENYAMANAN TAMPILAN, bukan pengamanan. Penegakan yang sebenarnya ada di server:
 * setiap endpoint memeriksa sesi pemanggilnya sendiri (docs/Steering/11-SECURITY.md §3.1).
 * Menyembunyikan halaman di peramban tidak menutup apa pun.
 */
export function SessionGuard({ children }: { children: ReactNode }) {
  const token = useSession((state) => state.token)
  const location = useLocation()

  if (!token) {
    // Lokasi yang dituju dititipkan supaya setelah masuk pengguna kembali ke sana,
    // bukan dilempar ke beranda.
    return <Navigate to="/masuk" replace state={{ from: location }} />
  }
  return <>{children}</>
}
