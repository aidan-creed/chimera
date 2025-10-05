"use client";

import { ColumnDef } from "@tanstack/react-table";
import { IngestionJob } from "@/lib/api"; // Assuming your api types are in this path
import { DataTableColumnHeader } from "@/components/ui/DataTableColumnHeader";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { format } from 'date-fns';

// Helper to format numbers with commas
const formatNumber = (num: number | null | undefined) => {
  if (num === null || num === undefined) return 'N/A';
  return new Intl.NumberFormat('en-US').format(num);
};

// Helper to format dates
const formatDate = (dateStr: string | null | undefined) => {
    if (!dateStr) return 'N/A';
    try {
        return format(new Date(dateStr), "MM/dd/yyyy h:mm a");
    } catch (e) {
        return 'Invalid Date';
    }
}


export const triageColumns: ColumnDef<IngestionJob>[] = [
  {
    accessorKey: "source_uri",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Filename" />
    ),
    cell: ({ row }) => {
      const fullPath = row.getValue("source_uri") as string || '';
      const filename = fullPath.split('/').pop() || 'N/A';
      return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <div className="truncate font-medium">{filename}</div>
                </TooltipTrigger>
                <TooltipContent>
                    <p>{fullPath}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
      );
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const status = row.getValue("status") as string;
      const variant: "outline" | "secondary" | "destructive" = 
        status.toLowerCase() === 'completed' ? 'secondary' :
        status.toLowerCase() === 'failed' ? 'destructive' :
        'outline';
      return <Badge variant={variant}>{status}</Badge>;
    },
    filterFn: (row, id, value) => {
        return value.includes(row.getValue(id));
    }
  },
  {
    accessorKey: "initial_error_count",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Errors" />
    ),
    cell: ({ row }) => {
        const errorCount = row.original.initial_error_count;
        const resolvedCount = row.original.resolved_rows_count || 0;
        const remaining = (errorCount || 0) - resolvedCount;
        
        if (errorCount === 0 || remaining <= 0) {
            return <div className="text-center">--</div>;
        }

        return <div className="text-center font-bold text-destructive">{remaining}</div>
    },
  },
  {
    accessorKey: "total_rows",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Total Rows" />
    ),
    cell: ({ row }) => <div className="text-center">{formatNumber(row.getValue("total_rows"))}</div>,
  },
  {
    accessorKey: "started_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Upload Date" />
    ),
    cell: ({ row }) => formatDate(row.getValue("started_at")),
  },
];
