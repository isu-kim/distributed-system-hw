#!/bin/bash

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 start <no_replica> <sync_mode> | destroy | client"
    echo "Ex) start 3 remote-write"
    exit 1
fi

# For Seph cluster start
if [ "$1" == "start" ]; then
    mkdir ./compose/config -p
    replicas="$2"
    sync_mode="$3"

    if [ "$sync_mode" == "local-write" ]; then
      echo "Sync mode is local-write"
    elif [ "$sync_mode" == "remote-write" ]; then
      echo "Sync mode is remote-write"
    else
      echo "Unknown sync mode: $sync_mode, available: local-write, remove-write"
      exit
    fi

    echo "version: '3'" > docker-compose.yml
    echo "services:" >> docker-compose.yml

    # Create config.json
    echo "{" > ./compose/config/config.json
    echo '  "servicePort": 5000,' >> ./compose/config/config.json
    echo "  \"sync\": \"$sync_mode\"," >> ./compose/config/config.json
    echo '  "replicas": [' >> ./compose/config/config.json

    for ((i=1; i<=replicas; i++)); do
        echo -n "    \"replica-$i:5000\"" >> ./compose/config/config.json
        if [ "$i" -lt "$replicas" ]; then
            echo "," >> ./compose/config/config.json
        else
            echo "" >> ./compose/config/config.json
        fi
        cat >> docker-compose.yml <<EOL
  replica-$i:
    image: isukim/seph:latest
    environment:
      - SEPH_DATA=/go/app/data
      - REPLICA_ID=replica-$i
$(if [ "$i" -eq "1" ]; then echo "      - IS_REPLICA_0=\"TRUE\""; fi)
    container_name: replica-$i
    volumes:
      - ./compose/config:/go/app/config/
      - ./compose/data${i}:/go/app/data/
    command: ./seph ./config/config.json
EOL
        rm -rf ./compose/data${i} # clean up existing replica
        mkdir -p ./compose/data${i} # create replica data directory
    done

    # Complete config.json
    echo '  ]' >> ./compose/config/config.json
    echo "}" >> ./compose/config/config.json

    cat >> docker-compose.yml <<EOL
  seph-client:
    image: isukim/seph-client:latest
    container_name: seph-client
    command: tail -f /dev/null  # Keep the container running in the background
    volumes:
      - ./compose/config:/go/app/config/
networks:
  seph:
    external:
      name: bridge
EOL

    docker-compose up -d --remove-orphans
    echo "Seph cluster created with $replicas replicas."
elif [ "$1" == "destroy" ]; then
    docker-compose kill
    rm -rf ./compose/data*
    echo "Seph cluster destroyed."
elif [ "$1" == "client" ]; then
    echo "Starting Seph client."
    docker exec -it seph-client ./seph-client ./config/config.json
    echo "Unknown command. Usage: $0 start <no_replica> | destroy"
    exit 1
fi
