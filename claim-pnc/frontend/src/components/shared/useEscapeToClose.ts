import { useEffect } from 'react'

/**
 * useEscapeToClose menutup dialog saat tombol Escape ditekan, kecuali selama `busy` —
 * dialog yang sedang memproses tidak boleh ditutup di tengah jalan.
 */
export function useEscapeToClose(onClose: () => void, busy: boolean) {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])
}
