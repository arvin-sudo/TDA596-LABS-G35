#!/bin/bash

URL="http://localhost:8082/server/files/tmp.txt"

for i in {1..20}
do
    echo "Request $i"
    curl -i -s "$URL" &
    sleep 0.3
    echo ""
done

wait
echo "All requests finished."