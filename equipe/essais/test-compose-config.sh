#!/bin/bash
# Script de test de la configuration Docker Compose
# Test de la propagation de REMOTE_OLLAMA_URL

set -e

echo "=== Test Docker Compose Config ==="
echo

cd /home/localadm/projets/Themis-gestion-finance

echo "1. Test syntaxe docker-compose.yml principal"
docker compose config > /dev/null
echo "✅ Syntaxe valide"
echo

echo "2. Test syntaxe compose-supabase/docker-compose.yml"
cd compose-supabase
docker compose config > /dev/null 2>&1
echo "✅ Syntaxe valide"
cd ..
echo

echo "3. Test avec REMOTE_OLLAMA_URL définie"
export REMOTE_OLLAMA_URL="http://host.docker.internal:11434"

echo "Service app OLLAMA_URL:"
docker compose config | grep -A 20 "services:" | grep -A 15 "app:" | grep OLLAMA_URL
echo

echo "Service knowledge-graph OLLAMA_URL:"
docker compose config | grep -A 50 "knowledge-graph:" | grep OLLAMA_URL
echo

echo "4. Test sans REMOTE_OLLAMA_URL"
unset REMOTE_OLLAMA_URL
echo "Service app OLLAMA_URL (devrait être vide):"
docker compose config | grep -A 20 "services:" | grep -A 15 "app:" | grep OLLAMA_URL
echo

echo "Test terminé"
