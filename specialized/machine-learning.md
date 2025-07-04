# Machine Learning Data Storage

This guide covers database design patterns for storing machine learning model outputs, training data, and metadata.

## Model Predictions and Scores

### Probability Scores vs Binary Classifications

Store probability scores rather than binary classifications for better flexibility:

```sql
-- Good: Store probability scores
CREATE TABLE spam_predictions (
    id SERIAL PRIMARY KEY,
    email_id UUID NOT NULL,
    spam_probability DECIMAL(5,4) NOT NULL CHECK (spam_probability >= 0 AND spam_probability <= 1),
    model_version VARCHAR(50) NOT NULL,
    prediction_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confidence_score DECIMAL(5,4),
    features JSONB,
    
    FOREIGN KEY (email_id) REFERENCES emails(id)
);

-- Bad: Only store binary results
CREATE TABLE spam_predictions_bad (
    id SERIAL PRIMARY KEY,
    email_id UUID NOT NULL,
    is_spam BOOLEAN NOT NULL  -- Lost information!
);
```

**Benefits of probability scores:**
- Adjustable thresholds for different use cases
- Better model evaluation and monitoring
- Ability to implement confidence-based filtering
- Support for A/B testing different cutoff points

### Model Versioning and Metadata

Track model versions and performance metrics:

```sql
CREATE TABLE ml_models (
    id SERIAL PRIMARY KEY,
    model_name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    algorithm VARCHAR(50),
    training_date TIMESTAMPTZ NOT NULL,
    performance_metrics JSONB,
    model_parameters JSONB,
    is_active BOOLEAN DEFAULT false,
    
    UNIQUE(model_name, version)
);

-- Link predictions to specific model versions
CREATE TABLE predictions (
    id SERIAL PRIMARY KEY,
    model_id INTEGER NOT NULL,
    input_data JSONB NOT NULL,
    prediction_value DECIMAL(10,4),
    prediction_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (model_id) REFERENCES ml_models(id)
);
```

## Feature Store Design

Store and version feature data for training and inference:

```sql
-- Feature definitions
CREATE TABLE feature_definitions (
    id SERIAL PRIMARY KEY,
    feature_name VARCHAR(100) NOT NULL UNIQUE,
    feature_type VARCHAR(50) NOT NULL, -- 'numerical', 'categorical', 'boolean', 'text'
    description TEXT,
    data_source VARCHAR(100),
    transformation_logic TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feature values with temporal support
CREATE TABLE feature_values (
    id SERIAL PRIMARY KEY,
    feature_id INTEGER NOT NULL,
    entity_id VARCHAR(100) NOT NULL, -- user_id, product_id, etc.
    feature_value JSONB NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    
    FOREIGN KEY (feature_id) REFERENCES feature_definitions(id),
    
    -- Ensure no overlapping periods for same entity/feature
    EXCLUDE USING gist (
        feature_id WITH =,
        entity_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);
```

## Training Data Management

### Dataset Versioning

Track training datasets and their lineage:

```sql
CREATE TABLE training_datasets (
    id SERIAL PRIMARY KEY,
    dataset_name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    data_source VARCHAR(100),
    feature_selection JSONB,
    data_filters JSONB,
    sample_size INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(dataset_name, version)
);

-- Link datasets to model training runs
CREATE TABLE model_training_runs (
    id SERIAL PRIMARY KEY,
    model_id INTEGER NOT NULL,
    dataset_id INTEGER NOT NULL,
    training_start TIMESTAMPTZ NOT NULL,
    training_end TIMESTAMPTZ,
    hyperparameters JSONB,
    training_metrics JSONB,
    validation_metrics JSONB,
    
    FOREIGN KEY (model_id) REFERENCES ml_models(id),
    FOREIGN KEY (dataset_id) REFERENCES training_datasets(id)
);
```

### Label Management

Handle different types of labels and their quality:

```sql
CREATE TABLE labels (
    id SERIAL PRIMARY KEY,
    entity_id VARCHAR(100) NOT NULL,
    label_type VARCHAR(50) NOT NULL, -- 'ground_truth', 'predicted', 'user_feedback'
    label_value JSONB NOT NULL,
    confidence_score DECIMAL(5,4),
    labeler_id VARCHAR(100),
    labeling_method VARCHAR(50), -- 'manual', 'automated', 'crowdsourced'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Track label changes
    previous_label_id INTEGER,
    FOREIGN KEY (previous_label_id) REFERENCES labels(id)
);

-- Index for efficient label lookups
CREATE INDEX idx_labels_entity_type ON labels(entity_id, label_type);
```

## Recommendation Systems

Store user interactions and recommendations:

