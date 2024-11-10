const std = @import("std");

pub const Strategy = struct {
    id: []const u8,
    enabled: bool,
    conditions: StrategyConditions,
    state: StrategyState,
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator, order: CreateOrderRequest) !*Strategy {
        const strategy = try allocator.create(Strategy);
        strategy.* = Strategy{
            .id = try allocator.dupe(u8, order.id),
            .enabled = true,
            .conditions = StrategyConditions{
                .pair = try allocator.dupe(u8, order.pair),
                .side = try allocator.dupe(u8, order.side),
                .amount = order.amount,
                .price = order.price,
            },
            .state = StrategyState{
                .status = "open",
                .executed_amount = 0,
            },
            .allocator = allocator,
        };
        return strategy;
    }

    pub fn deinit(self: *Strategy) void {
        self.allocator.free(self.id);
        self.allocator.free(self.conditions.pair);
        self.allocator.free(self.conditions.side);
        self.allocator.destroy(self);
    }
};

pub const StrategyConditions = struct {
    pair: []const u8,
    side: []const u8,
    amount: f64,
    price: f64,
};

pub const StrategyState = struct {
    status: []const u8,
    executed_amount: f64,
};

pub const CreateOrderRequest = struct {
    id: []const u8,
    pair: []const u8,
    side: []const u8,
    amount: f64,
    price: f64,

    pub fn deinit(self: *CreateOrderRequest) void {
        // Free allocated memory when using dynamic arrays
    }
};

pub const CancelOrderRequest = struct {
    strategy_id: []const u8,

    pub fn deinit(self: *CancelOrderRequest) void {
        // Free allocated memory when using dynamic arrays
    }
};