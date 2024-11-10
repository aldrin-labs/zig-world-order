import WebSocket from 'ws';

export class DataFeed {
  constructor() {
    this.subscribers = new Map();
    this.ws = null;
    this.connect();
  }

  connect() {
    this.ws = new WebSocket('wss://stream.binance.com:9443/ws/!ticker@arr');
    
    this.ws.on('message', (data) => {
      const tickers = JSON.parse(data);
      this.handleTickers(tickers);
    });

    this.ws.on('error', (error) => {
      console.error('WebSocket error:', error);
      setTimeout(() => this.connect(), 5000);
    });
  }

  handleTickers(tickers) {
    for (const ticker of tickers) {
      const symbol = ticker.s;
      const price = parseFloat(ticker.c);
      
      if (this.subscribers.has(symbol)) {
        const callbacks = this.subscribers.get(symbol);
        callbacks.forEach(callback => callback(price));
      }
    }
  }

  subscribe(symbol, callback) {
    if (!this.subscribers.has(symbol)) {
      this.subscribers.set(symbol, new Set());
    }
    this.subscribers.get(symbol).add(callback);
  }

  unsubscribe(symbol, callback) {
    if (this.subscribers.has(symbol)) {
      this.subscribers.get(symbol).delete(callback);
    }
  }
}