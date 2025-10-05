import { ColumnDef } from "@tanstack/react-table";

export type Upload = {
  id: string; // uuid
  status: "Pending" | "Processing" | "Success" | "Failed";
  report_type: string;
  uploadedAt: string;
  error_details: string | null;
  user: {
    first_name: string;
    last_name: string;
  };
};

export const columns: ColumnDef<Upload>[] = [
  {
    accessorKey: "status",
    header: "Status",
  },
  {
    accessorKey: "report_type",
    header: "Report Type",
  },
  {
    accessorKey: "uploadedAt",
    header: "Uploaded At",
  },
  {
    accessorKey: "error_details",
    header: "Error Details",
  },
  {
    accessorKey: "user.first_name",
    header: "First Name",
  },
  {
    accessorKey: "user.last_name",
    header: "Last Name",
  },
];