// frontend/src/pages/ClaimsPage.tsx

import { useEffect, useState, FormEvent, useRef } from "react";
import { DataTable } from "@/components/ui/DataTable";
import { columns, Claim } from "@/components/claims/columns";
import { DetailsDrawer } from "@/components/DetailsDrawer";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { useAuth } from "../lib/AuthMockProvider";
import { apiClient } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { SendHorizonal } from "lucide-react";
import toast from "react-hot-toast";

// Type for Chat Messages
interface Message {
  sender: 'user' | 'ai';
  content: string;
}

// Type for the AI's response structure
interface AiResponse {
  answer?: {
    actions: {
      type: string;
      payload: any;
    }[];
  };
}

export default function ClaimsPage() {
  // State for the data table and details drawer
  const [claims, setClaims] = useState<Claim[]>([]);
  const [selectedClaim, setSelectedClaim] = useState<any | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { getAccessTokenSilently } = useAuth();
  const chatContainerRef = useRef<HTMLDivElement>(null);

  // State for the Chat UI
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isAiLoading, setIsAiLoading] = useState(false);

  // --- DATA FETCHING AND HANDLERS ---

  const fetchClaims = async () => {
    setIsLoading(true);
    try {
      const token = await getAccessTokenSilently();
      const data = await apiClient.get('/api/insurance/claims', token);
      setClaims(data || []);
    } catch (error) {
      console.error("Failed to fetch claims:", error);
      toast.error("Failed to fetch claims data.");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchClaims();
  }, []); // Depend on nothing to run only once on mount

  const fetchClaimDetails = async (itemId: number) => {
    try {
      const token = await getAccessTokenSilently();
      const data = await apiClient.get(`/api/insurance/claims/${itemId}`, token);
      setSelectedClaim(data);
    } catch (error) {
      toast.error(`Failed to fetch details for claim.`);
      console.error(`Failed to fetch details for claim id ${itemId}:`, error);
    }
  };

  const handleRowClick = (claim: Claim) => {
    fetchClaimDetails(claim.id);
  };

  const handleDrawerClose = () => {
    setSelectedClaim(null);
  };

  const handleSave = async (updatedClaim: any) => {
    const originalClaims = [...claims];
    setClaims(prevClaims => prevClaims.map(c => c.id === updatedClaim.id ? { ...c, business_status: updatedClaim.business_status } : c));
    setSelectedClaim(null);
    const toastId = toast.loading("Saving status update...");

    try {
      const token = await getAccessTokenSilently();
      const payload = { business_status: updatedClaim.business_status };
      await apiClient.patch(`/api/insurance/claims/${updatedClaim.id}`, token, payload);
      toast.success("Status updated successfully!", { id: toastId });
    } catch (error) {
      toast.error("Failed to save status update.", { id: toastId });
      console.error("Failed to save claim:", error);
      setClaims(originalClaims);
    }
  };

  // --- AI CHAT HANDLERS ---

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [messages]);

  const handleAiResponse = (response: AiResponse) => {
    const actions = response.answer?.actions;
    if (!actions || actions.length === 0) {
      setMessages(prev => [...prev, { sender: 'ai', content: "I received a response, but it had no actions to perform." }]);
      return;
    }

    actions.forEach(action => {
      switch (action.type) {
        case 'text_response':
          setMessages(prev => [...prev, { sender: 'ai', content: action.payload }]);
          break;
        case 'render_table':
          toast.success("AI has updated the claims table for you.");
          setClaims(action.payload);
          break;
        case 'open_detail_drawer':
          toast.success(`AI is showing details for claim ${action.payload.claim_id}`);
          setSelectedClaim(action.payload);
          break;
        default:
          console.warn("Received unknown AI action type:", action.type);
      }
    });
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isAiLoading) return;

    const userMessage: Message = { sender: 'user', content: input };
    const newMessages: Message[] = [...messages, userMessage]; // Create the new message history
    
    setMessages(newMessages); // Update the UI immediately
    setInput("");
    setIsAiLoading(true);

    try {
      const token = await getAccessTokenSilently();
      
      // We'll now send a JSON payload
      const requestBody = {
        question: input,
        history: messages, // Send the history *before* the new message
      };

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/insurance/query`, {
        method: "POST",
        headers: { 
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json', // Set content type to JSON
        },
        body: JSON.stringify(requestBody), // Send the JSON string
      });

      if (!response.ok) { throw new Error(`API Error: ${response.statusText}`); }
      
      const data = await response.json();
      handleAiResponse(data);
    } catch (error) {
      toast.error("An error occurred while talking to the AI assistant.");
      console.error("AI Query failed:", error);
      setMessages(prev => [...prev, { sender: 'ai', content: "Sorry, I ran into an error. Please try again." }]);
    } finally {
      setIsAiLoading(false);
    }
  };


  if (isLoading) {
    return <div className="h-full flex items-center justify-center">Loading claims data...</div>;
  }

  return (
    <div className="h-full flex flex-row gap-6">
      {/* --- Column 1: AI Chat --- */}
      <div className="w-[30%] flex-shrink-0">
        <Card className="h-full flex flex-col">
          <CardHeader>
            <CardTitle>Chimera Command</CardTitle>
          </CardHeader>
          <CardContent className="flex-1 flex flex-col gap-4 min-h-0">
            <div ref={chatContainerRef} className="flex-1 overflow-y-auto pr-4 space-y-4">
              {messages.map((msg, index) => (
                <div key={index} className={`flex ${msg.sender === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div className={`p-3 rounded-lg max-w-[80%] break-words ${msg.sender === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}>
                    {msg.content}
                  </div>
                </div>
              ))}
              {isAiLoading && (
                <div className="flex justify-start">
                    <div className="p-3 rounded-lg bg-muted animate-pulse">Thinking...</div>
                </div>
              )}
            </div>
            <form onSubmit={handleSubmit} className="flex items-center gap-2 pt-4 border-t">
              <Textarea
                placeholder="Ask about claims..."
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSubmit(e); } }}
                rows={1}
                className="min-h-[40px] max-h-[150px] resize-y"
                disabled={isAiLoading}
              />
              <Button type="submit" size="icon" disabled={isAiLoading || !input.trim()}>
                <SendHorizonal className="h-4 w-4" />
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>

      {/* --- Column 2: Data Table --- */}
      <div className="w-[70%] flex flex-col min-w-0">
        <DataTable
          columns={columns}
          data={claims}
          title="General Securities Assurance - Policy Claims"
          description="Browse and manage all insurance claims."
          page={1}
          setPage={() => {}}
          hasMore={false}
          onRowClick={handleRowClick}
        />
      </div>

      {/* --- Details Drawer (slides from the right) --- */}
      <Sheet open={!!selectedClaim} onOpenChange={(isOpen) => !isOpen && handleDrawerClose()}>
        <SheetContent side="right" className="sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Claim Details: {selectedClaim?.claim_id}</SheetTitle>
            <SheetDescription>View and edit the details for the selected claim.</SheetDescription>
          </SheetHeader>
          {selectedClaim && (
            <DetailsDrawer
              data={selectedClaim}
                fields={{
                  main: [
                    { key: 'policy_number', label: 'Policy #' },
                    { key: 'claim_type', label: 'Claim Type' },
                    { key: 'date_of_loss', label: 'Date of Loss' },
                    { key: 'claim_amount', label: 'Claim Amount', type: 'currency' },
                    { key: 'adjuster_assigned', label: 'Adjuster' },
                    { key: 'policyholder_name', label: 'Policyholder' },
                    { key: 'customer_level', label: 'Customer Level' },
                    { key: 'customer_since_date', label: 'Customer Since' },
                  ],
                  status: [
                    { key: 'business_status', label: 'Status', options: ['Submitted', 'Under Review', 'Flagged for Fraud Review', 'Approved', 'Paid', 'Denied'] },
                  ],
                  comments: [],
                  // ADDED: This new property will render the description in its own section
                  fullWidth: [
                    { key: 'description_of_loss', label: 'Description of Loss' },
                  ],
                }}
                onSave={handleSave}
                onCancel={handleDrawerClose}
                id={selectedClaim.id}
                type="item"
            />
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}
