package models

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type MongoStrategyUpdateEvent struct {
	FullDocument MongoStrategy `json:"fullDocument" bson:"fullDocument"`
}

type MongoOrderUpdateEvent struct {
	FullDocument MongoOrder `json:"fullDocument" bson:"fullDocument"`
}

type MongoPositionUpdateEvent struct {
	FullDocument MongoPosition `json:"fullDocument" bson:"fullDocument"`
}

type MongoStrategyEvent struct {
	T    int64
	Data interface{}
}

type MongoT struct {
	T    int64
	Data interface{}
}

type RBAC struct {
	Id          primitive.ObjectID `json:"_id"`
	userId      string
	portfolioId string
	accessLevel int32
}

type MongoSocial struct {
	SharedWith []primitive.ObjectID `bson:"sharedWith"` // [RBAC]
	IsPrivate  bool
}

type MongoMarketDefaultProperties struct {
	PricePrecision    int64 `json:"pricePrecision" bson:"pricePrecision"`
	QuantityPrecision int64 `json:"quantityPrecision" bson:"quantityPrecision"`
}

type MongoMarketProperties struct {
	Binance MongoMarketDefaultProperties `json:"binance" bson:"binance"`
}

type MongoMarket struct {
	ID         *primitive.ObjectID   `json:"_id" bson:"_id"`
	Name       string                `json:"name" bson:"name"`
	MarketType int                   `json:"marketType" bson:"marketType"`
	Symbol     string                `json:"symbol" bson:"symbol"`
	BaseId     *primitive.ObjectID   `json:"baseId" bson:"baseId"`
	QuoteId    *primitive.ObjectID   `json:"quoteId" bson:"quoteId"`
	Properties MongoMarketProperties `json:"properties" bson:"properties"`
}

// MarketTypeString returns market type string like "spot" or "futures".
func (mm MongoMarket) MarketTypeString() (string, error) {
	switch mm.MarketType {
	case 0:
		return "spot", nil
	case 1:
		return "futures", nil
	default:
		return "", fmt.Errorf("unknown market type %v", mm.MarketType)
	}
}

// A MongoOrderFee represents optional field of order describes fees.
type MongoOrderFee struct {
	Cost     *string `json:"cost" bson:"cost"`
	Currency *string `json:"currency" bson:"currency"`
}

type MongoOrder struct {
	ID                     primitive.ObjectID `json:"_id" bson:"_id"`
	Status                 string             `json:"status,omitempty" bson:"status"`
	PositionSide           string             `json:"positionSide,omitempty" bson:"positionSide"`
	OrderId                string             `json:"id,omitempty" bson:"id"`
	PostOnlyFinalOrderId   string             `json:"postOnlyFinalOrderId,omitempty" bson:"postOnlyFinalOrderId"`
	PostOnlyInitialOrderId string             `json:"postOnlyInitialOrderId,omitempty" bson:"postOnlyInitialOrderId"`
	Filled                 float64            `json:"filled,omitempty" bson:"filled"`
	Amount                 float64            `json:"amount,omitempty" bson:"amount"`
	Fee                    MongoOrderFee      `json:"fee" bson:"fee"`
	Average                float64            `json:"average,omitempty" bson:"average"`
	Side                   string             `json:"side,omitempty" bson:"side"`
	Type                   string             `json:"type,omitempty" bson:"type"`
	Symbol                 string             `json:"symbol,omitempty" bson:"symbol"`
	ReduceOnly             bool               `json:"reduceOnly,omitempty" bson:"reduceOnly"`
	Price                  float64            `json:"price,omitempty" bson:"price"`
	StopPrice              float64            `json:"stopPrice,omitempty" bson:"stopPrice"`
	Timestamp              float64            `json:"timestamp,omitempty" bson:"timestamp"`
	UpdatedAt              time.Time          `json:"updatedAt,omitempty" bson:"updatedAt"`
}

type MongoPosition struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	KeyId       primitive.ObjectID `json:"keyId" bson:"keyId"`
	EntryPrice  float64            `json:"entryPrice,omitempty" bson:"entryPrice"`
	MarkPrice   float64            `json:"markPrice,omitempty" bson:"markPrice"`
	PositionAmt float64            `json:"positionAmt,omitempty" bson:"positionAmt"`
	Symbol      string             `json:"symbol,omitempty" bson:"symbol"`
	Leverage    float64            `json:"leverage,omitempty" bson:"leverage"`
	UpdatedAt   time.Time          `json:"updatedAt,omitempty" bson:"updatedAt"`
}

