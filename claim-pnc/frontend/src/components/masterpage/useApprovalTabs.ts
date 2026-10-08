import { useState } from 'react'

import { toggled } from '@/components/ApprovalControls'

/**
 * Keadaan bersama layar master bertab persetujuan: tab yang aktif dan baris yang dicentang
 * pada tab Waiting Approval.
 *
 * Tab yang tidak dikenal jatuh ke tab pertama, sehingga `active` selalu terisi.
 */
export function useApprovalTabs<TTabs extends readonly [{ id: string }, ...{ id: string }[]]>(
  tabs: TTabs,
  initial: TTabs[number]['id'],
) {
  const [tab, setTab] = useState<TTabs[number]['id']>(initial)
  const active: TTabs[number] = tabs.find((t) => t.id === tab) ?? tabs[0]
  const [chosen, setChosen] = useState<Set<string>>(new Set())

  return {
    tab,
    setTab,
    active,
    chosen,
    setChosen,
    /** Membalik centang satu baris. */
    toggle(id: string) {
      setChosen((current) => toggled(current, id))
    },
  }
}
