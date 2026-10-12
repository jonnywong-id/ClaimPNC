import {
  joinModules,
  joinSubModules,
  moduleChecklist,
  subModuleChecklist,
  toggleChecklist,
  type ChecklistItem,
} from './api'

type Props = {
  /** Isi kolom MODUL apa adanya. */
  modul: string
  /** Isi kolom SUBMODUL apa adanya, termasuk koma di ujungnya. */
  subModul: string
  disabled?: boolean | undefined
  /** Penanda unik supaya id kotak centang tidak bertabrakan antarbaris pada form Tambah. */
  idPrefix: string
  onChange: (change: { modul?: string; sub_modul?: string }) => void
}

/**
 * Isian MODUL dan SUB MODUL — berbentuk KOTAK CENTANG, sama seperti layar lama.
 *
 * Dipakai bersama oleh form Tambah dan form Ubah, karena keduanya memang memakai isian
 * yang sama: kolom TEMPLATE AKSES pada layar lama memuat kotak centang untuk MODUL dan
 * SUB MODUL, bukan isian teks.
 *
 * # Kenapa bukan teks bebas
 *
 * Teks bebas sempat dipakai di sini dan dicabut. Alasan memakainya dulu memang ada —
 * daftar pilihannya hilang dari export (`R-16`) — tetapi akibatnya lebih berat daripada
 * masalah yang dihindarinya: satu salah ketik pada `PNCSearchKlaim` menghasilkan baris
 * yang TAMPAK memberi kewenangan padahal tidak pernah cocok dengan modul mana pun, dan
 * tidak ada apa pun di layar yang menandakannya.
 *
 * Daftar pilihannya kini dipastikan dari tangkapan layar aplikasi berjalan dan dari
 * pembacaan langsung kolomnya — lihat MASKING_MODULES di `api.ts`.
 *
 * # Nilai di luar daftar tetap tampil
 *
 * Satu baris produksi menyimpan sub modul yang salah ketik. Ia ditampilkan sebagai kotak
 * centang tersendiri bertanda "di luar daftar" — bukan disembunyikan. Menyembunyikannya
 * akan membuat baris itu kehilangan nilainya begitu disimpan ulang.
 */
export function ModuleChecklist({ modul, subModul, disabled, idPrefix, onChange }: Props) {
  const modulItem = moduleChecklist(modul)
  const subItem = subModuleChecklist(subModul)

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <ChecklistGroup
        title="Modul"
        idPrefix={`${idPrefix}-modul`}
        item={modulItem}
        disabled={disabled}
        onToggle={(nilai, checked) =>
          onChange({ modul: toggleChecklist(modulItem, nilai, checked, joinModules) })
        }
      />
      <ChecklistGroup
        title="Sub Modul"
        idPrefix={`${idPrefix}-submodul`}
        item={subItem}
        disabled={disabled}
        onToggle={(nilai, checked) =>
          onChange({ sub_modul: toggleChecklist(subItem, nilai, checked, joinSubModules) })
        }
      />
    </div>
  )
}

function ChecklistGroup({
  title,
  idPrefix,
  item,
  disabled,
  onToggle,
}: {
  title: string
  idPrefix: string
  item: ChecklistItem[]
  disabled?: boolean | undefined
  onToggle: (nilai: string, checked: boolean) => void
}) {
  return (
    <fieldset>
      <legend className="text-sm font-medium text-slate-700">{title}</legend>
      <div className="mt-1.5 space-y-1.5">
        {item.map((row) => {
          const id = `${idPrefix}-${row.nilai.replace(/\s+/g, '-')}`
          return (
            <label
              key={row.nilai}
              htmlFor={id}
              className={`flex items-center gap-2 text-sm ${
                disabled ? 'cursor-not-allowed text-slate-400' : 'cursor-pointer text-slate-700'
              }`}
            >
              <input
                id={id}
                type="checkbox"
                disabled={disabled}
                checked={row.checked}
                onChange={(event) => onToggle(row.nilai, event.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
              />
              <span>{row.label}</span>
              {!row.dikenal && (
                <span className="rounded bg-amber-50 px-1.5 py-0.5 text-xs text-amber-800 ring-1 ring-amber-200">
                  di luar daftar
                </span>
              )}
            </label>
          )
        })}
      </div>
    </fieldset>
  )
}
