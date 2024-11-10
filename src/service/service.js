import { TradingClient } from './trading/trading.js';
import { Strategy } from './models.js';

export class StrategyService {
  constructor() {
    this.strategies = new Map();
    this.tradingClient = new TradingClient();
  }

  async init() {
    await this.tradingClient.init();
  }

  async createOrder(request) {
    const strategy = new Strategy({
      id: request.keyId,
      conditions: request.keyParams,
    });

    this.strategies.set(strategy.id, strategy);
    const response = await this.tradingClient.createOrder(request);
    return response;
  }

  async cancelOrder(request) {
    const strategy = this.strategies.get(request.keyId);
    if (!strategy) {
      throw new Error('Strategy not found');
    }

    strategy.enabled = false;
    const response = await this.tradingClient.cancelOrder(request);
    return response;
  }
}