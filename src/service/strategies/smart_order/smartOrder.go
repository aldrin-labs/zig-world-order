package smart_order

import (
	"context"
	"gitlab.com/crypto_project/core/strategy_service/src/trading/orders"
	"go.uber.org/zap"

	// "go.uber.org/zap"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/qmuntal/stateless"
	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"gitlab.com/crypto_project/core/strategy_service/src/sources/mongodb/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	WaitForEntry       = "WaitForEntry"
	TrailingEntry      = "TrailingEntry"
	PartiallyEntry     = "PartiallyEntry"
	InEntry            = "InEntry"
	InMultiEntry       = "InMultiEntry"
	TakeProfit         = "TakeProfit"
	Stoploss           = "Stoploss"
	WaitOrderOnTimeout = "WaitOrderOnTimeout"
	WaitOrder          = "WaitOrder"
	End                = "End"
	Canceled           = "Canceled"
	EnterNextTarget    = "EnterNextTarget"
	Timeout            = "Timeout"
	Error              = "Error"
	HedgeLoss          = "HedgeLoss"
	WaitLossHedge      = "WaitLossHedge"
)

const (
	TriggerTrade             = "Trade"
	TriggerSpread            = "Spread"
	TriggerAveragingEntryOrderExecuted = "TriggerAveragingEntryOrderExecuted"
	TriggerOrderExecuted     = "TriggerOrderExecuted"
	CheckExistingOrders      = "CheckExistingOrders"
	CheckHedgeLoss           = "CheckHedgeLoss"
	CheckProfitTrade         = "CheckProfitTrade"
	CheckTrailingProfitTrade = "CheckTrailingProfitTrade"
	CheckTrailingLossTrade   = "CheckTrailingLossTrade"
	CheckSpreadProfitTrade   = "CheckSpreadProfitTrade"
	CheckLossTrade           = "CheckLossTrade"
	Restart                  = "Restart"
	ReEntry                  = "ReEntry"
	TriggerTimeout           = "TriggerTimeout"
)

// A SmartOrder takes strategy to execute with context by the service runtime.
type SmartOrder struct {
	Strategy                interfaces.IStrategy // TODO(khassanov): can we use type instead of the interface here?
	State                   *stateless.StateMachine
	ExchangeName            string
	KeyId                   *primitive.ObjectID
	DataFeed                interfaces.IDataFeed
	ExchangeApi             interfaces.ITrading
	Statsd                  interfaces.IStatsClient
	StateMgmt               interfaces.IStateMgmt
	IsWaitingForOrder       sync.Map // TODO: this must be filled on start of SM if not first start (e.g. restore the state by checking order statuses)
	IsEntryOrderPlaced      bool     // we need it for case when response from createOrder was returned after entryTimeout was executed
	OrdersMap               map[string]bool
	StatusByOrderId         sync.Map
	QuantityAmountPrecision int64
	QuantityPricePrecision  int64
	Lock                    bool
	StopLock                bool
	LastTrailingTimestamp   int64
	SelectedExitTarget      int // number of current order
	SelectedEntryTarget     int // represents what amount of targets executed for the SM by averaging
	OrdersMux               sync.Mutex
	StopMux                 sync.Mutex
}

const (
	Nearest = iota
	Floor
	Ceil
)

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

// toFixed returns floating point number rounded to precision given to nearest, floor or ceil function.
func (sm *SmartOrder) toFixed(n float64, precision int64, mode int) float64 {
	rank := math.Pow(10, float64(precision))
	switch mode {
	case Ceil:
		return math.Ceil(n*rank) / rank
	case Nearest:
		return float64(round(n*rank)) / rank
	case Floor:
		return math.Floor(n*rank) / rank
	default:
		return n
	}
}

