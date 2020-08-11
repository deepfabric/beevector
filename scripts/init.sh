#!/bin/bash

defaultDim="512"
defaultDataPath="/data/beevector"
defaultMax="2000000"

while [[ $# > 0 ]]
do
        case "$1" in

                -d|--dim)
                        defaultDim="$2"
                        shift
                        ;;

                -p|--data)
                        defaultDataPath="$2"
                        shift
                        ;;

                -m|--max)
                        defaultMax="$2"
                        shift
                        ;;

                --help|*)
                        echo "Usage:"
                        echo "    --dim \"Vector dimension, default 512.\""
                        echo "    --data \"Beevector data path, default /data/beevector\""
                        echo "    --max \"Beevector max records per shard, default 2000000\""
                        echo "    --help"
                        exit 1
                        ;;
        esac
        shift
done

echo "init with Vector dimension:                $defaultDim"
echo "init with Beevector data path:             $defaultDataPath"
echo "init with Beevector max records per shard: $defaultMax"

rm -rf $defaultDataPath
echo "Beevector data path clear completed"

mkdir -p $defaultDataPath
echo "Beevector data path created"

rm -rf ./docker-compose.yml
cp ./docker-compose.yml.base ./docker-compose.yml
sed -i "s|{data}|$defaultDataPath|g" docker-compose.yml
sed -i "s|{dim}|$defaultDim|g" docker-compose.yml
sed -i "s|{max}|$defaultMax|g" docker-compose.yml