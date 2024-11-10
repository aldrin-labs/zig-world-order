package smart_order

import (
	"gitlab.com/crypto_project/core/strategy_service/src/sources/mongodb/models"
	//"gitlab.com/crypto_project/core/strategy_service/src/trading"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// returns conditions of smart order depending on the scenario
func GetTestSmartOrderStrategy(scenario string) models.MongoStrategy {
	smartOrder := models.MongoStrategy{
		// ID:          &primitive.ObjectID{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x61, 0x62},
		ID:          &primitive.ObjectID{},
		Conditions:  &models.MongoStrategyCondition{},
		State:       &models.MongoStrategyState{Amount: 1000},
		TriggerWhen: models.TriggerOptions{},
		Expiration:  models.ExpirationSchema{},
		OwnerId:     primitive.ObjectID{},
		Social:      models.MongoSocial{},
		Enabled:     true,
	}

	switch scenario {
	case "entryLong":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				Price:     7000,
				Amount:    0.001,
				OrderType: "limit",
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "entryShort":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "sell",
				Price:     7000,
				OrderType: "limit",
				Amount:    0.001,
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "entrySpread":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				Price:     7000,
				Amount:    0.01,
				OrderType: "limit",
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
			EntrySpreadHunter: true,
			EntryWaitingTime:  1000,
		}
	case "entryLongTimeout":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.002,
			},
			WaitingEntryTimeout: 2,
			ContinueIfEnded:     true,
			EntrySpreadHunter:   true,
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     0.00001,
					Amount:    100,
				},
			},
		}
	case "entryLongTimeout2":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.0015,
			},
			WaitingEntryTimeout: 2,
			ContinueIfEnded:     true,
			EntrySpreadHunter:   true,
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     0.00001,
					Amount:    100,
				},
			},
		}
	case "mandatoryForcedLoss":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:       "BTC_USDT",
			MarketType: 1,
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "market",
				Amount:    0.0015,
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     0.00001,
					Amount:    100,
				},
			},
			StopLossExternal:    true,
			TakeProfitExternal:  true,
			MandatoryForcedLoss: true,
			ForcedLoss:          10,
			StopLoss:            20,
		}
	case "multiEntryPlacing":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:             "BTC_USDT",
			MarketType:       1,
			Leverage:         125,
			SkipInitialSetup: true,
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.03,
			},
			EntryLevels: []*models.MongoEntryPoint{
				{
					Price:  6000,
					Amount: 0.01,
					Type:   0,
				},
				{
					Price:  20,
					Amount: 33,
					Type:   1,
				},
				{
					Price:  20,
					Amount: 33,
					Type:   1,
				},
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     20,
					Amount:    100,
				},
			},
			ForcedLoss:   10,
			StopLoss:     20,
			StopLossType: "market",
		}
	case "multiEntryPlacingTAP":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:             "BTC_USDT",
			MarketType:       1,
			Leverage:         125,
			SkipInitialSetup: true,
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.03,
			},
			EntryLevels: []*models.MongoEntryPoint{
				{
					Price:  6000,
					Amount: 0.01,
					Type:   0,
				},
				{
					Price:  20,
					Amount: 33,
					Type:   1,
				},
				{
					Price:  20,
					Amount: 33,
					Type:   1,
				},
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     20,
					Amount:    100,
				},
			},
			StopLoss:     15,
			StopLossType: "market",
		}
	case "multiEntryPlacingClosingAfterFirstTAP":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:             "BTC_USDT",
			MarketType:       1,
			Leverage:         125,
			SkipInitialSetup: true,
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.003,
			},
			EntryLevels: []*models.MongoEntryPoint{
				{
					Price:            5700,
					Amount:           0.001,
					Type:             0,
					PlaceWithoutLoss: true,
				},
				{
					Price:  20,
					Amount: 0.001,
					Type:   1,
				},
				{
					Price:  20,
					Amount: 0.001,
					Type:   1,
				},
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     2,
					Amount:    100,
				},
			},
			StopLoss:                   2000,
			StopLossType:               "market",
			CloseStrategyAfterFirstTAP: true,
		}
	case "multiEntryPlacingClosingByWithoutLoss":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:             "BTC_USDT",
			MarketType:       1,
			Leverage:         125,
			SkipInitialSetup: true,
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				OrderType: "limit",
				Amount:    0.003,
			},
			EntryLevels: []*models.MongoEntryPoint{
				{
					Price:            5700,
					Amount:           0.001,
					Type:             0,
					PlaceWithoutLoss: true,
				},
				{
					Price:  20,
					Amount: 0.001,
					Type:   1,
				},
				{
					Price:  20,
					Amount: 0.001,
					Type:   1,
				},
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     20,
					Amount:    100,
				},
			},
			StopLoss:                   2000,
			StopLossType:               "market",
			CloseStrategyAfterFirstTAP: true,
		}
	case "trailingEntryLong":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:           "buy",
				ActivatePrice:  7000,
				EntryDeviation: 1,
				OrderType:      "limit",
				Amount:         0.001,
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "trailingEntryShort":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:           "sell",
				ActivatePrice:  7000,
				EntryDeviation: 1,
				OrderType:      "limit",
				Amount:         0.001,
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "stopLossMarket":
		smartOrder.State = &models.MongoStrategyState{
			State:              "InEntry",
			EntryOrderId:       "",
			TakeProfitOrderIds: make([]string, 0),
			StopLossOrderIds:   make([]string, 0),
			StopLoss:           5,
			TrailingEntryPrice: 0,
			TrailingExitPrices: nil,
			EntryPrice:         7000,
			ExitPrice:          0,
			Amount:             0.05,
			Orders:             nil,
			ExecutedOrders:     nil,
			ExecutedAmount:     0,
			ReachedTargetCount: 0,
			StopLossAt:         0,
			LossableAt:         0,
			ProfitableAt:       0,
			ProfitAt:           0,
		}
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:         "BTC_USDT",
			EntryOrder:   &models.MongoEntryPoint{Side: "buy", ActivatePrice: 7000, Amount: 0.05, OrderType: "limit", EntryDeviation: 1},
			StopLoss:     5,
			Leverage:     1,
			StopLossType: "market",
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "market",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "stopLossMarketTimeout":
		smartOrder.State = &models.MongoStrategyState{
			State:      "InEntry",
			EntryPrice: 7000,
			Amount:     0.05,
		}
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:            "BTC_USDT",
			EntryOrder:      &models.MongoEntryPoint{Side: "buy", OrderType: "limit", Price: 7000, Amount: 0.05},
			TimeoutWhenLoss: 5,
			//TimeoutLoss: 100,
			StopLoss:     5,
			StopLossType: "market",
			Leverage:     1,
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "stopLossMarketTimeoutLoss":
		smartOrder.State = &models.MongoStrategyState{
			State:      "InEntry",
			EntryPrice: 7000,
			Amount:     0.05,
		}
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:       "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{Side: "buy", OrderType: "limit", ActivatePrice: 7000, EntryDeviation: 1, Amount: 0.05},
			//TimeoutWhenLoss: 5,
			MarketType:   1,
			TimeoutLoss:  5,
			StopLoss:     2,
			StopLossType: "limit",
			Leverage:     1,
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
		}
	case "takeProfitSpread":
		smartOrder.State = &models.MongoStrategyState{
			State:      "InEntry",
			EntryPrice: 7000,
			Amount:     0.05,
		}
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				Price:     7000,
				Amount:    0.001,
				OrderType: "limit",
			},
			ExitLevels: []*models.MongoEntryPoint{
				{
					Type:      1,
					OrderType: "limit",
					Price:     10,
					Amount:    100,
				},
			},
			EntrySpreadHunter:      true,
			TakeProfitSpreadHunter: true,
			TakeProfitWaitingTime:  1000,
		}
	case "TakeProfitMarket":
		smartOrder.State = &models.MongoStrategyState{
			State:      "InEntry",
			EntryPrice: 7000,
			Amount:     0.05,
		}
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:       "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{Side: "buy", OrderType: "limit", Price: 6999, Amount: 0.05},
			ExitLevels: []*models.MongoEntryPoint{
				{Price: 7050, Amount: 0.05, Type: 0, OrderType: "market"},
			},
			Leverage: 1,
		}
	case "takeProfit":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:         "BTC_USDT",
			Leverage:     100,
			StopLossType: "market",
			StopLoss:     5,
			EntryOrder: &models.MongoEntryPoint{
				Side:           "buy",
				ActivatePrice:  6950,
				Amount:         0.05,
				EntryDeviation: 3,
				OrderType:      "market",
			},
			ExitLevels: []*models.MongoEntryPoint{{
				OrderType: "market",
				Type:      1,
				Price:     5,
				Amount:    100,
			}},
		}
	case "trailingEntryExitLeverage":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:         "BTC_USDT",
			MarketType:   1,
			Leverage:     100,
			StopLossType: "market",
			StopLoss:     10,
			EntryOrder: &models.MongoEntryPoint{
				Side:           "buy",
				ActivatePrice:  6950,
				Amount:         0.05,
				EntryDeviation: 3,
				OrderType:      "market",
			},
			ExitLevels: []*models.MongoEntryPoint{{
				OrderType:      "market",
				Type:           1,
				ActivatePrice:  5,
				EntryDeviation: 3,
				Amount:         100,
			}},
		}
	case "marketEntryTrailingExitLeverage":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:         "BTC_USDT",
			Leverage:     100,
			StopLoss:     10,
			StopLossType: "limit",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				Amount:    0.05,
				OrderType: "market",
			},
			ExitLevels: []*models.MongoEntryPoint{{
				OrderType:      "limit",
				Type:           1,
				ActivatePrice:  15,
				EntryDeviation: 10,
				Amount:         100,
			}},
		}
	case "stopLossMultiTargets":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair:       "BTC_USDT",
			MarketType: 1,
			EntryOrder: &models.MongoEntryPoint{Side: "buy", Price: 6999, Amount: 0.15, OrderType: "market"},
			ExitLevels: []*models.MongoEntryPoint{
				{Price: 10, Amount: 50, Type: 1, OrderType: "limit"},
				{Price: 15, Amount: 25, Type: 1, OrderType: "limit"},
				{Price: 20, Amount: 25, Type: 1, OrderType: "limit"},
			},
			Leverage:     20,
			StopLossType: "limit",
			StopLoss:     10,
		}
	case "multiplePriceTargets":
		smartOrder.Conditions = &models.MongoStrategyCondition{
			Pair: "BTC_USDT",
			EntryOrder: &models.MongoEntryPoint{
				Side:      "buy",
				Price:     7000,
				Amount:    0.01,
				OrderType: "limit",
			},
			ExitLevels: []*models.MongoEntryPoint{
				{Price: 10, Amount: 50, Type: 1, OrderType: "limit"},
				{Price: 15, Amount: 25, Type: 1, OrderType: "limit"},
				{Price: 20, Amount: 25, Type: 1, OrderType: "limit"},
			},
		}
	}

	return smartOrder
}
