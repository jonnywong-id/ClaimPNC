import { useState, type ReactNode } from "react";

import { Button } from "@/components/Button";

import { cellText, countOf, formatCount, rowKeyOf } from "./cells";
import { DashboardFilterStrip, DetailExportStrip } from "./controls";
import { Card, CardHeader, Chip, Th } from "./parts";
import type { Cell, DashboardFilterInput, Filter, Panel, Tab } from "./types";

/**
 * Nilai baris ringkasan grid PIC, sekaligus nilai bawaan `tempFilter.UserTeknis` di
 * `PNCGetDashboardOSInbox_Act`. Mengekliknya berarti seluruh PIC.
 */
export const PIC_ALL = "ALL";

/** Banyaknya baris yang tampil sebelum "Lihat … lainnya" — dari rancangan layar. */
const VISIBLE_ROWS = 9;

type Props = {
  tab: Tab;
  panels: Panel[];
  filters: Filter[];
  isLoading: boolean;
  failure: ReactNode;

  selectedPIC: string;
  onSelectPIC: (pic: string) => void;

  draftFilter: DashboardFilterInput;
  onDraftChange: (filter: DashboardFilterInput) => void;
  onApplyFilter: () => void;
  onClearFilter: () => void;
};

/**
 * Tab Outstanding — dua kartu, mengikuti rancangan layar Work Owner 2026-10-14.
 *
 *   kartu 1  export detail klaim, lalu Per PIC dan Per COB berdampingan
 *   kartu 2  tabel silang Kategori OS × Reinsurer × tahun, dengan penyaringnya
 *
 * Isinya tetap isi Pega (`Section/PNCDashboardOS`): judul kolom, label tombol, dan angka
 * datang dari server apa adanya. Yang berubah hanya bentuknya — batang pembanding, urutan
 * terbanyak, dan kolom Total.
 *
 * # Nama PIC tetap TAUTAN
 *
 * Rancangan layarnya tidak menggambarnya, tetapi perilaku itu ada di Pega dan sudah diminta
 * Work Owner: klik nama PIC menyaring Per COB dan tabel silang ke PIC itu
 * (`PNCSetFilterPICDashboardOS`), dan "ALL" mengembalikannya.
 */
export function OutstandingView(props: Props) {
  const { panels, isLoading } = props;
  const [byCount, setByCount] = useState(true);

  const picPanel = panels.find((panel) => panel.penyaring_baris === "pic");
  const others = panels.filter((panel) => panel !== picPanel);
  const cobPanel = others[0];
  const summaryPanel = others[1];

  const allRow = picPanel?.baris.find(
    (row) => cellText(row[picPanel.kolom[0]?.kunci ?? ""]) === PIC_ALL,
  );
  const total = allRow
    ? countOf(allRow[picPanel?.kolom[1]?.kunci ?? ""])
    : undefined;

  return (
    <div className="space-y-4">
      <Card label={props.tab.nama}>
        <CardHeader
          title={props.tab.nama}
          description={props.tab.keterangan}
          aside={
            <>
              {total !== undefined ? (
                <Chip tone="aksen">{formatCount(total)} OS</Chip>
              ) : null}
              <div className="ml-auto flex items-center gap-2">
                <span className="text-xs text-slate-500">Urutkan</span>
                <Button
                  type="button"
                  aria-pressed={byCount}
                  onClick={() => setByCount(!byCount)}
                >
                  OS — terbanyak
                </Button>
              </div>
            </>
          }
        />

        {props.tab.punya_ekspor_detail ? <DetailExportStrip /> : null}

        {props.failure ? (
          <div className="border-b border-slate-200 px-5 py-3">
            {props.failure}
          </div>
        ) : null}

        <div className="grid grid-cols-1 items-start gap-4 p-5 lg:grid-cols-2">
          {picPanel ? (
            <BarTable
              heading="Per PIC"
              panel={picPanel}
              isLoading={isLoading}
              byCount={byCount}
              link={{
                selected: props.selectedPIC || PIC_ALL,
                onSelect: props.onSelectPIC,
              }}
            />
          ) : null}
          {cobPanel ? (
            <BarTable
              heading="Per COB"
              panel={cobPanel}
              isLoading={isLoading}
              byCount={byCount}
              // PIC yang sedang dipilih disebut di kepala Per COB — section Pega
              // menampilkan `tempFilter.UserTeknis` apa adanya, "ALL" sampai sebuah nama
              // diklik. Tanpa ini Per COB yang menyempit tidak menyebut milik siapa
              // angkanya.
              aside={
                picPanel ? (
                  <Chip tone="biru">
                    <span className="sr-only">PIC terpilih: </span>
                    {props.selectedPIC || PIC_ALL}
                  </Chip>
                ) : null
              }
            />
          ) : null}
        </div>
      </Card>

      {summaryPanel ? (
        <Card label="Kategori OS per tahun">
          <CardHeader
            title="Kategori OS per tahun"
            // Dasar tahunnya disebut apa adanya: tanggal klaim DIBUAT, bukan tanggal kejadian
            // — lihat selisih terencana pada grid ketiga.
            aside={<Chip>Tahun = tanggal klaim dibuat</Chip>}
          />
          {props.filters.length > 0 ? (
            <DashboardFilterStrip
              filters={props.filters}
              draft={props.draftFilter}
              onDraftChange={props.onDraftChange}
              onApply={props.onApplyFilter}
              onClear={props.onClearFilter}
            />
          ) : null}
          <SummaryTable panel={summaryPanel} isLoading={isLoading} />
        </Card>
      ) : null}
    </div>
  );
}

