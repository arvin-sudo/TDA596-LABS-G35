#!/bin/bash

URL="http://localhost:8080/files/tmp.txt"
# if you run at root, you may need add /server/ path.
#URL="http://localhost:8080/server/files/tmp.txt"

for i in {1..20}
do
    echo "Request $i"
    curl -i -s "$URL" &
    sleep 0.3
    echo ""
done

wait
echo "All requests finished."