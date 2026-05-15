from typing import Optional
from sqlmodel import Field, SQLModel


class User(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    username: str = Field(unique=True, index=True)
    hashed_password: str


class UserRegister(SQLModel):
    username: str
    password: str


class UserLogin(SQLModel):
    username: str
    password: str


class TokenResponse(SQLModel):
    access_token: str
    token_type: str = "bearer"


class UserPublic(SQLModel):
    id: int
    username: str
