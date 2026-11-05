#!/bin/sh
# The builder's entry point (install.sh app runs it). In:
#   /src         the LibreQuenda source (read-only)
#   /in/logo     optional: the app's square logo
#   /in/release.jks  the signing key
#   env          APP_ID, APP_NAME, API_URL, ICON_BACKGROUND, VERSION_NAME, VERSION_CODE,
#                ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD, OUT_OWNER
# Out: /out/<APP_ID>.apk (to install directly) and /out/<APP_ID>.aab (for Google Play).
set -eu

work=/tmp/work
rm -rf "$work"; mkdir -p "$work"
cp -r /src/app "$work/app"
brand="$work/brand"
mkdir -p "$brand/values"

# the name under the icon: escaped for an Android string resource
name=$(printf '%s' "$APP_NAME" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' -e "s/'/\\\\'/g" -e 's/"/\\"/g')
cat > "$brand/values/brand_strings.xml" <<XML
<?xml version="1.0" encoding="utf-8"?>
<resources>
  <string name="app_name" translatable="false">$name</string>
</resources>
XML
if [ -s /in/logo ]; then
	icons.sh /in/logo "${ICON_BACKGROUND:-#FFFFFF}" "$brand"
fi

cd "$work/app"
export ANDROID_KEYSTORE_FILE=/in/release.jks
./gradlew --no-daemon --console=plain -q assembleRelease bundleRelease \
	-PapplicationId="$APP_ID" -PapiBaseUrl="$API_URL" -Pbrand="$brand" \
	-PversionName="$VERSION_NAME" -PversionCode="$VERSION_CODE"

cp app/build/outputs/apk/release/app-release.apk "/out/$APP_ID.apk"
cp app/build/outputs/bundle/release/app-release.aab "/out/$APP_ID.aab"
[ -n "${OUT_OWNER:-}" ] && chown "$OUT_OWNER" "/out/$APP_ID.apk" "/out/$APP_ID.aab"
echo "built /out/$APP_ID.apk and /out/$APP_ID.aab"