// New instantiates new smart order with given strategy.
func New(strategy interfaces.IStrategy, DataFeed interfaces.IDataFeed, TradingAPI interfaces.ITrading, Statsd interfaces.IStatsClient, keyId *primitive.ObjectID, stateMgmt interfaces.IStateMgmt) *SmartOrder {

	sm := &SmartOrder{
		Strategy:           strategy,
		DataFeed:           DataFeed,
		ExchangeApi:        TradingAPI,
		Statsd:             Statsd,
		KeyId:              keyId,
		StateMgmt:          stateMgmt,
		Lock:               false,
		SelectedExitTarget: 0,
		OrdersMap:          map[string]bool{},
	}

	initState := WaitForEntry
	pricePrecision, amountPrecision := stateMgmt.GetMarketPrecision(strategy.GetModel().Conditions.Pair, strategy.GetModel().Conditions.MarketType)
	sm.QuantityPricePrecision = pricePrecision
	sm.QuantityAmountPrecision = amountPrecision
	// if state is not empty but if its in the end and open ended, then we skip state value, since want to start over
	if strategy.GetModel().State != nil && strategy.GetModel().State.State != "" && !(strategy.GetModel().State.State == End && strategy.GetModel().Conditions.ContinueIfEnded == true) {
		initState = strategy.GetModel().State.State
	}
	State := stateless.NewStateMachineWithMode(initState, 1) // TODO(khassanov): rename it, State is not a StateMachine
	// TODO(khassanov): prepare boilerplate state machine in package init and just copy it here
	State.OnTransitioned(func(ctx context.Context, tr stateless.Transition) {
		sm.Strategy.GetLogger().Info("sm state transition",
			zap.String("trigger", fmt.Sprintf("%v", tr.Trigger)),
			zap.String("source", fmt.Sprintf("%v", tr.Source)),
			zap.String("dest", fmt.Sprintf("%v", tr.Destination)),
		)
	})

	// define triggers and input types:
	State.SetTriggerParameters(TriggerTrade, reflect.TypeOf(interfaces.OHLCV{}))
	State.SetTriggerParameters(CheckExistingOrders, reflect.TypeOf(models.MongoOrder{}))
	State.SetTriggerParameters(TriggerSpread, reflect.TypeOf(interfaces.SpreadData{}))

	/*
		Smart Order life cycle:
			1) first need to go into entry, so we wait for entry and put orders before if possible
			2) we may go into waiting for trailing entry if activate price was specified ( and placing stop-limit/market orders to catch entry, and cancel existing )
			3) ok we are in entry and now wait for profit or loss ( also we can try to place all orders )
			4) we may go to exit on timeout if profit/loss
			5) so we'll wait for any target or trailing target
			6) or stop-loss
	*/

	State.Configure(WaitForEntry).
		PermitDynamic(TriggerTrade, sm.exitWaitEntry, sm.checkWaitEntry).
		PermitDynamic(TriggerSpread, sm.exitWaitEntry, sm.checkSpreadEntry).
		PermitDynamic(CheckExistingOrders, sm.exitWaitEntry, sm.checkExistingOrders).
		Permit(TriggerTimeout, Timeout).
		OnEntry(sm.onStart)

	State.Configure(TrailingEntry).
		Permit(TriggerTrade, InEntry, sm.checkTrailingEntry).
		Permit(CheckExistingOrders, InEntry, sm.checkExistingOrders).
		OnEntry(sm.enterTrailingEntry)

	State.Configure(InEntry).
		PermitDynamic(CheckProfitTrade, sm.exit, sm.checkProfit).
		PermitDynamic(CheckTrailingProfitTrade, sm.exit, sm.checkTrailingProfit).
		PermitDynamic(CheckLossTrade, sm.exit, sm.checkLoss).
		PermitDynamic(CheckExistingOrders, sm.exit, sm.checkExistingOrders).
		PermitDynamic(CheckHedgeLoss, sm.exit, sm.checkLossHedge).
		OnEntry(sm.enterEntry)

	State.Configure(InMultiEntry).
		PermitDynamic(CheckExistingOrders, sm.exit, sm.checkExistingOrders).
		PermitDynamic(TriggerAveragingEntryOrderExecuted, sm.enterMultiEntry)

	State.Configure(WaitLossHedge).
		PermitDynamic(CheckHedgeLoss, sm.exit, sm.checkLossHedge).
		OnEntry(sm.enterWaitLossHedge)

	State.Configure(TakeProfit).
		PermitDynamic(CheckProfitTrade, sm.exit, sm.checkProfit).
		PermitDynamic(CheckTrailingProfitTrade, sm.exit, sm.checkTrailingProfit).
		PermitDynamic(CheckLossTrade, sm.exit, sm.checkLoss).
		PermitDynamic(CheckExistingOrders, sm.exit, sm.checkExistingOrders).
		PermitDynamic(CheckHedgeLoss, sm.exit, sm.checkLossHedge).
		OnEntry(sm.enterTakeProfit)

	State.Configure(Stoploss).
		PermitDynamic(CheckProfitTrade, sm.exit, sm.checkProfit).
		PermitDynamic(CheckTrailingProfitTrade, sm.exit, sm.checkTrailingProfit).
		PermitDynamic(CheckLossTrade, sm.exit, sm.checkLoss).
		PermitDynamic(CheckExistingOrders, sm.exit, sm.checkExistingOrders).
		PermitDynamic(CheckHedgeLoss, sm.exit, sm.checkLossHedge).
		OnEntry(sm.enterStopLoss)

	State.Configure(HedgeLoss).
		PermitDynamic(CheckTrailingLossTrade, sm.exit, sm.checkTrailingHedgeLoss).
		PermitDynamic(CheckExistingOrders, sm.exit, sm.checkExistingOrders)

	State.Configure(Timeout).
		Permit(Restart, WaitForEntry)

	State.Configure(End).
		PermitReentry(CheckExistingOrders, sm.checkExistingOrders).
		OnEntry(sm.enterEnd)

	_ = State.Activate()

	sm.State = State
	sm.ExchangeName = sm.Strategy.GetModel().Conditions.Exchange
	// The following piece commented creates a file at ./graph.dot with graphviz formatted state chart and also prints
	// the same to stdout. Unfortunately I see it does not show PermitDymanic connections at least here at start.
	// fmt.Printf(sm.State.ToGraph())
	// graph := sm.State.ToGraph()
	// fd, _ := os.Create("./graph.dot")
	// writer := bufio.NewWriter(fd)
	// writer.WriteString(graph)
	// writer.Flush()
	// fmt.Println("State chart graph written to ./graph.dot")
	// os.Exit(0)
	_ = sm.onStart(nil)
	return sm
}

