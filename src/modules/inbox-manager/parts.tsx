import type { ReactNode } from "react";

import { EmptyBoxIcon } from "@/components/Icon";

/**
 * Potongan tampilan Inbox Manager — kartu, kepala kartu, cip, dan bilah isian.
 *
 * Bentuknya mengikuti rancangan layar yang diberikan Work Owner (2026-10-14): setiap dashboard
 * dan antrean adalah KARTU dengan kepala berjudul, cip ringkasan di sampingnya, dan bilah
 * isian berlatar abu-abu tepat di atas tabel yang dikenainya.
 */

export function Card({
  children,
  label,
  dashed = false,
}: {
  children: ReactNode;
  label?: string | undefined;
  dashed?: boolean;
}) {
  return (
    <section
      aria-label={label}
      className={[
        "overflow-hidden rounded-kartu border bg-white shadow-lembut",
        dashed ? "border-dashed border-slate-300" : "border-slate-200",
      ].join(" ")}
    >
      {children}
    </section>
  );
}

export function CardHeader({
  title,
  aside,
  description,
}: {
  title: string;
  aside?: ReactNode;
  description?: string | undefined;
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-slate-200 px-5 py-4">
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>
      {aside}
      {description ? (
        <p className="w-full text-xs leading-relaxed text-slate-500">
          {description}
        </p>
      ) : null}
    </div>
  );
}

export type ChipTone = "netral" | "aksen" | "biru" | "kuning" | "redup";

const chipTone: Record<ChipTone, string> = {
  netral: "bg-slate-100 text-slate-700",
  aksen: "bg-blue-600 text-white",
  biru: "bg-blue-50 text-blue-700",
  kuning: "bg-amber-50 text-amber-900",
  redup: "bg-slate-50 text-slate-400",
};

export function Chip({
  children,
  tone = "netral",
}: {
  children: ReactNode;
  tone?: ChipTone;
}) {
  return (
    <span
      className={`inline-flex h-6 items-center rounded-full px-2.5 text-xs font-semibold ${chipTone[tone]}`}
    >
      {children}
    </span>
  );
}

/** Bilah isian berlatar abu-abu di bawah kepala kartu. */
export function ControlStrip({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-end gap-3 border-b border-slate-200 bg-slate-50/70 px-5 py-3">
      {children}
    </div>
  );
}

/** Kelas isian tanggal dan pilihan di dalam bilah. */
export const inputClass =
  "rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm " +
  "transition-[border-color,box-shadow] duration-150 ease-halus hover:border-slate-400 " +
  "focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15";

export function FieldLabel({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-xs font-medium text-slate-700">
        {label}
      </span>
      {children}
    </label>
  );
}

export function EmptyState({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
      <span className="flex h-11 w-11 items-center justify-center rounded-full bg-slate-100 text-slate-400">
        <EmptyBoxIcon className="h-5 w-5" />
      </span>
      <p className="text-sm font-medium text-slate-700">{title}</p>
      {description ? (
        <p className="max-w-sm text-xs leading-relaxed text-slate-500">
          {description}
        </p>
      ) : null}
    </div>
  );
}

/** Kepala kolom tabel dashboard. */
export function Th({
  children,
  right = false,
  className = "",
  colSpan,
  scope = "col",
}: {
  children?: ReactNode;
  right?: boolean;
  className?: string;
  colSpan?: number;
  scope?: "col" | "colgroup";
}) {
  return (
    <th
      scope={scope}
      colSpan={colSpan}
      className={`px-3 py-2.5 text-xs font-semibold tracking-wide text-slate-600 ${right ? "text-right" : ""} ${className}`}
    >
      {children}
    </th>
  );
}
