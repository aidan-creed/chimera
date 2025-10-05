"use client";

import { ColumnDef } from "@tanstack/react-table";
import { IngestionError } from "@/lib/api";
import { DataTableColumnHeader } from "@/components/ui/DataTableColumnHeader";
import { Badge } from "@/components/ui/badge";

export const errorColumns: ColumnDef<IngestionError>[] = [
  {
    accessorKey: "reason_for_failure",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Reason for Failure" />
    ),
    cell: ({ row }) => {
      const reason = row.getValue("reason_for_failure") as string;
      return <div className="font-medium text-destructive">{reason}</div>;
    },
  },
  {
    accessorKey: "original_row_data",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Problematic Data" />
    ),
    cell: ({ row }) => {
      const data = row.getValue("original_row_data") as object;
      // Display a compact JSON string, truncated if too long
      const jsonString = JSON.stringify(data);
      const displayString = jsonString.length > 100 ? `${jsonString.substring(0, 100)}...` : jsonString;
      return <pre className="text-xs p-2 bg-muted rounded-md"><code>{displayString}</code></pre>;
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
        const status = row.getValue("status") as string;
        return <Badge variant="outline">{status}</Badge>;
    },
    filterFn: (row, id, value) => {
        return value.includes(row.getValue(id));
    }
  },
];
