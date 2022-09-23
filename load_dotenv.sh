#!/bin/sh

if [ ! -f .env ]; then
    export "cmd .env | .."
fi
