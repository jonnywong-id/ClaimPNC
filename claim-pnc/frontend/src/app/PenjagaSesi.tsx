import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { gunakanSesi } from '@/app/sesi'

/**
 * PenjagaSesi menahan rute yang menuntut pengguna sudah masuk.
 *
 * Ini KENYAMANAN TAMPILAN, bukan pengamanan. Penegakan yang sebenarnya ada di server:
 * setiap endpoint memeriksa sesi pemanggilnya sendiri (docs/Steering/11-SECURITY.md §3.1).
 * Menyembunyikan halaman di peramban tidak menutup apa pun.
 */
export function PenjagaSesi({ anak }: { anak: ReactNode }) {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const lokasi = useLocation()

  if (!token) {
    // Lokasi yang dituju dititipkan supaya setelah masuk pengguna kembali ke sana,
    // bukan dilempar ke beranda.
    return <Navigate to="/masuk" replace state={{ dari: lokasi }} />
  }
  return <>{anak}</>
}
