import { useEffect, useRef } from 'react'

/**
 * useDebouncedCommit menyerahkan ketikan kotak cari setelah jeda, bukan per huruf.
 *
 * Jedanya (bawaan 350 ms) cukup panjang untuk menelan satu kata yang diketik cepat, dan
 * cukup pendek untuk tidak terasa seperti layar yang menggantung. Mengirim satu permintaan
 * per huruf berarti beberapa kali penarikan penuh untuk satu kata.
 *
 * `commit` dibaca lewat ref, sehingga render ulang di tengah jeda TIDAK memulai ulang
 * hitungannya — yang memulainya hanyalah ketikan baru.
 */
export function useDebouncedCommit(
  draft: string,
  committed: string,
  commit: (value: string) => void,
  delay = 350,
): void {
  const latest = useRef(commit)
  useEffect(() => {
    latest.current = commit
  })

  useEffect(() => {
    if (draft === committed) return

    const timer = setTimeout(() => latest.current(draft), delay)
    return () => clearTimeout(timer)
  }, [draft, committed, delay])
}
