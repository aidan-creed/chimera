import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import toast from "react-hot-toast";
import { useAuth } from "../lib/AuthMockProvider";

interface UploadReportModalProps {
  onClose: () => void;
  onUploadSuccess: () => void;
}

const ALLOWED_REPORT_TYPES = [
  "PLACE_HOLDER",
];

export function UploadReportModal({ onClose, onUploadSuccess }: UploadReportModalProps) {
  const [selectedReportType, setSelectedReportType] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const { getAccessTokenSilently } = useAuth();

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files[0]) {
      setSelectedFile(event.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!selectedReportType) {
      toast.error("Please select a report type.");
      return;
    }
    if (!selectedFile) {
      toast.error("Please select a file to upload.");
      return;
    }

    setIsUploading(true);
    
    try {
      // 1. Get the access token
      const token = await getAccessTokenSilently({
        authorizationParams: {
          audience: import.meta.env.VITE_IDENTITY_PROVIDER_AUDIENCE,
        },
      });

      // 2. Prepare form data and headers
      const formData = new FormData();
      formData.append("report_file", selectedFile);

      const headers = {
        Authorization: `Bearer ${token}`,
      };

      // 3. Make the secure fetch request
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/upload/${selectedReportType}`, {
        method: "POST",
        headers, // No 'Content-Type', the browser sets it for FormData
        body: formData,
      });

      if (response.ok) {
        toast.success("Upload successful! Processing has begun.");
        onUploadSuccess();
        onClose();
      } else {
        const errorData = await response.json();
        toast.error(`Upload failed: ${errorData.message || response.statusText}`);
      }
    } catch (error: any) {
      toast.error(`An error occurred: ${error.message}`);
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="grid gap-4 py-4">
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="reportType" className="text-right">
          Report Type
        </Label>
        <Select onValueChange={setSelectedReportType} value={selectedReportType || ""}>
          <SelectTrigger className="col-span-3">
            <SelectValue placeholder="Select a report type" />
          </SelectTrigger>
          <SelectContent>
            {ALLOWED_REPORT_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="file" className="text-right">
          File
        </Label>
        <Input id="file" type="file" className="col-span-3" onChange={handleFileChange} />
      </div>
      <div className="flex flex-col gap-2">
        <Button onClick={handleUpload} disabled={isUploading || !selectedReportType || !selectedFile} className="w-full">
          {isUploading ? "Uploading..." : "Upload"}
        </Button>
      </div>
    </div>
  );
}
