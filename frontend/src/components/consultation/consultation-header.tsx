'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
    Plus,
    MoreVertical,
    Edit,
    Download,
    Archive,
    Trash2,
    FileText,
    Target,
    Settings,
    Laptop,
    Search,
    Shield,
    Clock,
    MessageSquare
} from 'lucide-react';
import { useUpdateConsultationSession, useDeleteConsultationSession, useExportConsultationSession } from '@/hooks/useConsultations';
import { useConsultationStore } from '@/stores/consultations';
import type { ConsultationSession } from '@/stores/consultations';
import type { ConsultationType } from '@/types';

const typeIcons: Record<ConsultationType, React.ComponentType<{ className?: string }>> = {
    policy: FileText,
    strategy: Target,
    operations: Settings,
    technology: Laptop,
    research: Search,
    compliance: Shield,
};

const statusColors = {
    active: 'bg-green-500/10 text-green-700 border-green-200 dark:bg-green-500/20 dark:text-green-300',
    completed: 'bg-blue-500/10 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300',
    draft: 'bg-yellow-500/10 text-yellow-700 border-yellow-200 dark:bg-yellow-500/20 dark:text-yellow-300',
    archived: 'bg-gray-500/10 text-gray-700 border-gray-200 dark:bg-gray-500/20 dark:text-gray-300',
    paused: 'bg-orange-500/10 text-orange-700 border-orange-200 dark:bg-orange-500/20 dark:text-orange-300',
};

interface ConsultationHeaderProps {
    session: ConsultationSession;
    onNewConsultation: () => void;
}

export function ConsultationHeader({ session, onNewConsultation }: ConsultationHeaderProps) {
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [editTitle, setEditTitle] = useState(session.title);
    const [editSummary, setEditSummary] = useState(session.summary || '');

    const updateSession = useUpdateConsultationSession();
    const deleteSession = useDeleteConsultationSession();
    const exportSession = useExportConsultationSession();
    const { setCurrentSession } = useConsultationStore();

    const TypeIcon = typeIcons[session.type as ConsultationType] || MessageSquare;

    const handleSaveEdit = async () => {
        try {
            await updateSession.mutateAsync({
                id: session.id,
                updates: {
                    title: editTitle,
                    ...(editSummary && { summary: editSummary }),
                },
            });
            setEditDialogOpen(false);
        } catch (error) {
            console.error('Failed to update session:', error);
        }
    };

    const handleStatusChange = async (status: ConsultationSession['status']) => {
        try {
            await updateSession.mutateAsync({
                id: session.id,
                updates: { status },
            });
        } catch (error) {
            console.error('Failed to update session status:', error);
        }
    };

    const handleDelete = async () => {
        if (confirm('Are you sure you want to delete this consultation? This action cannot be undone.')) {
            try {
                await deleteSession.mutateAsync(session.id);
                setCurrentSession(null);
            } catch (error) {
                console.error('Failed to delete session:', error);
            }
        }
    };

    const handleExport = async (format: 'pdf' | 'docx' | 'txt') => {
        try {
            await exportSession.mutateAsync({
                sessionId: session.id,
                format,
            });
        } catch (error) {
            console.error('Failed to export session:', error);
        }
    };

    const formatDate = (date: Date) => {
        return new Intl.DateTimeFormat('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        }).format(date);
    };

    return (
        <div className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="flex items-center justify-between p-4">
                <div className="flex items-center gap-4 min-w-0 flex-1">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-primary/10">
                            <TypeIcon className="h-5 w-5 text-primary" />
                        </div>
                        <div className="min-w-0 flex-1">
                            <h1 className="text-lg font-semibold truncate">{session.title}</h1>
                            <div className="flex items-center gap-3 text-sm text-muted-foreground">
                                <div className="flex items-center gap-1">
                                    <Clock className="h-3 w-3" />
                                    <span>Started {formatDate(session.createdAt)}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                    <MessageSquare className="h-3 w-3" />
                                    <span>{session.messageCount || 0} messages</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="flex items-center gap-2">
                        <Badge
                            variant="outline"
                            className={statusColors[session.status]}
                        >
                            {session.status.charAt(0).toUpperCase() + session.status.slice(1)}
                        </Badge>
                        <Badge variant="secondary" className="capitalize">
                            {session.type}
                        </Badge>
                    </div>
                </div>

                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={onNewConsultation}
                        className="gap-2"
                    >
                        <Plus className="h-4 w-4" />
                        New
                    </Button>

                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                                <MoreVertical className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-48">
                            <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
                                <DialogTrigger asChild>
                                    <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                                        <Edit className="h-4 w-4 mr-2" />
                                        Edit Details
                                    </DropdownMenuItem>
                                </DialogTrigger>
                                <DialogContent>
                                    <DialogHeader>
                                        <DialogTitle>Edit Consultation</DialogTitle>
                                        <DialogDescription>
                                            Update the title and summary for this consultation session.
                                        </DialogDescription>
                                    </DialogHeader>
                                    <div className="space-y-4">
                                        <div className="space-y-2">
                                            <Label htmlFor="edit-title">Title</Label>
                                            <Input
                                                id="edit-title"
                                                value={editTitle}
                                                onChange={(e) => setEditTitle(e.target.value)}
                                            />
                                        </div>
                                        <div className="space-y-2">
                                            <Label htmlFor="edit-summary">Summary (Optional)</Label>
                                            <Textarea
                                                id="edit-summary"
                                                value={editSummary}
                                                onChange={(e) => setEditSummary(e.target.value)}
                                                placeholder="Add a summary of this consultation..."
                                                className="min-h-[100px]"
                                            />
                                        </div>
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="outline"
                                                onClick={() => setEditDialogOpen(false)}
                                            >
                                                Cancel
                                            </Button>
                                            <Button
                                                onClick={handleSaveEdit}
                                                disabled={updateSession.isPending || !editTitle.trim()}
                                            >
                                                Save Changes
                                            </Button>
                                        </div>
                                    </div>
                                </DialogContent>
                            </Dialog>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem onClick={() => handleStatusChange('active')}>
                                <div className="h-2 w-2 rounded-full bg-green-500 mr-2" />
                                Mark Active
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleStatusChange('completed')}>
                                <div className="h-2 w-2 rounded-full bg-blue-500 mr-2" />
                                Mark Completed
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleStatusChange('draft')}>
                                <div className="h-2 w-2 rounded-full bg-orange-500 mr-2" />
                                Mark as Draft
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleStatusChange('archived')}>
                                <Archive className="h-4 w-4 mr-2" />
                                Archive
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem onClick={() => handleExport('pdf')}>
                                <Download className="h-4 w-4 mr-2" />
                                Export as PDF
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleExport('docx')}>
                                <Download className="h-4 w-4 mr-2" />
                                Export as Word
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleExport('txt')}>
                                <Download className="h-4 w-4 mr-2" />
                                Export as Text
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem
                                onClick={handleDelete}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>
        </div>
    );
}