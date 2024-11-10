package redis

import (
	"context"
	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	redsyncredigo "github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/gomodule/redigo/redis"
	"gitlab.com/crypto_project/core/strategy_service/src/logging"
	"gitlab.com/crypto_project/core/strategy_service/src/service/interfaces"
	"go.uber.org/zap"
	"os"
	"time"
)

var redisPool *redis.Pool
var pubsubredisPool *redis.Pool
var redisDLMPool *redis.Pool
var redsyncDLMPool redsyncredis.Pool
var redsyncToDLM *redsync.Redsync
var log interfaces.ILogger

func init() {
	logger, _ := logging.GetZapLogger()
	// TODO: handle the error
	log = logger.With(zap.String("logger", "srcRedis"))
}

func GetRedisPubsub() (*redis.PubSubConn, redis.Conn) {
	var conn = GetRedisClientInstance(true, false, false)
	return &redis.PubSubConn{Conn: conn}, conn
}

func GetRedisClientInstance(pubsub bool, master bool, newClient bool) redis.Conn {
	var con redis.Conn
	if pubsub {
		if pubsubredisPool == nil || newClient {
			log.Info("connect to pubsub redis")
			pubsubredisPool = &redis.Pool{
				MaxActive:   3000000,
				MaxIdle:     3000000,
				IdleTimeout: 260 * time.Second,

				// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
				Dial: func() (redis.Conn, error) {
					c, err := redis.Dial("tcp", os.Getenv("REDIS_HOST")+":"+os.Getenv("REDIS_PORT"))
					if err != nil {
						log.Error("pubsub dial1 error", zap.Error(err))
						return nil, err
					}
					if _, err := c.Do("AUTH", os.Getenv("REDIS_PASSWORD")); err != nil {
						log.Error("pubsub dial2 error", zap.Error(err))
						c.Close()
						return nil, err
					}
					if _, err := c.Do("SELECT", 0); err != nil {
						log.Error("pubsub dial3 error", zap.Error(err))
						c.Close()
						return nil, err
					}
					return c, nil
				},
			}
		}

		con = pubsubredisPool.Get()
	}
	if redisPool == nil {
		log.Info("connect to redis")
		redisPool = &redis.Pool{
			MaxActive:   300000,
			MaxIdle:     300000,
			IdleTimeout: 20 * time.Second,

			// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", os.Getenv("REDIS_HOST")+":"+os.Getenv("REDIS_PORT"))
				if err != nil {
					log.Error("pubsub dial1 error", zap.Error(err))
					return nil, err
				}
				if _, err := c.Do("AUTH", os.Getenv("REDIS_PASSWORD")); err != nil {
					log.Error("pubsub dial2 error", zap.Error(err))
					c.Close()
					return nil, err
				}
				if _, err := c.Do("SELECT", 0); err != nil {
					log.Error("pubsub dial3 error", zap.Error(err))
					c.Close()
					return nil, err
				}
				return c, nil
			},
		}
	}
	if !pubsub {
		con = redisPool.Get()
	}
	// Test the connection

	_, err := con.Do("PING")
	if err != nil {
		log.Error("can't connect to the redis database", zap.Error(err))
		time.Sleep(500 * time.Millisecond)
		return GetRedisClientInstance(pubsub, master, newClient)
	}
	return con
}

func GetRedsync() *redsync.Redsync {
	if redisDLMPool == nil {
		log.Info("connecting to redis DLM pool")
		redisDLMPool = &redis.Pool{
			MaxActive:   300000,
			MaxIdle:     300000,
			IdleTimeout: 20 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", os.Getenv("REDIS_HOST")+":"+os.Getenv("REDIS_PORT"))
				if err != nil {
					log.Error("redis DLM dial 1/3 error", zap.Error(err))
					return nil, err
				}
				if _, err := c.Do("AUTH", os.Getenv("REDIS_PASSWORD")); err != nil {
					log.Error("redis DLM dial 2/3 error", zap.Error(err))
					c.Close()
					return nil, err
				}
				if _, err := c.Do("SELECT", 0); err != nil {
					log.Error("redis DLM dial 3/3 error", zap.Error(err))
					c.Close()
					return nil, err
				}
				return c, nil
			},
		}
		redsyncDLMPool = redsyncredigo.NewPool(redisDLMPool)
		redsyncToDLM = redsync.New(redsyncDLMPool)
	}
	con := redisDLMPool.Get()
	_, err := con.Do("PING")
	if err != nil {
		log.Error("can't connect to redis DLM pool, retry", zap.Error(err))
		time.Sleep(500 * time.Millisecond)
		return GetRedsync()
	}
	log.Info("connected to redis DLM pool")
	return redsyncToDLM
}

func ListenPubSubChannels(ctx context.Context,
	onStart func() error,
	onMessage func(channel string, data []byte) error,
	channels ...string) error {
	// A ping is set to the server with this period to test for the health of
	// the connection and server.
	const healthCheckPeriod = time.Minute
	c := GetRedisClientInstance(true, false, false)
	defer c.Close()

	psc := redis.PubSubConn{Conn: c}
	if err := psc.PSubscribe(redis.Args{}.AddFlat(channels)...); err != nil {
		return err
	}

	done := make(chan error, 1)

	// Start a goroutine to receive notifications from the server.
	go func() {
		for {
			switch n := psc.Receive().(type) {
			case error:
				done <- n
				return
			case redis.Message:
				if err := onMessage(n.Channel, n.Data); err != nil { // you can run gorouitine in onMessage or keep your processing single-threaded
					done <- err
					return
				}
			case redis.Subscription:
				switch n.Count {
				case len(channels):
					// Notify application when all channels are subscribed.
					if err := onStart(); err != nil {
						done <- err
						return
					}
				case 0:
					// Return from the goroutine when all channels are unsubscribed.
					done <- nil
					return
				}
			}
		}
	}()

	ticker := time.NewTicker(healthCheckPeriod)
	defer ticker.Stop()
	var err error
Loop:
	for err == nil {
		select {
		case <-ticker.C:
			// Send ping to test health of connection and server. If
			// corresponding pong is not received, then receive on the
			// connection will timeout and the receive goroutine will exit.
			if err = psc.Ping(""); err != nil {
				break Loop
			}
		case <-ctx.Done():
			break Loop
		case err := <-done:
			// Return error from the receive goroutine.
			return err
		}
	}

	// Signal the receiving goroutine to exit by unsubscribing from all channels.
	err = psc.Unsubscribe()
	_ = psc.Close()
	if err != nil {
		log.Error("EXIT1 EOF", zap.Error(err))
	}

	// Wait for goroutine to complete.
	<-done
	log.Info("EXIT1 EOF")
	// os.Exit(1)
	//if resp != nil {
	//	log.Print("recursive call")
	//	return ListenPubSubChannels(ctx, onStart, onMessage, channels[0])
	//}
	return ListenPubSubChannels(ctx, onStart, onMessage, channels[0])
}
