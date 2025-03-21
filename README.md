Привет читатель, этой мой проект для внедрения новостной ленты на веб-сайт

Поддерживаемые новостные источники: Telegram group, Telegram channel, VK group

### Запуск

Перед запуском проекта не забудьте добавить токены в config/local.yaml

#### local.yaml
```
telegram
  admin: Username # Ваш telegram username, писать без @
  token: Telegram Bot Token

vk_api:
  token: VK Application server token
  
file_storage:
  addr: "192.168.0.102:9000"  # Замените на свой ip host
```

По желанию можно поменять уровни логирования:
```
local - text, уровень Debug, вывод в консоль
dev   - json, уровень Debug, вывод в консоль
prod  - json, уровень Info,  вывод в консоль
```


Запуск
```
docker-compose up --build
```
После запуска приложения и вывода в логах сообщения: 

"[TELEGRAM] Write to your chat bot on behalf of Admin, which was specified in the config file"

Нужно будет написать любой текст Telegram боту от имени аккаунта который был указан в конфиг файле, для добавления админа в бд приложения

### Функционал

Моё приложение состоит из двух компонентов:
1. Сервер для отправки новостных блоков
2. Telegram-бот для управления новостными источниками

Для проверки функционала по внедрению новостной ленты можно использовать заранее подготовленный front.html, внутри которого нужно будет заменить адрес для подключения

P. s. Я не fronted разработчик, front.html файл писался только ради тестов, оценивать его не нужно!

```azure
// 102 строка
<script>
    const socket = new WebSocket('http://192.168.0.102:8082/ws'); // Замените на ваш адрес http server

```

Telegram-бот команды:
```
/help - Выводит информацию о командах

/get sources  - Получение всех новостных источников
/get tg       - Получение всех Telegram источников
/get tg ch    - Получение Telegram каналов
/get tg group - Получение Telegram групп
/get vk       - Получение VK групп

/add user <@Username>    - Добавление Sub User
/delete user <@Username> - Удаление Sub User
<@Username>  -  Пример: @ExampleUsername  Обязательно использовать префикс @
Sub User     -  Пользователь который сможет управлять работой приложения

/add vk <Domain>    - Добавление VK группы как новостного источника
/delete vk <Domain> - Удаление VK группы
<Domain>  -  На сайте сообщества открываем "Подробная информация" и находим поле со значком "@"

Для добавления Telegram групп и каналов как новостных источников нужно добавить бота в них, и выдать права для доступа к сообщениям (права администратора)
```
