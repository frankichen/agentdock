package com.frankichen.ultimateagent;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.os.IBinder;

public final class UltimateAgentService extends Service {
    private static final int NOTIFICATION_ID = 18767;
    private static final String CHANNEL_ID = "ultimate_android_agent";
    private static volatile boolean running;

    private AgentServerRunner serverRunner;

    public static boolean isRunning() {
        return running;
    }

    @Override
    public void onCreate() {
        super.onCreate();
        running = true;
        createNotificationChannel();
        startForeground(NOTIFICATION_ID, buildNotification(false));
        serverRunner = new AgentServerRunner(this);
        serverRunner.start();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        AgentPrefs.setEnabled(this, true);
        return START_STICKY;
    }

    @Override
    public void onDestroy() {
        running = false;
        if (serverRunner != null) {
            serverRunner.close();
            serverRunner = null;
        }
        super.onDestroy();
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    void updateBoundNotification(boolean bound) {
        NotificationManager manager = getSystemService(NotificationManager.class);
        if (manager != null) {
            manager.notify(NOTIFICATION_ID, buildNotification(bound));
        }
    }

    private Notification buildNotification(boolean bound) {
        String text = getString(bound ? R.string.notification_bound : R.string.notification_waiting);
        return new Notification.Builder(this, CHANNEL_ID)
                .setContentTitle(getString(R.string.notification_title))
                .setContentText(text)
                .setSmallIcon(R.drawable.ic_agent)
                .setOngoing(true)
                .setCategory(Notification.CATEGORY_SERVICE)
                .build();
    }

    private void createNotificationChannel() {
        NotificationChannel channel = new NotificationChannel(
                CHANNEL_ID,
                getString(R.string.notification_channel),
                NotificationManager.IMPORTANCE_LOW);
        NotificationManager manager = getSystemService(NotificationManager.class);
        if (manager != null) {
            manager.createNotificationChannel(channel);
        }
    }
}
