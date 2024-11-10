const std = @import("std");
const trading = @import("trading/trading.zig");
const models = @import("models.zig");

pub const StrategyService = struct {
    allocator: std.mem.Allocator,
    strategies: std.StringHashMap(*models.Strategy),
    trading_client: trading.TradingClient,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator) !Self {
        return Self{
            .allocator = allocator,
            .strategies = std.StringHashMap(*models.Strategy).init(allocator),
            .trading_client = try trading.TradingClient.init(allocator),
        };
    }

    pub fn deinit(self: *Self) void {
        var it = self.strategies.iterator();
        while (it.next()) |entry| {
            self.allocator.destroy(entry.value_ptr.*);
        }
        self.strategies.deinit();
        self.trading_client.deinit();
    }

    pub fn createOrder(self: *Self, request_body: []const u8) ![]const u8 {
        const order = try std.json.parse(models.CreateOrderRequest, request_body);
        defer order.deinit();

        const strategy = try models.Strategy.init(self.allocator, order);
        try self.strategies.put(strategy.id, strategy);

        const response = try self.trading_client.createOrder(order);
        return response;
    }

    pub fn cancelOrder(self: *Self, request_body: []const u8) ![]const u8 {
        const cancel_request = try std.json.parse(models.CancelOrderRequest, request_body);
        defer cancel_request.deinit();

        if (self.strategies.get(cancel_request.strategy_id)) |strategy| {
            strategy.enabled = false;
            const response = try self.trading_client.cancelOrder(cancel_request);
            return response;
        }

        return "Strategy not found";
    }
};