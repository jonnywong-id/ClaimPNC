import { useState } from "react";

import { Button } from "@/components/Button";
import { DataTable, type Column } from "@/components/DataTable";
import { ErrorMessage } from "@/components/ErrorMessage";
import { formatDateTimeWIB } from "@/components/format";

import {
  useOpenPaymentDocument,
  usePaymentDetail,
  useSavePayment,
} from "./api";
import { formatAmount } from "./cells";
import { messageOf } from "./controls";
import { Card, CardHeader, Chip, FieldLabel, inputClass } from "./parts";
import type {
  DecisionRule,
  PaymentDetail,
  PaymentDocument,
  PaymentHistoryRow,
  PaymentSaveResponse,
} from "./types";

/** Kode "Tidak Setuju" — Prompt List `AcceptanceStatus`. */
const STATUS_NOT_ACCEPTED = "2";
/** Kode "Setuju" — jalur yang ditahan server selama rantai transfer kasir belum lengkap. */
const STATUS_ACCEPTED = "1";

type Props = {
  acceptedNo: string;
  rule: DecisionRule;
  onSaved: (status: string, result: PaymentSaveResponse) => void;
};

/**
 * Panel detail Payment Klaim Akseptasi — yang dibuka tombol DETAILS.
 *
 * Susunannya mengikuti `Section/Akseptasi_PaymentLeader1`: "Detail Penerima Akseptasi",
 * "Persetujuan Akseptasi" (Status Approval, Remark Approval, SIMPAN), Dokumen, dan grid
 * "History Payment Akseptasi Manager". Seluruh label dari section itu (`D-13`).
 */
export function PaymentDetailPanel({ acceptedNo, rule, onSaved }: Props) {
  const detail = usePaymentDetail(acceptedNo);

  if (detail.isLoading) {
    return (
      <Card label="Detail Penerima Akseptasi">
        <CardHeader
          title="Detail Penerima Akseptasi"
          aside={<Chip>{acceptedNo}</Chip>}
        />
        <p className="px-5 py-4 text-sm text-slate-500" role="status">
          Memuat…
        </p>
      </Card>
    );
  }

  if (detail.error || !detail.data) {
    return (
      <ErrorMessage
        tone="penolakan"
        title="Detail Penerima Akseptasi"
        description={messageOf(
          detail.error,
          "Isi bagian ini tidak dapat diambil.",
        )}
      />
    );
  }

  return <PaymentDetailBody data={detail.data} rule={rule} onSaved={onSaved} />;
}

