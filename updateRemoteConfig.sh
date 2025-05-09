########################################
# 🚧 LOCAL environment
########################################

curl --request PUT \
  --data-binary @configfiles/config.local.yaml \
  http://127.0.0.1:8500/v1/kv/config/tx-aggregator/local

curl http://127.0.0.1:8500/v1/kv/config/tx-aggregator/local


########################################
# 🚧 DEV environment
########################################

curl --request PUT \
  --data-binary @configfiles/config.dev.yaml \
  http://10.234.10.222:8501/v1/kv/config/tx-aggregator/dev

curl http://10.234.10.222:8501/v1/kv/config/tx-aggregator/dev


########################################
# 🚧 TEST environment
########################################

curl --request PUT \
  --data-binary @configfiles/config.test.yaml \
  http://aaaa:8500/v1/kv/config/tx-aggregator/test

curl http://aaaa:8500/v1/kv/config/tx-aggregator/test


########################################
# 🚀 PROD environment
########################################

curl --request PUT \
  --data-binary @configfiles/config.prod.yaml \
  http://wallet-consul-internal.tantin.com:8500/v1/kv/config/tx-aggregator/prod

curl http://wallet-consul-internal.tantin.com:8500/v1/kv/config/tx-aggregator/prod
