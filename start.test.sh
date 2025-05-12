docker compose --env-file .env.test.app1 -f docker-compose.yml -p app1 up -d --build
docker compose --env-file .env.test.app2 -f docker-compose.yml -p app2 up -d --build
