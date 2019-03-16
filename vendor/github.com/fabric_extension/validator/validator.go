package validator

import (
	"github.com/fabric_extension"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/handlers/library"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

const (
	channelConfigKey = "resourcesconfigtx.CHANNEL_CONFIG_KEY"
	peerNamespace    = ""
)

var peerLogger = flogging.MustGetLogger("peer")

func Init(cid string, gb *common.Block, peerLedger ledger.PeerLedger) (*txvalidator.TxValidator, error) {
	chanConf, err := retrievePersistedChannelConfig(peerLedger)
	if err != nil {
		return nil, err
	}

	var bundle *channelconfig.Bundle

	if chanConf != nil {
		bundle, err = channelconfig.NewBundle(cid, chanConf)
		if err != nil {
			return nil, err
		}
	} else {
		// Config was only stored in the statedb starting with v1.1 binaries
		// so if the config is not found there, extract it manually from the config block
		envelopeConfig, err := utils.ExtractEnvelope(gb, 0)
		if err != nil {
			return nil, err
		}

		bundle, err = channelconfig.NewBundleFromEnvelope(envelopeConfig)
		if err != nil {
			return nil, err
		}
	}
	ac, ok := bundle.ApplicationConfig()
	if !ok {
		ac = nil
	}

	cs := &chainSupport{
		Application: ac,
		ledger:      peerLedger}

	peerSingletonCallback := func(bundle *channelconfig.Bundle) {
		ac, ok := bundle.ApplicationConfig()
		if !ok {
			ac = nil
		}
		cs.Application = ac
		cs.Resources = bundle
	}
	cs.bundleSource = channelconfig.NewBundleSource(
		bundle,
		func(bundle *channelconfig.Bundle) {},
		func(bundle *channelconfig.Bundle) {},
		func(bundle *channelconfig.Bundle) {},
		peerSingletonCallback,
	)
	weighted := semaphore.NewWeighted(int64(fabric_extension.ThreadsPerBlock))
	sup := mockSupport{cs, weighted}

	ipRegistry := inproccontroller.NewRegistry()
	sccp := scc.NewProvider(Default, DefaultSupport, ipRegistry)

	libConf := library.Config{}
	if err = viperutil.EnhancedExactUnmarshalKey("peer.handlers", &libConf); err != nil {
		return nil, errors.WithMessage(err, "could not load YAML config")
	}
	reg := library.InitRegistry(libConf)
	val := txvalidator.NewTxValidator(sup, sccp, txvalidator.MapBasedPluginMapper(reg.Lookup(library.Validation).(map[string]validation.PluginFactory)))
	return val, nil

}

func retrievePersistedChannelConfig(ledger ledger.PeerLedger) (*common.Config, error) {
	qe, err := ledger.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()
	return retrievePersistedConf(qe, channelConfigKey)
}
func retrievePersistedConf(queryExecuter ledger.QueryExecutor, key string) (*common.Config, error) {
	serializedConfig, err := queryExecuter.GetState(peerNamespace, key)
	if err != nil {
		return nil, err
	}
	if serializedConfig == nil {
		return nil, nil
	}
	return deserialize(serializedConfig)
}

func deserialize(serializedConf []byte) (*common.Config, error) {
	conf := &common.Config{}
	if err := proto.Unmarshal(serializedConf, conf); err != nil {
		return nil, err
	}
	return conf, nil
}
