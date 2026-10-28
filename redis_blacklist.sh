#!/bin/bash
# ЛР-4: содержимое Redis — отозванные JWT (blacklist).
# Ключ blacklist:<jti> — короткий ID токена, значение — id пользователя, TTL — секунд до конца срока токена.
R="docker exec redis_store redis-cli -a redispassword --no-auth-warning"
echo "127.0.0.1:6379> KEYS blacklist:*"
$R KEYS 'blacklist:*'
for k in $($R KEYS 'blacklist:*'); do
  echo
  echo "127.0.0.1:6379> GET $k"
  echo "\"$($R GET "$k")\"      # id пользователя, который вышел"
  echo "127.0.0.1:6379> TTL $k"
  echo "(integer) $($R TTL "$k")   # секунд до удаления ключа"
done
