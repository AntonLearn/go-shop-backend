CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    price INTEGER NOT NULL,            -- Цена в копейках (100 руб 50 коп = 10050)
    stock INTEGER NOT NULL DEFAULT 0,  -- Доступное количество на складе
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Индекс для быстрого поиска по имени (пригодится для будущего поиска или сортировки)
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);