package chat

const (
	msgSuccessfullyGetPermission     = `Вам успешно предоставлены права доступа`
	msgSuccessfullyDeleteUser        = `Пользователь успешно удалён`
	msgSuccessfullyAddVKNewsGroup    = `VK группа успешна добавлена`
	msgSuccessfullyDeleteVkNewsGroup = `VK группа успешна удалена`

	msgNewsSourcesNotFound = `Новостные источники не найдены`
	msgUserNotFound        = `Пользователь не найден`
	msgNotEnoughArgs       = `Некорректное кол-во аргументов`
	msgIncorrectArgs       = `Неверные аргументы`
	msgVkGroupNotFound     = `VK группа не найдена`
	msgVkGroupIsPrivate    = `VK группа приватная`
	msgVkGroupIsExists     = `VK группа уже инициализирована`
)

const msgSuccessfullyAddUser = `Передайте пользователю секретный код для получения прав доступа
Code: %s

Команда для получения прав: /perm <code>
Код действителен в течении 3 часов после его создания`

const msgStart = "Привет! \n"

const msgSubUserHelp = `Команды:
/get sources  - Получение всех новостных источников
/get tg       - Получение всех Telegram источников
/get tg ch    - Получение Telegram каналов
/get tg group - Получение Telegram групп
/get vk       - Получение VK групп

/add vk <Domain>    - Добавление VK группы как новостного источника
/delete vk <Domain> - Удаление VK группы

Для добавления Telegram групп и каналов как новостных источников нужно добавить меня в них, и выдать права для доступа к сообщениям
`

const msgAdminHelp = `Команды:
/get sources  - Получение всех новостных источников
/get tg       - Получение всех Telegram источников
/get tg ch    - Получение Telegram каналов
/get tg group - Получение Telegram групп
/get vk       - Получение VK групп

/add user <@Username>    - Добавление Sub User
/delete user <@Username> - Удаление Sub User

/add vk <Domain>    - Добавление VK группы как новостного источника
/delete vk <Domain> - Удаление VK группы

Для добавления Telegram групп и каналов как новостных источников нужно добавить меня в них, и выдать права для доступа к сообщениям
`
