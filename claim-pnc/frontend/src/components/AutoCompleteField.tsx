import { useEffect, useId, useRef, useState } from 'react'

/** Satu baris pilihan: nama yang dibaca orang, dan penunjuk yang disimpan diam-diam. */
export type AutoCompleteChoice = {
  id: string
  nama: string
}

type Props = {
  id: string
  label: string

  /**
   * Seluruh pilihan, APA ADANYA — termasuk dua baris yang namanya sama.
   *
   * Pengulangan nama di sini bukan cacat data: satu klaim dapat punya dua objek bernama
   * "KANTOR" yang penunjuknya berbeda. Membuangnya akan membuat salah satunya tidak
   * pernah dapat dipilih.
   */
  choices: AutoCompleteChoice[]

  /** Nama yang sedang tertulis di kolom. Boleh di luar daftar. */
  value: string

  /**
   * Dipanggil setiap kali isinya berubah — baik karena dipilih dari daftar maupun
   * diketik bebas.
   *
   * `choice` terisi hanya ketika barisnya benar-benar dipilih dari daftar. Pada ketikan
   * bebas ia `undefined`, dan pemanggil WAJIB mengosongkan penunjuknya: penunjuk lama
   * yang melekat pada nama baru akan menyimpan pasangan yang tidak cocok.
   */
  onPick: (nama: string, choice: AutoCompleteChoice | undefined) => void

  error?: string | undefined
  hint?: string | undefined
  required?: boolean
  disabled?: boolean

  /** Teks saat tidak satu pun pilihan cocok. */
  emptyText?: string
}

/**
 * AutoCompleteField adalah padanan `pxAutoComplete` Pega: kolom teks yang membuka
 * **daftar pilihan di bawahnya**.
 *
 * # Kenapa bukan `<datalist>` seperti ComboField
 *
 * Karena keduanya tidak menjawab hal yang sama. `<datalist>` hanya menyarankan SETELAH
 * orang mengetik, dan di sebagian peramban tidak terbuka sama sekali ketika kolomnya
 * sekadar diklik. Pada layar Salvage itu berakibat nyata: petugas tidak punya cara
 * mengetahui objek apa saja yang dimiliki klaimnya tanpa menebak huruf pertamanya.
 *
 * Layar lama tidak begitu. `Section/TambahData_Salvage-Section.xml:2727` menyetel
 * `pyFormat=pxAutoComplete` dengan `pyDisplayAsComboBox=false`, yang menggambar daftar
 * saran tepat di bawah kolomnya begitu kolom itu disentuh.
 *
 * # Ketikan bebas TETAP diterima
 *
 * `pyAllowFreeFormInput=true` (`:2869`). Daftar ini membantu, bukan memagari — nama di
 * luar daftar tetap boleh diketik, dan ketika itu terjadi penunjuknya dikosongkan.
 *
 * # Penunjuk ikut terbawa saat sebuah baris dipilih
 *
 * Itu yang di Pega dikerjakan `pyAdditionalFields` ber-`pyShow=false` (`:2928`): `.CaseID`
 * milik baris yang dipilih disalin diam-diam ke `TempInsert.CaseID`. Di sini pemilihan
 * mengembalikan BARISNYA, bukan namanya — sehingga dua baris bernama sama tetap
 * terbedakan, hal yang mustahil bila pencocokannya lewat nama.
 */
