/**
 * Kotak centang penyaring di atas tabel inbox, mis. "See All Claim".
 *
 * Teksnya mengikuti layar Pega apa adanya (`D-13`), karena itu label diterima utuh dari
 * pemanggil.
 */
export function InboxCheckbox({
  label,
  checked,
  onChange,
}: Readonly<{
  label: string
  checked: boolean
  onChange: (value: boolean) => void
}>) {
  return (
    <label className="flex items-center gap-2 text-sm text-slate-700">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="size-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
      />
      <span>{label}</span>
    </label>
  )
}
