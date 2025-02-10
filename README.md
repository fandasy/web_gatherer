Перед запуском проекта не забудьте добавить токены в confi/local.yaml

### local.yaml
```
telegram
  token: Telegram Bot Token
  api_id:  Telegram Application id
  api_hash: Telegram Application hash

vk_api:
  token: VK Application server token
```

### Запуск

Имейте в виду что для запуска проекта требуется скомпилировать библиотеку Tdlib, это может занять некоторое время!
```
docker-compose up --build
```
