import { useState, useEffect } from "react";
import { DataTable } from "@/components/ui/DataTable";
import { IngestionJob, apiClient } from "@/lib/api";
import { triageColumns } from "@/components/triage/columns"; // Use the more detailed columns
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { useAuth } from "@/lib/AuthMockProvider";
import toast from "react-hot-toast";
import TriageJobDetails from "@/components/triage/TriageJobDetails";

export default function UploadsPage() {
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
      toast.error("Failed to fetch upload history.");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();
  }, []);

  const handleRowClick = (job: IngestionJob) => {
    setSelectedJob(job);
  };

  const handleDrawerClose = () => {
    setSelectedJob(null);
  };
  
  const getDrawerTitle = () => {
    if (!selectedJob || !selectedJob.source_uri) return "Triage Errors";
    const filename = selectedJob.source_uri.split('/').pop();
    return `Triage Errors: ${filename}`;
  };

  if (isLoading) {
    return <div className="h-full flex items-center justify-center">Loading upload history...</div>;
  }

  return (
    <div className="h-full flex flex-col gap-6">
       <DataTable
          columns={triageColumns}
          data={jobs}
          title="Upload Reporting"
          description="A history of all data uploads. Click a row with errors to begin triage."
          onRowClick={handleRowClick}
        />

      <Sheet open={!!selectedJob} onOpenChange={(isOpen) => !isOpen && handleDrawerClose()}>
        <SheetContent side="right" className="sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>{getDrawerTitle()}</SheetTitle>
            <SheetDescription>View and correct ingestion errors for the selected job.</SheetDescription>
          </SheetHeader>
          {selectedJob && <TriageJobDetails job={selectedJob} />}
        </SheetContent>
      </Sheet>
    </div>
  );
}
