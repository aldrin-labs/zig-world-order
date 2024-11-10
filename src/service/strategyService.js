import { Strategy } from '../models/strategy.js';

export class StrategyService {
  constructor(tradingClient, dataFeed) {
    this.tradingClient = tradingClient;
    this.dataFeed = dataFeed;
    this.strategies = new Map();
  }

  async createOrder(request) {
    const strategy = new Strategy({
      id: request.keyId,
      conditions: request.keyParams,
      dataFeed: this.dataFeed
    });

    this.strategies.set(strategy.id, strategy);
    
    const response = await this.tradingClient.createOrder(request);
    
    if (response.status === 'OK') {
      strategy.start();
    }
    
    return response;
  }

  async cancelOrder(request) {
    const strategy = this.strategies.get(request.keyId);
    if (!strategy) {
      throw new Error('Strategy not found');
    }

    strategy.stop();
    const response = await this.tradingClient.cancelOrder(request);
    return response;
  }
}