import { Badge } from "@/components/ui/badge";

interface StatusBadgeProps {
  status: string;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const colorMap: { [key: string]: "default" | "secondary" | "destructive" } = {
    "Paid": "default",
    "Approved": "default",
    "Submitted": "secondary",
    "Under Review": "secondary",
    "Denied": "destructive",
    "Flagged for Fraud Review": "destructive",
  };

  const variant = colorMap[status] || "secondary";

  return <Badge variant={variant}>{status}</Badge>;
}
