{
    "slug": "crazy-postgres-queries",
    "date": "2014-12-16T09:14:00.000Z",
    "tags": [],
    "title": "Crazy Postgres Queries",
    "publishdate": "2014-12-16T09:14:00.000Z"
}


I’ve been working on implementing search for documents. I’m not sure if
I’m every going to implement search for body content, but I thought I
should probably implement it for titles & users.

It turns out that PostgreSQL has pretty nice full text search support
with lexemes. I’ve been following this article pretty closely:

<http://blog.lostpropertyhq.com/postgres-full-text-search-is-good-enough/>

The only issue I’ve encountered is that it doesn’t do direct text
matching. For example if you have a title ‘Bananas are tasty!’ and you
search for 'ban’ it won’t match. To work around this I combined full
text search with an non case sensitive pattern match. My query is
getting kind of long, but it seems to work well.

Here’s the full thing:

```sql
SELECT id, name, permissions.user_email
FROM (SELECT
  ws_files.id as id,
  ws_files.name as name,
  ws_files.name || ' ' ||
  coalesce((string_agg(p1.user_email, ' ')), '') || ' ' ||
  regexp_replace(coalesce((string_agg(p1.user_email, ' ')), ''), '[@.+]', ' ', 'g') as text,
  to_tsvector(ws_files.name) ||
  to_tsvector(coalesce((string_agg(p1.user_email, ' ')), '')) ||
  to_tsvector(regexp_replace(coalesce((string_agg(p1.user_email, ' ')), ''), '[@.+]', ' ', 'g'))
  as document
  FROM ws_files
  JOIN permissions p1
  ON p1.file_id = ws_files.id
  JOIN permissions p2
  ON p2.file_id = ws_files.id
  WHERE p2.user_email='rice@outerearth.net'
  GROUP BY ws_files.id) f_search
JOIN permissions
ON permissions.file_id = id
WHERE permissions.level = 'owner'
AND ((f_search.document @@ to_tsquery('fn.lc')) OR
  f_search.text ILIKE ('%' || 'fn.lc' || '%'));
```

This matches all documents that 'rice@outerearth.net’ can access and
have the phrase 'fn.lc’ in their title or emails.

```
  id  |       name       |     user_email
------+------------------+---------------------
 1936 | Unnamed Document | rice@fn.lc
 ```

