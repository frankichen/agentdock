package com.frankichen.ultimateagent;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

public final class BootReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null || !Intent.ACTION_BOOT_COMPLETED.equals(intent.getAction())) {
            return;
        }
        if (!AgentPrefs.isEnabled(context)) {
            return;
        }
        Intent serviceIntent = new Intent(context, UltimateAgentService.class);
        context.startForegroundService(serviceIntent);
    }
}
