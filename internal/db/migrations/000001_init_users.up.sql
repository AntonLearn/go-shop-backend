CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL DEFAULT '', -- Добавили поле name для синхронизации с моделью Go
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'client', -- Роли: 'client', 'admin'
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (email, name, password_hash, role) 
VALUES ('admin@mail.com', 'Admin', '$2a$10$Dy9pUlzb0vhEk6BQgeDaXe0gyVWZ5lWPT4cU/LZcjai4uqdzRDhey', 'admin') 
ON CONFLICT (email) DO NOTHING;