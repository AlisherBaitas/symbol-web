# Symbol Web · Symbol Studio

<!-- [Fix] README переписан: прежняя версия описывала другой проект (cmd/web, studio.js,
     Makefile, scripts/, .env.studio.example, кэш, LLM_API_KEY/ADDR/APP_ROOT) — всего этого
     в репозитории нет. Теперь каждый пункт соответствует коду. -->

## Description

Веб-версия проекта **symbol-art**: сервер на Go превращает текст в ASCII-арт в одном из трёх стилей (**standard**, **shadow**, **thinkertoy**) и показывает результат на главной странице.

**Возможности**

- Генерация ASCII-арта из печатных ASCII-символов (коды 32–126), многострочный ввод (Enter в поле или `\n`), до 1000 символов.
- **AI-подсказки** (`/api/suggest`) — 3–5 продолжений текста от LLM, появляются через 300 мс после остановки набора.
- **Подбор баннера** (`/api/recommend-banner`) — детерминированные правила, без LLM и без сети.
- **Творческие варианты** (`/api/variations`, бонус) — 3–5 вариантов текста с подходящим баннером от LLM.
- **Mock-режим**: без настроенной LLM оба LLM-эндпоинта возвращают предсказуемые ответы — проект работает без ключей и без интернета.
- Копирование результата и история последних 8 надписей (в `localStorage` браузера).

**Технологии:** Go (только стандартная библиотека: `net/http`, `html/template`, `encoding/json`), HTML, CSS, JavaScript (`fetch`), без внешних зависимостей и сборки фронтенда.

## Authors

| Участник | Роль |
| --- | --- |
| **asamat** | Серверная структура, форма и шаблоны, генерация ASCII, `AIClient`, стили |
| **abaitas** | Участник команды |
| **ysoltamb** | HTTP-обработчики AI, подсказки и варианты, фронтенд-скрипты, тесты |

## Usage: How to Run

### Prerequisites

