import type { ReactNode } from "react";

import type { KomiteAttachment } from "@/api/types";
import type { Column } from "@/components/DataTable";

import { useAttachmentLink } from "./api";

/**
 * Membuka lampiran klaim dari layar Komite — dipakai Dokumen Pendukung dan tab Lampiran
 * Dokumen, supaya keduanya membuka berkas dengan cara yang sama.
 *
 * Padanan `GetLinkViewDoc2_Act` + `openUrlInWindow` pada `Section/PNCViewAttachmentKomite`:
 * alamat baca diminta ke tautan dokumen klaim saat nama ditekan, lalu dibuka di tab baru.
 * Hanya lampiran ber-IMAGEID (`tersimpan`) yang menjadi tautan; lampiran lama yang hanya
 * BLOB tetap teks, karena activity itu berhenti bila IMAGEID kosong.
 */
export function useAttachmentOpener(claimKey: string) {
  const link = useAttachmentLink(claimKey);

  // Jendela dibuka LEBIH DULU, saat klik masih berlangsung, lalu diarahkan setelah alamatnya
  // datang — membukanya sesudah permintaan selesai akan diblokir peramban sebagai popup.
  const open = (attachment: KomiteAttachment) => {
    const target = window.open("", "_blank");
    if (target) target.opener = null;
    link.mutate(attachment.id, {
      onSuccess: (found) => {
        if (target) target.location.href = found.url;
        else window.open(found.url, "_blank", "noopener,noreferrer");
      },
      onError: () => target?.close(),
    });
  };

  const nameCell = (r: KomiteAttachment): ReactNode =>
    claimKey && r.tersimpan ? (
      <button
        type="button"
        className="break-all text-left text-sky-700 underline hover:text-sky-900 disabled:cursor-wait disabled:opacity-60"
        disabled={link.isPending && link.variables === r.id}
        onClick={() => open(r)}
      >
        {r.nama || "dokumen"}
      </button>
    ) : (
      (r.nama ?? "")
    );

  /** Kolom yang kuncinya `nama` diberi sel tautan; kolom lain tidak diubah. */
  const withNameLink = (
    columns: Column<KomiteAttachment>[],
  ): Column<KomiteAttachment>[] =>
    columns.map((c) => (c.key === "nama" ? { ...c, render: nameCell } : c));

  const error = link.isError
    ? link.error instanceof Error
      ? link.error.message
      : "Dokumen tidak dapat dibuka."
    : null;

  return { withNameLink, error };
}

/** Pesan galat membuka dokumen, di bawah tabelnya. */
export function AttachmentOpenError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p role="alert" className="mt-2 text-sm text-red-700">
      {message}
    </p>
  );
}
