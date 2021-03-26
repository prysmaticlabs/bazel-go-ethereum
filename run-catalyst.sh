echo \{ \
  \"config\": \{ \
    \"chainId\": 220720, \
    \"homesteadBlock\": 0, \
    \"eip150Block\": 0, \
    \"eip155Block\": 0, \
    \"eip158Block\": 0, \
    \"byzantiumBlock\": 0, \
    \"constantinopleBlock\": 0, \
    \"petersburgBlock\": 0, \
    \"istanbulBlock\": 0 \
  \}, \
  \"alloc\": \{\}, \
  \"coinbase\": \"0x0000000000000000000000000000000000000000\", \
  \"difficulty\": \"0x20000\", \
  \"extraData\": \"\", \
  \"gasLimit\": \"0x2fefd8\", \
  \"nonce\": \"0x0000000000220720\", \
  \"mixhash\": \"0x0000000000000000000000000000000000000000000000000000000000000000\", \
  \"parentHash\": \"0x0000000000000000000000000000000000000000000000000000000000000000\", \
  \"timestamp\": \"0x00\" \
\} > /tmp/catalystgenesis.json

bazel run //cmd/geth -- --datadir /tmp/catalystchaindata init /tmp/catalystgenesis.json
bazel run //cmd/geth -- --rpc --rpcapi net,eth,eth2 --nodiscover --etherbase 0x1000000000000000000000000000000000000000 --datadir /tmp/catalystchaindata
