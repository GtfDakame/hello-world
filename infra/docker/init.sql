# Инициализация базы данных
# Создается при первом запуске контейнера PostgreSQL

-- Расширения для полнотекстового поиска и UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Примечание: Основные таблицы создаются через миграции в Go-коде (internal/database/database.go)
-- Этот скрипт предназначен для начальной настройки расширений и базовых конфигураций

-- Настройка поисковой конфигурации для русского языка (опционально)
-- ALTER TEXT SEARCH CONFIGURATION russian ADD UNACCENT;

COMMENT ON EXTENSION "uuid-ossp" IS 'Генерация UUID версий 1, 3, 4 и 5';
COMMENT ON EXTENSION "pg_trgm" IS 'Модуль для поиска по подстрокам (trigram similarity)';
