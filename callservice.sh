#!/usr/bin/env bash
echo "Invoking Lambda Service 1 via API Gateway"
echo ""
echo ""

time output=`curl -X POST \
  https://z7y3oqb9d5.execute-api.us-east-1.amazonaws.com/dev/ \
  -H 'Content-Type: application/json' \
  -H 'cache-control: no-cache' \
  -d '{
  "bucketname": "'$1'",
  "filename": "'$2'"
}'`

echo ""
echo $output