import bcrypt
from sqlmodel import Session, select

from app.database import engine
from app.models import User


def seed_users() -> None:
    with Session(engine) as session:
        existing = session.exec(select(User).where(User.username == "elder")).first()
        if not existing:
            hashed = bcrypt.hashpw(b"1234", bcrypt.gensalt()).decode()
            user = User(username="elder", hashed_password=hashed)
            session.add(user)
            session.commit()
