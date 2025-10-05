import { useState, useEffect } from "react";
import { DataTable } from "@/components/ui/DataTable";
import { IngestionJob, apiClient } from "@/lib/api";
import { triageColumns } from "@/components/triage/columns";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { useAuth } from "@/lib/AuthMockProvider"; // Using your custom auth provider
import toast from "react-hot-toast";

// We will create this component in the next step.
// import TriageJobDetails from "@/components/triage/TriageJobDetails";

export default function TriagePage() {
  const [jobs, setJobs] = useState<IngestionJob[]>([]);
  const [selectedJob, setSelectedJob] = useState<IngestionJob | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { getAccessTokenSilently } = useAuth();

  const fetchJobs = async () => {
    setIsLoading(true);
    try {
      const token = await getAccessTokenSilently();
      const data = await apiClient.getIngestionJobs(token);
      setJobs(data || []);
    } catch (error) {
      console.error("Failed to fetch ingestion jobs:", error);
      toast.error("Failed to fetch ingestion jobs.");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();
  }, []); // Intentionally empty to run once on mount

  const handleRowClick = (job: IngestionJob) => {
    // For now, we just set the selected job.
    // Unlike claims, we don't need a separate details fetch yet.
    setSelectedJob(job);
  };

  const handleDrawerClose = () => {
    setSelectedJob(null);
  };

  const getDrawerTitle = () => {
    if (!selectedJob || !selectedJob.source_uri) {
        return "Triage Errors";
    }
    const filename = selectedJob.source_uri.split('/').pop();
    return `Triage Errors: ${filename}`;
  };
  
  const getDrawerDescription = () => {
    if (!selectedJob) return "View and correct errors from an upload.";
    const errorCount = selectedJob.initial_error_count || 0;
    const resolvedCount = selectedJob.resolved_rows_count || 0;
    const remaining = errorCount - resolvedCount;
    if (remaining > 0) {
        return `There are ${remaining} errors remaining to be corrected.`;
    }
    return "All errors for this job have been resolved.";
  }


  if (isLoading) {
    return <div className="h-full flex items-center justify-center">Loading triage queue...</div>;
  }

  return (
    <div className="h-full flex flex-col gap-6">
       <DataTable
          columns={triageColumns}
          data={jobs}
          title="Data Triage Queue"
          description="Select an upload to view and correct ingestion errors."
          onRowClick={handleRowClick}
        />

      {/* --- Details Drawer (slides from the right) --- */}
      <Sheet open={!!selectedJob} onOpenChange={(isOpen) => !isOpen && handleDrawerClose()}>
        <SheetContent side="right" className="sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>{getDrawerTitle()}</SheetTitle>
            <SheetDescription>{getDrawerDescription()}</SheetDescription>
          </SheetHeader>
          {selectedJob && (
            <TriageJobDetails job={selectedJob} />
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}
