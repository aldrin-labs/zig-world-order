import { test } from 'node:test';
import assert from 'node:assert';
import { StrategyService } from '../src/service/service.js';

test('StrategyService - Create Order', async (t) => {
  const service = new StrategyService();
  await service.init();

  const request = {
    keyId: '123',
    keyParams: {
      symbol: 'BTCUSDT',
      side: 'BUY',
      amount: 0.1,
      price: 35000,
    },
  };

  const response = await service.createOrder(request);
  assert.equal(response.status, 'OK');
});

test('StrategyService - Cancel Order', async (t) => {
  const service = new StrategyService();
  await service.init();

  const request = {
    keyId: '123',
    keyParams: {
      orderId: '456',
      pair: 'BTCUSDT',
    },
  };

  const response = await service.cancelOrder(request);
  assert.equal(response.status, 'OK');
});