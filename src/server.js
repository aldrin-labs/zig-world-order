import express from 'express';

export function createServer(strategyService) {
  const app = express();
  app.use(express.json());

  app.get('/healthz', (req, res) => {
    res.send('alive!\n');
  });

  app.post('/createOrder', async (req, res) => {
    try {
      const response = await strategyService.createOrder(req.body);
      res.json(response);
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  app.post('/cancelOrder', async (req, res) => {
    try {
      const response = await strategyService.cancelOrder(req.body);
      res.json(response);
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  return app;
}