package gal.tribo.librequenda.ui.search

import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import gal.tribo.librequenda.data.DayFacility
import gal.tribo.librequenda.data.GeoPoint
import gal.tribo.librequenda.ui.theme.Lq
import org.maplibre.android.MapLibre
import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapLibreMap
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style
import org.maplibre.android.style.expressions.Expression
import org.maplibre.android.style.layers.CircleLayer
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.layers.SymbolLayer
import org.maplibre.android.style.sources.GeoJsonOptions
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.Point

// OpenStreetMap's own tiles, as the web's map (web/src/lib/maps.ts): no API key, no account
private const val STYLE = """{
  "version": 8,
  "sources": {"osm": {"type": "raster", "tiles": ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"], "tileSize": 256, "maxzoom": 19,
    "attribution": "© OpenStreetMap contributors"}},
  "layers": [{"id": "osm", "type": "raster", "source": "osm"}]
}"""

// Galicia, before the facilities frame the map
private val DEFAULT_CENTER = LatLng(42.75, -8.0)

// zoomed out, close pins gather in one bubble with how many; from this zoom on, every facility has its own
private const val CLUSTER_MAX_ZOOM = 11

private const val SOURCE = "facilities"
private const val ME_SOURCE = "me"
private const val PINS = "pins"
private const val SELECTED = "pin-selected"
private const val CLUSTERS = "clusters"

/**
 * The facilities the search found, on OpenStreetMap: a pin each (in bubbles with how
 * many when zoomed out; a bubble zooms in when tapped), the phone's place when known.
 * Tapping a pin picks its facility; the map frames them all whenever the set changes.
 */
