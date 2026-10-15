import type { ReactNode } from "react";

import { formatDate, formatDateTimeWIB } from "@/components/format";

import { cellText, isDimension, isNegative, rowKeyOf } from "./cells";
import { PeriodStrip } from "./controls";
import { Card, CardHeader, Chip, EmptyState, Th } from "./parts";
import type { Panel, Period, PeriodInput, Tab, TabColumn } from "./types";

type Props = {
  tab: Tab;
  panels: Panel[];
  isLoading: boolean;
  failure: ReactNode;
  period: PeriodInput | null;
  onPeriodChange: (period: PeriodInput | null) => void;

  /** Periode yang BENAR-BENAR dipakai server, termasuk nilai bawaannya. */
  applied?: Period | undefined;

  /** Waktu cuplikan `PEGA_DASHBOARDPNC` terakhir disegarkan. */
  refreshedAt?: string | undefined;
};

/**
 * Cip waktu cuplikan.
 *
 * Kedua dashboard ini membaca TABEL CUPLIKAN (`POOLDATA.PEGA_DASHBOARDPNC`), bukan data klaim
 * langsung — angkanya berumur sampai penyegaran berikutnya. Layar lama pun menampilkannya
 * (`Activity/LastRefresh_act`, `<pyCaption Last Refresh>`).
 */
function RefreshChip({ at }: { at?: string | undefined }) {
  if (!at) return null;
  return <Chip tone="kuning">Cuplikan {formatDateTimeWIB(at)}</Chip>;
}

function PeriodChip({ applied, suffix = "" }: { applied?: Period | undefined; suffix?: string }) {
  if (!applied) return null;
  return (
    <Chip tone="biru">
      {formatDate(applied.dari)} – {formatDate(applied.sampai)}
      {suffix}
    </Chip>
  );
}

/**
 * Tab Produktivitas Klaim — mengikuti rancangan layar Work Owner 2026-10-14.
 *
 * Delapan kolom pencacahnya adalah EMPAT keranjang × DUA periode. Rancangan layar
 * menggambarnya sebagai empat kolom: angka periode berjalan besar, pembanding tahun lalu kecil
 * di bawahnya. Judul kolomnya tetap dari Pega — "Total register tahun sama" dan pasangannya —
 * dipisah menjadi nama keranjang dan kedua periodenya.
 *
 * Kedua grid TETAP digambar (Nama PIC dan COB), meski rancangan layarnya hanya menggambar
 * satu: keduanya ada di `Section/DisplayInboxProduktivitas_Sect`.
 */
export function ProduktivitasView(props: Props) {
  return (
    <Card label={props.tab.nama}>
      <CardHeader
        title={props.tab.nama}
        description={props.tab.keterangan}
        aside={
          <>
            <PeriodChip applied={props.applied} suffix=" vs tahun lalu" />
            <RefreshChip at={props.refreshedAt} />
          </>
        }
      />
      <PeriodStrip
        tab={props.tab}
        value={props.period}
        onChange={props.onPeriodChange}
      />
      {props.failure ? (
        <div className="border-b border-slate-200 px-5 py-3">{props.failure}</div>
      ) : null}

      {props.panels.map((panel) => (
        <ComparisonTable key={panel.kunci} panel={panel} isLoading={props.isLoading} />
      ))}
    </Card>
  );
}

/** Akhiran kunci kolom periode berjalan dan pasangannya tahun lalu (tab.go). */
const NOW_SUFFIX = "_periode_ini";
const PRIOR_SUFFIX = "_periode_tahun_lalu";

/** Akhiran judul Pega yang dipisah dari nama keranjangnya. */
const NOW_TITLE = " tahun sama";
const PRIOR_TITLE = " tahun sebelumnya";

function stripTitle(title: string): string {
  const lower = title.toLowerCase();
  for (const suffix of [NOW_TITLE, PRIOR_TITLE]) {
    if (lower.endsWith(suffix)) return title.slice(0, title.length - suffix.length);
  }
  return title;
}

