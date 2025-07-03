# Geographic Data Types: Complete Guide

Geographic and spatial data handling is essential for location-based applications. This guide covers spatial data types, coordinate systems, indexing strategies, and common geographic operations.

## Table of Contents
- [Spatial Data Types](#spatial-data-types)
- [Coordinate Reference Systems](#coordinate-reference-systems)
- [Spatial Indexing](#spatial-indexing)
- [Common Geographic Operations](#common-geographic-operations)
- [Location-Based Queries](#location-based-queries)
- [Performance Optimization](#performance-optimization)
- [Best Practices](#best-practices)

## Spatial Data Types

### PostgreSQL with PostGIS

```sql
-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Spatial data types table
CREATE TABLE geographic_entities (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- Point locations (longitude, latitude)
    location GEOMETRY(POINT, 4326),         -- WGS84 coordinate system
    location_3d GEOMETRY(POINTZ, 4326),     -- 3D point with elevation
    
    -- Linear features
    route GEOMETRY(LINESTRING, 4326),       -- Roads, paths, routes
    boundary GEOMETRY(POLYGON, 4326),       -- Areas, regions, zones
    
    -- Complex geometries
    multipoint GEOMETRY(MULTIPOINT, 4326),   -- Multiple discrete locations
    multiline GEOMETRY(MULTILINESTRING, 4326), -- Complex routes
    multipoly GEOMETRY(MULTIPOLYGON, 4326),  -- Disconnected areas
    
    -- Mixed geometry types
    mixed_geom GEOMETRY(GEOMETRYCOLLECTION, 4326),
    
    -- Geographic types (uses spherical calculations)
    geo_location GEOGRAPHY(POINT, 4326),    -- More accurate for distances
    geo_area GEOGRAPHY(POLYGON, 4326),
    
    -- Metadata
    elevation DECIMAL(8,2), -- meters above sea level
    accuracy_radius DECIMAL(10,2), -- GPS accuracy in meters
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Real-world examples
INSERT INTO geographic_entities (name, location, elevation) VALUES
('Statue of Liberty', ST_SetSRID(ST_MakePoint(-74.0445, 40.6892), 4326), 93),
('Eiffel Tower', ST_SetSRID(ST_MakePoint(2.2945, 48.8584), 4326), 330),
('Sydney Opera House', ST_SetSRID(ST_MakePoint(151.2153, -33.8568), 4326), 65);
```

### PostgreSQL Built-in Spatial Support

```sql
-- Basic spatial support without PostGIS
CREATE TABLE basic_locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- Point type (basic PostgreSQL)
    coordinates POINT,
    
    -- Separate latitude/longitude columns
    latitude DECIMAL(10, 8) CHECK (latitude >= -90 AND latitude <= 90),
    longitude DECIMAL(11, 8) CHECK (longitude >= -180 AND longitude <= 180),
    
    -- Earth distance calculations (requires earthdistance extension)
    earth_location EARTH,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enable earth distance extension for basic geographic calculations
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

-- Convert lat/lng to earth coordinates
UPDATE basic_locations 
SET earth_location = ll_to_earth(latitude, longitude)
WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
```

### MySQL Spatial Data Types

```sql
-- MySQL spatial data types (MySQL 5.7+)
CREATE TABLE mysql_spatial (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    
    -- Point geometry
    location POINT NOT NULL,
    
    -- Linear geometry
    route LINESTRING,
    
    -- Polygon geometry
    area POLYGON,
    
    -- Multi-geometry types
    multiple_points MULTIPOINT,
    multiple_lines MULTILINESTRING,
    multiple_areas MULTIPOLYGON,
    
    -- Generic geometry
    geom GEOMETRY,
    
    -- Separate coordinate columns for compatibility
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Spatial index
    SPATIAL INDEX idx_location (location),
    
    -- Constraints
    CONSTRAINT valid_coordinates CHECK (
        latitude BETWEEN -90 AND 90 AND
        longitude BETWEEN -180 AND 180
    )
);

-- Insert spatial data in MySQL
INSERT INTO mysql_spatial (name, location, latitude, longitude) VALUES
('Central Park', ST_GeomFromText('POINT(-73.9665 40.7812)', 4326), 40.7812, -73.9665),
('Golden Gate Bridge', ST_GeomFromText('POINT(-122.4783 37.8199)', 4326), 37.8199, -122.4783);
```

## Coordinate Reference Systems

### Understanding SRID (Spatial Reference ID)

```sql
-- Common coordinate reference systems
-- SRID 4326: WGS84 (GPS coordinates) - most common
-- SRID 3857: Web Mercator (used by Google Maps, OpenStreetMap)
-- SRID 2154: RGF93 / Lambert-93 (France)
-- SRID 3826: TWD97 / TM2 zone 121 (Taiwan)

-- Working with different coordinate systems
CREATE TABLE coordinate_examples (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- WGS84 - Geographic coordinates (longitude, latitude)
    wgs84_location GEOMETRY(POINT, 4326),
    
    -- Web Mercator - Projected coordinates (meters)
    mercator_location GEOMETRY(POINT, 3857),
    
    -- Local coordinate system example
    local_coords GEOMETRY(POINT, 2154)
);

-- Transform between coordinate systems
INSERT INTO coordinate_examples (name, wgs84_location) VALUES
('Paris Center', ST_SetSRID(ST_MakePoint(2.3522, 48.8566), 4326));

-- Convert to Web Mercator
UPDATE coordinate_examples 
SET mercator_location = ST_Transform(wgs84_location, 3857)
WHERE wgs84_location IS NOT NULL;

-- Convert to local system (France Lambert-93)
UPDATE coordinate_examples 
SET local_coords = ST_Transform(wgs84_location, 2154)
WHERE wgs84_location IS NOT NULL AND name = 'Paris Center';

-- Query with coordinate system information
SELECT 
    name,
    ST_AsText(wgs84_location) as wgs84_text,
    ST_AsText(mercator_location) as mercator_text,
    ST_SRID(wgs84_location) as wgs84_srid,
    ST_SRID(mercator_location) as mercator_srid
FROM coordinate_examples;
```

## Spatial Indexing

### PostGIS Spatial Indexes

```sql
-- Create spatial indexes for performance
CREATE TABLE indexed_locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    location GEOMETRY(POINT, 4326),
    service_area GEOMETRY(POLYGON, 4326)
);

-- GiST index for spatial queries (most common)
CREATE INDEX idx_locations_geom ON indexed_locations USING GIST (location);
CREATE INDEX idx_service_areas_geom ON indexed_locations USING GIST (service_area);

-- SP-GiST index for specific geometric types
CREATE INDEX idx_locations_spgist ON indexed_locations USING SPGIST (location);

-- Composite indexes for spatial + attribute queries
CREATE INDEX idx_locations_category_geom ON indexed_locations USING GIST (category, location);

-- Partial spatial index for specific categories
CREATE INDEX idx_restaurants_location ON indexed_locations USING GIST (location)
WHERE category = 'restaurant';
```

### Index Performance Analysis

```sql
-- Analyze spatial index effectiveness
EXPLAIN (ANALYZE, BUFFERS) 
SELECT name, ST_Distance(location, ST_SetSRID(ST_MakePoint(-74.006, 40.7128), 4326)) as distance
FROM indexed_locations
WHERE ST_DWithin(location, ST_SetSRID(ST_MakePoint(-74.006, 40.7128), 4326), 0.01)
ORDER BY location <-> ST_SetSRID(ST_MakePoint(-74.006, 40.7128), 4326)
LIMIT 10;

-- Monitor index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE indexname LIKE '%geom%';
```

## Common Geographic Operations

### Distance Calculations

```sql
-- Distance calculations with PostGIS
CREATE OR REPLACE FUNCTION calculate_distances()
RETURNS TABLE(
    name1 TEXT,
    name2 TEXT,
    distance_meters NUMERIC,
    distance_km NUMERIC,
    distance_miles NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.name as name1,
        b.name as name2,
        ST_Distance(a.geo_location, b.geo_location) as distance_meters,
        ROUND(ST_Distance(a.geo_location, b.geo_location) / 1000, 2) as distance_km,
        ROUND(ST_Distance(a.geo_location, b.geo_location) * 0.000621371, 2) as distance_miles
    FROM geographic_entities a
    CROSS JOIN geographic_entities b
    WHERE a.id < b.id  -- Avoid duplicate pairs
      AND a.geo_location IS NOT NULL 
      AND b.geo_location IS NOT NULL;
END;
$$ LANGUAGE plpgsql;

-- Basic earth distance (without PostGIS)
SELECT 
    name,
    earth_distance(
        ll_to_earth(40.7128, -74.0060),  -- New York City
        ll_to_earth(latitude, longitude)
    ) as distance_meters
FROM basic_locations
WHERE latitude IS NOT NULL AND longitude IS NOT NULL
ORDER BY distance_meters;
```

### Geometric Calculations

```sql
-- Area and perimeter calculations
SELECT 
    name,
    ST_Area(boundary) as area_square_meters,
    ST_Area(boundary) / 1000000 as area_square_km,
    ST_Perimeter(boundary) as perimeter_meters,
    ST_Perimeter(boundary) / 1000 as perimeter_km
FROM geographic_entities
WHERE boundary IS NOT NULL;

-- Centroid and bounding box
SELECT 
    name,
    ST_AsText(ST_Centroid(boundary)) as centroid,
    ST_AsText(ST_Envelope(boundary)) as bounding_box,
    ST_XMin(boundary) as min_longitude,
    ST_XMax(boundary) as max_longitude,
    ST_YMin(boundary) as min_latitude,
    ST_YMax(boundary) as max_latitude
FROM geographic_entities
WHERE boundary IS NOT NULL;

-- Point in polygon tests
SELECT 
    p.name as point_name,
    poly.name as polygon_name,
    ST_Contains(poly.boundary, p.location) as is_inside
FROM geographic_entities p
CROSS JOIN geographic_entities poly
WHERE p.location IS NOT NULL 
  AND poly.boundary IS NOT NULL
  AND ST_Contains(poly.boundary, p.location);
```

### Spatial Relationships

```sql
-- Comprehensive spatial relationship queries
CREATE OR REPLACE FUNCTION analyze_spatial_relationships()
RETURNS TABLE(
    entity1 TEXT,
    entity2 TEXT,
    relationship TEXT,
    distance_meters NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH spatial_pairs AS (
        SELECT 
            a.name as name1,
            b.name as name2,
            a.location as geom1,
            b.location as geom2,
            a.boundary as poly1,
            b.boundary as poly2
        FROM geographic_entities a
        CROSS JOIN geographic_entities b
        WHERE a.id != b.id
    )
    SELECT 
        name1,
        name2,
        CASE 
            WHEN ST_Equals(geom1, geom2) THEN 'equal'
            WHEN ST_Contains(poly1, geom2) THEN 'contains'
            WHEN ST_Within(geom1, poly2) THEN 'within'
            WHEN ST_Intersects(geom1, geom2) THEN 'intersects'
            WHEN ST_Touches(poly1, poly2) THEN 'touches'
            WHEN ST_Disjoint(geom1, geom2) THEN 'disjoint'
            ELSE 'other'
        END as relationship,
        COALESCE(ST_Distance(geom1, geom2), 0) as distance_meters
    FROM spatial_pairs
    WHERE geom1 IS NOT NULL AND geom2 IS NOT NULL;
END;
$$ LANGUAGE plpgsql;
```

## Location-Based Queries

### Proximity Searches

```sql
-- Find nearby locations
CREATE OR REPLACE FUNCTION find_nearby_locations(
    search_lat DECIMAL,
    search_lng DECIMAL,
    radius_km DECIMAL DEFAULT 10,
    category_filter TEXT DEFAULT NULL,
    limit_count INTEGER DEFAULT 50
) RETURNS TABLE(
    id INTEGER,
    name TEXT,
    category TEXT,
    distance_km NUMERIC,
    latitude DECIMAL,
    longitude DECIMAL
) AS $$
DECLARE
    search_point GEOMETRY;
    radius_meters DECIMAL;
BEGIN
    search_point := ST_SetSRID(ST_MakePoint(search_lng, search_lat), 4326);
    radius_meters := radius_km * 1000;
    
    RETURN QUERY
    SELECT 
        l.id,
        l.name,
        l.category,
        ROUND(ST_Distance(l.geo_location, search_point::GEOGRAPHY) / 1000, 2) as distance_km,
        ST_Y(l.location) as latitude,
        ST_X(l.location) as longitude
    FROM indexed_locations l
    WHERE ST_DWithin(l.location, search_point, radius_meters / 111320) -- Approximate degree conversion
      AND (category_filter IS NULL OR l.category = category_filter)
    ORDER BY l.location <-> search_point  -- Use index for ordering
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Usage examples
SELECT * FROM find_nearby_locations(40.7128, -74.0060, 5.0, 'restaurant', 10);
```

### Geofencing and Area Queries

```sql
-- Geofencing implementation
CREATE TABLE geofences (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    area GEOMETRY(POLYGON, 4326) NOT NULL,
    fence_type TEXT CHECK (fence_type IN ('inclusion', 'exclusion', 'alert')),
    
    -- Trigger settings
    entry_trigger BOOLEAN DEFAULT TRUE,
    exit_trigger BOOLEAN DEFAULT TRUE,
    dwell_time_seconds INTEGER DEFAULT 0,
    
    -- Active status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Spatial index
    INDEX USING GIST (area)
);

-- Geofence entry/exit tracking
CREATE TABLE geofence_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    geofence_id INTEGER REFERENCES geofences(id),
    event_type TEXT CHECK (event_type IN ('enter', 'exit', 'dwell')),
    
    -- Event details
    user_location GEOMETRY(POINT, 4326) NOT NULL,
    event_time TIMESTAMPTZ DEFAULT NOW(),
    
    -- Additional context
    speed_kmh DECIMAL(5,2),
    accuracy_meters DECIMAL(6,2),
    device_id TEXT
);

-- Function to check geofence triggers
CREATE OR REPLACE FUNCTION check_geofences(
    p_user_id INTEGER,
    p_latitude DECIMAL,
    p_longitude DECIMAL
) RETURNS VOID AS $$
DECLARE
    user_point GEOMETRY;
    fence_record RECORD;
BEGIN
    user_point := ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326);
    
    -- Check all active geofences
    FOR fence_record IN 
        SELECT id, name, area, fence_type, entry_trigger
        FROM geofences 
        WHERE is_active = TRUE
          AND ST_Contains(area, user_point)
    LOOP
        -- Insert geofence event
        INSERT INTO geofence_events (
            user_id, geofence_id, event_type, user_location
        ) VALUES (
            p_user_id, fence_record.id, 'enter', user_point
        );
        
        -- Additional logic for notifications, etc.
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

### Route and Navigation Queries

```sql
-- Route analysis and waypoint management
CREATE TABLE routes (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    route_geometry GEOMETRY(LINESTRING, 4326) NOT NULL,
    
    -- Route metadata
    total_distance_meters DECIMAL(10,2),
    estimated_duration_seconds INTEGER,
    difficulty_level INTEGER CHECK (difficulty_level BETWEEN 1 AND 5),
    
    -- Route classification
    route_type TEXT CHECK (route_type IN ('driving', 'walking', 'cycling', 'public_transit')),
    surface_type TEXT CHECK (surface_type IN ('paved', 'gravel', 'trail', 'mixed')),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Waypoints along routes
CREATE TABLE route_waypoints (
    id SERIAL PRIMARY KEY,
    route_id INTEGER REFERENCES routes(id),
    sequence_number INTEGER NOT NULL,
    
    -- Waypoint location
    location GEOMETRY(POINT, 4326) NOT NULL,
    
    -- Waypoint details
    name TEXT,
    waypoint_type TEXT CHECK (waypoint_type IN ('start', 'end', 'checkpoint', 'poi', 'turn')),
    instructions TEXT,
    
    -- Distance along route
    distance_from_start DECIMAL(10,2),
    
    UNIQUE(route_id, sequence_number)
);

-- Route intersection queries
SELECT 
    r1.name as route1,
    r2.name as route2,
    ST_AsText(ST_Intersection(r1.route_geometry, r2.route_geometry)) as intersection_point
FROM routes r1
CROSS JOIN routes r2
WHERE r1.id < r2.id
  AND ST_Intersects(r1.route_geometry, r2.route_geometry)
  AND ST_GeometryType(ST_Intersection(r1.route_geometry, r2.route_geometry)) = 'ST_Point';
```

## Performance Optimization

### Spatial Query Optimization

```sql
-- Optimized proximity query with bounding box pre-filter
CREATE OR REPLACE FUNCTION optimized_nearby_search(
    center_lat DECIMAL,
    center_lng DECIMAL,
    radius_km DECIMAL
) RETURNS TABLE(
    id INTEGER,
    name TEXT,
    distance_km DECIMAL
) AS $$
DECLARE
    center_point GEOMETRY;
    bbox GEOMETRY;
    radius_degrees DECIMAL;
BEGIN
    center_point := ST_SetSRID(ST_MakePoint(center_lng, center_lat), 4326);
    
    -- Approximate degree conversion (faster than exact calculation)
    radius_degrees := radius_km / 111.32;
    
    -- Create bounding box for initial filter
    bbox := ST_Envelope(ST_Buffer(center_point, radius_degrees));
    
    RETURN QUERY
    SELECT 
        l.id,
        l.name,
        ROUND(ST_Distance(l.geo_location, center_point::GEOGRAPHY) / 1000, 2) as distance_km
    FROM indexed_locations l
    WHERE l.location && bbox  -- Fast bounding box intersection
      AND ST_DWithin(l.geo_location, center_point::GEOGRAPHY, radius_km * 1000)  -- Exact distance
    ORDER BY l.location <-> center_point;
END;
$$ LANGUAGE plpgsql;
```

### Clustering and Aggregation

```sql
-- Spatial clustering for map display
CREATE OR REPLACE FUNCTION cluster_locations_for_zoom(
    bbox_min_lng DECIMAL,
    bbox_min_lat DECIMAL, 
    bbox_max_lng DECIMAL,
    bbox_max_lat DECIMAL,
    zoom_level INTEGER
) RETURNS TABLE(
    cluster_id INTEGER,
    center_lng DECIMAL,
    center_lat DECIMAL,
    point_count INTEGER,
    avg_category TEXT
) AS $$
DECLARE
    grid_size DECIMAL;
    bbox GEOMETRY;
BEGIN
    -- Calculate grid size based on zoom level
    grid_size := POWER(2, 20 - zoom_level) * 0.00001;
    
    -- Create bounding box
    bbox := ST_MakeEnvelope(bbox_min_lng, bbox_min_lat, bbox_max_lng, bbox_max_lat, 4326);
    
    RETURN QUERY
    WITH clustered AS (
        SELECT 
            FLOOR(ST_X(location) / grid_size) as grid_x,
            FLOOR(ST_Y(location) / grid_y) as grid_y,
            location,
            category
        FROM indexed_locations
        WHERE location && bbox
    ),
    clusters AS (
        SELECT 
            ROW_NUMBER() OVER() as cluster_id,
            grid_x,
            grid_y,
            AVG(ST_X(location)) as center_lng,
            AVG(ST_Y(location)) as center_lat,
            COUNT(*) as point_count,
            MODE() WITHIN GROUP (ORDER BY category) as avg_category
        FROM clustered
        GROUP BY grid_x, grid_y
    )
    SELECT 
        c.cluster_id,
        c.center_lng,
        c.center_lat,
        c.point_count,
        c.avg_category
    FROM clusters c;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Choose Appropriate Data Types

```sql
-- ✅ Good: Use appropriate spatial types
CREATE TABLE location_best_practices (
    id SERIAL PRIMARY KEY,
    
    -- Use GEOGRAPHY for accurate distance calculations
    precise_location GEOGRAPHY(POINT, 4326),
    
    -- Use GEOMETRY for spatial operations and performance
    indexed_location GEOMETRY(POINT, 4326),
    
    -- Include separate lat/lng for compatibility
    latitude DECIMAL(10, 8) CHECK (latitude BETWEEN -90 AND 90),
    longitude DECIMAL(11, 8) CHECK (longitude BETWEEN -180 AND 180),
    
    -- Store accuracy and metadata
    gps_accuracy_meters DECIMAL(6,2),
    elevation_meters DECIMAL(8,2),
    
    -- Temporal information
    location_timestamp TIMESTAMPTZ DEFAULT NOW()
);

-- ❌ Avoid: Storing coordinates as strings
CREATE TABLE bad_locations (
    id SERIAL PRIMARY KEY,
    coordinates TEXT  -- Hard to query efficiently
);
```

### 2. Implement Proper Indexing

```sql
-- Comprehensive indexing strategy
CREATE TABLE well_indexed_locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    location GEOMETRY(POINT, 4326),
    service_area GEOMETRY(POLYGON, 4326),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Primary spatial index
CREATE INDEX idx_location_geom ON well_indexed_locations USING GIST (location);

-- Composite indexes for common query patterns
CREATE INDEX idx_active_category_location ON well_indexed_locations 
USING GIST (location) WHERE is_active = TRUE;

CREATE INDEX idx_category_location ON well_indexed_locations 
USING GIST (category, location);

-- Partial indexes for specific use cases
CREATE INDEX idx_restaurants_location ON well_indexed_locations 
USING GIST (location) WHERE category = 'restaurant' AND is_active = TRUE;
```

### 3. Data Validation and Constraints

```sql
-- Comprehensive validation
CREATE TABLE validated_geo_data (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- Validated coordinates
    location GEOMETRY(POINT, 4326) NOT NULL,
    
    -- Ensure valid coordinate ranges
    CONSTRAINT valid_location CHECK (
        ST_X(location) BETWEEN -180 AND 180 AND
        ST_Y(location) BETWEEN -90 AND 90
    ),
    
    -- Ensure geometry is valid
    CONSTRAINT valid_geometry CHECK (ST_IsValid(location)),
    
    -- Business logic constraints
    altitude_meters DECIMAL(8,2) CHECK (altitude_meters BETWEEN -11000 AND 9000),
    accuracy_meters DECIMAL(6,2) CHECK (accuracy_meters > 0)
);

-- Trigger for additional validation
CREATE OR REPLACE FUNCTION validate_geo_data()
RETURNS TRIGGER AS $$
BEGIN
    -- Ensure coordinates are within reasonable bounds for Earth
    IF ST_X(NEW.location) < -180 OR ST_X(NEW.location) > 180 OR
       ST_Y(NEW.location) < -90 OR ST_Y(NEW.location) > 90 THEN
        RAISE EXCEPTION 'Invalid coordinates: longitude must be between -180 and 180, latitude between -90 and 90';
    END IF;
    
    -- Validate geometry
    IF NOT ST_IsValid(NEW.location) THEN
        RAISE EXCEPTION 'Invalid geometry';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER geo_validation_trigger
    BEFORE INSERT OR UPDATE ON validated_geo_data
    FOR EACH ROW EXECUTE FUNCTION validate_geo_data();
```

### 4. Monitor Performance

```sql
-- Performance monitoring queries
CREATE VIEW spatial_performance_stats AS
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    avg_width,
    correlation
FROM pg_stats 
WHERE atttypid IN (
    'geometry'::regtype, 
    'geography'::regtype,
    'point'::regtype
);

-- Query plan analysis for spatial queries
CREATE OR REPLACE FUNCTION analyze_spatial_query_performance()
RETURNS TABLE(query_plan TEXT) AS $$
BEGIN
    RETURN QUERY
    SELECT query_plan
    FROM (
        VALUES 
        ('EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM indexed_locations WHERE ST_DWithin(location, ST_SetSRID(ST_MakePoint(-74, 40.7), 4326), 0.01)'),
        ('EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM indexed_locations ORDER BY location <-> ST_SetSRID(ST_MakePoint(-74, 40.7), 4326) LIMIT 10')
    ) AS plans(query_plan);
END;
$$ LANGUAGE plpgsql;
```

## Conclusion

Effective geographic data handling requires:

1. **Type Selection**: Use PostGIS GEOGRAPHY for accuracy, GEOMETRY for performance
2. **Coordinate Systems**: Understand SRID and choose appropriate reference systems
3. **Indexing**: Implement spatial indexes (GiST) for geographic queries
4. **Validation**: Ensure coordinate bounds and geometry validity
5. **Optimization**: Use bounding box pre-filters and appropriate clustering
6. **Standards**: Follow OGC standards and PostGIS best practices
7. **Performance**: Monitor spatial query performance and optimize accordingly

The key is balancing accuracy requirements with performance needs while maintaining data integrity and following established spatial data standards.
