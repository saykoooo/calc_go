# Распределённый вычислитель арифметических выражений

Учебный проект для курса "Программирование на Go | 24". Спринт 2.

Сделано в go v.1.23.1

## Задание
Пользователь хочет считать арифметические выражения. Он вводит строку 2 + 2 * 2 и хочет получить в ответ 6. Но наши операции сложения и умножения (также деление и вычитания) выполняются "очень-очень" долго. Поэтому вариант, при котором пользователь делает http-запрос и получает в качестве ответа результат, невозможна. Более того: вычисление каждой такой операции в нашей "альтернативной реальности" занимает "гигантские" вычислительные мощности. Соответственно каждое действие мы должны уметь выполнять отдельно и масштабировать эту систему можем добавлением вычислительных мощностей в нашу систему в виде новых "машин". Поэтому пользователь может с какой-то периодичностью уточнять у сервера "не посчиталось ли выражение"? Если выражение наконец будет вычислено - то он получит результат. Помните, что некоторые части арифметического выражения можно вычислять параллельно.

#### Требования:
- Должна быть оформлена документация (данный readme).
- У сервиса должна быть Back-end часть, с 2я частями: оркестратор и агент.
- Оркестратор принимает арифметическое выражение, переводит его в набор последовательных задач и обеспечивает порядок их выполнения.
- Агент получает от оркестратора задачу, выполняет и возвращает оркестратору результат.

#### Оркестратор
Сервер, который имеет следующие endpoint-ы:

- Добавление вычисления арифметического выражения
```bash
curl --location 'localhost/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": <строка с выражение>
}'
```

Коды ответа: 
 - 201 - выражение принято для вычисления, 
 - 422 - невалидные данные, 
 - 500 - что-то пошло не так

Тело ответа
```json
{
    "id": <уникальный идентификатор выражения>
}
```

- Получение списка выражений
```bash
curl --location 'localhost/api/v1/expressions'
```

Тело ответа
```json
{
    "expressions": [
        {
            "id": <идентификатор выражения>,
            "status": <статус вычисления выражения>,
            "result": <результат выражения>
        },
        {
            "id": <идентификатор выражения>,
            "status": <статус вычисления выражения>,
            "result": <результат выражения>
        }
    ]
}
```

Коды ответа:
 - 200 - успешно получен список выражений
 - 500 - что-то пошло не так

- Получение выражения по его идентификатору
```bash
curl --location 'localhost/api/v1/expressions/:id'
```

Коды ответа:
 - 200 - успешно получено выражение
 - 404 - нет такого выражения
 - 500 - что-то пошло не так

Тело ответа
```json
{
    "expression":
        {
            "id": <идентификатор выражения>,
            "status": <статус вычисления выражения>,
            "result": <результат выражения>
        }
}
```

Получение задачи для выполнения
```bash
curl --location 'localhost/internal/task'
```

Коды ответа:
 - 200 - успешно получена задача
 - 404 - нет задачи
 - 500 - что-то пошло не так

Тело ответа
```json
{
    "task":
        {
            "id": <идентификатор задачи>,
            "arg1": <имя первого аргумента>,
            "arg2": <имя второго аргумента>,
            "operation": <операция>,
            "operation_time": <время выполнения операции>
        }
}
``` 

- Прием результата обработки данных.
```bash
curl --location 'localhost/internal/task' \
--header 'Content-Type: application/json' \
--data '{
  "id": 1,
  "result": 2.5
}'
```

Коды ответа:
 - 200 - успешно записан результат
 - 404 - нет такой задачи
 - 422 - невалидные данные
 - 500 - что-то пошло не так

#### Агент
Демон, который получает выражение для вычисления с сервера, вычисляет его и отправляет на сервер результат выражения.

- Должен запускать несколько горутин, каждая из которых выступает в роли независимого вычислителя. Количество горутин регулируется переменной среды `COMPUTING_POWER`
- Агент обязательно общается с оркестратором по http
- Агент все время приходит к оркестратору с запросом "дай задачку поработать" (в ручку `GET internal/task` для получения задач). Оркестратор отдаёт задачу.
- Агент производит вычисление и в ручку оркестратора (`POST internal/task` для приема результатов обработки данных) отдаёт результат

## Настройки
- Порт сервера. Для его настрокйки нужно указать переменную окружения `PORT`. При её отсутствии, сервер запустится на порту по-умолчению - `8080`. 

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
3. Установите переменные среды (Опционально, можно пропустить - запустится с значениями по-умолчанию)
 - Windows CMD, например для установки порта 4200:
```
set PORT=4200
```
 - Linux (или git-bash на Windows), например для порта 4500:
```bash
export PORT=4500
```
4. Запустить оркестратор и агент можно по-отдельности:
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
5. Остановить сервер можно с помощью сочетания клавиш `CTRL+C`

## Тесты
Репозитарий содержит тесты:
 - для агента: `internal/application/agent_test.go`
 - для оркестратора: `internal/application/application_test.go`

Запуск тестов:
```bash
go test -v ./...
```

Помимо вышеуказанных тестов, проверить функциональность сервера можно с помощью утилиты `curl`. 
Обратите внимание, что существуют версии `curl` под Windows, которые не поддерживают 
запросы без шифрования, поэтому рекомендую использовать `curl` из `git-bash`.

- Валидный запрос/ответ:
```bash
curl --location 'localhost:8080/api/v1/calculate' --header 'Content-Type: application/json' --data '{"expression": "7 + (3 * 2)" }'
```
```JSON
{"id":"здесь будет идентификатор (например: 1741030482149934200-6)"}
```
- Невалидный запрос/ответ, статус ответа:
```bash
curl -o - -L -s -w "%{http_code}" --location 'localhost:8080/api/v1/calculate' --header 'Content-Type: application/json' --data '{ "expression": "a + b" }'
```
```
invalid expression
422
```
- Неправильный метод HTTP запроса/ответ, статус ответа:
```bash
curl -o - -L -s -w "%{http_code}" -X GET --location 'localhost:8080/api/v1/calculate' --header 'Content-Type: application/json' --data '{ "expression": "2+2*2" }'
```
```
Method not allowed
405
```
