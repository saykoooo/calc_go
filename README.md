# Распределённый вычислитель арифметических выражений. Финал

Учебный проект для курса "Программирование на Go | 24". Спринт 5.

Сделано в go v.1.23.1

## Задание
Расширить ранее реализованный функционал вычислителя для работы в контексте определенного пользователя.

#### Требования:
- Должна быть оформлена документация (данный readme).
- Регистрация и аутентификация пользователя.
- Хранение данных в SQLite.
- Взаимодействие оркестратора и агента по gRPC.
- Наличие тестов.

## Структура проекта
```
/
|cmd/ - обработка комманд запуска
|data/ - каталог базы данных SQLite
|internal/ 
|internal/agent/ - агент вычислителя 
|internal/application/ - оркестратор 
|internal/calc/ - парсер выражений 
|internal/db/ - пакет работы с базой данных 
|proto/ - файлы gRPC
|web/ - фронтенд
```

## Настройки
- Порт HTTP сервера. Для его настрокйки нужно указать переменную окружения `PORT`. При её отсутствии, сервер запустится на порту по-умолчению - `8080`. 

- Порт gRPC сервера. Для его настрокйки нужно указать переменную окружения `GRPC_PORT`. При её отсутствии, сервер запустится на порту по-умолчению - `5000`. 

- Ключевая фраза для генерации JWT токенов `JWT_SECRET`. При её отсутствии, только для демонстрационных целей, будет использована фраза по-умолчению.

- Время выполнения операций задается переменными, указанными ниже, в миллисекундах. При отсутствии, устанавливается значение - `1000`.
  - `TIME_ADDITION_MS` - время выполнения операции сложения в миллисекундах
  - `TIME_SUBTRACTION_MS` - время выполнения операции вычитания в миллисекундах
  - `TIME_MULTIPLICATIONS_MS` - время выполнения операции умножения в миллисекундах
  - `TIME_DIVISIONS_MS` - время выполнения операции деления в миллисекундах

- Количество горутин агента регулируется переменной среды `COMPUTING_POWER`. При отсутствии, задается значение - `1`.

## Запуск сервера
1. Клонируйте на свой компьютер данный репозитарий командой:
```bash
git clone https://github.com/saykoooo/calc_go.git
```
2. Перейдите в каталог `calc_go`
```bash
cd calc_go
```
3. Установите зависимости
```bash
go mod tidy
```
4. Установите переменные среды (Опционально, можно пропустить - запустится с значениями по-умолчанию)
 - Windows CMD, например для установки порта 4200:
```
set PORT=4200
```
 - Linux (или git-bash на Windows), например для порта 4500:
```bash
export PORT=4500
```
5. Запустить оркестратор и агент можно по-отдельности:
- только оркестратор:
```bash
go run ./cmd
```
- только агент:
```bash
go run ./cmd --agent
```
Так и сразу вместе - оркестратор+агент:
```bash
go run ./cmd --all
```
6. Остановить сервер можно с помощью сочетания клавиш `CTRL+C`

## Тесты
Репозитарий содержит тесты:
 - для агента:
   - `internal/agent/agent_test.go`
   - `internal/agent/integration_test.go`
 - для оркестратора:
   - `internal/application/config_test.go`
   - `internal/application/handlers_test.go`
 - для парсера выражений:
   - `internal/calc/evaluate_test.go`
   - `internal/calc/parser_test.go`
 - для базы данных:
   - `internal/db/db_test.go`
 
Запуск тестов:
```bash
go test -v ./...
```

Помимо вышеуказанных тестов, проверить функциональность сервера можно с помощью утилиты `curl`. 
Обратите внимание, что существуют версии `curl` под Windows, которые не поддерживают 
запросы без шифрования, поэтому рекомендую использовать `curl` из `git-bash`.

- Валидный запрос/ответ:
```bash
curl -o - -L -s -w "%{http_code}" --location 'localhost:8080/api/v1/register' --header 'Content-Type: application/json' --data '{"login": "username","password": "passwd"}'
```
Статус ответа при первом выполнении запроса:
```
200
```
Ответ и статус при повторном выполнении запроса:
```
User already exists
400
```
- Валидный запрос/ответ (при наличии в БД пользователя из вышестоящего запроса):
```bash
curl -o - -L -s -w "%{http_code}" --location 'localhost:8080/api/v1/login' --header 'Content-Type: application/json' --data '{"login": "username","password": "passwd"}'
```
```JSON
{"expires_in":"300","token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDY5NTgyNzQsImlhdCI6MTc0Njk1Nzk3NCwibmFtZSI6InVzZX
JuYW1lIiwibmJmIjoxNzQ2OTU3OTc0fQ.zojWgDzX5J6CDvmzeWcuraMV6wrADx7JggNhu2xc2l8"}
```
- Невалидный запрос/ответ, статус ответа:
```bash
curl -o - -L -s -w "%{http_code}" --location 'localhost:8080/api/v1/login' --header 'Content-Type: application/json' --data '{"login": "username","password": "xxx"}'
```
```
Auth failed
401
```
- Валидный запрос/ответ (при наличии в БД пользователя из вышестоящего запроса и подстановки вместо <ТОКЕН> - токена из запроса /api/v1/login):
```bash
curl -o - -L -s -w "%{http_code}" -X POST --location 'localhost:8080/api/v1/calculate' -H 'Content-Type: application/json' -H "Authorization: Bearer <ТОКЕН>" --data '{ "expression": "2+2*2" }'
```
```
{"id":"1746959115167947300-6"}
201
```
- Невалидный запрос/ответ:
```bash
curl -o - -L -s -w "%{http_code}" -X POST --location 'localhost:8080/api/v1/calculate' -H 'Content-Type: application/json' -H "Authorization: Bearer 1" --data '{ "expression": "2+2*2" }'
```
```
Unauthorized
401
```
- Неправильный метод HTTP запроса/ответ, статус ответа (при наличии в БД пользователя из вышестоящего запроса и испльзовании валидного токена из запроса /api/v1/login):
```bash
curl -o - -L -s -w "%{http_code}" -X GET --location 'localhost:8080/api/v1/calculate' -H 'Content-Type: application/json' -H "Authorization: Bearer <ТОКЕН>" --data '{ "expression": "2+2*2" }'
```
```
Method not allowed
405
```

## Веб-интерфейс

Веб-интерфейс вычислителя арифметических выражений открывается, при запущенном сервисе оркестратора, по адресу:
```
http://localhost:<порт>/web/
```
где <порт> - это тот же порт который используется для запуска оркестратора и указывается в переменной окружения `PORT`. По умолчанию - `8080`
Таким образом, стандартный адрес будет: 
```
http://localhost:8080/web/
```

После регистрации пользователя, можно пройти аутентификацию, используя указанные данные.

После входа - вводим выражение в форму, жмём "Вычислить". Статус выражения показывается ниже.
Обновление таблицы результатов вычислений происходит каждые 2 секунды.

Завершить сеанс можно использую кнопку "Выйти".
