package main

// bolt db wallet dump tool for goele

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/bisoncraft/go-electrum-client/wallet/bdb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println()
		fmt.Println("usage: bd <folder/directory where wallet.bdb is stored>\n\n" +
			"Example: ./bd /home/dev/.config/goele/btc/regtest")
		fmt.Println()
		os.Exit(1)
	}
	dbDirectoryPath := os.Args[1]
	if dbDirectoryPath == "wallet.bdb" {
		dbDirectoryPath, _ = os.Getwd()
	}
	var pw = ""
	if len(os.Args) > 2 {
		pw = os.Args[2]
	}

	// Open the db data file in dbDirectoryPath as read only. This can allow another db
	// instance to be live.
	// It will be created if it doesn't exist which is useless for our purpose.
	info, err := os.Stat(dbDirectoryPath)
	if err != nil {
		fmt.Printf("%s : %v\n - exiting", dbDirectoryPath, err)
		os.Exit(1)
	}
	if sz := info.Size(); sz <= 0 {
		fmt.Printf("%s size %d\n - exiting", dbDirectoryPath, sz)
		os.Exit(1)
	}

	// Awkwardness above is because we want to leverage the main code path into
	// database creation. Which requires a directory and attaches `wallet.bdb`
	// to that
	data, err := bdb.Create(dbDirectoryPath, true)
	if err != nil {
		fmt.Printf("%s Open: %v\n - exiting", dbDirectoryPath, err)
		os.Exit(1)
	}
	defer data.Close()

	// timestamp
	t, err := data.Cfg().GetCreationDate()
	if err == nil {
		fmt.Println("Creation Date:", t)
		fmt.Println("")
	} else {
		fmt.Printf("%s : %v\n", dbDirectoryPath, err)
	}

	// encrypted store
	if pw != "" {
		b, err := data.Enc().GetDecrypted(pw)
		if err == nil {
			fmt.Printf("Encrypted store:\n%x\n\n", b)
		} else {
			fmt.Printf("%s : %v\n\n", dbDirectoryPath, err)
		}
	}

	// subscriptions
	subs, err := data.Subscriptions().GetAll()
	if err == nil {
		fmt.Println("Subscriptions:")
		for _, sub := range subs {
			fmt.Printf(" PkScript:           %s\n", sub.PkScript)
			fmt.Printf(" ElectrumScripthash: %s\n", sub.ElectrumScripthash)
			fmt.Printf(" Address:            %s\n\n", sub.Address)
		}
		fmt.Println()
	} else {
		fmt.Printf("%s : %v\n", dbDirectoryPath, err)
	}

	// keys
	keys := data.Keys().GetDbg()
	fmt.Println("Keys:")
	fmt.Println(keys)

	// utxos
	utxos, err := data.Utxos().GetAll()
	if err == nil {
		fmt.Println("Utxos:")
		for _, utxo := range utxos {
			fmt.Printf(" Op:            %s\n", utxo.Op.String())
			fmt.Printf(" At height:     %d\n", utxo.AtHeight)
			fmt.Printf(" Value:         %d\n", utxo.Value)
			fmt.Printf(" Script Pubkey: %s\n", hex.EncodeToString(utxo.ScriptPubkey))
			fmt.Printf(" Watch only:    %v\n", utxo.WatchOnly)
			fmt.Printf(" Frozen:        %v\n\n", utxo.WatchOnly)
		}
		fmt.Println()
	} else {
		fmt.Printf("%s : %v\n", dbDirectoryPath, err)
	}

	// stxos
	stxos, err := data.Stxos().GetAll()
	if err == nil {
		fmt.Println("Stxos:")
		for _, stxo := range stxos {
			fmt.Printf(" Utxo Op:            %s\n", stxo.Utxo.Op.String())
			fmt.Printf(" Utxo At height:     %d\n", stxo.Utxo.AtHeight)
			fmt.Printf(" Utxo Value:         %d\n", stxo.Utxo.Value)
			fmt.Printf(" Utxo Script Pubkey: %s\n", hex.EncodeToString(stxo.Utxo.ScriptPubkey))
			fmt.Printf(" Utxo Watch only:    %v\n", stxo.Utxo.WatchOnly)
			fmt.Printf(" Utxo Frozen:        %v\n", stxo.Utxo.WatchOnly)
			fmt.Printf(" Spend height:       %d\n", stxo.SpendHeight)
			fmt.Printf(" Spend txid:         %v\n\n", stxo.SpendTxid.String())
		}
		fmt.Println()
	} else {
		fmt.Printf("%s : %v\n", dbDirectoryPath, err)
	}

	// txns
	txns, err := data.Txns().GetAll(true)
	if err == nil {
		fmt.Println("Txns:")
		for _, txn := range txns {
			fmt.Printf(" Txid:          %s\n", txn.Txid)
			fmt.Printf(" Value:         %d\n", txn.Value)
			fmt.Printf(" Height:        %d\n", txn.Height)
			fmt.Printf(" Timestamp:     %v\n", txn.Timestamp)
			fmt.Printf(" Watch only:    %v\n", txn.WatchOnly)
			fmt.Printf(" Bytes:         %v\n\n", hex.EncodeToString(txn.Bytes))
		}
		fmt.Println()
	} else {
		fmt.Printf("%s : %v\n", dbDirectoryPath, err)
	}
}
