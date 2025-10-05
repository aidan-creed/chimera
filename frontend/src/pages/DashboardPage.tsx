import { useState, useEffect } from "react";
import { StatCard } from "@/components/StatCard";
// FIXED: Removed the unused 'TrendingUp', 'TrendingDown', and 'Hourglass' imports
import { Scale, ReceiptText } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatCurrency } from "@/lib/utils";
import { useAuth0 } from "@auth0/auth0-react";
import { apiClient } from "@/lib/api";


// --- Data Structures from API ---
interface StatusSummary {
  current_status: string;
  status_count: number;
  total_value: number;
  percentage_of_total: number;
}

interface TimeWindowStats {
  new_items_count: number;
  new_items_value: string;
  avg_days_to_pfs: number;
  avg_days_for_pfs_complete: number;
  passed_to_pfs: number;
  completed_by_pfs: number;
}

interface AgingScheduleEntry {
  business_line: string;
  less_than_180_days_count: number;
  less_than_180_days_value: string;
  one_to_two_years_count: number;
  one_to_two_years_value: string;
  over_two_years_count: number;
  over_two_years_value: string;
  total_count: number;
  total_value: string;
}

interface DashboardData {
  item_status_summary: StatusSummary[];
  item_time_windows: {
    "7d": TimeWindowStats;
    "14d": TimeWindowStats;
    "21d": TimeWindowStats;
    "28d": TimeWindowStats;
  };
  items_status_summary: StatusSummary[];
  items_aging_schedule: AgingScheduleEntry[];
}

type TimeWindowKey = keyof DashboardData['item_time_windows'];

// --- Component ---
export function DashboardPage() {
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const { getAccessTokenSilently, isAuthenticated } = useAuth0();

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        const token = await getAccessTokenSilently({
          authorizationParams: {
            audience: import.meta.env.VITE_IDENTITY_PROVIDER_AUDIENCE,
           },
        });
        const data = await apiClient.get('/api/dashboard', token);
        setDashboardData(data);
      } catch (error) {
        console.error("Failed to fetch dashboard data:", error);
      }
    };

    if (isAuthenticated) {
      fetchDashboardData();
    }
  }, [isAuthenticated]);

  if (!dashboardData) {
    return <p>Loading dashboard...</p>;
  }

  const totalitems = dashboardData.item_status_summary.reduce((sum, item) => sum + item.status_count, 0);
  const totalitemValue = dashboardData.item_status_summary.reduce((sum, item) => sum + item.total_value, 0);


  const orderedStatuses = [
    'active',
    'inactive',
    'archived',
  ];

  const orderedStatusSummary = orderedStatuses.map(status => 
    dashboardData.item_status_summary.find(item => item.current_status === status)
  ).filter((item): item is StatusSummary => item !== undefined);

  const windows = ["7d", "14d", "21d", "28d"];
  const timeWindowLabels: Record<string, string> = { "7d": "Last 7 Days", "14d": "Last 14 Days", "21d": "Last 21 Days", "28d": "Last 28 Days" };

  const tableData = [
    ["New Items", ...windows.map(w => dashboardData.items_time_windows[w as TimeWindowKey].new_items_count.toLocaleString())],
    ["Value of New Items", ...windows.map(w => formatCurrency(parseFloat(dashboardData.items_time_windows[w as TimeWindowKey].new_items_value)))],
    ["Passed to PFS", ...windows.map(w => dashboardData.items_time_windows[w as TimeWindowKey].passed_to_pfs.toLocaleString())],
    ["Completed by PFS", ...windows.map(w => dashboardData.items_time_windows[w as TimeWindowKey].completed_by_pfs.toLocaleString())],
    ["Avg Days to PFS", ...windows.map(w => `${dashboardData.items_time_windows[w as TimeWindowKey].avg_days_to_pfs.toFixed(2)}`)],
    ["Avg PFS Completion", ...windows.map(w => `${dashboardData.items_time_windows[w as TimeWindowKey].avg_days_for_pfs_complete.toFixed(2)}`)],
  ];

  return (
    <div className="p-6 space-y-6">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
       <StatCard
          title="Total itemss"
          value={totalitemss.toLocaleString()}
          icon={<ReceiptText className="h-4 w-4 text-muted-foreground" />}
          description={`Total value: ${formatCurrency(totalitemssValue)}`}
        />
        <StatCard
          title="Total itemss"
          value={totalitemss.toLocaleString()}
          icon={<Scale className="h-4 w-4 text-muted-foreground" />}
          description={`Total value: ${formatCurrency(totalitemssValue)}`}
        />
     </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
         <Card>
          <CardHeader>
            <CardTitle>items Status Summary</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Count</TableHead>
                  <TableHead className="text-right">Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {orderedStatusSummary.map(item => (
                  <TableRow key={item.current_status}>
                    <TableCell>{item.current_status}</TableCell>
                    <TableCell className="text-right">{item.status_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(item.total_value)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>items Trends</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Metric</TableHead>
                  {windows.map(w => <TableHead key={w} className="text-right">{timeWindowLabels[w]}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {tableData.map((row, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-medium">{row[0]}</TableCell>
                    {row.slice(1).map((cell, j) => <TableCell key={j} className="text-right">{String(cell)}</TableCell>)}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>items Status Summary</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Count</TableHead>
                  <TableHead className="text-right">Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dashboardData.items_status_summary.map(item => (
                  <TableRow key={item.current_status}>
                    <TableCell>{item.current_status}</TableCell>
                    <TableCell className="text-right">{item.status_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(item.total_value)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>items Aging Schedule</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Business Line</TableHead>
                  <TableHead className="text-right">Less than 180 Days (Count)</TableHead>
                  <TableHead className="text-right">Less than 180 Days (Value)</TableHead>
                  <TableHead className="text-right">1-2 Years (Count)</TableHead>
                  <TableHead className="text-right">1-2 Years (Value)</TableHead>
                  <TableHead className="text-right">Over 2 Years (Count)</TableHead>
                  <TableHead className="text-right">Over 2 Years (Value)</TableHead>
                  <TableHead className="text-right">Total (Count)</TableHead>
                  <TableHead className="text-right">Total (Value)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dashboardData.items_aging_schedule.map((item, index) => (
                  <TableRow key={index}>
                    <TableCell>{item.business_line}</TableCell>
                    <TableCell className="text-right">{item.less_than_180_days_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(parseFloat(item.less_than_180_days_value))}</TableCell>
                    <TableCell className="text-right">{item.one_to_two_years_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(parseFloat(item.one_to_two_years_value))}</TableCell>
                    <TableCell className="text-right">{item.over_two_years_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(parseFloat(item.over_two_years_value))}</TableCell>
                    <TableCell className="text-right">{item.total_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatCurrency(parseFloat(item.total_value))}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

     </div>
    </div>
  );
}
