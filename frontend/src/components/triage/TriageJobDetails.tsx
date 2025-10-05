import { useState, useEffect } from "react";
import { IngestionJob, IngestionError, apiClient } from "@/lib/api";
import { useAuth } from "@/lib/AuthMockProvider";
import toast from "react-hot-toast";
import { DataTable } from "@/components/ui/DataTable";
import { errorColumns } from "./errorColumns";

interface TriageJobDetailsProps {
  job: IngestionJob;
}

export default function TriageJobDetails({ job }: TriageJobDetailsProps) {
  const [errors, setErrors] = useState<IngestionError[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { getAccessTokenSilently } = useAuth();

  useEffect(() => {
    // Fetch errors whenever the selected job changes
    if (job?.id) {
      const fetchErrors = async () => {
        setIsLoading(true);
        try {
          const token = await getAccessTokenSilently();
          const data = await apiClient.getIngestionErrors(token, job.id);
          setErrors(data || []);
        } catch (error) {
          console.error(`Failed to fetch errors for job ${job.id}:`, error);
          toast.error("Failed to fetch error details.");
        } finally {
          setIsLoading(false);
        }
      };

      fetchErrors();
    }
  }, [job, getAccessTokenSilently]);

  const handleRowClick = (error: IngestionError) => {
    // This is where we will open the editing form in the next step
    console.log("Selected error to edit:", error);
    toast.info("Editing form coming soon!");
  };

  if (isLoading) {
    return <div className="p-4">Loading error details...</div>;
  }

  return (
    <div className="mt-4 border-t pt-4">
      {errors.length > 0 ? (
        <DataTable
          columns={errorColumns}
          data={errors}
          onRowClick={handleRowClick}
          // We can remove the title/description for a cleaner look inside the drawer
        />
      ) : (
        <p className="text-center text-muted-foreground p-4">
          No correctable errors found for this job.
        </p>
      )}
    </div>
  );
}