func (sm *SmartOrder) checkIfShouldCancelIfAnyActive() {
	model := sm.Strategy.GetModel()

	if model.Conditions.CancelIfAnyActive && sm.StateMgmt.AnyActiveStrats(model) {
		model.Enabled = false
		sm.Strategy.GetModel().State.State = Canceled
		sm.StateMgmt.UpdateState(sm.Strategy.GetModel().ID, sm.Strategy.GetModel().State)
	}
}

func (sm *SmartOrder) onStart(ctx context.Context, args ...interface{}) error {
	sm.Strategy.GetLogger().Info("doing on start checks")
	sm.checkIfShouldCancelIfAnyActive()
	sm.hedge()
	sm.checkIfPlaceOrderInstantlyOnStart()
	go sm.checkTimeouts()
	return nil
}

func (sm *SmartOrder) checkIfPlaceOrderInstantlyOnStart() {
	model := sm.Strategy.GetModel()
	isMultiEntry := len(model.Conditions.EntryLevels) > 0
	isFirstRunSoStateIsEmpty := model.State.State == "" ||
		(model.State.State == WaitForEntry && model.Conditions.ContinueIfEnded && model.Conditions.WaitingEntryTimeout > 0)
	isFirstRunSoOrdersListsAreEmpty := (len(sm.Strategy.GetModel().State.Orders) +
		len(sm.Strategy.GetModel().State.ExecutedOrders)) == 0
	if isFirstRunSoStateIsEmpty && isFirstRunSoOrdersListsAreEmpty && model.Enabled &&
		!model.Conditions.EntrySpreadHunter && !isMultiEntry {
		entryIsNotTrailing := model.Conditions.EntryOrder.ActivatePrice == 0
		if entryIsNotTrailing { // then we must know exact price
			sm.IsWaitingForOrder.Store(WaitForEntry, true)
			sm.PlaceOrder(model.Conditions.EntryOrder.Price, 0.0, WaitForEntry)
		}
	}
	if isMultiEntry {
		sm.placeMultiEntryOrders(true)
	}
}

// getLastTargetAmount returns an amount for latest take a profit order.
func (sm *SmartOrder) getLastTargetAmount() float64 {
	sumAmount := 0.0
	length := len(sm.Strategy.GetModel().Conditions.ExitLevels)
	for i, target := range sm.Strategy.GetModel().Conditions.ExitLevels {
		if i < length-1 {
			baseAmount := target.Amount
			if target.Type == 1 {
				if baseAmount == 0 {
					baseAmount = 100
				}
				baseAmount = sm.Strategy.GetModel().Conditions.EntryOrder.Amount * (baseAmount / 100)
			}
			baseAmount = sm.toFixed(baseAmount, sm.QuantityAmountPrecision, Floor)
			sumAmount += baseAmount
		}
	}
	endTargetAmount := sm.Strategy.GetModel().Conditions.EntryOrder.Amount - sumAmount

	// This rounding is important due to limited IEEE-754 floating point arithmetics precision.
	// For instance if `endTargetAmount = 219.2 - 21.9`, you will get `197.29999999999998` instead of `197.3`.
	// Bug report related: https://cryptocurrenciesai.slack.com/archives/CCSQNTQ4W/p1615040296128100
	// We use Nearest because subtraction result may be slightly smaller or slightly bigger then true value.
	endTargetAmount = sm.toFixed(endTargetAmount, sm.QuantityAmountPrecision, Nearest)

	return endTargetAmount
}

func (sm *SmartOrder) exitWaitEntry(ctx context.Context, args ...interface{}) (stateless.State, error) {
	if sm.Strategy.GetModel().Conditions.EntryOrder.ActivatePrice != 0 {
		return TrailingEntry, nil
	}
	if len(sm.Strategy.GetModel().Conditions.EntryLevels) > 0 {
		return InMultiEntry, nil
	}
	return InEntry, nil
}

