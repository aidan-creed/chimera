// frontend/src/components/claims/columns.tsx

import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/ui/DataTableColumnHeader";
import { formatCurrency } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";

export type Claim = {
  id: number;
  claim_id: string;
  policy_number: string;
  policyholder_id: string;
  claim_type: string;
  date_of_loss: string;
  claim_amount: string;
  business_status: string;
  adjuster_assigned: string;
};

export const columns: ColumnDef<Claim>[] = [
  {
    accessorKey: "claim_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Claim ID" />
    ),
  },
  {
    accessorKey: "policy_number",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Policy #" />
    ),
  },
  {
    accessorKey: "business_status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const status = row.getValue("business_status") as string;
      return <StatusBadge status={status} />;
    },
  },
  {
    accessorKey: "claim_amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Amount" />
    ),
    cell: ({ row }) => {
      // Format the amount as currency
      const amount = parseFloat(row.getValue("claim_amount"));
      return <div className="text-right">{formatCurrency(amount)}</div>;
    },
  },
  {
    accessorKey: "date_of_loss",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Date of Loss" />
    ),
    cell: ({ row }) => {
      const date = new Date(row.getValue("date_of_loss"));
      return <span>{date.toLocaleDateString()}</span>;
    },
  },
  {
    accessorKey: "adjuster_assigned",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Adjuster" />
    ),
  },
];
