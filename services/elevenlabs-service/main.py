#!/usr/bin/env python3
"""
ElevenLabs Microservice
Provides TTS and STT functionality using the ElevenLabs SDK
"""

import os
import tempfile
import time
from pathlib import Path
from typing import Dict, List, Optional, Any
import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI, File, UploadFile, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse, Response
from pydantic import BaseModel
import uvicorn
from loguru import logger
from dotenv import load_dotenv

from elevenlabs_client import ElevenLabsClient

# Load environment variables
load_dotenv()

# Global client instance
elevenlabs_client: Optional[ElevenLabsClient] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager"""
    global elevenlabs_client

    # Startup
    logger.info("Starting ElevenLabs microservice...")

    api_key = os.getenv("ELEVENLABS_API_KEY")
    if not api_key or api_key == "your-elevenlabs-api-key-here":
        logger.error("ELEVENLABS_API_KEY not configured")
        raise RuntimeError("ElevenLabs API key is required")

    try:
        elevenlabs_client = ElevenLabsClient(api_key)
        await elevenlabs_client.initialize()
        logger.success("ElevenLabs client initialized successfully")
    except Exception as e:
        logger.error(f"Failed to initialize ElevenLabs client: {e}")
        raise

    yield

    # Shutdown
    logger.info("Shutting down ElevenLabs microservice...")
    if elevenlabs_client:
        await elevenlabs_client.close()

# Create FastAPI app
app = FastAPI(
    title="ElevenLabs Microservice",
    description="TTS and STT service using ElevenLabs SDK",
    version="1.0.0",
    lifespan=lifespan
)

# Request/Response models


class TTSRequest(BaseModel):
    text: str
    voice_id: Optional[str] = None
    model_id: Optional[str] = "eleven_monolingual_v1"
    voice_settings: Optional[Dict[str, float]] = None


class STTRequest(BaseModel):
    audio_data: str  # Base64 encoded audio
    model_id: Optional[str] = "whisper-1"
    language: Optional[str] = None


class TTSResponse(BaseModel):
    audio_data: str  # Base64 encoded audio
    voice_id: str
    model_id: str
    duration: float
    size: int
    generated_at: str


class STTResponse(BaseModel):
    text: str
    confidence: float
    language: str
    processing_time: float
    model_id: str


class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
    uptime: float
    elevenlabs_connected: bool


class VoiceInfo(BaseModel):
    voice_id: str
    name: str
    category: str
    description: Optional[str] = None
    preview_url: Optional[str] = None


# Global variables
start_time = time.time()


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    return HealthResponse(
        status="healthy",
        service="elevenlabs-microservice",
        version="1.0.0",
        uptime=time.time() - start_time,
        elevenlabs_connected=elevenlabs_client is not None and elevenlabs_client.is_connected()
    )


@app.post("/tts", response_model=TTSResponse)
async def text_to_speech(request: TTSRequest):
    """Convert text to speech using ElevenLabs TTS"""
    if not elevenlabs_client:
        raise HTTPException(
            status_code=503, detail="ElevenLabs client not initialized")

    try:
        logger.info(f"TTS request: {len(request.text)} characters")

        # Use default voice if not specified
        voice_id = request.voice_id or os.getenv(
            "ELEVENLABS_VOICE_ID", "21m00Tcm4TlvDq8ikWAM")

        # Generate speech
        start_time = time.time()
        audio_data = await elevenlabs_client.text_to_speech(
            text=request.text,
            voice_id=voice_id,
            model_id=request.model_id,
            voice_settings=request.voice_settings
        )
        processing_time = time.time() - start_time

        # Encode audio as base64
        import base64
        audio_b64 = base64.b64encode(audio_data).decode()

        logger.success(
            f"TTS completed in {processing_time:.2f}s, generated {len(audio_data)} bytes")

        return TTSResponse(
            audio_data=audio_b64,
            voice_id=voice_id,
            model_id=request.model_id,
            duration=processing_time,
            size=len(audio_data),
            generated_at=time.strftime("%Y-%m-%d %H:%M:%S")
        )

    except Exception as e:
        logger.error(f"TTS failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/stt", response_model=STTResponse)
async def speech_to_text(request: STTRequest):
    """Convert speech to text using ElevenLabs STT"""
    if not elevenlabs_client:
        raise HTTPException(
            status_code=503, detail="ElevenLabs client not initialized")

    try:
        logger.info("STT request received")

        # Decode base64 audio
        import base64
        audio_data = base64.b64decode(request.audio_data)

        # Transcribe audio
        start_time = time.time()
        result = await elevenlabs_client.speech_to_text(
            audio_data=audio_data,
            model_id=request.model_id,
            language=request.language
        )
        processing_time = time.time() - start_time

        logger.success(f"STT completed in {processing_time:.2f}s")

        return STTResponse(
            text=result["text"],
            confidence=result.get("confidence", 0.95),
            language=result.get("language", request.language or "en"),
            processing_time=processing_time,
            model_id=request.model_id or "whisper-1"
        )

    except Exception as e:
        logger.error(f"STT failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/stt-file", response_model=STTResponse)
async def speech_to_text_file(file: UploadFile = File(...), model_id: str = "whisper-1", language: Optional[str] = None):
    """Convert uploaded audio file to text"""
    if not elevenlabs_client:
        raise HTTPException(
            status_code=503, detail="ElevenLabs client not initialized")

    try:
        logger.info(f"STT file request: {file.filename}")

        # Read file data
        audio_data = await file.read()

        # Transcribe audio
        start_time = time.time()
        result = await elevenlabs_client.speech_to_text(
            audio_data=audio_data,
            model_id=model_id,
            language=language
        )
        processing_time = time.time() - start_time

        logger.success(f"STT file completed in {processing_time:.2f}s")

        return STTResponse(
            text=result["text"],
            confidence=result.get("confidence", 0.95),
            language=result.get("language", language or "en"),
            processing_time=processing_time,
            model_id=model_id
        )

    except Exception as e:
        logger.error(f"STT file failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/voices", response_model=List[VoiceInfo])
async def get_voices():
    """Get available voices"""
    if not elevenlabs_client:
        raise HTTPException(
            status_code=503, detail="ElevenLabs client not initialized")

    try:
        voices = await elevenlabs_client.get_voices()
        return [
            VoiceInfo(
                voice_id=voice["voice_id"],
                name=voice["name"],
                category=voice.get("category", "unknown"),
                description=voice.get("description"),
                preview_url=voice.get("preview_url")
            )
            for voice in voices
        ]
    except Exception as e:
        logger.error(f"Failed to get voices: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/models")
async def get_models():
    """Get available models"""
    return {
        "tts_models": [
            {
                "id": "eleven_monolingual_v1",
                "name": "Eleven Monolingual v1",
                "description": "High quality English model"
            },
            {
                "id": "eleven_multilingual_v1",
                "name": "Eleven Multilingual v1",
                "description": "Multilingual model supporting various languages"
            },
            {
                "id": "eleven_multilingual_v2",
                "name": "Eleven Multilingual v2",
                "description": "Latest multilingual model with improved quality"
            }
        ],
        "stt_models": [
            {
                "id": "whisper-1",
                "name": "Whisper v1",
                "description": "OpenAI Whisper model for speech recognition"
            }
        ]
    }


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "ElevenLabs Microservice",
        "version": "1.0.0",
        "status": "running",
        "endpoints": {
            "health": "/health",
            "tts": "/tts",
            "stt": "/stt",
            "stt_file": "/stt-file",
            "voices": "/voices",
            "models": "/models"
        }
    }

if __name__ == "__main__":
    port = int(os.getenv("PORT", 8001))
    logger.info(f"Starting ElevenLabs microservice on port {port}")

    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=port,
        log_level="info",
        access_log=True
    )
