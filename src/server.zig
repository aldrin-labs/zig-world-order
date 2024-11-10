const std = @import("std");
const service = @import("service/service.zig");

pub const Server = struct {
    allocator: std.mem.Allocator,
    strategy_service: *service.StrategyService,
    server: std.http.Server,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator, strategy_svc: *service.StrategyService) !Self {
        return Self{
            .allocator = allocator,
            .strategy_service = strategy_svc,
            .server = try std.http.Server.init(allocator, .{
                .address = "127.0.0.1",
                .port = 8080,
            }),
        };
    }

    pub fn deinit(self: *Self) void {
        self.server.deinit();
    }

    pub fn start(self: *Self) !void {
        std.debug.print("Server listening on port 8080\n", .{});
        
        while (true) {
            const conn = try self.server.accept();
            defer conn.deinit();

            try self.handleRequest(conn);
        }
    }

    fn handleRequest(self: *Self, conn: *std.http.Server.Connection) !void {
        const request = try conn.readRequest();
        defer request.deinit();

        if (std.mem.eql(u8, request.method, "POST")) {
            if (std.mem.eql(u8, request.path, "/createOrder")) {
                try self.handleCreateOrder(conn, request);
            } else if (std.mem.eql(u8, request.path, "/cancelOrder")) {
                try self.handleCancelOrder(conn, request);
            }
        } else if (std.mem.eql(u8, request.method, "GET")) {
            if (std.mem.eql(u8, request.path, "/healthz")) {
                try conn.writeResponse(.{
                    .status = .ok,
                    .body = "alive!\n",
                });
            }
        }
    }

    fn handleCreateOrder(self: *Self, conn: *std.http.Server.Connection, request: std.http.Request) !void {
        const body = try request.readBody();
        defer self.allocator.free(body);

        const response = try self.strategy_service.createOrder(body);
        try conn.writeResponse(.{
            .status = .ok,
            .body = response,
        });
    }

    fn handleCancelOrder(self: *Self, conn: *std.http.Server.Connection, request: std.http.Request) !void {
        const body = try request.readBody();
        defer self.allocator.free(body);

        const response = try self.strategy_service.cancelOrder(body);
        try conn.writeResponse(.{
            .status = .ok,
            .body = response,
        });
    }
};