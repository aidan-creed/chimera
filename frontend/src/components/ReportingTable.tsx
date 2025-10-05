import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface ReportingTableProps {
  title: string;
  headers: string[];
  rows: (string | number)[][];
}

export function ReportingTable({ title, headers, rows }: ReportingTableProps) {
  return (
    <div className="mt-8">
      <h3 className="text-xl font-semibold mb-2">{title}</h3>
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              {headers.map((header) => (
                <TableHead key={header} className={header !== 'Metric' ? 'text-right' : ''}>
                  {header}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row, rowIndex) => (
              <TableRow key={rowIndex}>
                {row.map((cell, cellIndex) => (
                  <TableCell
                    key={cellIndex}
                    className={`${
                      cellIndex === 0 ? "font-medium" : "text-right"
                    }`}
                  >
                    {cell}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
