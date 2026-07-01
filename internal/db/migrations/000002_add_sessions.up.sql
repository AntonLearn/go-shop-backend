CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    -- Каскадное удаление: если пользователя удалят, его сессии очистятся автоматически
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Индекс для быстрого поиска сессий по токену
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);