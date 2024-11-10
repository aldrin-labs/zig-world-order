export class Strategy {
  constructor({ id, conditions, dataFeed }) {
    this.id = id;
    this.enabled = true;
    this.conditions = conditions;
    this.dataFeed = dataFeed;
    this.state = {
      status: 'open',
      executedAmount: 0,
      entryPrice: 0,
      exitPrice: 0
    };
  }

  start() {
    this.enabled = true;
    this.dataFeed.subscribe(this.conditions.symbol, this.handlePriceUpdate.bind(this));
  }

  stop() {
    this.enabled = false;
    this.dataFeed.unsubscribe(this.conditions.symbol, this.handlePriceUpdate.bind(this));
  }

  handlePriceUpdate(price) {
    if (!this.enabled) return;

    // Implement smart order logic here
    if (this.conditions.side === 'buy' && price <= this.conditions.price) {
      this.executeBuy(price);
    } else if (this.conditions.side === 'sell' && price >= this.conditions.price) {
      this.executeSell(price);
    }
  }

  executeBuy(price) {
    this.state.entryPrice = price;
    this.state.executedAmount = this.conditions.amount;
    this.state.status = 'filled';
  }

  executeSell(price) {
    this.state.exitPrice = price;
    this.state.executedAmount = this.conditions.amount;
    this.state.status = 'filled';
  }
}