// A MongoStrategy is the root of a smart trade strategy description.
type MongoStrategy struct {
	ID              *primitive.ObjectID     `json:"_id" bson:"_id"`             // strategy unique identity
	Type            int64                   `json:"type,omitempty" bson:"type"` // 1 - smart order, 2 - maker only
	Enabled         bool                    `json:"enabled,omitempty" bson:"enabled"`
	AccountId       *primitive.ObjectID     `json:"accountId,omitempty" bson:"accountId"`
	Conditions      *MongoStrategyCondition `json:"conditions,omitempty" bson:"conditions"`
	State           *MongoStrategyState     `bson:"state,omitempty"`
	TriggerWhen     TriggerOptions          `bson:"triggerWhen,omitempty"`
	Expiration      ExpirationSchema
	LastUpdate      int64
	SignalIds       []primitive.ObjectID
	OrderIds        []primitive.ObjectID `bson:"orderIds,omitempty"`
	WaitForOrderIds []primitive.ObjectID `bson:"waitForOrderIds,omitempty"`
	OwnerId         primitive.ObjectID
	Social          MongoSocial `bson:"social"` // {sharedWith: [RBAC]}
	CreatedAt       time.Time   `json:"createdAt,omitempty" bson:"createdAt"`
}

type MongoStrategyType struct {
	SigType  string `json:"type"`
	Required interface{}
}

// A MongoStrategyState is a set of dynamic parameters for a smart trade.
type MongoStrategyState struct {
	ColdStart    bool   `json:"coldStart,omitempty" bson:"coldStart"`
	State        string `json:"state,omitempty" bson:"state"`
	Msg          string `json:"msg,omitempty" bson:"msg"`
	EntryOrderId string `json:"entryOrderId,omitempty" bson:"entryOrderId"`
	Iteration    int    `json:"iteration,omitempty" bson:"iteration"`
	// we save params to understand which was changed

	EntryPointPrice     float64 `json:"entryPointPrice,omitempty" bson:"entryPointPrice"`
	EntryPointType      string  `json:"entryPointType,omitempty" bson:"entryPointType"`
	EntryPointSide      string  `json:"entryPointSide,omitempty" bson:"entryPointSide"`
	EntryPointAmount    float64 `json:"entryPointAmount,omitempty" bson:"entryPointAmount"`
	EntryPointDeviation float64 `json:"entryPointDeviation,omitempty" bson:"entryPointDeviation"`
	// Orders created by strategy to entry.
	WaitForEntryIds []string `json:"waitForEntryIds,omitempty" bson:"waitForEntryIds"`
	StopLoss        float64  `json:"stopLoss,omitempty" bson:"stopLoss"`
	StopLossPrice   float64  `json:"stopLossPrice, omitempty" bson:"stopLossPrice"`
	// Orders created by strategy for regular stop-loss.
	StopLossOrderIds []string `json:"stopLossOrderIds,omitempty" bson:"stopLossOrderIds"`
	ForcedLoss       float64  `json:"forcedLoss,omitempty" bson:"forcedLoss"`
	ForcedLossPrice  float64  `json:"forcedLossPrice, omitempty" bson:"forcedLossPrice"`
	// Orders created by strategy for forced stop-loss.
	ForcedLossOrderIds   []string           `json:"forcedLossOrderIds,omitempty" bson:"forcedLossOrderIds"`
	TakeProfit           []*MongoEntryPoint `json:"takeProfit,omitempty" bson:"takeProfit"`
	TakeProfitPrice      float64            `json:"takeProfitPrice, omitempty" bson:"takeProfitPrice"`
	TakeProfitHedgePrice float64            `json:"takeProfitHedgePrice,omitempty" bson:"takeProfitHedgePrice"`
	// Orders created by strategy for take a profit.
	TakeProfitOrderIds []string `json:"takeProfitOrderIds,omitempty" bson:"takeProfitOrderIds"`
	// An accumulated part of amount paid for fees
	Commission float64 `json:"commission" bson:"commission"`

	TrailingEntryPrice     float64 `json:"trailingEntryPrice,omitempty" bson:"trailingEntryPrice"`
	HedgeExitPrice         float64 `json:"hedgeExitPrice,omitempty" bson:"hedgeExitPrice"`
	TrailingHedgeExitPrice float64 `json:"trailingHedgeExitPrice,omitempty" bson:"trailingHedgeExitPrice"`

	TrailingExitPrice  float64   `json:"trailingExitPrice,omitempty" bson:"trailingExitPrice"`
	TrailingExitPrices []float64 `json:"trailingExitPrices,omitempty" bson:"trailingExitPrices"`
	EntryPrice         float64   `json:"entryPrice,omitempty" bson:"entryPrice"`
	SavedEntryPrice    float64   `json:"savedEntryPrice,omitempty" bson:"savedEntryPrice"`
	ExitPrice          float64   `json:"exitPrice,omitempty" bson:"exitPrice"`
	Amount             float64   `json:"amount,omitempty" bson:"amount"`
	Orders             []string  `json:"orders,omitempty" bson:"orders"`
	ExecutedOrders     []string  `json:"executedOrders,omitempty" bson:"executedOrders"`
	ExecutedAmount     float64   `json:"executedAmount,omitempty" bson:"executedAmount"`
	ReachedTargetCount int       `json:"reachedTargetCount,omitempty" bson:"reachedTargetCount"`

	TrailingCheckAt            int64 `json:"trailingCheckAt,omitempty" bson:"trailingCheckAt"`
	StopLossAt                 int64 `json:"stopLossAt,omitempty" bson:"stopLossAt"`
	LossableAt                 int64 `json:"lossableAt,omitempty" bson:"lossableAt"`
	ProfitableAt               int64 `json:"profitableAt,omitempty" bson:"profitableAt"`
	ProfitAt                   int64 `json:"profitAt,omitempty" bson:"profitAt"`
	CloseStrategyAfterFirstTAP bool  `json:"closeStrategyAfterFirstTAP,omitempty" bson:"closeStrategyAfterFirstTAP"`

	PositionAmount           float64 `json:"positionAmount,omitempty" bson:"positionAmount"`
	ReceivedProfitAmount     float64 `json:"receivedProfitAmount,omitempty" bson:"receivedProfitAmount"`
	ReceivedProfitPercentage float64 `json:"receivedProfitPercentage,omitempty" bson:"receivedProfitPercentage"`
}

