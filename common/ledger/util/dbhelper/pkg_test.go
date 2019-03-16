/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbhelper

import (
	"github.com/fabric_extension/db"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
)

const testDBPath = "/tmp/fabric/ledgertests/util/dbhelper"

type testDBEnv struct {
	t    *testing.T
	path string
	db   *db.DB
}

type testDBProviderEnv struct {
	t        *testing.T
	path     string
	provider *Provider
}

func newTestProviderEnv(t *testing.T, path string) *testDBProviderEnv {
	testProviderEnv := &testDBProviderEnv{t: t, path: path}
	testProviderEnv.cleanup()
	testProviderEnv.provider = NewProvider()
	return testProviderEnv
}

func (dbEnv *testDBEnv) cleanup() {
	if dbEnv.db != nil {
		dbEnv.db.Close()
	}
	testutil.AssertNoError(dbEnv.t, os.RemoveAll(dbEnv.path), "")
}

func (providerEnv *testDBProviderEnv) cleanup() {
	if providerEnv.provider != nil {
		providerEnv.provider.Close()
	}
	testutil.AssertNoError(providerEnv.t, os.RemoveAll(providerEnv.path), "")
}
