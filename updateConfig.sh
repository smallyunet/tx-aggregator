

# dev
curl --request PUT \
  --data-binary @configfiles/config.dev.yaml \
  http://127.0.0.1:8500/v1/kv/config/tx-aggregator/dev

curl http://127.0.0.1:8500/v1/kv/config/tx-aggregator/dev


# test
curl --request PUT \
  --data-binary @configfiles/config.test.yaml \
  http://10.234.10.222:8501/v1/kv/config/tx-aggregator/test

curl http://10.234.10.222:8501/v1/kv/config/tx-aggregator/test


