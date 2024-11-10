package smart_order

/*
	This file contains test cases for entry in smart order
	for normal and trailing smart orders
*/

import (
	"context"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/qmuntal/stateless"
	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"gitlab.com/crypto_project/core/strategy_service/src/service/strategies"
	"gitlab.com/crypto_project/core/strategy_service/src/service/strategies/smart_order"
	"gitlab.com/crypto_project/core/strategy_service/tests"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// smart order should create limit order while still in waitingForEntry state if not trailing
func TestSmartOrderGetInEntryLong(t *testing.T) {

	smartOrderModel := GetTestSmartOrderStrategy("entryLong")
	// price dips in the middle (This has no meaning now, reuse and then remove fake data stream)
	fakeDataStream := []interfaces.OHLCV{{
		Open:   7100,
		High:   7101,
		Low:    7000,
		Close:  7005,
		Volume: 30,
	}, { // Activation price
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  6900,
		Volume: 30,
	}, { // Hit entry
		Open:   7305,
		High:   7305,
		Low:    7300,
		Close:  7300,
		Volume: 30,
	}}
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := tests.NewMockedTradingAPI()
	keyId := primitive.NewObjectID()
	sm := tests.NewMockedStateMgmt(tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd()

	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Datafeed: df,
		Statsd: statsd,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	smartOrder.State.OnTransitioned(func(context context.Context, transition stateless.Transition) {
		log.Print("transition: source ", transition.Source.(string), ", destination ", transition.Destination.(string), ", trigger ", transition.Trigger.(string), ", isReentry ", transition.IsReentry())
	})
	go smartOrder.Start()
	time.Sleep(50 * time.Millisecond)

	// one call with 'buy' and one with 'BTC_USDT' should be done
	buyCallCount, buyFound := tradingApi.CallCount.Load("buy")
	btcUsdtCallCount, usdtBtcFound := tradingApi.CallCount.Load("BTC_USDT")

	if !buyFound || !usdtBtcFound || buyCallCount == 0 || btcUsdtCallCount == 0 {
		t.Error("There were 0 trading api calls with buy params and 0 with BTC_USDT params")
	} else {
		fmt.Println("Success! There were " + strconv.Itoa(buyCallCount.(int)) + " trading api calls with buy params and " + strconv.Itoa(btcUsdtCallCount.(int)) + " with BTC_USDT params")
	}
}

// smart order should create limit order while still in waitingForEntry state if not trailing
func TestSmartOrderGetInEntryShort(t *testing.T) {
	smartOrderModel := GetTestSmartOrderStrategy("entryShort")
	// price rises (This has no meaning now, reuse and then remove fake data stream)
	fakeDataStream := []interfaces.OHLCV{{
		Open:   6800,
		High:   7101,
		Low:    6750,
		Close:  6900,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7100,
		Low:    6800,
		Close:  6900,
		Volume: 30,
	}, { // Hit entry
		Open:   6950,
		High:   7305,
		Low:    6950,
		Close:  7010,
		Volume: 30,
	}}
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := tests.NewMockedTradingAPI()
	keyId := primitive.NewObjectID()
	sm := tests.NewMockedStateMgmt(tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd();
	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Statsd: statsd,
		Datafeed: df,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	smartOrder.State.OnTransitioned(func(context context.Context, transition stateless.Transition) {
		log.Print("transition: source ", transition.Source.(string), ", destination ", transition.Destination.(string), ", trigger ", transition.Trigger.(string), ", isReentry ", transition.IsReentry())
	})
	go smartOrder.Start()
	time.Sleep(50 * time.Millisecond)

	// one call with 'sell' and one with 'BTC_USDT' should be done
	sellCallCount, sellOk := tradingApi.CallCount.Load("sell")
	btcUsdtCallCount, usdtBtcOk := tradingApi.CallCount.Load("BTC_USDT")
	if !sellOk || !usdtBtcOk || sellCallCount == 0 || btcUsdtCallCount == 0 {
		t.Error("There were 0 trading api calls with sell params and 0 with BTC_USDT params")
	} else {
		fmt.Println("Success! There were " + strconv.Itoa(sellCallCount.(int)) + " trading api calls with sell params and " + strconv.Itoa(btcUsdtCallCount.(int)) + " with BTC_USDT params")
	}
}

// smart order should transition to TrailingEntry state if ActivatePrice > 0 AND currect OHLCV close price is less than condition price
func TestSmartOrderGetInTrailingEntryLong(t *testing.T) {
	smartOrderModel := GetTestSmartOrderStrategy("trailingEntryLong")
	// price rises
	fakeDataStream := []interfaces.OHLCV{{
		Open:   7100,
		High:   7101,
		Low:    7000,
		Close:  6900,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  7001,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  7100,
		Volume: 30,
	}}
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := *tests.NewMockedTradingAPI()
	keyId := primitive.NewObjectID()
	//sm := mongodb.StateMgmt{}
	sm := tests.NewMockedStateMgmt(&tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd()
	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Statsd: statsd,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	go smartOrder.Start()
	time.Sleep(50 * time.Millisecond)
	isInState, _ := smartOrder.State.IsInState(smart_order.TrailingEntry)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not TrailingEntry (State: " + stateStr + ")")
	}
}

// smart order should wait for entry if price condition is not met
/*func TestSmartOrderShouldWaitForTrailingEntryLong(t *tests.T) {
	smartOrderModel := GetTestSmartOrderStrategy("trailingEntryLong")
	// price falls
	fakeDataStream := []strategies.OHLCV{{
		Open:   7100,
		High:   7101,
		Low:    7000,
		Close:  6900,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  6800,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  6700,
		Volume: 30,
	}}
	df := NewMockedDataFeed(fakeDataStream)
	tradingApi := *NewMockedTradingAPI()
	strategy := strategies.Strategy{
		Model: &smartOrderModel,
	}
	keyId := primitive.NewObjectID()
	//sm := mongodb.StateMgmt{}
	sm := MockStateMgmt{}
	smartOrder := strategies.New(&strategy, df, tradingApi, &keyId, &sm)
	go smartOrder.Start()
	time.Sleep(800 * time.Millisecond)
	isInState, _ := smartOrder.State.IsInState(strategies.WaitForEntry)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not WaitForEntry (State: " + stateStr + ")")
	}
}*/

// smart order should transition to TrailingEntry state if ActivatePrice > 0 AND currect OHLCV close price is more than condition price
func TestSmartOrderGetInTrailingEntryShort(t *testing.T) {
	smartOrderModel := GetTestSmartOrderStrategy("trailingEntryShort")
	// price falls
	fakeDataStream := []interfaces.OHLCV{{
		Open:   7100,
		High:   7101,
		Low:    7000,
		Close:  7005,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  6900,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  6800,
		Volume: 30,
	}}
	df := tests.NewMockedDataFeed(fakeDataStream)
	tradingApi := *tests.NewMockedTradingAPI()
	keyId := primitive.NewObjectID()
	sm := tests.NewMockedStateMgmt(&tradingApi, df)
	logger, statsd := tests.GetLoggerStatsd()
	strategy := strategies.Strategy{
		Model:     &smartOrderModel,
		StateMgmt: &sm,
		Log: logger,
		Statsd: statsd,
		SettlementMutex: &redsync.Mutex{},
	}
	smartOrder := smart_order.New(&strategy, df, tradingApi, strategy.Statsd, &keyId, &sm)
	go smartOrder.Start()
	time.Sleep(50 * time.Millisecond)
	isInState, _ := smartOrder.State.IsInState(smart_order.TrailingEntry)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not TrailingEntry (State: " + stateStr + ")")
	}
}

// smart order should wait for entry if price condition is not met
/*func TestSmartOrderShouldWaitForTrailingEntryShort(t *tests.T) {
	smartOrderModel := GetTestSmartOrderStrategy("trailingEntryShort")
	// price rises
	fakeDataStream := []strategies.OHLCV{{
		Open:   7100,
		High:   7101,
		Low:    7000,
		Close:  7105,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  7200,
		Volume: 30,
	}, {
		Open:   7005,
		High:   7005,
		Low:    6900,
		Close:  7300,
		Volume: 30,
	}}
	df := NewMockedDataFeed(fakeDataStream)
	tradingApi := *NewMockedTradingAPI()
	strategy := strategies.Strategy{
		Model: &smartOrderModel,
	}
	keyId := primitive.NewObjectID()
	sm := MockStateMgmt{}
	smartOrder := strategies.New(&strategy, df, tradingApi, &keyId, &sm)
	go smartOrder.Start()
	time.Sleep(800 * time.Millisecond)
	isInState, _ := smartOrder.State.IsInState(strategies.WaitForEntry)
	if !isInState {
		state, _ := smartOrder.State.State(context.Background())
		stateStr := fmt.Sprintf("%v", state)
		t.Error("SmartOrder state is not WaitForEntry (State: " + stateStr + ")")
	}
}*/
