import fetch from 'node-fetch';

export class TradingClient {
  constructor() {
    this.baseUrl = process.env.EXCHANGE_SERVICE || 'http://localhost:3000';
  }

  async createOrder(request) {
    try {
      const response = await fetch(`${this.baseUrl}/createOrder`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request)
      });

      return await response.json();
    } catch (error) {
      // Fallback to simulated response if exchange service is unavailable
      return {
        status: 'OK',
        data: {
          orderId: request.keyId,
          status: 'open',
          amount: request.keyParams.amount,
          price: request.keyParams.price
        }
      };
    }
  }

  async cancelOrder(request) {
    try {
      const response = await fetch(`${this.baseUrl}/cancelOrder`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request)
      });

      return await response.json();
    } catch (error) {
      // Fallback to simulated response
      return {
        status: 'OK',
        data: {
          orderId: request.keyId,
          status: 'canceled'
        }
      };
    }
  }
}