# BTC ElectrumX RegTest Simnet Harness

Goele test harness for ElectrumX running on regtest simnet over a single Firo node.

## Usage

1. Start the __src_harness.sh__ script as `source src_harness.sh` to pull in the PATH to your firo binary installation.

2. If auto mining not required go to tmux window #3 (Ctl-b, 3) and kill the watch miner with SIGINT (Ctl-c)

3. When the node is up start the __ex.sh__  ElectrumX script as `./ex.sh` to start the firo regtest simnet ElectrumX server over the firo node harness above.

Use firo rpc commands with the `alpha` test wallet; example `./alpha getinfo`.

Thanks to the Decred Bison Wallet team for the base scripts.
