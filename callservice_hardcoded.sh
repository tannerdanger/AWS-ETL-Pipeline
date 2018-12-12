#!/usr/bin/env bash
echo "Invoking Lambda Service 1 via API Gateway"
echo ""
echo ""

time output=`curl -X POST \
  https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/transform \
  -H 'Content-Type: application/json' \
  -H 'cache-control: no-cache' \
  -d '{
  "bucketname": "'tcss562.project.tsb'",
  "filename": "'50000SalesRecords.csv'"
}'`

echo ""
echo $output