@Composable
fun FacilityMap(
    facilities: List<DayFacility>,
    selected: String?, // a facility id: its pin stands out
    me: GeoPoint?,
    label: String, // what the map is, for screen readers
    onPick: (DayFacility) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val density = LocalDensity.current.density
    val primary = Lq.colors.primary.toArgb()
    val ink = Lq.colors.primaryForeground.toArgb()
    val pinned = facilities.filter { it.latitude != null && it.longitude != null }
    val byId = pinned.associateBy { it.id }
    val currentById by rememberUpdatedState(byId)
    val pick by rememberUpdatedState(onPick)

    // the native map view, created once and told about the screen's lifecycle
    val mapView = remember {
        MapLibre.getInstance(context)
        MapView(context).apply { onCreate(null) }
    }
    var map by remember { mutableStateOf<MapLibreMap?>(null) }
    var style by remember { mutableStateOf<Style?>(null) }
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    DisposableEffect(lifecycle) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> mapView.onStart()
                Lifecycle.Event.ON_RESUME -> mapView.onResume()
                Lifecycle.Event.ON_PAUSE -> mapView.onPause()
                Lifecycle.Event.ON_STOP -> mapView.onStop()
                else -> {}
            }
        }
        lifecycle.addObserver(observer)
        // already started when the map is shown: catch up
        if (lifecycle.currentState.isAtLeast(Lifecycle.State.STARTED)) mapView.onStart()
        if (lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)) mapView.onResume()
        onDispose {
            lifecycle.removeObserver(observer)
            mapView.onPause(); mapView.onStop(); mapView.onDestroy()
        }
    }

    // the style, the layers and the taps, once
    LaunchedEffect(mapView) {
        mapView.getMapAsync { m ->
            m.uiSettings.isLogoEnabled = false // the attribution («i») stays: OpenStreetMap asks for it
            m.uiSettings.isRotateGesturesEnabled = false
            m.uiSettings.isTiltGesturesEnabled = false
            m.cameraPosition = CameraPosition.Builder().target(DEFAULT_CENTER).zoom(7.0).build()
            m.setStyle(Style.Builder().fromJson(STYLE)) { s ->
                s.addSource(GeoJsonSource(SOURCE, FeatureCollection.fromFeatures(emptyList()),
                    GeoJsonOptions().withCluster(true).withClusterMaxZoom(CLUSTER_MAX_ZOOM).withClusterRadius(50)))
                s.addSource(GeoJsonSource(ME_SOURCE, FeatureCollection.fromFeatures(emptyList())))
                val single = Expression.not(Expression.has("point_count"))
                s.addLayer(CircleLayer("me", ME_SOURCE).withProperties(
                    PropertyFactory.circleRadius(7f), PropertyFactory.circleColor(0xFF2563EB.toInt()),
                    PropertyFactory.circleStrokeColor(Color.White.toArgb()), PropertyFactory.circleStrokeWidth(3f),
                ))
                s.addLayer(CircleLayer(PINS, SOURCE).withFilter(single).withProperties(
                    PropertyFactory.circleRadius(9f), PropertyFactory.circleColor(primary),
                    PropertyFactory.circleStrokeColor(Color.White.toArgb()), PropertyFactory.circleStrokeWidth(2.5f),
                ))
                s.addLayer(CircleLayer(SELECTED, SOURCE).withFilter(Expression.eq(Expression.literal(0), 1)).withProperties(
                    PropertyFactory.circleRadius(13f), PropertyFactory.circleColor(primary),
                    PropertyFactory.circleStrokeColor(Color.White.toArgb()), PropertyFactory.circleStrokeWidth(4f),
                ))
                // a bubble is an image with its count drawn in (a raster map has no fonts for text layers)
                s.addLayer(SymbolLayer(CLUSTERS, SOURCE).withFilter(Expression.has("point_count")).withProperties(
                    PropertyFactory.iconImage(Expression.concat(Expression.literal("cluster-"), Expression.toString(Expression.get("point_count")))),
                    PropertyFactory.iconAllowOverlap(true),
                    PropertyFactory.iconIgnorePlacement(true),
                ))
                style = s
            }
            m.addOnMapClickListener { at ->
                val p = m.projection.toScreenLocation(at)
                val r = 24 * density
                val box = RectF(p.x - r, p.y - r, p.x + r, p.y + r)
                val cluster = m.queryRenderedFeatures(box, CLUSTERS).firstOrNull()
                if (cluster != null) {
                    val point = cluster.geometry() as? Point
                    if (point != null) m.animateCamera(CameraUpdateFactory.newLatLngZoom(LatLng(point.latitude(), point.longitude()), m.cameraPosition.zoom + 2), 350)
                    return@addOnMapClickListener true
                }
                val pin = m.queryRenderedFeatures(box, SELECTED, PINS).firstOrNull()
                val f = pin?.getStringProperty("id")?.let { currentById[it] } ?: return@addOnMapClickListener false
                pick(f)
                true
            }
            map = m
        }
    }

    // the pins, and the map framing them, whenever the set changes
    val shownKey = pinned.joinToString(",") { it.id }
    LaunchedEffect(style, shownKey) {
        val s = style ?: return@LaunchedEffect
        val m = map ?: return@LaunchedEffect
        // a bubble can hold at most every pin: one image for each count it may show
        for (n in 2..pinned.size) if (s.getImage("cluster-$n") == null) s.addImage("cluster-$n", bubble(n, primary, ink, density))
        s.getSourceAs<GeoJsonSource>(SOURCE)?.setGeoJson(FeatureCollection.fromFeatures(pinned.map { f ->
            Feature.fromGeometry(Point.fromLngLat(f.longitude!!, f.latitude!!)).apply { addStringProperty("id", f.id) }
        }))
        val points = pinned.map { LatLng(it.latitude!!, it.longitude!!) }
        val padding = (48 * density).toInt()
        when {
            points.isEmpty() -> {}
            points.distinct().size == 1 -> m.animateCamera(CameraUpdateFactory.newLatLngZoom(points[0], 14.0), 400)
            else -> m.animateCamera(CameraUpdateFactory.newLatLngBounds(LatLngBounds.Builder().includes(points).build(), padding), 400)
        }
    }

    // the chosen pin, drawn bigger
    LaunchedEffect(style, selected) {
        val layer = style?.getLayerAs<CircleLayer>(SELECTED) ?: return@LaunchedEffect
        layer.setFilter(Expression.all(Expression.not(Expression.has("point_count")), Expression.eq(Expression.get("id"), Expression.literal(selected ?: ""))))
    }

    // the phone's place
    LaunchedEffect(style, me) {
        val source = style?.getSourceAs<GeoJsonSource>(ME_SOURCE) ?: return@LaunchedEffect
        source.setGeoJson(FeatureCollection.fromFeatures(listOfNotNull(me?.let { Feature.fromGeometry(Point.fromLngLat(it.longitude, it.latitude)) })))
    }

    AndroidView(factory = { mapView }, modifier = modifier.semantics { contentDescription = label })
}

/** A bubble for [count] pins close together: the brand's blue, a white ring, the count. */
private fun bubble(count: Int, fill: Int, ink: Int, density: Float): Bitmap {
    val size = ((if (count < 10) 36 else if (count < 100) 42 else 48) * density).toInt()
    val bitmap = Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888)
    val canvas = Canvas(bitmap)
    val paint = Paint(Paint.ANTI_ALIAS_FLAG)
    val c = size / 2f
    paint.color = android.graphics.Color.WHITE
    canvas.drawCircle(c, c, c, paint)
    paint.color = fill
    canvas.drawCircle(c, c, c - 3 * density, paint)
    paint.color = ink
    paint.textAlign = Paint.Align.CENTER
    paint.typeface = Typeface.DEFAULT_BOLD
    paint.textSize = 15 * density
    canvas.drawText(count.toString(), c, c - (paint.descent() + paint.ascent()) / 2, paint)
    return bitmap
}
