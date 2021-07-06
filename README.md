# tgnews

Агрегатор новостей для [Telegram Data Clustering Contest](https://contest.com/data-clustering-2). 

Автоматически выделяет новости на английском и русском языках из статей, классифицирует по тематикам и группирует по сюжетам. 
Хранит индекс в который можно добавлять и удалять статьи, а также получать ранжированные сюжеты за определенный период времени.

Для классификации текстов и word-to-vec используется fasttext, для кластеризации - модифицированный dbscan, база данных - BoltDB.
Для ранжирования используется комбинация нескольких метрик - размер кластера из статей, медиана времен публикации статей, а также "важность" новостного сайта (url)

### Build:

Требуется Go >= 1.16
```
make -j
rm *.o
go build -o tgnews
```

### Usage:

Подробная документация есть в [требованиях к конкурсу](https://contest.com/docs/data_clustering2)

Выделение текстов на русском и английском языках:

*\<source_dir\>* - путь до директории с html-файлами статей

Результат возвращается в stdout в JSON
```
./tgnews languages <source_dir>
```

Отделение новостей от других материалов:

```
./tgnews news <source_dir>
```

Группировка по тематике:

```
tgnews categories <source_dir>
```

Группировка похожих новостей в сюжеты:

```
./tgnews threads <source_dir>
```

### Server:

Запуск сервера:

```
./tgnews server <port>
```

Для добавления статьи в индекс нужно выполнить PUT-запрос на /<article.html> с телом статьи, где <article.html> - название файла.
Удаление - аналогичный DELETE http-запрос.

Ранжирование:
```
GET /threads?period=<period>&lang_code=<lang_code>&category=<category>
```
*\<period\>* – период времени в секундах (от 5 минут до 30 дней)\
*\<lang_code\>* – язык статей, en или ru\
*\<category\>* – тематика (society, economy, technology, sports, entertainment, science, other) или any


