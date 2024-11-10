package strategies

import (
	"context"
	"log"

	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"gitlab.com/crypto_project/core/strategy_service/src/service/strategies/makeronly_order"
	"gitlab.com/crypto_project/core/strategy_service/src/sources/mongodb"
	"gitlab.com/crypto_project/core/strategy_service/src/sources/mongodb/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RunMakerOnlyOrder(strategy *Strategy, df interfaces.IDataFeed, td interfaces.ITrading, keyId *primitive.ObjectID) interfaces.IStrategyRuntime {
	if strategy.Model.Conditions.Leverage == 0 {
		strategy.Model.Conditions.Leverage = 1
	}
	if strategy.Model.Conditions.MarketType == 0 {
		strategy.Model.Conditions.Leverage = 1
	}
	if keyId == nil {
		KeyAssets := mongodb.GetCollection("core_key_assets") // TODO: move to statemgmt, avoid any direct dependecies here
		keyAssetId := strategy.Model.Conditions.KeyAssetId.String()
		var request bson.D
		request = bson.D{
			{"_id", strategy.Model.Conditions.KeyAssetId},
		}
		log.Print(keyAssetId)
		ctx := context.Background()
		var keyAsset KeyAsset
		err := KeyAssets.FindOne(ctx, request).Decode(&keyAsset)
		if err != nil {
			log.Print("keyAssetsCursor", err.Error())
		}
		keyId = &keyAsset.KeyId
	}
	//if strategy.Model.Conditions.MarketType == 1 && !strategy.Model.Conditions.SkipInitialSetup {
	//	go td.UpdateLeverage(keyId, strategy.Model.Conditions.Leverage, strategy.Model.Conditions.Pair)
	//}
	if strategy.Model.State == nil {
		strategy.Model.State = &models.MongoStrategyState{}
	}
	// go strategy.StateMgmt.SaveStrategy(strategy.Model)
	// strategy.StateMgmt.SaveStrategyConditions(strategy.Model)
	runtime := makeronly_order.NewMakerOnlyOrder(strategy, df, td, keyId, strategy.StateMgmt)
	go runtime.Start()

	return runtime
}
