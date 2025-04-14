#!/bin/bash

# run as: source ./src_harness.sh

# all testing is against my bitcoin binary install .. your PATH may be different
export PATH=$PATH:$HOME/bitcoin-22.1/bin
which bitcoind
$(pwd)/harness.sh


