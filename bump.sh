#!/bin/bash -e

old=$(cat ./version)
new=$(expr $old + 1)

echo $new > ./version
cat ./version