export function AutoCompleteField({
  id,
  label,
  choices,
  value,
  onPick,
  error,
  hint,
  required,
  disabled,
  emptyText = 'Tidak ada pilihan yang cocok. Nama boleh diketik langsung.',
}: Props) {
  const listID = `${useId()}-daftar`

  const [open, setOpen] = useState(false)

  // Baris yang sedang disorot papan ketik. `-1` berarti belum ada yang disorot — Enter
  // pada keadaan itu tidak memilih apa pun.
  const [sorot, setSorot] = useState(-1)

  // Menyaring daftar hanya MASUK AKAL setelah orang mengetik.
  //
  // Tanpa pembeda ini, membuka kembali kolom yang sudah terisi akan menyaring daftarnya
  // dengan nama yang sudah dipilih — menyisakan satu baris, dan menutup jalan untuk
  // berpindah ke pilihan lain tanpa menghapus isinya lebih dulu.
  const [menyaring, setMenyaring] = useState(false)

  const kotak = useRef<HTMLDivElement>(null)
  const daftar = useRef<HTMLUListElement>(null)

  const kueri = value.trim().toLowerCase()
  const tampil =
    menyaring && kueri !== ''
      ? choices.filter((choice) => choice.nama.toLowerCase().includes(kueri))
      : choices

  // Sorot dikembalikan ke awal setiap kali isi daftarnya berubah: nomor baris ketiga
  // pada daftar lama menunjuk baris yang sama sekali lain pada daftar yang baru disaring.
  useEffect(() => {
    setSorot(-1)
  }, [kueri, menyaring, choices])

  // Baris yang disorot digulirkan ke dalam pandangan.
  //
  // Daftarnya bergulir sendiri saat pilihannya banyak, dan tanpa ini menekan panah bawah
  // berulang kali menyorot baris yang sudah berada di luar layar.
  useEffect(() => {
    if (!open || sorot < 0) return
    const baris = daftar.current?.children[sorot]
    if (baris instanceof HTMLElement) baris.scrollIntoView({ block: 'nearest' })
  }, [open, sorot])

  function buka() {
    if (disabled) return
    setMenyaring(false)
    setOpen(true)
  }

  function pilih(choice: AutoCompleteChoice) {
    onPick(choice.nama, choice)
    setMenyaring(false)
    setOpen(false)
    setSorot(-1)
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      if (!open) {
        buka()
        return
      }
      if (tampil.length === 0) return

      const arah = event.key === 'ArrowDown' ? 1 : -1
      setSorot((kini) => {
        const berikut = kini + arah
        if (berikut < 0) return tampil.length - 1
        if (berikut >= tampil.length) return 0
        return berikut
      })
      return
    }

    if (event.key === 'Enter') {
      // Enter MEMILIH, dan tidak mengirim form.
      //
      // Tanpa `preventDefault`, menekan Enter untuk memilih objek akan sekaligus
      // mengirim pengajuan yang isian lainnya belum terisi.
      if (open && sorot >= 0 && tampil[sorot] !== undefined) {
        event.preventDefault()
        pilih(tampil[sorot])
      }
      return
    }

    if (event.key === 'Escape') {
      if (open) {
        event.preventDefault()
        setOpen(false)
        setSorot(-1)
      }
      return
    }

    if (event.key === 'Tab') setOpen(false)
  }

  const kelasInput =
    'mt-1 w-full rounded-kontrol border bg-white px-3 py-2 text-slate-900 shadow-lembut ' +
    'transition-[border-color,box-shadow] duration-150 ease-halus ' +
    'focus:outline-none focus-visible:ring-4 ' +
    'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500 ' +
    (error
      ? 'border-red-400 focus-visible:border-red-500 focus-visible:ring-red-500/25'
      : 'border-slate-300 focus-visible:border-blue-500 focus-visible:ring-blue-500/25')

  return (
    <div
      ref={kotak}
      className="relative"
      // Daftar ditutup ketika fokus benar-benar MENINGGALKAN kotak ini — bukan sekadar
      // berpindah dari input ke salah satu barisnya.
      onBlur={(event) => {
        if (kotak.current?.contains(event.relatedTarget as Node | null)) return
        setOpen(false)
        setSorot(-1)
      }}
    >
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>

      <input
        id={id}
        type="text"
        role="combobox"
        aria-expanded={open}
        aria-controls={listID}
        aria-autocomplete="list"
        aria-activedescendant={open && sorot >= 0 ? `${listID}-${sorot}` : undefined}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-galat` : hint ? `${id}-petunjuk` : undefined}
        // autoComplete dimatikan supaya saran peramban dari riwayat pengisian tidak
        // bercampur dengan daftar bisnis yang sesungguhnya.
        autoComplete="off"
        required={required}
        disabled={disabled}
        value={value}
        onChange={(event) => {
          const diketik = event.target.value
          setMenyaring(true)
          setOpen(true)

          // Nama yang persis sama dengan sebuah baris tetap membawa penunjuknya, supaya
          // mengetik lengkap tanpa mengklik daftarnya tidak kehilangan penunjuk itu.
          const sama = choices.filter(
            (choice) => choice.nama.toLowerCase() === diketik.trim().toLowerCase(),
          )

          // Hanya bila JAWABANNYA TUNGGAL. Dua objek bernama sama tidak dapat dibedakan
          // dari ketikan, dan menebak salah satunya akan menyimpan penunjuk yang keliru.
          onPick(diketik, sama.length === 1 ? sama[0] : undefined)
        }}
        onFocus={buka}
        onMouseDown={buka}
        onKeyDown={onKeyDown}
        className={kelasInput}
      />

      {open && (
        <ul
          ref={daftar}
          id={listID}
          role="listbox"
          aria-label={label}
          className="absolute z-20 mt-1 max-h-60 w-full overflow-y-auto rounded-kontrol border border-slate-300 bg-white py-1 shadow-angkat"
        >
          {tampil.length === 0 ? (
            <li className="px-3 py-2 text-sm text-slate-500">{emptyText}</li>
          ) : (
            tampil.map((choice, index) => (
              <li
                // Kunci barisnya POSISI, bukan nama maupun penunjuknya: keduanya dapat
                // berulang pada satu klaim, dan penunjuk dapat pula kosong.
                key={index}
                id={`${listID}-${index}`}
                role="option"
                aria-selected={index === sorot}
                // `mousedown` dibatalkan supaya kolomnya tidak kehilangan fokus sebelum
                // klik sempat terbaca — tanpa ini daftarnya tertutup lebih dulu dan
                // kliknya tidak pernah sampai.
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => pilih(choice)}
                onMouseEnter={() => setSorot(index)}
                className={
                  'cursor-pointer px-3 py-2 text-sm ' +
                  (index === sorot ? 'bg-blue-50 text-blue-900' : 'text-slate-900')
                }
              >
                {choice.nama === '' ? (
                  <span className="text-slate-400 italic">(tanpa nama)</span>
                ) : (
                  choice.nama
                )}

                {/*
                  Penunjuk digambar sebagai keterangan kedua, dan hanya bila ada.

                  Itu satu-satunya hal yang membedakan dua baris bernama sama — di layar
                  lama keduanya tergambar serupa, dan petugas tidak punya cara mengetahui
                  mana yang baru saja dipilihnya.
                */}
                {choice.id !== '' && (
                  <span className="ml-2 text-xs text-slate-500">{choice.id}</span>
                )}
              </li>
            ))
          )}
        </ul>
      )}

      {error ? (
        <p id={`${id}-galat`} className="mt-1 text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p id={`${id}-petunjuk`} className="mt-1 text-sm text-slate-500">
          {hint}
        </p>
      ) : null}
    </div>
  )
}
