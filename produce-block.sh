# This script provides a demonstration of producing a block via the
# eth2_produceBlock. It doesn't necessarily work, but it shows the request
# structure.

ZERO_HASH="\"0x0000000000000000000000000000000000000000000000000000000000000000\""

# Note: you might have to get the root of the genesis block via
# `./rpc_get_genesis_block.sh`. Presently there's a bug in Catalyst where the
# handler will crash if you send the wrong root.
PARENT_HASH='"0xddce9c6ed083fc7daf708d0c4ed4fc73320a4c0b752929d677f70461266acc4d"'
RANDAO_MIX=$ZERO_HASH
# Note: the slot is *not* encoded as hex.
SLOT=1
# Note: the timestamp is *not* encoded as hex.
TIMESTAMP=$(date +%s)
RECENT_BLOCK_ROOTS="[$ZERO_HASH, $ZERO_HASH, $ZERO_HASH, $ZERO_HASH, $ZERO_HASH, $ZERO_HASH, $ZERO_HASH, $ZERO_HASH]"

# Note: the params are expressed as a single JSON object, not spread out across
# the array like typical eth JSON RPC calls.
PARAMS="[{ \"parent_hash\": $PARENT_HASH, \"randao_mix\": $RANDAO_MIX, \"slot\": $SLOT, \"timestamp\": $TIMESTAMP, \"recent_block_roots\": $RECENT_BLOCK_ROOTS}]"

DATA="{\"jsonrpc\":\"2.0\",\"method\":\"eth2_produceBlock\",\"params\":$PARAMS,\"id\":67}"

curl \
	-X \
	POST \
	-H "Content-Type: application/json" \
	--data "$DATA" \
	http://localhost:8545
