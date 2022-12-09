#/bin/sh

find . -type f | grep -E "((go)|(md)|(html))$" | entr -rc go run .