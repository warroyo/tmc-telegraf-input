#! /bin/bash

docker run -it --rm -v $PWD:/telegraf-input -w /telegraf-input golang:1.15.5 /bin/bash