plugins {
  alias(libs.plugins.android.application)
  alias(libs.plugins.compose.compiler)
}

android {
  namespace = "gal.tribo.librequenda"
  compileSdk = 36
  // A council's own app (install.sh): its package, name and launcher icon. The redirect
  // scheme is the package, so the server must list it in ANDROID_APP_IDS (zitadel/setup.sh).
  val appId = providers.gradleProperty("applicationId").orNull ?: "gal.tribo.librequenda"
  require(appId.matches(Regex("[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)+"))) { "Invalid applicationId" }
  defaultConfig {
    applicationId = appId
    minSdk = 26
    targetSdk = 36
    testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    // CI passes the tag (v1.2.3 -> 1.2.3) and its run number; 1 and 0.1 for a local build
    versionCode = providers.gradleProperty("versionCode").orNull?.toIntOrNull() ?: 1
    versionName = providers.gradleProperty("versionName").orNull ?: "0.1"
    val apiUrl = providers.gradleProperty("apiBaseUrl")
      .orElse(providers.environmentVariable("ANDROID_API_BASE_URL"))
      .orElse("http://localhost:8080/api").get().trimEnd('/')
    require(apiUrl.matches(Regex("https?://[^\\s\\\"\\\\]+"))) { "Invalid API URL" }
    buildConfigField("String", "API_BASE_URL", "\"$apiUrl\"")
    // AppAuth's redirect activity: Zitadel sends the sign-in back to <applicationId>:/oauth2redirect
    manifestPlaceholders["appAuthRedirectScheme"] = appId
  }

  // -Pbrand=<dir>: Android resources (app_name, launcher icons) over the main ones
  providers.gradleProperty("brand").orNull?.let { brand ->
    sourceSets.getByName("release").res.srcDir(file(brand))
  }

  // the release key, only when CI has it (secrets): never in the repository
  val keystore = providers.environmentVariable("ANDROID_KEYSTORE_FILE").orNull
  signingConfigs {
    if (keystore != null) {
      create("release") {
        storeFile = file(keystore)
        storePassword = providers.environmentVariable("ANDROID_KEYSTORE_PASSWORD").get()
        keyAlias = providers.environmentVariable("ANDROID_KEY_ALIAS").get()
        keyPassword = providers.environmentVariable("ANDROID_KEY_PASSWORD").get()
      }
    }
  }

  buildTypes {
    release {
      if (keystore != null) signingConfig = signingConfigs.getByName("release")
      isMinifyEnabled = false
      proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
    }
  }
  // Play splits an App Bundle by language and installs only the device's ones: the app's own
  // language picker then finds no strings for gl or ca. Every language goes in every install.
  bundle {
    language {
      enableSplit = false
    }
  }
  compileOptions {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
  }
  buildFeatures {
    compose = true
    aidl = false
    buildConfig = true
    shaders = false
  }

  packaging {
    resources {
      excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }
  }
}

kotlin {
  jvmToolchain(17)
}

dependencies {
  val composeBom = platform(libs.androidx.compose.bom)
  implementation(composeBom)
  androidTestImplementation(composeBom)

  // Core Android dependencies
  implementation(libs.androidx.core.ktx)
  implementation(libs.androidx.lifecycle.runtime.ktx)
  implementation(libs.androidx.activity.compose)

  // Sign-in with Zitadel (OIDC, PKCE) in a Custom Tab
  implementation(libs.appauth)

  // Images from the API (the installation's logo, avatars)
  implementation(libs.coil.compose)

  // Notifications without push: the background check for new notices
  implementation(libs.androidx.work.runtime)

  // Checking in: the booking's QR for the reader, and reading the door's printed QR (ZXing, Apache-2.0)
  implementation(libs.zxing.core)
  implementation(libs.zxing.android)

  // The facilities on a map: OpenStreetMap's tiles, as on the web (MapLibre, BSD-2; no API key)
  implementation(libs.maplibre.android)

  // Arch Components
  implementation(libs.androidx.lifecycle.runtime.compose)
  implementation(libs.androidx.lifecycle.viewmodel.compose)

  // Compose
  implementation(libs.androidx.compose.ui)
  implementation(libs.androidx.compose.ui.tooling.preview)
  implementation(libs.androidx.compose.material3)
  // Tooling
  debugImplementation(libs.androidx.compose.ui.tooling)
  // Instrumented tests
  androidTestImplementation(libs.androidx.compose.ui.test.junit4)
  debugImplementation(libs.androidx.compose.ui.test.manifest)

  // Local tests: jUnit, coroutines, Android runner
  testImplementation(libs.junit)
  testImplementation(libs.kotlinx.coroutines.test)

  // Instrumented tests: jUnit rules and runners
  androidTestImplementation(libs.androidx.test.core)
  androidTestImplementation(libs.androidx.test.ext.junit)
  androidTestImplementation(libs.androidx.test.runner)
  androidTestImplementation(libs.androidx.test.espresso.core)

}
