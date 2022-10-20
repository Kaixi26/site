#/bin/sh
set -xe

docker run -it \
    -p 443:8080 \
    --mount type=bind,source=/etc/letsencrypt,target=/etc/letsencrypt \
    $1 site.sh -tls