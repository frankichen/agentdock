plugins {
    id("com.android.application")
}

val appVersion = providers.gradleProperty("appVersion").orElse("0.1.0-dev")
val gitSha = providers.gradleProperty("gitSha").orElse("unknown")

android {
    namespace = "com.frankichen.ultimateagent"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.frankichen.ultimateagent"
        minSdk = 30
        targetSdk = 35
        versionCode = 1
        versionName = appVersion.get()
        buildConfigField("String", "APP_VERSION", "\"${appVersion.get()}\"")
        buildConfigField("String", "GIT_SHA", "\"${gitSha.get()}\"")
    }

    buildFeatures { buildConfig = true }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    lint {
        abortOnError = true
        warningsAsErrors = true
    }
}

dependencies {
    implementation("org.nanohttpd:nanohttpd:2.3.1")
    testImplementation("junit:junit:4.13.2")
}