function PaymentDetailBody({
  data,
  rule,
  onSaved,
}: {
  data: PaymentDetail;
  rule: DecisionRule;
  onSaved: Props["onSaved"];
}) {
  const [status, setStatus] = useState("");
  const [remark, setRemark] = useState("");
  const save = useSavePayment(data.no_akseptasi);
  const openDocument = useOpenPaymentDocument(data.no_akseptasi);

  const approveBlocked = Boolean(rule.alasan_setuju_ditahan);
  const remarkMissing =
    status === STATUS_NOT_ACCEPTED &&
    rule.alasan_wajib_saat_menolak &&
    remark.trim() === "";

  function submit() {
    save.mutate(
      { status, catatan: remark },
      { onSuccess: (result) => onSaved(status, result) },
    );
  }

  const money = (value: string) => formatAmount(value) || "—";
  const text = (value: string) => value.trim() || "—";

  const fields: [string, string][] = [
    ["Nama Tertanggung", text(data.nama_tertanggung)],
    ["No Akseptasi", text(data.no_akseptasi)],
    ["Penerima Klaim", text(data.penerima_klaim)],
    ["Tipe Pembayaran", text(data.tipe_pembayaran)],
    ["Gross Value", money(data.gross_value)],
    ["Nilai Klaim (ASM Share)", money(data.nilai_klaim)],
    ["Mata Uang", text(data.mata_uang)],
    ["Share ASM (%)", text(data.share_asm)],
    ["Remark Akseptasi", text(data.remark_akseptasi)],
  ];

  const historyColumns: Column<PaymentHistoryRow>[] = [
    { key: "no_klaim", title: "No Klaim", value: (row) => row.no_klaim },
    {
      key: "no_akseptasi",
      title: "No Akseptasi",
      value: (row) => row.no_akseptasi,
    },
    {
      key: "tanggal",
      title: "Tanggal",
      value: (row) => row.tanggal ?? "",
      render: (row) => formatDateTimeWIB(row.tanggal ?? undefined),
    },
    {
      key: "note_manager",
      title: "Note Manager",
      value: (row) => row.note_manager,
    },
    { key: "note_pic", title: "Note PIC", value: (row) => row.note_pic },
    { key: "status", title: "Status", value: (row) => row.status },
  ];

  return (
    <div className="space-y-4">
      <Card label="Detail Penerima Akseptasi">
        <CardHeader
          title="Detail Penerima Akseptasi"
          aside={<Chip>{data.no_klaim}</Chip>}
        />
        <dl className="grid gap-x-6 gap-y-3 px-5 py-4 sm:grid-cols-2 lg:grid-cols-3">
          {fields.map(([label, value]) => (
            <div key={label}>
              <dt className="text-xs font-medium text-slate-500">{label}</dt>
              <dd className="mt-0.5 break-words text-sm text-slate-900">
                {value}
              </dd>
            </div>
          ))}
        </dl>
      </Card>

      <Card label="Persetujuan Akseptasi">
        <CardHeader title="Persetujuan Akseptasi" />
        <div className="space-y-3 px-5 py-4">
          <div className="grid gap-3 sm:grid-cols-[14rem_minmax(0,1fr)]">
            <FieldLabel label="Status Approval">
              <select
                value={status}
                onChange={(event) => setStatus(event.target.value)}
                className={`w-full ${inputClass}`}
              >
                <option value="" disabled>
                  —
                </option>
                {data.pilihan_status.map((option) => (
                  <option
                    key={option.nilai}
                    value={option.nilai}
                    disabled={
                      option.nilai === STATUS_ACCEPTED && approveBlocked
                    }
                  >
                    {option.label}
                  </option>
                ))}
              </select>
            </FieldLabel>
            <FieldLabel label={rule.label_alasan ?? "Remark Approval"}>
              <textarea
                value={remark}
                onChange={(event) => setRemark(event.target.value)}
                rows={2}
                className={`w-full ${inputClass}`}
              />
            </FieldLabel>
          </div>

          {approveBlocked ? (
            <p className="rounded-kontrol border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              <span className="font-medium">Persetujuan sedang ditahan. </span>
              {rule.alasan_setuju_ditahan}
            </p>
          ) : null}

          {save.error ? (
            <ErrorMessage
              tone="penolakan"
              title="Keputusan tidak dapat disimpan"
              description={messageOf(save.error, "Keputusan gagal.")}
            />
          ) : null}

          <div className="flex justify-end">
            <Button
              type="button"
              tone="utama"
              disabled={status === "" || remarkMissing || save.isPending}
              onClick={submit}
            >
              SIMPAN
            </Button>
          </div>
        </div>
      </Card>

      <Card label="Dokumen">
        <CardHeader title="Dokumen" />
        {openDocument.error ? (
          <div className="px-5 pt-4">
            <ErrorMessage
              tone="penolakan"
              title="Dokumen"
              description={messageOf(
                openDocument.error,
                "Berkas tidak dapat dibuka.",
              )}
            />
          </div>
        ) : null}
        {data.dokumen.length === 0 ? (
          <p className="px-5 py-4 text-sm text-slate-500">Belum ada data.</p>
        ) : (
          <ul className="divide-y divide-slate-100 px-5 py-2">
            {data.dokumen.map((document) => (
              <DocumentRow
                key={document.kunci}
                document={document}
                disabled={openDocument.isPending}
                onOpen={() => openDocument.mutate(document)}
              />
            ))}
          </ul>
        )}
      </Card>

      <DataTable<PaymentHistoryRow>
        title="History Payment Akseptasi Manager"
        label="History Payment Akseptasi Manager"
        columns={historyColumns}
        rows={data.riwayat}
        rowKey={(row) =>
          `${row.no_akseptasi}|${row.tanggal ?? ""}|${row.kode_status}`
        }
        hideSearch
        showHeaderWhenEmpty
      />
    </div>
  );
}

function DocumentRow({
  document,
  disabled,
  onOpen,
}: {
  document: PaymentDocument;
  disabled: boolean;
  onOpen: () => void;
}) {
  const name = document.nama.trim() || document.kunci;

  return (
    <li className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2">
      <span className="text-sm text-slate-800">
        {/* Lampiran tanpa isi tetap didaftar, seperti di Pega, tetapi bukan tautan. */}
        {document.dapat_dibuka ? (
          <button
            type="button"
            onClick={onOpen}
            disabled={disabled}
            className="font-medium text-blue-700 underline underline-offset-2 hover:text-blue-800 disabled:opacity-60"
          >
            {name}
          </button>
        ) : (
          <span className="font-medium">{name}</span>
        )}
        {document.catatan ? (
          <span className="ml-2 text-slate-500">· {document.catatan}</span>
        ) : null}
      </span>
      {document.tanggal_input ? (
        <span className="text-xs text-slate-500">
          {formatDateTimeWIB(document.tanggal_input)}
        </span>
      ) : null}
    </li>
  );
}
