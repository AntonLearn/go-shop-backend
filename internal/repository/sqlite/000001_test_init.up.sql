-- Архитектурный файл инициализации схемы данных для тестирования.
-- Создает минимально необходимую структуру таблиц в изолированной СУБД SQLite in-memory.

-- Таблица пользователей. Используется для обеспечения целостности внешних ключей (FOREIGN KEY) корзины.
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL
);

-- Таблица товаров. Необходима для связи JOIN, чтобы подтягивать названия и цены в чеки корзины.
CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    price REAL NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0
);

-- Таблица элементов корзины. 
-- Уникальный составной индекс UNIQUE(user_id, product_id) обеспечивает работоспособность механизма Smart Upsert.
CREATE TABLE IF NOT EXISTS cart_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    UNIQUE(user_id, product_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(product_id) REFERENCES products(id) ON DELETE CASCADE
);