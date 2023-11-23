#!/bin/bash
echo
echo "add '-v' on command line for a detailed output"
echo
echo "add '-tags=live' on command line for long running/network tests"
echo
echo "add '-tags=harness' on command line for tests that require regtest blockchain + electrumX harness(es)"
echo
go mod tidy
go test $@ -count=1 ./...
