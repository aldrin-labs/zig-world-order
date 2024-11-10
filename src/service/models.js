export class Strategy {
  constructor({ id, conditions }) {
    this.id = id;
    this.enabled = true;
    this.conditions = conditions;
    this.state = {
      status: 'open',
      executedAmount: 0,
      entryPrice: 0,
      exitPrice: 0,
    };
  }
}

export class Order {
  constructor({ symbol, side, amount, price, type = 'limit' }) {
    this.symbol = symbol;
    this.side = side;
    this.amount = amount;
    this.price = price;
    this.type = type;
  }
}