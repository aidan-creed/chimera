// frontend/src/components/StatusHistory.tsx

import { useState, useEffect } from 'react';
import { useAuth } from '../lib/AuthMockProvider';
import { apiClient } from '@/lib/api';

interface StatusHistoryEvent {
  ID: number;
  event_timestamp: string;
  event_data: {
    old_status: string;
    new_status: string;
  };
  user_name: string;
}

interface StatusHistoryProps {
  id: number;
  type: 'items';
}

export function StatusHistory({ id }: StatusHistoryProps) {
  const [history, setHistory] = useState<StatusHistoryEvent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { getAccessTokenSilently, isAuthenticated } = useAuth();

  useEffect(() => {
    const fetchHistory = async () => {
      if (!id) return;
      
      try {
        setIsLoading(true);
        const token = await getAccessTokenSilently();
        const url = `/api/insurance/claims/${id}/history`; 
        const data = await apiClient.get(url, token);
        setHistory(data || []);
      } catch (error) {
        console.error('StatusHistory: Failed to fetch status history:', error);
      } finally {
        setIsLoading(false);
      }
    };

    if (id && isAuthenticated) {
      fetchHistory();
    }
  }, [id, isAuthenticated, getAccessTokenSilently]);

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Loading history...</p>;
  }

  if (!history || history.length === 0) {
    return <p className="text-sm text-muted-foreground">No status history available for this claim.</p>;
  }

  return (
    <div className="space-y-4">
      {history.map((event) => (
        <div key={event.ID} className="p-3 border rounded-md text-sm">
          <p>
            Status changed from <strong>{event.event_data.old_status || 'N/A'}</strong> to <strong>{event.event_data.new_status}</strong>
          </p>
          <p className="text-sm text-muted-foreground">
            by {event.user_name} on {new Date(event.event_timestamp).toLocaleString()}
          </p>
        </div>
      ))}
    </div>
  );
}
