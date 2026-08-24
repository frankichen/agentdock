package com.frankichen.ultimateagent;

import java.io.IOException;

final class AgentServerRunner implements AutoCloseable {
    private static final long RETRY_DELAY_MS = 2_000L;

    private final UltimateAgentService service;
    private final Thread thread;
    private volatile boolean stopped;
    private AgentHttpServer server;

    AgentServerRunner(UltimateAgentService service) {
        this.service = service;
        thread = new Thread(this::runLoop, "ultimate-agent-bind-loop");
        thread.setDaemon(true);
    }

    void start() {
        thread.start();
    }

    private void runLoop() {
        String bindIp = AgentPrefs.bindIp(service);
        int port = AgentPrefs.port(service);
        while (!stopped) {
            boolean addressAvailable = NetworkAddressUtil.hasLocalAddress(bindIp);
            if (addressAvailable && server == null) {
                try {
                    server = new AgentHttpServer(service, bindIp, port);
                    server.start(8_000, false);
                    service.updateBoundNotification(true);
                } catch (IOException | RuntimeException error) {
                    if (server != null) {
                        server.stop();
                        server = null;
                    }
                    service.updateBoundNotification(false);
                }
            } else if (!addressAvailable && server != null) {
                server.stop();
                server = null;
                service.updateBoundNotification(false);
            }

            try {
                Thread.sleep(RETRY_DELAY_MS);
            } catch (InterruptedException ignored) {
                Thread.currentThread().interrupt();
                return;
            }
        }
    }

    @Override
    public void close() {
        stopped = true;
        thread.interrupt();
        AgentHttpServer current = server;
        server = null;
        if (current != null) {
            current.stop();
        }
        service.updateBoundNotification(false);
    }
}