func (sm *SmartOrder) checkWaitEntry(ctx context.Context, args ...interface{}) bool {
	isWaitingForOrder, ok := sm.IsWaitingForOrder.Load(WaitForEntry)
	if ok && isWaitingForOrder.(bool) {
		return false
	}
	if sm.Strategy.GetModel().Conditions.EntrySpreadHunter {
		return false
	}
	if len(sm.Strategy.GetModel().Conditions.EntryLevels) > 0 {
		return false
	}
	currentOHLCV := args[0].(interfaces.OHLCV)
	model := sm.Strategy.GetModel()
	conditionPrice := model.Conditions.EntryOrder.Price
	isInstantMarketOrder := model.Conditions.EntryOrder.ActivatePrice == 0 && model.Conditions.EntryOrder.OrderType == "market"
	if model.Conditions.EntryOrder.ActivatePrice == -1 || isInstantMarketOrder {
		return true
	}
	isTrailing := model.Conditions.EntryOrder.ActivatePrice != 0
	if isTrailing {
		conditionPrice = model.Conditions.EntryOrder.ActivatePrice
	}

	switch model.Conditions.EntryOrder.Side {
	case "buy":
		if currentOHLCV.Close <= conditionPrice {
			return true
		}
		break
	case "sell":
		if currentOHLCV.Close >= conditionPrice {
			return true
		}
		break
	}

	return false
}

func (sm *SmartOrder) enterEntry(ctx context.Context, args ...interface{}) error {

	if len(sm.Strategy.GetModel().Conditions.EntryLevels) > 0 {
		return nil
	}
	isSpot := sm.Strategy.GetModel().Conditions.MarketType == 0
	// if we returned to InEntry from Stoploss timeout
	if sm.Strategy.GetModel().State.StopLossAt == -1 {
		sm.Strategy.GetModel().State.StopLossAt = 0
		sm.StateMgmt.UpdateState(sm.Strategy.GetModel().ID, sm.Strategy.GetModel().State)

		if isSpot {
			sm.TryCancelAllOrdersConsistently(sm.Strategy.GetModel().State.Orders)
			sm.PlaceOrder(0, 0.0, TakeProfit)
		}
		return nil
	}
	if currentOHLCV, ok := args[0].(interfaces.OHLCV); ok {
		sm.Strategy.GetModel().State.EntryPrice = currentOHLCV.Close
	}
	sm.Strategy.GetModel().State.State = InEntry
	sm.Strategy.GetModel().State.TrailingEntryPrice = 0
	sm.Strategy.GetModel().State.ExecutedOrders = []string{}
	go sm.StateMgmt.UpdateState(sm.Strategy.GetModel().ID, sm.Strategy.GetModel().State)

	//if !sm.Strategy.GetModel().Conditions.EntrySpreadHunter {
	if isSpot {
		sm.PlaceOrder(sm.Strategy.GetModel().State.EntryPrice, 0.0, InEntry)
		if !sm.Strategy.GetModel().Conditions.TakeProfitExternal {
			sm.Strategy.GetLogger().Info("placing take-profit")
			sm.PlaceOrder(0, 0.0, TakeProfit)
		}
	} else {
		go sm.PlaceOrder(sm.Strategy.GetModel().State.EntryPrice, 0.0, InEntry)
		if !sm.Strategy.GetModel().Conditions.TakeProfitExternal {
			sm.PlaceOrder(0, 0.0, TakeProfit)
		}
	}
	//}

	if !sm.Strategy.GetModel().Conditions.StopLossExternal && !isSpot {
		go sm.PlaceOrder(0, 0.0, Stoploss)
	}

	forcedLossOnSpot := !isSpot || (sm.Strategy.GetModel().Conditions.MandatoryForcedLoss && sm.Strategy.GetModel().Conditions.TakeProfitExternal)

	if sm.Strategy.GetModel().Conditions.ForcedLoss > 0 && forcedLossOnSpot &&
		(!sm.Strategy.GetModel().Conditions.StopLossExternal || sm.Strategy.GetModel().Conditions.MandatoryForcedLoss) {
		go sm.PlaceOrder(0, 0.0, "ForcedLoss")
	}
	// wait for creating hedgeStrategy
	if sm.Strategy.GetModel().Conditions.Hedging && sm.Strategy.GetModel().Conditions.HedgeStrategyId == nil ||
		sm.Strategy.GetModel().Conditions.HedgeStrategyId != nil {
		for {
			if sm.Strategy.GetModel().Conditions.HedgeStrategyId == nil {
				time.Sleep(1 * time.Second)
			} else {
				go sm.waitForHedge()
				return nil
			}
		}
	}

	return nil
}