function ComparisonTable({ panel, isLoading }: { panel: Panel; isLoading: boolean }) {
  const dimension = panel.kolom[0];
  const pairs: { now: TabColumn; prior?: TabColumn | undefined }[] = [];
  for (const column of panel.kolom.slice(1)) {
    if (column.kunci.endsWith(PRIOR_SUFFIX)) continue;
    const priorKey = column.kunci.endsWith(NOW_SUFFIX)
      ? column.kunci.slice(0, -NOW_SUFFIX.length) + PRIOR_SUFFIX
      : "";
    pairs.push({
      now: column,
      prior: panel.kolom.find((candidate) => candidate.kunci === priorKey),
    });
  }
  if (!dimension) return null;

  return (
    <div className="overflow-x-auto border-b border-slate-200 last:border-b-0">
      <table className="w-full border-collapse text-sm">
        <thead className="border-b border-slate-200 bg-slate-50/80 text-left">
          <tr>
            <Th className="align-bottom">{dimension.judul}</Th>
            {pairs.map(({ now, prior }) => (
              <Th key={now.kunci} right className="align-bottom">
                {stripTitle(now.judul)}
                {prior ? (
                  <span className="block font-normal text-slate-500">
                    {NOW_TITLE.trim()} · {PRIOR_TITLE.trim()}
                  </span>
                ) : null}
              </Th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {panel.baris.length === 0 ? (
            <tr>
              <td colSpan={pairs.length + 1}>
                <EmptyState
                  title={isLoading ? "Memuat…" : "Tidak ada angka untuk penyaring ini."}
                />
              </td>
            </tr>
          ) : null}

          {panel.baris.map((row, index) => {
            const name = cellText(row[dimension.kunci]);
            const summary = name === "ALL";
            return (
              <tr
                key={rowKeyOf(panel, row, index)}
                className={summary ? "bg-slate-50/60" : "hover:bg-blue-50/50"}
              >
                <td className="px-3 py-3 text-[13px] font-semibold text-slate-900">{name}</td>
                {pairs.map(({ now, prior }) => (
                  <td key={now.kunci} className="px-3 py-3 text-right">
                    <span className="block text-base font-semibold tabular-nums text-slate-900">
                      {cellText(row[now.kunci]) || "0"}
                    </span>
                    {prior ? (
                      <span className="block text-[11px] tabular-nums text-slate-500">
                        <span className="sr-only">{prior.judul}: </span>
                        {cellText(row[prior.kunci]) || "0"}
                      </span>
                    ) : null}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

/**
 * Tab Klaim — mengikuti rancangan layar Work Owner 2026-10-14.
 *
 * Judul kolom BERTINGKAT: pasangan "CASE X" / "NILAI X (Rp)" dari Pega dikelompokkan di bawah
 * X. Judulnya tetap dari Pega; yang berubah hanya letaknya. Rupiah TETAP penuh — tidak
 * dibulatkan ke miliar (keputusan Work Owner 2026-10-14) — dan nilai negatif diberi tanda
 * minus serta warna.
 *
 * Kedua grid digambar, termasuk "Total Klaim CauseOfLoss": ia sudah dipindahkan dan berisi,
 * berbeda dari rancangan layar yang menandainya belum.
 */
export function KlaimView(props: Props) {
  // Tanpa satu panel pun, kartu pertama tetap digambar: bilah periodenya menempel di sana, dan
  // ia harus tetap ada supaya penyaring yang ditolak dapat dibetulkan di tempat.
  const cards: (Panel | null)[] = props.panels.length > 0 ? props.panels : [null];

  return (
    <div className="space-y-4">
      {cards.map((panel, index) => (
        // Dikunci URUTANNYA, bukan nama panelnya: bentuk panel dari keterangan layar
        // digantikan isi dari server saat data tiba, dan kunci yang berganti memasang ulang
        // kartunya — membuang isian periode yang sedang diketik.
        <Card key={index} label={panel?.judul || props.tab.nama}>
          <CardHeader
            title={panel?.judul || props.tab.nama}
            description={index === 0 ? props.tab.keterangan : undefined}
            aside={
              <>
                <PeriodChip applied={props.applied} />
                <RefreshChip at={props.refreshedAt} />
              </>
            }
          />
          {index === 0 ? (
            <>
              <PeriodStrip
                tab={props.tab}
                value={props.period}
                onChange={props.onPeriodChange}
                // Teks dari rancangan layar Work Owner 2026-10-14.
                note={
                  props.applied
                    ? undefined
                    : "Tanpa penyaring periode, angka mencakup seluruh periode."
                }
              />
              {props.failure ? (
                <div className="border-b border-slate-200 px-5 py-3">{props.failure}</div>
              ) : null}
            </>
          ) : null}
          {panel ? <GroupedTable panel={panel} isLoading={props.isLoading} /> : null}
        </Card>
      ))}
    </div>
  );
}

type Group =
  | { kind: "single"; column: TabColumn }
  | { kind: "pair"; name: string; count: TabColumn; amount: TabColumn };

/** groupColumns memasangkan "CASE X" dengan "NILAI X (Rp)" yang menyusulnya. */
function groupColumns(columns: TabColumn[]): Group[] {
  const groups: Group[] = [];
  for (let i = 0; i < columns.length; i++) {
    const column = columns[i];
    const next = columns[i + 1];
    if (!column) continue;
    if (column.judul.startsWith("CASE ") && next?.uang && next.judul.startsWith("NILAI ")) {
      groups.push({ kind: "pair", name: column.judul.slice(5), count: column, amount: next });
      i++;
      continue;
    }
    groups.push({ kind: "single", column });
  }
  return groups;
}

function amountCell(panelRowCell: Parameters<typeof cellText>[0], key: string) {
  const negative = isNegative(panelRowCell);
  return (
    <td
      key={key}
      className={`px-3 py-2.5 text-right text-[13px] tabular-nums ${negative ? "font-semibold text-red-700" : "text-slate-900"}`}
    >
      {cellText(panelRowCell, true)}
    </td>
  );
}

function GroupedTable({ panel, isLoading }: { panel: Panel; isLoading: boolean }) {
  const groups = groupColumns(panel.kolom);
  const hasPairs = groups.some((group) => group.kind === "pair");
  const width = groups.reduce((acc, group) => acc + (group.kind === "pair" ? 2 : 1), 0);

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[56rem] border-collapse text-sm">
        <thead className="border-b border-slate-200 bg-slate-50/80 text-left">
          <tr>
            {groups.map((group) =>
              group.kind === "single" ? (
                <th
                  key={group.column.kunci}
                  scope="col"
                  rowSpan={hasPairs ? 2 : 1}
                  className={`px-3 py-2.5 align-bottom text-xs font-semibold tracking-wide text-slate-600 ${isDimension(group.column, panel.baris) ? "" : "text-right"}`}
                >
                  {group.column.judul}
                </th>
              ) : (
                <Th
                  key={group.count.kunci}
                  scope="colgroup"
                  colSpan={2}
                  className="border-l border-slate-200 text-center"
                >
                  {group.name}
                </Th>
              ),
            )}
          </tr>
          {hasPairs ? (
            <tr className="border-t border-slate-100">
              {groups.map((group) =>
                group.kind === "pair" ? (
                  [
                    <Th key={group.count.kunci} right className="border-l border-slate-200">
                      CASE
                    </Th>,
                    <Th key={group.amount.kunci} right>
                      {group.amount.judul.replace(` ${group.name}`, "")}
                    </Th>,
                  ]
                ) : null,
              )}
            </tr>
          ) : null}
        </thead>
        <tbody className="divide-y divide-slate-100">
          {panel.baris.length === 0 ? (
            <tr>
              <td colSpan={width}>
                <EmptyState
                  title={isLoading ? "Memuat…" : "Tidak ada angka untuk penyaring ini."}
                />
              </td>
            </tr>
          ) : null}

          {panel.baris.map((row, index) => (
            <tr
              key={rowKeyOf(panel, row, index)}
              className="transition-colors duration-150 ease-halus hover:bg-blue-50/50"
            >
              {groups.map((group) => {
                if (group.kind === "pair") {
                  return [
                    <td
                      key={group.count.kunci}
                      className="border-l border-slate-100 px-3 py-2.5 text-right text-[13px] tabular-nums text-slate-600"
                    >
                      {cellText(row[group.count.kunci]) || "0"}
                    </td>,
                    amountCell(row[group.amount.kunci], group.amount.kunci),
                  ];
                }
                const column = group.column;
                if (column.uang) return amountCell(row[column.kunci], column.kunci);
                const cell = row[column.kunci];
                // Pencacah bernilai nol dikirim server sebagai sel kosong (`omitempty`), jadi
                // jenis kolomnya dibaca dari kolomnya, bukan dari sel ini.
                const dimension = isDimension(column, panel.baris);
                return (
                  <td
                    key={column.kunci}
                    className={`px-3 py-2.5 text-[13px] ${dimension ? "font-semibold text-slate-900" : "text-right tabular-nums text-slate-900"}`}
                  >
                    {cellText(cell) || (dimension ? "" : "0")}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
