from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.database import create_db_and_tables
from app.seed import seed_users
from app.routers.auth import router as auth_router
from app.routers.chat import router as chat_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    create_db_and_tables()
    seed_users()
    yield


app = FastAPI(lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(auth_router)
app.include_router(chat_router)


@app.get("/")
def read_root():
    return {"Hello": "World"}