func (sm *SmartOrder) checkProfit(ctx context.Context, args ...interface{}) bool {
	isWaitingForOrder, ok := sm.IsWaitingForOrder.Load(TakeProfit)
	model := sm.Strategy.GetModel()
	if ok &&
		isWaitingForOrder.(bool) &&
		model.Conditions.TimeoutIfProfitable == 0 &&
		model.Conditions.TimeoutLoss == 0 &&
		model.Conditions.WithoutLossAfterProfit == 0 {
		return false
	}

	if model.Conditions.TakeProfitExternal || model.Conditions.TakeProfitSpreadHunter {
		return false
	}
	if len(sm.Strategy.GetModel().Conditions.EntryLevels) > 0 {
		return false
	}

	currentOHLCV := args[0].(interfaces.OHLCV)
	if model.Conditions.TimeoutIfProfitable > 0 {
		isProfitable := (model.Conditions.EntryOrder.Side == "buy" && model.State.EntryPrice < currentOHLCV.Close) ||
			(model.Conditions.EntryOrder.Side == "sell" && model.State.EntryPrice > currentOHLCV.Close)
		if isProfitable && model.State.ProfitableAt == 0 {
			model.State.ProfitableAt = time.Now().Unix()
			go func(profitableAt int64) {
				time.Sleep(time.Duration(model.Conditions.TimeoutIfProfitable) * time.Second)
				stillTimeout := profitableAt == model.State.ProfitableAt
				if stillTimeout {
					sm.PlaceOrder(-1, 0.0, TakeProfit)
				}
			}(model.State.ProfitableAt)
		} else if !isProfitable {
			model.State.ProfitableAt = 0
		}
	}

	if model.Conditions.ExitLevels != nil {
		amount := 0.0
		switch model.Conditions.EntryOrder.Side {
		case "buy":
			for i, level := range model.Conditions.ExitLevels {
				if model.State.ReachedTargetCount < i+1 && level.ActivatePrice == 0 {
					if level.Type == 1 && currentOHLCV.Close >= (model.State.EntryPrice*(100+level.Price/model.Conditions.Leverage)/100) ||
						level.Type == 0 && currentOHLCV.Close >= level.Price {
						model.State.ReachedTargetCount += 1
						if level.Type == 0 {
							amount += level.Amount
						} else if level.Type == 1 {
							amount += model.Conditions.EntryOrder.Amount * (level.Amount / 100)
						}
					}
				}

				//if currentOHLCV.Close >= model.State.EntryPrice * (1 + model.Conditions.WithoutLossAfterProfit/model.Conditions.Leverage/100){
				//	sm.PlaceOrder(-1, "WithoutLoss")
				//}
			}
			break
		case "sell":
			for i, level := range model.Conditions.ExitLevels {
				if model.State.ReachedTargetCount < i+1 && level.ActivatePrice == 0 {
					if level.Type == 1 && currentOHLCV.Close <= (model.State.EntryPrice*((100-level.Price/model.Conditions.Leverage)/100)) ||
						level.Type == 0 && currentOHLCV.Close <= level.Price {
						model.State.ReachedTargetCount += 1
						if level.Type == 0 {
							amount += level.Amount
						} else if level.Type == 1 {
							amount += model.Conditions.EntryOrder.Amount * (level.Amount / 100)
						}
					}
					//
					//if currentOHLCV.Close <= model.State.EntryPrice * (1 - model.Conditions.WithoutLossAfterProfit/model.Conditions.Leverage/100){
					//	sm.PlaceOrder(-1, "WithoutLoss")
					//}
				}
			}
			break
		}
		if model.State.ExecutedAmount == model.Conditions.EntryOrder.Amount {
			model.State.Amount = 0
			model.State.State = TakeProfit // took all profits, exit now
			sm.StateMgmt.UpdateState(model.ID, model.State)
			return true
		}
		if amount > 0 {
			model.State.Amount = amount
			model.State.State = TakeProfit
			currentState, _ := sm.State.State(context.Background())
			if currentState == TakeProfit {
				model.State.State = EnterNextTarget
				sm.StateMgmt.UpdateState(model.ID, model.State)
			}
			return true
		}
	}
	return false
}

