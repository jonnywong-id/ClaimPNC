import { Link } from 'react-router-dom'

/**
 * Nomor klaim sebagai tautan ke layar rinciannya.
 *
 * Yang dikirim di alamat adalah nomor klaim, bukan kunci teknis Pega: alamatnya terbaca
 * orang, dapat disalin ke percakapan, dan tidak membocorkan bentuk kunci internal Pega ke
 * bilah alamat.
 *
 * Baris tanpa nomor klaim TIDAK menjadi tautan. Tautan yang alamatnya kosong tetap dapat
 * diklik dan membawa pengguna ke layar yang pasti gagal; yang digambar adalah tanda pisah
 * yang sama dengan sel kosong lain.
 */
export function ClaimNumberLink({
  value,
  basePath,
}: Readonly<{
  value: string
  /** Awalan rute tujuan, mis. `/outstanding-claim`. */
  basePath: string
}>) {
  if (value === '') return <span className="text-slate-400">—</span>

  return (
    <Link
      to={`${basePath}/${encodeURIComponent(value)}`}
      className={[
        'font-medium text-blue-700 underline-offset-2 hover:underline',
        'focus:outline-none focus-visible:rounded-kontrol',
        'focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      {value}
    </Link>
  )
}