type MongoEntryPoint struct {
	ActivatePrice           float64 `json:"activatePrice,omitempty" bson:"activatePrice"`
	EntryDeviation          float64 `json:"entryDeviation,omitempty" bson:"entryDeviation"`
	Price                   float64 `json:"price,omitempty" bson:"price"`
	Side                    string  `json:"side,omitempty" bson:"side"`
	ReduceOnly              bool    `json:"reduceOnly,omitempty" bson:"reduceOnly"`
	Amount                  float64 `json:"amount,omitempty" bson:"amount"`
	HedgeEntry              float64 `json:"hedgeEntry,omitempty" bson:"hedgeEntry"`
	HedgeActivation         float64 `json:"hedgeActivation,omitempty" bson:"hedgeActivation"`
	HedgeOppositeActivation float64 `json:"hedgeOppositeActivation,omitempty" bson:"hedgeOppositeActivation"`
	PlaceWithoutLoss        bool    `json:"placeWithoutLoss,omitempty" bson:"placeWithoutLoss"`
	// Type: 0 means absolute price, 1 means price is relative to entry price
	Type      int64  `json:"type,omitempty" bson:"type"`
	OrderType string `json:"orderType,omitempty" bson:"orderType"`
}

// A MongoStrategyCondition is a set of static (persistent) parameters for a smart trade.
type MongoStrategyCondition struct {
	Exchange string               `json:"exchange,omitempty" bson:"exchange"`
	AccountId *primitive.ObjectID `json:"accountId,omitempty" bson:"accountId"`

	Hedging         bool                `json:"hedging,omitempty" bson:"hedging"`
	HedgeMode       bool                `json:"hedgeMode,omitempty" bson:"hedgeMode"`
	HedgeKeyId      *primitive.ObjectID `json:"hedgeKeyId,omitempty" bson:"hedgeKeyId"`
	HedgeStrategyId *primitive.ObjectID `json:"hedgeStrategyId,omitempty" bson:"hedgeStrategyId"`

	MakerOrderId *primitive.ObjectID `json:"makerOrderId,omitempty" bson:"makerOrderId"`

	TemplateToken          string              `json:"templateToken,omitempty" bson:"templateToken"`
	MandatoryForcedLoss    bool                `json:"mandatoryForcedLoss,omitempty" bson:"mandatoryForcedLoss"`
	PositionWasClosed      bool                `json:"positionWasClosed, omitempty" bson:"positionWasClosed"`
	SkipInitialSetup       bool                `json:"skipInitialSetup, omitempty" bson:"skipInitialSetup"`
	CancelIfAnyActive      bool                `json:"cancelIfAnyActive,omitempty" bson:"cancelIfAnyActive"`
	TrailingExitExternal   bool                `json:"trailingExitExternal,omitempty" bson:"trailingExitExternal"`
	TrailingExitPrice      float64             `json:"trailingExitPrice,omitempty" bson:"trailingExitPrice"`
	TrailingExit           bool                `json:"trailingExit,omitempty" bson:"trailingExit"`
	StopLossPrice          float64             `json:"stopLossPrice,omitempty" bson:"stopLossPrice"`
	ForcedLossPrice        float64             `json:"forcedLossPrice,omitempty" bson:"forcedLossPrice"`
	TakeProfitPrice        float64             `json:"takeProfitPrice,omitempty" bson:"takeProfitPrice"`
	TakeProfitHedgePrice   float64             `json:"takeProfitHedgePrice,omitempty" bson:"takeProfitHedgePrice"`
	StopLossExternal       bool                `json:"stopLossExternal,omitempty" bson:"stopLossExternal"`
	TakeProfitExternal     bool                `json:"takeProfitExternal,omitempty" bson:"takeProfitExternal"`
	WithoutLossAfterProfit float64             `json:"withoutLossAfterProfit,omitempty" bson:"withoutLossAfterProfit"`
	EntrySpreadHunter      bool                `json:"entrySpreadHunter,omitempty" bson:"entrySpreadHunter"`
	EntryWaitingTime       int64               `json:"entryWaitingTime,omitempty" bson:"entryWaitingTime"`
	TakeProfitSpreadHunter bool                `json:"takeProfitSpreadHunter,omitempty" bson:"takeProfitSpreadHunter"`
	TakeProfitWaitingTime  int64               `json:"takeProfitWaitingTime,omitempty" bson:"takeProfitWaitingTime"`
	KeyAssetId             *primitive.ObjectID `json:"keyAssetId,omitempty" bson:"keyAssetId"`
	Pair                   string              `json:"pair,omitempty" bson:"pair"`
	MarketType             int64               `json:"marketType,omitempty" bson:"marketType"`
	EntryOrder             *MongoEntryPoint    `json:"entryOrder,omitempty" bson:"entryOrder"`

	WaitingEntryTimeout   float64 `json:"waitingEntryTimeout,omitempty" bson:"waitingEntryTimeout"`
	ActivationMoveStep    float64 `json:"activationMoveStep,omitempty" bson:"activationMoveStep"`
	ActivationMoveTimeout float64 `json:"activationMoveTimeout,omitempty" bson:"activationMoveTimeout"`

	TimeoutIfProfitable float64 `json:"timeoutIfProfitable,omitempty" bson:"timeoutIfProfitable"`
	// then take profit after some time
	TimeoutWhenProfit float64 `json:"timeoutWhenProfit,omitempty" bson:"timeoutWhenProfit"` // if position became profitable at takeProfit,
	// then dont exit but wait N seconds and exit, so you may catch pump

	ContinueIfEnded           bool    `json:"continueIfEnded,omitempty" bson:"continueIfEnded"`                     // open opposite position, or place buy if sold, or sell if bought // , if entrypoints specified, trading will be within entrypoints, if not exit on takeProfit or timeout or stoploss
	TimeoutBeforeOpenPosition float64 `json:"timeoutBeforeOpenPosition,omitempty" bson:"timeoutBeforeOpenPosition"` // wait after closing position before opening new one
	ChangeTrendIfLoss         bool    `json:"changeTrendIfLoss,omitempty" bson:"changeTrendIfLoss"`
	ChangeTrendIfProfit       bool    `json:"changeTrendIfProfit,omitempty" bson:"changeTrendIfProfit"`

	MoveStopCloser        bool    `json:"moveStopClose,omitempty" bson:"moveStopCloser"`
	MoveForcedStopAtEntry bool    `json:"moveForcedStopAtEntry,omitempty" bson:"moveForcedStopAtEntry"`
	TimeoutWhenLoss       float64 `json:"timeoutWhenLoss,omitempty" bson:"timeoutWhenLoss"` // wait after hit SL and gives it a chance to grow back
	TimeoutLoss           float64 `json:"timeoutLoss,omitempty" bson:"timeoutLoss"`         // if ROE negative it counts down and if still negative then exit
	StopLoss              float64 `json:"stopLoss,omitempty" bson:"stopLoss"`
	StopLossType          string  `json:"stopLossType,omitempty" bson:"stopLossType"`
	ForcedLoss            float64 `json:"forcedLoss,omitempty" bson:"forcedLoss"`
	HedgeLossDeviation    float64 `json:"hedgeLossDeviation,omitempty" bson:"hedgeLossDeviation"`

	CreatedByTemplate  bool                `json:"createdByTemplate,omitempty" bson:"createdByTemplate"`
	TemplateStrategyId *primitive.ObjectID `json:"templateStrategyId,omitempty" bson:"templateStrategyId"`

	Leverage                   float64            `json:"leverage,omitempty" bson:"leverage"`
	EntryLevels                []*MongoEntryPoint `json:"entryLevels,omitempty" bson:"entryLevels"`
	ExitLevels                 []*MongoEntryPoint `json:"exitLevels,omitempty" bson:"exitLevels"`
	CloseStrategyAfterFirstTAP bool               `json:"closeStrategyAfterFirstTAP,omitempty" bson:"closeStrategyAfterFirstTAP"`
	PlaceEntryAfterTAP         bool               `json:"placeEntryAfterTAP,omitempty" bson:"placeEntryAfterTAP"`
}
