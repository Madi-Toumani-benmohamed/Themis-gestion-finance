#!/bin/bash
set -e
cd /home/localadm/projets/Themis-gestion-finance
echo "Test Ollama Remote"
docker compose config > /dev/null && echo "docker-compose valide"
