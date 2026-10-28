#!/bin/sh
# Loads lib_poems.csv.gz (from build.py) into lib_poems, replacing what is there; starred marks survive
# by title + author + first line.
#   load.sh <lib_poems.csv.gz> <psql command…>
# e.g. on the server:
#   sh load.sh lib_poems.csv.gz sudo docker exec -i sub2api-postgres psql -U sub2api -d sub2api
set -e
file="$1"; shift
{
  cat <<'SQL'
BEGIN;
CREATE TEMP TABLE lib_poems_starred AS SELECT title, author, split_part(content, E'\n', 1) AS first FROM lib_poems WHERE starred;
TRUNCATE lib_poems RESTART IDENTITY;
\copy lib_poems (collection, title, author, dynasty, content, notes, tags, famous, popularity) FROM STDIN WITH (FORMAT csv, HEADER true, FORCE_NOT_NULL (notes))
SQL
  gzip -dc "$file"
  cat <<'SQL'
\.
UPDATE lib_poems p SET starred = TRUE FROM lib_poems_starred s
 WHERE p.title = s.title AND p.author = s.author AND split_part(p.content, E'\n', 1) = s.first;
COMMIT;
ANALYZE lib_poems;
SELECT collection, COUNT(*) FROM lib_poems GROUP BY collection ORDER BY 2 DESC;
SQL
} | "$@" -v ON_ERROR_STOP=1
