const std = @import("std");
const Server = @import("server.zig").Server;
const service = @import("service/service.zig");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    // Initialize strategy service
    var strategy_service = try service.StrategyService.init(allocator);
    defer strategy_service.deinit();

    // Initialize HTTP server
    var server = try Server.init(allocator, &strategy_service);
    defer server.deinit();

    try server.start();
}