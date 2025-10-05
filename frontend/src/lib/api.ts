// This type represents a single ingestion job (the master view)
export interface IngestionJob {
  id: string;
  source_uri: string | null;
  item_type: string;
  status: string;
  total_rows: number | null;
  processed_rows: number | null;
  initial_error_count: number | null;
  resolved_rows_count: number | null;
  started_at: string;
  completed_at: string | null;
  user_id: number | null;
}

// This type represents a single errored row that needs triage (the detail view)
export interface IngestionError {
  id: string;
  job_id: string;
  timestamp: string;
  original_row_data: Record<string, any>;
  reason_for_failure: string;
  status: 'new' | 'pending_revalidation' | 'resolved' | 'ignored';
  corrected_data: Record<string, any> | null;
  resolved_at: string | null;
  resolved_by: number | null;
}

export const apiClient = {
  request: async (
    url: string,
    method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
    token: string,
    body?: any
  ) => {
    try {
      const headers: HeadersInit = {
        Authorization: `Bearer ${token}`,
      };

      // Only add the Content-Type header if we are sending a body
      if (body) {
        headers['Content-Type'] = 'application/json';
      }

      const options: RequestInit = {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      };

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}${url}`, options);

      if (!response.ok) {
        let errorData = { message: `HTTP error! status: ${response.status}` };
        try {
          errorData = await response.json();
        } catch (e) {
          // Ignore if the error response is not JSON
        }
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      if (response.status === 204) {
        return null; // Handle no content responses
      }

      return response.json();

    } catch (error) {
      console.error(`API request failed: ${method} ${url}`, error);
      throw error;
    }
  },

  get: async (url: string, token: string) => apiClient.request(url, 'GET', token),

  post: async (url: string, token: string, body: any) => apiClient.request(url, 'POST', token, body),

  patch: async (url: string, token: string, body: any) => apiClient.request(url, 'PATCH', token, body),
 /**
   * Fetches a paginated list of all ingestion jobs.
   */
  getIngestionJobs: async (token: string, limit = 20, offset = 0): Promise<IngestionJob[]> => {
    return apiClient.get(`/api/ingestion-jobs?limit=${limit}&offset=${offset}`, token);
  },

  /**
   * Fetches all unresolved errors for a specific ingestion job.
   * @param token The user's auth token.
   * @param jobId The UUID of the ingestion job.
   */
  getIngestionErrors: async (token: string, jobId: string): Promise<IngestionError[]> => {
    return apiClient.get(`/api/ingestion-jobs/${jobId}/errors`, token);
  },

  /**
   * Submits corrected data for a single errored row.
   * @param token The user's auth token.
   * @param errorId The UUID of the ingestion error record.
   * @param correctedData An object containing the corrected data.
   */
  updateIngestionError: async (token: string, errorId: string, correctedData: Record<string, any>): Promise<IngestionError> => {
    const body = { corrected_data: correctedData };
    return apiClient.patch(`/api/ingestion-errors/${errorId}`, token, body);
  },
};