- **Go 1.27.1+** (версия из `go.mod`).
- API-ключ **не нужен**. Для живого режима — локальный [Ollama](https://ollama.com) или любой OpenAI-совместимый сервер.

### Installation

```sh
git clone https://01.tomorrow-school.ai/git/asamat/symbol-web.git
cd symbol-web
```

### Environment setup

| Переменная | Если не задана | Назначение |
| --- | --- | --- |
| `LLM_BASE_URL` | **mock-режим** | Базовый адрес OpenAI-совместимого API, например `http://127.0.0.1:11434` для Ollama |
| `LLM_MODEL` | — | Имя модели, например `llama3:latest`. Обязательно в живом режиме |

Пример — в `.env.example`. Go не читает `.env` сам, переменные экспортируются в оболочке.

### Running the server

Сервер нужно запускать **из корня проекта** (он читает `templates/`, `static/` и файлы баннеров по относительному пути).

```sh
# Mock-режим (без LLM)
unset LLM_BASE_URL LLM_MODEL
go run ./cmd/server
# Server running on http://localhost:8080
```

```sh
# Живой режим с Ollama
ollama pull llama3
export LLM_BASE_URL="http://127.0.0.1:11434"
export LLM_MODEL="llama3:latest"
go run ./cmd/server
```

### Accessing the web interface

Откройте <http://localhost:8080>, введите текст, выберите стиль и нажмите **«Создать ASCII-арт»**. Остановка сервера — `Ctrl+C`.

### Tests

```sh
go test -race ./...
go vet ./...
```

Тесты не требуют сети и LLM: живой режим проверяется локальными `httptest`-серверами (включая таймаут 10 с).

## Implementation Details

### Структура

```
cmd/server/main.go            запуск сервера на :8080
internal/handlers/routes.go   маршруты (NewRouter)
internal/handlers/ascii.go    GET /, POST /symbol-art, страницы ошибок
internal/handlers/ai.go       /api/suggest, /api/recommend-banner, /api/variations
internal/ai/                  клиент LLM, mock-ответы, правила баннера, варианты
internal/ascii/generator.go   генерация ASCII-арта
templates/                    index.html, result.html (блок результата), error.html
static/                       style.css и скрипты suggestions/recommendations/variations/history
```

### ASCII art generation algorithm

1. Ввод проверяется: не пустой, не длиннее 1000 символов, только ASCII 32–126 и перенос строки. `\r\n` из формы браузера приводится к `\n`, набранное `\n` тоже считается переносом.
2. Имя баннера сверяется с белым списком (`standard`, `shadow`, `thinkertoy`, с `.txt` или без). Любое другое значение — 400.
3. Каждый символ занимает в файле баннера 9 строк (пустая + 8 строк рисунка); начало символа: `(код − 31) × 9 − 7`. Читаются только нужные символы.
4. Каждая строка текста выводится как 8 строк: на каждой строке склеиваются соответствующие строки всех символов. Пустая строка ввода даёт пустую строку вывода.
5. Результат собирается в локальном `strings.Builder` каждого запроса — параллельные запросы не влияют друг на друга.

### AI integration approach

- `ai.NewAIClient()` читает `LLM_BASE_URL` и `LLM_MODEL` из окружения при каждом запросе. Пустой `LLM_BASE_URL` → mock-режим, сеть не используется.
- Живой режим: `POST {LLM_BASE_URL}/v1/chat/completions` (если адрес уже оканчивается на `/v1`, добавляется только `/chat/completions`), формат OpenAI, `stream: false`.
- Таймаут HTTP-клиента — **10 секунд**; запрос также отменяется, если браузер закрыл соединение.
- Ответ LLM проверяется перед отправкой: размер до 64 KiB, 3–5 строк, каждая короче 50 символов, только печатный ASCII, без дубликатов; варианты — JSON-массив с допустимыми баннерами.
- Ошибки: LLM недоступна или таймаут → **503**; некорректный ответ → **500**; подробности пишутся в лог сервера, пользователь видит понятное сообщение. Сбой LLM не роняет сервер.

### API endpoints

| Метод и путь | Запрос | Ответ | Ошибки |
| --- | --- | --- | --- |
| `GET /` | — | HTML главной страницы | 404 — шаблон не найден, 500 — ошибка шаблона |
| `POST /symbol-art` | форма `inputWord`, `banners` | HTML с результатом | 400 — пустой/длинный текст, недопустимые символы, неверный баннер; 404 — файл баннера не найден |
| `GET /symbol-art` | — | 303 → `/` | — |
| `POST /api/suggest` | `{"text": "..."}` | `{"suggestions": [...]}` | 400, 415, 503, 500 — `{"error": "..."}` |
| `POST /api/recommend-banner` | `{"text": "..."}` | `{"recommended", "reasoning", "alternatives"}` | 400, 415 — `{"error": "..."}` |
| `POST /api/variations` | `{"text": "..."}` | `{"variations": [...]}` | 400, 415, 503, 500 — `{"error": "..."}` |
| любой другой путь | — | 404 (HTML; для `/api/...` — JSON) | GET на API-путь — 405 |

Примеры:

```sh
curl -X POST localhost:8080/symbol-art --data-urlencode 'inputWord=Hello' --data 'banners=shadow.txt'

curl -X POST localhost:8080/api/suggest -H 'Content-Type: application/json' -d '{"text":"Happy Birth"}'
# {"suggestions":["Happy Birth!","Happy Birth World","~ Happy Birth ~","HAPPY BIRTH"]}

curl -X POST localhost:8080/api/recommend-banner -H 'Content-Type: application/json' -d '{"text":"WELCOME"}'
# {"recommended":"shadow","reasoning":"Short or uppercase text is bold and stands out.","alternatives":[...]}

curl -X POST localhost:8080/api/variations -H 'Content-Type: application/json' -d '{"text":"hello"}'
# {"variations":[{"text":"Hello","description":"Professional style","suggested_banner":"standard"}, ...]}
```

### Frontend architecture

Обычная HTML-форма отправляет `POST /symbol-art` (работает и без JavaScript). Скрипты в `static/js/` дополняют её через `fetch`, без перезагрузки страницы:

- `suggestions.js` — debounce 300 мс, запрос только от 3 символов, выпадающий список, клик подставляет текст, отмена устаревших запросов, таймаут 12 с.
- `recommendations.js` — кнопка «Подобрать стиль»: состояние загрузки, ответ с объяснением и альтернативами, подсветка рекомендованного баннера, кнопка применения.
- `variations.js` — кнопка «Идеи для текста»: панель вариантов, клик подставляет текст и баннер.
- `history.js` — копирование результата, пример «Hello, world!», история 8 последних надписей в `localStorage`.

Все ответы сервера вставляются через `textContent`, а не `innerHTML`.

## AI Features

### Text suggestions

`/api/suggest` отправляет LLM промпт из задания и возвращает 3–5 продолжений. В mock-режиме ответ детерминирован: для `Hello` — `Hello!`, `Hello World`, `~ Hello ~`, `HELLO`. Текст короче 3 символов → 400 (фронтенд такой запрос не отправляет).

### Banner recommendation (rule-based, без LLM)

Правила проверяются по порядку, внешние пробелы не учитываются:

1. Есть перенос строки → **standard** (многострочный текст читается лучше).
2. Доля символов-знаков среди непробельных ≥ 30% → **thinkertoy** (игривый стиль для символов).
3. Все буквы заглавные **или** длина ≤ 6 → **shadow** (жирный, заметный).
4. Длина > 12 → **standard** (читаемость длинного текста).
5. Иначе → **thinkertoy** (короткий текст в смешанном регистре).

Вместе с рекомендацией возвращаются две альтернативы с фиксированными оценками `score` и причинами. Эндпоинт не обращается к сети и работает всегда.

### Creative variations (бонус)

`/api/variations` просит LLM 4 варианта (деловой, выразительный, дружелюбный, декоративный) в виде JSON-массива и проверяет каждое поле. Mock-режим возвращает 4 варианта на основе введённого текста.

### API setup

1. Установите Ollama и скачайте модель: `ollama pull llama3`.
2. Убедитесь, что сервер Ollama работает: `curl http://127.0.0.1:11434/v1/models`.
3. Экспортируйте `LLM_BASE_URL` и `LLM_MODEL` (см. выше) и перезапустите `go run ./cmd/server`.
4. Если LLM недоступна, подсказки и варианты отвечают 503 с понятным сообщением; генерация и подбор баннера продолжают работать. Чтобы вернуться в mock-режим, выполните `unset LLM_BASE_URL` и перезапустите сервер.
