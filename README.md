# adapters

[![coverage report](https://github.com/pure-golang/adapters/badges/main/coverage.svg)](https://github.com/pure-golang/adapters/-/commits/main)

[![coverage](https://img.shields.io/badge/coverage-84.9%25-brightgreen)](https://htmlpreview.github.io/?https://github.com/pure-golang/adapters/blob/main/.coverage/.html)

Первый уровень каталога - поставляемая услуга (интерфейс), второй - поставщик-услуги. Например:
- queue/rabbitmq
- storage/pg
- storage/redis
- log/std

L0 - Мониторинг:
- Logger
- Tracing
- Metrics

L1 - Драйвера сервисов:
- Postgres
- RabbitMQ

## Старт

```bash
make
task test
```

### Настройка автодополнения для task

```bash
echo 'eval "$(task --completion zsh)"' >> ~/.zshrc
source ~/.zshrc
```
