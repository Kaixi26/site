#/bin/sh
set -xe

# $1 is the name of the image being run
docker run -it \
    -p 443:8080 \
    --mount type=bind,source=/etc/letsencrypt,target=/etc/letsencrypt \
    $1 site.sh -tls
