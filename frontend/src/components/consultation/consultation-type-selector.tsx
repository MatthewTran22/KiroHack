'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
    FileText,
    Target,
    Settings,
    Laptop,
    Search,
    Shield,
    ArrowRight,
    Loader2
} from 'lucide-react';
import type { ConsultationType } from '@/types';

interface ConsultationTypeOption {
    id: ConsultationType;
    name: string;
    description: string;
    icon: React.ComponentType<{ className?: string }>;
    examples: string[];
    color: string;
}

const consultationTypes: ConsultationTypeOption[] = [
    {
        id: 'policy',
        name: 'Policy Analysis',
        description: 'Analyze existing policies, draft new regulations, and assess policy impacts',
        icon: FileText,
        examples: ['Policy impact assessment', 'Regulatory compliance review', 'Legislative drafting'],
        color: 'bg-blue-500/10 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-800',
    },
    {
        id: 'strategy',
        name: 'Strategic Planning',
        description: 'Develop long-term strategies, set objectives, and plan implementation roadmaps',
        icon: Target,
        examples: ['Strategic planning', 'Goal setting', 'Implementation roadmaps'],
        color: 'bg-green-500/10 text-green-700 border-green-200 dark:bg-green-500/20 dark:text-green-300 dark:border-green-800',
    },
    {
        id: 'operations',
        name: 'Operations Management',
        description: 'Optimize processes, improve efficiency, and manage day-to-day operations',
        icon: Settings,
        examples: ['Process optimization', 'Workflow improvement', 'Resource allocation'],
        color: 'bg-orange-500/10 text-orange-700 border-orange-200 dark:bg-orange-500/20 dark:text-orange-300 dark:border-orange-800',
    },
    {
        id: 'technology',
        name: 'Technology Advisory',
        description: 'Technology adoption, digital transformation, and IT strategy guidance',
        icon: Laptop,
        examples: ['Digital transformation', 'Technology assessment', 'IT strategy'],
        color: 'bg-purple-500/10 text-purple-700 border-purple-200 dark:bg-purple-500/20 dark:text-purple-300 dark:border-purple-800',
    },
    {
        id: 'research',
        name: 'Research & Analysis',
        description: 'Conduct research, analyze data, and provide evidence-based insights',
        icon: Search,
        examples: ['Market research', 'Data analysis', 'Evidence gathering'],
        color: 'bg-indigo-500/10 text-indigo-700 border-indigo-200 dark:bg-indigo-500/20 dark:text-indigo-300 dark:border-indigo-800',
    },
    {
        id: 'compliance',
        name: 'Compliance & Risk',
        description: 'Ensure regulatory compliance, assess risks, and develop mitigation strategies',
        icon: Shield,
        examples: ['Risk assessment', 'Compliance audit', 'Security review'],
        color: 'bg-red-500/10 text-red-700 border-red-200 dark:bg-red-500/20 dark:text-red-300 dark:border-red-800',
    },
];

interface ConsultationTypeSelectorProps {
    onStartConsultation: (type: ConsultationType, title?: string, context?: string) => void;
    isLoading?: boolean;
}

