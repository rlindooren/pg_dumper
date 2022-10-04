#!/bin/bash

make
docker build --progress=plain -t pg_dumper:latest .
