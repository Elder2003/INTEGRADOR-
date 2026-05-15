from fastapi import APIRouter, Depends, HTTPException
from google import genai
from pydantic import BaseModel

from app.config import GEMINI_API_KEY, GEMINI_MODEL
from app.dependencies import get_current_user

router = APIRouter(prefix="/chat", tags=["chat"])

client = genai.Client(api_key=GEMINI_API_KEY)


class ChatMessage(BaseModel):
    role: str   # "user" or "assistant"
    content: str


class ChatRequest(BaseModel):
    message: str
    history: list[ChatMessage] = []


class ChatResponse(BaseModel):
    response: str


@router.post("", response_model=ChatResponse)
def send_message(
    body: ChatRequest,
    current_user: dict = Depends(get_current_user),
):
    gemini_history = []
    for msg in body.history:
        role = "model" if msg.role == "assistant" else "user"
        gemini_history.append({"role": role, "parts": [{"text": msg.content}]})

    try:
        result = client.models.generate_content(
            model=GEMINI_MODEL,
            contents=[
                *gemini_history,
                {"role": "user", "parts": [{"text": body.message}]},
            ],
        )
        return ChatResponse(response=result.text)
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"Gemini API error: {str(e)}")
