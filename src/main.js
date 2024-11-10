import { createServer } from './server.js';
import { StrategyService } from './service/service.js';

async function main() {
  const strategyService = new StrategyService();
  await strategyService.init();

  const server = createServer(strategyService);
  const port = process.env.PORT || 8080;

  server.listen(port, () => {
    console.log(`Server listening on port ${port}`);
  });
}

main().catch(console.error);