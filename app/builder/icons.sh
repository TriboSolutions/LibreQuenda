#!/bin/sh
# icons.sh <logo> <background colour> <res dir>: the launcher icons from a square logo
# (PNG, JPG or WebP): legacy and round icons, and the adaptive icon for Android 8+.
set -eu
logo=$1 bg=$2 res=$3
cmd=convert; command -v magick >/dev/null 2>&1 && cmd=magick

for d in mdpi:48 hdpi:72 xhdpi:96 xxhdpi:144 xxxhdpi:192; do
	density=${d%%:*} size=${d##*:}
	dir="$res/mipmap-$density"; mkdir -p "$dir"
	inner=$((size * 3 / 4))
	# legacy: the logo on a square of the background colour
	$cmd -size "${size}x${size}" "xc:$bg" \( "$logo" -background none -resize "${inner}x${inner}" \) \
		-gravity center -composite "$dir/ic_launcher.png"
	# round: on a circle
	half=$((size / 2))
	$cmd -size "${size}x${size}" xc:none -fill "$bg" -draw "circle $half,$half $half,0" \
		\( "$logo" -background none -resize "${inner}x${inner}" \) -gravity center -composite "$dir/ic_launcher_round.png"
	# adaptive foreground: 108dp, the logo inside the 66dp safe zone
	full=$((size * 108 / 48)); fg=$((full * 58 / 100))
	$cmd "$logo" -background none -resize "${fg}x${fg}" -gravity center -extent "${full}x${full}" \
		"$dir/ic_launcher_brand_foreground.png"
done

mkdir -p "$res/mipmap-anydpi-v26" "$res/values"
for name in ic_launcher ic_launcher_round; do
	cat > "$res/mipmap-anydpi-v26/$name.xml" <<XML
<?xml version="1.0" encoding="utf-8"?>
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
  <background android:drawable="@color/ic_launcher_brand_background"/>
  <foreground android:drawable="@mipmap/ic_launcher_brand_foreground"/>
</adaptive-icon>
XML
done
cat > "$res/values/brand_colors.xml" <<XML
<?xml version="1.0" encoding="utf-8"?>
<resources>
  <color name="ic_launcher_brand_background">$bg</color>
</resources>
XML
