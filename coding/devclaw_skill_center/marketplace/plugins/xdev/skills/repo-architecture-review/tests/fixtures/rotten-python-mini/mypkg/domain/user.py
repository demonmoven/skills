from mypkg.infra.db import open_db

def get_user(user_id: int):
    db = open_db()
    return db.query(f"SELECT * FROM users WHERE id = {user_id}")
