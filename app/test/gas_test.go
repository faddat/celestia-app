package app_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/testutil/testnode"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

func TestGasConsumptiontionTestSuite(t *testing.T) {
	// t.Skip("Skipping Gas Consumption Test")
	suite.Run(t, new(GasConsumptionTestSuite))
}

type GasConsumptionTestSuite struct {
	suite.Suite

	cleanups []func()
	accounts []string
	cctx     testnode.Context

	mut            sync.Mutex
	accountCounter int
}

func (s *GasConsumptionTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping test in unit-tests or race-detector mode.")
	}

	s.T().Log("setting up integration test suite")
	require := s.Require()

	// we create an arbitrary number of funded accounts
	for i := 0; i < 300; i++ {
		s.accounts = append(s.accounts, tmrand.Str(9))
	}

	genState, kr, err := testnode.DefaultGenesisState(s.accounts...)
	require.NoError(err)

	tmCfg := testnode.DefaultTendermintConfig()
	tmCfg.Consensus.TimeoutCommit = time.Second

	tmNode, app, cctx, err := testnode.New(
		s.T(),
		testnode.DefaultParams(),
		tmCfg,
		false,
		genState,
		kr,
	)
	require.NoError(err)

	cctx, stopNode, err := testnode.StartNode(tmNode, cctx)
	require.NoError(err)
	s.cleanups = append(s.cleanups, stopNode)

	cctx, cleanupGRPC, err := testnode.StartGRPCServer(app, testnode.DefaultAppConfig(), cctx)
	require.NoError(err)
	s.cleanups = append(s.cleanups, cleanupGRPC)

	s.cctx = cctx
}

func (s *GasConsumptionTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	for _, c := range s.cleanups {
		c()
	}
}

func (s *GasConsumptionTestSuite) unusedAccount() string {
	s.mut.Lock()
	acc := s.accounts[s.accountCounter]
	s.accountCounter++
	s.mut.Unlock()
	return acc
}

func (s *GasConsumptionTestSuite) TestStandardGasConsumption() {
	encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

	type gasTest struct {
		name    string
		msgFunc func() (msg sdk.Msg, signer string)
		hash    string
	}
	tests := []gasTest{
		{
			name: "send 1 TIA",
			msgFunc: func() (msg sdk.Msg, signer string) {
				account1, account2 := s.unusedAccount(), s.unusedAccount()
				msgSend := banktypes.NewMsgSend(
					getAddress(account1, s.cctx.Keyring),
					getAddress(account2, s.cctx.Keyring),
					sdk.NewCoins(sdk.NewCoin(app.BondDenom, sdk.NewInt(1000000))),
				)
				return msgSend, account1
			},
		},
		{
			name: "send 1 TIA",
			msgFunc: func() (msg sdk.Msg, signer string) {
				account1, account2 := s.unusedAccount(), s.unusedAccount()
				msgSend := banktypes.NewMsgSend(
					getAddress(account1, s.cctx.Keyring),
					getAddress(account2, s.cctx.Keyring),
					sdk.NewCoins(sdk.NewCoin(app.BondDenom, sdk.NewInt(1000000))),
				)
				return msgSend, account1
			},
		},
	}
	for i, tt := range tests {
		res, err := testnode.SignAndBroadcastTx(encCfg, s.cctx, tt.signingAccount, tt.msg)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), res)
		require.Equal(s.T(), abci.CodeTypeOK, res.Code)
		tests[i].hash = res.TxHash
	}
	err := s.cctx.WaitForNextBlock()
	require.NoError(s.T(), err)
	err = s.cctx.WaitForNextBlock()
	require.NoError(s.T(), err)
	for _, tt := range tests {
		res, err := queryTx(s.cctx.Context, tt.hash, false)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), res)
		fmt.Println("sync new", res.TxResult.GasUsed)
	}
}

func getAddress(account string, kr keyring.Keyring) sdk.AccAddress {
	rec, err := kr.Key(account)
	if err != nil {
		panic(err)
	}
	addr, err := rec.GetAddress()
	if err != nil {
		panic(err)
	}
	return addr
}
