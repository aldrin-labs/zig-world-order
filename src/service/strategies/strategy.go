package strategies

import (
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"gitlab.com/crypto_project/core/strategy_service/src/logging"
	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"gitlab.com/crypto_project/core/strategy_service/src/sources/mongodb/models"
	"gitlab.com/crypto_project/core/strategy_service/src/sources/redis"
	statsd_client "gitlab.com/crypto_project/core/strategy_service/src/statsd"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"time"
)

// GetStrategy instantiates strategy with resources created and given.
func GetStrategy(cur *mongo.Cursor, df interfaces.IDataFeed, tr interfaces.ITrading, sm interfaces.IStateMgmt, createOrder interfaces.ICreateRequest, sd *statsd_client.StatsdClient) (*Strategy, error) {
	var model models.MongoStrategy
	err := cur.Decode(&model)
	rs := redis.GetRedsync()
	mutexName := fmt.Sprintf("strategy:%v:%v:%v", model.Conditions.MarketType, model.Conditions.Pair,
		model.ID.Hex())
	mutex := rs.NewMutex(mutexName,
		redsync.WithTries(2),
		redsync.WithRetryDelay(1*time.Second),
		redsync.WithExpiry(10*time.Second), // TODO(khassanov): use parameter to conform with extend call period
	) // upsert
	logger, _ := logging.GetZapLogger()
	loggerName := fmt.Sprintf("sm-%v", model.ID.Hex())
	logger = logger.With(zap.String("logger", loggerName))
	return &Strategy{
		Model:           &model,
		SettlementMutex: mutex,
		Datafeed:        df,
		Trading:         tr,
		StateMgmt:       sm,
		Statsd:          sd,
		Singleton:       createOrder,
		Log:             logger,
	}, err
}

// A Strategy describes user defined rules to order trades and what are interfaces to execute these rules.
type Strategy struct {
	Model           *models.MongoStrategy
	StrategyRuntime interfaces.IStrategyRuntime
	SettlementMutex *redsync.Mutex
	Datafeed        interfaces.IDataFeed
	Trading         interfaces.ITrading
	StateMgmt       interfaces.IStateMgmt
	Statsd          interfaces.IStatsClient
	Singleton       interfaces.ICreateRequest
	Log             interfaces.ILogger
}

func (strategy *Strategy) GetModel() *models.MongoStrategy {
	return strategy.Model
}

func (strategy *Strategy) GetSingleton() interfaces.ICreateRequest {
	return strategy.Singleton
}

func (strategy *Strategy) GetRuntime() interfaces.IStrategyRuntime {
	return strategy.StrategyRuntime
}

func (strategy *Strategy) GetSettlementMutex() *redsync.Mutex {
	return strategy.SettlementMutex
}

func (strategy *Strategy) GetDatafeed() interfaces.IDataFeed {
	return strategy.Datafeed
}

func (strategy *Strategy) GetTrading() interfaces.ITrading {
	return strategy.Trading
}

func (strategy *Strategy) GetStateMgmt() interfaces.IStateMgmt {
	return strategy.StateMgmt
}

func (strategy *Strategy) GetStatsd() interfaces.IStatsClient {
	return strategy.Statsd
}

func (strategy *Strategy) GetLogger() interfaces.ILogger {
	return strategy.Log
}

// ID returns unique identifier the strategy holds.
func (strategy *Strategy) ID() string {
	return fmt.Sprintf("%q", strategy.Model.ID.Hex())
}

// Start starts a runtime for the strategy.
func (strategy *Strategy) Start() {
	switch strategy.Model.Type {
	case 1:
		strategy.Log.Info("running smart order",
			zap.String("id", strategy.ID()),
			zap.Int64("type", strategy.Model.Type),
		)
		strategy.StrategyRuntime = RunSmartOrder(strategy, strategy.Datafeed, strategy.Trading, strategy.Statsd, strategy.Model.AccountId)
		strategy.Statsd.Inc("smart_order.runtime_start")
	case 2:
		strategy.Log.Info("running maker only order",
			zap.String("id", strategy.ID()),
			zap.Int64("type", strategy.Model.Type),
		)
		strategy.StrategyRuntime = RunMakerOnlyOrder(strategy, strategy.Datafeed, strategy.Trading, strategy.Model.AccountId)
	default:
		strategy.Log.Warn("strategy type not supported",
			zap.String("id", strategy.ID()),
			zap.Int64("type", strategy.Model.Type),
		)
	}
}

// HotReload updates strategy in runtime to keep consistency with persistent state.
func (strategy *Strategy) HotReload(mongoStrategy models.MongoStrategy) {
	strategy.Log.Info("hot reloading",
		zap.String("id", strategy.ID()),
	)
	strategy.Model.Enabled = mongoStrategy.Enabled
	strategy.Model.Conditions = mongoStrategy.Conditions
	if mongoStrategy.Enabled == false {
		if strategy.StrategyRuntime != nil {
			strategy.StrategyRuntime.Stop() // stop runtime if disabled by DB, externally
		}
	}
	strategy.Statsd.Inc("strategy.hot_reload")
}

// Settle takes the strategy to work in the instance trying to set a distributed lock.
func (strategy *Strategy) Settle() (bool, error) {
	strategy.Log.Debug("trying to lock mutex", zap.String("name", strategy.SettlementMutex.Name()))
	if err := strategy.SettlementMutex.Lock(); err != nil {
		strategy.Log.Debug("mutex lock failed",
			zap.String("name", strategy.SettlementMutex.Name()),
			zap.Error(err),
		)
		if err == redsync.ErrFailed {
			return false, nil // already locked
		}
		return false, err // unexpected error
	}
	strategy.Log.Info("mutex locked", zap.String("name", strategy.SettlementMutex.Name()))
	// extend settlement
	go func() {
		for {
			time.Sleep(3 * time.Second) // TODO(khassanov): connect this with watchdog time
			strategy.Log.Debug("extending settlement", zap.String("name", strategy.SettlementMutex.Name()))
			success, err := strategy.SettlementMutex.Extend()
			if !success || err != nil {
				strategy.Log.Error("settlement mutex extension",
					zap.Bool("success", success),
					zap.String("name", strategy.SettlementMutex.Name()),
					zap.Error(err),
				)
				return
			}
			if !strategy.Model.Enabled {
				strategy.Log.Info("disable settlement mutex extension",
					zap.String("name", strategy.SettlementMutex.Name()),
					zap.Bool("strategy.Model.Enabled", strategy.Model.Enabled),
					zap.String("strategy.Model.State.State", strategy.Model.State.State),
				)
				return
			}
		}
	}()
	return true, nil
}

func (strategy *Strategy) Relieve() error {
	// TODO(khassanov)
	return nil
}
