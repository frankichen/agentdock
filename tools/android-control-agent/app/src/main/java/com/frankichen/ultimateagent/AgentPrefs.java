package com.frankichen.ultimateagent;

import android.content.Context;
import android.content.SharedPreferences;
import android.util.Base64;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.SecureRandom;

final class AgentPrefs {
    static final String DEFAULT_BIND_IP = "100.74.240.112";
    static final int DEFAULT_PORT = 18767;

    private static final String PREFS = "ultimate_agent";
    private static final String KEY_ENABLED = "enabled";
    private static final String KEY_TOKEN = "bearer_token";
    private static final String KEY_BIND_IP = "bind_ip";
    private static final String KEY_PORT = "port";

    private AgentPrefs() {
    }

    static boolean isEnabled(Context context) {
        return prefs(context).getBoolean(KEY_ENABLED, true);
    }

    static void setEnabled(Context context, boolean enabled) {
        prefs(context).edit().putBoolean(KEY_ENABLED, enabled).apply();
    }

    static String bindIp(Context context) {
        return prefs(context).getString(KEY_BIND_IP, DEFAULT_BIND_IP);
    }

    static int port(Context context) {
        return prefs(context).getInt(KEY_PORT, DEFAULT_PORT);
    }

    static String token(Context context) {
        SharedPreferences preferences = prefs(context);
        String current = preferences.getString(KEY_TOKEN, "");
        if (current != null && !current.isBlank()) {
            return current;
        }
        return rotateToken(context);
    }

    static String rotateToken(Context context) {
        byte[] random = new byte[32];
        new SecureRandom().nextBytes(random);
        String token = Base64.encodeToString(
                random,
                Base64.URL_SAFE | Base64.NO_PADDING | Base64.NO_WRAP);
        prefs(context).edit().putString(KEY_TOKEN, token).commit();
        return token;
    }

    static boolean tokenMatches(Context context, String candidate) {
        if (candidate == null || candidate.isBlank()) {
            return false;
        }
        byte[] expected = token(context).getBytes(StandardCharsets.UTF_8);
        byte[] supplied = candidate.getBytes(StandardCharsets.UTF_8);
        return MessageDigest.isEqual(expected, supplied);
    }

    static String maskedToken(Context context) {
        String token = token(context);
        if (token.length() <= 12) {
            return "********";
        }
        return token.substring(0, 6) + "…" + token.substring(token.length() - 6);
    }

    private static SharedPreferences prefs(Context context) {
        return context.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }
}
