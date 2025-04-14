package wltdash

import (
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/wire"
)

// var BumpFeeAlreadyConfirmedError = errors.New("transaction is confirmed, cannot bump fee")
// var BumpFeeTransactionDeadError = errors.New("cannot bump fee of dead transaction")
// var BumpFeeNotFoundError = errors.New("transaction either doesn't exist or has already been spent")

func (w *DashElectrumWallet) BumpFee(txid string) (*wire.MsgTx, error) {

	return nil, wallet.ErrWalletFnNotImplemented // FIXME: SweepAddress prevOutputFInder

	// txn, err := w.txstore.Txns().Get(txid)
	// if err != nil {
	// 	return nil, err
	// }
	// if txn.Height > 0 {
	// 	return nil, BumpFeeAlreadyConfirmedError
	// }
	// if txn.Height < 0 {
	// 	return nil, BumpFeeTransactionDeadError
	// }

	// // As a policy this wallet will never do RBF.

	// // Check utxos for CPFP
	// utxos, _ := w.txstore.Utxos().GetAll()
	// for _, u := range utxos {
	// 	if u.Op.Hash.IsEqual(&txid) && u.AtHeight == 0 {
	// 		addr, err := w.ScriptToAddress(u.ScriptPubkey)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		in := wallet.TransactionInput{
	// 			LinkedAddress: addr,
	// 			OutpointIndex: u.Op.Index,
	// 			OutpointHash:  u.Op.Hash.CloneBytes(),
	// 			Value:         u.Value,
	// 		}

	// 		newAddress, err := w.GetUnusedAddress(wallet.INTERNAL)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		tx, err := w.SweepAddress([]wallet.TransactionInput{in}, newAddress, key, nil, wallet.FEE_BUMP)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	}
	// }

	// return nil, BumpFeeNotFoundError
}
