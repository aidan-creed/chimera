// frontend/src/components/claims/Comments.tsx

import { useState, useEffect } from 'react';
import { useAuth } from '../../lib/AuthMockProvider';
import { apiClient } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import toast from 'react-hot-toast';

interface Comment {
  id: number;
  comment: string;
  created_at: string;
  display_name: string;
  mentioned_users: any[]; // to be used later for @mentions
}

interface CommentsProps {
  itemId: number;
}

export function Comments({ itemId }: CommentsProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [newComment, setNewComment] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const { getAccessTokenSilently } = useAuth();

  const fetchComments = async () => {
    try {
      setIsLoading(true);
      const token = await getAccessTokenSilently();
      const url = `/api/insurance/claims/${itemId}/comments`;
      const data = await apiClient.get(url, token);
      setComments(data || []);
    } catch (error) {
      console.error('Comments: Failed to fetch comments:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (itemId) {
      fetchComments();
    }
  }, [itemId]);

  const handleCommentSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newComment.trim()) return;

    const toastId = toast.loading("Posting comment...");
    try {
      const token = await getAccessTokenSilently();
      const url = `/api/insurance/claims/${itemId}/comments`;
      const payload = { comment_text: newComment };
      await apiClient.post(url, token, payload);
      
      toast.success("Comment posted!", { id: toastId });
      setNewComment(""); // Clear the input
      await fetchComments(); // Refetch comments to show the new one
    } catch (error) {
      toast.error("Failed to post comment.", { id: toastId });
      console.error('Comments: Failed to post comment:', error);
    }
  };

  return (
    <div className="space-y-4">
      {/* Form for adding a new comment */}
      <form onSubmit={handleCommentSubmit} className="space-y-2">
        <Textarea
          placeholder="Add a comment..."
          value={newComment}
          onChange={(e) => setNewComment(e.target.value)}
        />
        <Button type="submit" size="sm" disabled={!newComment.trim()}>
          Post Comment
        </Button>
      </form>

      {/* List of existing comments */}
      <div className="space-y-4">
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading comments...</p>
        ) : comments.length > 0 ? (
          comments.map((comment) => (
            <div key={comment.id} className="p-3 border rounded-md text-sm">
              <p>{comment.comment}</p>
              <p className="text-xs text-muted-foreground mt-2">
                - {comment.display_name} on {new Date(comment.created_at).toLocaleString()}
              </p>
            </div>
          ))
        ) : (
          <p className="text-sm text-muted-foreground">No comments yet.</p>
        )}
      </div>
    </div>
  );
}
