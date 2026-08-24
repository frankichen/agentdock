package com.frankichen.ultimateagent;

import android.Manifest;
import android.app.Activity;
import android.app.AlertDialog;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.ComponentName;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.provider.Settings;
import android.text.TextUtils;
import android.view.View;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import java.util.Locale;

public final class MainActivity extends Activity {
    private TextView statusView;
    private TextView tokenView;
    private boolean tokenVisible;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        AgentPrefs.token(this);
        requestNotificationPermissionIfNeeded();
        setContentView(buildContent());
        if (AgentPrefs.isEnabled(this)) {
            startAgentService();
        }
    }

    @Override
    protected void onResume() {
        super.onResume();
        refreshStatus();
    }

    private View buildContent() {
        int padding = dp(20);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(padding, padding, padding, padding);

        TextView title = new TextView(this);
        title.setText("Ultimate Android Agent");
        title.setTextSize(26);
        root.addView(title);

        TextView contract = new TextView(this);
        contract.setText(
                "Tailscale 完全由你手动启动。这个 Agent 不会启动、唤醒、配置或修改 Tailscale。\n"
                        + "手机开机后 Agent 可以自己等待；当 100.74.240.112 出现时才绑定私网服务。\n"
                        + "正常手机控制走 Android Accessibility；ADB 仅保留给维护/调试。\n"
                        + "版本：" + BuildConfig.APP_VERSION + " / " + BuildConfig.GIT_SHA);
        contract.setTextSize(16);
        contract.setPadding(0, dp(16), 0, dp(16));
        root.addView(contract);

        statusView = new TextView(this);
        statusView.setTextSize(17);
        root.addView(statusView);

        Button startButton = button("启动 Agent");
        startButton.setOnClickListener(view -> {
            AgentPrefs.setEnabled(this, true);
            startAgentService();
            refreshStatus();
        });
        root.addView(startButton);

        Button stopButton = button("停止 Agent");
        stopButton.setOnClickListener(view -> {
            AgentPrefs.setEnabled(this, false);
            stopService(new Intent(this, UltimateAgentService.class));
            refreshStatus();
        });
        root.addView(stopButton);

        Button accessibilityButton = button("打开无障碍设置");
        accessibilityButton.setOnClickListener(view ->
                startActivity(new Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS)));
        root.addView(accessibilityButton);

        tokenView = new TextView(this);
        tokenView.setTextSize(15);
        tokenView.setTextIsSelectable(true);
        tokenView.setPadding(0, dp(18), 0, dp(8));
        root.addView(tokenView);

        Button showTokenButton = button("显示 / 隐藏 Bearer Token");
        showTokenButton.setOnClickListener(view -> {
            tokenVisible = !tokenVisible;
            refreshToken();
        });
        root.addView(showTokenButton);

        Button copyTokenButton = button("复制 Bearer Token");
        copyTokenButton.setOnClickListener(view -> {
            ClipboardManager clipboard = getSystemService(ClipboardManager.class);
            if (clipboard != null) {
                clipboard.setPrimaryClip(ClipData.newPlainText(
                        "Ultimate Agent Token",
                        AgentPrefs.token(this)));
            }
        });
        root.addView(copyTokenButton);

        Button rotateTokenButton = button("轮换 Bearer Token");
        rotateTokenButton.setOnClickListener(view -> new AlertDialog.Builder(this)
                .setTitle("轮换 Token？")
                .setMessage("轮换后 Ubuntu Bridge 必须同步新 Token。")
                .setNegativeButton("取消", null)
                .setPositiveButton("轮换", (dialog, which) -> {
                    AgentPrefs.rotateToken(this);
                    refreshToken();
                })
                .show());
        root.addView(rotateTokenButton);

        ScrollView scrollView = new ScrollView(this);
        scrollView.addView(root);
        return scrollView;
    }

    private Button button(String text) {
        Button button = new Button(this);
        button.setText(text);
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT);
        params.topMargin = dp(8);
        button.setLayoutParams(params);
        return button;
    }

    private void refreshStatus() {
        if (statusView == null) {
            return;
        }
        boolean hasTailnetAddress = NetworkAddressUtil.hasLocalAddress(AgentPrefs.bindIp(this));
        statusView.setText(
                "Agent 开机等待：" + (AgentPrefs.isEnabled(this) ? "开启" : "关闭") + "\n"
                        + "Foreground Service：" + (UltimateAgentService.isRunning() ? "运行中" : "未运行") + "\n"
                        + "无障碍服务：" + (isAccessibilityEnabled() ? "已启用" : "未启用") + "\n"
                        + "Tailscale 私网 IP：" + (hasTailnetAddress ? "已出现" : "未出现（等待你手动启动 Tailscale）") + "\n"
                        + String.format(
                                Locale.ROOT,
                                "监听目标：http://%s:%d",
                                AgentPrefs.bindIp(this),
                                AgentPrefs.port(this)));
        refreshToken();
    }

    private void refreshToken() {
        if (tokenView == null) {
            return;
        }
        tokenView.setText("Bearer Token：" + (tokenVisible ? AgentPrefs.token(this) : AgentPrefs.maskedToken(this)));
    }

    private boolean isAccessibilityEnabled() {
        String enabled = Settings.Secure.getString(
                getContentResolver(),
                Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES);
        if (TextUtils.isEmpty(enabled)) {
            return false;
        }
        ComponentName component = new ComponentName(this, UltimateAccessibilityService.class);
        return enabled.toLowerCase(Locale.ROOT).contains(
                component.flattenToString().toLowerCase(Locale.ROOT));
    }

    private void startAgentService() {
        startForegroundService(new Intent(this, UltimateAgentService.class));
    }

    private void requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= 33
                && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, 1001);
        }
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }
}
