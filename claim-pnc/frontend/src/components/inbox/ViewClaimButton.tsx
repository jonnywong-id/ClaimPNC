import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/Button'

/**
 * Tombol "Lihat Detail Klaim" pada kolom aksi antrean inbox.
 *
 * Ia menuju penampung `/view-claim/:referensi` — layar "View Claim" yang belum dibangun —
 * dengan kunci yang diberikan pemanggil (biasanya referensi objek kerja, jatuh ke nomor
 * case/klaim bila kosong). Kunci yang kosong mematikan tombolnya: tautan tanpa alamat
 * membawa pengguna ke layar yang pasti gagal.
 */
export function ViewClaimButton({
  reference,
  label = 'Lihat Detail Klaim',
}: Readonly<{ reference: string; label?: string | undefined }>) {
  const navigate = useNavigate()

  return (
    <Button
      tone="halus"
      disabled={reference === ''}
      onClick={() => navigate(`/view-claim/${encodeURIComponent(reference)}`)}
    >
      {label}
    </Button>
  )
}
