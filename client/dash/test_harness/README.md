# BTC ElectrumX RegTest Simnet Harness

Goele test harness for ElectrumX running on regtest simnet over a single Bitcoin node.

## Usage

1. Start the __src_harness.sh__ script as `source src_harness.sh` to pullin the PATH to your bitcoin binary installation.

2. If auto mining not required go to tmux window #3 (Ctl-b, 3) and kill the watch miner with SIGINT (Ctl-c)

3. When the node is up start the __ex.sh__  ElectrumX script as `./ex.sh` to start the bitcoin regtest simnet ElectrumX server over the bitcoin node harness above.

Use bitcoin rpc commands with the `alpha` test wallet; example `./alpha -getinfo`.

Thanks to the Decred DCRDEX team for the base scripts.