func (sm *SmartOrder) checkLoss(ctx context.Context, args ...interface{}) bool {

	isWaitingForOrder, ok := sm.IsWaitingForOrder.Load(Stoploss)
	model := sm.Strategy.GetModel()
	existTimeout := model.Conditions.TimeoutWhenLoss != 0 || model.Conditions.TimeoutLoss != 0 || model.Conditions.ForcedLoss != 0
	isSpot := model.Conditions.MarketType == 0
	forcedSLWithAlert := model.Conditions.StopLossExternal && model.Conditions.MandatoryForcedLoss

	if ok && isWaitingForOrder.(bool) && !existTimeout && !model.Conditions.MandatoryForcedLoss {
		return false
	}
	if model.Conditions.StopLossExternal && !model.Conditions.MandatoryForcedLoss {
		return false
	}
	if model.State.ExecutedAmount >= model.Conditions.EntryOrder.Amount {
		return false
	}
	if len(sm.Strategy.GetModel().Conditions.EntryLevels) > 0 {
		return false
	}

	currentOHLCV := args[0].(interfaces.OHLCV)
	isTrailingHedgeOrder := model.Conditions.HedgeStrategyId != nil || model.Conditions.Hedging == true
	if isTrailingHedgeOrder {
		return false
	}
	stopLoss := model.Conditions.StopLoss / model.Conditions.Leverage
	forcedLoss := model.Conditions.ForcedLoss / model.Conditions.Leverage
	currentState := model.State.State
	stateFromStateMachine, _ := sm.State.State(ctx)

	// if we did not have time to go out to check existing orders (market)
	if currentState == End || stateFromStateMachine == End {
		return true
	}

	// after return from Stoploss state to InEntry we should change state in machine
	if stateFromStateMachine == Stoploss && model.State.StopLossAt == -1 && model.Conditions.TimeoutLoss > 0 {
		return true
	}

	// try exit on timeout
	if model.Conditions.TimeoutWhenLoss > 0 && !forcedSLWithAlert {
		isLoss := (model.Conditions.EntryOrder.Side == "buy" && model.State.EntryPrice > currentOHLCV.Close) || (model.Conditions.EntryOrder.Side == "sell" && model.State.EntryPrice < currentOHLCV.Close)
		if isLoss && model.State.LossableAt == 0 {
			model.State.LossableAt = time.Now().Unix()
			go func(lossAt int64) {
				time.Sleep(time.Duration(model.Conditions.TimeoutWhenLoss) * time.Second)
				stillTimeout := lossAt == model.State.LossableAt
				if stillTimeout {
					sm.PlaceOrder(-1, 0.0, Stoploss)
				}
			}(model.State.LossableAt)
		} else if !isLoss {
			model.State.LossableAt = 0
		}
		return false
	}

	switch model.Conditions.EntryOrder.Side {
	case "buy":
		if isSpot && forcedLoss > 0 && (1-currentOHLCV.Close/model.State.EntryPrice)*100 >= forcedLoss {
			sm.PlaceOrder(currentOHLCV.Close, 0.0, Stoploss)
			return false
		}

		if forcedSLWithAlert {
			return false
		}

		if (1-currentOHLCV.Close/model.State.EntryPrice)*100 >= stopLoss {
			if model.State.ExecutedAmount < model.Conditions.EntryOrder.Amount {
				model.State.Amount = model.Conditions.EntryOrder.Amount - model.State.ExecutedAmount
			}

			// if order go to StopLoss from InEntry state or returned to Stoploss while timeout
			if currentState == InEntry {
				model.State.State = Stoploss
				sm.StateMgmt.UpdateState(model.ID, model.State)

				if model.State.StopLossAt == 0 {
					return true
				}
			}
		} else {
			if currentState == Stoploss {
				model.State.State = InEntry
				sm.StateMgmt.UpdateState(model.ID, model.State)
			}
		}
		break
	case "sell":
		if isSpot && forcedLoss > 0 && (currentOHLCV.Close/model.State.EntryPrice-1)*100 >= forcedLoss {
			sm.PlaceOrder(currentOHLCV.Close, 0.0, Stoploss)
			return false
		}

		if forcedSLWithAlert {
			return false
		}

		if (currentOHLCV.Close/model.State.EntryPrice-1)*100 >= stopLoss {
			if model.State.ExecutedAmount < model.Conditions.EntryOrder.Amount {
				model.State.Amount = model.Conditions.EntryOrder.Amount - model.State.ExecutedAmount
			}

			if currentState == InEntry {
				model.State.State = Stoploss
				sm.StateMgmt.UpdateState(model.ID, model.State)

				if model.State.StopLossAt == 0 {
					return true
				}
			}
		} else {
			if currentState == Stoploss {
				model.State.State = InEntry
				sm.StateMgmt.UpdateState(model.ID, model.State)
			}
		}
		break
	}

	return false
}
func (sm *SmartOrder) enterEnd(ctx context.Context, args ...interface{}) error {
	sm.Strategy.GetModel().State.State = End
	sm.StateMgmt.UpdateState(sm.Strategy.GetModel().ID, sm.Strategy.GetModel().State)
	return nil
}

func (sm *SmartOrder) enterTakeProfit(ctx context.Context, args ...interface{}) error {
	if currentOHLCV, ok := args[0].(interfaces.OHLCV); ok {
		if sm.Strategy.GetModel().State.Amount > 0 {

			if sm.Lock {
				return nil
			}
			sm.Lock = true
			sm.PlaceOrder(currentOHLCV.Close, 0.0, TakeProfit)

			//if sm.Strategy.GetModel().State.ExecutedAmount - sm.Strategy.GetModel().Conditions.EntryOrder.Amount == 0 { // re-check all takeprofit conditions to exit trade ( no additional trades needed no )
			//	ohlcv := args[0].(OHLCV)
			//	err := sm.State.FireCtx(context.TODO(), TriggerTrade, ohlcv)
			//
			//	sm.Lock = false
			//	return err
			//}
		}
		sm.Lock = false
	}
	return nil
}

func (sm *SmartOrder) enterStopLoss(ctx context.Context, args ...interface{}) error {
	if currentOHLCV, ok := args[0].(interfaces.OHLCV); ok {
		if sm.Strategy.GetModel().State.Amount > 0 {
			side := "buy"
			if sm.Strategy.GetModel().Conditions.EntryOrder.Side == side {
				side = "sell"
			}
			if sm.Lock {
				return nil
			}
			sm.Lock = true
			if sm.Strategy.GetModel().Conditions.MarketType == 0 {
				if len(sm.Strategy.GetModel().State.Orders) < 2 {
					// if we go to stop loss once place TAP and didn't receive TAP id yet
					time.Sleep(3 * time.Second)
				}
				sm.TryCancelAllOrdersConsistently(sm.Strategy.GetModel().State.Orders)
			}
			// sm.cancelOpenOrders(sm.Strategy.GetModel().Conditions.Pair)
			defer sm.PlaceOrder(currentOHLCV.Close, 0.0, Stoploss)
		}
		// if timeout specified then do this sell on timeout
		sm.Lock = false
	}
	_ = sm.State.Fire(CheckLossTrade, args[0])
	return nil
}

