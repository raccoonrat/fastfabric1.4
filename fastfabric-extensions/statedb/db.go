package statedb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/fastfabric-extensions/experiment"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var KeyNotFound = errors.New("key not found")

type DB struct {
	db *ValueHashtable
}

func CreateDB() (*DB){
	db := &DB {db:NewHT()}
	ccname := "PaymentApp"
	cdbytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    ccname,
		Version: "v1",
		Vscc:    "vscc",
		Policy:  signedByAnyMember([]string{"SampleOrg"}),
	})

	db.Put(constructLevelKey(experiment.Current.ChannelId, constructCompositeKey("lscc", ccname)),
		encodeValue(cdbytes, version.NewHeight(0, 0)), true)

	return db
}

func (ledger *DB) Open() {
}

func (ledger *DB) Close() error {
	ledger.db.Cleanup()
	return nil
}

func (ledger *DB) Get(bytes []byte) ([]byte, error) {
	if val, err := ledger.db.Get(bytes); err != nil {
		return nil, KeyNotFound
	} else {
		return val, nil
	}
}
func (ledger *DB) Put(key []byte, value []byte, sync bool) error {
	return ledger.db.Put(key, value)
}
func (ledger *DB) Delete(key []byte, sync bool) error {
	return ledger.db.Remove(key)
}
func (ledger *DB) GetIterator(sk []byte, ek []byte) iterator.Iterator {
	keys := ledger.db.GetKeys(sk, ek)
	return &Iterimpl{keys:keys, idx:0, db:ledger.db}
}

func (ledger *DB) GetKeys() []string {
	var keys []string
	fmt.Println("db:", ledger.db)
	for _, key:= range ledger.db.GetKeys([]byte{0}, []byte{255,255,255,255,255,255,255,255,255,255,255,255,255,255}){
		keys = append(keys, string(key))
	}
	return keys
}

type Iterimpl struct {
	keys [][]byte
	idx  int
	err  error
	db *ValueHashtable
}

func (i *Iterimpl) First() bool {
	i.idx=0
	return i.keys != nil
}

func (i *Iterimpl) Last() bool {
	i.idx = len(i.keys)-1
	return i.keys != nil
}

func (i *Iterimpl) Seek(key []byte) bool {
	for tempidx := 0;tempidx< len(i.keys);tempidx++{
		if bytes.Compare(key,i.keys[tempidx])>-1{
			i.idx = tempidx
			return true
		}
	}
	return false
}

func (i *Iterimpl) Next() bool {
	i.idx++
	return i.idx < len(i.keys)
}

func (i *Iterimpl) Prev() bool {
	i.idx--
	return i.idx >=0
}

func (Iterimpl) Release() {
	return
}

func (Iterimpl) SetReleaser(releaser util.Releaser) {
	return
}

func (i *Iterimpl) Valid() bool {
	return i.idx >=0 && i.idx < len(i.keys)
}

func (i *Iterimpl) Error() error {
	return i.err
}

func (i *Iterimpl) Key() []byte {
	if !i.Valid() {
		return nil
	}
	return i.keys[i.idx]
}

func (i *Iterimpl) Value() []byte {
	if !i.Valid() {
		return nil
	}

	var val []byte
	val,i.err = i.db.Get(i.Key())
	return val
}


func signedByAnyMember(ids []string) []byte {
	p := cauthdsl.SignedByAnyMember(ids)
	return utils.MarshalOrPanic(p)
}

var compositeKeySep = []byte{0x00}

func constructCompositeKey(ns string, key string) []byte {
	return append(append([]byte(ns), compositeKeySep...), []byte(key)...)
}

func encodeValue(value []byte, version *version.Height) []byte {
	encodedValue := version.ToBytes()
	if value != nil {
		encodedValue = append(encodedValue, value...)
	}
	return encodedValue
}