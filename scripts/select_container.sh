#!/bin/bash
CONTAINER="docker"
read -p "Choose 'docker' or 'podman' to use (default: docker): " CONTAINER
case $CONTAINER in
    docker) echo "Using docker" ;;
    podman) echo "Using podman" ;;
    *) echo "Unrecognized selection: $CONTAINER"
        exit 
        ;;
esac