/**
 * Tabel ringkas dua kolom (dimensi, jumlah) dengan batang pembanding.
 *
 * Baris "ALL" — baris ringkasan yang memang dikirim kueri Pega di atas grid PIC — selalu di
 * atas, tidak ikut diurutkan, tidak ikut dihitung sebagai salah satu dari sembilan, dan tidak
 * diberi batang: ia jumlah, bukan pembanding.
 */
function BarTable({
  heading,
  panel,
  isLoading,
  byCount,
  link,
  aside,
}: {
  heading: string;
  panel: Panel;
  isLoading: boolean;
  byCount: boolean;
  link?: { selected: string; onSelect: (value: string) => void };
  aside?: ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);

  const nameKey = panel.kolom[0]?.kunci ?? "";
  const countKey = panel.kolom[1]?.kunci ?? "";
  const nameTitle = panel.kolom[0]?.judul ?? "";
  const countTitle = panel.kolom[1]?.judul ?? "";

  const nameOf = (row: Record<string, Cell>) => cellText(row[nameKey]);
  const allRow = panel.baris.find((row) => nameOf(row) === PIC_ALL);
  const rest = panel.baris.filter((row) => row !== allRow);

  const ordered = byCount
    ? [...rest].sort((a, b) => countOf(b[countKey]) - countOf(a[countKey]))
    : rest;
  const shown = expanded ? ordered : ordered.slice(0, VISIBLE_ROWS);
  const hidden = ordered.length - shown.length;
  const max = Math.max(1, ...rest.map((row) => countOf(row[countKey])));
  const sum = rest.reduce((acc, row) => acc + countOf(row[countKey]), 0);

  function nameCell(row: Record<string, Cell>, bold = false) {
    const name = nameOf(row);
    if (!link || name === "") {
      return (
        <span
          className={`block truncate text-[13px] text-slate-900 ${bold ? "font-semibold" : "font-medium"}`}
          title={name}
        >
          {name}
        </span>
      );
    }
    const active = name === link.selected;
    return (
      <button
        type="button"
        onClick={() => link.onSelect(name)}
        aria-current={active ? "true" : undefined}
        title={name}
        className={[
          "block max-w-full truncate rounded text-left text-[13px] text-blue-700",
          "underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
          active || bold ? "font-semibold" : "font-medium",
        ].join(" ")}
      >
        {name}
      </button>
    );
  }

  return (
    <div className="overflow-hidden rounded-kartu border border-slate-200">
      <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50/80 px-4 py-2.5">
        <h3 className="text-sm font-semibold text-slate-900">{heading}</h3>
        <Chip>
          {rest.length} {nameTitle}
        </Chip>
        {aside ? <span className="ml-auto">{aside}</span> : null}
      </div>

      <table className="w-full table-fixed border-collapse text-sm">
        <thead className="border-b border-slate-200 text-left">
          <tr>
            <Th>{nameTitle}</Th>
            <Th right className="w-24">
              {countTitle}
            </Th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {isLoading && panel.baris.length === 0 ? (
            <tr>
              <td colSpan={2} className="px-3 py-6 text-center text-xs text-slate-500">
                Memuat…
              </td>
            </tr>
          ) : null}

          {!isLoading && panel.baris.length === 0 ? (
            <tr>
              <td colSpan={2} className="px-3 py-6 text-center text-xs text-slate-500">
                Tidak ada angka untuk penyaring ini.
              </td>
            </tr>
          ) : null}

          {allRow ? (
            <tr className="bg-slate-50/60">
              <td className="px-3 py-2">{nameCell(allRow, true)}</td>
              <td className="px-3 py-2 text-right text-[13px] font-semibold tabular-nums">
                {cellText(allRow[countKey])}
              </td>
            </tr>
          ) : null}

          {shown.map((row, index) => {
            const value = countOf(row[countKey]);
            return (
              <tr
                key={rowKeyOf(panel, row, index)}
                className="transition-colors duration-150 ease-halus hover:bg-blue-50/50"
              >
                <td className="px-3 py-2">
                  {nameCell(row)}
                  <span className="mt-1.5 block h-1 rounded-full bg-blue-100" aria-hidden="true">
                    <span
                      className="block h-1 rounded-full bg-blue-600"
                      style={{ width: `${Math.max(2, Math.round((value / max) * 100))}%` }}
                    />
                  </span>
                </td>
                <td className="px-3 py-2 text-right align-top text-[13px] font-semibold tabular-nums">
                  {cellText(row[countKey])}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {rest.length > 0 ? (
        hidden > 0 ? (
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="block w-full border-t border-slate-100 bg-slate-50/60 px-4 py-2.5 text-center text-xs font-medium text-blue-700 hover:bg-blue-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            Lihat {hidden} {nameTitle} lainnya
          </button>
        ) : (
          <p className="border-t border-slate-100 bg-slate-50/60 px-4 py-2.5 text-center text-xs font-medium text-slate-600">
            Seluruh {nameTitle} sudah tampil · total {formatCount(sum)}
          </p>
        )
      ) : null}
    </div>
  );
}

/**
 * Tabel silang Kategori OS × Reinsurer × tahun.
 *
 * Dua tambahan dari rancangan layar, keduanya tampilan saja: nama kategori ditulis SEKALI per
 * kelompok, dan kolom Total di kanan. Angka nol ditulis "—" supaya sel yang berisi terbaca.
 *
 * Kolom tahun tetap ditentukan data — server menambahkannya ke `kolom` sesudah dua kolom
 * tetap — sehingga tidak ada tahun yang ditulis di sini.
 */
function SummaryTable({ panel, isLoading }: { panel: Panel; isLoading: boolean }) {
  const [categoryColumn, reinsurerColumn, ...yearColumns] = panel.kolom;
  if (!categoryColumn || !reinsurerColumn) return null;

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead className="border-b border-slate-200 bg-slate-50/80 text-left">
            <tr>
              <Th>{categoryColumn.judul}</Th>
              <Th>{reinsurerColumn.judul}</Th>
              {yearColumns.map((column) => (
                <Th key={column.kunci} right>
                  {column.judul}
                </Th>
              ))}
              <Th right className="border-l border-slate-200 bg-slate-100/70">
                Total
              </Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {panel.baris.length === 0 ? (
              <tr>
                <td
                  colSpan={yearColumns.length + 3}
                  className="px-3 py-6 text-center text-xs text-slate-500"
                >
                  {isLoading ? "Memuat…" : "Tidak ada angka untuk penyaring ini."}
                </td>
              </tr>
            ) : null}

            {panel.baris.map((row, index) => {
              const category = cellText(row[categoryColumn.kunci]);
              const previous = panel.baris[index - 1];
              const first =
                index === 0 ||
                cellText(previous?.[categoryColumn.kunci]) !== category;
              const total = yearColumns.reduce(
                (acc, column) => acc + countOf(row[column.kunci]),
                0,
              );

              return (
                <tr
                  key={rowKeyOf(panel, row, index)}
                  className={[
                    "transition-colors duration-150 ease-halus hover:bg-blue-50/50",
                    first && index > 0 ? "border-t border-slate-200" : "",
                  ].join(" ")}
                >
                  <td className="px-3 py-2.5 text-[13px] font-semibold text-slate-900">
                    {first ? category : <span className="sr-only">{category}</span>}
                  </td>
                  <td className="px-3 py-2.5 text-[13px] text-slate-600">
                    {cellText(row[reinsurerColumn.kunci])}
                  </td>
                  {yearColumns.map((column) => {
                    const value = countOf(row[column.kunci]);
                    return (
                      <td
                        key={column.kunci}
                        className={`px-3 py-2.5 text-right text-[13px] tabular-nums ${value ? "text-slate-900" : "text-slate-300"}`}
                      >
                        {value ? formatCount(value) : "—"}
                      </td>
                    );
                  })}
                  <td className="border-l border-slate-100 bg-slate-50/70 px-3 py-2.5 text-right text-[13px] font-semibold tabular-nums">
                    {formatCount(total)}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Teks dari rancangan layar Work Owner 2026-10-14. */}
      <p className="border-t border-slate-200 px-5 py-3 text-xs text-slate-600">
        Nama kategori ditulis sekali per kelompok. Kolom{" "}
        <span className="font-medium text-slate-800">Total</span> di kanan
        ditambahkan supaya satu baris terbaca tanpa menjumlah kolom tahun sendiri.
      </p>
    </>
  );
}
