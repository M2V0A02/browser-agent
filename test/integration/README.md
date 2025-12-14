# Integration Tests

Интеграционные тесты для browser automation tools.

## Запуск тестов

```bash
# Все интеграционные тесты
cd test/integration
APP_ENV=test go test -v

# Конкретный тест
APP_ENV=test go test -v -run TestSearchByTextContains

# С покрытием
APP_ENV=test go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Что тестируется

### GetPageStructure
- ✅ Находит семантические элементы (header, main, section, aside, footer)
- ✅ Находит элементы с ID
- ✅ Находит заголовки (h1-h6)
- ✅ Все элементы имеют селекторы

### Observe (Structure Mode)
- ✅ Находит featured article секцию
- ✅ Находит products секцию
- ✅ Находит элементы с префиксом классов (mp-*)

### Observe (Interactive Mode)
- ✅ Находит кнопки
- ✅ Находит ссылки
- ✅ Обрабатывает inputs (могут быть не в viewport)

### Search (type="text")
- ✅ Находит элементы с точным совпадением текста
- ✅ Возвращает селектор и parent info
- ✅ Не находит несуществующий текст

### Search (type="contains")
- ✅ Находит элементы с частичным совпадением текста
- ✅ "Featured" находит "Featured Article"
- ✅ Все результаты содержат искомый текст
- ✅ Возвращает поле `match` с искомым текстом

### Search (type="selector")
- ✅ Находит элементы по wildcard классам: `[class*="product-"]`
- ✅ Находит элементы по wildcard ID: `[id*="mp-"]`
- ✅ Находит элементы по точному селектору: `#mp-tfp`
- ✅ Возвращает parent info и attributes

### Search (type="id")
- ✅ Находит элементы по точному ID
- ✅ Находит элементы по частичному ID
- ✅ Backward compatibility с старым форматом

### Search Results
- ✅ Включают информацию о родительском элементе
- ✅ Могут быть сериализованы в JSON
- ✅ Содержат все необходимые поля (selector, element, text, classes, id)

## Тестовые данные

Тесты используют `testdata/test_page.html` - HTML файл с:
- Семантическими элементами (header, nav, main, section, aside, footer)
- Элементами с ID и классами
- Заголовками разных уровней
- Продуктовым каталогом
- Формами и инпутами
- Featured article секцией (имитация Wikipedia)

## Примечания

- Тесты запускаются в headless режиме
- Используется `file://` протокол для загрузки локального HTML
- Каждый тест создает свой собственный экземпляр браузера
- Timeout: 5 минут на все тесты