func (sm *SmartOrder) SetSelectedExitTarget(selectedExitTarget int) {
	sm.SelectedExitTarget = selectedExitTarget
}

func (sm *SmartOrder) IsOrderExistsInMap(orderId string) bool {
	sm.OrdersMux.Lock()
	_, ok := sm.OrdersMap[orderId]
	sm.OrdersMux.Unlock()
	if ok {
		return true
	}

	return false
}

func (sm *SmartOrder) TryCancelAllOrdersConsistently(orderIds []string) {
	for _, orderId := range orderIds {
		if orderId != "0" {
			sm.ExchangeApi.CancelOrder(orders.CancelOrderRequest{
				KeyId: sm.KeyId,
				KeyParams: orders.CancelOrderRequestParams{
					OrderId:    orderId,
					MarketType: sm.Strategy.GetModel().Conditions.MarketType,
					Pair:       sm.Strategy.GetModel().Conditions.Pair,
				},
			})
		}
	}
}

func (sm *SmartOrder) TryCancelAllOrders(orderIds []string) {
	for _, orderId := range orderIds {
		if orderId != "0" {
			go sm.ExchangeApi.CancelOrder(orders.CancelOrderRequest{
				KeyId: sm.KeyId,
				KeyParams: orders.CancelOrderRequestParams{
					OrderId:    orderId,
					MarketType: sm.Strategy.GetModel().Conditions.MarketType,
					Pair:       sm.Strategy.GetModel().Conditions.Pair,
				},
			})
		}
	}
}

// Start continuously checks the state and runs another event loop cycle or stops the smart order if conditions met.
func (sm *SmartOrder) Start() {
	ctx := context.TODO()

	state, _ := sm.State.State(context.Background())
	localState := sm.Strategy.GetModel().State.State
	sm.Statsd.Inc("smart_order.start")
	var lastValidityCheckAt = time.Now().Add(-1 * time.Second)
	for state != End && localState != End && state != Canceled && state != Timeout {
		if time.Since(lastValidityCheckAt) > 2*time.Second { // TODO: remove magic number
			sm.Strategy.GetLogger().Debug("settlement mutex validity check")
			if valid, err := sm.Strategy.GetSettlementMutex().Valid(); !valid || err != nil {
				sm.Strategy.GetLogger().Error("invalid settlement mutex, breaking event loop",
					zap.Bool("mutex valid", valid),
					zap.Error(err),
				)
				break
			}
			lastValidityCheckAt = time.Now()
		}
		if sm.Strategy.GetModel().Enabled == false {
			state, _ = sm.State.State(ctx)
			break
		}
		if !sm.Lock {
			if sm.Strategy.GetModel().Conditions.EntrySpreadHunter && state != InEntry {
				sm.processSpreadEventLoop()
			} else {
				sm.processEventLoop()
			}
		}
		time.Sleep(60 * time.Millisecond)
		state, _ = sm.State.State(ctx)
		localState = sm.Strategy.GetModel().State.State
	}
	sm.Stop()
	sm.Strategy.GetLogger().Info("stopped smart order",
		zap.String("state", state.(string)),
	)
}

