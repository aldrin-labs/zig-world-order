import fetch from 'node-fetch';
import WebSocket from 'ws';

export class TradingClient {
  constructor() {
    this.baseUrl = process.env.EXCHANGE_SERVICE || 'http://localhost:3000';
    this.ws = null;
  }

  async init() {
    // Initialize WebSocket connection for real-time data
    this.ws = new WebSocket('wss://stream.binance.com:9443/ws/!ticker@arr');
    
    this.ws.on('message', (data) => {
      this.handleTickerUpdate(JSON.parse(data));
    });
  }

  async createOrder(request) {
    const response = await fetch(`${this.baseUrl}/createOrder`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });

    return response.json();
  }

  async cancelOrder(request) {
    const response = await fetch(`${this.baseUrl}/cancelOrder`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });

    return response.json();
  }

  handleTickerUpdate(data) {
    // Process real-time market data
    console.log('Received market update:', data);
  }
}