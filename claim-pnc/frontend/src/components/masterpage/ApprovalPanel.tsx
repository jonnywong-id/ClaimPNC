import {
  ApprovalTabNav,
  DecisionBar,
  DecisionFeedback,
  PortalNote,
  type ApprovalTab,
} from '@/components/ApprovalControls'

type DecisionResult = { jumlah_berubah: number; status_label: string }

/**
 * Bagian persetujuan layar master bertab Approve, Reject, Waiting Approval: deretan tab,
 * keterangan tab beserta portal entitas, pesan hasil keputusan, dan — pada tab Waiting
 * Approval — bar keputusan untuk baris yang dicentang.
 *
 * Berpindah tab membuang centang dan hasil keputusan sebelumnya: baris yang dipilih milik
 * tab sebelumnya, dan menyimpannya berarti keputusan dapat mengenai baris yang tidak sedang
 * dilihat siapa pun. Yang khusus layar (menutup form, mengosongkan catatan) dikerjakan
 * `onBeforeSwitch`.
 */
export function ApprovalPanel<Id extends string>({
  tabLabel,
  tabs,
  active,
  onSwitch,
  onBeforeSwitch,
  portal,
  decide,
  feedbackNoun,
  barNoun,
  chosenCount,
  onApprove,
  onReject,
  onClear,
  note,
}: Readonly<{
  tabLabel: string
  tabs: readonly ApprovalTab<Id>[]
  active: ApprovalTab<Id>
  onSwitch: (id: Id) => void
  onBeforeSwitch: () => void
  portal: string | null | undefined
  decide: {
    isError: boolean
    error: unknown
    isSuccess: boolean
    isPending: boolean
    data?: DecisionResult | undefined
    reset: () => void
  }
  /** Kata benda pada pesan hasil, mis. "tipe sparepart". */
  feedbackNoun: string
  /** Kata benda pada bar keputusan, mis. "tipe". */
  barNoun: string
  chosenCount: number
  onApprove: () => void
  onReject: () => void
  onClear: () => void
  note?: { value: string; onChange: (value: string) => void }
}>) {
  return (
    <>
      <ApprovalTabNav
        label={tabLabel}
        tabs={tabs}
        active={active.id}
        onSelect={(id) => {
          onBeforeSwitch()
          onClear()
          decide.reset()
          onSwitch(id)
        }}
      />

      <PortalNote description={active.description} portal={portal} />

      <DecisionFeedback decide={decide} noun={feedbackNoun} />

      {active.id === 'menunggu' && (
        <DecisionBar
          noun={barNoun}
          count={chosenCount}
          isBusy={decide.isPending}
          onApprove={onApprove}
          onReject={onReject}
          onClear={onClear}
          {...(note ? { note } : {})}
        />
      )}
    </>
  )
}