func (sm *SmartOrder) Stop() {
	model := sm.Strategy.GetModel()
	// use it to not execute this func twice
	if sm.StopLock {
		if model.Conditions.ContinueIfEnded == false {
			sm.StateMgmt.DisableStrategy(model.ID)
		}
		sm.Strategy.GetLogger().Info("cancel orders in stop at start")
		go sm.TryCancelAllOrders(model.State.Orders)
		return
	}

	sm.StopLock = true
	state, _ := sm.State.State(context.Background())

	// cancel orders
	if model.Conditions.MarketType == 0 && state != End {
		sm.Strategy.GetLogger().Info("cancel orders in stop for stop")
		sm.TryCancelAllOrdersConsistently(model.State.Orders)
	} else {
		sm.Strategy.GetLogger().Info("cancel orders for futures and step not End")
		go sm.TryCancelAllOrders(model.State.Orders)

		// handle case when order was creating while Stop func execution
		go func() {
			time.Sleep(5 * time.Second)
			go sm.TryCancelAllOrders(model.State.Orders)
		}()
	}

	// TODO: we should get rid of it, it should not work like this
	StateS := model.State.State

	if state != End && StateS != Timeout && sm.Strategy.GetModel().Conditions.EntrySpreadHunter == false {
		sm.Statsd.Inc("strategy_service.ended_with_not_end_status")
		sm.PlaceOrder(0, 0.0, Canceled)
	}

	if state != End && StateS != Timeout && !model.Conditions.EntrySpreadHunter {
		sm.Statsd.Inc("strategy_service.ended_with_not_end_status")
		sm.PlaceOrder(0, 0.0, Canceled)

		// we should check after some time if we have opened order and this one got executed before been canceled
		go func() {
			sm.Statsd.Inc("smart_order.place_cancel_order_in_stop_func_attempt")
			time.Sleep(5 * time.Second)
			if model.State.PositionAmount > 0 && model.State.EntryPrice > 0 {
				sm.Strategy.GetLogger().Warn("placing canceled order in stop for SM",
					zap.Float64("position amount", model.State.PositionAmount),
					zap.Float64("entry price", model.State.EntryPrice),
					zap.String("smart order state", state.(string)),
					zap.String("strategy state", StateS),
				)
				sm.Statsd.Inc("smart_order.place_cancel_order_in_stop_func_with_nonzero_amount")
				sm.PlaceOrder(0, model.State.PositionAmount, Canceled)
			}
		}()
	}

	if !model.Conditions.ContinueIfEnded {
		sm.StateMgmt.DisableStrategy(model.ID)
	}
	sm.StopLock = false

	// continue if ended option - go to new iteration
	if (StateS == Timeout || state == Timeout) &&
		model.Conditions.ContinueIfEnded && !model.Conditions.PositionWasClosed {
		sm.IsWaitingForOrder = sync.Map{}
		sm.IsEntryOrderPlaced = false
		sm.StateMgmt.EnableStrategy(model.ID)
		model.Enabled = true
		stateModel := model.State
		stateModel.State = WaitForEntry
		stateModel.EntryPrice = 0
		stateModel.ExecutedAmount = 0
		stateModel.Amount = 0
		stateModel.Orders = []string{}
		stateModel.Iteration += 1
		sm.StateMgmt.UpdateState(model.ID, stateModel)
		sm.StateMgmt.UpdateExecutedAmount(model.ID, stateModel)
		sm.StateMgmt.SaveStrategyConditions(model)
		_ = sm.State.Fire(Restart)
		//_ = sm.onStart(nil)
		sm.Start()
	}

	if amount := sm.Strategy.GetModel().State.PositionAmount; amount != 0.0 {
		sm.Strategy.GetLogger().Warn("stopped with non-zero amount",
			zap.Float64("amount left", amount),
		)
		sm.Statsd.Inc("smart_order.stopped_with_nonzero_amount")
	}
	sm.Statsd.Inc("smart_order.stop_attempt")
}

// processEventLoop takes new OHCLV data to supply it for the smart order state transition attempt.
func (sm *SmartOrder) processEventLoop() {
	currentOHLCVp := sm.DataFeed.GetPriceForPairAtExchange(sm.Strategy.GetModel().Conditions.Pair, sm.ExchangeName, sm.Strategy.GetModel().Conditions.MarketType)
	if currentOHLCVp != nil {
		currentOHLCV := *currentOHLCVp
		state, err := sm.State.State(context.TODO())
		err = sm.State.FireCtx(context.TODO(), TriggerTrade, currentOHLCV)
		if err == nil {
			return
		}
		if state == InEntry || state == TakeProfit || state == Stoploss || state == HedgeLoss {
			err = sm.State.FireCtx(context.TODO(), CheckLossTrade, currentOHLCV)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckProfitTrade, currentOHLCV)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckTrailingLossTrade, currentOHLCV)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckTrailingProfitTrade, currentOHLCV)
			if err == nil {
				return
			}
		}
		// log.Print(sm.Strategy.GetModel().Conditions.Pair, sm.Strategy.GetModel().State.TrailingEntryPrice, currentOHLCV.Close, err.Error())
	}
}

func (sm *SmartOrder) processSpreadEventLoop() {
	currentSpreadP := sm.DataFeed.GetSpreadForPairAtExchange(sm.Strategy.GetModel().Conditions.Pair, sm.ExchangeName, sm.Strategy.GetModel().Conditions.MarketType)
	if currentSpreadP != nil {
		currentSpread := *currentSpreadP
		ohlcv := interfaces.OHLCV{
			Close: currentSpread.BestBid,
		}
		state, err := sm.State.State(context.TODO())
		err = sm.State.FireCtx(context.TODO(), TriggerSpread, currentSpread)
		if err == nil {
			return
		}
		if state == InEntry || state == TakeProfit || state == Stoploss || state == HedgeLoss {
			err = sm.State.FireCtx(context.TODO(), CheckSpreadProfitTrade, currentSpread)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckLossTrade, ohlcv)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckProfitTrade, ohlcv)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckTrailingLossTrade, ohlcv)
			if err == nil {
				return
			}
			err = sm.State.FireCtx(context.TODO(), CheckTrailingProfitTrade, ohlcv)
			if err == nil {
				return
			}
		}
		// log.Print(sm.Strategy.GetModel().Conditions.Pair, sm.Strategy.GetModel().State.TrailingEntryPrice, currentOHLCV.Close, err.Error())
	}
}
