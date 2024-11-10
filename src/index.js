import express from 'express';
import { StrategyService } from './service/strategyService.js';
import { TradingClient } from './trading/tradingClient.js';
import { DataFeed } from './service/dataFeed.js';

const app = express();
const port = process.env.PORT || 8080;

app.use(express.json());

// Initialize services
const dataFeed = new DataFeed();
const tradingClient = new TradingClient();
const strategyService = new StrategyService(tradingClient, dataFeed);

// Health check endpoint
app.get('/healthz', (req, res) => {
  res.send('alive!\n');
});

// Create order endpoint
app.post('/createOrder', async (req, res) => {
  try {
    const response = await strategyService.createOrder(req.body);
    res.json(response);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Cancel order endpoint
app.post('/cancelOrder', async (req, res) => {
  try {
    const response = await strategyService.cancelOrder(req.body);
    res.json(response);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Start server
app.listen(port, () => {
  console.log(`Server listening on port ${port}`);
});