package com.frankichen.ultimateagent;

import java.net.InetAddress;
import java.net.NetworkInterface;
import java.net.SocketException;
import java.util.Collections;

final class NetworkAddressUtil {
    private NetworkAddressUtil() {
    }

    static boolean hasLocalAddress(String expectedAddress) {
        try {
            for (NetworkInterface networkInterface : Collections.list(NetworkInterface.getNetworkInterfaces())) {
                if (!networkInterface.isUp()) {
                    continue;
                }
                for (InetAddress address : Collections.list(networkInterface.getInetAddresses())) {
                    if (expectedAddress.equals(address.getHostAddress())) {
                        return true;
                    }
                }
            }
        } catch (SocketException ignored) {
            return false;
        }
        return false;
    }
}
