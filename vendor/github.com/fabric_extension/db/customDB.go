package db

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/fabric_extension/experiment"
	"github.com/fabric_extension/hashtable"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/utils"
)

var KeyNotFound = errors.New("key not found")

type DB struct {
	db *hashtable.ValueHashtable
}

var Current *DB

func New() (*DB){
	db := &DB {db:hashtable.New()}
	ccname := "PaymentApp"
	cdbytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    ccname,
		Version: "v1",
		Vscc:    "vscc",
		Policy:  signedByAnyMember([]string{"SampleOrg"}),
	})

	db.Put(constructLevelKey(experiment.Current.ChannelId, constructCompositeKey("lscc", ccname)),
		encodeValue(cdbytes, version.NewHeight(0, 0)), true)

	Current = db
	return db
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
func (ledger *DB) GetIterator(sk []byte, ek []byte) Iterator {
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
	db *hashtable.ValueHashtable
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

func (Iterimpl) SetReleaser(releaser Releaser) {
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

type Iterator interface {
	CommonIterator

	// Key returns the key of the current key/value pair, or nil if done.
	// The caller should not modify the contents of the returned slice, and
	// its contents may change on the next call to any 'seeks method'.
	Key() []byte

	// Value returns the value of the current key/value pair, or nil if done.
	// The caller should not modify the contents of the returned slice, and
	// its contents may change on the next call to any 'seeks method'.
	Value() []byte
}

// CommonIterator is the interface that wraps common iterator methods.
type CommonIterator interface {
	IteratorSeeker

	// util.Releaser is the interface that wraps basic Release method.
	// When called Release will releases any resources associated with the
	// iterator.
	Releaser

	// util.ReleaseSetter is the interface that wraps the basic SetReleaser
	// method.
	ReleaseSetter

	// TODO: Remove this when ready.
	Valid() bool

	// Error returns any accumulated error. Exhausting all the key/value pairs
	// is not considered to be an error.
	Error() error
}

// Releaser is the interface that wraps the basic Release method.
type Releaser interface {
	// Release releases associated resources. Release should always success
	// and can be called multiple times without causing error.
	Release()
}

// ReleaseSetter is the interface that wraps the basic SetReleaser method.
type ReleaseSetter interface {
	// SetReleaser associates the given releaser to the resources. The
	// releaser will be called once coresponding resources released.
	// Calling SetReleaser with nil will clear the releaser.
	//
	// This will panic if a releaser already present or coresponding
	// resource is already released. Releaser should be cleared first
	// before assigned a new one.
	SetReleaser(releaser Releaser)
}

// IteratorSeeker is the interface that wraps the 'seeks method'.
type IteratorSeeker interface {
	// First moves the iterator to the first key/value pair. If the iterator
	// only contains one key/value pair then First and Last would moves
	// to the same key/value pair.
	// It returns whether such pair exist.
	First() bool

	// Last moves the iterator to the last key/value pair. If the iterator
	// only contains one key/value pair then First and Last would moves
	// to the same key/value pair.
	// It returns whether such pair exist.
	Last() bool

	// Seek moves the iterator to the first key/value pair whose key is greater
	// than or equal to the given key.
	// It returns whether such pair exist.
	//
	// It is safe to modify the contents of the argument after Seek returns.
	Seek(key []byte) bool

	// Next moves the iterator to the next key/value pair.
	// It returns whether the iterator is exhausted.
	Next() bool

	// Prev moves the iterator to the previous key/value pair.
	// It returns whether the iterator is exhausted.
	Prev() bool
}


func signedByAnyMember(ids []string) []byte {
	p := cauthdsl.SignedByAnyMember(ids)
	return utils.MarshalOrPanic(p)
}

var compositeKeySep = []byte{0x00}

func constructCompositeKey(ns string, key string) []byte {
	return append(append([]byte(ns), compositeKeySep...), []byte(key)...)
}

var dbNameKeySep = []byte{0x00}

func constructLevelKey(dbName string, key []byte) []byte {
	return append(append([]byte(dbName), dbNameKeySep...), key...)
}

func encodeValue(value []byte, version *version.Height) []byte {
	encodedValue := version.ToBytes()
	if value != nil {
		encodedValue = append(encodedValue, value...)
	}
	return encodedValue
}