```sql
-- User interactions
CREATE TABLE user_interactions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    item_id UUID NOT NULL,
    interaction_type VARCHAR(50) NOT NULL, -- 'view', 'click', 'purchase', 'rating'
    interaction_value DECIMAL(5,2), -- rating value, duration, etc.
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context JSONB, -- device, location, etc.
    
    INDEX idx_user_interactions_user_time (user_id, timestamp),
    INDEX idx_user_interactions_item_time (item_id, timestamp)
);

-- Recommendation results
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    item_id UUID NOT NULL,
    score DECIMAL(8,6) NOT NULL,
    rank INTEGER NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    recommendation_session_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Track which recommendations were shown/clicked
    shown_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    
    UNIQUE(user_id, item_id, recommendation_session_id)
);
```

## A/B Testing and Experimentation

Track model performance across different experiments:

```sql
CREATE TABLE experiments (
    id SERIAL PRIMARY KEY,
    experiment_name VARCHAR(100) NOT NULL,
    description TEXT,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    traffic_allocation DECIMAL(5,4), -- percentage of traffic
    status VARCHAR(20) DEFAULT 'draft', -- 'draft', 'running', 'completed', 'paused'
    
    UNIQUE(experiment_name)
);

CREATE TABLE experiment_variants (
    id SERIAL PRIMARY KEY,
    experiment_id INTEGER NOT NULL,
    variant_name VARCHAR(50) NOT NULL,
    model_id INTEGER NOT NULL,
    traffic_percentage DECIMAL(5,4) NOT NULL,
    
    FOREIGN KEY (experiment_id) REFERENCES experiments(id),
    FOREIGN KEY (model_id) REFERENCES ml_models(id),
    
    UNIQUE(experiment_id, variant_name)
);

-- Track user assignments to experiments
CREATE TABLE experiment_assignments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    experiment_id INTEGER NOT NULL,
    variant_id INTEGER NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (experiment_id) REFERENCES experiments(id),
    FOREIGN KEY (variant_id) REFERENCES experiment_variants(id),
    
    UNIQUE(user_id, experiment_id)
);
```

## Dynamic Pricing

Store pricing predictions and factors:

```sql
CREATE TABLE pricing_models (
    id SERIAL PRIMARY KEY,
    product_id UUID NOT NULL,
    base_price DECIMAL(10,2) NOT NULL,
    predicted_price DECIMAL(10,2) NOT NULL,
    demand_score DECIMAL(5,4),
    competition_factor DECIMAL(5,4),
    seasonality_factor DECIMAL(5,4),
    model_version VARCHAR(50) NOT NULL,
    prediction_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    
    -- Track price history as time series
    INDEX idx_pricing_product_time (product_id, prediction_timestamp),
    
    -- Ensure no overlapping validity periods
    EXCLUDE USING gist (
        product_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);
```

## Model Monitoring and Drift Detection

Track model performance over time:

```sql
CREATE TABLE model_performance_metrics (
    id SERIAL PRIMARY KEY,
    model_id INTEGER NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    metric_value DECIMAL(10,6) NOT NULL,
    measurement_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data_window_start TIMESTAMPTZ NOT NULL,
    data_window_end TIMESTAMPTZ NOT NULL,
    
    FOREIGN KEY (model_id) REFERENCES ml_models(id),
    
    INDEX idx_model_metrics_time (model_id, measurement_timestamp)
);

-- Data drift detection
CREATE TABLE data_drift_reports (
    id SERIAL PRIMARY KEY,
    model_id INTEGER NOT NULL,
    feature_name VARCHAR(100) NOT NULL,
    drift_score DECIMAL(8,6) NOT NULL,
    drift_threshold DECIMAL(8,6) NOT NULL,
    is_drifted BOOLEAN NOT NULL,
    detection_method VARCHAR(50) NOT NULL,
    report_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (model_id) REFERENCES ml_models(id)
);
```

## Best Practices

1. **Always version your models** - Track model versions, training data, and hyperparameters
2. **Store probabilities, not just classifications** - Maintains flexibility for threshold adjustments
3. **Implement feature stores** - Centralize feature management and ensure consistency
4. **Track model lineage** - Maintain relationships between data, features, models, and predictions
5. **Monitor model performance** - Implement automated drift detection and performance tracking
6. **Use appropriate data types** - DECIMAL for scores, JSONB for flexible metadata
7. **Implement proper indexing** - Optimize for time-series queries and entity lookups
8. **Handle temporal data correctly** - Use ranges for validity periods and avoid overlaps

## Common Pitfalls

- Storing only binary classifications instead of probabilities
- Not tracking model versions or training metadata
- Ignoring data drift and model degradation
- Poor indexing for time-series queries
- Not handling concurrent model updates properly
- Insufficient monitoring and alerting for model performance 