export function ConsultationTypeSelector({ onStartConsultation, isLoading }: ConsultationTypeSelectorProps) {
    const [selectedType, setSelectedType] = useState<ConsultationType | null>(null);
    const [title, setTitle] = useState('');
    const [context, setContext] = useState('');
    const [showDetails, setShowDetails] = useState(false);

    const selectedTypeOption = consultationTypes.find(type => type.id === selectedType);

    const handleTypeSelect = (type: ConsultationType) => {
        setSelectedType(type);
        setShowDetails(true);

        // Set default title based on type
        const typeOption = consultationTypes.find(t => t.id === type);
        if (typeOption) {
            setTitle(`${typeOption.name} Session`);
        }
    };

    const handleStart = () => {
        if (selectedType) {
            onStartConsultation(selectedType, title || undefined, context || undefined);
        }
    };

    const handleQuickStart = (type: ConsultationType) => {
        onStartConsultation(type);
    };

    if (showDetails && selectedTypeOption) {
        return (
            <div className="space-y-6">
                <div className="flex items-center gap-3">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            setShowDetails(false);
                            setSelectedType(null);
                            setTitle('');
                            setContext('');
                        }}
                    >
                        ← Back
                    </Button>
                    <div className="flex items-center gap-2">
                        <selectedTypeOption.icon className="h-5 w-5" />
                        <h2 className="text-xl font-semibold">{selectedTypeOption.name}</h2>
                    </div>
                </div>

                <Card>
                    <CardHeader>
                        <CardTitle>Consultation Details</CardTitle>
                        <CardDescription>
                            Provide additional context to help the AI assistant better understand your needs.
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="space-y-2">
                            <Label htmlFor="title">Session Title</Label>
                            <Input
                                id="title"
                                value={title}
                                onChange={(e) => setTitle(e.target.value)}
                                placeholder="Enter a descriptive title for this consultation"
                            />
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="context">Context & Background (Optional)</Label>
                            <Textarea
                                id="context"
                                value={context}
                                onChange={(e) => setContext(e.target.value)}
                                placeholder="Provide any relevant background information, specific requirements, or context that would help the AI assistant provide better guidance..."
                                className="min-h-[120px]"
                            />
                        </div>

                        <div className="space-y-2">
                            <Label>Example Use Cases</Label>
                            <div className="flex flex-wrap gap-2">
                                {selectedTypeOption.examples.map((example) => (
                                    <Badge key={example} variant="secondary" className="text-xs">
                                        {example}
                                    </Badge>
                                ))}
                            </div>
                        </div>

                        <div className="flex gap-3 pt-4">
                            <Button
                                onClick={handleStart}
                                disabled={isLoading || !title.trim()}
                                className="gap-2"
                            >
                                {isLoading ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                    <ArrowRight className="h-4 w-4" />
                                )}
                                Start Consultation
                            </Button>
                            <Button
                                variant="outline"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (selectedType) {
                                        handleQuickStart(selectedType);
                                    }
                                }}
                                disabled={isLoading}
                            >
                                Quick Start
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>Choose Consultation Type</CardTitle>
                    <CardDescription>
                        Select the type of consultation that best matches your needs. Each type is optimized for specific government use cases.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {consultationTypes.map((type) => {
                            const Icon = type.icon;
                            return (
                                <Card
                                    key={type.id}
                                    className={`cursor-pointer transition-all hover:shadow-md border-2 ${selectedType === type.id
                                        ? 'border-primary shadow-md'
                                        : 'border-border hover:border-primary/50'
                                        }`}
                                    onClick={() => handleTypeSelect(type.id)}
                                >
                                    <CardHeader className="pb-3">
                                        <div className="flex items-center gap-3">
                                            <div className={`p-2 rounded-lg ${type.color}`}>
                                                <Icon className="h-5 w-5" />
                                            </div>
                                            <div className="flex-1">
                                                <CardTitle className="text-base">{type.name}</CardTitle>
                                            </div>
                                        </div>
                                    </CardHeader>
                                    <CardContent className="pt-0">
                                        <CardDescription className="text-sm mb-3">
                                            {type.description}
                                        </CardDescription>
                                        <div className="flex flex-wrap gap-1">
                                            {type.examples.slice(0, 2).map((example) => (
                                                <Badge key={example} variant="outline" className="text-xs">
                                                    {example}
                                                </Badge>
                                            ))}
                                            {type.examples.length > 2 && (
                                                <Badge variant="outline" className="text-xs">
                                                    +{type.examples.length - 2} more
                                                </Badge>
                                            )}
                                        </div>
                                        <div className="mt-3 pt-3 border-t">
                                            <Button
                                                size="sm"
                                                variant="ghost"
                                                className="w-full gap-2 text-xs"
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    handleQuickStart(type.id);
                                                }}
                                                disabled={isLoading}
                                            >
                                                {isLoading ? (
                                                    <Loader2 className="h-3 w-3 animate-spin" />
                                                ) : (
                                                    <ArrowRight className="h-3 w-3" />
                                                )}
                                                Quick Start
                                            </Button>
                                        </div>
                                    </CardContent>
                                </Card>
                            );
                        })}
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}