'use client';

import { useState, useRef, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Slider } from '@/components/ui/slider';
import { Label } from '@/components/ui/label';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import {
    Mic,
    Square,
    Play,
    Pause,
    Volume2,
    Settings,
    X,
    Loader2,
    CheckCircle,
    AlertCircle
} from 'lucide-react';
import { useConsultationStore } from '@/stores/consultations';
import { useTranscribeAudio, useAvailableVoices } from '@/hooks/useConsultations';

interface VoicePanelProps {
    onVoiceMessage: (content: string) => void;
    onClose: () => void;
}

export function VoicePanel({ onVoiceMessage, onClose }: VoicePanelProps) {
    const [isRecording, setIsRecording] = useState(false);
    const [audioLevel, setAudioLevel] = useState(0);
    const [recordedAudio, setRecordedAudio] = useState<Blob | null>(null);
    const [transcription, setTranscription] = useState('');
    const [isPlaying, setIsPlaying] = useState(false);
    const [recordingTime, setRecordingTime] = useState(0);
    const [showSettings, setShowSettings] = useState(false);

    const mediaRecorderRef = useRef<MediaRecorder | null>(null);
    const audioContextRef = useRef<AudioContext | null>(null);
    const analyserRef = useRef<AnalyserNode | null>(null);
    const streamRef = useRef<MediaStream | null>(null);
    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    const audioRef = useRef<HTMLAudioElement | null>(null);

    const {
        voiceSettings,
        setVoiceSettings,
        transcriptionActive,
        setTranscriptionActive,
    } = useConsultationStore();

    const transcribeAudio = useTranscribeAudio();
    const { data: availableVoices } = useAvailableVoices();

    // Initialize audio context and media recorder
    useEffect(() => {
        return () => {
            cleanup();
        };
    }, []);

    // Recording timer
    useEffect(() => {
        if (isRecording) {
            intervalRef.current = setInterval(() => {
                setRecordingTime(prev => prev + 1);
            }, 1000);
        } else {
            if (intervalRef.current) {
                clearInterval(intervalRef.current);
            }
            setRecordingTime(0);
        }

        return () => {
            if (intervalRef.current) {
                clearInterval(intervalRef.current);
            }
        };
    }, [isRecording]);

    const cleanup = () => {
        if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
            mediaRecorderRef.current.stop();
        }
        if (streamRef.current) {
            streamRef.current.getTracks().forEach(track => track.stop());
        }
        if (audioContextRef.current) {
            audioContextRef.current.close();
        }
        if (intervalRef.current) {
            clearInterval(intervalRef.current);
        }
    };

    const startRecording = async () => {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    echoCancellation: true,
                    noiseSuppression: true,
                    autoGainControl: true,
                }
            });

            streamRef.current = stream;

            // Set up audio level monitoring
            audioContextRef.current = new AudioContext();
            analyserRef.current = audioContextRef.current.createAnalyser();
            const source = audioContextRef.current.createMediaStreamSource(stream);
            source.connect(analyserRef.current);

            analyserRef.current.fftSize = 256;
            const bufferLength = analyserRef.current.frequencyBinCount;
            const dataArray = new Uint8Array(bufferLength);

            const updateAudioLevel = () => {
                if (analyserRef.current && isRecording) {
                    analyserRef.current.getByteFrequencyData(dataArray);
                    const average = dataArray.reduce((a, b) => a + b) / bufferLength;
                    setAudioLevel(average / 255);
                    requestAnimationFrame(updateAudioLevel);
                }
            };
            updateAudioLevel();

            // Set up media recorder
            mediaRecorderRef.current = new MediaRecorder(stream, {
                mimeType: 'audio/webm;codecs=opus'
            });

            const chunks: BlobPart[] = [];
            mediaRecorderRef.current.ondataavailable = (event) => {
                if (event.data.size > 0) {
                    chunks.push(event.data);
                }
            };

            mediaRecorderRef.current.onstop = () => {
                const audioBlob = new Blob(chunks, { type: 'audio/webm;codecs=opus' });
                setRecordedAudio(audioBlob);

                // Auto-transcribe if enabled
                if (transcriptionActive) {
                    handleTranscribe(audioBlob);
                }
            };

            mediaRecorderRef.current.start();
            setIsRecording(true);
        } catch (error) {
            console.error('Failed to start recording:', error);
            alert('Failed to access microphone. Please check your permissions.');
        }
    };

    const stopRecording = () => {
        if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
            mediaRecorderRef.current.stop();
        }
        if (streamRef.current) {
            streamRef.current.getTracks().forEach(track => track.stop());
        }
        if (audioContextRef.current) {
            audioContextRef.current.close();
        }
        setIsRecording(false);
        setAudioLevel(0);
    };

    const handleTranscribe = async (audioBlob: Blob) => {
        try {
            const result = await transcribeAudio.mutateAsync({
                audioBlob,
                options: { language: voiceSettings.language },
            });
            setTranscription(result.text);
        } catch (error) {
            console.error('Transcription failed:', error);
        }
    };

    const playRecording = () => {
        if (recordedAudio) {
            const audioUrl = URL.createObjectURL(recordedAudio);
            audioRef.current = new Audio(audioUrl);
            audioRef.current.onended = () => {
                setIsPlaying(false);
                URL.revokeObjectURL(audioUrl);
            };
            audioRef.current.play();
            setIsPlaying(true);
        }
    };

    const pausePlayback = () => {
        if (audioRef.current) {
            audioRef.current.pause();
            setIsPlaying(false);
        }
    };

    const sendVoiceMessage = () => {
        if (transcription.trim()) {
            onVoiceMessage(transcription);
            resetRecording();
        }
    };

    const resetRecording = () => {
        setRecordedAudio(null);
        setTranscription('');
        setIsPlaying(false);
        if (audioRef.current) {
            audioRef.current.pause();
            audioRef.current = null;
        }
    };

    const formatTime = (seconds: number) => {
        const mins = Math.floor(seconds / 60);
        const secs = seconds % 60;
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    };

    return (
        <Card className="border-b-0 rounded-b-none">
            <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                    <CardTitle className="text-base flex items-center gap-2">
                        <Mic className="h-4 w-4" />
                        Voice Input
                    </CardTitle>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setShowSettings(!showSettings)}
                            className={showSettings ? 'bg-accent' : ''}
                        >
                            <Settings className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={onClose}>
                            <X className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            </CardHeader>

            <CardContent className="space-y-4">
                {/* Settings Panel */}
                {showSettings && (
                    <div className="space-y-4 p-4 bg-muted/50 rounded-lg">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <Label>Voice</Label>
                                <Select
                                    value={voiceSettings.voice}
                                    onValueChange={(value) => setVoiceSettings({ voice: value })}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="default">Default</SelectItem>
                                        {availableVoices?.map((voice) => (
                                            <SelectItem key={voice.id} value={voice.id}>
                                                {voice.name} ({voice.language})
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="space-y-2">
                                <Label>Language</Label>
                                <Select
                                    value={voiceSettings.language}
                                    onValueChange={(value) => setVoiceSettings({ language: value })}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="en-US">English (US)</SelectItem>
                                        <SelectItem value="en-GB">English (UK)</SelectItem>
                                        <SelectItem value="es-ES">Spanish</SelectItem>
                                        <SelectItem value="fr-FR">French</SelectItem>
                                        <SelectItem value="de-DE">German</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                        </div>

                        <div className="space-y-2">
                            <Label>Speech Rate: {voiceSettings.speechRate}x</Label>
                            <Slider
                                value={[voiceSettings.speechRate]}
                                onValueChange={([value]) => setVoiceSettings({ speechRate: value })}
                                min={0.5}
                                max={2}
                                step={0.1}
                                className="w-full"
                            />
                        </div>

                        <div className="flex items-center gap-4">
                            <label className="flex items-center gap-2 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={transcriptionActive}
                                    onChange={(e) => setTranscriptionActive(e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm">Auto-transcribe</span>
                            </label>

                            <label className="flex items-center gap-2 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={voiceSettings.autoPlayResponses}
                                    onChange={(e) => setVoiceSettings({ autoPlayResponses: e.target.checked })}
                                    className="rounded"
                                />
                                <span className="text-sm">Auto-play responses</span>
                            </label>
                        </div>
                    </div>
                )}

                {/* Recording Controls */}
                <div className="flex items-center justify-center gap-4">
                    {!isRecording && !recordedAudio && (
                        <Button
                            onClick={startRecording}
                            size="lg"
                            className="gap-2"
                        >
                            <Mic className="h-5 w-5" />
                            Start Recording
                        </Button>
                    )}

                    {isRecording && (
                        <>
                            <div className="flex items-center gap-3">
                                <div className="flex items-center gap-2">
                                    <div className="w-3 h-3 bg-red-500 rounded-full animate-pulse" />
                                    <span className="text-sm font-medium">Recording</span>
                                    <Badge variant="outline">{formatTime(recordingTime)}</Badge>
                                </div>

                                {/* Audio Level Indicator */}
                                <div className="flex items-center gap-1">
                                    {Array.from({ length: 10 }).map((_, i) => (
                                        <div
                                            key={i}
                                            className={`w-1 h-4 rounded-full transition-colors ${audioLevel * 10 > i ? 'bg-green-500' : 'bg-muted'
                                                }`}
                                        />
                                    ))}
                                </div>
                            </div>

                            <Button
                                onClick={stopRecording}
                                variant="destructive"
                                size="lg"
                                className="gap-2"
                            >
                                <Square className="h-5 w-5" />
                                Stop
                            </Button>
                        </>
                    )}
                </div>

                {/* Recorded Audio Controls */}
                {recordedAudio && (
                    <div className="space-y-4">
                        <div className="flex items-center justify-center gap-2">
                            <Button
                                onClick={isPlaying ? pausePlayback : playRecording}
                                variant="outline"
                                size="sm"
                                className="gap-2"
                            >
                                {isPlaying ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                                {isPlaying ? 'Pause' : 'Play'}
                            </Button>

                            <Button
                                onClick={() => handleTranscribe(recordedAudio)}
                                variant="outline"
                                size="sm"
                                disabled={transcribeAudio.isPending}
                                className="gap-2"
                            >
                                {transcribeAudio.isPending ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                    <Volume2 className="h-4 w-4" />
                                )}
                                Transcribe
                            </Button>

                            <Button
                                onClick={resetRecording}
                                variant="outline"
                                size="sm"
                            >
                                Reset
                            </Button>
                        </div>

                        {/* Transcription */}
                        {transcription && (
                            <div className="space-y-3">
                                <div className="p-3 bg-muted/50 rounded-lg">
                                    <div className="flex items-center gap-2 mb-2">
                                        <CheckCircle className="h-4 w-4 text-green-600" />
                                        <span className="text-sm font-medium">Transcription</span>
                                    </div>
                                    <p className="text-sm">{transcription}</p>
                                </div>

                                <div className="flex justify-center gap-2">
                                    <Button
                                        onClick={sendVoiceMessage}
                                        className="gap-2"
                                    >
                                        <Mic className="h-4 w-4" />
                                        Send Voice Message
                                    </Button>
                                    <Button
                                        onClick={() => setTranscription('')}
                                        variant="outline"
                                    >
                                        Clear
                                    </Button>
                                </div>
                            </div>
                        )}

                        {transcribeAudio.isError && (
                            <div className="flex items-center gap-2 p-3 bg-destructive/10 text-destructive rounded-lg">
                                <AlertCircle className="h-4 w-4" />
                                <span className="text-sm">Failed to transcribe audio. Please try again.</span>
                            </div>
                        )}
                    </div>
                )}

                {/* Instructions */}
                <div className="text-xs text-muted-foreground text-center space-y-1">
                    <p>💡 <strong>Tip:</strong> Speak clearly and avoid background noise for better transcription</p>
                    <p>🎤 <strong>Quality:</strong> Use a good microphone for best results</p>
                </div>
            </CardContent>
        </Card>
    );
}