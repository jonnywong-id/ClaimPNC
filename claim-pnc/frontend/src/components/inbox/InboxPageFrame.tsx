import type { ReactNode } from 'react'

type Props = {
  /** Judul layar — teksnya mengikuti layar Pega (`D-13`). */
  title: string
  /** Kalimat pembuka di bawah judul. */
  intro: string
  /** Keterangan tambahan di bawah kalimat pembuka. */
  extra?: ReactNode
  /** Kontrol di sisi kanan kepala layar (mis. tombol ekspor). */
  aside?: ReactNode
  children: ReactNode
}

/**
 * InboxPageFrame adalah bingkai baku layar-layar inbox: kepala berisi judul dan kalimat
 * pembuka, lalu isi layar. Bila ada kontrol di kanan (`aside`), kepalanya disusun dua kolom.
 */
export function InboxPageFrame({ title, intro, extra, aside, children }: Readonly<Props>) {
  const text = (
    <>
      <h1 className="text-xl font-semibold text-slate-900">{title}</h1>
      <p className="mt-1 text-sm text-slate-600">{intro}</p>
      {extra}
    </>
  )

  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      {aside === undefined ? (
        <header className="border-b border-slate-200 pb-4">{text}</header>
      ) : (
        <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
          <div>{text}</div>
          {aside}
        </header>
      )}
      {children}
    </div>
  )
}
