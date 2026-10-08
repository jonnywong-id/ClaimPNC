import type { ReactNode } from 'react'

/**
 * Tombol yang tampil seperti tautan, untuk nomor case/klaim yang membuka layar kerja.
 *
 * Ia tombol, bukan `<a>`, karena tujuannya disusun pemanggil (mis. membawa tab dan halaman
 * asal supaya tombol kembali mendarat di tempat yang sama).
 */
export function LinkButton({
  title,
  onClick,
  children,
  disabledStyles = false,
}: Readonly<{
  title: string
  onClick: () => void
  children: ReactNode
  /** Sertakan gaya keadaan `disabled`. */
  disabledStyles?: boolean | undefined
}>) {
  const classes = [
    'rounded-kontrol text-left font-medium text-blue-700 underline-offset-2',
    'transition-colors duration-150 ease-halus hover:underline',
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
  ]
  if (disabledStyles) {
    classes.push('disabled:cursor-not-allowed disabled:text-slate-400 disabled:no-underline')
  }

  return (
    <button type="button" onClick={onClick} title={title} className={classes.join(' ')}>
      {children}
    </button>
  )
}
