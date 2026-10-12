/** Tanggal WIB — sama persis dengan yang dipakai daftarnya; keduanya tidak boleh berbeda bentuk. */
export function formatTanggal(value: string | undefined): string {
  if (!value) return '—'
  const saat = new Date(value)
  if (Number.isNaN(saat.getTime())) return '—'
  return saat.toLocaleDateString('id-ID', {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })
}

/** Tanggal dan jam WIB — untuk CREATE COMITEE DATE, yang di Pega `.pxCreateDateTime`. */
export function formatWaktu(value: string | undefined): string {
  if (!value) return '—'
  const saat = new Date(value)
  if (Number.isNaN(saat.getTime())) return '—'
  return saat.toLocaleString('id-ID', {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Persen seperti Pega menampilkannya: angka apa adanya diikuti " %". */
export function formatPersen(value: string | undefined): string {
  const v = (value ?? '').trim()
  return v === '' ? '—' : `${v} %`
}
