package smart_order

import (
	"context"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"log"
	"testing"
	"time"

	"github.com/qmuntal/stateless"
	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"gitlab.com/crypto_project/core/strategy_service/src/service/strategies"
	"gitlab.com/crypto_project/core/strategy_service/src/service/strategies/smart_order"
	"gitlab.com/crypto_project/core/strategy_service/tests"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSmartOrderMarketEntryAndTrailingExit(t *testing.T) {
	fakeDataStream := []interfaces.OHLCV{
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		},
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		}, { // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		}, { // Going up..
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7005,
			Volume: 30,
		}, { // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		}, { // Oh wow, its pump!
			Open:   7045,
			High:   7045,
			Low:    7045,
			Close:  7045,
			Volume: 30,
		}, { // Spiked down, ok, up trend is over, we are taking profits now
			Open:   7145,
			High:   7145,
			Low:    7145,
			Close:  7145,
			Volume: 30,
		}, { // Ok, its going down?
			Open:   7142,
			High:   7142,
			Low:    7142,
			Close:  7142,
			Volume: 30,
		}, { // Are exiting now? Its down 12.25% now, should be closed already
			Open:   7138,
			High:   7138,
			Low:    7138,
			Close:  7138,
			Volume: 30,
		},
	}
	smartOrderModel := GetTestSmartOrderStrategy("marketEntryTrailingExitLeverage")
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := tests.NewMockedTradingAPIWithMarketAccess(df)
	tradingApi.BuyDelay = 300
	tradingApi.SellDelay = 300
	keyId := primitive.NewObjectID()
	sm := tests.NewMockedStateMgmt(tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd()
	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Statsd: statsd,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	smartOrder.State.OnTransitioned(func(context context.Context, transition stateless.Transition) {
		log.Print("transition: source ", transition.Source.(string), ", destination ", transition.Destination.(string), ", trigger ", transition.Trigger.(string), ", isReentry ", transition.IsReentry())
	})
	go smartOrder.Start()
	time.Sleep(4 * time.Second) //TODO: takes way too long
	isInState, _ := smartOrder.State.IsInState(smart_order.End)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not End (State: " + stateStr + ")")
	}
}

func TestSmartOrderMarketEntryAndThenFollowTrailing(t *testing.T) {
	fakeDataStream := []interfaces.OHLCV{
		{
			Open:   7100,
			High:   7101,
			Low:    7000,
			Close:  7005,
			Volume: 30,
		}, { // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		}, { // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		},
		{ // Its trading around, like in real life
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7000,
			Volume: 30,
		}, { // Going up..
			Open:   7005,
			High:   7005,
			Low:    7005,
			Close:  7005,
			Volume: 30,
		}, { // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		},
		{ // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		},
		{ // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		},
		{ // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		},
		{ // Going up....
			Open:   7015,
			High:   7015,
			Low:    7015,
			Close:  7015,
			Volume: 30,
		}, { // Oh wow, its pump!
			Open:   7045,
			High:   7045,
			Low:    7045,
			Close:  7045,
			Volume: 30,
		}, { // Spiked down, ok, up trend is over, we are taking profits now
			Open:   7145,
			High:   7145,
			Low:    7145,
			Close:  7145,
			Volume: 30,
		}, { // Ok, its going down?
			Open:   7142,
			High:   7142,
			Low:    7142,
			Close:  7142,
			Volume: 30,
		}, { // Are exiting now? Its down 12.25% now, should be closed already
			Open:   7138,
			High:   7138,
			Low:    7138,
			Close:  7138,
			Volume: 30,
		},
	}
	smartOrderModel := GetTestSmartOrderStrategy("marketEntryTrailingExitLeverage")
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := tests.NewMockedTradingAPIWithMarketAccess(df)
	tradingApi.BuyDelay = 300
	tradingApi.SellDelay = 300
	keyId := primitive.NewObjectID()
	sm := tests.NewMockedStateMgmt(tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd()
	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Statsd: statsd,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	smartOrder.State.OnTransitioned(func(context context.Context, transition stateless.Transition) {
		log.Print("transition: source ", transition.Source.(string), ", destination ", transition.Destination.(string), ", trigger ", transition.Trigger.(string), ", isReentry ", transition.IsReentry())
	})
	go smartOrder.Start()
	time.Sleep(10 * time.Second) //TODO: takes way too long
	isInState, _ := smartOrder.State.IsInState(smart_order.End)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not End (State: " + stateStr + ")")
	}
}
