#!/bin/bash

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 start <no_replica> | destroy | client"
    exit 1
fi

# For Seph cluster start
if [ "$1" == "start" ]; then
    mkdir ./compose/config -p
    replicas="$2"
    echo "version: '3'" > docker-compose.yml
    echo "services:" >> docker-compose.yml

    # Create config.json
    echo "{" > ./compose/config/config.json
    echo '  "servicePort": 5000,' >> ./compose/config/config.json
    echo '  "sync": "local-write",' >> ./compose/config/config.json
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
    image: ubuntu:latest
    container_name: seph-client
    command: tail -f /dev/null  # Keep the container running in the background

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
    docker exec -it seph-client ./seph-client
    echo "Unknown command. Usage: $0 start <no_replica> | destroy"
    exit 1
fi
